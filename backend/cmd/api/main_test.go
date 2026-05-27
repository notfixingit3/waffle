package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/middleware"
	"github.com/syrup/backend/internal/models"
)

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)
	code := m.Run()
	os.Exit(code)
}

func createToken(role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin_id": uuid.New().String(),
		"email":    "test@syrup.test",
		"role":     role,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))
	return tokenString
}

func setupTestRouter() *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Simulate the production route groups with the same middleware chains.
	// The goal is to test route-level access control, not the actual handlers.

	// Admin auth (no middleware)
	admin := r.Group("/api/admin")
	admin.POST("/login", func(c *gin.Context) { c.JSON(200, gin.H{"token": "fake"}) })

	// Auth routes (RequireAuth only)
	adminAuth := admin.Group("", middleware.RequireAuth)
	adminAuth.GET("/me", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// User management (RequireSuperAdmin)
	adminUsers := admin.Group("/admins", middleware.RequireAuth, middleware.RequireSuperAdmin)
	adminUsers.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"admins": []interface{}{}}) })

	// Waffle CRUD + winner (RequireAuth only) — accessible to waffle_manager
	adminWaffles := admin.Group("/waffles", middleware.RequireAuth)
	adminWaffles.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"waffles": []interface{}{}}) })
	adminWaffles.POST("/", func(c *gin.Context) { c.JSON(201, gin.H{"created": true}) })
	adminWaffles.PATCH("/:id", func(c *gin.Context) { c.JSON(200, gin.H{"updated": true}) })
	adminWaffles.POST("/:id/winner", func(c *gin.Context) { c.JSON(200, gin.H{"winner_set": true}) })

	// Archive/delete/unarchive (RequireRole admin/super_admin) — NOT accessible to waffle_manager
	adminManagerAPI := admin.Group("/waffles", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin))
	adminManagerAPI.POST("/:id/archive", func(c *gin.Context) { c.JSON(200, gin.H{"archived": true}) })
	adminManagerAPI.POST("/:id/unarchive", func(c *gin.Context) { c.JSON(200, gin.H{"unarchived": true}) })
	adminManagerAPI.DELETE("/:id", func(c *gin.Context) { c.JSON(200, gin.H{"deleted": true}) })

	// Reports (RequireAuth only) — accessible to waffle_manager
	adminReports := admin.Group("/reports", middleware.RequireAuth)
	adminReports.GET("/drought", func(c *gin.Context) { c.JSON(200, gin.H{"drought": []interface{}{}}) })

	// Spots (RequireAuth only) — accessible to waffle_manager
	adminSpots := admin.Group("/spots", middleware.RequireAuth)
	adminSpots.POST("/:id/pay", func(c *gin.Context) { c.JSON(200, gin.H{"paid": true}) })
	adminSpots.POST("/:id/release", func(c *gin.Context) { c.JSON(200, gin.H{"released": true}) })

	return r
}

func doRequest(r *gin.Engine, method, url, role string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, nil)
	if role != "" {
		req.AddCookie(&http.Cookie{Name: "admin_token", Value: createToken(role)})
	}
	r.ServeHTTP(w, req)
	return w
}

// Archive/delete/unarchive API: waffle_manager gets 403

func TestWaffleManager_ArchiveAPI_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/archive", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on archive API, got %d", w.Code)
	}
}

func TestWaffleManager_UnarchiveAPI_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/unarchive", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on unarchive API, got %d", w.Code)
	}
}

func TestWaffleManager_DeleteAPI_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "DELETE", "/api/admin/waffles/"+uuid.New().String(), models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on delete API, got %d", w.Code)
	}
}

// Archive/delete/unarchive API: admin gets 200

func TestAdmin_ArchiveAPI_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/archive", models.RoleAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin on archive API, got %d", w.Code)
	}
}

func TestAdmin_UnarchiveAPI_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/unarchive", models.RoleAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin on unarchive API, got %d", w.Code)
	}
}

func TestAdmin_DeleteAPI_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "DELETE", "/api/admin/waffles/"+uuid.New().String(), models.RoleAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin on delete API, got %d", w.Code)
	}
}

func TestSuperAdmin_ArchiveAPI_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/archive", models.RoleSuperAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin on archive API, got %d", w.Code)
	}
}

// Waffle CRUD + winner: waffle_manager gets 200

func TestWaffleManager_ListWaffles_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/api/admin/waffles/", models.RoleWaffleManager)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for waffle_manager on waffle list API, got %d", w.Code)
	}
}

func TestWaffleManager_CreateWaffle_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/", models.RoleWaffleManager)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for waffle_manager on create waffle API, got %d", w.Code)
	}
}

func TestWaffleManager_UpdateWaffle_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "PATCH", "/api/admin/waffles/"+uuid.New().String(), models.RoleWaffleManager)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for waffle_manager on update waffle API, got %d", w.Code)
	}
}

func TestWaffleManager_SetWinner_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/winner", models.RoleWaffleManager)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for waffle_manager on set winner API, got %d", w.Code)
	}
}

// Reports: waffle_manager gets 200

func TestWaffleManager_Reports_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/api/admin/reports/drought", models.RoleWaffleManager)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for waffle_manager on reports API, got %d", w.Code)
	}
}

// Spots: waffle_manager gets 200

func TestWaffleManager_PaySpot_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/spots/"+uuid.New().String()+"/pay", models.RoleWaffleManager)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for waffle_manager on pay spot API, got %d", w.Code)
	}
}

func TestWaffleManager_ReleaseSpot_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/spots/"+uuid.New().String()+"/release", models.RoleWaffleManager)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for waffle_manager on release spot API, got %d", w.Code)
	}
}

// User management: waffle_manager gets 403 (RequireSuperAdmin)

func TestWaffleManager_UserManagement_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/api/admin/admins/", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on user management API, got %d", w.Code)
	}
}

func TestAdmin_UserManagement_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/api/admin/admins/", models.RoleAdmin)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for admin on user management API, got %d", w.Code)
	}
}

func TestSuperAdmin_UserManagement_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/api/admin/admins/", models.RoleSuperAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin on user management API, got %d", w.Code)
	}
}

// Unauthenticated: 401

func TestUnauthenticated_ArchiveAPI_Unauthorized(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/archive", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated archive request, got %d", w.Code)
	}
}

func TestUnauthenticated_ListWaffles_Unauthorized(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/api/admin/waffles/", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated waffle list request, got %d", w.Code)
	}
}
