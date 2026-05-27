package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/services"
	ws "github.com/syrup/backend/internal/websocket"
)

func adminNavData(c *gin.Context) gin.H {
	adminIDStr, _ := c.Get("admin_id")
	role, _ := c.Get("admin_role")

	displayName := "Admin"
	if idStr, ok := adminIDStr.(string); ok {
		if id, err := uuid.Parse(idStr); err == nil {
			if admin, err := services.GetAdminByID(id); err == nil && admin.DisplayName != nil {
				displayName = *admin.DisplayName
			} else if err == nil {
				displayName = admin.Username
			}
		}
	}

	roleStr := "admin"
	if r, ok := role.(string); ok {
		roleStr = r
	}

	csrfToken := ""
	if t, err := c.Cookie("csrf_token"); err == nil {
		csrfToken = t
	}

	total, active, _ := services.CountWaffles()

	timezone := "UTC"
	if idStr, ok := adminIDStr.(string); ok {
		if id, err := uuid.Parse(idStr); err == nil {
			if admin, err := services.GetAdminByID(id); err == nil && admin.Timezone != "" {
				timezone = admin.Timezone
			}
		}
	}

	return gin.H{
		"Role":          roleStr,
		"DisplayName":   displayName,
		"CurrentPath":   c.Request.URL.Path,
		"CSRFToken":     csrfToken,
		"Version":       AppVersion,
		"DevMode":       strings.Contains(strings.ToLower(AppVersion), "dev"),
		"TotalWaffles":  total,
		"ActiveWaffles": active,
		"ServerTime":    time.Now().UTC().Format("3:04 PM"),
		"Timezone":      timezone,
	}
}

func AdminDashboard(c *gin.Context) {
	archived := c.Query("archived") == "true"

	waffles, err := services.ListWaffles(archived)
	if err != nil {
		waffles = []models.Waffle{}
	}

	data := adminNavData(c)
	data["title"] = "Dashboard - Project Syrup"
	data["Waffles"] = waffles
	data["Archived"] = archived
	renderers["dashboard.html"].Render(c, "dashboard.html", data)
}

func ManageWafflePage(c *gin.Context) {
	slug := c.Param("slug")

	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Waffle not found")
		return
	}

	spots, err := services.GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load spots")
		return
	}

	stats, err := services.GetWaffleStats(waffle.ID)
	if err != nil {
		stats = map[string]interface{}{}
	}

	data := adminNavData(c)
	data["title"] = waffle.Title + " - Admin - Project Syrup"
	data["waffle"] = waffle
	data["spots"] = spots
	data["stats"] = stats

	renderers["waffle_manage.html"].Render(c, "waffle_manage.html", data)
}

func EditWafflePage(c *gin.Context) {
	slug := c.Param("slug")

	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Waffle not found")
		return
	}

	data := adminNavData(c)
	data["title"] = "Edit Waffle - " + waffle.Title + " - Project Syrup"
	data["waffle"] = waffle
	data["Slug"] = slug

	renderers["waffle_edit.html"].Render(c, "waffle_edit.html", data)
}

func EditWafflePost(c *gin.Context) {
	if !validateCSRF(c) {
		c.String(http.StatusBadRequest, "Invalid or missing CSRF token")
		return
	}

	slug := c.Param("slug")

	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Waffle not found")
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	paymentInfo := strings.TrimSpace(c.PostForm("payment_info"))
	spotPriceStr := c.PostForm("spot_price")

	var errors gin.H
	var hasError bool

	if title == "" {
		errors = gin.H{"Title": "Title is required"}
		hasError = true
	}

	spotPrice, err := strconv.Atoi(spotPriceStr)
	if err != nil || spotPrice <= 0 {
		if errors == nil {
			errors = gin.H{}
		}
		errors["SpotPrice"] = "Price per spot must be greater than 0"
		hasError = true
	}

	mediaLinks := c.PostFormArray("instagram_media_links[]")
	var cleanLinks []string
	for _, link := range mediaLinks {
		link = strings.TrimSpace(link)
		if link != "" {
			cleanLinks = append(cleanLinks, link)
		}
	}

	if hasError {
		if errors == nil {
			errors = gin.H{}
		}
		data := adminNavData(c)
		data["title"] = "Edit Waffle - " + waffle.Title + " - Project Syrup"
		data["waffle"] = waffle
		data["Slug"] = slug
		data["Error"] = "Please fix the errors below"
		data["Errors"] = errors
		data["Title"] = title
		data["Description"] = description
		data["SpotPrice"] = spotPriceStr
		data["PaymentInfo"] = paymentInfo
		data["InstagramMediaLinks"] = cleanLinks
		renderers["waffle_edit.html"].Render(c, "waffle_edit.html", data)
		return
	}

	var descPtr, paymentPtr *string
	if description != "" {
		descPtr = &description
	}
	if paymentInfo != "" {
		paymentPtr = &paymentInfo
	}

	req := models.UpdateWaffleRequest{
		Title:               title,
		Description:         descPtr,
		SpotPrice:           spotPrice,
		PaymentInfo:         paymentPtr,
		InstagramMediaLinks: cleanLinks,
	}

	_, err = services.UpdateWaffle(waffle.ID, req)
	if err != nil {
		data := adminNavData(c)
		data["title"] = "Edit Waffle - " + waffle.Title + " - Project Syrup"
		data["waffle"] = waffle
		data["Slug"] = slug
		data["Error"] = "Failed to update waffle: " + err.Error()
		data["Title"] = title
		data["Description"] = description
		data["SpotPrice"] = spotPriceStr
		data["PaymentInfo"] = paymentInfo
		data["InstagramMediaLinks"] = cleanLinks
		renderers["waffle_edit.html"].Render(c, "waffle_edit.html", data)
		return
	}

	c.Redirect(http.StatusFound, "/admin/waffles/"+slug)
}

func MarkSpotPaidAPI(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := services.MarkSpotPaid(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	spot, err := services.GetSpotByID(id)
	if err == nil {
		waffle, err := services.GetWaffleByID(spot.WaffleID)
		if err == nil {
			handle := ""
			if spot.ClaimedByHandle != nil {
				handle = *spot.ClaimedByHandle
			}
			ws.BroadcastSpotUpdate(waffle.Slug, spot.Number, string(spot.Status), handle)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "spot marked as paid"})
}

func ReleaseSpotAPI(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	spot, _ := services.GetSpotByID(id)

	if err := services.ReleaseSpot(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if spot != nil {
		waffle, err := services.GetWaffleByID(spot.WaffleID)
		if err == nil {
			ws.BroadcastSpotUpdate(waffle.Slug, spot.Number, string(models.SpotStatusAvailable), "")
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "spot released"})
}

func SetWinnerAPI(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.SetWinnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := services.SetWinner(id, req.WinningSpotNumber); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	waffle, err := services.GetWaffleByID(id)
	if err == nil {
		ws.BroadcastWaffleCompleted(waffle.Slug, req.WinningSpotNumber)
	}

	c.JSON(http.StatusOK, gin.H{"message": "winner set successfully"})
}

func NewWafflePage(c *gin.Context) {
	data := adminNavData(c)
	data["title"] = "New Waffle - Project Syrup"
	data["Templates"] = []gin.H{
		{"Name": "Small (10 spots, $2)", "TotalSpots": 10, "SpotPrice": 2},
		{"Name": "Standard (25 spots, $5)", "TotalSpots": 25, "SpotPrice": 5},
		{"Name": "Medium (50 spots, $3)", "TotalSpots": 50, "SpotPrice": 3},
		{"Name": "Large (100 spots, $2)", "TotalSpots": 100, "SpotPrice": 2},
	}
	renderers["waffle_new.html"].Render(c, "waffle_new.html", data)
}

func CreateWafflePost(c *gin.Context) {
	if !validateCSRF(c) {
		renderNewWaffleForm(c, gin.H{"Error": "Invalid or missing CSRF token. Please try again."})
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	paymentInfo := strings.TrimSpace(c.PostForm("payment_info"))

	totalSpotsStr := c.PostForm("total_spots")
	spotPriceStr := c.PostForm("spot_price")

	var errors gin.H
	var hasError bool

	if title == "" {
		errors = gin.H{"Title": "Title is required"}
		hasError = true
	}

	totalSpots, err := strconv.Atoi(totalSpotsStr)
	if err != nil || totalSpots < 2 {
		if errors == nil {
			errors = gin.H{}
		}
		errors["TotalSpots"] = "Total spots must be at least 2"
		hasError = true
	}

	spotPrice, err := strconv.Atoi(spotPriceStr)
	if err != nil || spotPrice <= 0 {
		if errors == nil {
			errors = gin.H{}
		}
		errors["SpotPrice"] = "Price per spot must be greater than 0"
		hasError = true
	}

	mediaLinks := c.PostFormArray("instagram_media_links[]")
	var cleanLinks []string
	for _, link := range mediaLinks {
		link = strings.TrimSpace(link)
		if link != "" {
			cleanLinks = append(cleanLinks, link)
		}
	}

	if hasError {
		if errors == nil {
			errors = gin.H{}
		}
		errData := gin.H{
			"Title":               title,
			"Description":         description,
			"TotalSpots":          totalSpotsStr,
			"SpotPrice":           spotPriceStr,
			"PaymentInfo":         paymentInfo,
			"InstagramMediaLinks": cleanLinks,
			"Errors":              errors,
		}
		renderNewWaffleForm(c, errData)
		return
	}

	var descPtr, paymentPtr *string
	if description != "" {
		descPtr = &description
	}
	if paymentInfo != "" {
		paymentPtr = &paymentInfo
	}

	req := models.CreateWaffleRequest{
		Title:               title,
		Description:         descPtr,
		TotalSpots:          totalSpots,
		SpotPrice:           spotPrice,
		PaymentInfo:         paymentPtr,
		InstagramMediaLinks: cleanLinks,
	}

	waffle, err := services.CreateWaffle(req)
	if err != nil {
		errData := gin.H{
			"Title":               title,
			"Description":         description,
			"TotalSpots":          totalSpotsStr,
			"SpotPrice":           spotPriceStr,
			"PaymentInfo":         paymentInfo,
			"InstagramMediaLinks": cleanLinks,
			"Error":               "Failed to create waffle: " + err.Error(),
		}
		renderNewWaffleForm(c, errData)
		return
	}

	c.Redirect(http.StatusFound, "/admin/waffles/"+waffle.Slug)
}

func renderNewWaffleForm(c *gin.Context, extra gin.H) {
	data := adminNavData(c)
	data["title"] = "New Waffle - Project Syrup"
	data["Templates"] = []gin.H{
		{"Name": "Small (10 spots, $2)", "TotalSpots": 10, "SpotPrice": 2},
		{"Name": "Standard (25 spots, $5)", "TotalSpots": 25, "SpotPrice": 5},
		{"Name": "Medium (50 spots, $3)", "TotalSpots": 50, "SpotPrice": 3},
		{"Name": "Large (100 spots, $2)", "TotalSpots": 100, "SpotPrice": 2},
	}
	for k, v := range extra {
		data[k] = v
	}
	renderers["waffle_new.html"].Render(c, "waffle_new.html", data)
}

func ArchiveWafflePost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	if err := services.ArchiveWaffle(id, true); err != nil {
		log.Printf("Failed to archive waffle: %v", err)
	}

	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/admin/dashboard"
	}
	c.Redirect(http.StatusFound, referer)
}

func UnarchiveWafflePost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	if err := services.ArchiveWaffle(id, false); err != nil {
		log.Printf("Failed to unarchive waffle: %v", err)
	}

	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/admin/dashboard"
	}
	c.Redirect(http.StatusFound, referer)
}

func DeleteWafflePost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	if c.PostForm("confirm") != "DELETE" {
		referer := c.Request.Header.Get("Referer")
		if referer == "" {
			referer = "/admin/dashboard"
		}
		c.Redirect(http.StatusFound, referer)
		return
	}

	if err := services.DeleteWaffle(id); err != nil {
		log.Printf("Failed to delete waffle: %v", err)
	}

	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/admin/dashboard"
	}
	c.Redirect(http.StatusFound, referer)
}

func ReportsPage(c *gin.Context) {
	data := adminNavData(c)
	data["title"] = "Reports - Project Syrup"
	renderers["reports.html"].Render(c, "reports.html", data)
}

func AdminManagementPage(c *gin.Context) {
	admins, err := services.ListAdmins()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load admins")
		return
	}

	data := adminNavData(c)
	if adminIDStr, ok := c.Get("admin_id"); ok {
		if idStr, ok2 := adminIDStr.(string); ok2 {
			data["CurrentAdminID"] = idStr
		}
	}
	data["title"] = "Admin Management - Project Syrup"
	data["Admins"] = admins
	renderers["admins.html"].Render(c, "admins.html", data)
}

func validateCreateAdminForm(username, email, password string) []string {
	var errors []string
	if username == "" {
		errors = append(errors, "Username is required")
	}
	if email == "" {
		errors = append(errors, "Email is required")
	}
	if password == "" {
		errors = append(errors, "Password is required")
	} else if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters")
	}
	return errors
}

func CreateAdminPost(c *gin.Context) {
	if !validateCSRF(c) {
		renderAdminManagementWithError(c, "Invalid or missing CSRF token. Please try again.", nil)
		return
	}

	username := strings.TrimSpace(c.PostForm("username"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	displayName := strings.TrimSpace(c.PostForm("display_name"))
	role := c.PostForm("role")

	errors := validateCreateAdminForm(username, email, password)
	if role != models.RoleAdmin && role != models.RoleSuperAdmin && role != models.RoleWaffleManager {
		role = models.RoleAdmin
	}

	if len(errors) > 0 {
		renderAdminManagementWithError(c, strings.Join(errors, "; "), gin.H{
			"Username":    username,
			"Email":       email,
			"DisplayName": displayName,
			"Role":        role,
		})
		return
	}

	var displayNamePtr *string
	if displayName != "" {
		displayNamePtr = &displayName
	}

	_, err := services.CreateAdmin(models.CreateAdminRequest{
		Username:    username,
		Email:       email,
		Password:    password,
		DisplayName: displayNamePtr,
		Role:        role,
	})
	if err != nil {
		renderAdminManagementWithError(c, "Failed to create admin: "+err.Error(), gin.H{
			"Username":    username,
			"Email":       email,
			"DisplayName": displayName,
			"Role":        role,
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/admins")
}

func renderAdminManagementWithError(c *gin.Context, msg string, formData gin.H) {
	admins, err := services.ListAdmins()
	if err != nil {
		admins = []models.Admin{}
	}
	data := adminNavData(c)
	if adminIDStr, ok := c.Get("admin_id"); ok {
		if idStr, ok2 := adminIDStr.(string); ok2 {
			data["CurrentAdminID"] = idStr
		}
	}
	data["title"] = "Admin Management - Project Syrup"
	data["Admins"] = admins
	data["Error"] = msg
	for k, v := range formData {
		data[k] = v
	}
	renderers["admins.html"].Render(c, "admins.html", data)
}

func UpdateAdminRoleAPI(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	admin, err := services.UpdateAdmin(id, models.UpdateAdminRequest{Role: &req.Role})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func DeactivateAdminAPI(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var active = false
	admin, err := services.UpdateAdmin(id, models.UpdateAdminRequest{Active: &active})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func ResetAdminPasswordAPI(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	callerIDStr := c.GetString("admin_id")
	callerID, err := uuid.Parse(callerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid caller id"})
		return
	}

	if targetID == callerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use /api/admin/change-password to change your own password"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	if _, err := services.GetAdminByID(targetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin not found"})
		return
	}

	if err := services.UpdateAdminPassword(targetID, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
