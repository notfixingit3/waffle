package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

// testHookWafflePrefix identifies hook-test waffles for cleanup.
const testHookWafflePrefix = "test-task5-hook-"

// testHookHandlePrefix identifies hook-test Instagram handles for cleanup.
// Handles must match the buyer card filename charset (^[a-z0-9_.]{1,30}$) or
// invalidation is a validated no-op, so dots are used instead of dashes.
const testHookHandlePrefix = "task5."

func cleanupBuyerCardHookTests(t *testing.T) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(),
		`DELETE FROM waffles WHERE slug LIKE $1`, testHookWafflePrefix+"%"); err != nil {
		t.Fatalf("cleanup hook test waffles: %v", err)
	}
	if _, err := db.Pool.Exec(context.Background(),
		`DELETE FROM buyer_stats WHERE instagram_handle LIKE $1`, testHookHandlePrefix+"%"); err != nil {
		t.Fatalf("cleanup hook test stats: %v", err)
	}
	if _, err := db.Pool.Exec(context.Background(),
		`DELETE FROM users WHERE instagram_handle LIKE $1`, testHookHandlePrefix+"%"); err != nil {
		t.Fatalf("cleanup hook test users: %v", err)
	}
}

// useTempBuyerCardCacheDir points ShareCardCacheDir at a temp dir for the
// duration of the test.
func useTempBuyerCardCacheDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = dir
	t.Cleanup(func() { ShareCardCacheDir = originalDir })
	return dir
}

// createPaidWaffleForHookTest mirrors createCompletedWaffleForBuyer but stops
// before SetWinner so tests can seed cache files between claiming and the
// winner/delete flow under test. The waffle title is set after creation so
// the slug stays prefixed for cleanup.
func createPaidWaffleForHookTest(t *testing.T, title string, totalSpots int, claims map[string][]int) *models.Waffle {
	t.Helper()

	ensureDefaultTemplateExists(t)

	waffle, err := CreateWaffle(models.CreateWaffleRequest{
		Title:      testHookWafflePrefix + uuid.New().String()[:8],
		TotalSpots: totalSpots,
		SpotPrice:  1,
	})
	if err != nil {
		t.Fatalf("CreateWaffle: %v", err)
	}

	_, err = db.Pool.Exec(context.Background(),
		`UPDATE waffles SET title = $1 WHERE id = $2`, title, waffle.ID)
	if err != nil {
		t.Fatalf("update waffle title: %v", err)
	}

	for handle, spotNumbers := range claims {
		if err := ClaimSpots(waffle.ID, spotNumbers, handle); err != nil {
			t.Fatalf("ClaimSpots for %s: %v", handle, err)
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

	return waffle
}

// writeBuyerCardCacheFiles seeds both buyer card formats for handle and
// returns the paths written.
func writeBuyerCardCacheFiles(t *testing.T, dir, handle string) []string {
	t.Helper()
	var paths []string
	for _, format := range []string{ShareCardFormatStory, ShareCardFormatSquare} {
		path := filepath.Join(dir, fmt.Sprintf("buyer-%s-%s.png", handle, format))
		if err := os.WriteFile(path, []byte("fake-png"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return paths
}

func assertFilesRemoved(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", filepath.Base(path))
		}
	}
}

func assertFilesPresent(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to survive", filepath.Base(path))
		}
	}
}

func TestSetWinner_InvalidatesParticipantBuyerCards(t *testing.T) {
	defer cleanupBuyerCardHookTests(t)
	dir := useTempBuyerCardCacheDir(t)

	// Given: a filled waffle where alpha holds multiple spots and beta holds
	// one, with cached cards for both plus a non-participant and a waffle card.
	alpha := testHookHandlePrefix + "alpha"
	beta := testHookHandlePrefix + "beta"
	waffle := createPaidWaffleForHookTest(t, "Hook SetWinner Test", 4, map[string][]int{
		alpha: {1, 2, 3},
		beta:  {4},
	})
	alphaFiles := writeBuyerCardCacheFiles(t, dir, alpha)
	betaFiles := writeBuyerCardCacheFiles(t, dir, beta)
	gammaFiles := writeBuyerCardCacheFiles(t, dir, testHookHandlePrefix+"gamma")
	waffleCard := filepath.Join(dir, "cool-waffle-abc123-story.png")
	if err := os.WriteFile(waffleCard, []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}

	// When: the winner is set on their waffle.
	if err := SetWinner(waffle.ID, []int{1}); err != nil {
		t.Fatalf("SetWinner: %v", err)
	}

	// Then: both participants' cards are invalidated (multi-spot alpha exactly
	// covers the dedup path); non-participant and waffle files survive.
	assertFilesRemoved(t, alphaFiles...)
	assertFilesRemoved(t, betaFiles...)
	assertFilesPresent(t, gammaFiles...)
	assertFilesPresent(t, waffleCard)
}

func TestClearWinner_InvalidatesParticipantBuyerCards(t *testing.T) {
	defer cleanupBuyerCardHookTests(t)
	dir := useTempBuyerCardCacheDir(t)

	// Given: a completed waffle, with participant cards cached after the win.
	alpha := testHookHandlePrefix + "calpha"
	beta := testHookHandlePrefix + "cbeta"
	waffle := createPaidWaffleForHookTest(t, "Hook ClearWinner Test", 4, map[string][]int{
		alpha: {1, 2},
		beta:  {3, 4},
	})
	if err := SetWinner(waffle.ID, []int{1}); err != nil {
		t.Fatalf("SetWinner: %v", err)
	}
	alphaFiles := writeBuyerCardCacheFiles(t, dir, alpha)
	betaFiles := writeBuyerCardCacheFiles(t, dir, beta)

	// When: the winner is cleared.
	if err := ClearWinner(waffle.ID); err != nil {
		t.Fatalf("ClearWinner: %v", err)
	}

	// Then: both participants' cards are invalidated.
	assertFilesRemoved(t, alphaFiles...)
	assertFilesRemoved(t, betaFiles...)
}

func TestChangeWinner_InvalidatesParticipantBuyerCards(t *testing.T) {
	defer cleanupBuyerCardHookTests(t)
	dir := useTempBuyerCardCacheDir(t)

	// Given: a completed waffle, with participant cards cached after the win.
	alpha := testHookHandlePrefix + "xalpha"
	beta := testHookHandlePrefix + "xbeta"
	waffle := createPaidWaffleForHookTest(t, "Hook ChangeWinner Test", 4, map[string][]int{
		alpha: {1, 2},
		beta:  {3, 4},
	})
	if err := SetWinner(waffle.ID, []int{1}); err != nil {
		t.Fatalf("SetWinner: %v", err)
	}
	alphaFiles := writeBuyerCardCacheFiles(t, dir, alpha)
	betaFiles := writeBuyerCardCacheFiles(t, dir, beta)

	// When: the winner is changed to another spot.
	if err := ChangeWinner(waffle.ID, []int{3}); err != nil {
		t.Fatalf("ChangeWinner: %v", err)
	}

	// Then: both participants' cards are invalidated.
	assertFilesRemoved(t, alphaFiles...)
	assertFilesRemoved(t, betaFiles...)
}

func TestDeleteWaffle_InvalidatesParticipantBuyerCards(t *testing.T) {
	defer cleanupBuyerCardHookTests(t)
	dir := useTempBuyerCardCacheDir(t)

	// Given: a filled waffle with cached cards for both participants and a
	// non-participant.
	alpha := testHookHandlePrefix + "dalpha"
	beta := testHookHandlePrefix + "dbeta"
	waffle := createPaidWaffleForHookTest(t, "Hook DeleteWaffle Test", 2, map[string][]int{
		alpha: {1},
		beta:  {2},
	})
	alphaFiles := writeBuyerCardCacheFiles(t, dir, alpha)
	betaFiles := writeBuyerCardCacheFiles(t, dir, beta)
	gammaFiles := writeBuyerCardCacheFiles(t, dir, testHookHandlePrefix+"dgamma")

	// When: the waffle is deleted.
	if err := DeleteWaffle(waffle.ID); err != nil {
		t.Fatalf("DeleteWaffle: %v", err)
	}

	// Then: both participants' cards are invalidated; the non-participant's
	// survives.
	assertFilesRemoved(t, alphaFiles...)
	assertFilesRemoved(t, betaFiles...)
	assertFilesPresent(t, gammaFiles...)
}

func TestSetWinner_ToleratesMissingCacheDir(t *testing.T) {
	defer cleanupBuyerCardHookTests(t)

	// Given: a filled waffle and a cache dir that does not exist.
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = filepath.Join(t.TempDir(), "never-created")
	t.Cleanup(func() { ShareCardCacheDir = originalDir })

	alpha := testHookHandlePrefix + "ncalpha"
	waffle := createPaidWaffleForHookTest(t, "Hook No Cache Dir Test", 2, map[string][]int{
		alpha: {1},
		testHookHandlePrefix + "ncbeta": {2},
	})

	// When: the winner is set.
	// Then: the invalidation error is tolerated and the winner flow commits.
	if err := SetWinner(waffle.ID, []int{1}); err != nil {
		t.Fatalf("SetWinner with missing cache dir: %v", err)
	}
}
