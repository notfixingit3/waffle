package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func createTestContextForSecurity(method, url string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, url, nil)
	return c, w
}

func TestSecurityHeaders_SetsXContentTypeOptions(t *testing.T) {
	c, w := createTestContextForSecurity("GET", "/")
	handler := SecurityHeaders()
	handler(c)

	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff, got: %q", got)
	}
}

func TestSecurityHeaders_SetsXFrameOptions(t *testing.T) {
	c, w := createTestContextForSecurity("GET", "/")
	handler := SecurityHeaders()
	handler(c)

	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options: DENY, got: %q", got)
	}
}

func TestSecurityHeaders_SetsReferrerPolicy(t *testing.T) {
	c, w := createTestContextForSecurity("GET", "/")
	handler := SecurityHeaders()
	handler(c)

	if got := w.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("expected Referrer-Policy: strict-origin-when-cross-origin, got: %q", got)
	}
}

func TestSecurityHeaders_DoesNotSetHSTS(t *testing.T) {
	c, w := createTestContextForSecurity("GET", "/")
	handler := SecurityHeaders()
	handler(c)

	// HSTS header names: Strict-Transport-Security
	for _, h := range w.Header().Values("Strict-Transport-Security") {
		if h != "" {
			t.Fatalf("expected no Strict-Transport-Security header, got: %q", h)
		}
	}
}

func TestSecurityHeaders_DoesNotSetCSP(t *testing.T) {
	c, w := createTestContextForSecurity("GET", "/")
	handler := SecurityHeaders()
	handler(c)

	if v := w.Header().Get("Content-Security-Policy"); v != "" {
		t.Fatalf("expected no Content-Security-Policy header, got: %q", v)
	}
	if v := w.Header().Get("X-Content-Security-Policy"); v != "" {
		t.Fatalf("expected no X-Content-Security-Policy header, got: %q", v)
	}
	if v := w.Header().Get("X-WebKit-CSP"); v != "" {
		t.Fatalf("expected no X-WebKit-CSP header, got: %q", v)
	}
}

func TestSecurityHeaders_CalledNext(t *testing.T) {
	c, _ := createTestContextForSecurity("GET", "/")
	var nextCalled bool
	handler := SecurityHeaders()
	wrapped := func(c *gin.Context) {
		handler(c)
		nextCalled = true
	}
	wrapped(c)

	if !nextCalled {
		t.Fatal("expected c.Next() to be called")
	}

	if c.IsAborted() {
		t.Fatal("expected context not to be aborted")
	}
}

func TestSecurityHeaders_AllHeadersSet(t *testing.T) {
	c, w := createTestContextForSecurity("GET", "/")
	handler := SecurityHeaders()
	handler(c)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for key, expectedValue := range expected {
		if got := w.Header().Get(key); got != expectedValue {
			t.Errorf("expected header %s: %q, got: %q", key, expectedValue, got)
		}
	}
}

func TestSecurityHeaders_AllMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodHead,
	}

	for _, method := range methods {
		c, w := createTestContextForSecurity(method, "/")
		handler := SecurityHeaders()
		handler(c)

		if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("[%s] expected X-Content-Type-Options: nosniff, got: %q", method, got)
		}
		if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Errorf("[%s] expected X-Frame-Options: DENY, got: %q", method, got)
		}
		if got := w.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
			t.Errorf("[%s] expected Referrer-Policy: strict-origin-when-cross-origin, got: %q", method, got)
		}
	}
}
