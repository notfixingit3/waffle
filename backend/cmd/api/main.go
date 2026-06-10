package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
	"github.com/syrup/backend/migrations"
)

var Version = "v0.1.21"

var roleHierarchy = map[string]int{
	"super_admin":    3,
	"admin":          2,
	"waffle_manager": 1,
}

func initLogger() {
	if strings.ToLower(os.Getenv("GIN_MODE")) == "release" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	}
}

func main() {
	initLogger()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8383"
	}

	database, err := db.Connect()
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.RunMigrations(database, migrations.FS); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	log.Println("backfilling users...")
	backfilledCount, backfillErr := services.BackfillUsers()
	if backfillErr != nil {
		slog.Error("Backfill users failed", "error", backfillErr)
	} else {
		log.Printf("backfilled %d users\n", backfilledCount)
	}

	services.PurgeOldEntries()

	ws.InitHub()

	r := gin.New()
	proxies := parseTrustedProxies()
	if err := r.SetTrustedProxies(proxies); err != nil {
		slog.Warn("Failed to set trusted proxies, falling back to defaults", "error", err)
		r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}) // #nosec G104
	}
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.Use(middleware.SecurityHeaders())
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
		slog.Error("Failed to load base templates", "error", err)
		os.Exit(1)
	}

	// Create per-page renderers by cloning base and adding page template
	pageRenderers := make(map[string]*renderer.Renderer)
	pageTemplates := []string{
		"templates/pages/public/home.html",
		"templates/pages/public/waffles.html",
		"templates/pages/public/waffle_detail.html",
		"templates/pages/public/buyer_stats.html",
		"templates/pages/public/about.html",
		"templates/pages/admin/login.html",
		"templates/pages/admin/reset_password.html",
		"templates/pages/admin/dashboard.html",
		"templates/pages/admin/admins.html",
		"templates/pages/admin/waffle_new.html",
		"templates/pages/admin/waffle_edit.html",
		"templates/pages/admin/waffle_manage.html",
		"templates/pages/admin/reports.html",
		"templates/pages/admin/settings.html",
		"templates/pages/admin/login_history.html",
		"templates/pages/admin/audit_log.html",
		"templates/pages/admin/users.html",
	}
	for _, page := range pageTemplates {
		clone, err := baseTmpl.Clone()
		if err != nil {
			slog.Error("Failed to clone renderer", "error", err)
			os.Exit(1)
		}
		if err := clone.AddFromFiles(page); err != nil {
			slog.Error("Failed to load page template", "template", page, "error", err)
			os.Exit(1)
		}
		name := filepath.Base(page)
		pageRenderers[name] = clone
	}
	// Clone base for direct base.html rendering (e.g., buyer stats page)
	baseClone, err := baseTmpl.Clone()
	if err != nil {
		slog.Error("Failed to clone base renderer", "error", err)
		os.Exit(1)
	}
	pageRenderers["base.html"] = baseClone
	handlers.InitRenderers(pageRenderers)
	handlers.AppVersion = Version

	// Serve embedded static files (CSS, JS, images, manifest)
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		slog.Error("Failed to create static sub-filesystem", "error", err)
		os.Exit(1)
	}
	r.StaticFS("/static", http.FS(staticSub))

	r.GET("/ws/:slug", ws.HandleWebSocketUpgrade)

	r.GET("/", handlers.HomePage)
	r.GET("/waffles", handlers.WaffleListPage)
	r.GET("/waffle/:slug", handlers.WaffleDetailPage)
	r.GET("/buyer/:handle", handlers.BuyerStatsPage)
	r.GET("/about", handlers.AboutPage)

	r.GET("/health", middleware.RequestID(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.GET("/ready", middleware.RequestID(), func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			slog.Warn("Readiness check failed", "error", err, "request_id", c.GetString("request_id"))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"db":     "disconnected",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"db":     "connected",
		})
	})

	r.GET("/admin/login", handlers.LoginPage)
	r.POST("/admin/login", middleware.AuthRateLimit, handlers.LoginPost)
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
	adminPages.GET("/reports", handlers.ReportsPage)
	adminPages.GET("/settings", handlers.SettingsPage)
	adminPages.GET("/login-history", handlers.LoginHistoryPage)
	adminPages.GET("/audit", middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), handlers.AuditLogPage)
	adminPages.GET("/users", handlers.UsersListPage)

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
	claims.POST("/random", middleware.RateLimitClaims, createRandomClaim)

	buyers := api.Group("/buyers")
	buyers.GET("/:handle/stats", handlers.GetBuyerStats)
	buyers.GET("/:handle/history", handlers.GetBuyerHistory)

	admin := api.Group("/admin")
	admin.POST("/login", middleware.AuthRateLimit, adminLogin)
	admin.POST("/forgot-password", forgotPassword)
	admin.POST("/reset-password", resetPassword)

	adminAuth := admin.Group("", middleware.RequireAuth)
	adminAuth.GET("/me", getCurrentAdmin)
	adminAuth.GET("/me/login-history", handlers.GetMyLoginHistoryAPI)
	adminAuth.POST("/change-password", changePassword)
	adminAuth.PATCH("/me/timezone", handlers.UpdateTimezoneAPI)
	adminAuth.PATCH("/me/profile", handlers.UpdateProfileAPI)
	adminAuth.GET("/login-history", handlers.GetAllLoginHistoryAPI)
	adminAuth.GET("/audit", middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), handlers.GetAuditLogAPI)
	adminAuth.GET("/audit/:id", middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), handlers.GetAuditLogEntryAPI)
	adminAuth.GET("/audit/export", middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin), exportAuditCSV)
	adminAuth.GET("/settings/whois-server", handlers.GetWhoisSettingsAPI)
	adminAuth.GET("/users", listUsers)

	adminSuperSettings := admin.Group("/settings", middleware.RequireAuth, middleware.RequireSuperAdmin)
	adminSuperSettings.GET("/", handlers.GetAllSettingsAPI)
	adminSuperSettings.PATCH("/:key", handlers.UpdateSettingAPI)
	adminSuperSettings.PATCH("/whois-server", handlers.UpdateWhoisSettingsAPI)

	adminUsers := admin.Group("/admins", middleware.RequireAuth, middleware.RequireSuperAdmin)
	adminUsers.GET("/", listAdmins)
	adminUsers.GET("/:id", getAdmin)
	adminUsers.POST("/", createAdmin)
	adminUsers.PATCH("/:id", updateAdmin)
	adminUsers.DELETE("/:id", deactivateAdmin)
	adminUsers.PATCH("/:id/password", handlers.ResetAdminPasswordAPI)
	adminUsers.PATCH("/:id/profile", handlers.UpdateAdminProfileAPI)

	adminWaffles := admin.Group("/waffles", middleware.RequireAuth)
	adminWaffles.POST("/", createWaffle)
	adminWaffles.GET("/", listWaffles)
	adminWaffles.PATCH("/:id", updateWaffle)
	adminWaffles.POST("/:id/winner", handlers.SetWinnerAPI)

	adminManagerAPI := admin.Group("/waffles", middleware.RequireAuth, middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin))
	adminManagerAPI.POST("/:id/archive", archiveWaffle)
	adminManagerAPI.POST("/:id/unarchive", unarchiveWaffle)
	adminManagerAPI.DELETE("/:id", deleteWaffle)
	adminManagerAPI.POST("/:id/clear-winner", handlers.ClearWinnerAPI)
	adminManagerAPI.POST("/:id/change-winner", handlers.ChangeWinnerAPI)

	adminReports := admin.Group("/reports", middleware.RequireAuth)
	adminReports.GET("/drought", getDroughtList)
	adminReports.GET("/power-buyers", getPowerBuyers)
	adminReports.GET("/monthly-activity", getMonthlyActivity)
	adminReports.GET("/spot-velocity", getSpotVelocity)

	adminSpots := admin.Group("/spots", middleware.RequireAuth)
	adminSpots.POST("/:id/pay", handlers.MarkSpotPaidAPI)
	adminSpots.POST("/:id/release", handlers.ReleaseSpotAPI)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("Server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	ws.GetHub().Stop()

	slog.Info("server stopped")
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

	c.Writer.WriteString("spot_number,status,instagram_handle,claimed_at,paid_at\n") // #nosec G104 — CSV header write failure will cascade to HTTP response error; no recovery possible
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

func exportAuditCSV(c *gin.Context) {
	var filters services.AuditLogFilters
	filters.Action = c.Query("action")
	filters.TargetType = c.Query("target_type")

	if adminIDStr := c.Query("admin_id"); adminIDStr != "" {
		if adminID, err := uuid.Parse(adminIDStr); err == nil {
			filters.AdminID = &adminID
		}
	}

	if fromStr := c.Query("from"); fromStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			fromTime, err = time.Parse("2006-01-02", fromStr)
		}
		if err == nil {
			filters.From = &fromTime
		}
	}

	if toStr := c.Query("to"); toStr != "" {
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			toTime, err = time.Parse("2006-01-02", toStr)
		}
		if err == nil {
			filters.To = &toTime
		}
	}

	filters.Page = 1
	filters.Limit = 10000

	entries, _, err := services.QueryAudit(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query audit log"})
		return
	}

	timestamp := time.Now().Format("20060102-150405")
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"audit-log-%s.csv\"", timestamp))

	c.Writer.WriteString("id,admin_id,action,target_type,target_id,details,ip_address,created_at\n") // #nosec G104
	for _, entry := range entries {
		details := entry.Details
		if strings.Contains(details, ",") || strings.Contains(details, "\"") {
			details = fmt.Sprintf("\"%s\"", strings.ReplaceAll(details, "\"", "\"\""))
		}
		ipAddress := entry.IPAddress
		if strings.Contains(ipAddress, ",") || strings.Contains(ipAddress, "\"") {
			ipAddress = fmt.Sprintf("\"%s\"", strings.ReplaceAll(ipAddress, "\"", "\"\""))
		}
		fmt.Fprintf(c.Writer, "%s,%s,%s,%s,%s,%s,%s,%s\n",
			entry.ID.String(),
			entry.AdminID.String(),
			entry.Action,
			entry.TargetType,
			entry.TargetID,
			details,
			ipAddress,
			entry.CreatedAt.Format(time.RFC3339),
		)
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
		slog.Error("Failed to get waffle for broadcast", "error", err, "request_id", c.GetString("request_id"))
	} else {
		handle := services.NormalizeInstagramHandle(req.InstagramHandle)
		for _, spotNum := range req.Spots {
			ws.BroadcastSpotUpdate(waffle.Slug, spotNum, string(models.SpotStatusPending), handle)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "spots claimed successfully"})
}

func createRandomClaim(c *gin.Context) {
	var req models.RandomClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	waffleID, err := uuid.Parse(req.WaffleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid waffle id"})
		return
	}

	if req.Count <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "count must be greater than 0"})
		return
	}

	spotNumbers, err := services.ClaimRandomSpots(waffleID, req.Count, req.InstagramHandle)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not active") ||
			strings.Contains(err.Error(), "required") ||
			strings.Contains(err.Error(), "greater than 0") ||
			strings.Contains(err.Error(), "no available spots") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	waffle, err := services.GetWaffleByID(waffleID)
	if err != nil {
		slog.Error("Failed to get waffle for broadcast", "error", err, "request_id", c.GetString("request_id"))
	} else {
		handle := services.NormalizeInstagramHandle(req.InstagramHandle)
		for _, spotNum := range spotNumbers {
			ws.BroadcastSpotUpdate(waffle.Slug, spotNum, string(models.SpotStatusPending), handle)
		}
	}

	claimedCount := len(spotNumbers)
	msg := fmt.Sprintf("Claimed %d spots", claimedCount)
	if claimedCount < req.Count {
		msg = fmt.Sprintf("Claimed %d of %d requested spots", claimedCount, req.Count)
	}

	c.JSON(http.StatusOK, gin.H{
		"claimed_count":   claimedCount,
		"requested_count": req.Count,
		"spot_numbers":    spotNumbers,
		"message":         msg,
	})
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

	handlers.RecordAudit(c, "create_waffle", "waffle", waffle.ID.String(), "created waffle '"+waffle.Title+"'")

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

	handlers.RecordAudit(c, "update_waffle", "waffle", id.String(), "updated waffle '"+waffle.Title+"'")

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

	handlers.RecordAudit(c, "archive_waffle", "waffle", id.String(), "waffle archived")

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

	handlers.RecordAudit(c, "unarchive_waffle", "waffle", id.String(), "waffle unarchived")

	c.JSON(http.StatusOK, gin.H{"message": "waffle unarchived"})
}

func deleteWaffle(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := handlers.VerifyPasswordConfirmation(c); err != nil {
		if err.Error() == "password confirmation required" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "password confirmation required"})
		} else if err.Error() == "invalid password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		}
		return
	}

	if err := services.DeleteWaffle(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	handlers.RecordAudit(c, "delete_waffle", "waffle", id.String(), "waffle deleted permanently")

	c.JSON(http.StatusOK, gin.H{"message": "waffle deleted permanently"})
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
			to = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
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
		services.RecordFailedLoginAttempt(c.ClientIP(), req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	services.ResetLoginAttempts(c.ClientIP(), req.Username)

	token, err := services.GenerateAdminToken(admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	loginID, err := services.RecordLogin(admin.ID.String(), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		slog.Error("RecordLogin error", "error", err, "request_id", c.GetString("request_id"))
	} else if !services.IsPrivateIP(c.ClientIP()) {
		go func(id uuid.UUID) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("WHOIS enrichment panic", "error", r)
				}
			}()
			if enrichErr := services.EnrichLoginWithWHOIS(id); enrichErr != nil {
				slog.Error("WHOIS enrichment error", "error", enrichErr)
			}
		}(loginID)
	}

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

	_, err = services.CreatePasswordResetToken(admin.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "if the email exists, a reset link has been sent",
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

	if err := services.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpdateAdminPassword(adminID, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	services.MarkResetTokenUsed(req.Token) // #nosec G104 — MarkResetTokenUsed is best-effort cleanup after successful password reset

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

	if err := services.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := services.GetAdminPasswordHash(adminID)
	if err != nil || !services.CheckPassword(req.CurrentPassword, hash) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
		return
	}

	if err := services.UpdateAdminPassword(adminID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	handlers.RecordAudit(c, "change_password", "admin", adminID.String(), "password changed")

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

func getAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	admin, err := services.GetAdminByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	c.JSON(http.StatusOK, admin)
}

func createAdmin(c *gin.Context) {
	var req models.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	if err := services.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, err := services.CreateAdmin(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	handlers.RecordAudit(c, "create_admin", "admin", admin.ID.String(), "created admin '"+admin.Username+"' with role "+admin.Role)

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

	if req.Role != nil {
		targetAdmin, err := services.GetAdminByID(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "admin not found"})
			return
		}
		targetLevel := roleHierarchy[targetAdmin.Role]
		newLevel := roleHierarchy[*req.Role]
		if newLevel < targetLevel {
	if err := handlers.VerifyPasswordConfirmation(c); err != nil {
		if err.Error() == "password confirmation required" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "password confirmation required"})
				} else if err.Error() == "invalid password" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
				} else {
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				}
				return
			}
		}
	}

	admin, err := services.UpdateAdmin(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	handlers.RecordAudit(c, "update_admin", "admin", id.String(), "updated admin '"+admin.Username+"'")

	c.JSON(http.StatusOK, admin)
}

func deactivateAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := handlers.VerifyPasswordConfirmation(c); err != nil {
		if err.Error() == "password confirmation required" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "password confirmation required"})
		} else if err.Error() == "invalid password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		}
		return
	}

	admin, err := services.UpdateAdmin(id, models.UpdateAdminRequest{Active: boolPtr(false)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	handlers.RecordAudit(c, "deactivate_admin", "admin", id.String(), "deactivated admin '"+admin.Username+"'")

	c.JSON(http.StatusOK, admin)
}

func parseTrustedProxies() []string {
	envVal := os.Getenv("TRUSTED_PROXIES")
	if envVal == "" {
		return []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	}

	parts := strings.Split(envVal, ",")
	var proxies []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(p); err != nil {
			slog.Warn("Invalid CIDR in TRUSTED_PROXIES, skipping", "cidr", p, "error", err)
			continue
		}
		proxies = append(proxies, p)
	}

	if len(proxies) == 0 {
		slog.Warn("TRUSTED_PROXIES contained no valid CIDRs, falling back to defaults")
		return []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	}

	return proxies
}

func listUsers(c *gin.Context) {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "25"))
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 100 {
		perPage = 100
	}

	limit := perPage
	offset := (page - 1) * perPage

	users, total, err := services.ListUsers(search, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":    users,
		"page":     page,
		"per_page": perPage,
		"total":    total,
	})
}

func boolPtr(b bool) *bool {
	return &b
}
