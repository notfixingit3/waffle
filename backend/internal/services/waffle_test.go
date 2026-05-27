package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func TestMain(m *testing.M) {
	pool, err := db.Connect()
	if err != nil {
		// Postgres not available — skip all service tests
		os.Exit(0)
	}
	defer pool.Close()

	os.Exit(m.Run())
}

// testSlugPrefix is used to identify and clean up test waffles.
const testSlugPrefix = "test-count-waffles-"

func insertTestWaffle(t *testing.T, status models.WaffleStatus, archived bool) {
	t.Helper()

	slug := testSlugPrefix + uuid.New().String()[:8]
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO waffles (id, slug, title, total_spots, spot_price, status, archived, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.New(), slug, "Test Waffle", 10, 5, status, archived, time.Now())
	if err != nil {
		t.Fatalf("insert test waffle: %v", err)
	}
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
