package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/migrations"
)

func TestMain(m *testing.M) {
	pool, err := db.Connect()
	if err != nil {
		// Postgres not available — skip all service tests
		os.Exit(0)
	}
	defer pool.Close()

	if err := db.RunMigrations(pool, migrations.FS); err != nil {
		panic("Failed to run database migrations for tests: " + err.Error())
	}

	os.Exit(m.Run())
}

// testSlugPrefix is used to identify and clean up test waffles.
const testSlugPrefix = "test-count-waffles-"

func insertTestWaffle(t *testing.T, status models.WaffleStatus, archived bool) uuid.UUID {
	t.Helper()

	id := uuid.New()
	slug := testSlugPrefix + id.String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO waffles (id, slug, title, total_spots, spot_price, status, archived, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, slug, "Test Waffle", 10, 5, status, archived, time.Now())
	if err != nil {
		t.Fatalf("insert test waffle: %v", err)
	}
	return id
}

func cleanupTestWaffles(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `DELETE FROM waffles WHERE slug LIKE $1 || '%'`, testSlugPrefix)
	if err != nil {
		t.Fatalf("cleanup test waffles: %v", err)
	}
}

func TestCountWaffles_Empty(t *testing.T) {
	defer cleanupTestWaffles(t)

	total, active, err := CountWaffles()
	if err != nil {
		t.Fatalf("CountWaffles() returned error: %v", err)
	}
	if total < 0 {
		t.Errorf("expected total >= 0, got %d", total)
	}
	if active < 0 {
		t.Errorf("expected active >= 0, got %d", active)
	}
	if active > total {
		t.Errorf("active (%d) should not exceed total (%d)", active, total)
	}
}

func TestCountWaffles_Mixed(t *testing.T) {
	defer cleanupTestWaffles(t)

	// Capture baseline counts before inserting test data.
	baselineTotal, baselineActive, err := CountWaffles()
	if err != nil {
		t.Fatalf("baseline CountWaffles() failed: %v", err)
	}

	// Insert 2 active non-archived waffles (counts toward active + total).
	insertTestWaffle(t, models.WaffleStatusActive, false)
	insertTestWaffle(t, models.WaffleStatusActive, false)

	// Insert 1 completed non-archived waffle (counts toward total only).
	insertTestWaffle(t, models.WaffleStatusCompleted, false)

	// Insert 1 active but archived waffle (counts toward total only).
	insertTestWaffle(t, models.WaffleStatusActive, true)

	total, active, err := CountWaffles()
	if err != nil {
		t.Fatalf("CountWaffles() returned error: %v", err)
	}

	expectedTotal := baselineTotal + 4
	expectedActive := baselineActive + 2 // only non-archived active

	if total != expectedTotal {
		t.Errorf("expected total %d, got %d", expectedTotal, total)
	}
	if active != expectedActive {
		t.Errorf("expected active %d, got %d", expectedActive, active)
	}
}

// --- ClearWinner / ChangeWinner test helpers ---

const winnerTestSlugPrefix = "test-winner-"

// setupCompletedWaffle creates a 5-spot waffle, claims all spots, marks them paid,
// sets spot 3 as winner, and returns the waffle plus all its spots.
func setupCompletedWaffle(t *testing.T) (waffle *models.Waffle, spots []models.Spot) {
	t.Helper()

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            "ClearWinner Test Waffle",
		TotalSpots:       5,
		SpotPrice:        10,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	// Claim spots 1-3 for buyer_a, spots 4-5 for buyer_b
	handles := []string{"buyer_a", "buyer_b"}
	for i, handle := range handles {
		start := i*3 + 1
		end := start + 2
		if i == 1 {
			end = 5 // second buyer gets spots 4-5
		}
		spotNums := make([]int, 0)
		for s := start; s <= end; s++ {
			spotNums = append(spotNums, s)
		}
		if err := ClaimSpots(waffle.ID, spotNums, handle); err != nil {
			t.Fatalf("ClaimSpots for %s: %v", handle, err)
		}
	}

	// Mark all spots paid
	spots, err = GetSpotsByWaffleID(waffle.ID)
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

	// Set spot 3 as winner
	if err := SetWinner(waffle.ID, []int{3}); err != nil {
		t.Fatalf("SetWinner: %v", err)
	}

	// Reload waffle and spots
	waffle, err = GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID: %v", err)
	}
	spots, err = GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}

	return waffle, spots
}

// setupActiveWaffle creates a 3-spot active waffle with one spot claimed (pending).
func setupActiveWaffle(t *testing.T) *models.Waffle {
	t.Helper()

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            "Active Test Waffle",
		TotalSpots:       3,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	// Claim spot 1 so there's some activity
	if err := ClaimSpots(waffle.ID, []int{1}, "test_buyer"); err != nil {
		t.Fatalf("ClaimSpots: %v", err)
	}

	waffle, err = GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID: %v", err)
	}

	return waffle
}

func cleanupWinnerTestWaffles(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM waffles WHERE slug LIKE $1`, winnerTestSlugPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup winner test waffles: %v", err)
	}
}

func TestClearWinner_Success(t *testing.T) {
	defer cleanupWinnerTestWaffles(t)

	waffle, spots := setupCompletedWaffle(t)

	// Verify preconditions
	if waffle.Status != models.WaffleStatusCompleted {
		t.Fatalf("expected waffle to be completed, got %s", waffle.Status)
	}
	if waffle.WinningSpotNumber == nil || *waffle.WinningSpotNumber != 3 {
		t.Fatalf("expected winning spot to be 3, got %v", waffle.WinningSpotNumber)
	}

	hasWinner := false
	hasLoser := false
	for _, s := range spots {
		if s.Status == models.SpotStatusWinner {
			hasWinner = true
		}
		if s.Status == models.SpotStatusLoser {
			hasLoser = true
		}
	}
	if !hasWinner || !hasLoser {
		t.Fatal("expected at least one winner and one loser spot before clearing")
	}

	// Clear the winner
	if err := ClearWinner(waffle.ID); err != nil {
		t.Fatalf("ClearWinner: %v", err)
	}

	// Reload and verify
	waffle, err := GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID after clear: %v", err)
	}
	if waffle.Status != models.WaffleStatusActive {
		t.Errorf("expected waffle status active, got %s", waffle.Status)
	}
	if waffle.WinningSpotNumber != nil {
		t.Errorf("expected winning_spot_number to be nil, got %v", waffle.WinningSpotNumber)
	}
	if waffle.WinningInstagramHandle != nil {
		t.Errorf("expected winning_instagram_handle to be nil, got %v", waffle.WinningInstagramHandle)
	}
	if waffle.CompletedAt != nil {
		t.Errorf("expected completed_at to be nil, got %v", waffle.CompletedAt)
	}

	// Verify all spots are back to paid
	spots, err = GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID after clear: %v", err)
	}
	for _, s := range spots {
		if s.Status != models.SpotStatusPaid {
			t.Errorf("spot %d: expected status paid, got %s", s.Number, s.Status)
		}
	}
}

func TestClearWinner_FailsOnActiveWaffle(t *testing.T) {
	defer cleanupWinnerTestWaffles(t)

	waffle := setupActiveWaffle(t)

	err := ClearWinner(waffle.ID)
	if err == nil {
		t.Fatal("expected error clearing winner on active waffle, got nil")
	}
}

func TestChangeWinner_Success(t *testing.T) {
	defer cleanupWinnerTestWaffles(t)

	waffle, _ := setupCompletedWaffle(t)

	// Verify preconditions: spot 3 is winner, others are losers
	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}
	for _, s := range spots {
		if s.Number == 3 && s.Status != models.SpotStatusWinner {
			t.Fatalf("expected spot 3 to be winner, got %s", s.Status)
		}
	}

	// Change winner to spot 5 (which should now be loser)
	if err := ChangeWinner(waffle.ID, []int{5}); err != nil {
		t.Fatalf("ChangeWinner: %v", err)
	}

	// Reload spots and verify
	spots, err = GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID after change: %v", err)
	}

	var oldSpotStatus, newSpotStatus models.SpotStatus
	for _, s := range spots {
		switch s.Number {
		case 3:
			oldSpotStatus = s.Status
		case 5:
			newSpotStatus = s.Status
		}
	}

	if oldSpotStatus != models.SpotStatusLoser {
		t.Errorf("old winner spot 3: expected loser, got %s", oldSpotStatus)
	}
	if newSpotStatus != models.SpotStatusWinner {
		t.Errorf("new winner spot 5: expected winner, got %s", newSpotStatus)
	}

	// Verify waffle records updated
	waffle, err = GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID after change: %v", err)
	}
	if waffle.WinningSpotNumber == nil || *waffle.WinningSpotNumber != 5 {
		t.Errorf("expected winning_spot_number 5, got %v", waffle.WinningSpotNumber)
	}
	if waffle.WinningInstagramHandle == nil || *waffle.WinningInstagramHandle != "buyer_b" {
		t.Errorf("expected winning handle buyer_b, got %v", waffle.WinningInstagramHandle)
	}

	// Verity waffle stays completed
	if waffle.Status != models.WaffleStatusCompleted {
		t.Errorf("expected waffle status completed, got %s", waffle.Status)
	}
}

func TestChangeWinner_FailsOnActiveWaffle(t *testing.T) {
	defer cleanupWinnerTestWaffles(t)

	waffle := setupActiveWaffle(t)

	err := ChangeWinner(waffle.ID, []int{1})
	if err == nil {
		t.Fatal("expected error changing winner on active waffle, got nil")
	}
}

func TestChangeWinner_NewSpotNotAvailable(t *testing.T) {
	defer cleanupWinnerTestWaffles(t)

	waffle, _ := setupCompletedWaffle(t)

	// Spot 99 does not exist
	err := ChangeWinner(waffle.ID, []int{99})
	if err == nil {
		t.Fatal("expected error for non-existent spot, got nil")
	}
}

func TestSetWinner_MultipleItems_Success(t *testing.T) {
	defer cleanupWinnerTestWaffles(t)

	// Create waffle with 3 items
	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            winnerTestSlugPrefix + "multi-item",
		TotalSpots:       6,
		SpotPrice:        10,
		ItemCount:        3,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	// Claim and pay all spots
	for i := 1; i <= 6; i++ {
		handle := fmt.Sprintf("buyer_%d", i)
		if err := ClaimSpots(waffle.ID, []int{i}, handle); err != nil {
			t.Fatalf("ClaimSpot %d: %v", i, err)
		}
		spots, err := GetSpotsByWaffleID(waffle.ID)
		if err != nil {
			t.Fatalf("GetSpots: %v", err)
		}
		for _, s := range spots {
			if s.Number == i {
				if err := MarkSpotPaid(s.ID); err != nil {
					t.Fatalf("MarkPaid %d: %v", i, err)
				}
			}
		}
	}

	// Set 3 winners: spot 2, spot 4, spot 5
	winningSpots := []int{2, 4, 5}
	if err := SetWinner(waffle.ID, winningSpots); err != nil {
		t.Fatalf("SetWinner for multiple items: %v", err)
	}

	// Reload waffle and verify winners
	waffle, err = GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID: %v", err)
	}

	if waffle.Status != models.WaffleStatusCompleted {
		t.Errorf("expected status completed, got %s", waffle.Status)
	}
	if len(waffle.WinningSpotNumbers) != 3 {
		t.Errorf("expected 3 winning spots, got %d", len(waffle.WinningSpotNumbers))
	}
	for i, expectedSpot := range winningSpots {
		if waffle.WinningSpotNumbers[i] != expectedSpot {
			t.Errorf("expected winner %d to be spot %d, got %d", i+1, expectedSpot, waffle.WinningSpotNumbers[i])
		}
	}

	// Verify spots status: 2, 4, 5 should be winner, others loser
	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpots: %v", err)
	}
	for _, s := range spots {
		isWinner := false
		for _, w := range winningSpots {
			if s.Number == w {
				isWinner = true
			}
		}
		if isWinner {
			if s.Status != models.SpotStatusWinner {
				t.Errorf("spot %d: expected winner status, got %s", s.Number, s.Status)
			}
		} else {
			if s.Status != models.SpotStatusLoser {
				t.Errorf("spot %d: expected loser status, got %s", s.Number, s.Status)
			}
		}
	}
}

func TestChangeWinner_MultipleItems_Success(t *testing.T) {
	defer cleanupWinnerTestWaffles(t)

	// Create waffle with 2 items
	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            winnerTestSlugPrefix + "change-multi",
		TotalSpots:       4,
		SpotPrice:        10,
		ItemCount:        2,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	// Claim and pay all spots
	for i := 1; i <= 4; i++ {
		handle := fmt.Sprintf("buyer_%d", i)
		if err := ClaimSpots(waffle.ID, []int{i}, handle); err != nil {
			t.Fatalf("ClaimSpot %d: %v", i, err)
		}
		spots, err := GetSpotsByWaffleID(waffle.ID)
		if err != nil {
			t.Fatalf("GetSpots: %v", err)
		}
		for _, s := range spots {
			if s.Number == i {
				if err := MarkSpotPaid(s.ID); err != nil {
					t.Fatalf("MarkPaid %d: %v", i, err)
				}
			}
		}
	}

	// Set initial winners: spot 1 and 3
	if err := SetWinner(waffle.ID, []int{1, 3}); err != nil {
		t.Fatalf("SetWinner: %v", err)
	}

	// Change winners to spot 2 and 4
	if err := ChangeWinner(waffle.ID, []int{2, 4}); err != nil {
		t.Fatalf("ChangeWinner: %v", err)
	}

	// Reload waffle and verify new winners
	waffle, err = GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID: %v", err)
	}
	if len(waffle.WinningSpotNumbers) != 2 || waffle.WinningSpotNumbers[0] != 2 || waffle.WinningSpotNumbers[1] != 4 {
		t.Errorf("expected winning spots [2, 4], got %v", waffle.WinningSpotNumbers)
	}

	// Verify spot status
	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpots: %v", err)
	}
	for _, s := range spots {
		if s.Number == 2 || s.Number == 4 {
			if s.Status != models.SpotStatusWinner {
				t.Errorf("spot %d: expected winner, got %s", s.Number, s.Status)
			}
		} else {
			if s.Status != models.SpotStatusLoser {
				t.Errorf("spot %d: expected loser, got %s", s.Number, s.Status)
			}
		}
	}
}

// --- ClaimRandomSpots tests ---

const randomTestSlugPrefix = "test-random-"

func cleanupRandomTestWaffles(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM waffles WHERE slug LIKE $1`, randomTestSlugPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup random test waffles: %v", err)
	}
}

func TestClaimRandomSpots_HappyPath(t *testing.T) {
	defer cleanupRandomTestWaffles(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            randomTestSlugPrefix + "happy",
		TotalSpots:       10,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	claimed, err := ClaimRandomSpots(waffle.ID, 3, "testuser")
	if err != nil {
		t.Fatalf("ClaimRandomSpots: %v", err)
	}
	if len(claimed) != 3 {
		t.Fatalf("expected 3 claimed spots, got %d", len(claimed))
	}

	// Verify uniqueness
	seen := make(map[int]bool)
	for _, n := range claimed {
		if seen[n] {
			t.Fatalf("duplicate spot number %d", n)
		}
		seen[n] = true
		if n < 1 || n > 10 {
			t.Fatalf("spot number %d out of range", n)
		}
	}

	// Verify DB state
	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}

	pendingCount := 0
	for _, s := range spots {
		if s.Status == models.SpotStatusPending {
			pendingCount++
			if s.ClaimedByHandle == nil || *s.ClaimedByHandle != "testuser" {
				t.Fatalf("spot %d: expected claimed_by_handle 'testuser', got %v", s.Number, s.ClaimedByHandle)
			}
			if !seen[s.Number] {
				t.Fatalf("spot %d: pending but not in returned list %v", s.Number, claimed)
			}
		}
	}
	if pendingCount != 3 {
		t.Fatalf("expected 3 pending spots in DB, got %d", pendingCount)
	}
}

func TestClaimRandomSpots_PartialFulfillment(t *testing.T) {
	defer cleanupRandomTestWaffles(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            randomTestSlugPrefix + "partial",
		TotalSpots:       2,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	claimed, err := ClaimRandomSpots(waffle.ID, 5, "testuser")
	if err != nil {
		t.Fatalf("ClaimRandomSpots: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed spots, got %d", len(claimed))
	}

	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}
	pendingCount := 0
	for _, s := range spots {
		if s.Status == models.SpotStatusPending {
			pendingCount++
		}
	}
	if pendingCount != 2 {
		t.Fatalf("expected 2 pending spots in DB, got %d", pendingCount)
	}
}

func TestClaimRandomSpots_ZeroAvailability(t *testing.T) {
	defer cleanupRandomTestWaffles(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            randomTestSlugPrefix + "zero",
		TotalSpots:       3,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	// Claim all spots manually first
	if err := ClaimSpots(waffle.ID, []int{1, 2, 3}, "firstuser"); err != nil {
		t.Fatalf("ClaimSpots: %v", err)
	}

	// Now try to claim random spots — none available
	claimed, err := ClaimRandomSpots(waffle.ID, 3, "seconduser")
	if err != nil {
		t.Fatalf("ClaimRandomSpots should not error when no spots available: %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("expected empty slice, got %v", claimed)
	}
}

func TestClaimRandomSpots_InvalidCountZero(t *testing.T) {
	defer cleanupRandomTestWaffles(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            randomTestSlugPrefix + "zero-count",
		TotalSpots:       5,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	_, err = ClaimRandomSpots(waffle.ID, 0, "testuser")
	if err == nil {
		t.Fatal("expected error for count=0, got nil")
	}
}

func TestClaimRandomSpots_InvalidCountNegative(t *testing.T) {
	defer cleanupRandomTestWaffles(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            randomTestSlugPrefix + "neg-count",
		TotalSpots:       5,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	_, err = ClaimRandomSpots(waffle.ID, -1, "testuser")
	if err == nil {
		t.Fatal("expected error for count=-1, got nil")
	}
}

func TestClaimRandomSpots_InactiveWaffle(t *testing.T) {
	defer cleanupRandomTestWaffles(t)
	defer cleanupWinnerTestWaffles(t)

	// Use a completed waffle (inactive)
	waffle, _ := setupCompletedWaffle(t)

	_, err := ClaimRandomSpots(waffle.ID, 1, "testuser")
	if err == nil {
		t.Fatal("expected error for inactive waffle, got nil")
	}
}

func TestClaimRandomSpots_Concurrent(t *testing.T) {
	defer cleanupRandomTestWaffles(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            randomTestSlugPrefix + "concurrent",
		TotalSpots:       7,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	type result struct {
		spots []int
		err   error
	}

	results := make(chan result, 2)
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			spots, err := ClaimRandomSpots(waffle.ID, 5, fmt.Sprintf("user%d", id))
			results <- result{spots: spots, err: err}
		}(i)
	}

	wg.Wait()
	close(results)

	allClaimed := make(map[int]bool)
	totalClaimed := 0
	for res := range results {
		if res.err != nil {
			t.Fatalf("unexpected error from goroutine: %v", res.err)
		}
		for _, n := range res.spots {
			if allClaimed[n] {
				t.Fatalf("duplicate spot %d across goroutines", n)
			}
			allClaimed[n] = true
		}
		totalClaimed += len(res.spots)
	}

	if totalClaimed != 7 {
		t.Fatalf("expected total claimed = 7, got %d", totalClaimed)
	}

	// Verify DB state
	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}
	pendingCount := 0
	for _, s := range spots {
		if s.Status == models.SpotStatusPending {
			pendingCount++
		}
	}
	if pendingCount != 7 {
		t.Fatalf("expected 7 pending spots in DB, got %d", pendingCount)
	}
}

// --- ArchiveWaffle tests ---

func TestArchiveWaffle_Archives(t *testing.T) {
	defer cleanupTestWaffles(t)

	waffleID := insertTestWaffle(t, models.WaffleStatusActive, false)

	if err := ArchiveWaffle(waffleID, true); err != nil {
		t.Fatalf("ArchiveWaffle: %v", err)
	}

	var archived bool
	err := db.Pool.QueryRow(context.Background(),
		"SELECT archived FROM waffles WHERE id=$1", waffleID).Scan(&archived)
	if err != nil {
		t.Fatalf("query archived: %v", err)
	}
	if !archived {
		t.Errorf("expected archived=true, got false")
	}
}

func TestArchiveWaffle_Unarchives(t *testing.T) {
	defer cleanupTestWaffles(t)

	waffleID := insertTestWaffle(t, models.WaffleStatusActive, true)

	if err := ArchiveWaffle(waffleID, false); err != nil {
		t.Fatalf("ArchiveWaffle: %v", err)
	}

	var archived bool
	err := db.Pool.QueryRow(context.Background(),
		"SELECT archived FROM waffles WHERE id=$1", waffleID).Scan(&archived)
	if err != nil {
		t.Fatalf("query archived: %v", err)
	}
	if archived {
		t.Errorf("expected archived=false, got true")
	}
}

func TestArchiveWaffle_AlreadyArchived(t *testing.T) {
	defer cleanupTestWaffles(t)

	waffleID := insertTestWaffle(t, models.WaffleStatusActive, true)

	if err := ArchiveWaffle(waffleID, true); err != nil {
		t.Fatalf("ArchiveWaffle: %v", err)
	}

	var archived bool
	err := db.Pool.QueryRow(context.Background(),
		"SELECT archived FROM waffles WHERE id=$1", waffleID).Scan(&archived)
	if err != nil {
		t.Fatalf("query archived: %v", err)
	}
	if !archived {
		t.Errorf("expected archived=true after idempotent archive, got false")
	}
}

func TestArchiveWaffle_AlreadyUnarchived(t *testing.T) {
	defer cleanupTestWaffles(t)

	waffleID := insertTestWaffle(t, models.WaffleStatusActive, false)

	if err := ArchiveWaffle(waffleID, false); err != nil {
		t.Fatalf("ArchiveWaffle: %v", err)
	}

	var archived bool
	err := db.Pool.QueryRow(context.Background(),
		"SELECT archived FROM waffles WHERE id=$1", waffleID).Scan(&archived)
	if err != nil {
		t.Fatalf("query archived: %v", err)
	}
	if archived {
		t.Errorf("expected archived=false after idempotent unarchive, got true")
	}
}

func TestClaimRandomSpots_HandleNormalization(t *testing.T) {
	defer cleanupRandomTestWaffles(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            randomTestSlugPrefix + "normalize",
		TotalSpots:       5,
		SpotPrice:        5,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	claimed, err := ClaimRandomSpots(waffle.ID, 1, "@TestUser")
	if err != nil {
		t.Fatalf("ClaimRandomSpots: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("expected 1 claimed spot, got %d", len(claimed))
	}

	spots, err := GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("GetSpotsByWaffleID: %v", err)
	}
	for _, s := range spots {
		if s.Number == claimed[0] {
			if s.ClaimedByHandle == nil || *s.ClaimedByHandle != "testuser" {
				t.Fatalf("expected normalized handle 'testuser', got %v", s.ClaimedByHandle)
			}
			return
		}
	}
	t.Fatalf("claimed spot %d not found in DB", claimed[0])
}

// --- Share template tests ---

const shareTmplTestSlugPrefix = "test-share-tmpl-"

func cleanupShareTmplTestWaffles(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(),
		`DELETE FROM waffles WHERE slug LIKE $1`, shareTmplTestSlugPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup share template test waffles: %v", err)
	}
}

func TestCreateWaffle_ShareTemplatePopulated(t *testing.T) {
	defer cleanupShareTmplTestWaffles(t)

	// Ensure a default template exists for this test.
	adminID := uuid.New()
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, adminID, "test-st-"+adminID.String()[:8], "test-st-" + adminID.String() + "@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test admin: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, adminID)
	}()

	_, err = GetDefaultMessageTemplate()
	if err != nil {
		tmpl, err := CreateMessageTemplate("Test Default", "Test body for {item}", adminID)
		if err != nil {
			t.Fatalf("CreateMessageTemplate: %v", err)
		}
		if err := SetDefaultMessageTemplate(tmpl.ID); err != nil {
			t.Fatalf("SetDefaultMessageTemplate: %v", err)
		}
	}

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            shareTmplTestSlugPrefix + "populated",
		TotalSpots:       5,
		SpotPrice:        10,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	if waffle.ShareTemplateID == nil {
		t.Fatal("expected share_template_id to be populated from default template, got nil")
	}

	if waffle.ShareMessage == nil || *waffle.ShareMessage == "" {
		t.Fatal("expected share_message to be non-empty after creation")
	}
}

func TestCreateWaffle_ShareMessageContainsTitle(t *testing.T) {
	defer cleanupShareTmplTestWaffles(t)

	// Ensure a default template exists for this test.
	adminID := uuid.New()
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, adminID, "test-st-"+adminID.String()[:8], "test-st-" + adminID.String() + "@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test admin: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, adminID)
	}()

	_, err = GetDefaultMessageTemplate()
	if err != nil {
		tmpl, err := CreateMessageTemplate("Test Default", "Test body for {item}", adminID)
		if err != nil {
			t.Fatalf("CreateMessageTemplate: %v", err)
		}
		if err := SetDefaultMessageTemplate(tmpl.ID); err != nil {
			t.Fatalf("SetDefaultMessageTemplate: %v", err)
		}
	}

	title := shareTmplTestSlugPrefix + "message-content"
	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            title,
		TotalSpots:       5,
		SpotPrice:        10,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	if waffle.ShareMessage == nil {
		t.Fatal("expected share_message to be set")
	}

	if !strings.Contains(*waffle.ShareMessage, title) {
		t.Errorf("expected share_message to contain waffle title %q, got %q", title, *waffle.ShareMessage)
	}
}

func TestSetWaffleShareTemplate_RerendersMessage(t *testing.T) {
	defer cleanupShareTmplTestWaffles(t)

	// Ensure a default template exists for this test.
	adminID := uuid.New()
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, adminID, "test-st-"+adminID.String()[:8], "test-st-" + adminID.String() + "@example.com", "test-hash", "admin", true, "UTC", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create test admin: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM admins WHERE id = $1`, adminID)
	}()

	_, err = GetDefaultMessageTemplate()
	if err != nil {
		tmpl, err := CreateMessageTemplate("Test Default", "Test body for {item}", adminID)
		if err != nil {
			t.Fatalf("CreateMessageTemplate: %v", err)
		}
		if err := SetDefaultMessageTemplate(tmpl.ID); err != nil {
			t.Fatalf("SetDefaultMessageTemplate: %v", err)
		}
	}

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:            shareTmplTestSlugPrefix + "rerender",
		TotalSpots:       5,
		SpotPrice:        10,
		PaymentMethodIDs: nil,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	originalMessage := *waffle.ShareMessage

	tmpl, err := CreateMessageTemplate("Rerender Test", "Custom body for {item}", adminID)
	if err != nil {
		t.Fatalf("CreateMessageTemplate: %v", err)
	}
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM message_templates WHERE id = $1`, tmpl.ID)
	}()

	if err := SetWaffleShareTemplate(waffle.ID, tmpl.ID); err != nil {
		t.Fatalf("SetWaffleShareTemplate: %v", err)
	}

	updated, err := GetWaffleByID(waffle.ID)
	if err != nil {
		t.Fatalf("GetWaffleByID: %v", err)
	}

	if updated.ShareMessage == nil || *updated.ShareMessage == "" {
		t.Fatal("expected share_message to be non-empty after template change")
	}

	if *updated.ShareMessage == originalMessage {
		t.Error("expected share_message to change after template update, but it stayed the same")
	}

	if !strings.Contains(*updated.ShareMessage, waffle.Title) {
		t.Errorf("expected updated share_message to contain waffle title, got %q", *updated.ShareMessage)
	}
}
