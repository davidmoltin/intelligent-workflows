package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client represents a WebSocket client connection
type Client struct {
	id            string
	userID        string
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]*Subscription // channel -> subscription
	mu            sync.RWMutex
	logger        *zap.Logger
	ctx           context.Context
	cancel        context.CancelFunc
}

// Subscription represents a channel subscription with filters
type Subscription struct {
	Channel string
	Filters Filters
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID string, logger *zap.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		id:            uuid.New().String(),
		userID:        userID,
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]*Subscription),
		logger:        logger.With(zap.String("client_id", uuid.New().String()), zap.String("user_id", userID)),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the client's read and write goroutines
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// Close closes the client connection
func (c *Client) Close() {
	c.cancel()
	c.hub.unregister <- c
}

// Subscribe adds a subscription for the client
func (c *Client) Subscribe(channel string, filters Filters) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriptions[channel] = &Subscription{
		Channel: channel,
		Filters: filters,
	}

	c.logger.Info("client subscribed to channel",
		zap.String("channel", channel),
		zap.Any("filters", filters),
	)
}

// Unsubscribe removes a subscription for the client
func (c *Client) Unsubscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.subscriptions, channel)

	c.logger.Info("client unsubscribed from channel",
		zap.String("channel", channel),
	)
}

// IsSubscribed checks if the client is subscribed to a channel
func (c *Client) IsSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.subscriptions[channel]
	return exists
}

// MatchesFilters checks if an event matches the client's subscription filters
func (c *Client) MatchesFilters(channel string, workflowID, executionID, status string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sub, exists := c.subscriptions[channel]
	if !exists {
		return false
	}

	// If no filters, match everything
	if len(sub.Filters.WorkflowIDs) == 0 &&
	   len(sub.Filters.ExecutionIDs) == 0 &&
	   len(sub.Filters.Statuses) == 0 {
		return true
	}

	// Check workflow ID filter
	if len(sub.Filters.WorkflowIDs) > 0 && !contains(sub.Filters.WorkflowIDs, workflowID) {
		return false
	}

	// Check execution ID filter
	if len(sub.Filters.ExecutionIDs) > 0 && !contains(sub.Filters.ExecutionIDs, executionID) {
		return false
	}

	// Check status filter
	if len(sub.Filters.Statuses) > 0 && !contains(sub.Filters.Statuses, status) {
		return false
	}

	return true
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer c.Close()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, messageData, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.Error("websocket read error", zap.Error(err))
				}
				return
			}

			c.handleMessage(messageData)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(data []byte) {
	msg, err := ParseMessage(data)
	if err != nil {
		c.logger.Error("failed to parse message", zap.Error(err))
		c.sendError("PARSE_ERROR", "Invalid message format")
		return
	}

	switch msg.Type {
	case MessageTypePing:
		c.sendPong()

	case MessageTypeSubscribe:
		var subData SubscriptionData
		if err := json.Unmarshal(msg.Data, &subData); err != nil {
			c.sendError("INVALID_SUBSCRIPTION", "Invalid subscription data")
			return
		}
		c.Subscribe(subData.Channel, subData.Filters)
		c.sendSubscribed(subData.Channel)

	case MessageTypeUnsubscribe:
		var subData SubscriptionData
		if err := json.Unmarshal(msg.Data, &subData); err != nil {
			c.sendError("INVALID_SUBSCRIPTION", "Invalid subscription data")
			return
		}
		c.Unsubscribe(subData.Channel)
		c.sendUnsubscribed(subData.Channel)

	default:
		c.logger.Warn("unknown message type", zap.String("type", string(msg.Type)))
	}
}

// sendPong sends a pong message
func (c *Client) sendPong() {
	msg, _ := NewMessage(MessageTypePong, nil)
	data, _ := msg.ToJSON()
	select {
	case c.send <- data:
	default:
		c.logger.Warn("send channel full, dropping pong message")
	}
}

// sendError sends an error message
func (c *Client) sendError(code, message string) {
	msg, _ := NewMessage(MessageTypeError, ErrorData{
		Code:    code,
		Message: message,
	})
	data, _ := msg.ToJSON()
	select {
	case c.send <- data:
	default:
		c.logger.Warn("send channel full, dropping error message")
	}
}

// sendSubscribed sends a subscription confirmation
func (c *Client) sendSubscribed(channel string) {
	msg, _ := NewMessage(MessageTypeSubscribed, map[string]string{
		"channel": channel,
	})
	data, _ := msg.ToJSON()
	select {
	case c.send <- data:
	default:
		c.logger.Warn("send channel full, dropping subscribed message")
	}
}

// sendUnsubscribed sends an unsubscription confirmation
func (c *Client) sendUnsubscribed(channel string) {
	msg, _ := NewMessage(MessageTypeUnsubscribed, map[string]string{
		"channel": channel,
	})
	data, _ := msg.ToJSON()
	select {
	case c.send <- data:
	default:
		c.logger.Warn("send channel full, dropping unsubscribed message")
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
