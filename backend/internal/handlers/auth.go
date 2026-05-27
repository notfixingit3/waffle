package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/services"
)

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		slog.Error("Failed to generate CSRF token", "error", err)
		return "", errors.New("failed to generate CSRF token")
	}
	return hex.EncodeToString(b), nil
}

func setCSRFCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		MaxAge:   3600,
		Path:     "/",
		Secure:   useSecureCookie(c),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func getCSRFToken(c *gin.Context) string {
	token, err := c.Cookie("csrf_token")
	if err != nil || token == "" {
		token, err = generateCSRFToken()
		if err != nil {
			slog.Error("Failed to generate CSRF token", "error", err, "request_id", c.GetString("request_id"))
			token = ""
		}
	}
	setCSRFCookie(c, token)
	return token
}

func validateCSRF(c *gin.Context) bool {
	cookieToken, err := c.Cookie("csrf_token")
	if err != nil || cookieToken == "" {
		return false
	}
	formToken := c.PostForm("csrf_token")
	if formToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(formToken)) == 1
}

func useSecureCookie(c *gin.Context) bool {
	switch strings.ToLower(os.Getenv("COOKIE_SECURE")) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}

	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}

func LoginPage(c *gin.Context) {
	tok := getCSRFToken(c)
	renderers["login.html"].Render(c, "login.html", mergeMaps(pageData(), gin.H{
		"CSRFToken": tok,
	}))
}

func LoginPost(c *gin.Context) {
	if !validateCSRF(c) {
		tok := getCSRFToken(c)
		renderers["login.html"].Render(c, "login.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Error":     "Invalid or missing CSRF token. Please try again.",
			"Username":  c.PostForm("username"),
		}))
		return
	}

	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	admin, err := services.AuthenticateAdmin(username, password)
	if err != nil {
		services.RecordFailedLoginAttempt(c.ClientIP(), username)
		tok := getCSRFToken(c)
		renderers["login.html"].Render(c, "login.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Error":     "Invalid username or password",
			"Username":  username,
		}))
		return
	}

	services.ResetLoginAttempts(c.ClientIP(), username)

	token, err := services.GenerateAdminToken(admin)
	if err != nil {
		tok := getCSRFToken(c)
		renderers["login.html"].Render(c, "login.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Error":     "Internal error. Please try again.",
			"Username":  username,
		}))
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

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "admin_token",
		Value:    token,
		MaxAge:   7 * 24 * 3600,
		Path:     "/",
		Secure:   useSecureCookie(c),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	c.Redirect(http.StatusFound, "/admin/dashboard")
}

func ForgotPasswordPage(c *gin.Context) {
	tok := getCSRFToken(c)
	renderers["login.html"].Render(c, "login.html", mergeMaps(pageData(), gin.H{
		"CSRFToken": tok,
		"ShowReset": true,
	}))
}

func ForgotPasswordPost(c *gin.Context) {
	if !validateCSRF(c) {
		tok := getCSRFToken(c)
		renderers["login.html"].Render(c, "login.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Error":     "Invalid or missing CSRF token. Please try again.",
			"ShowReset": true,
		}))
		return
	}

	email := strings.TrimSpace(c.PostForm("email"))
	admin, err := services.GetAdminByEmail(email)
	if err == nil && admin != nil {
		_, _ = services.CreatePasswordResetToken(admin.ID)
	}

	tok := getCSRFToken(c)
	renderers["login.html"].Render(c, "login.html", mergeMaps(pageData(), gin.H{
		"CSRFToken": tok,
		"ShowReset": true,
		"ResetSent": true,
	}))
}

func LogoutPost(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   useSecureCookie(c),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	c.Redirect(http.StatusFound, "/admin/login")
}

func ResetPasswordPage(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		tok := getCSRFToken(c)
		renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Error":     "Missing reset token. Please request a new password reset.",
		}))
		return
	}

	tok := getCSRFToken(c)
	renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
		"CSRFToken": tok,
		"Token":     token,
	}))
}

func ResetPasswordPost(c *gin.Context) {
	if !validateCSRF(c) {
		tok := getCSRFToken(c)
		renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Token":     c.PostForm("token"),
			"Error":     "Invalid or missing CSRF token. Please try again.",
		}))
		return
	}

	token := c.PostForm("token")
	password := c.PostForm("password")
	confirm := c.PostForm("confirm_password")

	if token == "" {
		tok := getCSRFToken(c)
		renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Error":     "Missing reset token. Please request a new password reset.",
		}))
		return
	}

	if len(password) < 8 {
		tok := getCSRFToken(c)
		renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Token":     token,
			"Error":     "Password must be at least 8 characters.",
		}))
		return
	}

	if password != confirm {
		tok := getCSRFToken(c)
		renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Token":     token,
			"Error":     "Passwords do not match.",
		}))
		return
	}

	adminID, err := services.ValidatePasswordResetToken(token)
	if err != nil {
		tok := getCSRFToken(c)
		renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Error":     "Invalid or expired reset token. Please request a new password reset.",
		}))
		return
	}

	if err := services.UpdateAdminPassword(adminID, password); err != nil {
		tok := getCSRFToken(c)
		renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
			"CSRFToken": tok,
			"Token":     token,
			"Error":     "Failed to update password. Please try again.",
		}))
		return
	}

	services.MarkResetTokenUsed(token)

	tok := getCSRFToken(c)
	renderers["reset_password.html"].Render(c, "reset_password.html", mergeMaps(pageData(), gin.H{
		"CSRFToken": tok,
		"Success":   "Your password has been reset successfully.",
	}))
}
