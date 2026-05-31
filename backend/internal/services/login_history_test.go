package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

// migration007Up and down are loaded from the migration files.
var migration007Up, migration007Down string

func init() {
	// Resolve migration file paths relative to the project root.
	// The test runs from backend/internal/services/, so migrations are at ../../migrations/.
	base := filepath.Join("..", "..", "migrations")

	upPath := filepath.Join(base, "007_add_login_history_and_settings.up.sql")
	downPath := filepath.Join(base, "007_add_login_history_and_settings.down.sql")

	upBytes, err := os.ReadFile(upPath)
	if err == nil {
		migration007Up = string(upBytes)
	}
	downBytes, err := os.ReadFile(downPath)
	if err == nil {
		migration007Down = string(downBytes)
	}
}

func TestMigration007_UpAndDown(t *testing.T) {
	if db.Pool == nil {
		pool, err := db.Connect()
		if err != nil {
			t.Skipf("Postgres not available: %v", err)
		}
		defer pool.Close()
	}
	if migration007Up == "" || migration007Down == "" {
		t.Skip("Migration 007 files not found — skipping")
	}

	ctx := context.Background()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, migration007Down); err != nil {
		t.Fatalf("initial down migration failed: %v", err)
	}

	// --- Apply UP migration ---
	if _, err := tx.Exec(ctx, migration007Up); err != nil {
		t.Fatalf("up migration failed: %v", err)
	}

	// Verify login_history table exists
	var tableExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'login_history'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("check login_history exists: %v", err)
	}
	if !tableExists {
		t.Error("login_history table not found after up migration")
	}

	// Verify system_settings table exists
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'system_settings'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("check system_settings exists: %v", err)
	}
	if !tableExists {
		t.Error("system_settings table not found after up migration")
	}

	// Verify indexes exist on login_history
	var idxCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_indexes
		WHERE tablename = 'login_history'
		AND indexname IN ('idx_login_history_admin_id_created_at', 'idx_login_history_created_at')
	`).Scan(&idxCount)
	if err != nil {
		t.Fatalf("check indexes: %v", err)
	}
	if idxCount != 2 {
		t.Errorf("expected 2 indexes on login_history, got %d", idxCount)
	}

	// Verify seed data
	var whoisValue string
	err = tx.QueryRow(ctx, `SELECT value FROM system_settings WHERE key = 'whois_server'`).Scan(&whoisValue)
	if err != nil {
		t.Fatalf("read whois_server seed: %v", err)
	}
	if whoisValue != "whois.pwhois.org" {
		t.Errorf("expected whois_server='whois.pwhois.org', got %q", whoisValue)
	}

	// --- Apply DOWN migration ---
	if _, err := tx.Exec(ctx, migration007Down); err != nil {
		t.Fatalf("down migration failed: %v", err)
	}

	// Verify login_history table is gone
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'login_history'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("check login_history dropped: %v", err)
	}
	if tableExists {
		t.Error("login_history table still exists after down migration")
	}

	// Verify system_settings table is gone
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'system_settings'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("check system_settings dropped: %v", err)
	}
	if tableExists {
		t.Error("system_settings table still exists after down migration")
	}

	// Verify indexes are gone
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_indexes
		WHERE tablename = 'login_history'
		AND indexname IN ('idx_login_history_admin_id_created_at', 'idx_login_history_created_at')
	`).Scan(&idxCount)
	if err != nil {
		t.Fatalf("check indexes dropped: %v", err)
	}
	if idxCount != 0 {
		t.Errorf("expected 0 indexes on login_history after down, got %d", idxCount)
	}
}

// --- LoginHistory service test helpers ---

const testLoginAdminPrefix = "test-login-"

func createTestAdmin(t *testing.T, role string) *models.Admin {
	t.Helper()
	username := testLoginAdminPrefix + uuid.New().String()[:8]
	email := username + "@test.local"
	admin, err := CreateAdmin(models.CreateAdminRequest{
		Username:    username,
		Email:       &email,
		Password:    "testpass123",
		DisplayName: nil,
		Role:        role,
	})
	if err != nil {
		t.Fatalf("CreateAdmin(%s): %v", role, err)
	}
	return admin
}

func cleanupTestLoginAdmins(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM login_history WHERE admin_id IN (SELECT id FROM admins WHERE username LIKE $1)`,
		testLoginAdminPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup login_history: %v", err)
	}
	_, err = db.Pool.Exec(context.Background(),
		`DELETE FROM admins WHERE username LIKE $1`,
		testLoginAdminPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup admins: %v", err)
	}
}

// cleanupLoginHistoryForAdmin removes all login history for a specific admin.
func cleanupLoginHistoryForAdmin(t *testing.T, adminID uuid.UUID) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM login_history WHERE admin_id = $1`, adminID)
	if err != nil {
		t.Fatalf("cleanup login_history for admin: %v", err)
	}
}

func cleanupAllLoginHistory(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `DELETE FROM login_history`)
	if err != nil {
		t.Fatalf("cleanup all login_history: %v", err)
	}
}

// --- parseUserAgent tests ---

func TestParseUserAgent_ChromeDesktop(t *testing.T) {
	browser, osName, deviceType := parseUserAgent(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	)
	if browser != "Chrome" {
		t.Errorf("expected browser 'Chrome', got %q", browser)
	}
	if osName != "Windows" {
		t.Errorf("expected os 'Windows', got %q", osName)
	}
	if deviceType != "desktop" {
		t.Errorf("expected deviceType 'desktop', got %q", deviceType)
	}
}

func TestParseUserAgent_FirefoxMac(t *testing.T) {
	browser, osName, deviceType := parseUserAgent(
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0",
	)
	if browser != "Firefox" {
		t.Errorf("expected browser 'Firefox', got %q", browser)
	}
	if osName != "macOS" {
		t.Errorf("expected os 'macOS', got %q", osName)
	}
	if deviceType != "desktop" {
		t.Errorf("expected deviceType 'desktop', got %q", deviceType)
	}
}

func TestParseUserAgent_SafariIPhone(t *testing.T) {
	browser, osName, deviceType := parseUserAgent(
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
	)
	if browser != "Safari" {
		t.Errorf("expected browser 'Safari', got %q", browser)
	}
	if osName != "iOS" {
		t.Errorf("expected os 'iOS', got %q", osName)
	}
	if deviceType != "mobile" {
		t.Errorf("expected deviceType 'mobile', got %q", deviceType)
	}
}

func TestParseUserAgent_InstagramInApp(t *testing.T) {
	browser, osName, deviceType := parseUserAgent(
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 300.0.0.0.100",
	)
	if browser != "Instagram" {
		t.Errorf("expected browser 'Instagram', got %q", browser)
	}
	if osName != "iOS" {
		t.Errorf("expected os 'iOS', got %q", osName)
	}
	if deviceType != "mobile" {
		t.Errorf("expected deviceType 'mobile', got %q", deviceType)
	}
}

func TestParseUserAgent_AndroidChrome(t *testing.T) {
	browser, osName, deviceType := parseUserAgent(
		"Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
	)
	if browser != "Chrome" {
		t.Errorf("expected browser 'Chrome', got %q", browser)
	}
	if osName != "Android" {
		t.Errorf("expected os 'Android', got %q", osName)
	}
	if deviceType != "mobile" {
		t.Errorf("expected deviceType 'mobile', got %q", deviceType)
	}
}

func TestParseUserAgent_EdgeDesktop(t *testing.T) {
	browser, osName, deviceType := parseUserAgent(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	)
	if browser != "Edge" {
		t.Errorf("expected browser 'Edge', got %q", browser)
	}
	if osName != "Windows" {
		t.Errorf("expected os 'Windows', got %q", osName)
	}
	if deviceType != "desktop" {
		t.Errorf("expected deviceType 'desktop', got %q", deviceType)
	}
}

func TestParseUserAgent_IPadSafari(t *testing.T) {
	browser, osName, deviceType := parseUserAgent(
		"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
	)
	if browser != "Safari" {
		t.Errorf("expected browser 'Safari', got %q", browser)
	}
	if osName != "iPadOS" {
		t.Errorf("expected os 'iPadOS', got %q", osName)
	}
	if deviceType != "tablet" {
		t.Errorf("expected deviceType 'tablet', got %q", deviceType)
	}
}

func TestParseUserAgent_Empty(t *testing.T) {
	browser, osName, deviceType := parseUserAgent("")
	if browser != "Unknown" {
		t.Errorf("expected browser 'Unknown', got %q", browser)
	}
	if osName != "Unknown" {
		t.Errorf("expected os 'Unknown', got %q", osName)
	}
	if deviceType != "unknown" {
		t.Errorf("expected deviceType 'unknown', got %q", deviceType)
	}
}

// --- RecordLogin tests ---

func TestRecordLogin_Success(t *testing.T) {
	defer cleanupTestLoginAdmins(t)

	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 300.0.0.0.100"

	_, err := RecordLogin(admin.ID.String(), "203.0.113.1", ua)
	if err != nil {
		t.Fatalf("RecordLogin: %v", err)
	}

	// Verify the record was inserted
	var count int
	err = db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM login_history WHERE admin_id = $1`, admin.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count login_history: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 login history record, got %d", count)
	}

	// Verify parsed fields
	var browser, osName, deviceType string
	err = db.Pool.QueryRow(context.Background(),
		`SELECT browser, os, device_type FROM login_history WHERE admin_id = $1`, admin.ID).
		Scan(&browser, &osName, &deviceType)
	if err != nil {
		t.Fatalf("fetch login_history: %v", err)
	}
	if browser != "Instagram" {
		t.Errorf("expected browser 'Instagram', got %q", browser)
	}
	if osName != "iOS" {
		t.Errorf("expected os 'iOS', got %q", osName)
	}
	if deviceType != "mobile" {
		t.Errorf("expected deviceType 'mobile', got %q", deviceType)
	}
}

func TestRecordLogin_EmptyUserAgent(t *testing.T) {
	defer cleanupTestLoginAdmins(t)

	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	_, err := RecordLogin(admin.ID.String(), "192.168.1.1", "")
	if err != nil {
		t.Fatalf("RecordLogin with empty UA: %v", err)
	}

	var browser, osName, deviceType string
	err = db.Pool.QueryRow(context.Background(),
		`SELECT browser, os, device_type FROM login_history WHERE admin_id = $1`, admin.ID).
		Scan(&browser, &osName, &deviceType)
	if err != nil {
		t.Fatalf("fetch login_history: %v", err)
	}
	if browser != "Unknown" {
		t.Errorf("expected browser 'Unknown', got %q", browser)
	}
	if osName != "Unknown" {
		t.Errorf("expected os 'Unknown', got %q", osName)
	}
	if deviceType != "unknown" {
		t.Errorf("expected deviceType 'unknown', got %q", deviceType)
	}
}

func TestRecordLogin_InvalidAdminID(t *testing.T) {
	_, err := RecordLogin("not-a-uuid", "10.0.0.1", "TestAgent/1.0")
	if err == nil {
		t.Fatal("expected error for invalid admin ID, got nil")
	}
}

func TestRecordLogin_NonExistentAdmin(t *testing.T) {
	_, err := RecordLogin(uuid.New().String(), "10.0.0.1", "TestAgent/1.0")
	if err == nil {
		t.Fatal("expected error for non-existent admin, got nil")
	}
}

// --- GetLoginHistory tests ---

func TestGetLoginHistory_Empty(t *testing.T) {
	defer cleanupTestLoginAdmins(t)

	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	records, total, err := GetLoginHistory(admin.ID, 1, 10)
	if err != nil {
		t.Fatalf("GetLoginHistory: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestGetLoginHistory_Paginated(t *testing.T) {
	defer cleanupTestLoginAdmins(t)

	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	// Insert 5 login records
	adminIDStr := admin.ID.String()
	for i := 0; i < 5; i++ {
		if _, err := RecordLogin(adminIDStr, "203.0.113."+string(rune('1'+i%9)),
			"Mozilla/5.0 TestAgent/"+string(rune('1'+i))); err != nil {
			t.Fatalf("RecordLogin %d: %v", i, err)
		}
	}

	// Page 1: limit 3 → should return 3 records, total 5
	records, total, err := GetLoginHistory(admin.ID, 1, 3)
	if err != nil {
		t.Fatalf("GetLoginHistory page 1: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records on page 1, got %d", len(records))
	}

	// Page 2: limit 3 → should return 2 records, total 5
	records, total, err = GetLoginHistory(admin.ID, 2, 3)
	if err != nil {
		t.Fatalf("GetLoginHistory page 2: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records on page 2, got %d", len(records))
	}

	// Page 3: limit 3 → should return 0 records, total 5
	records, total, err = GetLoginHistory(admin.ID, 3, 3)
	if err != nil {
		t.Fatalf("GetLoginHistory page 3: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records on page 3, got %d", len(records))
	}

	// Verify ordering: most recent first
	records, _, err = GetLoginHistory(admin.ID, 1, 5)
	if err != nil {
		t.Fatalf("GetLoginHistory all: %v", err)
	}
	for i := 1; i < len(records); i++ {
		if records[i-1].CreatedAt.Before(records[i].CreatedAt) {
			t.Error("records should be ordered by created_at DESC")
			break
		}
	}
}

func TestGetLoginHistory_DefaultLimit(t *testing.T) {
	defer cleanupTestLoginAdmins(t)

	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	// Insert 2 records
	adminIDStr := admin.ID.String()
	for i := 0; i < 2; i++ {
		if _, err := RecordLogin(adminIDStr, "203.0.113."+string(rune('1'+i)),
			"Mozilla/5.0"); err != nil {
			t.Fatalf("RecordLogin %d: %v", i, err)
		}
	}

	// Page 0 and limit 0 should be clamped to safe defaults
	records, total, err := GetLoginHistory(admin.ID, 0, 0)
	if err != nil {
		t.Fatalf("GetLoginHistory with defaults: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records (page clamped), got %d", len(records))
	}
}

// --- GetAllLoginHistory tests ---

func TestGetAllLoginHistory_SuperAdminSeesAll(t *testing.T) {
	cleanupAllLoginHistory(t)
	defer cleanupTestLoginAdmins(t)

	superAdmin := createTestAdmin(t, models.RoleSuperAdmin)
	admin := createTestAdmin(t, models.RoleAdmin)
	waffleManager := createTestAdmin(t, models.RoleWaffleManager)
	defer cleanupLoginHistoryForAdmin(t, superAdmin.ID)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)
	defer cleanupLoginHistoryForAdmin(t, waffleManager.ID)

	// Record logins for each
	_, _ = RecordLogin(superAdmin.ID.String(), "203.0.113.1", "SuperAgent/1.0")
	_, _ = RecordLogin(admin.ID.String(), "203.0.113.2", "AdminAgent/1.0")
	_, _ = RecordLogin(waffleManager.ID.String(), "203.0.113.3", "ManagerAgent/1.0")

	records, total, err := GetAllLoginHistory(models.RoleSuperAdmin, superAdmin.ID, 1, 20)
	if err != nil {
		t.Fatalf("GetAllLoginHistory super_admin: %v", err)
	}
	if total != 3 {
		t.Errorf("super_admin: expected total 3, got %d", total)
	}
	if len(records) != 3 {
		t.Errorf("super_admin: expected 3 records, got %d", len(records))
	}
}

func TestGetAllLoginHistory_AdminSeesSelfAndWaffleManagers(t *testing.T) {
	cleanupAllLoginHistory(t)
	defer cleanupTestLoginAdmins(t)

	viewerAdmin := createTestAdmin(t, models.RoleAdmin)
	superAdmin := createTestAdmin(t, models.RoleSuperAdmin)
	otherAdmin := createTestAdmin(t, models.RoleAdmin)
	waffleManager := createTestAdmin(t, models.RoleWaffleManager)
	defer cleanupLoginHistoryForAdmin(t, viewerAdmin.ID)
	defer cleanupLoginHistoryForAdmin(t, superAdmin.ID)
	defer cleanupLoginHistoryForAdmin(t, otherAdmin.ID)
	defer cleanupLoginHistoryForAdmin(t, waffleManager.ID)

	// Record logins for each
	_, _ = RecordLogin(viewerAdmin.ID.String(), "203.0.113.1", "ViewerAgent/1.0")
	_, _ = RecordLogin(superAdmin.ID.String(), "203.0.113.2", "SuperAgent/1.0")
	_, _ = RecordLogin(otherAdmin.ID.String(), "203.0.113.3", "OtherAgent/1.0")
	_, _ = RecordLogin(waffleManager.ID.String(), "203.0.113.4", "ManagerAgent/1.0")

	records, total, err := GetAllLoginHistory(models.RoleAdmin, viewerAdmin.ID, 1, 20)
	if err != nil {
		t.Fatalf("GetAllLoginHistory admin: %v", err)
	}
	// admin sees self + waffle_manager (not super_admin, not other admin)
	if total != 2 {
		t.Errorf("admin: expected total 2 (self + waffle_manager), got %d", total)
	}
	for _, r := range records {
		if r.AdminID != viewerAdmin.ID && r.AdminID != waffleManager.ID {
			t.Errorf("admin: unexpected admin_id in results: %v", r.AdminID)
		}
	}
}

func TestGetAllLoginHistory_WaffleManagerSeesSelfOnly(t *testing.T) {
	cleanupAllLoginHistory(t)
	defer cleanupTestLoginAdmins(t)

	waffleManager := createTestAdmin(t, models.RoleWaffleManager)
	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, waffleManager.ID)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	_, _ = RecordLogin(waffleManager.ID.String(), "203.0.113.1", "ManagerAgent/1.0")
	_, _ = RecordLogin(admin.ID.String(), "203.0.113.2", "AdminAgent/1.0")

	records, total, err := GetAllLoginHistory(models.RoleWaffleManager, waffleManager.ID, 1, 20)
	if err != nil {
		t.Fatalf("GetAllLoginHistory waffle_manager: %v", err)
	}
	if total != 1 {
		t.Errorf("waffle_manager: expected total 1, got %d", total)
	}
	if len(records) == 1 && records[0].AdminID != waffleManager.ID {
		t.Errorf("waffle_manager: expected own admin_id, got %v", records[0].AdminID)
	}
}

// --- EnrichLoginWithWHOIS tests ---

func TestEnrichLoginWithWHOIS_NoServerConfigured(t *testing.T) {
	defer cleanupTestLoginAdmins(t)

	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	// Record a login
	_, err := RecordLogin(admin.ID.String(), "8.8.8.8", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("RecordLogin: %v", err)
	}

	// Get the login record ID
	var loginID uuid.UUID
	err = db.Pool.QueryRow(context.Background(),
		`SELECT id FROM login_history WHERE admin_id = $1 ORDER BY created_at DESC LIMIT 1`,
		admin.ID).Scan(&loginID)
	if err != nil {
		t.Fatalf("fetch login ID: %v", err)
	}

	// Remove whois_server setting
	_, err = db.Pool.Exec(context.Background(),
		`DELETE FROM system_settings WHERE key = 'whois_server'`)
	if err != nil {
		t.Fatalf("delete whois_server setting: %v", err)
	}

	// Enrich should succeed (no-op) with no error
	err = EnrichLoginWithWHOIS(loginID)
	if err != nil {
		t.Fatalf("EnrichLoginWithWHOIS (no server): %v", err)
	}
}

func TestEnrichLoginWithWHOIS_RFC1918(t *testing.T) {
	defer cleanupTestLoginAdmins(t)

	admin := createTestAdmin(t, models.RoleAdmin)
	defer cleanupLoginHistoryForAdmin(t, admin.ID)

	// Record login with private IP
	_, err := RecordLogin(admin.ID.String(), "192.168.1.100", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("RecordLogin: %v", err)
	}

	var loginID uuid.UUID
	err = db.Pool.QueryRow(context.Background(),
		`SELECT id FROM login_history WHERE admin_id = $1 ORDER BY created_at DESC LIMIT 1`,
		admin.ID).Scan(&loginID)
	if err != nil {
		t.Fatalf("fetch login ID: %v", err)
	}

	// Ensure whois_server exists
	_, err = db.Pool.Exec(context.Background(),
		`INSERT INTO system_settings (key, value) VALUES ('whois_server', 'whois.pwhois.org')
		 ON CONFLICT (key) DO UPDATE SET value = 'whois.pwhois.org'`)
	if err != nil {
		t.Fatalf("upsert whois_server: %v", err)
	}

	// Enrich should skip RFC1918 IP (no error, no-op)
	err = EnrichLoginWithWHOIS(loginID)
	if err != nil {
		t.Fatalf("EnrichLoginWithWHOIS (RFC1918): got error: %v (expected skip)", err)
	}
}

func TestEnrichLoginWithWHOIS_InvalidLoginID(t *testing.T) {
	err := EnrichLoginWithWHOIS(uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent login ID, got nil")
	}
}
