package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecovery_CatchesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rr := httptest.NewRecorder()
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON response body: %v", err)
	}

	if body["error"] != "internal server error" {
		t.Fatalf("expected error message 'internal server error', got %q", body["error"])
	}
}

func TestRecovery_NoPanicDetailsLeaked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rr := httptest.NewRecorder()
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("secret details that must not leak")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(rr, req)

	body := rr.Body.String()
	if body != `{"error":"internal server error"}` {
		t.Fatalf("expected generic error message, got %q", body)
	}
}

func TestRecovery_PassthroughNormalHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rr := httptest.NewRecorder()
	r := gin.New()
	r.Use(Recovery())
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
