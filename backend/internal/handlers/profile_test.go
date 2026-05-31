package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/services"
)

func TestUpdateProfileAPI_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/profile", UpdateProfileAPI)

	body, _ := json.Marshal(map[string]string{"first_name": "Test"})
	req := httptest.NewRequest("PATCH", "/api/admin/me/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProfileAPI_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/profile", func(c *gin.Context) {
		c.Set("admin_id", "not-a-uuid")
		UpdateProfileAPI(c)
	})

	body, _ := json.Marshal(map[string]string{"first_name": "Test"})
	req := httptest.NewRequest("PATCH", "/api/admin/me/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProfileAPI_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/profile", func(c *gin.Context) {
		c.Set("admin_id", "550e8400-e29b-41d4-a716-446655440000")
		UpdateProfileAPI(c)
	})

	req := httptest.NewRequest("PATCH", "/api/admin/me/profile", bytes.NewReader([]byte(`{"invalid"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProfileAPI_InvalidSocialPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PATCH("/api/admin/me/profile", func(c *gin.Context) {
		c.Set("admin_id", "550e8400-e29b-41d4-a716-446655440000")
		UpdateProfileAPI(c)
	})

	body, _ := json.Marshal(map[string]interface{}{
		"social_links": []map[string]string{
			{"platform": "myspace", "handle": "test"},
		},
	})
	req := httptest.NewRequest("PATCH", "/api/admin/me/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidateSocialLinks_Valid(t *testing.T) {
	links := []models.SocialLink{
		{Platform: "instagram", Handle: "test"},
		{Platform: "tiktok", Handle: "test"},
	}
	errors := services.ValidateSocialLinks(links)
	if len(errors) > 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestValidateSocialLinks_InvalidPlatform(t *testing.T) {
	links := []models.SocialLink{
		{Platform: "myspace", Handle: "test"},
	}
	errors := services.ValidateSocialLinks(links)
	if len(errors) == 0 {
		t.Fatal("expected validation error for invalid platform")
	}
}

func TestValidateSocialLinks_DuplicatePlatform(t *testing.T) {
	links := []models.SocialLink{
		{Platform: "instagram", Handle: "test1"},
		{Platform: "instagram", Handle: "test2"},
	}
	errors := services.ValidateSocialLinks(links)
	if len(errors) == 0 {
		t.Fatal("expected validation error for duplicate platform")
	}
}

func TestValidateSocialLinks_TooMany(t *testing.T) {
	links := []models.SocialLink{
		{Platform: "instagram", Handle: "1"},
		{Platform: "tiktok", Handle: "2"},
		{Platform: "x", Handle: "3"},
		{Platform: "facebook", Handle: "4"},
		{Platform: "youtube", Handle: "5"},
		{Platform: "discord", Handle: "6"},
		{Platform: "instagram", Handle: "7"},
		{Platform: "tiktok", Handle: "8"},
		{Platform: "x", Handle: "9"},
		{Platform: "facebook", Handle: "10"},
		{Platform: "youtube", Handle: "11"},
	}
	errors := services.ValidateSocialLinks(links)
	if len(errors) == 0 {
		t.Fatal("expected validation error for too many links")
	}
}
