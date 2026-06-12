package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/middleware"
	"github.com/syrup/backend/internal/models"
	ws "github.com/syrup/backend/internal/websocket"
)

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)
	ws.InitHub()

	// Initialize DB pool if Postgres is available so service calls don't nil-panic.
	_, _ = db.Connect()

	code := m.Run()

	if db.Pool != nil {
		db.Pool.Close()
	}
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

func doFormRequest(r *gin.Engine, method, url, role, csrfToken string, data url.Values) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	body := strings.NewReader(data.Encode())
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if role != "" {
		req.AddCookie(&http.Cookie{Name: "admin_token", Value: createToken(role)})
	}
	if csrfToken != "" {
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
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
		if strings.Contains(e, "password must be at least") {
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

func TestArchiveWafflePost_MissingCSRF(t *testing.T) {
	r := gin.New()
	r.POST("/admin/waffles/:id/archive", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), ArchiveWafflePost)
	w := doFormRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/archive", models.RoleAdmin, "", nil)
	if w.Code == http.StatusFound {
		t.Fatalf("expected non-redirect status when CSRF is missing, got 302")
	}
}

func TestArchiveWafflePost_Valid(t *testing.T) {
	r := gin.New()
	r.POST("/admin/waffles/:id/archive", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), ArchiveWafflePost)
	data := url.Values{"csrf_token": {"test-csrf-token"}}
	w := doFormRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/archive", models.RoleAdmin, "test-csrf-token", data)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for valid archive request, got %d", w.Code)
	}
}

func TestArchiveWafflePost_InvalidID(t *testing.T) {
	r := gin.New()
	r.POST("/admin/waffles/:id/archive", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), ArchiveWafflePost)
	data := url.Values{"csrf_token": {"test-csrf-token"}}
	w := doFormRequest(r, "POST", "/admin/waffles/not-a-uuid/archive", models.RoleAdmin, "test-csrf-token", data)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect for invalid ID, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/admin/dashboard") {
		t.Fatalf("expected redirect to /admin/dashboard, got %s", loc)
	}
}

func TestUnarchiveWafflePost_Valid(t *testing.T) {
	r := gin.New()
	r.POST("/admin/waffles/:id/unarchive", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), UnarchiveWafflePost)
	data := url.Values{"csrf_token": {"test-csrf-token"}}
	w := doFormRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/unarchive", models.RoleAdmin, "test-csrf-token", data)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for valid unarchive request, got %d", w.Code)
	}
}

func TestUnarchiveWafflePost_MissingCSRF(t *testing.T) {
	r := gin.New()
	r.POST("/admin/waffles/:id/unarchive", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), UnarchiveWafflePost)
	w := doFormRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/unarchive", models.RoleAdmin, "", nil)
	if w.Code == http.StatusFound {
		t.Fatalf("expected non-redirect status when CSRF is missing, got 302")
	}
}

func TestDeleteWafflePost_MissingCSRF(t *testing.T) {
	r := gin.New()
	r.POST("/admin/waffles/:id/delete", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), DeleteWafflePost)
	w := doFormRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/delete", models.RoleAdmin, "", nil)
	if w.Code == http.StatusFound {
		t.Fatalf("expected non-redirect status when CSRF is missing, got 302")
	}
}

func TestDeleteWafflePost_Valid(t *testing.T) {
	r := gin.New()
	r.POST("/admin/waffles/:id/delete", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), DeleteWafflePost)
	data := url.Values{
		"csrf_token": {"test-csrf-token"},
		"confirm":    {"DELETE"},
	}
	w := doFormRequest(r, "POST", "/admin/waffles/"+uuid.New().String()+"/delete", models.RoleAdmin, "test-csrf-token", data)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 for valid delete request, got %d", w.Code)
	}
}
