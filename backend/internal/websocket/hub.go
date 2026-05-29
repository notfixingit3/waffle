package websocket

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	gorillaWs "github.com/gorilla/websocket"
)

var upgrader = gorillaWs.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket connections
	},
}

type Hub struct {
	clients    map[string]map[*gorillaWs.Conn]bool
	broadcast  chan Message
	Register   chan Subscription
	Unregister chan Subscription
	done       chan struct{}
	mu         sync.RWMutex
}

type Subscription struct {
	Conn *gorillaWs.Conn
	Room string
}

type Message struct {
	Room    string      `json:"-"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var hub *Hub

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*gorillaWs.Conn]bool),
		broadcast:  make(chan Message),
		Register:   make(chan Subscription),
		Unregister: make(chan Subscription),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			return
		case sub := <-h.Register:
			h.mu.Lock()
			if h.clients[sub.Room] == nil {
				h.clients[sub.Room] = make(map[*gorillaWs.Conn]bool)
			}
			h.clients[sub.Room][sub.Conn] = true
			h.mu.Unlock()
			slog.Info("Client registered to room", "room", sub.Room)

		case sub := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.clients[sub.Room]; ok {
				delete(clients, sub.Conn)
				if len(clients) == 0 {
					delete(h.clients, sub.Room)
				}
			}
			h.mu.Unlock()
			if err := sub.Conn.Close(); err != nil {
				slog.Error("Failed to close WebSocket connection", "error", err, "room", sub.Room)
			}
			slog.Info("Client unregistered from room", "room", sub.Room)

		case msg := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[msg.Room]
			h.mu.RUnlock()

			for client := range clients {
				if err := client.WriteJSON(msg); err != nil {
					slog.Error("Error writing to client", "error", err)
					if closeErr := client.Close(); closeErr != nil {
						slog.Error("Failed to close client connection", "error", closeErr)
					}
					h.Unregister <- Subscription{Conn: client, Room: msg.Room}
				}
			}
		}
	}
}

func (h *Hub) Stop() {
	h.mu.Lock()
	for _, clients := range h.clients {
		for client := range clients {
			closeMsg := gorillaWs.FormatCloseMessage(gorillaWs.CloseGoingAway, "server shutting down")
			if err := client.WriteMessage(gorillaWs.CloseMessage, closeMsg); err != nil {
				slog.Error("Failed to send close message", "error", err)
			}
			if err := client.Close(); err != nil {
				slog.Error("Failed to close client connection during shutdown", "error", err)
			}
		}
	}
	h.mu.Unlock()

	close(h.done)
}

func (h *Hub) Broadcast(room string, msgType string, payload interface{}) {
	h.broadcast <- Message{
		Room:    room,
		Type:    msgType,
		Payload: payload,
	}
}

func (h *Hub) TotalClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}

func InitHub() {
	hub = NewHub()
	go hub.Run()
}

func GetHub() *Hub {
	return hub
}

// HandleWebSocketUpgrade upgrades HTTP connection to WebSocket and manages the subscription.
func HandleWebSocketUpgrade(c *gin.Context) {
	slug := c.Param("slug")
	room := "waffle:" + slug

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return
	}

	sub := Subscription{Conn: conn, Room: room}
	hub.Register <- sub

	defer func() {
		hub.Unregister <- sub
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if gorillaWs.IsUnexpectedCloseError(err, gorillaWs.CloseGoingAway, gorillaWs.CloseAbnormalClosure) {
				slog.Error("WebSocket error", "error", err)
			}
			return
		}
	}
}

func BroadcastSpotUpdate(slug string, spotNumber int, status string, claimedByHandle string) {
	room := "waffle:" + slug
	hub.Broadcast(room, "SPOT_UPDATED", map[string]interface{}{
		"spot_number":       spotNumber,
		"status":            status,
		"claimed_by_handle": claimedByHandle,
	})
}

func BroadcastWaffleCompleted(slug string, winningSpotNumber int) {
	room := "waffle:" + slug
	hub.Broadcast(room, "WAFFLE_COMPLETED", map[string]interface{}{
		"winning_spot_number": winningSpotNumber,
	})
}

func BroadcastActivityEvent(slug string, eventType string, message string, instagramHandle string, spotNumbers []int) {
	room := "waffle:" + slug
	hub.Broadcast(room, "ACTIVITY_EVENT", map[string]interface{}{
		"event_type":       eventType,
		"message":          message,
		"instagram_handle": instagramHandle,
		"spot_numbers":     spotNumbers,
	})
}

func BroadcastWinnerCleared(slug string) {
	room := "waffle:" + slug
	hub.Broadcast(room, "WINNER_CLEARED", map[string]interface{}{
		"slug": slug,
	})
}

func BroadcastWinnerChanged(slug string, winningSpotNumber int) {
	room := "waffle:" + slug
	hub.Broadcast(room, "WINNER_CHANGED", map[string]interface{}{
		"winning_spot_number": winningSpotNumber,
	})
}
