package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

const (
	testReportSlugPrefix   = "test-report-waffle-"
	testReportHandlePrefix = "test-report-buyer-"
)

func cleanupReportTestData(t *testing.T) {
	t.Helper()
	// Clean up spots
	_, _ = db.Pool.Exec(context.Background(), `
		DELETE FROM spots WHERE waffle_id IN (
			SELECT id FROM waffles WHERE slug LIKE $1 || '%'
		)
	`, testReportSlugPrefix)
	// Clean up waffles
	_, _ = db.Pool.Exec(context.Background(), `
		DELETE FROM waffles WHERE slug LIKE $1 || '%'
	`, testReportSlugPrefix)
	// Clean up buyer stats
	_, _ = db.Pool.Exec(context.Background(), `
		DELETE FROM buyer_stats WHERE instagram_handle LIKE $1 || '%'
	`, testReportHandlePrefix)
}

func clearAllWafflesAndSpots(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `TRUNCATE TABLE waffles CASCADE`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}

func insertReportWaffle(t *testing.T, slugSuffix string, status models.WaffleStatus, price int, totalSpots int, createdAt, completedAt *time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	slug := testReportSlugPrefix + slugSuffix + "-" + id.String()[:8]
	
	query := `
		INSERT INTO waffles (id, slug, title, total_spots, spot_price, status, archived, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8)
	`
	_, err := db.Pool.Exec(context.Background(), query, id, slug, "Test Report Waffle", totalSpots, price, status, createdAt, completedAt)
	if err != nil {
		t.Fatalf("insertReportWaffle: %v", err)
	}
	return id
}

func insertReportSpot(t *testing.T, waffleID uuid.UUID, number int, status models.SpotStatus, buyer string, claimedAt, paidAt *time.Time) {
	t.Helper()
	id := uuid.New()
	query := `
		INSERT INTO spots (id, waffle_id, number, status, claimed_by_handle, claimed_at, paid_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Pool.Exec(context.Background(), query, id, waffleID, number, status, buyer, claimedAt, paidAt)
	if err != nil {
		t.Fatalf("insertReportSpot: %v", err)
	}
}

func TestGetDroughtList(t *testing.T) {
	if db.Pool == nil {
		t.Skip("Postgres not available")
	}
	clearAllWafflesAndSpots(t)
	defer cleanupReportTestData(t)

	now := time.Now()
	waffleCreated := now.Add(-10 * 24 * time.Hour)
	waffleCompleted := now.Add(-9 * 24 * time.Hour)
	waffleID := insertReportWaffle(t, "drought-w1", models.WaffleStatusCompleted, 10, 10, &waffleCreated, &waffleCompleted)

	buyer := testReportHandlePrefix + "drought"
	// Claim spot as paid/loser
	claimedAt := waffleCreated.Add(1 * time.Hour)
	paidAt := waffleCreated.Add(2 * time.Hour)
	insertReportSpot(t, waffleID, 1, models.SpotStatusLoser, buyer, &claimedAt, &paidAt)

	// Another spot marked as winner for someone else
	insertReportSpot(t, waffleID, 2, models.SpotStatusWinner, testReportHandlePrefix+"other", &claimedAt, &paidAt)

	// Drought range covers the claim time
	from := now.Add(-15 * 24 * time.Hour)
	to := now.Add(1 * 24 * time.Hour)

	entries, err := GetDroughtList(from, to)
	if err != nil {
		t.Fatalf("GetDroughtList failed: %v", err)
	}

	var found bool
	for _, e := range entries {
		if e.InstagramHandle == buyer {
			found = true
			if e.TotalEntries != 1 {
				t.Errorf("expected 1 entry, got %d", e.TotalEntries)
			}
			if e.LongestDrought != 99999 { // Never won
				t.Errorf("expected drought of 99999 (never won), got %d", e.LongestDrought)
			}
		}
	}
	if !found {
		t.Errorf("expected to find drought entry for %s", buyer)
	}
}

func TestGetPowerBuyers(t *testing.T) {
	if db.Pool == nil {
		t.Skip("Postgres not available")
	}
	clearAllWafflesAndSpots(t)
	defer cleanupReportTestData(t)

	now := time.Now()
	waffleCreated := now.Add(-5 * 24 * time.Hour)
	waffleID := insertReportWaffle(t, "pb-w1", models.WaffleStatusActive, 1000, 10, &waffleCreated, nil) // $10 per spot

	buyer := testReportHandlePrefix + "power"
	claimedAt := waffleCreated.Add(1 * time.Hour)
	paidAt := waffleCreated.Add(2 * time.Hour)
	// Claim 3 spots (marked paid)
	insertReportSpot(t, waffleID, 1, models.SpotStatusPaid, buyer, &claimedAt, &paidAt)
	insertReportSpot(t, waffleID, 2, models.SpotStatusPaid, buyer, &claimedAt, &paidAt)
	insertReportSpot(t, waffleID, 3, models.SpotStatusPaid, buyer, &claimedAt, &paidAt)

	from := now.Add(-10 * 24 * time.Hour)
	to := now.Add(1 * 24 * time.Hour)

	entries, err := GetPowerBuyers(from, to, 10)
	if err != nil {
		t.Fatalf("GetPowerBuyers failed: %v", err)
	}

	var found bool
	for _, e := range entries {
		if e.InstagramHandle == buyer {
			found = true
			if e.TotalSpots != 3 {
				t.Errorf("expected 3 spots, got %d", e.TotalSpots)
			}
			if e.TotalSpent != 3000 { // 3 * 1000 cents
				t.Errorf("expected 3000 cents spent, got %d", e.TotalSpent)
			}
		}
	}
	if !found {
		t.Errorf("expected to find power buyer entry for %s", buyer)
	}
}

func TestGetMonthlyActivity(t *testing.T) {
	if db.Pool == nil {
		t.Skip("Postgres not available")
	}
	clearAllWafflesAndSpots(t)
	defer cleanupReportTestData(t)

	now := time.Now()
	// Create waffle in current month
	waffleCreated := now.Add(-5 * 24 * time.Hour)
	waffleID := insertReportWaffle(t, "ma-w1", models.WaffleStatusActive, 500, 5, &waffleCreated, nil)

	buyer := testReportHandlePrefix + "ma"
	claimedAt := waffleCreated.Add(1 * time.Hour)
	paidAt := waffleCreated.Add(2 * time.Hour)
	insertReportSpot(t, waffleID, 1, models.SpotStatusPaid, buyer, &claimedAt, &paidAt)

	from := now.Add(-30 * 24 * time.Hour)
	to := now.Add(1 * 24 * time.Hour)

	entries, err := GetMonthlyActivity(from, to)
	if err != nil {
		t.Fatalf("GetMonthlyActivity failed: %v", err)
	}

	targetMonthStr := waffleCreated.Format("2006-01")
	var found bool
	for _, e := range entries {
		if e.Month == targetMonthStr {
			found = true
			if e.Waffles < 1 {
				t.Errorf("expected at least 1 waffle in %s, got %d", targetMonthStr, e.Waffles)
			}
			if e.SpotsClaimed < 1 {
				t.Errorf("expected at least 1 spot claimed, got %d", e.SpotsClaimed)
			}
			if e.Revenue < 500 {
				t.Errorf("expected at least 500 revenue cents, got %d", e.Revenue)
			}
		}
	}
	if !found {
		t.Errorf("expected activity entry for month %s", targetMonthStr)
	}
}

func TestGetSpotVelocity(t *testing.T) {
	if db.Pool == nil {
		t.Skip("Postgres not available")
	}
	clearAllWafflesAndSpots(t)
	defer cleanupReportTestData(t)

	now := time.Now()
	// Waffle created 4 hours ago, completed 2 hours ago
	waffleCreated := now.Add(-4 * time.Hour)
	waffleCompleted := now.Add(-2 * time.Hour)
	waffleID := insertReportWaffle(t, "vel-w1", models.WaffleStatusCompleted, 500, 2, &waffleCreated, &waffleCompleted)

	buyer := testReportHandlePrefix + "vel"
	// First claim was 1 hour after creation (3 hours ago)
	firstClaim := waffleCreated.Add(1 * time.Hour)
	paidAt := firstClaim.Add(15 * time.Minute)
	insertReportSpot(t, waffleID, 1, models.SpotStatusWinner, buyer, &firstClaim, &paidAt)

	entries, err := GetSpotVelocity("completed")
	if err != nil {
		t.Fatalf("GetSpotVelocity failed: %v", err)
	}

	var found bool
	for _, e := range entries {
		if e.Status == "completed" {
			found = true
			// Avg first claim should be 1.0 hours
			if e.AvgFirstClaimHours != 1.0 {
				t.Errorf("expected avg first claim hours = 1.0, got %f", e.AvgFirstClaimHours)
			}
			// Avg completion should be 2.0 hours (completed - created)
			if e.AvgCompletionHours != 2.0 {
				t.Errorf("expected avg completion hours = 2.0, got %f", e.AvgCompletionHours)
			}
		}
	}
	if !found {
		t.Errorf("expected completed velocity entry")
	}
}
