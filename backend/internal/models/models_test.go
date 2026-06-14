package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestDomainConstantsMatchPersistedValues(t *testing.T) {
	tests := map[string]string{
		"active waffle":       string(WaffleStatusActive),
		"completed waffle":    string(WaffleStatusCompleted),
		"available spot":      string(SpotStatusAvailable),
		"pending spot":        string(SpotStatusPending),
		"paid spot":           string(SpotStatusPaid),
		"winner spot":         string(SpotStatusWinner),
		"loser spot":          string(SpotStatusLoser),
		"super admin role":    RoleSuperAdmin,
		"admin role":          RoleAdmin,
		"waffle manager role": RoleWaffleManager,
	}

	expected := map[string]string{
		"active waffle":       "active",
		"completed waffle":    "completed",
		"available spot":      "available",
		"pending spot":        "pending",
		"paid spot":           "paid",
		"winner spot":         "winner",
		"loser spot":          "loser",
		"super admin role":    "super_admin",
		"admin role":          "admin",
		"waffle manager role": "waffle_manager",
	}

	for name, got := range tests {
		if got != expected[name] {
			t.Fatalf("%s = %q, want %q", name, got, expected[name])
		}
	}
}

func TestAdminJSONIncludesAuditSecurityFields(t *testing.T) {
	ip := "127.0.0.1"
	admin := Admin{
		Username:    "admin",
		Email:       "admin@example.test",
		Role:        RoleSuperAdmin,
		Active:      true,
		LastLoginIP: &ip,
	}

	data, err := json.Marshal(admin)
	if err != nil {
		t.Fatalf("marshal admin: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal admin json: %v", err)
	}
	if payload["last_login_ip"] != ip {
		t.Fatalf("last_login_ip = %v, want %q", payload["last_login_ip"], ip)
	}
	if _, ok := payload["password"]; ok {
		t.Fatal("admin JSON must not expose password")
	}
}

func TestAuditLogJSONShape(t *testing.T) {
	entry := AuditLog{
		Action:     "delete_waffle",
		TargetType: "waffle",
		TargetID:   "abc",
		Details:    "deleted waffle",
		IPAddress:  "127.0.0.1",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal audit log: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal audit log json: %v", err)
	}
	for _, key := range []string{"action", "target_type", "target_id", "details", "ip_address"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing audit json key %q in %s", key, string(data))
		}
	}
}

func TestWaffleJSONHidesShareFields(t *testing.T) {
	id := uuid.New()
	msg := "secret share message"
	waffle := Waffle{
		ID:              id,
		Slug:            "test-waffle",
		Title:           "Test Waffle",
		ShareTemplateID: &id,
		ShareMessage:    &msg,
	}

	data, err := json.Marshal(waffle)
	if err != nil {
		t.Fatalf("marshal waffle: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal waffle json: %v", err)
	}
	if _, ok := payload["share_template_id"]; ok {
		t.Fatal("waffle JSON must not expose share_template_id")
	}
	if _, ok := payload["share_message"]; ok {
		t.Fatal("waffle JSON must not expose share_message")
	}
	if _, ok := payload["title"]; !ok {
		t.Fatal("waffle JSON should still expose title")
	}
}
