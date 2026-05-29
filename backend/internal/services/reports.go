package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func GetDroughtList(from, to time.Time) ([]models.DroughtEntry, error) {
	rows, err := db.Pool.Query(context.Background(), `
		WITH user_wins AS (
			SELECT s2.claimed_by_handle, MAX(w2.completed_at) AS last_win_date
			FROM spots s2
			JOIN waffles w2 ON w2.id = s2.waffle_id
			WHERE s2.status = 'winner'
			  AND s2.claimed_by_handle IS NOT NULL
			  AND w2.completed_at IS NOT NULL
			GROUP BY s2.claimed_by_handle
		)
		SELECT
			s.claimed_by_handle,
			COUNT(DISTINCT s.waffle_id),
			MAX(s.claimed_at),
			COALESCE(
				EXTRACT(DAY FROM (NOW() - uw.last_win_date))::int,
				99999
			)
		FROM spots s
		LEFT JOIN user_wins uw ON uw.claimed_by_handle = s.claimed_by_handle
		WHERE s.claimed_by_handle IS NOT NULL
		  AND s.claimed_at IS NOT NULL
		  AND s.status != 'available'
		  AND s.claimed_at >= $1
		  AND s.claimed_at <= $2
		GROUP BY s.claimed_by_handle, uw.last_win_date
		ORDER BY COALESCE(EXTRACT(DAY FROM (NOW() - uw.last_win_date))::int, 99999) DESC
	`, from, to)
	if err != nil {
		return nil, fmt.Errorf("get drought list: %w", err)
	}
	defer rows.Close()

	var entries []models.DroughtEntry
	for rows.Next() {
		var e models.DroughtEntry
		if err := rows.Scan(&e.InstagramHandle, &e.TotalEntries, &e.LastEntryDate, &e.LongestDrought); err != nil {
			return nil, fmt.Errorf("scan drought entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func GetPowerBuyers(from, to time.Time, limit int) ([]models.PowerBuyerEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.Pool.Query(context.Background(), `
		SELECT
			s.claimed_by_handle,
			COUNT(*),
			COALESCE(SUM(CASE WHEN s.status IN ('paid','winner','loser') THEN w.spot_price ELSE 0 END), 0),
			CASE
				WHEN COUNT(*) FILTER (WHERE s.status IN ('winner','loser')) > 0
				THEN ROUND(
					COUNT(*) FILTER (WHERE s.status = 'winner')::decimal /
					COUNT(*) FILTER (WHERE s.status IN ('winner','loser')) * 100, 1
				)
				ELSE 0
			END
		FROM spots s
		JOIN waffles w ON w.id = s.waffle_id
		WHERE s.claimed_by_handle IS NOT NULL
		  AND s.status NOT IN ('available','pending')
		  AND s.claimed_at IS NOT NULL
		  AND s.claimed_at >= $1
		  AND s.claimed_at <= $2
		GROUP BY s.claimed_by_handle
		ORDER BY COUNT(*) DESC
		LIMIT $3
	`, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("get power buyers: %w", err)
	}
	defer rows.Close()

	var entries []models.PowerBuyerEntry
	for rows.Next() {
		var e models.PowerBuyerEntry
		if err := rows.Scan(&e.InstagramHandle, &e.TotalSpots, &e.TotalSpent, &e.WinRate); err != nil {
			return nil, fmt.Errorf("scan power buyer entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func GetMonthlyActivity(from, to time.Time) ([]models.MonthlyActivity, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT
			TO_CHAR(DATE_TRUNC('month', d.date), 'YYYY-MM'),
			COALESCE(SUM(d.waffles), 0),
			COALESCE(SUM(d.spots_claimed), 0),
			COALESCE(SUM(d.revenue), 0)
		FROM (
			SELECT DATE_TRUNC('month', created_at) AS date, 1 AS waffles, 0 AS spots_claimed, 0 AS revenue
			FROM waffles
			WHERE created_at >= $1 AND created_at <= $2

			UNION ALL

			SELECT DATE_TRUNC('month', claimed_at) AS date, 0 AS waffles, 1 AS spots_claimed, 0 AS revenue
			FROM spots
			WHERE claimed_at IS NOT NULL AND claimed_at >= $1 AND claimed_at <= $2

			UNION ALL

			SELECT DATE_TRUNC('month', s.paid_at) AS date, 0 AS waffles, 0 AS spots_claimed, w.spot_price AS revenue
			FROM spots s
			JOIN waffles w ON w.id = s.waffle_id
			WHERE s.paid_at IS NOT NULL AND s.paid_at >= $1 AND s.paid_at <= $2
		) d
		GROUP BY DATE_TRUNC('month', d.date)
		ORDER BY 1
	`, from, to)
	if err != nil {
		return nil, fmt.Errorf("get monthly activity: %w", err)
	}
	defer rows.Close()

	var entries []models.MonthlyActivity
	for rows.Next() {
		var e models.MonthlyActivity
		if err := rows.Scan(&e.Month, &e.Waffles, &e.SpotsClaimed, &e.Revenue); err != nil {
			return nil, fmt.Errorf("scan monthly activity: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func GetPaymentLag() ([]map[string]interface{}, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT 
			s.id, s.number, s.claimed_by_handle, s.claimed_at,
			w.id as waffle_id, w.slug, w.title,
			EXTRACT(EPOCH FROM (NOW() - s.claimed_at))/3600 as hours_pending
		FROM spots s
		JOIN waffles w ON w.id = s.waffle_id
		WHERE s.status = 'pending'
			AND w.status = 'active'
		ORDER BY s.claimed_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("get payment lag: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var spotID, waffleID uuid.UUID
		var spotNumber int
		var handle, slug, title string
		var claimedAt time.Time
		var hoursPending float64

		if err := rows.Scan(
			&spotID, &spotNumber, &handle, &claimedAt,
			&waffleID, &slug, &title, &hoursPending,
		); err != nil {
			return nil, fmt.Errorf("scan payment lag: %w", err)
		}

		entries = append(entries, map[string]interface{}{
			"spot_id":          spotID,
			"spot_number":      spotNumber,
			"instagram_handle": handle,
			"claimed_at":       claimedAt,
			"waffle_id":        waffleID,
			"waffle_slug":      slug,
			"waffle_title":     title,
			"hours_pending":    hoursPending,
		})
	}

	return entries, nil
}

func GetSpotVelocity(status string) ([]models.SpotVelocity, error) {
	query := `
		WITH first_claim AS (
			SELECT waffle_id, MIN(claimed_at) AS first_claim_at
			FROM spots
			WHERE claimed_at IS NOT NULL
			GROUP BY waffle_id
		)
		SELECT
			w.status,
			COUNT(w.id),
			COALESCE(ROUND(AVG(EXTRACT(EPOCH FROM (fc.first_claim_at - w.created_at)) / 3600)::numeric, 1), 0),
			COALESCE(ROUND(AVG(EXTRACT(EPOCH FROM (w.completed_at - w.created_at)) / 3600)::numeric, 1), 0)
		FROM waffles w
		LEFT JOIN first_claim fc ON fc.waffle_id = w.id
		WHERE fc.first_claim_at IS NOT NULL
		  AND ($1 = '' OR w.status = $1)
		GROUP BY w.status
		ORDER BY w.status`

	rows, err := db.Pool.Query(context.Background(), query, status)
	if err != nil {
		return nil, fmt.Errorf("get spot velocity: %w", err)
	}
	defer rows.Close()

	var entries []models.SpotVelocity
	for rows.Next() {
		var e models.SpotVelocity
		if err := rows.Scan(&e.Status, &e.WaffleCount, &e.AvgFirstClaimHours, &e.AvgCompletionHours); err != nil {
			return nil, fmt.Errorf("scan spot velocity: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}
