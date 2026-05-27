package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/handlers"
	"github.com/syrup/backend/internal/middleware"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/renderer"
	"github.com/syrup/backend/internal/services"
	ws "github.com/syrup/backend/internal/websocket"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8383"
	}

	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	ws.InitHub()

	r := gin.New()
	r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	r.Use(gin.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
	}))

	// Initialize template renderer (base layout + partials only)
	baseTmpl := renderer.New(nil)
	if err := baseTmpl.AddFromFiles(
		"templates/layouts/base.html",
		"templates/layouts/admin_base.html",
		"templates/partials/*.html",
	); err != nil {
		log.Fatalf("Failed to load base templates: %v", err)
	}

	// Create per-page renderers by cloning base and adding page template
	pageRenderers := make(map[string]*renderer.Renderer)
	pageTemplates := []string{
		"templates/pages/public/home.html",
		"templates/pages/public/waffles.html",
		"templates/pages/public/waffle_detail.html",
		"templates/pages/public/buyer_stats.html",
		"templates/pages/admin/login.html",
		"templates/pages/admin/reset_password.html",
		"templates/pages/admin/dashboard.html",
		"templates/pages/admin/admins.html",
		"templates/pages/admin/waffle_new.html",
		"templates/pages/admin/waffle_edit.html",
		"templates/pages/admin/waffle_manage.html",
		"templates/pages/admin/reports.html",
	}
	for _, page := range pageTemplates {
		clone, err := baseTmpl.Clone()
		if err != nil {
			log.Fatalf("Failed to clone renderer: %v", err)
		}
		if err := clone.AddFromFiles(page); err != nil {
			log.Fatalf("Failed to load page template %s: %v", page, err)
		}
		name := filepath.Base(page)
		pageRenderers[name] = clone
	}
	// Clone base for direct base.html rendering (e.g., buyer stats page)
	baseClone, err := baseTmpl.Clone()
	if err != nil {
		log.Fatalf("Failed to clone base renderer: %v", err)
	}
	pageRenderers["base.html"] = baseClone
	handlers.InitRenderers(pageRenderers)

	// Serve embedded static files (CSS, JS, images, manifest)
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create static sub-filesystem: %v", err)
	}
	r.StaticFS("/static", http.FS(staticSub))

	r.GET("/ws/:slug", ws.HandleWebSocketUpgrade)

	r.GET("/", handlers.HomePage)
	r.GET("/waffles", handlers.WaffleListPage)
	r.GET("/waffle/:slug", handlers.WaffleDetailPage)
	r.GET("/buyer/:handle", handlers.BuyerStatsPage)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"db":     "connected",
		})
	})

	r.GET("/admin/login", handlers.LoginPage)
	r.POST("/admin/login", handlers.LoginPost)
	r.GET("/admin/forgot-password", handlers.ForgotPasswordPage)
	r.POST("/admin/forgot-password", handlers.ForgotPasswordPost)
	r.POST("/admin/logout", handlers.LogoutPost)
	r.GET("/admin", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin/dashboard")
	})

	adminPages := r.Group("/admin", middleware.RequireAuth)
	adminPages.GET("/dashboard", handlers.AdminDashboard)
	adminPages.GET("/waffles/:slug", handlers.ManageWafflePage)
	adminPages.GET("/waffles/:slug/edit", handlers.EditWafflePage)
	adminPages.POST("/waffles/:slug/edit", handlers.EditWafflePost)
	adminPages.GET("/waffles/new", handlers.NewWafflePage)
	adminPages.POST("/waffles/new", handlers.CreateWafflePost)
	adminPages.POST("/waffles/:slug/archive", handlers.ArchiveWafflePost)
	adminPages.POST("/waffles/:slug/unarchive", handlers.UnarchiveWafflePost)
	adminPages.POST("/waffles/:slug/delete", handlers.DeleteWafflePost)
	adminPages.GET("/reports", handlers.ReportsPage)

	adminSuperPages := adminPages.Group("", middleware.RequireSuperAdmin)
	adminSuperPages.GET("/admins", handlers.AdminManagementPage)
	adminSuperPages.POST("/admins", handlers.CreateAdminPost)

	api := r.Group("/api")

	waffles := api.Group("/waffles")
	waffles.GET("/", listPublicWaffles)
	waffles.GET("/:slug", getWaffle)
	waffles.GET("/:slug/spots", getSpots)
	waffles.GET("/:slug/export", exportWaffleCSV)

	claims := api.Group("/claims")
	claims.POST("/", middleware.RateLimitClaims, createClaim)

	buyers := api.Group("/buyers")
	buyers.GET("/:handle/stats", handlers.GetBuyerStats)
	buyers.GET("/:handle/history", handlers.GetBuyerHistory)

	admin := api.Group("/admin")
	admin.POST("/login", adminLogin)
	admin.POST("/forgot-password", forgotPassword)
	admin.POST("/reset-password", resetPassword)

	adminAuth := admin.Group("", middleware.RequireAuth)
	adminAuth.GET("/me", getCurrentAdmin)
	adminAuth.POST("/change-password", changePassword)

	adminUsers := admin.Group("/admins", middleware.RequireAuth, middleware.RequireSuperAdmin)
	adminUsers.GET("/", listAdmins)
	adminUsers.POST("/", createAdmin)
	adminUsers.PATCH("/:id", updateAdmin)
	adminUsers.DELETE("/:id", deactivateAdmin)
	adminUsers.PATCH("/:id/password", handlers.ResetAdminPasswordAPI)

	adminWaffles := admin.Group("/waffles", middleware.RequireAuth)
	adminWaffles.POST("/", createWaffle)
	adminWaffles.GET("/", listWaffles)
	adminWaffles.PATCH("/:id", updateWaffle)
	adminWaffles.POST("/:id/winner", handlers.SetWinnerAPI)
	adminWaffles.POST("/:id/archive", archiveWaffle)
	adminWaffles.POST("/:id/unarchive", unarchiveWaffle)
	adminWaffles.DELETE("/:id", deleteWaffle)

	adminReports := admin.Group("/reports", middleware.RequireAuth)
	adminReports.GET("/drought", getDroughtList)
	adminReports.GET("/power-buyers", getPowerBuyers)
	adminReports.GET("/monthly-activity", getMonthlyActivity)
	adminReports.GET("/spot-velocity", getSpotVelocity)

	adminSpots := admin.Group("/spots", middleware.RequireAuth)
	adminSpots.POST("/:id/pay", handlers.MarkSpotPaidAPI)
	adminSpots.POST("/:id/release", handlers.ReleaseSpotAPI)

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func listPublicWaffles(c *gin.Context) {
	waffles, err := services.ListWaffles(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"waffles": waffles})
}

func exportWaffleCSV(c *gin.Context) {
	slug := c.Param("slug")
	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "waffle not found"})
		return
	}

	spots, err := services.GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-spots.csv\"", slug))

	c.Writer.WriteString("spot_number,status,instagram_handle,claimed_at,paid_at\n")
	for _, spot := range spots {
		claimedAt := ""
		if spot.ClaimedAt != nil {
			claimedAt = spot.ClaimedAt.Format(time.RFC3339)
		}
		paidAt := ""
		if spot.PaidAt != nil {
			paidAt = spot.PaidAt.Format(time.RFC3339)
		}
		handle := ""
		if spot.ClaimedByHandle != nil {
			handle = *spot.ClaimedByHandle
		}
		fmt.Fprintf(c.Writer, "%d,%s,%s,%s,%s\n",
			spot.Number, spot.Status, handle, claimedAt, paidAt)
	}
}

func getWaffle(c *gin.Context) {
	slug := c.Param("slug")
	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "waffle not found"})
		return
	}

	stats, err := services.GetWaffleStats(waffle.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"waffle": waffle,
		"stats":  stats,
	})
}

func getSpots(c *gin.Context) {
	slug := c.Param("slug")
	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "waffle not found"})
		return
	}

	spots, err := services.GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get spots"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"spots": spots})
}

func createClaim(c *gin.Context) {
	var req models.CreateClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	waffleID, err := uuid.Parse(req.WaffleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid waffle id"})
		return
	}

	if err := services.ClaimSpots(waffleID, req.Spots, req.InstagramHandle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	waffle, err := services.GetWaffleByID(waffleID)
	if err != nil {
		log.Printf("Failed to get waffle for broadcast: %v", err)
	} else {
		handle := services.NormalizeInstagramHandle(req.InstagramHandle)
		for _, spotNum := range req.Spots {
			ws.BroadcastSpotUpdate(waffle.Slug, spotNum, string(models.SpotStatusPending), handle)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "spots claimed successfully"})
}

func createWaffle(c *gin.Context) {
	var req models.CreateWaffleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Title == "" || req.TotalSpots <= 0 || req.SpotPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, total_spots, and spot_price are required"})
		return
	}

	waffle, err := services.CreateWaffle(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, waffle)
}

func updateWaffle(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.UpdateWaffleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	waffle, err := services.UpdateWaffle(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, waffle)
}

func listWaffles(c *gin.Context) {
	includeArchived := false
	if c.Query("archived") == "true" {
		includeArchived = true
	}
	waffles, err := services.ListWaffles(includeArchived)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"waffles": waffles})
}

func archiveWaffle(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := services.ArchiveWaffle(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "waffle archived"})
}

func unarchiveWaffle(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := services.ArchiveWaffle(id, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "waffle unarchived"})
}

func deleteWaffle(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := services.DeleteWaffle(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "waffle deleted permanently"})
}

func setWinner(c *gin.Context) {
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

func markPaid(c *gin.Context) {
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

func releaseSpot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := services.ReleaseSpot(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "spot released"})
}

func parseDateRange(c *gin.Context) (time.Time, time.Time) {
	fromStr := c.DefaultQuery("from", "")
	toStr := c.DefaultQuery("to", "")

	to := time.Now()
	from := to.AddDate(0, -1, 0)

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	return from, to
}

func getDroughtList(c *gin.Context) {
	from, to := parseDateRange(c)
	entries, err := services.GetDroughtList(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

func getPowerBuyers(c *gin.Context) {
	from, to := parseDateRange(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	entries, err := services.GetPowerBuyers(from, to, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

func getMonthlyActivity(c *gin.Context) {
	from, to := parseDateRange(c)
	entries, err := services.GetMonthlyActivity(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

func getSpotVelocity(c *gin.Context) {
	status := c.Query("status")
	entries, err := services.GetSpotVelocity(status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

func adminLogin(c *gin.Context) {
	var req models.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	admin, err := services.AuthenticateAdmin(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	token, err := services.GenerateAdminToken(admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	services.RecordLogin(admin.ID)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"admin": admin,
	})
}

func forgotPassword(c *gin.Context) {
	var req models.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	admin, err := services.GetAdminByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	token, err := services.CreatePasswordResetToken(admin.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "if the email exists, a reset link has been sent",
		"token":   token,
	})
}

func resetPassword(c *gin.Context) {
	var req models.PasswordResetConfirm
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	adminID, err := services.ValidatePasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpdateAdminPassword(adminID, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	services.MarkResetTokenUsed(req.Token)

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

func getCurrentAdmin(c *gin.Context) {
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

	admin, err := services.GetAdminByID(adminID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func changePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

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

	if err := services.UpdateAdminPassword(adminID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

func listAdmins(c *gin.Context) {
	admins, err := services.ListAdmins()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

func createAdmin(c *gin.Context) {
	var req models.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username, email, and password are required"})
		return
	}

	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	admin, err := services.CreateAdmin(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, admin)
}

func updateAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	admin, err := services.UpdateAdmin(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func deactivateAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	admin, err := services.UpdateAdmin(id, models.UpdateAdminRequest{Active: boolPtr(false)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func boolPtr(b bool) *bool {
	return &b
}
