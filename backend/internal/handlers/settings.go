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

func GetWhoisSettingsAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	_, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	value, err := services.GetSetting("whois_server")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"whois_server": value})
}

func UpdateWhoisSettingsAPI(c *gin.Context) {
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
		WhoisServer string `json:"whois_server"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.WhoisServer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "whois_server is required"})
		return
	}

	if err := services.SetSetting("whois_server", req.WhoisServer, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"whois_server": req.WhoisServer})
}
