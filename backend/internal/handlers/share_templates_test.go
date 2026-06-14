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
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/services"
)

var ensureTemplatesTableOnce sync.Once

// setupShareTemplatesDB connects to DB, ensures table, creates test admin and a seed template.
// Returns adminID. Skips if DB unavailable.
func setupShareTemplatesDB(t *testing.T) uuid.UUID {
	t.Helper()
	if db.Pool == nil {
		_, err := db.Connect()
		if err != nil {
			t.Skipf("Postgres not available: %v", err)
		}
	}

	ensureTemplatesTableOnce.Do(func() {
		ctx := context.Background()
		// Ensure the message_templates table exists (from migration 015)
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

	id := uuid.New()
	username := "test-st-" + id.String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, username, username+"@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test admin: %v", err)
	}

	// Seed a default template so tests that need one can use it
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO message_templates (id, name, body, is_default, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, uuid.New(), "Test Default Template", "Test body {item}", true, id, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("seed default template: %v", err)
	}

	return id
}

func cleanupShareTemplatesTest(t *testing.T, adminID uuid.UUID) {
	t.Helper()
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE created_by = $1`, adminID)
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, adminID)
}

// ---------------------------------------------------------------------------
// Auth required tests (401 without cookie/token)
// ---------------------------------------------------------------------------

func TestListShareTemplatesAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/share-templates", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), ListShareTemplatesAPI)

	req := httptest.NewRequest("GET", "/api/admin/share-templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateShareTemplateAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), CreateShareTemplateAPI)

	body, _ := json.Marshal(map[string]string{"name": "Test", "body": "Body"})
	req := httptest.NewRequest("POST", "/api/admin/share-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateShareTemplateAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/share-templates/:id", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), UpdateShareTemplateAPI)

	body, _ := json.Marshal(map[string]string{"name": "Test", "body": "Body"})
	req := httptest.NewRequest("PATCH", "/api/admin/share-templates/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteShareTemplateAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.DELETE("/api/admin/share-templates/:id", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), DeleteShareTemplateAPI)

	req := httptest.NewRequest("DELETE", "/api/admin/share-templates/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetDefaultShareTemplateAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates/:id/default", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), SetDefaultShareTemplateAPI)

	req := httptest.NewRequest("POST", "/api/admin/share-templates/"+uuid.New().String()+"/default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Role gating tests (waffle_manager allowed, lower role blocked with 403)
// ---------------------------------------------------------------------------

func TestShareTemplatesAPI_WaffleManager_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/share-templates", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), ListShareTemplatesAPI)

	w := doRequest(r, "GET", "/api/admin/share-templates", models.RoleWaffleManager, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for waffle_manager, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareTemplatesAPI_InvalidRole_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/share-templates", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), ListShareTemplatesAPI)

	// Use a role that is not in the allowed list
	w := doRequest(r, "GET", "/api/admin/share-templates", "viewer", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for invalid role, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Validation error tests (empty name/body) — no DB needed
// ---------------------------------------------------------------------------

func TestCreateShareTemplateAPI_EmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		CreateShareTemplateAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"name": "", "body": "Some body"})
	req := httptest.NewRequest("POST", "/api/admin/share-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "template name is required" {
		t.Fatalf("expected 'template name is required', got %q", resp["error"])
	}
}

func TestCreateShareTemplateAPI_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		CreateShareTemplateAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"name": "Test", "body": ""})
	req := httptest.NewRequest("POST", "/api/admin/share-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "template body is required" {
		t.Fatalf("expected 'template body is required', got %q", resp["error"])
	}
}

func TestUpdateShareTemplateAPI_EmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/share-templates/:id", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Params = []gin.Param{{Key: "id", Value: uuid.New().String()}}
		UpdateShareTemplateAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"name": "", "body": "Some body"})
	req := httptest.NewRequest("PATCH", "/api/admin/share-templates/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "template name is required" {
		t.Fatalf("expected 'template name is required', got %q", resp["error"])
	}
}

func TestUpdateShareTemplateAPI_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/share-templates/:id", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Params = []gin.Param{{Key: "id", Value: uuid.New().String()}}
		UpdateShareTemplateAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"name": "Test", "body": ""})
	req := httptest.NewRequest("PATCH", "/api/admin/share-templates/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "template body is required" {
		t.Fatalf("expected 'template body is required', got %q", resp["error"])
	}
}

func TestUpdateShareTemplateAPI_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/share-templates/:id", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Params = []gin.Param{{Key: "id", Value: "not-a-uuid"}}
		UpdateShareTemplateAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"name": "Test", "body": "Body"})
	req := httptest.NewRequest("PATCH", "/api/admin/share-templates/not-a-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteShareTemplateAPI_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.DELETE("/api/admin/share-templates/:id", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Params = []gin.Param{{Key: "id", Value: "not-a-uuid"}}
		DeleteShareTemplateAPI(c)
	})

	req := httptest.NewRequest("DELETE", "/api/admin/share-templates/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetDefaultShareTemplateAPI_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates/:id/default", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Params = []gin.Param{{Key: "id", Value: "not-a-uuid"}}
		SetDefaultShareTemplateAPI(c)
	})

	req := httptest.NewRequest("POST", "/api/admin/share-templates/not-a-uuid/default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// CRUD happy path tests (require DB)
// ---------------------------------------------------------------------------

func TestShareTemplatesAPI_ListTemplates(t *testing.T) {
	adminID := setupShareTemplatesDB(t)
	defer cleanupShareTemplatesTest(t, adminID)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/admin/share-templates", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), ListShareTemplatesAPI)

	w := doRequestWithID(r, "GET", "/api/admin/share-templates", adminID.String(), "admin", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Templates []models.MessageTemplate `json:"templates"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Templates) == 0 {
		t.Fatal("expected at least one template")
	}
}

func TestShareTemplatesAPI_CreateAndGet(t *testing.T) {
	adminID := setupShareTemplatesDB(t)
	defer cleanupShareTemplatesTest(t, adminID)

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), CreateShareTemplateAPI)

	body, _ := json.Marshal(map[string]string{"name": "New Template", "body": "New body {item}"})
	w := doRequestWithID(r, "POST", "/api/admin/share-templates", adminID.String(), "admin", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created models.MessageTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.Name != "New Template" {
		t.Errorf("expected name %q, got %q", "New Template", created.Name)
	}
	if created.Body != "New body {item}" {
		t.Errorf("expected body %q, got %q", "New body {item}", created.Body)
	}
	if created.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
}

func TestShareTemplatesAPI_Update(t *testing.T) {
	adminID := setupShareTemplatesDB(t)
	defer cleanupShareTemplatesTest(t, adminID)

	// Create a template first
	created, err := services.CreateMessageTemplate("Original", "Original body", adminID)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/share-templates/:id", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), UpdateShareTemplateAPI)

	body, _ := json.Marshal(map[string]string{"name": "Updated", "body": "Updated body"})
	w := doRequestWithID(r, "PATCH", "/api/admin/share-templates/"+created.ID.String(), adminID.String(), "admin", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the update persisted
	updated, err := services.GetMessageTemplateByID(created.ID)
	if err != nil {
		t.Fatalf("get updated template: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("expected name %q, got %q", "Updated", updated.Name)
	}
	if updated.Body != "Updated body" {
		t.Errorf("expected body %q, got %q", "Updated body", updated.Body)
	}
}

func TestShareTemplatesAPI_Delete(t *testing.T) {
	adminID := setupShareTemplatesDB(t)
	defer cleanupShareTemplatesTest(t, adminID)

	// Create two templates so we can delete one
	t1, err := services.CreateMessageTemplate("Keep", "Keep body", adminID)
	if err != nil {
		t.Fatalf("create template 1: %v", err)
	}
	t2, err := services.CreateMessageTemplate("Delete", "Delete body", adminID)
	if err != nil {
		t.Fatalf("create template 2: %v", err)
	}

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.DELETE("/api/admin/share-templates/:id", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), DeleteShareTemplateAPI)

	w := doRequestWithID(r, "DELETE", "/api/admin/share-templates/"+t2.ID.String(), adminID.String(), "admin", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	_, err = services.GetMessageTemplateByID(t2.ID)
	if err == nil {
		t.Fatal("expected error after deletion, got nil")
	}

	// Verify t1 still exists
	_, err = services.GetMessageTemplateByID(t1.ID)
	if err != nil {
		t.Fatalf("expected t1 to exist after deletion: %v", err)
	}
}

func TestShareTemplatesAPI_DeleteLastTemplate_Returns400(t *testing.T) {
	adminID := setupShareTemplatesDB(t)
	defer cleanupShareTemplatesTest(t, adminID)

	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates`)

	// Create exactly one template
	t1, err := services.CreateMessageTemplate("Only", "Only body", adminID)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.DELETE("/api/admin/share-templates/:id", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), DeleteShareTemplateAPI)

	w := doRequestWithID(r, "DELETE", "/api/admin/share-templates/"+t1.ID.String(), adminID.String(), "admin", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for deleting last template, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "cannot delete the last message template" {
		t.Fatalf("expected 'cannot delete the last message template', got %q", resp["error"])
	}
}

func TestShareTemplatesAPI_SetDefault(t *testing.T) {
	adminID := setupShareTemplatesDB(t)
	defer cleanupShareTemplatesTest(t, adminID)

	// Create two templates
	_, err := services.CreateMessageTemplate("First", "First body", adminID)
	if err != nil {
		t.Fatalf("create template 1: %v", err)
	}
	t2, err := services.CreateMessageTemplate("Second", "Second body", adminID)
	if err != nil {
		t.Fatalf("create template 2: %v", err)
	}

	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates/:id/default", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), SetDefaultShareTemplateAPI)

	// Set t2 as default
	w := doRequestWithID(r, "POST", "/api/admin/share-templates/"+t2.ID.String()+"/default", adminID.String(), "admin", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify t2 is now default
	defaultTmpl, err := services.GetDefaultMessageTemplate()
	if err != nil {
		t.Fatalf("get default template: %v", err)
	}
	if defaultTmpl.ID != t2.ID {
		t.Errorf("expected default to be t2 (%s), got %s", t2.ID, defaultTmpl.ID)
	}
}

func TestShareTemplatesAPI_Delete_NotFound(t *testing.T) {
	adminID := setupShareTemplatesDB(t)
	defer cleanupShareTemplatesTest(t, adminID)

	// Create a couple templates so the "last template" guard doesn't trigger
	_, err := services.CreateMessageTemplate("Keep1", "Body1", adminID)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	_, err = services.CreateMessageTemplate("Keep2", "Body2", adminID)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	gin.SetMode(gin.TestMode)

	nonExistentID := uuid.New()

	r := gin.New()
	r.DELETE("/api/admin/share-templates/:id", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin, models.RoleWaffleManager), DeleteShareTemplateAPI)

	w := doRequestWithID(r, "DELETE", "/api/admin/share-templates/"+nonExistentID.String(), adminID.String(), "admin", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent template, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareTemplatesAPI_Update_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/share-templates/:id", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Params = []gin.Param{{Key: "id", Value: uuid.New().String()}}
		UpdateShareTemplateAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"name": "Test", "body": "Body"})
	req := httptest.NewRequest("PATCH", "/api/admin/share-templates/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent template, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareTemplatesAPI_SetDefault_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/admin/share-templates/:id/default", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Params = []gin.Param{{Key: "id", Value: uuid.New().String()}}
		SetDefaultShareTemplateAPI(c)
	})

	req := httptest.NewRequest("POST", "/api/admin/share-templates/"+uuid.New().String()+"/default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent template, got %d: %s", w.Code, w.Body.String())
	}
}
