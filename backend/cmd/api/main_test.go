package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/middleware"
	"github.com/syrup/backend/internal/models"
)

func TestHealth_Returns200(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health, got %d", w.Code)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestReady_Returns503_WhenNoDB(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/ready", "")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for /ready (no DB), got %d", w.Code)
	}
}

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

	// Public health/readiness endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/ready", func(c *gin.Context) {
		// No DB in test context — simulate failure
		c.JSON(503, gin.H{"status": "error", "db": "disconnected"})
	})

	// Simulate the production route groups with the same middleware chains.
	// The goal is to test route-level access control, not the actual handlers.

	// Public version endpoint
	r.Group("/api").GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": Version})
	})

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

	adminManagerAPI := admin.Group("/waffles", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin))
	adminManagerAPI.POST("/:id/archive", func(c *gin.Context) { c.JSON(200, gin.H{"archived": true}) })
	adminManagerAPI.POST("/:id/unarchive", func(c *gin.Context) { c.JSON(200, gin.H{"unarchived": true}) })
	adminManagerAPI.DELETE("/:id", func(c *gin.Context) { c.JSON(200, gin.H{"deleted": true}) })
	adminManagerAPI.POST("/:id/clear-winner", func(c *gin.Context) { c.JSON(200, gin.H{"winner_cleared": true}) })
	adminManagerAPI.POST("/:id/change-winner", func(c *gin.Context) { c.JSON(200, gin.H{"winner_changed": true}) })

	// Reports (RequireAuth only) — accessible to waffle_manager
	adminReports := admin.Group("/reports", middleware.RequireAuth)
	adminReports.GET("/drought", func(c *gin.Context) { c.JSON(200, gin.H{"drought": []interface{}{}}) })

	// Spots (RequireAuth only) — accessible to waffle_manager
	adminSpots := admin.Group("/spots", middleware.RequireAuth)
	adminSpots.POST("/:id/pay", func(c *gin.Context) { c.JSON(200, gin.H{"paid": true}) })
	adminSpots.POST("/:id/release", func(c *gin.Context) { c.JSON(200, gin.H{"released": true}) })

	// Rendered admin form routes (auth + role) — simulate production admin pages group
	adminPages := r.Group("/admin", middleware.RequireAuth)
	adminPages.POST("/waffles/:id/archive", middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin/dashboard") })
	adminPages.POST("/waffles/:id/unarchive", middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin/dashboard") })
	adminPages.POST("/waffles/:id/delete", middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin/dashboard") })

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

func TestGetVersion_ReturnsVersion(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "GET", "/api/version", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/version, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "version") {
		t.Fatalf("expected version in body, got %s", body)
	}
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

func TestWaffleManager_ClearWinner_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/clear-winner", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on clear-winner API, got %d", w.Code)
	}
}

func TestWaffleManager_ChangeWinner_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/change-winner", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on change-winner API, got %d", w.Code)
	}
}

func TestAdmin_ClearWinner_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/clear-winner", models.RoleAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin on clear-winner API, got %d", w.Code)
	}
}

func TestAdmin_ChangeWinner_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/change-winner", models.RoleAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin on change-winner API, got %d", w.Code)
	}
}

func TestSuperAdmin_ClearWinner_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/clear-winner", models.RoleSuperAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin on clear-winner API, got %d", w.Code)
	}
}

func TestSuperAdmin_ChangeWinner_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/change-winner", models.RoleSuperAdmin)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin on change-winner API, got %d", w.Code)
	}
}

func TestUnauthenticated_ClearWinner_Unauthorized(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/clear-winner", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated clear-winner request, got %d", w.Code)
	}
}

func TestUnauthenticated_ChangeWinner_Unauthorized(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/change-winner", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated change-winner request, got %d", w.Code)
	}
}

// Rendered admin form routes: waffle_manager gets 403

func TestWaffleManager_ArchiveForm_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/archive", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on archive form, got %d", w.Code)
	}
}

func TestWaffleManager_UnarchiveForm_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/unarchive", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on unarchive form, got %d", w.Code)
	}
}

func TestWaffleManager_DeleteForm_Forbidden(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/delete", models.RoleWaffleManager)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on delete form, got %d", w.Code)
	}
}

// Rendered admin form routes: admin gets 302

func TestAdmin_ArchiveForm_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/archive", models.RoleAdmin)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for admin on archive form, got %d", w.Code)
	}
}

func TestAdmin_UnarchiveForm_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/unarchive", models.RoleAdmin)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for admin on unarchive form, got %d", w.Code)
	}
}

func TestAdmin_DeleteForm_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/delete", models.RoleAdmin)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for admin on delete form, got %d", w.Code)
	}
}

// Rendered admin form routes: super_admin gets 302

func TestSuperAdmin_ArchiveForm_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/archive", models.RoleSuperAdmin)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for super_admin on archive form, got %d", w.Code)
	}
}

func TestSuperAdmin_UnarchiveForm_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/unarchive", models.RoleSuperAdmin)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for super_admin on unarchive form, got %d", w.Code)
	}
}

func TestSuperAdmin_DeleteForm_Allowed(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/delete", models.RoleSuperAdmin)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for super_admin on delete form, got %d", w.Code)
	}
}

// Rendered admin form routes: unauthenticated gets 302 (redirect to login)

func TestUnauthenticated_ArchiveForm_Unauthorized(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/archive", "")
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for unauthenticated archive form, got %d", w.Code)
	}
}

func TestUnauthenticated_UnarchiveForm_Unauthorized(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/unarchive", "")
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for unauthenticated unarchive form, got %d", w.Code)
	}
}

func TestUnauthenticated_DeleteForm_Unauthorized(t *testing.T) {
	r := setupTestRouter()
	w := doRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/delete", "")
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for unauthenticated delete form, got %d", w.Code)
	}
}
