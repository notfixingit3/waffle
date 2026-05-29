package services

import (
	"strings"
	"testing"
)

func TestValidatePassword_RejectsShortPassword(t *testing.T) {
	err := ValidatePassword("short")
	if err == nil || !strings.Contains(err.Error(), "at least 8 characters") {
		t.Fatalf("expected minimum length error, got %v", err)
	}
}

func TestValidatePassword_RejectsCommonPassword(t *testing.T) {
	err := ValidatePassword("password")
	if err == nil || !strings.Contains(err.Error(), "too common") {
		t.Fatalf("expected common password error, got %v", err)
	}
}

func TestValidatePassword_RejectsAdmin123(t *testing.T) {
	err := ValidatePassword("admin123")
	if err == nil || !strings.Contains(err.Error(), "too common") {
		t.Fatalf("expected common password error, got %v", err)
	}
}

func TestValidatePassword_AcceptsStrongPassword(t *testing.T) {
	if err := ValidatePassword("syrup_is_tasty_2024"); err != nil {
		t.Fatalf("expected strong password to pass, got %v", err)
	}
}
