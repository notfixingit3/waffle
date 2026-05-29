package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/services"
)

func AuditLogPage(c *gin.Context) {
	page, limit := parsePagination(c)
	filters, err := auditFiltersFromRequest(c, page, limit)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid filters")
		return
	}

	entries, total, err := services.QueryAudit(filters)
	if err != nil {
		entries = []models.AuditLog{}
		total = 0
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	adminMap := auditAdminMap(entries)
	data := adminNavData(c)
	data["title"] = "Audit Log - Project Syrup"
	data["Entries"] = entries
	data["Total"] = total
	data["Page"] = page
	data["Limit"] = limit
	data["TotalPages"] = totalPages
	data["AdminMap"] = adminMap
	data["Action"] = c.Query("action")
	data["TargetType"] = c.Query("target_type")
	data["From"] = c.Query("from")
	data["To"] = c.Query("to")

	renderers["audit_log.html"].Render(c, "audit_log.html", data)
}

func GetAuditLogAPI(c *gin.Context) {
	page, limit := parsePagination(c)
	filters, err := auditFiltersFromRequest(c, page, limit)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	entries, total, err := services.QueryAudit(filters)
	if err != nil {
		ServerErrorResponse(c, err)
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	c.JSON(http.StatusOK, gin.H{
		"entries":     entries,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func GetAuditLogEntryAPI(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid audit id")
		return
	}
	entry, err := services.GetAuditByID(id)
	if err != nil {
		ServerErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

func auditFiltersFromRequest(c *gin.Context, page, limit int) (services.AuditLogFilters, error) {
	filters := services.AuditLogFilters{
		Action:     c.Query("action"),
		TargetType: c.Query("target_type"),
		Page:       page,
		Limit:      limit,
	}
	if adminIDStr := c.Query("admin_id"); adminIDStr != "" {
		adminID, err := uuid.Parse(adminIDStr)
		if err != nil {
			return filters, err
		}
		filters.AdminID = &adminID
	}
	if from := c.Query("from"); from != "" {
		parsed, err := time.Parse("2006-01-02", from)
		if err != nil {
			return filters, err
		}
		filters.From = &parsed
	}
	if to := c.Query("to"); to != "" {
		parsed, err := time.Parse("2006-01-02", to)
		if err != nil {
			return filters, err
		}
		endOfDay := parsed.Add(24*time.Hour - time.Nanosecond)
		filters.To = &endOfDay
	}
	return filters, nil
}

func auditAdminMap(entries []models.AuditLog) map[string]string {
	adminMap := make(map[string]string)
	for _, entry := range entries {
		idStr := entry.AdminID.String()
		if _, ok := adminMap[idStr]; ok {
			continue
		}
		admin, err := services.GetAdminByID(entry.AdminID)
		if err != nil {
			adminMap[idStr] = idStr
			continue
		}
		if admin.DisplayName != nil && *admin.DisplayName != "" {
			adminMap[idStr] = *admin.DisplayName
		} else {
			adminMap[idStr] = admin.Username
		}
	}
	return adminMap
}
