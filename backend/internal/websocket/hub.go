package websocket

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	gorillaWs "github.com/gorilla/websocket"
)

var upgrader = gorillaWs.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket connections
	},
}

// clientConn wraps a gorilla WebSocket connection with a mutex for serialized writes.
// gorilla/websocket requires that only one goroutine writes to a connection at a time.
type clientConn struct {
	Conn *gorillaWs.Conn
	mu   sync.Mutex
}

type Hub struct {
	clients    map[string]map[*clientConn]bool
	broadcast  chan Message
	Register   chan Subscription
	Unregister chan Subscription
	done       chan struct{}
	mu         sync.RWMutex
}

type Subscription struct {
	Conn *clientConn
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
		clients:    make(map[string]map[*clientConn]bool),
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
				h.clients[sub.Room] = make(map[*clientConn]bool)
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
			if err := sub.Conn.Conn.Close(); err != nil {
				slog.Error("Failed to close WebSocket connection", "error", err, "room", sub.Room)
			}
			slog.Info("Client unregistered from room", "room", sub.Room)

		case msg := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[msg.Room]
			h.mu.RUnlock()

			for client := range clients {
				client.mu.Lock()
				err := client.Conn.WriteJSON(msg)
				client.mu.Unlock()
				if err != nil {
					slog.Error("Error writing to client", "error", err)
					if closeErr := client.Conn.Close(); closeErr != nil {
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
			if err := client.Conn.WriteMessage(gorillaWs.CloseMessage, closeMsg); err != nil {
				slog.Error("Failed to send close message", "error", err)
			}
			if err := client.Conn.Close(); err != nil {
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

const (
	// heartbeatInterval is how often the server sends a ping frame to each client.
	heartbeatInterval = 30 * time.Second
	// readDeadline is the max time without receiving a message or pong before the connection is closed.
	// Must be longer than heartbeatInterval to allow for network latency (2 missed pongs = disconnect).
	readDeadline = 60 * time.Second
	// writeWait is the write deadline for ping/pong write control frames.
	writeWait = 10 * time.Second
)

// HandleWebSocketUpgrade upgrades HTTP connection to WebSocket and manages the subscription.
// It starts a heartbeat goroutine that sends ping frames every 30s.
// If no message or pong is received within 60s, the read loop exits and the connection closes.
func HandleWebSocketUpgrade(c *gin.Context) {
	slug := c.Param("slug")
	room := "waffle:" + slug

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("WebSocket upgrade error", "error", err)
		return
	}

	cc := &clientConn{Conn: conn}
	sub := Subscription{Conn: cc, Room: room}

	// Set initial read deadline. If no message/pong within readDeadline, ReadMessage returns an error.
	conn.SetReadDeadline(time.Now().Add(readDeadline))

	// On receiving a ping from the peer, reset the read deadline and auto-respond with a pong.
	conn.SetPingHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		return conn.WriteMessage(gorillaWs.PongMessage, []byte(appData))
	})

	// On receiving a pong (response to our ping), reset the read deadline.
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	// Start heartbeat goroutine: send ping frames every heartbeatInterval.
	ticker := time.NewTicker(heartbeatInterval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				cc.mu.Lock()
				if err := conn.WriteControl(gorillaWs.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
					slog.Error("Failed to send ping", "error", err)
					cc.mu.Unlock()
					return
				}
				cc.mu.Unlock()
			case <-done:
				return
			}
		}
	}()

	hub.Register <- sub

	defer func() {
		ticker.Stop()
		close(done)
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
		// Reset the read deadline on every successfully read message.
		conn.SetReadDeadline(time.Now().Add(readDeadline))
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

func BroadcastWaffleCompleted(slug string, winningSpotNumbers []int) {
	room := "waffle:" + slug
	first := 0
	if len(winningSpotNumbers) > 0 {
		first = winningSpotNumbers[0]
	}
	hub.Broadcast(room, "WAFFLE_COMPLETED", map[string]interface{}{
		"winning_spot_numbers": winningSpotNumbers,
		"winning_spot_number":  first,
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

func BroadcastWinnerChanged(slug string, winningSpotNumbers []int) {
	room := "waffle:" + slug
	first := 0
	if len(winningSpotNumbers) > 0 {
		first = winningSpotNumbers[0]
	}
	hub.Broadcast(room, "WINNER_CHANGED", map[string]interface{}{
		"winning_spot_numbers": winningSpotNumbers,
		"winning_spot_number":  first,
	})
}
