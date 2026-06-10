package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

// testPMPrefix is used to identify and clean up test payment methods.
const testPMPrefix = "test-payment-method-"

// testPMWafflePrefix is used to identify and clean up test waffles created for payment method tests.
const testPMWafflePrefix = "test-pm-waffle-"

// createTestPaymentMethod creates a payment method and returns it.
// If isActive is false, the method is immediately deactivated.
func createTestPaymentMethod(t *testing.T, pmType, displayName, handleOrURL string, isActive bool) *models.PaymentMethod {
	t.Helper()

	pm, err := CreatePaymentMethod(models.CreatePaymentMethodRequest{
		Type:        pmType,
		DisplayName: displayName,
		HandleOrURL: handleOrURL,
	})
	if err != nil {
		t.Fatalf("create test payment method: %v", err)
	}

	if !isActive {
		if err := DeactivatePaymentMethod(pm.ID); err != nil {
			t.Fatalf("deactivate test payment method: %v", err)
		}
		pm, err = GetPaymentMethodByID(pm.ID)
		if err != nil {
			t.Fatalf("get deactivated payment method: %v", err)
		}
	}

	return pm
}

// cleanupTestPaymentMethods removes all test payment methods and any waffle_payment_method links.
func cleanupTestPaymentMethods(t *testing.T) {
	t.Helper()

	// Delete waffle_payment_method links first (FK references payment_methods)
	_, err := db.Pool.Exec(context.Background(), `
		DELETE FROM waffle_payment_methods
		WHERE payment_method_id IN (
			SELECT id FROM payment_methods WHERE display_name LIKE $1
		)
	`, testPMPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup waffle_payment_methods: %v", err)
	}

	_, err = db.Pool.Exec(context.Background(),
		`DELETE FROM payment_methods WHERE display_name LIKE $1`, testPMPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup test payment methods: %v", err)
	}
}

// cleanupTestPMWaffles removes test waffles created by payment method tests.
func cleanupTestPMWaffles(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM waffles WHERE slug LIKE $1`, testPMWafflePrefix+"%")
	if err != nil {
		t.Fatalf("cleanup test pm waffles: %v", err)
	}
}

func TestCreatePaymentMethod(t *testing.T) {
	defer cleanupTestPaymentMethods(t)

	pm, err := CreatePaymentMethod(models.CreatePaymentMethodRequest{
		Type:        models.PaymentMethodTypeVenmo,
		DisplayName: testPMPrefix + "Venmo",
		HandleOrURL: "@testvenmo",
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod returned error: %v", err)
	}
	if pm.ID == uuid.Nil {
		t.Error("expected non-nil UUID for new payment method")
	}
	if pm.Type != models.PaymentMethodTypeVenmo {
		t.Errorf("expected type %q, got %q", models.PaymentMethodTypeVenmo, pm.Type)
	}
	if pm.DisplayName != testPMPrefix+"Venmo" {
		t.Errorf("expected display_name %q, got %q", testPMPrefix+"Venmo", pm.DisplayName)
	}
	if pm.HandleOrURL != "@testvenmo" {
		t.Errorf("expected handle_or_url %q, got %q", "@testvenmo", pm.HandleOrURL)
	}
	if !pm.IsActive {
		t.Error("expected new payment method to be active")
	}
}

func TestListPaymentMethods(t *testing.T) {
	defer cleanupTestPaymentMethods(t)

	// Create one active and one inactive payment method
	activeDisplayName := testPMPrefix + "Active PayPal"
	inactiveDisplayName := testPMPrefix + "Inactive Zelle"

	createTestPaymentMethod(t, models.PaymentMethodTypePayPal, activeDisplayName, "paypal.me/test", true)
	createTestPaymentMethod(t, models.PaymentMethodTypeZelle, inactiveDisplayName, "zelle@test.com", false)

	methods, err := ListPaymentMethods()
	if err != nil {
		t.Fatalf("ListPaymentMethods returned error: %v", err)
	}

	foundActive := false
	foundInactive := false
	for _, m := range methods {
		if m.DisplayName == activeDisplayName {
			foundActive = true
		}
		if m.DisplayName == inactiveDisplayName {
			foundInactive = true
		}
	}

	if !foundActive {
		t.Errorf("expected active payment method %q in list", activeDisplayName)
	}
	if foundInactive {
		t.Errorf("expected inactive payment method %q NOT in active-only list", inactiveDisplayName)
	}
}

func TestListAllPaymentMethods(t *testing.T) {
	defer cleanupTestPaymentMethods(t)

	activeDisplayName := testPMPrefix + "Active CashApp"
	inactiveDisplayName := testPMPrefix + "Inactive Venmo"

	createTestPaymentMethod(t, models.PaymentMethodTypeCashApp, activeDisplayName, "cash.me/test", true)
	createTestPaymentMethod(t, models.PaymentMethodTypeVenmo, inactiveDisplayName, "@inactive", false)

	methods, err := ListAllPaymentMethods()
	if err != nil {
		t.Fatalf("ListAllPaymentMethods returned error: %v", err)
	}

	foundActive := false
	foundInactive := false
	for _, m := range methods {
		if m.DisplayName == activeDisplayName {
			foundActive = true
		}
		if m.DisplayName == inactiveDisplayName {
			foundInactive = true
		}
	}

	if !foundActive {
		t.Errorf("expected active payment method %q in all list", activeDisplayName)
	}
	if !foundInactive {
		t.Errorf("expected inactive payment method %q in all list", inactiveDisplayName)
	}
}

func TestUpdatePaymentMethod(t *testing.T) {
	defer cleanupTestPaymentMethods(t)

	pm := createTestPaymentMethod(t, models.PaymentMethodTypeVenmo, testPMPrefix+"Update Me", "@before", true)

	newDisplayName := testPMPrefix + "Updated Name"
	newHandle := "@after"

	updated, err := UpdatePaymentMethod(pm.ID, models.UpdatePaymentMethodRequest{
		DisplayName: &newDisplayName,
		HandleOrURL: &newHandle,
	})
	if err != nil {
		t.Fatalf("UpdatePaymentMethod returned error: %v", err)
	}
	if updated.DisplayName != newDisplayName {
		t.Errorf("expected display_name %q, got %q", newDisplayName, updated.DisplayName)
	}
	if updated.HandleOrURL != newHandle {
		t.Errorf("expected handle_or_url %q, got %q", newHandle, updated.HandleOrURL)
	}
	if updated.Type != models.PaymentMethodTypeVenmo {
		t.Errorf("expected type unchanged %q, got %q", models.PaymentMethodTypeVenmo, updated.Type)
	}

	// Verify partial update (nil fields should not change)
	partialUpdated, err := UpdatePaymentMethod(pm.ID, models.UpdatePaymentMethodRequest{
		DisplayName: nil,
		HandleOrURL: nil,
	})
	if err != nil {
		t.Fatalf("UpdatePaymentMethod partial returned error: %v", err)
	}
	if partialUpdated.DisplayName != newDisplayName {
		t.Errorf("expected display_name unchanged %q, got %q", newDisplayName, partialUpdated.DisplayName)
	}
	if partialUpdated.HandleOrURL != newHandle {
		t.Errorf("expected handle_or_url unchanged %q, got %q", newHandle, partialUpdated.HandleOrURL)
	}
}

func TestDeactivatePaymentMethod(t *testing.T) {
	defer cleanupTestPaymentMethods(t)

	pm := createTestPaymentMethod(t, models.PaymentMethodTypePayPal, testPMPrefix+"Deactivate Me", "@paypal", true)
	if !pm.IsActive {
		t.Fatal("expected payment method to be active before deactivation")
	}

	err := DeactivatePaymentMethod(pm.ID)
	if err != nil {
		t.Fatalf("DeactivatePaymentMethod returned error: %v", err)
	}

	reloaded, err := GetPaymentMethodByID(pm.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodByID after deactivate: %v", err)
	}
	if reloaded.IsActive {
		t.Error("expected payment method to be inactive after deactivation")
	}
}

func TestGetPaymentMethodsForWaffle(t *testing.T) {
	defer cleanupTestPaymentMethods(t)
	defer cleanupTestPMWaffles(t)

	// Create payment methods
	pm1 := createTestPaymentMethod(t, models.PaymentMethodTypeVenmo, testPMPrefix+"Waffle Venmo", "@venmo", true)
	pm2 := createTestPaymentMethod(t, models.PaymentMethodTypePayPal, testPMPrefix+"Waffle PayPal", "paypal.me/test", true)
	_ = createTestPaymentMethod(t, models.PaymentMethodTypeCashApp, testPMPrefix+"Waffle CashApp", "cash.me/test", true)

	// Create a waffle
	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:      testPMWafflePrefix + "payment-methods",
		TotalSpots: 5,
		SpotPrice:  10,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	// Link pm1 and pm2 to the waffle
	err = SetWafflePaymentMethods(waffle.ID, []uuid.UUID{pm1.ID, pm2.ID})
	if err != nil {
		t.Fatalf("SetWafflePaymentMethods: %v", err)
	}

	methods, err := GetPaymentMethodsForWaffle(waffle.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodsForWaffle returned error: %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("expected 2 payment methods, got %d", len(methods))
	}

	foundIDs := make(map[uuid.UUID]bool)
	for _, m := range methods {
		foundIDs[m.ID] = true
	}
	if !foundIDs[pm1.ID] {
		t.Errorf("expected pm1 (%s) in waffle payment methods", pm1.ID)
	}
	if !foundIDs[pm2.ID] {
		t.Errorf("expected pm2 (%s) in waffle payment methods", pm2.ID)
	}
}

func TestGetPaymentMethodUsageCount(t *testing.T) {
	defer cleanupTestPaymentMethods(t)
	defer cleanupTestPMWaffles(t)
	defer cleanupWinnerTestWaffles(t)

	pm := createTestPaymentMethod(t, models.PaymentMethodTypeZelle, testPMPrefix+"Usage Zelle", "zelle@test.com", true)

	// Initially no usage
	count, err := GetPaymentMethodUsageCount(pm.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodUsageCount returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected usage count 0, got %d", count)
	}

	// Create an active waffle and link the payment method
	activeWaffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:      testPMWafflePrefix + "active-usage",
		TotalSpots: 5,
		SpotPrice:  10,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	err = SetWafflePaymentMethods(activeWaffle.ID, []uuid.UUID{pm.ID})
	if err != nil {
		t.Fatalf("SetWafflePaymentMethods: %v", err)
	}

	count, err = GetPaymentMethodUsageCount(pm.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodUsageCount after active link: %v", err)
	}
	if count != 1 {
		t.Errorf("expected usage count 1 for active waffle, got %d", count)
	}

	// Create a completed waffle and link the payment method — should NOT count
	completedWaffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:      testPMWafflePrefix + "completed-usage",
		TotalSpots: 2,
		SpotPrice:  5,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	// Claim and pay all spots, then set winner to complete the waffle
	if err := ClaimSpots(completedWaffle.ID, []int{1, 2}, "buyer_test"); err != nil {
		t.Fatalf("ClaimSpots: %v", err)
	}
	spots, err := GetSpotsByWaffleID(completedWaffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}
	for _, s := range spots {
		if s.Status == models.SpotStatusPending {
			if err := MarkSpotPaid(s.ID); err != nil {
				t.Fatalf("MarkSpotPaid: %v", err)
			}
		}
	}
	if err := SetWinner(completedWaffle.ID, []int{1}); err != nil {
		t.Fatalf("SetWinner: %v", err)
	}

	err = SetWafflePaymentMethods(completedWaffle.ID, []uuid.UUID{pm.ID})
	if err != nil {
		t.Fatalf("SetWafflePaymentMethods for completed waffle: %v", err)
	}

	// Count should still be 1 (only active waffles count)
	count, err = GetPaymentMethodUsageCount(pm.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodUsageCount after completed link: %v", err)
	}
	if count != 1 {
		t.Errorf("expected usage count 1 (active only), got %d", count)
	}
}

func TestCreatePaymentMethod_InvalidType(t *testing.T) {
	defer cleanupTestPaymentMethods(t)

	_, err := CreatePaymentMethod(models.CreatePaymentMethodRequest{
		Type:        "bitcoin",
		DisplayName: testPMPrefix + "Invalid",
		HandleOrURL: "btc:test",
	})
	if err == nil {
		t.Fatal("expected error for invalid payment method type, got nil")
	}
}

func TestUpdatePaymentMethod_NotFound(t *testing.T) {
	defer cleanupTestPaymentMethods(t)

	randomID := uuid.New()
	newName := testPMPrefix + "Does Not Matter"
	_, err := UpdatePaymentMethod(randomID, models.UpdatePaymentMethodRequest{
		DisplayName: &newName,
	})
	if err == nil {
		t.Fatal("expected error updating non-existent payment method, got nil")
	}
}
