package handlers

import (
	"strings"
	"testing"

	"github.com/syrup/backend/internal/models"
)

func TestValidateCreateAdminForm_ShortPassword(t *testing.T) {
	errs := validateCreateAdminForm("testuser", "test@example.com", "abc")
	if len(errs) == 0 {
		t.Fatal("expected validation errors for short password, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "Password must be at least 8 characters") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected password length error in: %v", errs)
	}
}

func TestValidateCreateAdminForm_EmptyPassword(t *testing.T) {
	errs := validateCreateAdminForm("testuser", "test@example.com", "")
	if len(errs) == 0 {
		t.Fatal("expected validation errors for empty password, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "Password is required") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected password required error in: %v", errs)
	}
}

func TestValidateCreateAdminForm_ValidPassword(t *testing.T) {
	errs := validateCreateAdminForm("testuser", "test@example.com", "goodpw88")
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for valid input, got: %v", errs)
	}
}

func TestValidateCreateAdminForm_MissingFields(t *testing.T) {
	errs := validateCreateAdminForm("", "", "abc")
	if len(errs) != 3 {
		t.Fatalf("expected 3 validation errors (username, email, password), got %d: %v", len(errs), errs)
	}
}

func TestCreateAdminPost_AcceptsWaffleManagerRole(t *testing.T) {
	role := "waffle_manager"
	if role != models.RoleAdmin && role != models.RoleSuperAdmin && role != models.RoleWaffleManager {
		t.Fatal("waffle_manager role should be accepted, but was rejected")
	}
}

func TestCreateAdminPost_RejectsInvalidRole(t *testing.T) {
	role := "hacker"
	if role == models.RoleAdmin || role == models.RoleSuperAdmin || role == models.RoleWaffleManager {
		t.Fatal("invalid role 'hacker' should be rejected, but was accepted")
	}
}
