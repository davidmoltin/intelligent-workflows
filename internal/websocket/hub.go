package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// Redis pub/sub channels
	redisChannelPrefix = "ws:broadcast:"
	redisChannelAll    = "ws:broadcast:all"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by user ID
	clients map[string]map[*Client]bool // userID -> clients

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to clients
	broadcast chan *BroadcastMessage

	// Redis client for pub/sub
	redisClient *redis.Client

	// Redis pub/sub
	redisPubSub *redis.PubSub

	// Logger
	logger *zap.Logger

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// BroadcastMessage represents a message to broadcast to clients
type BroadcastMessage struct {
	Channel     string          // e.g., "executions", "executions:{id}", "workflows:{id}"
	Message     *Message
	WorkflowID  string
	ExecutionID string
	Status      string
	UserID      string // If set, only broadcast to this user
}

// NewHub creates a new Hub
func NewHub(redisClient *redis.Client, logger *zap.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		clients:     make(map[string]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *BroadcastMessage, 256),
		redisClient: redisClient,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}

	return hub
}

// Start starts the hub
func (h *Hub) Start() error {
	// Subscribe to Redis pub/sub for distributed broadcasting
	if h.redisClient != nil {
		h.redisPubSub = h.redisClient.Subscribe(h.ctx, redisChannelAll)
		go h.handleRedisPubSub()
	}

	go h.run()

	h.logger.Info("WebSocket hub started")
	return nil
}

// Stop stops the hub
func (h *Hub) Stop() error {
	h.cancel()

	if h.redisPubSub != nil {
		h.redisPubSub.Close()
	}

	h.logger.Info("WebSocket hub stopped")
	return nil
}

// run handles the hub's main loop
func (h *Hub) run() {
	for {
		select {
		case <-h.ctx.Done():
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.userID] == nil {
		h.clients[client.userID] = make(map[*Client]bool)
	}
	h.clients[client.userID][client] = true

	h.logger.Info("client registered",
		zap.String("client_id", client.id),
		zap.String("user_id", client.userID),
		zap.Int("total_clients", h.getTotalClients()),
	)
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.userID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.send)

			if len(clients) == 0 {
				delete(h.clients, client.userID)
			}

			h.logger.Info("client unregistered",
				zap.String("client_id", client.id),
				zap.String("user_id", client.userID),
				zap.Int("total_clients", h.getTotalClients()),
			)
		}
	}
}

// broadcastMessage broadcasts a message to relevant clients
func (h *Hub) broadcastMessage(bm *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	messageData, err := bm.Message.ToJSON()
	if err != nil {
		h.logger.Error("failed to marshal broadcast message", zap.Error(err))
		return
	}

	// If userID is specified, only send to that user's clients
	if bm.UserID != "" {
		if clients, ok := h.clients[bm.UserID]; ok {
			for client := range clients {
				h.sendToClient(client, bm, messageData)
			}
		}
		return
	}

	// Otherwise, send to all relevant clients
	sentCount := 0
	for _, clients := range h.clients {
		for client := range clients {
			if h.shouldSendToClient(client, bm) {
				h.sendToClient(client, bm, messageData)
				sentCount++
			}
		}
	}

	h.logger.Debug("broadcast message sent",
		zap.String("channel", bm.Channel),
		zap.String("type", string(bm.Message.Type)),
		zap.Int("recipients", sentCount),
	)
}

// shouldSendToClient checks if a message should be sent to a client
func (h *Hub) shouldSendToClient(client *Client, bm *BroadcastMessage) bool {
	// Check if client is subscribed to the channel
	if !client.IsSubscribed(bm.Channel) {
		return false
	}

	// Check if message matches client's filters
	return client.MatchesFilters(bm.Channel, bm.WorkflowID, bm.ExecutionID, bm.Status)
}

// sendToClient sends a message to a specific client
func (h *Hub) sendToClient(client *Client, bm *BroadcastMessage, messageData []byte) {
	select {
	case client.send <- messageData:
	default:
		// Client's send channel is full, close the connection
		h.logger.Warn("client send channel full, closing connection",
			zap.String("client_id", client.id),
			zap.String("user_id", client.userID),
		)
		go client.Close()
	}
}

// Broadcast sends a message to all relevant clients (local and via Redis)
func (h *Hub) Broadcast(channel string, message *Message, workflowID, executionID, status, userID string) {
	bm := &BroadcastMessage{
		Channel:     channel,
		Message:     message,
		WorkflowID:  workflowID,
		ExecutionID: executionID,
		Status:      status,
		UserID:      userID,
	}

	// Send to local clients
	select {
	case h.broadcast <- bm:
	default:
		h.logger.Warn("broadcast channel full, dropping message")
	}

	// Publish to Redis for other instances
	if h.redisClient != nil {
		h.publishToRedis(bm)
	}
}

// publishToRedis publishes a broadcast message to Redis
func (h *Hub) publishToRedis(bm *BroadcastMessage) {
	data, err := json.Marshal(bm)
	if err != nil {
		h.logger.Error("failed to marshal broadcast message for Redis", zap.Error(err))
		return
	}

	if err := h.redisClient.Publish(h.ctx, redisChannelAll, data).Err(); err != nil {
		h.logger.Error("failed to publish to Redis", zap.Error(err))
	}
}

// handleRedisPubSub handles incoming messages from Redis pub/sub
func (h *Hub) handleRedisPubSub() {
	ch := h.redisPubSub.Channel()

	for {
		select {
		case <-h.ctx.Done():
			return

		case msg := <-ch:
			var bm BroadcastMessage
			if err := json.Unmarshal([]byte(msg.Payload), &bm); err != nil {
				h.logger.Error("failed to unmarshal Redis message", zap.Error(err))
				continue
			}

			// Broadcast to local clients only (don't re-publish to Redis)
			select {
			case h.broadcast <- &bm:
			default:
				h.logger.Warn("broadcast channel full, dropping Redis message")
			}
		}
	}
}

// BroadcastExecutionEvent broadcasts an execution event
func (h *Hub) BroadcastExecutionEvent(msgType MessageType, data *ExecutionEventData) {
	message, err := NewMessage(msgType, data)
	if err != nil {
		h.logger.Error("failed to create execution event message", zap.Error(err))
		return
	}

	// Broadcast to multiple channels
	channels := []string{
		"executions",
		fmt.Sprintf("executions:%s", data.ExecutionID),
		fmt.Sprintf("workflows:%s", data.WorkflowID),
	}

	for _, channel := range channels {
		h.Broadcast(channel, message, data.WorkflowID, data.ExecutionID, data.Status, "")
	}
}

// BroadcastStepEvent broadcasts a step event
func (h *Hub) BroadcastStepEvent(msgType MessageType, data *StepEventData) {
	message, err := NewMessage(msgType, data)
	if err != nil {
		h.logger.Error("failed to create step event message", zap.Error(err))
		return
	}

	// Broadcast to execution-specific channel
	channel := fmt.Sprintf("executions:%s", data.ExecutionID)
	h.Broadcast(channel, message, "", data.ExecutionID, data.Status, "")
}

// BroadcastApprovalEvent broadcasts an approval event
func (h *Hub) BroadcastApprovalEvent(msgType MessageType, data *ApprovalEventData) {
	message, err := NewMessage(msgType, data)
	if err != nil {
		h.logger.Error("failed to create approval event message", zap.Error(err))
		return
	}

	// Broadcast to multiple channels
	channels := []string{
		"approvals",
		fmt.Sprintf("executions:%s", data.ExecutionID),
		fmt.Sprintf("workflows:%s", data.WorkflowID),
	}

	for _, channel := range channels {
		h.Broadcast(channel, message, data.WorkflowID, data.ExecutionID, data.Status, "")
	}
}

// GetClientCount returns the total number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getTotalClients()
}

// getTotalClients returns the total number of connected clients (must be called with lock held)
func (h *Hub) getTotalClients() int {
	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

// GetUserClientCount returns the number of clients for a specific user
func (h *Hub) GetUserClientCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		return len(clients)
	}
	return 0
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"total_clients": h.getTotalClients(),
		"total_users":   len(h.clients),
		"channels": map[string]int{
			"register":   len(h.register),
			"unregister": len(h.unregister),
			"broadcast":  len(h.broadcast),
		},
	}
}

// ParseChannel parses a channel string and extracts resource type and ID
func ParseChannel(channel string) (resourceType, resourceID string) {
	parts := strings.SplitN(channel, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return channel, ""
}
