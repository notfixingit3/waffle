package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/syrup/backend/internal/db"
)

// allowedSettings is the whitelist of keys that SetSetting accepts.
var allowedSettings = map[string]bool{
	"whois_server":                   true,
	"jwt_expiration_hours":           true,
	"password_min_length":            true,
	"audit_retention_days":           true,
	"login_history_retention_days":   true,
}

// GetSetting retrieves the value for the given key from system_settings.
// Returns empty string if the key does not exist.
func GetSetting(key string) (string, error) {
	var value string
	err := db.Pool.QueryRow(context.Background(), `
		SELECT value FROM system_settings WHERE key = $1
	`, key).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

// validateHostname checks that value looks like a valid hostname: non-empty,
// at least one dot, no spaces.
func validateHostname(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	if strings.Contains(value, " ") {
		return fmt.Errorf("hostname cannot contain spaces")
	}
	if !strings.Contains(value, ".") {
		return fmt.Errorf("hostname must contain at least one dot")
	}
	return nil
}

func validatePositiveInteger(value string) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("value must be an integer")
	}
	if n <= 0 {
		return fmt.Errorf("value must be greater than zero")
	}
	return nil
}

func validateRetentionDays(value string) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("value must be an integer")
	}
	if n < 1 || n > 365 {
		return fmt.Errorf("value must be between 1 and 365 days")
	}
	return nil
}

func validateSettingValue(key, value string) error {
	switch key {
	case "whois_server":
		return validateHostname(value)
	case "jwt_expiration_hours", "password_min_length":
		return validatePositiveInteger(value)
	case "audit_retention_days", "login_history_retention_days":
		return validateRetentionDays(value)
	default:
		return fmt.Errorf("invalid setting key: %q", key)
	}
}

// SetSetting upserts a system setting after validating the key is in the
// allowed whitelist and the value is a valid hostname.
func SetSetting(key, value string, updatedBy uuid.UUID) error {
	if !allowedSettings[key] {
		return fmt.Errorf("invalid setting key: %q", key)
	}

	if err := validateSettingValue(key, value); err != nil {
		return fmt.Errorf("invalid value for %q: %w", key, err)
	}

	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO system_settings (key, value, updated_by, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
	`, key, value, updatedBy, time.Now())
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}

	return nil
}
