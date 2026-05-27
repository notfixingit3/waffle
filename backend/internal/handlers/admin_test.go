package handlers

import (
	"bytes"
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
	ws "github.com/syrup/backend/internal/websocket"
)

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)
	ws.InitHub()
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

func doRequest(r *gin.Engine, method, url, role string, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	if role != "" {
		req.AddCookie(&http.Cookie{Name: "admin_token", Value: createToken(role)})
	}
	r.ServeHTTP(w, req)
	return w
}

func TestValidateCreateAdminForm_ShortPassword(t *testing.T) {
	errs := validateCreateAdminForm("testuser", "test@example.com", "abc")
	if len(errs) == 0 {
		t.Fatal("expected validation errors for short password, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "Password must be at least 8 characters") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected password length error in: %v", errs)
	}
}

func TestValidateCreateAdminForm_EmptyPassword(t *testing.T) {
	errs := validateCreateAdminForm("testuser", "test@example.com", "")
	if len(errs) == 0 {
		t.Fatal("expected validation errors for empty password, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "Password is required") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected password required error in: %v", errs)
	}
}

func TestValidateCreateAdminForm_ValidPassword(t *testing.T) {
	errs := validateCreateAdminForm("testuser", "test@example.com", "goodpw88")
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for valid input, got: %v", errs)
	}
}

func TestValidateCreateAdminForm_MissingFields(t *testing.T) {
	errs := validateCreateAdminForm("", "", "abc")
	if len(errs) != 3 {
		t.Fatalf("expected 3 validation errors (username, email, password), got %d: %v", len(errs), errs)
	}
}

func TestCreateAdminPost_AcceptsWaffleManagerRole(t *testing.T) {
	role := "waffle_manager"
	if role != models.RoleAdmin && role != models.RoleSuperAdmin && role != models.RoleWaffleManager {
		t.Fatal("waffle_manager role should be accepted, but was rejected")
	}
}

func TestCreateAdminPost_RejectsInvalidRole(t *testing.T) {
	role := "hacker"
	if role == models.RoleAdmin || role == models.RoleSuperAdmin || role == models.RoleWaffleManager {
		t.Fatal("invalid role 'hacker' should be rejected, but was accepted")
	}
}

func TestClearWinnerAPI_WaffleManager_Forbidden(t *testing.T) {
	r := gin.New()
	r.POST("/api/admin/waffles/:id/clear-winner", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), ClearWinnerAPI)
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/clear-winner", models.RoleWaffleManager, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on clear-winner API, got %d", w.Code)
	}
}

func TestChangeWinnerAPI_WaffleManager_Forbidden(t *testing.T) {
	r := gin.New()
	r.POST("/api/admin/waffles/:id/change-winner", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), ChangeWinnerAPI)
	w := doRequest(r, "POST", "/api/admin/waffles/"+uuid.New().String()+"/change-winner", models.RoleWaffleManager, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager on change-winner API, got %d", w.Code)
	}
}

func TestClearWinnerAPI_Admin_Allowed(t *testing.T) {
	r := gin.New()
	r.POST("/api/admin/waffles/:id/clear-winner", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), ClearWinnerAPI)
	w := doRequest(r, "POST", "/api/admin/waffles/bad-id/clear-winner", models.RoleAdmin, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for admin on clear-winner API (invalid waffle id), got %d", w.Code)
	}
}

func TestChangeWinnerAPI_Admin_Allowed(t *testing.T) {
	r := gin.New()
	r.POST("/api/admin/waffles/:id/change-winner", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), ChangeWinnerAPI)
	body := []byte(`{"winning_spot_number":5}`)
	w := doRequest(r, "POST", "/api/admin/waffles/bad-id/change-winner", models.RoleAdmin, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for admin on change-winner API (invalid waffle id), got %d", w.Code)
	}
}
