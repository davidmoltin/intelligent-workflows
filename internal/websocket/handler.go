package websocket

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should check the origin properly
		// For now, allow all origins
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub    *Hub
	logger *zap.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, logger *zap.Logger) *Handler {
	return &Handler{
		hub:    hub,
		logger: logger,
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Warn("websocket connection attempted without user_id in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", zap.Error(err))
		return
	}

	// Create new client
	client := NewClient(h.hub, conn, userID, h.logger)

	// Register client with hub
	h.hub.register <- client

	// Start client goroutines
	client.Start()

	h.logger.Info("websocket connection established",
		zap.String("client_id", client.id),
		zap.String("user_id", userID),
		zap.String("remote_addr", r.RemoteAddr),
	)
}

// HandleStats returns WebSocket statistics
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats := h.hub.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Simple JSON encoding
	w.Write([]byte("{"))
	w.Write([]byte("\"total_clients\":"))
	w.Write([]byte(fmt.Sprintf("%d", stats["total_clients"])))
	w.Write([]byte(",\"total_users\":"))
	w.Write([]byte(fmt.Sprintf("%d", stats["total_users"])))
	w.Write([]byte("}"))
}
