package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/services"
)

func SettingsPage(c *gin.Context) {
	data := adminNavData(c)
	data["title"] = "Settings - Project Syrup"

	adminIDStr, exists := c.Get("admin_id")
	if exists {
		if idStr, ok := adminIDStr.(string); ok {
			if id, err := uuid.Parse(idStr); err == nil {
				if admin, err := services.GetAdminByID(id); err == nil {
					data["Timezone"] = admin.Timezone
				}
			}
		}
	}

	renderers["settings.html"].Render(c, "settings.html", data)
}

func UpdateTimezoneAPI(c *gin.Context) {
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

	var req struct {
		Timezone string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Timezone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "timezone is required"})
		return
	}

	if _, err := time.LoadLocation(req.Timezone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timezone"})
		return
	}

	admin, err := services.UpdateAdminTimezone(adminID, req.Timezone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}
