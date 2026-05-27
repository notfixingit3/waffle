package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
