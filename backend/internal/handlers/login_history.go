package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/services"
)

var (
	getLoginHistory    = services.GetLoginHistory
	getAllLoginHistory = services.GetAllLoginHistory
)

func LoginHistoryPage(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}

	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}

	role, exists := c.Get("admin_role")
	if !exists {
		c.String(http.StatusForbidden, "role not found")
		return
	}

	roleStr, ok := role.(string)
	if !ok {
		c.String(http.StatusForbidden, "role not found")
		return
	}

	page, limit := parsePagination(c)

	records, total, err := services.GetAllLoginHistory(roleStr, adminID, page, limit)
	if err != nil {
		records = []models.LoginHistory{}
		total = 0
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	adminMap := make(map[string]string)
	for _, r := range records {
		idStr := r.AdminID.String()
		if _, ok := adminMap[idStr]; !ok {
			if admin, err := services.GetAdminByID(r.AdminID); err == nil {
				if admin.DisplayName != nil && *admin.DisplayName != "" {
					adminMap[idStr] = *admin.DisplayName
				} else {
					adminMap[idStr] = admin.Username
				}
			} else {
				adminMap[idStr] = idStr
			}
		}
	}

	data := adminNavData(c)
	data["title"] = "Login History - Project Syrup"
	data["Records"] = records
	data["Total"] = total
	data["Page"] = page
	data["Limit"] = limit
	data["TotalPages"] = totalPages
	data["AdminMap"] = adminMap

	renderers["login_history.html"].Render(c, "login_history.html", data)
}

func parsePagination(c *gin.Context) (page, limit int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return
}

func buildPaginationResponse(records []models.LoginHistory, total, page, limit int) gin.H {
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return gin.H{
		"records":     records,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	}
}

func GetMyLoginHistoryAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	page, limit := parsePagination(c)

	records, total, err := getLoginHistory(adminID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, buildPaginationResponse(records, total, page, limit))
}

func GetAllLoginHistoryAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	role, exists := c.Get("admin_role")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}

	roleStr, ok := role.(string)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}

	page, limit := parsePagination(c)

	records, total, err := getAllLoginHistory(roleStr, adminID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, buildPaginationResponse(records, total, page, limit))
}
