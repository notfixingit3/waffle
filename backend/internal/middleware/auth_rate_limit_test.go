package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/syrup/backend/internal/services"
)

func TestAuthRateLimit_Passthrough_Form(t *testing.T) {
	services.ResetLoginAttempts("10.0.0.1", "testuser")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=testuser"))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request.RemoteAddr = "10.0.0.1:12345"

	AuthRateLimit(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if c.IsAborted() {
		t.Fatal("expected context not to be aborted")
	}
}

func TestAuthRateLimit_Passthrough_JSON(t *testing.T) {
	services.ResetLoginAttempts("10.0.0.1", "testuser")

	body := `{"username":"testuser","password":"sekret"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.RemoteAddr = "10.0.0.1:12345"

	AuthRateLimit(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if c.IsAborted() {
		t.Fatal("expected context not to be aborted")
	}
}

func TestAuthRateLimit_Returns429_WhenLockedOut(t *testing.T) {
	// Lock out this IP+username
	for i := 0; i < 5; i++ {
		services.RecordFailedLoginAttempt("10.0.0.2", "testuser")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=testuser"))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request.RemoteAddr = "10.0.0.2:12345"

	AuthRateLimit(c)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Fatal("expected context to be aborted")
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "too many failed login attempts, try again in 15 minutes" {
		t.Fatalf("unexpected error message: %q", resp["error"])
	}

	services.ResetLoginAttempts("10.0.0.2", "testuser")
}

func TestAuthRateLimit_LockoutPerIP(t *testing.T) {
	// Lock out 10.0.0.3 but not 10.0.0.4
	for i := 0; i < 5; i++ {
		services.RecordFailedLoginAttempt("10.0.0.3", "testuser")
	}

	// Locked IP
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=testuser"))
	c1.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c1.Request.RemoteAddr = "10.0.0.3:12345"
	AuthRateLimit(c1)
	if w1.Code != http.StatusTooManyRequests {
		t.Fatalf("expected locked IP to get 429, got %d", w1.Code)
	}

	// Different IP — should pass
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=testuser"))
	c2.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c2.Request.RemoteAddr = "10.0.0.4:12345"
	AuthRateLimit(c2)
	if w2.Code == http.StatusTooManyRequests {
		t.Fatal("expected different IP to pass, got 429")
	}
	if c2.IsAborted() {
		t.Fatal("expected different IP context not to be aborted")
	}

	services.ResetLoginAttempts("10.0.0.3", "testuser")
}

func TestAuthRateLimit_LockoutPerUsername(t *testing.T) {
	// Lock out 10.0.0.5 for "user1" but not for "user2"
	for i := 0; i < 5; i++ {
		services.RecordFailedLoginAttempt("10.0.0.5", "user1")
	}

	// Locked user
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=user1"))
	c1.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c1.Request.RemoteAddr = "10.0.0.5:12345"
	AuthRateLimit(c1)
	if w1.Code != http.StatusTooManyRequests {
		t.Fatalf("expected locked user to get 429, got %d", w1.Code)
	}

	// Different username — should pass
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=user2"))
	c2.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c2.Request.RemoteAddr = "10.0.0.5:12345"
	AuthRateLimit(c2)
	if w2.Code == http.StatusTooManyRequests {
		t.Fatal("expected different username to pass, got 429")
	}
	if c2.IsAborted() {
		t.Fatal("expected different username context not to be aborted")
	}

	services.ResetLoginAttempts("10.0.0.5", "user1")
}

func TestAuthRateLimit_NoUsername_Passes(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader(""))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request.RemoteAddr = "10.0.0.6:12345"

	AuthRateLimit(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected request without username to pass, got %d", w.Code)
	}
	if c.IsAborted() {
		t.Fatal("expected context not to be aborted when no username")
	}
}
