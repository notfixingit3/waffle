package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
)

// ensureSettingsTable creates the system_settings table and seeds the
// whois_server value once for the test run.
var ensureSettingsTableOnce sync.Once

func ensureSettingsTable(t *testing.T) {
	t.Helper()
	ensureSettingsTableOnce.Do(func() {
		ctx := context.Background()

		_, err := db.Pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS system_settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_by UUID REFERENCES admins(id)
			)
		`)
		if err != nil {
			t.Fatalf("create system_settings table: %v", err)
		}

		// Seed the default whois_server value if not present.
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO system_settings (key, value, updated_at)
			VALUES ('whois_server', 'whois.pwhois.org', NOW())
			ON CONFLICT (key) DO NOTHING
		`)
		if err != nil {
			t.Fatalf("seed whois_server: %v", err)
		}
	})
}

// createTestSettingsAdmin inserts a temporary admin for SetSetting tests and
// returns its ID. Tests using this helper must call cleanupTestSettingsAdmin.
func createTestSettingsAdmin(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	username := "test-settings-" + id.String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, username, username+"@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test settings admin: %v", err)
	}
	return id
}

// cleanupTestSettingsAdmin removes the admin created by createTestSettingsAdmin.
func cleanupTestSettingsAdmin(t *testing.T, id uuid.UUID) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, id)
	if err != nil {
		t.Fatalf("cleanup test settings admin: %v", err)
	}
}

// cleanupTestSettings removes test-prefixed keys and restores the whois_server
// seed value so tests never leak state.
func cleanupTestSettings(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `DELETE FROM system_settings WHERE key LIKE 'test-%'`)
	if err != nil {
		t.Fatalf("cleanup test settings: %v", err)
	}
	// Restore the seed value that may have been modified by SetSetting tests.
	// Clear updated_by so admin FK cleanup works.
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO system_settings (key, value, updated_at, updated_by)
		VALUES ('whois_server', 'whois.pwhois.org', NOW(), NULL)
		ON CONFLICT (key) DO UPDATE SET value = 'whois.pwhois.org', updated_at = NOW(), updated_by = NULL
	`)
	if err != nil {
		t.Fatalf("restore whois_server seed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetSetting tests
// ---------------------------------------------------------------------------

func TestGetSetting_ReturnsEmptyForMissingKey(t *testing.T) {
	ensureSettingsTable(t)
	defer cleanupTestSettings(t)

	val, err := GetSetting("test-nonexistent-key")
	if err != nil {
		t.Fatalf("GetSetting returned error for missing key: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestGetSetting_ReturnsExistingValue(t *testing.T) {
	ensureSettingsTable(t)
	defer cleanupTestSettings(t)

	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO system_settings (key, value, updated_at) VALUES ($1, $2, $3)
	`, "test-get-existing", "hello-world", time.Now())
	if err != nil {
		t.Fatalf("insert test setting: %v", err)
	}

	val, err := GetSetting("test-get-existing")
	if err != nil {
		t.Fatalf("GetSetting returned error: %v", err)
	}
	if val != "hello-world" {
		t.Errorf("expected %q, got %q", "hello-world", val)
	}
}

func TestGetSetting_ReturnsSeedValue(t *testing.T) {
	ensureSettingsTable(t)
	cleanupTestSettings(t)
	defer cleanupTestSettings(t)
	// The seed from migration 007 is always present.
	val, err := GetSetting("whois_server")
	if err != nil {
		t.Fatalf("GetSetting returned error for seed key: %v", err)
	}
	if val != "whois.pwhois.org" {
		t.Errorf("expected %q, got %q", "whois.pwhois.org", val)
	}
}

// ---------------------------------------------------------------------------
// SetSetting validation tests
// ---------------------------------------------------------------------------

func TestSetSetting_RejectsInvalidKey(t *testing.T) {
	ensureSettingsTable(t)
	adminID := createTestSettingsAdmin(t)
	defer cleanupTestSettings(t)
	defer cleanupTestSettingsAdmin(t, adminID)

	err := SetSetting("invalid_key", "some-value", adminID)
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestSetSetting_RejectsEmptyHostname(t *testing.T) {
	ensureSettingsTable(t)
	adminID := createTestSettingsAdmin(t)
	defer cleanupTestSettings(t)
	defer cleanupTestSettingsAdmin(t, adminID)

	err := SetSetting("whois_server", "", adminID)
	if err == nil {
		t.Fatal("expected error for empty hostname, got nil")
	}
}

func TestSetSetting_RejectsHostnameWithoutDot(t *testing.T) {
	ensureSettingsTable(t)
	adminID := createTestSettingsAdmin(t)
	defer cleanupTestSettings(t)
	defer cleanupTestSettingsAdmin(t, adminID)

	err := SetSetting("whois_server", "invalidhostname", adminID)
	if err == nil {
		t.Fatal("expected error for hostname without dot, got nil")
	}
}

func TestSetSetting_RejectsHostnameWithSpaces(t *testing.T) {
	ensureSettingsTable(t)
	adminID := createTestSettingsAdmin(t)
	defer cleanupTestSettings(t)
	defer cleanupTestSettingsAdmin(t, adminID)

	err := SetSetting("whois_server", "invalid hostname", adminID)
	if err == nil {
		t.Fatal("expected error for hostname with spaces, got nil")
	}
}

// ---------------------------------------------------------------------------
// SetSetting success tests
// ---------------------------------------------------------------------------

func TestSetSetting_ValidHostname(t *testing.T) {
	ensureSettingsTable(t)
	adminID := createTestSettingsAdmin(t)
	defer cleanupTestSettingsAdmin(t, adminID)
	defer cleanupTestSettings(t)

	err := SetSetting("whois_server", "whois.verisign-grs.com", adminID)
	if err != nil {
		t.Fatalf("SetSetting returned error: %v", err)
	}

	val, err := GetSetting("whois_server")
	if err != nil {
		t.Fatalf("GetSetting returned error: %v", err)
	}
	if val != "whois.verisign-grs.com" {
		t.Errorf("expected %q, got %q", "whois.verisign-grs.com", val)
	}
}

func TestSetSetting_UpsertsExistingKey(t *testing.T) {
	ensureSettingsTable(t)
	adminID := createTestSettingsAdmin(t)
	defer cleanupTestSettingsAdmin(t, adminID)
	defer cleanupTestSettings(t)

	err := SetSetting("whois_server", "whois.verisign-grs.com", adminID)
	if err != nil {
		t.Fatalf("first SetSetting failed: %v", err)
	}

	err = SetSetting("whois_server", "whois.markmonitor.com", adminID)
	if err != nil {
		t.Fatalf("second SetSetting failed: %v", err)
	}

	val, err := GetSetting("whois_server")
	if err != nil {
		t.Fatalf("GetSetting returned error: %v", err)
	}
	if val != "whois.markmonitor.com" {
		t.Errorf("expected %q, got %q", "whois.markmonitor.com", val)
	}
}
