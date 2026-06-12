package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func GetBuyerStats(instagramHandle string) (*models.BuyerStats, error) {
	stats := &models.BuyerStats{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT instagram_handle, total_waffles_entered, total_wins, total_losses, total_spots_claimed, updated_at
		FROM buyer_stats WHERE instagram_handle = $1
	`, instagramHandle).Scan(
		&stats.InstagramHandle, &stats.TotalWafflesEntered, &stats.TotalWins,
		&stats.TotalLosses, &stats.TotalSpotsClaimed, &stats.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get buyer stats: %w", err)
	}
	return stats, nil
}

func GetBuyerWaffleHistory(instagramHandle string) ([]models.BuyerWaffleHistory, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT 
			w.id, w.slug, w.title, w.spot_price, w.status,
			w.winning_spot_number, w.winning_instagram_handle,
			w.created_at, w.completed_at,
			w.total_spots,
			ARRAY_AGG(s.number ORDER BY s.number) as spot_numbers,
			BOOL_OR(s.status = 'winner') as is_winner
		FROM waffles w
		JOIN spots s ON s.waffle_id = w.id
		WHERE s.claimed_by_handle = $1
		GROUP BY w.id, w.slug, w.title, w.spot_price, w.status,
			w.winning_spot_number, w.winning_instagram_handle,
			w.created_at, w.completed_at, w.total_spots
		ORDER BY w.created_at DESC
	`, instagramHandle)
	if err != nil {
		return nil, fmt.Errorf("get buyer history: %w", err)
	}
	defer rows.Close()

	var history []models.BuyerWaffleHistory
	for rows.Next() {
		var h models.BuyerWaffleHistory
		var isWinner bool
		err := rows.Scan(
			&h.WaffleID, &h.Slug, &h.Title, &h.SpotPrice, &h.Status,
			&h.WinningSpotNumber, &h.WinningInstagramHandle,
			&h.CreatedAt, &h.CompletedAt,
			&h.TotalSpots,
			&h.SpotNumbers, &isWinner,
		)
		if err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		h.IsWinner = isWinner
		history = append(history, h)
	}

	return history, nil
}

func GetBuyerStatsWithRank(instagramHandle string) (*models.BuyerStatsWithRank, error) {
	stats, err := GetBuyerStats(instagramHandle)
	if err != nil {
		return nil, err
	}

	var rank int
	err = db.Pool.QueryRow(context.Background(), `
		SELECT rank FROM (
			SELECT instagram_handle, ROW_NUMBER() OVER (ORDER BY total_wins DESC, total_spots_claimed DESC) as rank
			FROM buyer_stats
		) ranked WHERE instagram_handle = $1
	`, instagramHandle).Scan(&rank)
	if err != nil {
		rank = 0
	}

	return &models.BuyerStatsWithRank{
		BuyerStats: *stats,
		Rank:       rank,
	}, nil
}

func ListAllBuyers(limit int) ([]models.BuyerStats, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.Pool.Query(context.Background(), `
		SELECT instagram_handle, total_waffles_entered, total_wins, total_losses, total_spots_claimed, updated_at
		FROM buyer_stats
		ORDER BY total_wins DESC, total_spots_claimed DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list buyers: %w", err)
	}
	defer rows.Close()

	var buyers []models.BuyerStats
	for rows.Next() {
		var b models.BuyerStats
		err := rows.Scan(
			&b.InstagramHandle, &b.TotalWafflesEntered, &b.TotalWins,
			&b.TotalLosses, &b.TotalSpotsClaimed, &b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan buyer: %w", err)
		}
		buyers = append(buyers, b)
	}

	return buyers, nil
}

func RecordActivityEvent(waffleID string, eventType, message, instagramHandle string, spotNumbers []int) error {
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO activity_events (waffle_id, event_type, message, instagram_handle, spot_numbers)
		VALUES ($1, $2, $3, $4, $5)
	`, waffleID, eventType, message, instagramHandle, spotNumbers)
	if err != nil {
		return fmt.Errorf("record activity: %w", err)
	}
	return nil
}

func GetActivityEvents(waffleID string, limit int) ([]models.ActivityEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Pool.Query(context.Background(), `
		SELECT id, waffle_id, event_type, message, instagram_handle, spot_numbers, created_at
		FROM activity_events
		WHERE waffle_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, waffleID, limit)
	if err != nil {
		return nil, fmt.Errorf("get activity: %w", err)
	}
	defer rows.Close()

	var events []models.ActivityEvent
	for rows.Next() {
		var e models.ActivityEvent
		err := rows.Scan(
			&e.ID, &e.WaffleID, &e.EventType, &e.Message,
			&e.InstagramHandle, &e.SpotNumbers, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

// BuyerCardData holds computed data for the buyer card / buyer stats page.
type BuyerCardData struct {
	InstagramHandle  string              `json:"instagram_handle"`
	Stats            *models.BuyerStats  `json:"stats,omitempty"`
	WinRate          float64             `json:"win_rate"`
	ExpectedWinRate  float64             `json:"expected_win_rate"`
	LuckRating       float64             `json:"luck_rating"`
	Trophies         []string            `json:"trophies"`
	WaffleHistory    []models.BuyerWaffleHistory `json:"waffle_history"`
}

// ComputeBuyerCardData assembles all buyer card data: stats, luck rating, and trophies.
func ComputeBuyerCardData(handle string) (*BuyerCardData, error) {
	handle = NormalizeInstagramHandle(handle)

	card := &BuyerCardData{
		InstagramHandle: handle,
		Trophies:        []string{},
		WaffleHistory:   []models.BuyerWaffleHistory{},
	}

	stats, err := GetBuyerStats(handle)
	if err != nil {
		return card, nil
	}
	card.Stats = stats

	history, err := GetBuyerWaffleHistory(handle)
	if err != nil {
		return nil, fmt.Errorf("compute buyer card: %w", err)
	}
	card.WaffleHistory = history

	// WinRate is the buyer's overall win rate across every spot they have ever
	// claimed (TotalWins / TotalSpotsClaimed). This intentionally spans all
	// waffles, not just completed ones, per the plan's buyer-card formula.
	if stats.TotalSpotsClaimed > 0 {
		card.WinRate = float64(stats.TotalWins) / float64(stats.TotalSpotsClaimed)
	}

	// ExpectedWinRate and LuckRating are computed over completed waffles only,
	// using the buyer's share of spots in completed waffles versus the total
	// spots available in those completed waffles (guard against zero total).
	var buyerSpotsInCompleted, totalSpotsInCompleted int
	for _, h := range history {
		if h.Status == "completed" {
			buyerSpotsInCompleted += len(h.SpotNumbers)
			totalSpotsInCompleted += h.TotalSpots
		}
	}
	if totalSpotsInCompleted > 0 {
		card.ExpectedWinRate = float64(buyerSpotsInCompleted) / float64(totalSpotsInCompleted)
	}

	card.LuckRating = card.WinRate - card.ExpectedWinRate

	seen := make(map[string]bool)
	for _, h := range history {
		if h.Status == "completed" && h.IsWinner {
			for _, item := range parseTrophyItems(h.Title) {
				if !seen[item] {
					seen[item] = true
					card.Trophies = append(card.Trophies, item)
				}
			}
		}
	}

	return card, nil
}

// parseTrophyItems extracts item names from inside balanced double quotes in a title.
// Returns items in order of appearance. Returns empty slice if quotes are unmatched.
func parseTrophyItems(title string) []string {
	var items []string
	i := 0
	for i < len(title) {
		openIdx := strings.IndexByte(title[i:], '"')
		if openIdx == -1 {
			break
		}
		openIdx += i

		rest := title[openIdx+1:]
		closeIdx := strings.IndexByte(rest, '"')
		if closeIdx == -1 {
			return nil
		}
		closeIdx += openIdx + 1

		item := title[openIdx+1 : closeIdx]
		if item != "" {
			items = append(items, item)
		}

		i = closeIdx + 1
	}
	return items
}
