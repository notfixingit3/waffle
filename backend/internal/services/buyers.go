package services

import (
	"context"
	"fmt"

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
			ARRAY_AGG(s.number ORDER BY s.number) as spot_numbers,
			BOOL_OR(s.status = 'winner') as is_winner
		FROM waffles w
		JOIN spots s ON s.waffle_id = w.id
		WHERE s.claimed_by_handle = $1
		GROUP BY w.id, w.slug, w.title, w.spot_price, w.status,
			w.winning_spot_number, w.winning_instagram_handle,
			w.created_at, w.completed_at
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
