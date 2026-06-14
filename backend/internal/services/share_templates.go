package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

// ErrMessageTemplateNotFound is returned when a requested message template
// does not exist in the database. Callers can use errors.Is to detect it.
var ErrMessageTemplateNotFound = errors.New("message template not found")

// ListMessageTemplates returns all message templates ordered by name.
func ListMessageTemplates() ([]models.MessageTemplate, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT m.id, m.name, m.body, m.is_default, m.created_by, m.created_at, m.updated_at, a.username AS created_by_name
		FROM message_templates m
		LEFT JOIN admins a ON m.created_by = a.id
		ORDER BY m.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list message templates: %w", err)
	}
	defer rows.Close()

	var templates []models.MessageTemplate
	for rows.Next() {
		var t models.MessageTemplate
		err := rows.Scan(&t.ID, &t.Name, &t.Body, &t.IsDefault, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.CreatedByName)
		if err != nil {
			return nil, fmt.Errorf("scan message template: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message templates: %w", err)
	}

	return templates, nil
}

// GetMessageTemplateByID returns a single message template by ID.
// Returns ErrMessageTemplateNotFound if not found.
func GetMessageTemplateByID(id uuid.UUID) (*models.MessageTemplate, error) {
	var t models.MessageTemplate
	err := db.Pool.QueryRow(context.Background(), `
		SELECT m.id, m.name, m.body, m.is_default, m.created_by, m.created_at, m.updated_at, a.username AS created_by_name
		FROM message_templates m
		LEFT JOIN admins a ON m.created_by = a.id
		WHERE m.id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Body, &t.IsDefault, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.CreatedByName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageTemplateNotFound
		}
		return nil, fmt.Errorf("get message template: %w", err)
	}
	return &t, nil
}

// CreateMessageTemplate creates a new message template after validating that
// name and body are non-empty. Returns the created template.
func CreateMessageTemplate(name, body string, createdBy uuid.UUID) (*models.MessageTemplate, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("template body is required")
	}

	t := &models.MessageTemplate{
		ID:        uuid.New(),
		Name:      name,
		Body:      body,
		CreatedBy: &createdBy,
	}

	err := db.Pool.QueryRow(context.Background(), `
		INSERT INTO message_templates (id, name, body, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING is_default, created_at, updated_at
	`, t.ID, t.Name, t.Body, t.CreatedBy).Scan(&t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create message template: %w", err)
	}

	return t, nil
}

// UpdateMessageTemplate updates the name and body of an existing template.
// Validates that name and body are non-empty. Returns ErrMessageTemplateNotFound
// if the template does not exist.
func UpdateMessageTemplate(id uuid.UUID, name, body string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("template body is required")
	}

	result, err := db.Pool.Exec(context.Background(), `
		UPDATE message_templates
		SET name = $1, body = $2, updated_at = NOW()
		WHERE id = $3
	`, name, body, id)
	if err != nil {
		return fmt.Errorf("update message template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMessageTemplateNotFound
	}

	return nil
}

// DeleteMessageTemplate deletes a message template. If the deleted template was
// the default, the oldest remaining template is promoted to default. Returns an
// error if this would leave zero templates.
func DeleteMessageTemplate(id uuid.UUID) error {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Check how many templates exist and whether this one is the default.
	var count int
	var wasDefault bool
	err = tx.QueryRow(context.Background(), `
		SELECT
			(SELECT COUNT(*) FROM message_templates),
			COALESCE((SELECT is_default FROM message_templates WHERE id = $1), false)
	`, id).Scan(&count, &wasDefault)
	if err != nil {
		return fmt.Errorf("check template count: %w", err)
	}

	if count <= 1 {
		return fmt.Errorf("cannot delete the last message template")
	}

	// Delete the template.
	result, err := tx.Exec(context.Background(), `
		DELETE FROM message_templates WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete message template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMessageTemplateNotFound
	}

	// If it was the default, promote the oldest remaining template.
	if wasDefault {
		_, err = tx.Exec(context.Background(), `
			UPDATE message_templates SET is_default = true
			WHERE id = (
				SELECT id FROM message_templates
				ORDER BY created_at ASC
				LIMIT 1
			)
		`)
		if err != nil {
			return fmt.Errorf("promote default template: %w", err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetDefaultMessageTemplate returns the default message template.
// Returns an error if no default is set.
func GetDefaultMessageTemplate() (*models.MessageTemplate, error) {
	var t models.MessageTemplate
	err := db.Pool.QueryRow(context.Background(), `
		SELECT m.id, m.name, m.body, m.is_default, m.created_by, m.created_at, m.updated_at, a.username AS created_by_name
		FROM message_templates m
		LEFT JOIN admins a ON m.created_by = a.id
		WHERE m.is_default = true
		LIMIT 1
	`).Scan(
		&t.ID, &t.Name, &t.Body, &t.IsDefault, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.CreatedByName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no default message template found")
		}
		return nil, fmt.Errorf("get default message template: %w", err)
	}
	return &t, nil
}

// SetDefaultMessageTemplate sets one template as default and unsets all others.
func SetDefaultMessageTemplate(id uuid.UUID) error {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Verify the template exists.
	var exists bool
	err = tx.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM message_templates WHERE id = $1)
	`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check template exists: %w", err)
	}
	if !exists {
		return ErrMessageTemplateNotFound
	}

	// Unset all defaults.
	_, err = tx.Exec(context.Background(), `
		UPDATE message_templates SET is_default = false
	`)
	if err != nil {
		return fmt.Errorf("unset defaults: %w", err)
	}

	// Set the new default.
	_, err = tx.Exec(context.Background(), `
		UPDATE message_templates SET is_default = true WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("set default: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// RenderShareMessage replaces template placeholders with values derived from
// the waffle and stats map. Supported placeholders:
//
//	{item}         — waffle title
//	{price}        — spot price in integer dollars
//	{total_spots}  — total spots from stats
//	{spots_left}   — total - paid - pending
//	{spots_claimed} — paid + pending
//	{url}          — https://{host}/waffle/{slug}
//
// TODO: Handlers still call this with the old (string, error) signature;
// update them in the follow-up handler fix task.
func RenderShareMessage(templateBody string, waffle *models.Waffle, stats map[string]interface{}, host string) string {
	totalSpots, _ := toInt(stats["total_spots"])
	paid, _ := toInt(stats["paid"])
	pending, _ := toInt(stats["pending"])

	spotsLeft := totalSpots - paid - pending
	spotsClaimed := paid + pending

	result := templateBody
	result = strings.ReplaceAll(result, "{item}", waffle.Title)
	result = strings.ReplaceAll(result, "{price}", fmt.Sprintf("%d", waffle.SpotPrice))
	result = strings.ReplaceAll(result, "{total_spots}", fmt.Sprintf("%d", totalSpots))
	result = strings.ReplaceAll(result, "{spots_left}", fmt.Sprintf("%d", spotsLeft))
	result = strings.ReplaceAll(result, "{spots_claimed}", fmt.Sprintf("%d", spotsClaimed))
	result = strings.ReplaceAll(result, "{url}", fmt.Sprintf("https://%s/waffle/%s", host, waffle.Slug))

	return result
}

// toInt extracts an int from interface{} (float64 from JSON or int from Go map).
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
