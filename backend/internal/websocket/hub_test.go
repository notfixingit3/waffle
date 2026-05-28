package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewHubStartsEmpty(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	if h.TotalClients() != 0 {
		t.Fatalf("TotalClients = %d, want 0", h.TotalClients())
	}
	if h.clients == nil || h.broadcast == nil || h.Register == nil || h.Unregister == nil || h.done == nil {
		t.Fatal("hub channels and client map must be initialized")
	}
}

func TestMessageJSONOmitsRoom(t *testing.T) {
	data, err := json.Marshal(Message{Room: "waffle:test", Type: "SPOT_UPDATED", Payload: map[string]int{"spot_number": 7}})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if _, ok := payload["Room"]; ok {
		t.Fatalf("Room must be omitted from websocket JSON: %s", string(data))
	}
	if _, ok := payload["room"]; ok {
		t.Fatalf("room must be omitted from websocket JSON: %s", string(data))
	}
	if payload["type"] != "SPOT_UPDATED" {
		t.Fatalf("type = %v, want SPOT_UPDATED", payload["type"])
	}
}

func TestHubBroadcastQueuesMessage(t *testing.T) {
	h := NewHub()
	go h.Broadcast("waffle:test", "WINNER_CLEARED", map[string]string{"slug": "test"})

	select {
	case msg := <-h.broadcast:
		if msg.Room != "waffle:test" || msg.Type != "WINNER_CLEARED" {
			t.Fatalf("message = %#v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}
}

func TestInitHubSetsGlobalHub(t *testing.T) {
	InitHub()
	if GetHub() == nil {
		t.Fatal("GetHub returned nil after InitHub")
	}
	GetHub().Stop()
}
