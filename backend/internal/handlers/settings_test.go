package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/middleware"
	"github.com/syrup/backend/internal/services"
)

var ensureSettingsTableOnce sync.Once

func createTokenWithID(id, role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin_id": id,
		"email":    "test@syrup.test",
		"role":     role,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))
	return tokenString
}

func doRequestWithID(r *gin.Engine, method, url, adminID, role string, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	if role != "" {
		req.AddCookie(&http.Cookie{Name: "admin_token", Value: createTokenWithID(adminID, role)})
	}
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateTimezoneAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/timezone", UpdateTimezoneAPI)

	body, _ := json.Marshal(map[string]string{"timezone": "America/New_York"})
	req := httptest.NewRequest("PATCH", "/api/admin/me/timezone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTimezoneAPI_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/timezone", func(c *gin.Context) {
		c.Set("admin_id", "not-a-uuid")
		UpdateTimezoneAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"timezone": "America/New_York"})
	req := httptest.NewRequest("PATCH", "/api/admin/me/timezone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTimezoneAPI_InvalidTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/timezone", func(c *gin.Context) {
		c.Set("admin_id", "550e8400-e29b-41d4-a716-446655440000")
		UpdateTimezoneAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"timezone": "NotA/RealZone"})
	req := httptest.NewRequest("PATCH", "/api/admin/me/timezone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "invalid timezone" {
		t.Fatalf("expected 'invalid timezone' error, got %s", resp["error"])
	}
}

func TestUpdateTimezoneAPI_EmptyTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/timezone", func(c *gin.Context) {
		c.Set("admin_id", "550e8400-e29b-41d4-a716-446655440000")
		UpdateTimezoneAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"timezone": ""})
	req := httptest.NewRequest("PATCH", "/api/admin/me/timezone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "timezone is required" {
		t.Fatalf("expected 'timezone is required' error, got %s", resp["error"])
	}
}

func TestUpdateTimezoneAPI_ValidTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Skip("skipping: requires database connection")
}

func TestGetWhoisSettingsAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/settings/whois-server", GetWhoisSettingsAPI)

	req := httptest.NewRequest("GET", "/api/admin/settings/whois-server", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetWhoisSettingsAPI_Success(t *testing.T) {
	t.Skip("skipping: requires database connection")
}

func TestUpdateWhoisSettingsAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/whois-server", UpdateWhoisSettingsAPI)

	body, _ := json.Marshal(map[string]string{"whois_server": "whois.example.com"})
	req := httptest.NewRequest("PATCH", "/api/admin/settings/whois-server", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateWhoisSettingsAPI_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/whois-server", func(c *gin.Context) {
		c.Set("admin_id", "not-a-uuid")
		UpdateWhoisSettingsAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"whois_server": "whois.example.com"})
	req := httptest.NewRequest("PATCH", "/api/admin/settings/whois-server", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateWhoisSettingsAPI_EmptyWhoisServer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/whois-server", func(c *gin.Context) {
		c.Set("admin_id", "550e8400-e29b-41d4-a716-446655440000")
		UpdateWhoisSettingsAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"whois_server": ""})
	req := httptest.NewRequest("PATCH", "/api/admin/settings/whois-server", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "whois_server is required" {
		t.Fatalf("expected 'whois_server is required' error, got %s", resp["error"])
	}
}

func TestUpdateWhoisSettingsAPI_Valid(t *testing.T) {
	t.Skip("skipping: requires database connection")
}

// ---------------------------------------------------------------------------
// Generic settings API tests (GetAllSettingsAPI / UpdateSettingAPI)
// ---------------------------------------------------------------------------

// setupSettingsDB connects to DB, ensures table, creates test admin. Skips if DB unavailable.
func setupSettingsDB(t *testing.T) uuid.UUID {
	t.Helper()
	if db.Pool == nil {
		_, err := db.Connect()
		if err != nil {
			t.Skipf("Postgres not available: %v", err)
		}
	}

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
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO system_settings (key, value, updated_at)
			VALUES ('whois_server', 'whois.pwhois.org', NOW())
			ON CONFLICT (key) DO NOTHING
		`)
		if err != nil {
			t.Fatalf("seed whois_server: %v", err)
		}
	})

	id := uuid.New()
	username := "test-gs-" + id.String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, username, username+"@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test admin: %v", err)
	}
	return id
}

func cleanupSettingsTest(t *testing.T, adminID uuid.UUID) {
	t.Helper()
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM system_settings WHERE key LIKE 'test-%'`)
	_, _ = db.Pool.Exec(context.Background(), `
		INSERT INTO system_settings (key, value, updated_at, updated_by)
		VALUES ('whois_server', 'whois.pwhois.org', NOW(), NULL)
		ON CONFLICT (key) DO UPDATE SET value = 'whois.pwhois.org', updated_at = NOW(), updated_by = NULL
	`)
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, adminID)
}

// ---------------------------------------------------------------------------
// GetAllSettingsAPI tests
// ---------------------------------------------------------------------------

func TestGetAllSettingsAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/settings", middleware.RequireAuth, middleware.RequireSuperAdmin, GetAllSettingsAPI)

	req := httptest.NewRequest("GET", "/api/admin/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAllSettingsAPI_NonSuperAdmin_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/settings", middleware.RequireAuth, middleware.RequireSuperAdmin, GetAllSettingsAPI)

	w := doRequest(r, "GET", "/api/admin/settings", "admin", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for admin role, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAllSettingsAPI_ReturnsAllSettings(t *testing.T) {
	adminID := setupSettingsDB(t)
	defer cleanupSettingsTest(t, adminID)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/settings", middleware.RequireAuth, middleware.RequireSuperAdmin, GetAllSettingsAPI)

	w := doRequest(r, "GET", "/api/admin/settings", "super_admin", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Settings []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Settings) != 5 {
		t.Fatalf("expected 5 settings, got %d", len(resp.Settings))
	}

	keys := map[string]bool{}
	for _, s := range resp.Settings {
		keys[s.Key] = true
	}
	expected := []string{"whois_server", "jwt_expiration_hours", "password_min_length", "audit_retention_days", "login_history_retention_days"}
	for _, k := range expected {
		if !keys[k] {
			t.Errorf("missing expected key %q", k)
		}
	}
}

// ---------------------------------------------------------------------------
// UpdateSettingAPI tests
// ---------------------------------------------------------------------------

func TestUpdateSettingAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/:key", middleware.RequireAuth, middleware.RequireSuperAdmin, UpdateSettingAPI)

	body, _ := json.Marshal(map[string]string{"value": "24"})
	req := httptest.NewRequest("PATCH", "/api/admin/settings/jwt_expiration_hours", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSettingAPI_NonSuperAdmin_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/:key", middleware.RequireAuth, middleware.RequireSuperAdmin, UpdateSettingAPI)

	w := doRequest(r, "PATCH", "/api/admin/settings/jwt_expiration_hours", "admin", []byte(`{"value":"24"}`))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for admin role, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSettingAPI_RejectsInvalidKey(t *testing.T) {
	adminID := setupSettingsDB(t)
	defer cleanupSettingsTest(t, adminID)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/:key", middleware.RequireAuth, middleware.RequireSuperAdmin, UpdateSettingAPI)

	w := doRequestWithID(r, "PATCH", "/api/admin/settings/invalid_key", adminID.String(), "super_admin", []byte(`{"value":"some-value"}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid key, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatal("expected error message in response body")
	}
}

func TestUpdateSettingAPI_RejectsInvalidRetentionDays(t *testing.T) {
	adminID := setupSettingsDB(t)
	defer cleanupSettingsTest(t, adminID)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/:key", middleware.RequireAuth, middleware.RequireSuperAdmin, UpdateSettingAPI)

	w := doRequestWithID(r, "PATCH", "/api/admin/settings/audit_retention_days", adminID.String(), "super_admin", []byte(`{"value":"500"}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for retention_days > 365, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatal("expected error message in response body")
	}
}

func TestUpdateSettingAPI_Success(t *testing.T) {
	adminID := setupSettingsDB(t)
	defer cleanupSettingsTest(t, adminID)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/settings/:key", middleware.RequireAuth, middleware.RequireSuperAdmin, UpdateSettingAPI)

	err := services.SetSetting("whois_server", "whois.test.example.com", adminID)
	if err != nil {
		t.Fatalf("seed test setting: %v", err)
	}

	w := doRequestWithID(r, "PATCH", "/api/admin/settings/jwt_expiration_hours", adminID.String(), "super_admin", []byte(`{"value":"48"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["key"] != "jwt_expiration_hours" {
		t.Errorf("expected key %q, got %q", "jwt_expiration_hours", resp["key"])
	}
	if resp["value"] != "48" {
		t.Errorf("expected value %q, got %q", "48", resp["value"])
	}

	val, err := services.GetSetting("jwt_expiration_hours")
	if err != nil {
		t.Fatalf("failed to read back setting: %v", err)
	}
	if val != "48" {
		t.Errorf("expected persisted value %q, got %q", "48", val)
	}
}
