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
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/middleware"
)

var ensureShareMessagesTestOnce sync.Once

// setupShareMessagesDB connects to DB, ensures tables, creates test admin, waffle, and template.
// Returns adminID, waffleSlug, templateID.
func setupShareMessagesDB(t *testing.T) (uuid.UUID, string, uuid.UUID) {
	t.Helper()
	if db.Pool == nil {
		_, err := db.Connect()
		if err != nil {
			t.Skipf("Postgres not available: %v", err)
		}
	}

	ensureShareMessagesTestOnce.Do(func() {
		ctx := context.Background()
		// Ensure the message_templates table exists
		_, err := db.Pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS message_templates (
				id UUID PRIMARY KEY,
				name TEXT NOT NULL,
				body TEXT NOT NULL,
				is_default BOOLEAN NOT NULL DEFAULT false,
				created_by UUID REFERENCES admins(id),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			)
		`)
		if err != nil {
			t.Fatalf("create message_templates table: %v", err)
		}
	})

	adminID := uuid.New()
	username := "test-sm-" + adminID.String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, adminID, username, username+"@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test admin: %v", err)
	}

	// Create a template
	tmplID := uuid.New()
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO message_templates (id, name, body, is_default, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, tmplID, "Test Template", "Test body {item} {price} {total_spots} {spots_left} {spots_claimed} {url}", true, adminID, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}

	// Create a waffle with the template assigned
	waffleID := uuid.New()
	waffleSlug := "test-waffle-" + waffleID.String()[:8]
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO waffles (id, slug, title, description, total_spots, spot_price, status, share_template_id, share_message, created_at, item_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, waffleID, waffleSlug, "Test Waffle", "A test waffle", 25, 5, "active", tmplID, "Test body Test Waffle 5 25 25 0 https://waffle.social/waffle/"+waffleSlug, time.Now(), 1)
	if err != nil {
		t.Fatalf("create test waffle: %v", err)
	}

	return adminID, waffleSlug, tmplID
}

func cleanupShareMessagesTest(t *testing.T, adminID uuid.UUID, waffleSlug string) {
	t.Helper()
	// Delete waffle by slug
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM spots WHERE waffle_id = (SELECT id FROM waffles WHERE slug = $1)`, waffleSlug)
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM waffles WHERE slug = $1`, waffleSlug)
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE created_by = $1`, adminID)
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, adminID)
}

// ---------------------------------------------------------------------------
// Auth required tests
// ---------------------------------------------------------------------------

func TestGetWaffleShareMessageAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/waffles/:id/share-message", middleware.RequireAuth, GetWaffleShareMessageAPI)

	req := httptest.NewRequest("GET", "/api/admin/waffles/"+uuid.New().String()+"/share-message", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateWaffleShareMessageAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/waffles/:id/share-message", middleware.RequireAuth, UpdateWaffleShareMessageAPI)

	body, _ := json.Marshal(map[string]string{"template_id": uuid.New().String()})
	req := httptest.NewRequest("PATCH", "/api/admin/waffles/"+uuid.New().String()+"/share-message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenderWaffleShareMessageAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/waffles/:id/share-message/render", middleware.RequireAuth, RenderWaffleShareMessageAPI)

	body, _ := json.Marshal(map[string]string{"template_id": uuid.New().String()})
	req := httptest.NewRequest("POST", "/api/admin/waffles/"+uuid.New().String()+"/share-message/render", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET happy path
// ---------------------------------------------------------------------------

func TestGetWaffleShareMessageAPI_Success(t *testing.T) {
	adminID, waffleSlug, tmplID := setupShareMessagesDB(t)
	defer cleanupShareMessagesTest(t, adminID, waffleSlug)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/waffles/:id/share-message", middleware.RequireAuth, GetWaffleShareMessageAPI)

	w := doRequestWithID(r, "GET", "/api/admin/waffles/"+waffleSlug+"/share-message", adminID.String(), "admin", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message            *string    `json:"message"`
		Templates          []any      `json:"templates"`
		SelectedTemplateID *uuid.UUID `json:"selected_template_id"`
		Waffle             struct {
			ID    uuid.UUID `json:"id"`
			Slug  string    `json:"slug"`
			Title string    `json:"title"`
		} `json:"waffle"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message == nil || *resp.Message == "" {
		t.Fatal("expected non-empty share message")
	}
	if len(resp.Templates) == 0 {
		t.Fatal("expected at least one template")
	}
	if resp.SelectedTemplateID == nil || *resp.SelectedTemplateID != tmplID {
		t.Fatalf("expected selected_template_id %s, got %v", tmplID, resp.SelectedTemplateID)
	}
	if resp.Waffle.Slug != waffleSlug {
		t.Fatalf("expected waffle slug %s, got %s", waffleSlug, resp.Waffle.Slug)
	}
	if resp.Waffle.Title != "Test Waffle" {
		t.Fatalf("expected waffle title 'Test Waffle', got %s", resp.Waffle.Title)
	}
}

func TestGetWaffleShareMessageAPI_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/waffles/:id/share-message", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		GetWaffleShareMessageAPI(c)
	})

	req := httptest.NewRequest("GET", "/api/admin/waffles/nonexistent-slug/share-message", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// PATCH happy path
// ---------------------------------------------------------------------------

func TestUpdateWaffleShareMessageAPI_UpdateTemplate(t *testing.T) {
	adminID, waffleSlug, _ := setupShareMessagesDB(t)
	defer cleanupShareMessagesTest(t, adminID, waffleSlug)

	// Create a second template to switch to
	tmpl2ID := uuid.New()
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO message_templates (id, name, body, is_default, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, tmpl2ID, "Second Template", "Second body {item}", false, adminID, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create second template: %v", err)
	}

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/waffles/:id/share-message", middleware.RequireAuth, UpdateWaffleShareMessageAPI)

	body, _ := json.Marshal(map[string]string{"template_id": tmpl2ID.String()})
	w := doRequestWithID(r, "PATCH", "/api/admin/waffles/"+waffleSlug+"/share-message", adminID.String(), "admin", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message            *string    `json:"message"`
		SelectedTemplateID *uuid.UUID `json:"selected_template_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.SelectedTemplateID == nil || *resp.SelectedTemplateID != tmpl2ID {
		t.Fatalf("expected selected_template_id %s, got %v", tmpl2ID, resp.SelectedTemplateID)
	}
	// Message should have been re-rendered with the new template
	if resp.Message == nil || *resp.Message == "" {
		t.Fatal("expected non-empty re-rendered message")
	}
	if *resp.Message != "Second body Test Waffle" {
		t.Fatalf("expected re-rendered message 'Second body Test Waffle', got %q", *resp.Message)
	}
}

func TestUpdateWaffleShareMessageAPI_UpdateCustomMessage(t *testing.T) {
	adminID, waffleSlug, _ := setupShareMessagesDB(t)
	defer cleanupShareMessagesTest(t, adminID, waffleSlug)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/waffles/:id/share-message", middleware.RequireAuth, UpdateWaffleShareMessageAPI)

	customMsg := "Custom share message for this waffle!"
	body, _ := json.Marshal(map[string]string{"message": customMsg})
	w := doRequestWithID(r, "PATCH", "/api/admin/waffles/"+waffleSlug+"/share-message", adminID.String(), "admin", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message == nil || *resp.Message != customMsg {
		t.Fatalf("expected message %q, got %q", customMsg, *resp.Message)
	}
}

func TestUpdateWaffleShareMessageAPI_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/waffles/:id/share-message", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		UpdateWaffleShareMessageAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"template_id": uuid.New().String()})
	req := httptest.NewRequest("PATCH", "/api/admin/waffles/nonexistent-slug/share-message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Render (preview) happy path
// ---------------------------------------------------------------------------

func TestRenderWaffleShareMessageAPI_Success(t *testing.T) {
	adminID, waffleSlug, tmplID := setupShareMessagesDB(t)
	defer cleanupShareMessagesTest(t, adminID, waffleSlug)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/waffles/:id/share-message/render", middleware.RequireAuth, RenderWaffleShareMessageAPI)

	body, _ := json.Marshal(map[string]string{"template_id": tmplID.String()})
	w := doRequestWithID(r, "POST", "/api/admin/waffles/"+waffleSlug+"/share-message/render", adminID.String(), "admin", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message == "" {
		t.Fatal("expected non-empty rendered message")
	}
	// Should contain the waffle title
	if resp.Message != "Test body Test Waffle 5 25 25 0 https://waffle.social/waffle/"+waffleSlug {
		t.Fatalf("unexpected rendered message: %q", resp.Message)
	}
}

func TestRenderWaffleShareMessageAPI_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/waffles/:id/share-message/render", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		RenderWaffleShareMessageAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"template_id": uuid.New().String()})
	req := httptest.NewRequest("POST", "/api/admin/waffles/nonexistent-slug/share-message/render", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenderWaffleShareMessageAPI_TemplateNotFound(t *testing.T) {
	adminID, waffleSlug, _ := setupShareMessagesDB(t)
	defer cleanupShareMessagesTest(t, adminID, waffleSlug)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/waffles/:id/share-message/render", middleware.RequireAuth, RenderWaffleShareMessageAPI)

	nonExistentID := uuid.New()
	body, _ := json.Marshal(map[string]string{"template_id": nonExistentID.String()})
	w := doRequestWithID(r, "POST", "/api/admin/waffles/"+waffleSlug+"/share-message/render", adminID.String(), "admin", body)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent template, got %d: %s", w.Code, w.Body.String())
	}
}
