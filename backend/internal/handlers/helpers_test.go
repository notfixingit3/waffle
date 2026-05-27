package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupErrorTest creates a gin test context with a ResponseRecorder.
func setupErrorTest() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		message        string
		wantStatus     int
		wantErrorField string
	}{
		{
			name:           "bad request",
			status:         http.StatusBadRequest,
			message:        "invalid input",
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "invalid input",
		},
		{
			name:           "not found",
			status:         http.StatusNotFound,
			message:        "resource not found",
			wantStatus:     http.StatusNotFound,
			wantErrorField: "resource not found",
		},
		{
			name:           "unauthorized",
			status:         http.StatusUnauthorized,
			message:        "not authenticated",
			wantStatus:     http.StatusUnauthorized,
			wantErrorField: "not authenticated",
		},
		{
			name:           "forbidden",
			status:         http.StatusForbidden,
			message:        "permission denied",
			wantStatus:     http.StatusForbidden,
			wantErrorField: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupErrorTest()
			ErrorResponse(c, tt.status, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if body["error"] != tt.wantErrorField {
				t.Errorf("error field = %q, want %q", body["error"], tt.wantErrorField)
			}
		})
	}
}

func TestServerErrorResponse(t *testing.T) {
	c, w := setupErrorTest()
	originalErr := errors.New("database connection refused")
	ServerErrorResponse(c, originalErr)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body["error"] != "internal server error" {
		t.Errorf("error field = %q, want %q", body["error"], "internal server error")
	}

	// Ensure the original error message is NOT leaked in the response
	if body["error"] == originalErr.Error() {
		t.Error("internal error details leaked in response body")
	}
}
