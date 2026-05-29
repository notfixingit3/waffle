package services

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/syrup/backend/internal/db"
)

const (
	defaultRetentionDays = 90
)

// PurgeOldEntries reads retention settings and deletes old records from
// audit_log and login_history. Missing or invalid settings default to 90 days.
// Called once during startup — no background goroutine or ticker.
func PurgeOldEntries() {
	purgeTable := func(table, settingKey string) {
		days := getRetentionDays(settingKey)

		tag, err := db.Pool.Exec(context.Background(),
			`DELETE FROM `+table+` WHERE created_at < NOW() - ($1 || ' days')::INTERVAL`,
			strconv.Itoa(days),
		)
		if err != nil {
			slog.Warn("Retention purge failed", "table", table, "days", days, "error", err)
			return
		}

		if tag.RowsAffected() > 0 {
			slog.Info("Purged old entries", "table", table, "count", tag.RowsAffected(), "retention_days", days)
		}
	}

	purgeTable("audit_log", "audit_retention_days")
	purgeTable("login_history", "login_history_retention_days")
}

func getRetentionDays(key string) int {
	val, err := GetSetting(key)
	if err != nil || val == "" {
		return defaultRetentionDays
	}

	days, err := strconv.Atoi(val)
	if err != nil || days < 1 || days > 365 {
		return defaultRetentionDays
	}

	return days
}
