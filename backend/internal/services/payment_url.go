package services

import (
	"fmt"
	"strings"

	"github.com/syrup/backend/internal/models"
)

// GeneratePaymentURL returns a clickable payment URL for the given payment method.
// If handle_or_url already starts with "http", it's returned as-is.
// Otherwise, a platform-specific URL is generated based on the payment method type.
func GeneratePaymentURL(pm models.PaymentMethod) string {
	handleOrURL := strings.TrimSpace(pm.HandleOrURL)
	if handleOrURL == "" {
		return ""
	}

	if strings.HasPrefix(strings.ToLower(handleOrURL), "http") {
		return handleOrURL
	}

	switch pm.Type {
	case models.PaymentMethodTypeVenmo:
		return fmt.Sprintf("https://venmo.com/%s", handleOrURL)
	case models.PaymentMethodTypePayPal:
		return fmt.Sprintf("https://paypal.me/%s", handleOrURL)
	case models.PaymentMethodTypeCashApp:
		return fmt.Sprintf("https://cash.app/$%s", handleOrURL)
	case models.PaymentMethodTypeZelle:
		return handleOrURL
	default:
		return handleOrURL
	}
}