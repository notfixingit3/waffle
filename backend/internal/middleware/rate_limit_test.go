package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func createTestContextWithIP(method, url, ip string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, url, nil)
	if ip != "" {
		c.Request.RemoteAddr = ip + ":12345"
	}
	return c, w
}

func TestRateLimitAllowsBurst(t *testing.T) {
	rateLimitClients = sync.Map{}

	for i := 0; i < 10; i++ {
		c, w := createTestContextWithIP("POST", "/api/claims", "192.168.1.1")
		RateLimitClaims(c)

		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d should have succeeded, got 429", i+1)
		}
		if c.IsAborted() {
			t.Fatalf("request %d should not have been aborted", i+1)
		}
	}
}

func TestRateLimitBlocksAfterBurst(t *testing.T) {
	rateLimitClients = sync.Map{}

	for i := 0; i < 10; i++ {
		c, _ := createTestContextWithIP("POST", "/api/claims", "192.168.1.1")
		RateLimitClaims(c)
	}

	c, w := createTestContextWithIP("POST", "/api/claims", "192.168.1.1")
	RateLimitClaims(c)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Fatal("expected context to be aborted")
	}

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter != "6" {
		t.Fatalf("expected Retry-After: 6, got %q", retryAfter)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp["error"] != "rate limit exceeded" {
		t.Fatalf("expected error 'rate limit exceeded', got %q", resp["error"])
	}
}

func TestRateLimitPerIP(t *testing.T) {
	rateLimitClients = sync.Map{}

	for i := 0; i < 10; i++ {
		c, _ := createTestContextWithIP("POST", "/api/claims", "10.0.0.1")
		RateLimitClaims(c)
	}

	c1, w1 := createTestContextWithIP("POST", "/api/claims", "10.0.0.1")
	RateLimitClaims(c1)
	if w1.Code != http.StatusTooManyRequests {
		t.Fatalf("expected IP1 11th request to be 429, got %d", w1.Code)
	}

	c2, w2 := createTestContextWithIP("POST", "/api/claims", "10.0.0.2")
	RateLimitClaims(c2)
	if w2.Code == http.StatusTooManyRequests {
		t.Fatal("expected IP2 request to succeed, got 429")
	}
	if c2.IsAborted() {
		t.Fatal("expected IP2 context not to be aborted")
	}
}

func TestOptionsBypass(t *testing.T) {
	rateLimitClients = sync.Map{}

	for i := 0; i < 10; i++ {
		c, _ := createTestContextWithIP("POST", "/api/claims", "192.168.1.1")
		RateLimitClaims(c)
	}

	cPost, wPost := createTestContextWithIP("POST", "/api/claims", "192.168.1.1")
	RateLimitClaims(cPost)
	if wPost.Code != http.StatusTooManyRequests {
		t.Fatalf("expected POST 11th request to be 429, got %d", wPost.Code)
	}

	cOpt, wOpt := createTestContextWithIP("OPTIONS", "/api/claims", "192.168.1.1")
	RateLimitClaims(cOpt)
	if wOpt.Code == http.StatusTooManyRequests {
		t.Fatal("expected OPTIONS request to bypass rate limit, got 429")
	}
	if cOpt.IsAborted() {
		t.Fatal("expected OPTIONS context not to be aborted")
	}
}

func TestCleanupRemovesStaleEntries(t *testing.T) {
	rateLimitClients = sync.Map{}

	ip := "192.168.1.1"

	oldEntry := &rateLimitClientEntry{
		limiter:  rate.NewLimiter(rate.Every(6*time.Second), 10),
		lastSeen: time.Now().Add(-10 * time.Minute),
	}
	rateLimitClients.Store(ip, oldEntry)

	if _, ok := rateLimitClients.Load(ip); !ok {
		t.Fatal("expected entry to exist before cleanup")
	}

	cleanupStaleRateLimiters()

	if _, ok := rateLimitClients.Load(ip); ok {
		t.Fatal("expected stale entry to be removed after cleanup")
	}
}
