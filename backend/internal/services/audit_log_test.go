package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
)

const (
	testAuditAdminPrefix = "test-audit-admin-"
)

func cleanupAuditTestData(t *testing.T) {
	t.Helper()
	// Clean up audit logs
	_, _ = db.Pool.Exec(context.Background(), `
		DELETE FROM audit_log WHERE admin_id IN (
			SELECT id FROM admins WHERE username LIKE $1 || '%'
		)
	`, testAuditAdminPrefix)
	// Clean up admins
	_, _ = db.Pool.Exec(context.Background(), `
		DELETE FROM admins WHERE username LIKE $1 || '%'
	`, testAuditAdminPrefix)
}

func TestAuditLog_RecordAndQuery(t *testing.T) {
	if db.Pool == nil {
		t.Skip("Postgres not available")
	}
	defer cleanupAuditTestData(t)

	// Create test admin first
	adminID := uuid.New()
	username := testAuditAdminPrefix + adminID.String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
	`, adminID, username, username+"@syrup.test", "hash", "admin")
	if err != nil {
		t.Fatalf("failed to insert test admin: %v", err)
	}

	// Record an audit entry
	action := "test_action"
	targetType := "waffle"
	targetID := uuid.New().String()
	details := "tested audit logging"
	ip := "127.0.0.1"

	err = RecordAudit(adminID, action, targetType, targetID, details, ip)
	if err != nil {
		t.Fatalf("RecordAudit failed: %v", err)
	}

	// Query audit log
	filters := AuditLogFilters{
		AdminID:    &adminID,
		Action:     action,
		TargetType: targetType,
	}

	entries, total, err := QueryAudit(filters)
	if err != nil {
		t.Fatalf("QueryAudit failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 audit entry, got %d", total)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry returned, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Action != action || entry.TargetType != targetType || entry.TargetID != targetID || entry.Details != details || entry.IPAddress != ip {
		t.Errorf("audit log details mismatch, got %+v", entry)
	}

	// Test GetAuditByID
	byID, err := GetAuditByID(entry.ID)
	if err != nil {
		t.Fatalf("GetAuditByID failed: %v", err)
	}
	if byID.ID != entry.ID {
		t.Errorf("expected audit ID %s, got %s", entry.ID, byID.ID)
	}
}
