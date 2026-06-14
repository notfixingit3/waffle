package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

// testTemplatePrefix is used to identify and clean up test templates.
const testTemplatePrefix = "test-template-"

// createTestTemplateAdmin inserts a temporary admin for template tests and
// returns its ID. Tests using this helper must call cleanupTestTemplateAdmin.
func createTestTemplateAdmin(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	username := "test-tpl-" + id.String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, username, username+"@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test template admin: %v", err)
	}
	return id
}

func cleanupTestTemplateAdmin(t *testing.T, id uuid.UUID) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, id)
	if err != nil {
		t.Fatalf("cleanup test template admin: %v", err)
	}
}

func cleanupTestTemplates(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE name LIKE $1 || '%'`, testTemplatePrefix)
	if err != nil {
		t.Fatalf("cleanup test templates: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListMessageTemplates tests
// ---------------------------------------------------------------------------

func TestListMessageTemplates_ReturnsAllOrderedByName(t *testing.T) {
	defer cleanupTestTemplates(t)
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	t1, err := CreateMessageTemplate(testTemplatePrefix+"B", "body B", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate B: %v", err)
	}
	t2, err := CreateMessageTemplate(testTemplatePrefix+"A", "body A", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate A: %v", err)
	}

	templates, err := ListMessageTemplates()
	if err != nil {
		t.Fatalf("ListMessageTemplates: %v", err)
	}

	// Find our test templates in the results.
	var found []models.MessageTemplate
	for _, tmpl := range templates {
		if strings.HasPrefix(tmpl.Name, testTemplatePrefix) {
			found = append(found, tmpl)
		}
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 test templates, got %d", len(found))
	}
	// Should be ordered A then B.
	if found[0].Name != testTemplatePrefix+"A" {
		t.Errorf("expected first template name %q, got %q", testTemplatePrefix+"A", found[0].Name)
	}
	if found[1].Name != testTemplatePrefix+"B" {
		t.Errorf("expected second template name %q, got %q", testTemplatePrefix+"B", found[1].Name)
	}

	// Clean up created templates.
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, t1.ID)
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, t2.ID)
}

// ---------------------------------------------------------------------------
// GetMessageTemplateByID tests
// ---------------------------------------------------------------------------

func TestGetMessageTemplateByID_ReturnsTemplate(t *testing.T) {
	defer cleanupTestTemplates(t)
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	created, err := CreateMessageTemplate(testTemplatePrefix+"get-by-id", "test body", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate: %v", err)
	}

	got, err := GetMessageTemplateByID(created.ID)
	if err != nil {
		t.Fatalf("GetMessageTemplateByID: %v", err)
	}
	if got.Name != created.Name {
		t.Errorf("expected name %q, got %q", created.Name, got.Name)
	}
	if got.Body != "test body" {
		t.Errorf("expected body %q, got %q", "test body", got.Body)
	}

	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, created.ID)
}

func TestGetMessageTemplateByID_NotFound(t *testing.T) {
	_, err := GetMessageTemplateByID(uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}
}

// ---------------------------------------------------------------------------
// CreateMessageTemplate tests
// ---------------------------------------------------------------------------

func TestCreateMessageTemplate_Success(t *testing.T) {
	defer cleanupTestTemplates(t)
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	tmpl, err := CreateMessageTemplate(testTemplatePrefix+"create", "Hello {item}!", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate: %v", err)
	}
	if tmpl.Name != testTemplatePrefix+"create" {
		t.Errorf("expected name %q, got %q", testTemplatePrefix+"create", tmpl.Name)
	}
	if tmpl.Body != "Hello {item}!" {
		t.Errorf("expected body %q, got %q", "Hello {item}!", tmpl.Body)
	}
	if tmpl.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if tmpl.CreatedBy == nil || *tmpl.CreatedBy != adminID {
		t.Errorf("expected created_by %s, got %v", adminID, tmpl.CreatedBy)
	}

	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, tmpl.ID)
}

func TestCreateMessageTemplate_RejectsEmptyName(t *testing.T) {
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	_, err := CreateMessageTemplate("", "body", adminID)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestCreateMessageTemplate_RejectsEmptyBody(t *testing.T) {
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	_, err := CreateMessageTemplate("name", "", adminID)
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

// ---------------------------------------------------------------------------
// UpdateMessageTemplate tests
// ---------------------------------------------------------------------------

func TestUpdateMessageTemplate_Success(t *testing.T) {
	defer cleanupTestTemplates(t)
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	created, err := CreateMessageTemplate(testTemplatePrefix+"update", "original", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate: %v", err)
	}

	err = UpdateMessageTemplate(created.ID, testTemplatePrefix+"updated", "new body")
	if err != nil {
		t.Fatalf("UpdateMessageTemplate: %v", err)
	}

	got, err := GetMessageTemplateByID(created.ID)
	if err != nil {
		t.Fatalf("GetMessageTemplateByID after update: %v", err)
	}
	if got.Name != testTemplatePrefix+"updated" {
		t.Errorf("expected name %q, got %q", testTemplatePrefix+"updated", got.Name)
	}
	if got.Body != "new body" {
		t.Errorf("expected body %q, got %q", "new body", got.Body)
	}

	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, created.ID)
}

func TestUpdateMessageTemplate_NotFound(t *testing.T) {
	err := UpdateMessageTemplate(uuid.New(), "name", "body")
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}
}

func TestUpdateMessageTemplate_RejectsEmptyName(t *testing.T) {
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	created, err := CreateMessageTemplate(testTemplatePrefix+"update-empty-name", "body", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, created.ID)
	}()

	err = UpdateMessageTemplate(created.ID, "", "body")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestUpdateMessageTemplate_RejectsEmptyBody(t *testing.T) {
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	created, err := CreateMessageTemplate(testTemplatePrefix+"update-empty-body", "body", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, created.ID)
	}()

	err = UpdateMessageTemplate(created.ID, "name", "")
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

// ---------------------------------------------------------------------------
// DeleteMessageTemplate tests
// ---------------------------------------------------------------------------

func TestDeleteMessageTemplate_Success(t *testing.T) {
	defer cleanupTestTemplates(t)
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	t1, err := CreateMessageTemplate(testTemplatePrefix+"del-1", "body 1", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 1: %v", err)
	}
	t2, err := CreateMessageTemplate(testTemplatePrefix+"del-2", "body 2", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 2: %v", err)
	}

	err = DeleteMessageTemplate(t1.ID)
	if err != nil {
		t.Fatalf("DeleteMessageTemplate: %v", err)
	}

	// Verify it's gone.
	_, err = GetMessageTemplateByID(t1.ID)
	if err == nil {
		t.Error("expected error for deleted template, got nil")
	}

	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, t2.ID)
}

func TestDeleteMessageTemplate_NotFound(t *testing.T) {
	err := DeleteMessageTemplate(uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}
}

func TestDeleteMessageTemplate_RejectsLastTemplate(t *testing.T) {
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	// Start from a known clean state: nullify FK references, then delete all
	// templates so leftover rows from other tests don't inflate the count.
	_, _ = db.Pool.Exec(context.Background(), `UPDATE waffles SET share_template_id = NULL WHERE share_template_id IS NOT NULL`)
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates`)

	// Create three test templates, delete two, then verify the last one can't be deleted.
	t1, err := CreateMessageTemplate(testTemplatePrefix+"last-1", "body 1", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 1: %v", err)
	}
	t2, err := CreateMessageTemplate(testTemplatePrefix+"last-2", "body 2", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 2: %v", err)
	}
	t3, err := CreateMessageTemplate(testTemplatePrefix+"last-3", "body 3", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 3: %v", err)
	}

	err = DeleteMessageTemplate(t1.ID)
	if err != nil {
		t.Fatalf("DeleteMessageTemplate t1: %v", err)
	}
	err = DeleteMessageTemplate(t2.ID)
	if err != nil {
		t.Fatalf("DeleteMessageTemplate t2: %v", err)
	}

	err = DeleteMessageTemplate(t3.ID)
	if err == nil {
		t.Fatal("expected error when deleting the last template, got nil")
	}

	// Manually delete the remaining template so admin FK cleanup works.
	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, t3.ID)

	// Re-seed the default template for other tests that depend on it.
	_, _ = db.Pool.Exec(context.Background(), `
		INSERT INTO message_templates (name, body, is_default, created_at, updated_at)
		VALUES ('Default Hype Drop', E'🧇 NEW WAFFLE DROP 🧇\n\n{item}\n\n${price}/spot • {spots_left} of {total_spots} left\n\nClaim your spot 👇\n{url}', true, NOW(), NOW())
	`)
}

func TestDeleteMessageTemplate_PromotesOldestOnDefaultDelete(t *testing.T) {
	defer cleanupTestTemplates(t)
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	// Create two templates.
	t1, err := CreateMessageTemplate(testTemplatePrefix+"promote-1", "body 1", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 1: %v", err)
	}
	t2, err := CreateMessageTemplate(testTemplatePrefix+"promote-2", "body 2", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 2: %v", err)
	}

	// Set t1 as default.
	err = SetDefaultMessageTemplate(t1.ID)
	if err != nil {
		t.Fatalf("SetDefaultMessageTemplate: %v", err)
	}

	// Delete t1 (the default) — the oldest remaining template should become default.
	err = DeleteMessageTemplate(t1.ID)
	if err != nil {
		t.Fatalf("DeleteMessageTemplate: %v", err)
	}

	defaultTmpl, err := GetDefaultMessageTemplate()
	if err != nil {
		t.Fatalf("GetDefaultMessageTemplate: %v", err)
	}
	if defaultTmpl.ID == t1.ID {
		t.Error("expected default to NOT be the deleted template")
	}
	if !defaultTmpl.IsDefault {
		t.Error("expected default template to have is_default = true")
	}

	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, t2.ID)
}

// ---------------------------------------------------------------------------
// GetDefaultMessageTemplate tests
// ---------------------------------------------------------------------------

func TestGetDefaultMessageTemplate_ReturnsDefault(t *testing.T) {
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	tmpl, err := CreateMessageTemplate(testTemplatePrefix+"default-test", "body", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, tmpl.ID)
	}()

	err = SetDefaultMessageTemplate(tmpl.ID)
	if err != nil {
		t.Fatalf("SetDefaultMessageTemplate: %v", err)
	}

	got, err := GetDefaultMessageTemplate()
	if err != nil {
		t.Fatalf("GetDefaultMessageTemplate: %v", err)
	}
	if got.ID != tmpl.ID {
		t.Errorf("expected default template ID %s, got %s", tmpl.ID, got.ID)
	}
	if !got.IsDefault {
		t.Error("expected template to be marked as default")
	}
}

// ---------------------------------------------------------------------------
// SetDefaultMessageTemplate tests
// ---------------------------------------------------------------------------

func TestSetDefaultMessageTemplate_SetsOneDefault(t *testing.T) {
	defer cleanupTestTemplates(t)
	adminID := createTestTemplateAdmin(t)
	defer cleanupTestTemplateAdmin(t, adminID)

	t1, err := CreateMessageTemplate(testTemplatePrefix+"setdef-1", "body 1", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 1: %v", err)
	}
	t2, err := CreateMessageTemplate(testTemplatePrefix+"setdef-2", "body 2", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate 2: %v", err)
	}

	err = SetDefaultMessageTemplate(t1.ID)
	if err != nil {
		t.Fatalf("SetDefaultMessageTemplate t1: %v", err)
	}

	got1, _ := GetMessageTemplateByID(t1.ID)
	if !got1.IsDefault {
		t.Error("expected t1 to be default")
	}
	got2, _ := GetMessageTemplateByID(t2.ID)
	if got2.IsDefault {
		t.Error("expected t2 to NOT be default")
	}

	// Switch default to t2.
	err = SetDefaultMessageTemplate(t2.ID)
	if err != nil {
		t.Fatalf("SetDefaultMessageTemplate t2: %v", err)
	}

	got1, _ = GetMessageTemplateByID(t1.ID)
	if got1.IsDefault {
		t.Error("expected t1 to no longer be default after switch")
	}
	got2, _ = GetMessageTemplateByID(t2.ID)
	if !got2.IsDefault {
		t.Error("expected t2 to be default after switch")
	}

	_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id IN ($1, $2)`, t1.ID, t2.ID)
}

func TestSetDefaultMessageTemplate_NotFound(t *testing.T) {
	err := SetDefaultMessageTemplate(uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}
}

// ---------------------------------------------------------------------------
// RenderShareMessage tests
// ---------------------------------------------------------------------------

func TestRenderShareMessage_AllVariables(t *testing.T) {
	waffle := &models.Waffle{
		Title:     "Test Drop",
		Slug:      "test-drop-abc123",
		SpotPrice: 25,
	}
	stats := map[string]interface{}{
		"total_spots": 100,
		"paid":        45,
		"pending":     5,
	}
	host := "example.com"

	templateBody := "🧇 {item}\n${price}/spot • {spots_left} left of {total_spots}\n{spots_claimed} claimed\n{url}"

	result := RenderShareMessage(templateBody, waffle, stats, host)

	if !strings.Contains(result, "Test Drop") {
		t.Errorf("expected result to contain item title, got: %s", result)
	}
	if !strings.Contains(result, "25") {
		t.Errorf("expected result to contain price 25, got: %s", result)
	}
	if !strings.Contains(result, "100") {
		t.Errorf("expected result to contain total_spots 100, got: %s", result)
	}
	// spots_left = 100 - 45 - 5 = 50
	if !strings.Contains(result, "50") {
		t.Errorf("expected result to contain spots_left 50, got: %s", result)
	}
	// spots_claimed = 45 + 5 = 50
	if !strings.Contains(result, "50") {
		t.Errorf("expected result to contain spots_claimed 50, got: %s", result)
	}
	if !strings.Contains(result, "https://example.com/waffle/test-drop-abc123") {
		t.Errorf("expected result to contain URL, got: %s", result)
	}
}

func TestRenderShareMessage_NoPlaceholders(t *testing.T) {
	waffle := &models.Waffle{
		Title: "Test",
		Slug:  "test",
	}
	stats := map[string]interface{}{
		"total_spots": 10,
		"paid":        0,
		"pending":     0,
	}

	result := RenderShareMessage("plain text message", waffle, stats, "example.com")
	if result != "plain text message" {
		t.Errorf("expected unchanged message, got %q", result)
	}
}

func TestRenderShareMessage_ZeroSpots(t *testing.T) {
	waffle := &models.Waffle{
		Title:     "Empty Drop",
		Slug:      "empty-drop",
		SpotPrice: 10,
	}
	stats := map[string]interface{}{
		"total_spots": 0,
		"paid":        0,
		"pending":     0,
	}

	result := RenderShareMessage("{spots_left} {spots_claimed}", waffle, stats, "example.com")
	if result != "0 0" {
		t.Errorf("expected '0 0', got %q", result)
	}
}
