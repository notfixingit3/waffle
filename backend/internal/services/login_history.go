package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func parseUserAgent(ua string) (browser, os, deviceType string) {
	if ua == "" {
		return "Unknown", "Unknown", "unknown"
	}

	uaLower := strings.ToLower(ua)

	// Detect browser — check in priority order
	switch {
	case strings.Contains(uaLower, "instagram"):
		browser = "Instagram"
	case strings.Contains(uaLower, "edg/"):
		browser = "Edge"
	case strings.Contains(uaLower, "firefox/"):
		browser = "Firefox"
	case strings.Contains(uaLower, "chrome/") && !strings.Contains(uaLower, "edg/"):
		browser = "Chrome"
	case strings.Contains(uaLower, "safari/") && !strings.Contains(uaLower, "chrome"):
		browser = "Safari"
	default:
		browser = "Unknown"
	}

	// Detect OS
	switch {
	case strings.Contains(ua, "iPad"):
		os = "iPadOS"
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPod"):
		os = "iOS"
	case strings.Contains(uaLower, "android"):
		os = "Android"
	case strings.Contains(ua, "Windows") || strings.Contains(ua, "Win"):
		os = "Windows"
	case strings.Contains(ua, "Mac OS") || strings.Contains(ua, "Macintosh") || strings.Contains(ua, "macOS"):
		os = "macOS"
	case strings.Contains(uaLower, "linux") || strings.Contains(uaLower, "x11"):
		os = "Linux"
	default:
		os = "Unknown"
	}

	// Detect device type
	switch {
	case strings.Contains(ua, "iPad") || strings.Contains(uaLower, "tablet"):
		deviceType = "tablet"
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPod") ||
		strings.Contains(uaLower, "android") && strings.Contains(uaLower, "mobile") ||
		strings.Contains(uaLower, "mobi"):
		deviceType = "mobile"
	default:
		deviceType = "desktop"
	}

	return
}

func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return true
	}
	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() {
		return true
	}
	return false
}

func RecordLogin(adminIDStr, ipAddress, userAgent string) (uuid.UUID, error) {
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse admin ID: %w", err)
	}

	browser, osName, deviceType := parseUserAgent(userAgent)

	id := uuid.New()
	now := time.Now()

	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO login_history (id, admin_id, ip_address, user_agent, browser, os, device_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, adminID, ipAddress, userAgent, browser, osName, deviceType, now)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("insert login history: %w", err)
	}

	_, err = db.Pool.Exec(context.Background(),
		`UPDATE admins SET last_login_at = $1 WHERE id = $2`, now, adminID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("update last login: %w", err)
	}

	// #nosec G104 — non-critical last_login_ip update; failure is acceptable per v0.1.7 design
	_, _ = db.Pool.Exec(context.Background(),
		`UPDATE admins SET last_login_ip = $1 WHERE id = $2`, ipAddress, adminID)

	return id, nil
}

func GetLoginHistory(adminID uuid.UUID, page, limit int) ([]models.LoginHistory, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	var total int
	err := db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM login_history WHERE admin_id = $1`, adminID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count login history: %w", err)
	}

	offset := (page - 1) * limit

	rows, err := db.Pool.Query(context.Background(), `
		SELECT id, admin_id, ip_address, user_agent, browser, os, device_type,
		       ip_org, ip_country, ip_city, ip_asn, whois_server, created_at
		FROM login_history
		WHERE admin_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, adminID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query login history: %w", err)
	}
	defer rows.Close()

	var records []models.LoginHistory
	for rows.Next() {
		var lh models.LoginHistory
		if err := rows.Scan(
			&lh.ID, &lh.AdminID, &lh.IPAddress, &lh.UserAgent, &lh.Browser, &lh.OS, &lh.DeviceType,
			&lh.IPOrg, &lh.IPCountry, &lh.IPCity, &lh.IPASN, &lh.WhoisServer, &lh.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan login history: %w", err)
		}
		records = append(records, lh)
	}

	return records, total, nil
}

func GetAllLoginHistory(viewerRole string, viewerID uuid.UUID, page, limit int) ([]models.LoginHistory, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit

	var countQuery, dataQuery string
	var countArgs, dataArgs []interface{}

	switch viewerRole {
	case models.RoleSuperAdmin:
		countQuery = `SELECT COUNT(*) FROM login_history`
		countArgs = nil
		dataQuery = `
			SELECT id, admin_id, ip_address, user_agent, browser, os, device_type,
			       ip_org, ip_country, ip_city, ip_asn, whois_server, created_at
			FROM login_history
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`
		dataArgs = []interface{}{limit, offset}

	case models.RoleAdmin:
		countQuery = `
			SELECT COUNT(*)
			FROM login_history lh
			LEFT JOIN admins a ON a.id = lh.admin_id
			WHERE lh.admin_id = $1 OR a.role = $2`
		countArgs = []interface{}{viewerID, models.RoleWaffleManager}
		dataQuery = `
			SELECT lh.id, lh.admin_id, lh.ip_address, lh.user_agent, lh.browser, lh.os, lh.device_type,
			       lh.ip_org, lh.ip_country, lh.ip_city, lh.ip_asn, lh.whois_server, lh.created_at
			FROM login_history lh
			LEFT JOIN admins a ON a.id = lh.admin_id
			WHERE lh.admin_id = $1 OR a.role = $2
			ORDER BY lh.created_at DESC
			LIMIT $3 OFFSET $4`
		dataArgs = []interface{}{viewerID, models.RoleWaffleManager, limit, offset}

	case models.RoleWaffleManager:
		countQuery = `SELECT COUNT(*) FROM login_history WHERE admin_id = $1`
		countArgs = []interface{}{viewerID}
		dataQuery = `
			SELECT id, admin_id, ip_address, user_agent, browser, os, device_type,
			       ip_org, ip_country, ip_city, ip_asn, whois_server, created_at
			FROM login_history
			WHERE admin_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`
		dataArgs = []interface{}{viewerID, limit, offset}

	default:
		return nil, 0, fmt.Errorf("unknown role: %s", viewerRole)
	}

	var total int
	err := db.Pool.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count login history: %w", err)
	}

	rows, err := db.Pool.Query(context.Background(), dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query login history: %w", err)
	}
	defer rows.Close()

	var records []models.LoginHistory
	for rows.Next() {
		var lh models.LoginHistory
		if err := rows.Scan(
			&lh.ID, &lh.AdminID, &lh.IPAddress, &lh.UserAgent, &lh.Browser, &lh.OS, &lh.DeviceType,
			&lh.IPOrg, &lh.IPCountry, &lh.IPCity, &lh.IPASN, &lh.WhoisServer, &lh.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan login history: %w", err)
		}
		records = append(records, lh)
	}

	return records, total, nil
}

func getLoginRecord(loginID uuid.UUID) (*models.LoginHistory, error) {
	lh := &models.LoginHistory{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, admin_id, ip_address, user_agent, browser, os, device_type,
		       ip_org, ip_country, ip_city, ip_asn, whois_server, created_at
		FROM login_history WHERE id = $1
	`, loginID).Scan(
		&lh.ID, &lh.AdminID, &lh.IPAddress, &lh.UserAgent, &lh.Browser, &lh.OS, &lh.DeviceType,
		&lh.IPOrg, &lh.IPCountry, &lh.IPCity, &lh.IPASN, &lh.WhoisServer, &lh.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get login record: %w", err)
	}
	return lh, nil
}

func EnrichLoginWithWHOIS(loginID uuid.UUID) error {
	lh, err := getLoginRecord(loginID)
	if err != nil {
		return err
	}

	// Skip private/RFC1918 IPs
	if isPrivateIP(lh.IPAddress) {
		return nil
	}

	// Get WHOIS server from system_settings
	var whoisServer string
	err = db.Pool.QueryRow(context.Background(),
		`SELECT value FROM system_settings WHERE key = 'whois_server'`).Scan(&whoisServer)
	if err != nil {
		// No whois_server configured — fall back to default
		whoisServer = "whois.pwhois.org"
	}

	// Query the WHOIS server
	response, err := queryWHOIS(whoisServer, lh.IPAddress)
	if err != nil {
		return fmt.Errorf("whois query: %w", err)
	}

	org, country, city, asn := parseWHOISResponse(response)

	_, err = db.Pool.Exec(context.Background(), `
		UPDATE login_history SET ip_org = $1, ip_country = $2, ip_city = $3, ip_asn = $4, whois_server = $5
		WHERE id = $6
	`, org, country, city, asn, whoisServer, loginID)
	if err != nil {
		return fmt.Errorf("update login history with whois: %w", err)
	}

	return nil
}

func queryWHOIS(server, ip string) (string, error) {
	conn, err := net.DialTimeout("tcp", server+":43", 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("connect to whois: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second)) // #nosec G104

	if _, err := fmt.Fprintf(conn, "%s\r\n", ip); err != nil {
		return "", fmt.Errorf("write whois query: %w", err)
	}

	respBytes, err := io.ReadAll(conn)
	if err != nil {
		return "", fmt.Errorf("read whois response: %w", err)
	}

	return string(respBytes), nil
}

func parseWHOISResponse(response string) (org, country, city, asn string) {
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)

		switch {
		case strings.HasPrefix(lower, "originas:") || strings.HasPrefix(lower, "origin:"):
			asn = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(lower, "orgname:") || strings.HasPrefix(lower, "descr:") ||
			strings.HasPrefix(lower, "org-name:") || strings.HasPrefix(lower, "owner:"):
			if org == "" {
				org = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			}
		case strings.HasPrefix(lower, "country:") || strings.HasPrefix(lower, "countrycode:"):
			if country == "" {
				country = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			}
		case strings.HasPrefix(lower, "city:") || strings.HasPrefix(lower, "address:"):
			if city == "" {
				city = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			}
		}
	}

	return
}
