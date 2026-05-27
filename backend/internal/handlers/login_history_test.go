package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func init() {
	if db.Pool == nil {
		pool, err := db.Connect()
		if err == nil {
			_ = pool
		}
	}
}

func TestGetMyLoginHistoryAPI_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/me/login-history", GetMyLoginHistoryAPI)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/me/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "not authenticated" {
		t.Fatalf("expected 'not authenticated' error, got %v", resp["error"])
	}
}

func TestGetMyLoginHistoryAPI_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/me/login-history", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		GetMyLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/me/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if _, ok := resp["records"]; !ok {
		t.Fatal("expected 'records' field in response")
	}
	if _, ok := resp["total"]; !ok {
		t.Fatal("expected 'total' field in response")
	}
	if _, ok := resp["page"]; !ok {
		t.Fatal("expected 'page' field in response")
	}
	if _, ok := resp["limit"]; !ok {
		t.Fatal("expected 'limit' field in response")
	}
}

func TestGetMyLoginHistoryAPI_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/me/login-history", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		GetMyLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/me/login-history?page=2&limit=5", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	page, _ := resp["page"].(float64)
	limit, _ := resp["limit"].(float64)
	if int(page) != 2 {
		t.Fatalf("expected page 2, got %v", page)
	}
	if int(limit) != 5 {
		t.Fatalf("expected limit 5, got %v", limit)
	}
}

func TestGetAllLoginHistoryAPI_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/login-history", GetAllLoginHistoryAPI)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetAllLoginHistoryAPI_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/login-history", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Set("admin_role", models.RoleWaffleManager)
		GetAllLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if _, ok := resp["records"]; !ok {
		t.Fatal("expected 'records' field in response")
	}
	if _, ok := resp["total"]; !ok {
		t.Fatal("expected 'total' field in response")
	}
	if _, ok := resp["page"]; !ok {
		t.Fatal("expected 'page' field in response")
	}
	if _, ok := resp["limit"]; !ok {
		t.Fatal("expected 'limit' field in response")
	}
}

func TestGetAllLoginHistoryAPI_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/login-history", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Set("admin_role", models.RoleWaffleManager)
		GetAllLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/login-history?page=3&limit=20", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	page, _ := resp["page"].(float64)
	limit, _ := resp["limit"].(float64)
	if int(page) != 3 {
		t.Fatalf("expected page 3, got %v", page)
	}
	if int(limit) != 20 {
		t.Fatalf("expected limit 20, got %v", limit)
	}
}

func TestGetAllLoginHistoryAPI_DefaultPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/login-history", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Set("admin_role", models.RoleSuperAdmin)
		GetAllLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	page, _ := resp["page"].(float64)
	limit, _ := resp["limit"].(float64)
	if int(page) != 1 {
		t.Fatalf("expected default page 1, got %v", page)
	}
	if int(limit) != 20 {
		t.Fatalf("expected default limit 20, got %v", limit)
	}
}

func TestParsePagination_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		page, limit := parsePagination(c)
		c.JSON(http.StatusOK, gin.H{"page": page, "limit": limit})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if int(resp["page"].(float64)) != 1 {
		t.Fatalf("expected default page 1, got %v", resp["page"])
	}
	if int(resp["limit"].(float64)) != 20 {
		t.Fatalf("expected default limit 20, got %v", resp["limit"])
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		page, limit := parsePagination(c)
		c.JSON(http.StatusOK, gin.H{"page": page, "limit": limit})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=5&limit=25", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if int(resp["page"].(float64)) != 5 {
		t.Fatalf("expected page 5, got %v", resp["page"])
	}
	if int(resp["limit"].(float64)) != 25 {
		t.Fatalf("expected limit 25, got %v", resp["limit"])
	}
}

func TestParsePagination_InvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		page, limit := parsePagination(c)
		c.JSON(http.StatusOK, gin.H{"page": page, "limit": limit})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=abc&limit=xyz", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if int(resp["page"].(float64)) != 1 {
		t.Fatalf("expected default page 1 for invalid input, got %v", resp["page"])
	}
	if int(resp["limit"].(float64)) != 20 {
		t.Fatalf("expected default limit 20 for invalid input, got %v", resp["limit"])
	}
}

func TestParsePagination_NegativeValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		page, limit := parsePagination(c)
		c.JSON(http.StatusOK, gin.H{"page": page, "limit": limit})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=-1&limit=-5", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if int(resp["page"].(float64)) != 1 {
		t.Fatalf("expected default page 1 for negative input, got %v", resp["page"])
	}
	if int(resp["limit"].(float64)) != 20 {
		t.Fatalf("expected default limit 20 for negative input, got %v", resp["limit"])
	}
}

func TestGetMyLoginHistoryAPI_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/me/login-history", func(c *gin.Context) {
		c.Set("admin_id", "not-a-uuid")
		GetMyLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/me/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid admin ID, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "invalid admin" {
		t.Fatalf("expected 'invalid admin' error, got %v", resp["error"])
	}
}

func TestGetAllLoginHistoryAPI_MissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/login-history", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		// No admin_role set
		GetAllLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing role, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "role not found" {
		t.Fatalf("expected 'role not found' error, got %v", resp["error"])
	}
}

func TestGetAllLoginHistoryAPI_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/login-history", func(c *gin.Context) {
		c.Set("admin_id", "not-a-uuid")
		c.Set("admin_role", models.RoleSuperAdmin)
		GetAllLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid admin ID, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "invalid admin" {
		t.Fatalf("expected 'invalid admin' error, got %v", resp["error"])
	}
}

func TestPaginationResponseStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	records := []models.LoginHistory{
		{
			ID:         uuid.New(),
			AdminID:    uuid.New(),
			IPAddress:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
			Browser:    "Chrome",
			OS:         "macOS",
			DeviceType: "desktop",
			CreatedAt:  now,
		},
	}

	resp := buildPaginationResponse(records, 1, 1, 10)

	if resp["records"] == nil {
		t.Fatal("expected records in response")
	}
	if resp["total"] != 1 {
		t.Fatalf("expected total 1, got %v", resp["total"])
	}
	if resp["page"] != 1 {
		t.Fatalf("expected page 1, got %v", resp["page"])
	}
	if resp["limit"] != 10 {
		t.Fatalf("expected limit 10, got %v", resp["limit"])
	}
	if resp["total_pages"] != 1 {
		t.Fatalf("expected total_pages 1, got %v", resp["total_pages"])
	}

	recs, ok := resp["records"].([]models.LoginHistory)
	if !ok {
		t.Fatalf("expected records to be []models.LoginHistory, got %T", resp["records"])
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
}

func TestPaginationResponse_TotalPages(t *testing.T) {
	tests := []struct {
		total     int
		limit     int
		wantPages int
	}{
		{0, 10, 0},
		{5, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{25, 10, 3},
		{30, 10, 3},
		{31, 10, 4},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.total)+"_total_"+strconv.Itoa(tt.limit)+"_limit", func(t *testing.T) {
			resp := buildPaginationResponse([]models.LoginHistory{}, tt.total, 1, tt.limit)
			if resp["total_pages"] != tt.wantPages {
				t.Fatalf("expected %d total_pages, got %v", tt.wantPages, resp["total_pages"])
			}
		})
	}
}

func TestGetMyLoginHistoryAPI_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/me/login-history", func(c *gin.Context) {
		// Use a random UUID that won't have login history records
		// The service should return empty results, not error
		c.Set("admin_id", uuid.New().String())
		GetMyLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/me/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even with no records, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	records, ok := resp["records"].([]interface{})
	if !ok {
		// Could be nil or empty array
		if resp["records"] != nil {
			t.Fatalf("expected records to be array or nil, got %T", resp["records"])
		}
	} else if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}

	if resp["total"] == nil {
		t.Fatal("expected total field")
	}
}

func TestGetAllLoginHistoryAPI_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/login-history", func(c *gin.Context) {
		c.Set("admin_id", uuid.New().String())
		c.Set("admin_role", models.RoleSuperAdmin)
		GetAllLoginHistoryAPI(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/login-history", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even with no records, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	records, ok := resp["records"].([]interface{})
	if !ok {
		if resp["records"] != nil {
			t.Fatalf("expected records to be array or nil, got %T", resp["records"])
		}
	} else if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}
