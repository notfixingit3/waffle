package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

type AuditLogFilters struct {
	AdminID    *uuid.UUID
	Action     string
	TargetType string
	From       *time.Time
	To         *time.Time
	Page       int
	Limit      int
}

func RecordAudit(adminID uuid.UUID, action, targetType, targetID, details, ipAddress string) error {
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO audit_log (admin_id, action, target_type, target_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, adminID, action, targetType, targetID, details, ipAddress)
	if err != nil {
		return fmt.Errorf("record audit: %w", err)
	}
	return nil
}

func QueryAudit(filters AuditLogFilters) ([]models.AuditLog, int, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 {
		filters.Limit = 20
	}

	where, args := auditWhere(filters)
	countQuery := `SELECT COUNT(*) FROM audit_log` + where

	var total int
	if err := db.Pool.QueryRow(context.Background(), countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit log: %w", err)
	}

	offset := (filters.Page - 1) * filters.Limit
	dataArgs := append(args, filters.Limit, offset)
	limitParam := len(args) + 1
	offsetParam := len(args) + 2
	dataQuery := fmt.Sprintf(`
		SELECT id, admin_id, action, target_type, target_id, details, ip_address, created_at
		FROM audit_log%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, limitParam, offsetParam)

	rows, err := db.Pool.Query(context.Background(), dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	entries := []models.AuditLog{}
	for rows.Next() {
		var entry models.AuditLog
		if err := rows.Scan(
			&entry.ID, &entry.AdminID, &entry.Action, &entry.TargetType,
			&entry.TargetID, &entry.Details, &entry.IPAddress, &entry.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, total, nil
}

func GetAuditByID(id uuid.UUID) (*models.AuditLog, error) {
	var entry models.AuditLog
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, admin_id, action, target_type, target_id, details, ip_address, created_at
		FROM audit_log
		WHERE id = $1
	`, id).Scan(
		&entry.ID, &entry.AdminID, &entry.Action, &entry.TargetType,
		&entry.TargetID, &entry.Details, &entry.IPAddress, &entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	return &entry, nil
}

func auditWhere(filters AuditLogFilters) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}

	add := func(clause string, value interface{}) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}

	if filters.AdminID != nil {
		add("admin_id = $%d", *filters.AdminID)
	}
	if strings.TrimSpace(filters.Action) != "" {
		add("action = $%d", strings.TrimSpace(filters.Action))
	}
	if strings.TrimSpace(filters.TargetType) != "" {
		add("target_type = $%d", strings.TrimSpace(filters.TargetType))
	}
	if filters.From != nil {
		add("created_at >= $%d", *filters.From)
	}
	if filters.To != nil {
		add("created_at <= $%d", *filters.To)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
