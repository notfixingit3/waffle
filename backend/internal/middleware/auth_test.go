package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func createTestContextWithRole(method, url, role string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, url, nil)
	if role != "" {
		c.Set("admin_role", role)
	}
	return c, w
}

func TestRequireRole_AllowsAdmin(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/api/admin/waffles", "admin")
	handler := RequireRole("admin", "super_admin")
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin role, got %d", w.Code)
	}
	if c.IsAborted() {
		t.Fatal("expected context not to be aborted for allowed role")
	}
}

func TestRequireRole_BlocksWaffleManager(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/api/admin/waffles", "waffle_manager")
	handler := RequireRole("admin", "super_admin")
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for waffle_manager, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Fatal("expected context to be aborted for disallowed role")
	}
}

func TestRequireRole_AllowsSuperAdmin(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/api/admin/admins", "super_admin")
	handler := RequireRole("admin", "super_admin")
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin, got %d", w.Code)
	}
	if c.IsAborted() {
		t.Fatal("expected context not to be aborted for allowed role")
	}
}

func TestRequireRole_BlocksWhenNoRoleSet(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/api/admin/waffles", "")
	handler := RequireRole("admin", "super_admin")
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when no role set, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Fatal("expected context to be aborted when no role set")
	}
}

func TestRequireRole_SingleRoleAllows(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/api/admin/reports", "admin")
	handler := RequireRole("admin")
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when single role matches, got %d", w.Code)
	}
	if c.IsAborted() {
		t.Fatal("expected context not to be aborted when single role matches")
	}
}

func TestRequireRole_SingleRoleBlocks(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/api/admin/reports", "waffle_manager")
	handler := RequireRole("admin")
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when single role does not match, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Fatal("expected context to be aborted when single role does not match")
	}
}

func TestRequireRole_JSONErrorForAPI(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/api/admin/waffles", "waffle_manager")
	handler := RequireRole("admin", "super_admin")
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type for API request, got %q", contentType)
	}
}

func TestRequireRole_PlaintextForPage(t *testing.T) {
	c, w := createTestContextWithRole("GET", "/admin/waffles", "waffle_manager")
	handler := RequireRole("admin", "super_admin")
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "access denied" {
		t.Fatalf("expected plaintext 'access denied', got %q", body)
	}
}
