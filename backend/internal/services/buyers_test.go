package services

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

// TestMain is intentionally omitted here; the package-wide setup is provided by
// waffle_test.go (db.Connect + db.RunMigrations). Keeping a single TestMain per
// package avoids a duplicate-definition compile error.

// testBuyerCardWafflePrefix identifies test waffles for cleanup.
const testBuyerCardWafflePrefix = "test-buyer-card-"

// testBuyerCardHandlePrefix identifies test Instagram handles for cleanup.
const testBuyerCardHandlePrefix = "testbuyercard"

func cleanupBuyerCardTestWaffles(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM waffles WHERE slug LIKE $1`, testBuyerCardWafflePrefix+"%")
	if err != nil {
		t.Fatalf("cleanup buyer card test waffles: %v", err)
	}
}

func cleanupBuyerCardTestStats(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM buyer_stats WHERE instagram_handle LIKE $1`, testBuyerCardHandlePrefix+"%")
	if err != nil {
		t.Fatalf("cleanup buyer card test stats: %v", err)
	}
}

func cleanupBuyerCardTestUsers(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM users WHERE instagram_handle LIKE $1`, testBuyerCardHandlePrefix+"%")
	if err != nil {
		t.Fatalf("cleanup buyer card test users: %v", err)
	}
}

// cleanupBuyerCardTests removes all test data in dependency order.
func cleanupBuyerCardTests(t *testing.T) {
	t.Helper()
	cleanupBuyerCardTestWaffles(t)
	cleanupBuyerCardTestStats(t)
	cleanupBuyerCardTestUsers(t)
}

// floatApprox compares two float64 values within a small epsilon.
func floatApprox(a, b float64) bool {
	const epsilon = 1e-9
	return math.Abs(a-b) <= epsilon
}

// ensureDefaultTemplateExists checks if a default message template exists and
// creates one if not. This prevents FK violations when CreateWaffle looks up the
// default template but other tests have deleted all templates from the shared DB.
func ensureDefaultTemplateExists(t *testing.T) {
	t.Helper()
	_, err := GetDefaultMessageTemplate()
	if err == nil {
		return
	}
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO message_templates (name, body, is_default, created_at, updated_at)
		VALUES ('Default Hype Drop', E'🧇 NEW WAFFLE DROP 🧇\n\n{item}\n\n${price}/spot • {spots_left} of {total_spots} left\n\nClaim your spot 👇\n{url}', true, NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("ensure default template: %v", err)
	}
}

// createCompletedWaffleForBuyer creates a waffle, claims buyerSpotCount spots for
// buyerHandle, fills the rest with other test handles, marks every spot paid, and
// completes the waffle with the supplied winning spot numbers. The waffle title is
// set to title after creation so the slug stays prefixed for cleanup.
func createCompletedWaffleForBuyer(t *testing.T, title string, totalSpots, itemCount, buyerSpotCount int, buyerHandle string, winningSpotNumbers []int) *models.Waffle {
	t.Helper()

	if buyerSpotCount > totalSpots {
		t.Fatalf("buyerSpotCount (%d) cannot exceed totalSpots (%d)", buyerSpotCount, totalSpots)
	}

	ensureDefaultTemplateExists(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:      testBuyerCardWafflePrefix + uuid.New().String()[:8],
		TotalSpots: totalSpots,
		SpotPrice:  1,
		ItemCount:  itemCount,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	_, err = db.Pool.Exec(context.Background(),
		`UPDATE waffles SET title = $1 WHERE id = $2`, title, waffle.ID)
	if err != nil {
		t.Fatalf("update waffle title: %v", err)
	}

	buyerSpots := make([]int, 0, buyerSpotCount)
	for i := 1; i <= buyerSpotCount; i++ {
		buyerSpots = append(buyerSpots, i)
	}
	if err := ClaimSpots(waffle.ID, buyerSpots, buyerHandle); err != nil {
		t.Fatalf("ClaimSpots for buyer: %v", err)
	}

	for i := buyerSpotCount + 1; i <= totalSpots; i++ {
		handle := fmt.Sprintf("%s-other-%d", testBuyerCardHandlePrefix, i)
		if err := ClaimSpots(waffle.ID, []int{i}, handle); err != nil {
			t.Fatalf("ClaimSpots for other buyer: %v", err)
		}
	}

	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}
	for _, spot := range spots {
		if spot.Status == models.SpotStatusPending {
			if err := MarkSpotPaid(spot.ID); err != nil {
				t.Fatalf("MarkSpotPaid spot %d: %v", spot.Number, err)
			}
		}
	}

	if err := SetWinner(waffle.ID, winningSpotNumbers); err != nil {
		t.Fatalf("SetWinner: %v", err)
	}

	waffle, err = GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID: %v", err)
	}

	return waffle
}

func TestParseTrophyItems(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected []string
	}{
		{
			name:     "single quoted item",
			title:    `"Jordan 1"`,
			expected: []string{"Jordan 1"},
		},
		{
			name:     "multiple quoted items",
			title:    `Waffle for "Jordan 1" and "Dunk Low"`,
			expected: []string{"Jordan 1", "Dunk Low"},
		},
		{
			name:     "no quotes",
			title:    "No quotes here",
			expected: []string{},
		},
		{
			name:     "unmatched quote",
			title:    `Unmatched "quote`,
			expected: nil,
		},
		{
			name:     "empty quotes",
			title:    `Empty "" quotes`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTrophyItems(tt.title)
			if tt.expected == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, got)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("item %d: expected %q, got %q", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestBuyerCard_Normal(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-normal"
	_ = createCompletedWaffleForBuyer(t, "Normal Test", 20, 2, 10, buyer, []int{1, 2})

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if card.InstagramHandle != buyer {
		t.Errorf("expected handle %q, got %q", buyer, card.InstagramHandle)
	}
	if card.Stats == nil {
		t.Fatal("expected stats, got nil")
	}
	if card.Stats.TotalWins != 2 || card.Stats.TotalSpotsClaimed != 10 {
		t.Errorf("expected 2 wins / 10 spots, got %d / %d", card.Stats.TotalWins, card.Stats.TotalSpotsClaimed)
	}
	if !floatApprox(card.WinRate, 0.2) {
		t.Errorf("expected win rate 0.2, got %v", card.WinRate)
	}
	if !floatApprox(card.ExpectedWinRate, 0.5) {
		t.Errorf("expected expected-win-rate 0.5, got %v", card.ExpectedWinRate)
	}
	if !floatApprox(card.LuckRating, -0.3) {
		t.Errorf("expected luck -0.3, got %v", card.LuckRating)
	}
}

func TestBuyerCard_NegativeLuck(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-lucky"
	_ = createCompletedWaffleForBuyer(t, "Lucky Test", 20, 3, 10, buyer, []int{1, 2, 3})

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if card.InstagramHandle != buyer {
		t.Errorf("expected handle %q, got %q", buyer, card.InstagramHandle)
	}
	if !floatApprox(card.WinRate, 0.3) {
		t.Errorf("expected win rate 0.3, got %v", card.WinRate)
	}
	if !floatApprox(card.ExpectedWinRate, 0.5) {
		t.Errorf("expected expected-win-rate 0.5, got %v", card.ExpectedWinRate)
	}
	if !floatApprox(card.LuckRating, -0.2) {
		t.Errorf("expected luck -0.2, got %v", card.LuckRating)
	}
}

func TestBuyerCard_Hot(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-hot"
	_ = createCompletedWaffleForBuyer(t, "Hot Test", 20, 5, 10, buyer, []int{1, 2, 3, 4, 5})

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if card.InstagramHandle != buyer {
		t.Errorf("expected handle %q, got %q", buyer, card.InstagramHandle)
	}
	if !floatApprox(card.WinRate, 0.5) {
		t.Errorf("expected win rate 0.5, got %v", card.WinRate)
	}
	if !floatApprox(card.ExpectedWinRate, 0.5) {
		t.Errorf("expected expected-win-rate 0.5, got %v", card.ExpectedWinRate)
	}
	if !floatApprox(card.LuckRating, 0.0) {
		t.Errorf("expected luck 0.0, got %v", card.LuckRating)
	}
}

func TestBuyerCard_NoSpots(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-nospots"
	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if card.InstagramHandle != buyer {
		t.Errorf("expected handle %q, got %q", buyer, card.InstagramHandle)
	}
	if card.Stats != nil {
		t.Errorf("expected nil stats, got %+v", card.Stats)
	}
	if !floatApprox(card.WinRate, 0.0) {
		t.Errorf("expected win rate 0.0, got %v", card.WinRate)
	}
	if !floatApprox(card.ExpectedWinRate, 0.0) {
		t.Errorf("expected expected-win-rate 0.0, got %v", card.ExpectedWinRate)
	}
	if !floatApprox(card.LuckRating, 0.0) {
		t.Errorf("expected luck 0.0, got %v", card.LuckRating)
	}
	if len(card.Trophies) != 0 {
		t.Errorf("expected empty trophies, got %v", card.Trophies)
	}
	if len(card.WaffleHistory) != 0 {
		t.Errorf("expected empty history, got %v", card.WaffleHistory)
	}
}

func TestBuyerCard_NoCompletedWaffles(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-nocompleted"
	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:      testBuyerCardWafflePrefix + uuid.New().String()[:8],
		TotalSpots: 20,
		SpotPrice:  1,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	if err := ClaimSpots(waffle.ID, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, buyer); err != nil {
		t.Fatalf("ClaimSpots: %v", err)
	}

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if card.InstagramHandle != buyer {
		t.Errorf("expected handle %q, got %q", buyer, card.InstagramHandle)
	}
	if card.Stats != nil {
		t.Errorf("expected nil stats for active-waffle-only buyer, got %+v", card.Stats)
	}
	if !floatApprox(card.WinRate, 0.0) {
		t.Errorf("expected win rate 0.0, got %v", card.WinRate)
	}
	if !floatApprox(card.ExpectedWinRate, 0.0) {
		t.Errorf("expected expected-win-rate 0.0, got %v", card.ExpectedWinRate)
	}
	if !floatApprox(card.LuckRating, 0.0) {
		t.Errorf("expected luck 0.0, got %v", card.LuckRating)
	}
	if len(card.Trophies) != 0 {
		t.Errorf("expected empty trophies, got %v", card.Trophies)
	}
}

func TestBuyerCard_ZeroTotalSpotsGuard(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-zerototal"
	waffle := createCompletedWaffleForBuyer(t, "Zero Total Guard Test", 20, 1, 5, buyer, []int{1})

	_, err := db.Pool.Exec(context.Background(),
		`UPDATE waffles SET total_spots = 0 WHERE id = $1`, waffle.ID)
	if err != nil {
		t.Fatalf("update waffle total_spots to 0: %v", err)
	}

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if card.InstagramHandle != buyer {
		t.Errorf("expected handle %q, got %q", buyer, card.InstagramHandle)
	}
	if card.Stats == nil {
		t.Fatal("expected stats, got nil")
	}
	if !floatApprox(card.ExpectedWinRate, 0.0) {
		t.Errorf("expected expected-win-rate 0.0 when completed waffle has zero total spots, got %v", card.ExpectedWinRate)
	}
}

func TestBuyerCard_TrophyQuotedTitle(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-trophy-quoted"
	_ = createCompletedWaffleForBuyer(t, `"Jordan 1"`, 20, 1, 1, buyer, []int{1})

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if len(card.Trophies) != 1 || card.Trophies[0] != "Jordan 1" {
		t.Errorf("expected trophies [Jordan 1], got %v", card.Trophies)
	}
}

func TestBuyerCard_TrophyMultipleQuotedItems(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-trophy-multi"
	_ = createCompletedWaffleForBuyer(t, `"Jordan 1" and "Dunk Low"`, 20, 2, 2, buyer, []int{1, 2})

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if len(card.Trophies) != 2 {
		t.Fatalf("expected 2 trophies, got %v", card.Trophies)
	}
	if card.Trophies[0] != "Jordan 1" || card.Trophies[1] != "Dunk Low" {
		t.Errorf("expected trophies [Jordan 1 Dunk Low], got %v", card.Trophies)
	}
}

func TestBuyerCard_TrophyNoQuotedTitle(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-trophy-none"
	_ = createCompletedWaffleForBuyer(t, "No quotes here", 20, 1, 1, buyer, []int{1})

	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if len(card.Trophies) != 0 {
		t.Errorf("expected empty trophies, got %v", card.Trophies)
	}
}

func TestBuyerCard_UnknownBuyer(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	buyer := testBuyerCardHandlePrefix + "-unknown"
	card, err := ComputeBuyerCardData(buyer)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData returned error for unknown buyer: %v", err)
	}
	if card.InstagramHandle != buyer {
		t.Errorf("expected handle %q, got %q", buyer, card.InstagramHandle)
	}
	if card.Stats != nil {
		t.Errorf("expected nil stats for unknown buyer, got %+v", card.Stats)
	}
	if !floatApprox(card.WinRate, 0.0) || !floatApprox(card.ExpectedWinRate, 0.0) || !floatApprox(card.LuckRating, 0.0) {
		t.Errorf("expected zero rates, got win=%v expected=%v luck=%v", card.WinRate, card.ExpectedWinRate, card.LuckRating)
	}
	if len(card.Trophies) != 0 || len(card.WaffleHistory) != 0 {
		t.Errorf("expected empty trophies/history, got trophies=%v history=%v", card.Trophies, card.WaffleHistory)
	}
}

func TestBuyerCard_HandleNormalization(t *testing.T) {
	defer cleanupBuyerCardTests(t)

	rawHandle := "@" + testBuyerCardHandlePrefix + "-Normalize"
	normalizedHandle := testBuyerCardHandlePrefix + "-normalize"

	_ = createCompletedWaffleForBuyer(t, "Normalize Test", 20, 1, 1, rawHandle, []int{1})

	card, err := ComputeBuyerCardData(rawHandle)
	if err != nil {
		t.Fatalf("ComputeBuyerCardData: %v", err)
	}
	if card.InstagramHandle != normalizedHandle {
		t.Errorf("expected normalized handle %q, got %q", normalizedHandle, card.InstagramHandle)
	}
	if card.Stats == nil || card.Stats.TotalWins != 1 {
		t.Errorf("expected 1 win for normalized buyer, got stats=%+v", card.Stats)
	}
}
