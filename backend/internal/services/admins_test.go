package services

import (
	"context"
	"testing"

	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

const (
	testAdminPrefix = "test-services-admin-"
)

func cleanupAdminsTestData(t *testing.T) {
	t.Helper()
	// Clean up reset tokens
	_, _ = db.Pool.Exec(context.Background(), `
		DELETE FROM password_reset_tokens WHERE admin_id IN (
			SELECT id FROM admins WHERE username LIKE $1 || '%'
		)
	`, testAdminPrefix)
	// Clean up admins
	_, _ = db.Pool.Exec(context.Background(), `
		DELETE FROM admins WHERE username LIKE $1 || '%'
	`, testAdminPrefix)
}

func boolPtr(b bool) *bool {
	return &b
}

func TestAdminService_CRUDAndAuth(t *testing.T) {
	if db.Pool == nil {
		t.Skip("Postgres not available")
	}
	defer cleanupAdminsTestData(t)

	username := testAdminPrefix + "service"
	email := username + "@syrup.test"
	req := models.CreateAdminRequest{
		Username:    username,
		Password:    "super_secure_syrup_pass",
		Email:       &email,
		FirstName:   strPtr("Test"),
		LastName:    strPtr("User"),
		DisplayName: strPtr("Tester"),
		Role:        "admin",
	}

	// Create
	admin, err := CreateAdmin(req)
	if err != nil {
		t.Fatalf("CreateAdmin failed: %v", err)
	}

	if admin.Username != username {
		t.Errorf("expected username %s, got %s", username, admin.Username)
	}

	// Get by username
	byUsername, err := GetAdminByUsername(username)
	if err != nil {
		t.Fatalf("GetAdminByUsername failed: %v", err)
	}
	if byUsername.ID != admin.ID {
		t.Errorf("ID mismatch: %s vs %s", byUsername.ID, admin.ID)
	}

	// Get by ID
	byID, err := GetAdminByID(admin.ID)
	if err != nil {
		t.Fatalf("GetAdminByID failed: %v", err)
	}
	if byID.ID != admin.ID {
		t.Errorf("ID mismatch: %s vs %s", byID.ID, admin.ID)
	}

	// Get by email
	byEmail, err := GetAdminByEmail(email)
	if err != nil {
		t.Fatalf("GetAdminByEmail failed: %v", err)
	}
	if byEmail.ID != admin.ID {
		t.Errorf("ID mismatch: %s vs %s", byEmail.ID, admin.ID)
	}

	// Authenticate Success
	authAdmin, err := AuthenticateAdmin(username, "super_secure_syrup_pass")
	if err != nil {
		t.Fatalf("AuthenticateAdmin failed: %v", err)
	}
	if authAdmin.ID != admin.ID {
		t.Errorf("authenticated admin ID mismatch: %s vs %s", authAdmin.ID, admin.ID)
	}

	// Authenticate Fail
	_, err = AuthenticateAdmin(username, "wrong_password")
	if err == nil {
		t.Errorf("expected authentication failure for wrong password")
	}

	// Update Admin profile/role
	updateReq := models.UpdateAdminRequest{
		DisplayName: strPtr("New Display Name"),
		Role:        strPtr("super_admin"),
		Active:      boolPtr(true),
	}
	updated, err := UpdateAdmin(admin.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateAdmin failed: %v", err)
	}
	if updated.DisplayName == nil || *updated.DisplayName != "New Display Name" {
		t.Errorf("expected display name 'New Display Name', got %v", updated.DisplayName)
	}
	if updated.Role != "super_admin" {
		t.Errorf("expected role 'super_admin', got %s", updated.Role)
	}

	// Update Timezone
	tzAdmin, err := UpdateAdminTimezone(admin.ID, "America/New_York")
	if err != nil {
		t.Fatalf("UpdateAdminTimezone failed: %v", err)
	}
	if tzAdmin.Timezone != "America/New_York" {
		t.Errorf("expected timezone America/New_York, got %s", tzAdmin.Timezone)
	}

	// Token Generation
	token, err := GenerateAdminToken(admin)
	if err != nil {
		t.Fatalf("GenerateAdminToken failed: %v", err)
	}
	if token == "" {
		t.Errorf("expected non-empty JWT token")
	}

	// Password Reset Tokens
	resetToken, err := CreatePasswordResetToken(admin.ID)
	if err != nil {
		t.Fatalf("CreatePasswordResetToken failed: %v", err)
	}
	if resetToken == "" {
		t.Errorf("expected non-empty reset token")
	}

	validID, err := ValidatePasswordResetToken(resetToken)
	if err != nil {
		t.Fatalf("ValidatePasswordResetToken failed: %v", err)
	}
	if validID != admin.ID {
		t.Errorf("expected reset token admin ID %s, got %s", admin.ID, validID)
	}

	// Deactivate
	_, err = db.Pool.Exec(context.Background(), `UPDATE admins SET active = false WHERE id = $1`, admin.ID)
	if err != nil {
		t.Fatalf("failed to deactivate admin: %v", err)
	}
	deactAdmin, err := GetAdminByID(admin.ID)
	if err != nil {
		t.Fatalf("GetAdminByID failed: %v", err)
	}
	if deactAdmin.Active {
		t.Errorf("expected admin to be inactive")
	}

	// List
	list, err := ListAdmins()
	if err != nil {
		t.Fatalf("ListAdmins failed: %v", err)
	}
	if len(list) < 1 {
		t.Errorf("expected at least 1 admin, got %d", len(list))
	}
}
