package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/models"
)

func TestGeneratePaymentURL(t *testing.T) {
	tests := []struct {
		name     string
		pm       models.PaymentMethod
		expected string
	}{
		{
			name:     "venmo handle",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeVenmo, HandleOrURL: "mxkxng"},
			expected: "https://venmo.com/mxkxng",
		},
		{
			name:     "paypal handle",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypePayPal, HandleOrURL: "syrup"},
			expected: "https://paypal.me/syrup",
		},
		{
			name:     "cashapp handle adds dollar prefix",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeCashApp, HandleOrURL: "notfixingit"},
			expected: "https://cash.app/$notfixingit",
		},
		{
			name:     "zelle returns handle as-is",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeZelle, HandleOrURL: "mike@zelle.com"},
			expected: "mike@zelle.com",
		},
		{
			name:     "venmo full url passthrough",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeVenmo, HandleOrURL: "https://venmo.com/mxkxng"},
			expected: "https://venmo.com/mxkxng",
		},
		{
			name:     "paypal full url passthrough",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypePayPal, HandleOrURL: "https://paypal.me/syrup"},
			expected: "https://paypal.me/syrup",
		},
		{
			name:     "cashapp full url passthrough",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeCashApp, HandleOrURL: "https://cash.app/$test"},
			expected: "https://cash.app/$test",
		},
		{
			name:     "zelle full url passthrough",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeZelle, HandleOrURL: "https://zelle.com/pay"},
			expected: "https://zelle.com/pay",
		},
		{
			name:     "http URL passthrough",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeVenmo, HandleOrURL: "http://example.com/pay"},
			expected: "http://example.com/pay",
		},
		{
			name:     "unknown type returns handle as-is",
			pm:       models.PaymentMethod{Type: "square", HandleOrURL: "squarehandle"},
			expected: "squarehandle",
		},
		{
			name:     "empty handle returns empty",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeVenmo, HandleOrURL: ""},
			expected: "",
		},
		{
			name:     "whitespace handle trimmed and empty",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypePayPal, HandleOrURL: "  "},
			expected: "",
		},
		{
			name:     "venmo handle with spaces is preserved",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeVenmo, HandleOrURL: "  mxkxng  "},
			expected: "https://venmo.com/mxkxng",
		},
		{
			name:     "HTTPS uppercase passthrough",
			pm:       models.PaymentMethod{Type: models.PaymentMethodTypeVenmo, HandleOrURL: "HTTPS://venmo.com/mxkxng"},
			expected: "HTTPS://venmo.com/mxkxng",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePaymentURL(tt.pm)
			if result != tt.expected {
				t.Errorf("GeneratePaymentURL(%+v) = %q, want %q", tt.pm, result, tt.expected)
			}
		})
	}
}

// TestGeneratePaymentURL_NoID ensures the function works without populating the ID field.
func TestGeneratePaymentURL_NoID(t *testing.T) {
	pm := models.PaymentMethod{
		Type:        models.PaymentMethodTypeVenmo,
		DisplayName: "Venmo",
		HandleOrURL: "testuser",
		IsActive:    true,
	}
	expected := "https://venmo.com/testuser"
	result := GeneratePaymentURL(pm)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestGeneratePaymentURL_WithID ensures the UUID field is properly ignored during generation.
func TestGeneratePaymentURL_WithID(t *testing.T) {
	pm := models.PaymentMethod{
		ID:          uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
		Type:        models.PaymentMethodTypeCashApp,
		DisplayName: "Cash App",
		HandleOrURL: "cashuser",
		IsActive:    true,
	}
	expected := "https://cash.app/$cashuser"
	result := GeneratePaymentURL(pm)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}