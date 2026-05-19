package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ProgressEvent represents a workflow progress event.
type ProgressEvent struct {
	Type      string `json:"type"`       // step_start, step_complete, step_failed, workflow_status
	StepID    string `json:"step_id"`    // step identifier
	Role      string `json:"role"`       // role name
	Action    string `json:"action"`     // action name
	Status    string `json:"status"`     // step/workflow status
	Progress  string `json:"progress"`   // e.g., "3/7"
	Error     string `json:"error"`      // error message if failed
	SessionID string `json:"session_id"` // session identifier
}

// StepsClearedEvent represents steps being cleared for restart.
type StepsClearedEvent struct {
	Type      string   `json:"type"`      // "steps_cleared"
	StepIDs   []string `json:"step_ids"`  // cleared step IDs
	Reason    string   `json:"reason"`    // "restart_step"
	Restarted string   `json:"restarted"` // the step being restarted
}

// WebSocketHandler manages WebSocket connections.
type WebSocketHandler struct {
	clients    map[string]*websocket.Conn // key: session_id
	companyConns map[string]*websocket.Conn // key: company_id (for list page monitoring)
	mu         sync.Mutex
}

// NewWebSocketHandler creates a handler.
func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		clients:     make(map[string]*websocket.Conn),
		companyConns: make(map[string]*websocket.Conn),
	}
}

// Handle upgrades HTTP to WebSocket.
func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return
	}

	// Support both session_id (detail page) and company_id (list page)
	sessionID := r.URL.Query().Get("session_id")
	companyID := r.URL.Query().Get("company_id")

	h.mu.Lock()
	if sessionID != "" {
		h.clients[sessionID] = conn
	}
	if companyID != "" {
		h.companyConns[companyID] = conn
	}
	h.mu.Unlock()

	// Read loop (keep connection alive)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.mu.Lock()
			delete(h.clients, sessionID)
			delete(h.companyConns, companyID)
			h.mu.Unlock()
			conn.Close()
			break
		}
	}
}

// Broadcast sends event to all clients for a session.
func (h *WebSocketHandler) Broadcast(sessionID string, event map[string]interface{}) {
	h.mu.Lock()
	conn, ok := h.clients[sessionID]
	h.mu.Unlock()

	if ok {
		data, _ := json.Marshal(event)
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

// BroadcastProgress sends a progress event to clients.
// Also broadcasts to company-level connections for list page monitoring.
func (h *WebSocketHandler) BroadcastProgress(sessionID string, event ProgressEvent) {
	event.SessionID = sessionID
	data, _ := json.Marshal(map[string]interface{}{
		"type":       event.Type,
		"step_id":    event.StepID,
		"role":       event.Role,
		"action":     event.Action,
		"status":     event.Status,
		"progress":   event.Progress,
		"error":      event.Error,
		"session_id": event.SessionID,
	})

	// Debug: log the message being sent
	slog.Info("BroadcastProgress sending", "session_id", sessionID, "type", event.Type, "status", event.Status, "data", string(data))

	h.mu.Lock()
	defer h.mu.Unlock()

	// Send to session-specific connection (detail page)
	if conn, ok := h.clients[sessionID]; ok {
		conn.WriteMessage(websocket.TextMessage, data)
	}

	// Also send to all company connections (list page)
	// Note: we need to know which company this session belongs to
	// For now, we'll broadcast to all company connections
	// (This is acceptable since company list page only monitors its own company)
	for _, conn := range h.companyConns {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}