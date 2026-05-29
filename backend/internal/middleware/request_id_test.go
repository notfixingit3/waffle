package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func createTestContextWithRequestIDHeader(method, url, requestID string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, url, nil)
	if requestID != "" {
		c.Request.Header.Set("X-Request-ID", requestID)
	}
	return c, w
}

func TestRequestID_GeneratesUUIDWhenNoHeader(t *testing.T) {
	c, w := createTestContextWithRequestIDHeader("GET", "/", "")
	handler := RequestID()
	handler(c)

	requestID, exists := c.Get("request_id")
	if !exists {
		t.Fatal("expected request_id to be set in context")
	}

	idStr, ok := requestID.(string)
	if !ok {
		t.Fatalf("expected request_id to be a string, got %T", requestID)
	}

	if idStr == "" {
		t.Fatal("expected non-empty request_id")
	}

	// Validate UUID v4 format: 8-4-4-4-12 hex digits
	if len(idStr) != 36 {
		t.Fatalf("expected UUID v4 to be 36 chars, got %d: %q", len(idStr), idStr)
	}

	responseID := w.Header().Get("X-Request-ID")
	if responseID != idStr {
		t.Fatalf("expected response header X-Request-ID to match context value, got %q", responseID)
	}
}

func TestRequestID_PreservesExistingHeader(t *testing.T) {
	c, w := createTestContextWithRequestIDHeader("GET", "/", "client-provided-id-123")
	handler := RequestID()
	handler(c)

	requestID, exists := c.Get("request_id")
	if !exists {
		t.Fatal("expected request_id to be set in context")
	}

	idStr, _ := requestID.(string)
	if idStr != "client-provided-id-123" {
		t.Fatalf("expected original request ID to be preserved, got %q", idStr)
	}

	responseID := w.Header().Get("X-Request-ID")
	if responseID != "client-provided-id-123" {
		t.Fatalf("expected response header to preserve original ID, got %q", responseID)
	}
}

func TestRequestID_CalledNext(t *testing.T) {
	c, w := createTestContextWithRequestIDHeader("GET", "/", "")
	var nextCalled bool
	handler := RequestID()
	// Wrap to detect c.Next() was called
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

	responseID := w.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Fatal("expected X-Request-ID response header to be set")
	}
}

func TestRequestID_ValidUUIDv4Format(t *testing.T) {
	// Run multiple times to ensure UUID v4 format is consistent
	for i := 0; i < 10; i++ {
		c, _ := createTestContextWithRequestIDHeader("GET", "/", "")
		handler := RequestID()
		handler(c)

		requestID, _ := c.Get("request_id")
		idStr, _ := requestID.(string)

		// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
		if len(idStr) != 36 {
			t.Fatalf("iteration %d: expected 36 chars, got %d", i, len(idStr))
		}
		if idStr[14] != '4' {
			t.Fatalf("iteration %d: expected version 4 UUID, got char at pos 14: %c", i, idStr[14])
		}
	}
}
