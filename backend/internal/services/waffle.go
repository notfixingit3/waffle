package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func CreateWaffle(req models.CreateWaffleRequest) (*models.Waffle, error) {
	slug := generateSlug(req.Title)
	itemCount := req.ItemCount
	if itemCount <= 0 {
		itemCount = 1
	}
	if itemCount > 10 {
		itemCount = 10
	}
	
	waffle := &models.Waffle{
		ID:                    uuid.New(),
		Slug:                  slug,
		Title:                 req.Title,
		Description:           req.Description,
		ImageURL:              req.ImageURL,
		TotalSpots:            req.TotalSpots,
		SpotPrice:             req.SpotPrice,
		PaymentInfo:           req.PaymentInfo,
		InstagramMediaLinks:   req.InstagramMediaLinks,
		Status:                models.WaffleStatusActive,
		ItemCount:             itemCount,
		CreatedAt:             time.Now(),
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `
		INSERT INTO waffles (id, slug, title, description, image_url, total_spots, spot_price, payment_info, status, instagram_media_links, created_at, item_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, waffle.ID, waffle.Slug, waffle.Title, waffle.Description, waffle.ImageURL,
		waffle.TotalSpots, waffle.SpotPrice, waffle.PaymentInfo, waffle.Status, waffle.InstagramMediaLinks, waffle.CreatedAt, waffle.ItemCount)
	if err != nil {
		return nil, fmt.Errorf("insert waffle: %w", err)
	}

	for i := 1; i <= req.TotalSpots; i++ {
		_, err = tx.Exec(context.Background(), `
			INSERT INTO spots (id, waffle_id, number, status)
			VALUES ($1, $2, $3, $4)
		`, uuid.New(), waffle.ID, i, models.SpotStatusAvailable)
		if err != nil {
			return nil, fmt.Errorf("insert spot %d: %w", i, err)
		}
	}

	for _, pmID := range req.PaymentMethodIDs {
		var exists bool
		err := tx.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM payment_methods WHERE id = $1 AND is_active = true)`, pmID).Scan(&exists)
		if err != nil || !exists {
			return nil, fmt.Errorf("invalid or inactive payment method: %s", pmID)
		}
	}

	for _, pmID := range req.PaymentMethodIDs {
		_, err := tx.Exec(context.Background(), `INSERT INTO waffle_payment_methods (waffle_id, payment_method_id) VALUES ($1, $2)`, waffle.ID, pmID)
		if err != nil {
			return nil, fmt.Errorf("link payment method %s: %w", pmID, err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return waffle, nil
}

func GetWaffleBySlug(slug string) (*models.Waffle, error) {
	waffle := &models.Waffle{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, slug, title, description, image_url, total_spots, spot_price, payment_info, status, winning_spot_number, winning_instagram_handle, instagram_media_links, archived, created_at, completed_at, item_count, winning_spot_numbers, winning_instagram_handles
		FROM waffles WHERE slug = $1
	`, slug).Scan(
		&waffle.ID, &waffle.Slug, &waffle.Title, &waffle.Description, &waffle.ImageURL,
		&waffle.TotalSpots, &waffle.SpotPrice, &waffle.PaymentInfo, &waffle.Status,
		&waffle.WinningSpotNumber, &waffle.WinningInstagramHandle, &waffle.InstagramMediaLinks, &waffle.Archived,
		&waffle.CreatedAt, &waffle.CompletedAt, &waffle.ItemCount, &waffle.WinningSpotNumbers, &waffle.WinningInstagramHandles,
	)
	if err != nil {
		return nil, fmt.Errorf("get waffle: %w", err)
	}
	return waffle, nil
}

func GetWaffleByID(id uuid.UUID) (*models.Waffle, error) {
	waffle := &models.Waffle{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, slug, title, description, image_url, total_spots, spot_price, payment_info, status, winning_spot_number, winning_instagram_handle, instagram_media_links, archived, created_at, completed_at, item_count, winning_spot_numbers, winning_instagram_handles
		FROM waffles WHERE id = $1
	`, id).Scan(
		&waffle.ID, &waffle.Slug, &waffle.Title, &waffle.Description, &waffle.ImageURL,
		&waffle.TotalSpots, &waffle.SpotPrice, &waffle.PaymentInfo, &waffle.Status,
		&waffle.WinningSpotNumber, &waffle.WinningInstagramHandle, &waffle.InstagramMediaLinks, &waffle.Archived,
		&waffle.CreatedAt, &waffle.CompletedAt, &waffle.ItemCount, &waffle.WinningSpotNumbers, &waffle.WinningInstagramHandles,
	)
	if err != nil {
		return nil, fmt.Errorf("get waffle: %w", err)
	}
	return waffle, nil
}

func GetSpotsByWaffleID(waffleID uuid.UUID) ([]models.Spot, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT id, waffle_id, number, status, claimed_by_handle, claimed_at, paid_at
		FROM spots WHERE waffle_id = $1 ORDER BY number
	`, waffleID)
	if err != nil {
		return nil, fmt.Errorf("query spots: %w", err)
	}
	defer rows.Close()

	var spots []models.Spot
	for rows.Next() {
		var spot models.Spot
		err := rows.Scan(
			&spot.ID, &spot.WaffleID, &spot.Number, &spot.Status,
			&spot.ClaimedByHandle, &spot.ClaimedAt, &spot.PaidAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan spot: %w", err)
		}
		spots = append(spots, spot)
	}

	return spots, nil
}

func ClaimSpots(waffleID uuid.UUID, spotNumbers []int, instagramHandle string) error {
	instagramHandle = NormalizeInstagramHandle(instagramHandle)
	if instagramHandle == "" {
		return fmt.Errorf("instagram handle is required")
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	var waffleStatus string
	err = tx.QueryRow(context.Background(), `
		SELECT status FROM waffles WHERE id = $1 FOR UPDATE
	`, waffleID).Scan(&waffleStatus)
	if err != nil {
		return fmt.Errorf("get waffle status: %w", err)
	}
	if waffleStatus != string(models.WaffleStatusActive) {
		return fmt.Errorf("waffle is not active")
	}

	for _, num := range spotNumbers {
		var spotID uuid.UUID
		var currentStatus string
		err := tx.QueryRow(context.Background(), `
			SELECT id, status FROM spots WHERE waffle_id = $1 AND number = $2 FOR UPDATE
		`, waffleID, num).Scan(&spotID, &currentStatus)
		if err != nil {
			return fmt.Errorf("spot %d not found: %w", num, err)
		}
		if currentStatus != string(models.SpotStatusAvailable) {
			return fmt.Errorf("spot %d is not available", num)
		}

		_, err = tx.Exec(context.Background(), `
			UPDATE spots SET status = $1, claimed_by_handle = $2, claimed_at = $3
			WHERE id = $4
		`, models.SpotStatusPending, instagramHandle, time.Now(), spotID)
		if err != nil {
			return fmt.Errorf("claim spot %d: %w", num, err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	if _, err := GetOrCreateUser(instagramHandle); err != nil {
		slog.Error("Failed to create user record after claim", "handle", instagramHandle, "error", err)
	}

	return nil
}

func ClaimRandomSpots(waffleID uuid.UUID, count int, instagramHandle string) ([]int, error) {
	instagramHandle = NormalizeInstagramHandle(instagramHandle)
	if instagramHandle == "" {
		return nil, fmt.Errorf("instagram handle is required")
	}

	if count <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	var waffleStatus string
	err = tx.QueryRow(context.Background(), `
		SELECT status FROM waffles WHERE id = $1 FOR UPDATE
	`, waffleID).Scan(&waffleStatus)
	if err != nil {
		return nil, fmt.Errorf("get waffle status: %w", err)
	}
	if waffleStatus != string(models.WaffleStatusActive) {
		return nil, fmt.Errorf("waffle is not active")
	}

	rows, err := tx.Query(context.Background(), `
		SELECT id, number FROM spots WHERE waffle_id = $1 AND status = $2 ORDER BY RANDOM() LIMIT $3 FOR UPDATE
	`, waffleID, models.SpotStatusAvailable, count)
	if err != nil {
		return nil, fmt.Errorf("query available spots: %w", err)
	}
	defer rows.Close()

	var spotIDs []uuid.UUID
	var spotNumbers []int
	for rows.Next() {
		var spotID uuid.UUID
		var spotNumber int
		if err := rows.Scan(&spotID, &spotNumber); err != nil {
			return nil, fmt.Errorf("scan spot: %w", err)
		}
		spotIDs = append(spotIDs, spotID)
		spotNumbers = append(spotNumbers, spotNumber)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spots: %w", err)
	}

	if len(spotIDs) == 0 {
		return []int{}, nil
	}

	for i, spotID := range spotIDs {
		_, err = tx.Exec(context.Background(), `
			UPDATE spots SET status = $1, claimed_by_handle = $2, claimed_at = $3
			WHERE id = $4
		`, models.SpotStatusPending, instagramHandle, time.Now(), spotID)
		if err != nil {
			return nil, fmt.Errorf("claim spot %d: %w", spotNumbers[i], err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if _, err := GetOrCreateUser(instagramHandle); err != nil {
		slog.Error("Failed to create user record after random claim", "handle", instagramHandle, "error", err)
	}

	return spotNumbers, nil
}

func MarkSpotPaid(spotID uuid.UUID) error {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE spots SET status = $1, paid_at = $2
		WHERE id = $3 AND status = $4
	`, models.SpotStatusPaid, time.Now(), spotID, models.SpotStatusPending)
	if err != nil {
		return fmt.Errorf("mark paid: %w", err)
	}
	return nil
}

func GetSpotByID(spotID uuid.UUID) (*models.Spot, error) {
	spot := &models.Spot{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, waffle_id, number, status, claimed_by_handle, claimed_at, paid_at
		FROM spots WHERE id = $1
	`, spotID).Scan(
		&spot.ID, &spot.WaffleID, &spot.Number, &spot.Status,
		&spot.ClaimedByHandle, &spot.ClaimedAt, &spot.PaidAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get spot: %w", err)
	}
	return spot, nil
}

func ReleaseSpot(spotID uuid.UUID) error {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE spots SET status = $1, claimed_by_handle = NULL, claimed_at = NULL
		WHERE id = $2 AND status = $3
	`, models.SpotStatusAvailable, spotID, models.SpotStatusPending)
	if err != nil {
		return fmt.Errorf("release spot: %w", err)
	}
	return nil
}

func SetWinner(waffleID uuid.UUID, winningSpotNumbers []int) error {
	if len(winningSpotNumbers) == 0 {
		return fmt.Errorf("no winning spot numbers provided")
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	var totalSpots, paidSpots, itemCount int
	var status string
	err = tx.QueryRow(context.Background(), `
		SELECT total_spots, status, item_count,
			(SELECT COUNT(*) FROM spots WHERE waffle_id = $1 AND status = $2)
		FROM waffles WHERE id = $1
	`, waffleID, models.SpotStatusPaid).Scan(&totalSpots, &status, &itemCount, &paidSpots)
	if err != nil {
		return fmt.Errorf("check filled status: %w", err)
	}
	if status != string(models.WaffleStatusActive) {
		return fmt.Errorf("waffle is not active")
	}
	if totalSpots != paidSpots {
		return fmt.Errorf("cannot set winner: only %d of %d spots are paid", paidSpots, totalSpots)
	}
	if len(winningSpotNumbers) != itemCount {
		return fmt.Errorf("must provide exactly %d winning spot numbers, got %d", itemCount, len(winningSpotNumbers))
	}

	var winningSpotIDs []uuid.UUID
	var winningHandles []string
	for _, spotNum := range winningSpotNumbers {
		var spotID uuid.UUID
		var handle string
		err = tx.QueryRow(context.Background(), `
			SELECT id, claimed_by_handle FROM spots 
			WHERE waffle_id = $1 AND number = $2 AND status = $3
			FOR UPDATE
		`, waffleID, spotNum, models.SpotStatusPaid).Scan(&spotID, &handle)
		if err != nil {
			return fmt.Errorf("winning spot #%d not found or not paid: %w", spotNum, err)
		}
		winningSpotIDs = append(winningSpotIDs, spotID)
		winningHandles = append(winningHandles, handle)
	}

	for _, spotID := range winningSpotIDs {
		_, err = tx.Exec(context.Background(), `
			UPDATE spots SET status = $1 WHERE id = $2
		`, models.SpotStatusWinner, spotID)
		if err != nil {
			return fmt.Errorf("mark winner: %w", err)
		}
	}

	_, err = tx.Exec(context.Background(), `
		UPDATE spots SET status = $1 
		WHERE waffle_id = $2 AND status = $3
	`, models.SpotStatusLoser, waffleID, models.SpotStatusPaid)
	if err != nil {
		return fmt.Errorf("mark losers: %w", err)
	}

	firstWinningSpot := winningSpotNumbers[0]
	firstWinningHandle := winningHandles[0]

	_, err = tx.Exec(context.Background(), `
		UPDATE waffles SET status = $1, winning_spot_number = $2, winning_instagram_handle = $3, completed_at = $4, winning_spot_numbers = $5, winning_instagram_handles = $6
		WHERE id = $7
	`, models.WaffleStatusCompleted, firstWinningSpot, firstWinningHandle, time.Now(), winningSpotNumbers, winningHandles, waffleID)
	if err != nil {
		return fmt.Errorf("complete waffle: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	spots, err := GetSpotsByWaffleID(waffleID)
	if err != nil {
		slog.Error("Failed to get spots for buyer stats update", "error", err)
	} else {
		for _, spot := range spots {
			if spot.ClaimedByHandle != nil {
				isWin := spot.Status == models.SpotStatusWinner
				if err := UpdateBuyerStats(*spot.ClaimedByHandle, isWin); err != nil {
					slog.Error("Failed to update buyer stats", "handle", *spot.ClaimedByHandle, "error", err)
				}
			}
		}
	}

	return nil
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug + "-" + uuid.New().String()[:8]
}

func NormalizeInstagramHandle(handle string) string {
	handle = strings.TrimPrefix(handle, "@")
	handle = strings.ToLower(handle)
	return handle
}

func UpdateWaffle(id uuid.UUID, req models.UpdateWaffleRequest) (*models.Waffle, error) {
	itemCount := req.ItemCount
	if itemCount <= 0 {
		itemCount = 1
	}
	if itemCount > 10 {
		itemCount = 10
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `
		UPDATE waffles SET title = $1, description = $2, image_url = $3, spot_price = $4, payment_info = $5, instagram_media_links = $6, item_count = $7
		WHERE id = $8
	`, req.Title, req.Description, req.ImageURL, req.SpotPrice, req.PaymentInfo, req.InstagramMediaLinks, itemCount, id)
	if err != nil {
		return nil, fmt.Errorf("update waffle: %w", err)
	}

	_, err = tx.Exec(context.Background(), `
		DELETE FROM waffle_payment_methods WHERE waffle_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("delete existing payment methods: %w", err)
	}

	for _, pmID := range req.PaymentMethodIDs {
		var exists bool
		err = tx.QueryRow(context.Background(), `
			SELECT EXISTS(SELECT 1 FROM payment_methods WHERE id = $1)
		`, pmID).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("validate payment method %s: %w", pmID, err)
		}
		if !exists {
			return nil, fmt.Errorf("invalid payment method ID: %s", pmID)
		}

		_, err = tx.Exec(context.Background(), `
			INSERT INTO waffle_payment_methods (waffle_id, payment_method_id)
			VALUES ($1, $2)
		`, id, pmID)
		if err != nil {
			return nil, fmt.Errorf("insert payment method %s: %w", pmID, err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return GetWaffleByID(id)
}

func ArchiveWaffle(id uuid.UUID, archived bool) error {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE waffles SET archived = $1 WHERE id = $2
	`, archived, id)
	if err != nil {
		return fmt.Errorf("archive waffle: %w", err)
	}
	return nil
}

func DeleteWaffle(id uuid.UUID) error {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `DELETE FROM spots WHERE waffle_id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete spots: %w", err)
	}

	_, err = tx.Exec(context.Background(), `DELETE FROM activity_events WHERE waffle_id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete activity events: %w", err)
	}

	_, err = tx.Exec(context.Background(), `DELETE FROM waffles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete waffle: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func CheckWaffleFilled(waffleID uuid.UUID) (bool, error) {
	var totalSpots, paidSpots int
	err := db.Pool.QueryRow(context.Background(), `
		SELECT 
			(SELECT total_spots FROM waffles WHERE id = $1),
			(SELECT COUNT(*) FROM spots WHERE waffle_id = $1 AND status = $2)
	`, waffleID, models.SpotStatusPaid).Scan(&totalSpots, &paidSpots)
	if err != nil {
		return false, fmt.Errorf("check filled: %w", err)
	}
	return totalSpots == paidSpots, nil
}

func UpdateBuyerStats(instagramHandle string, isWin bool) error {
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO buyer_stats (instagram_handle, total_waffles_entered, total_wins, total_losses, total_spots_claimed, updated_at)
		VALUES ($1, 1, $2, $3, 1, $4)
		ON CONFLICT (instagram_handle) DO UPDATE SET
			total_waffles_entered = buyer_stats.total_waffles_entered + 1,
			total_wins = buyer_stats.total_wins + $2,
			total_losses = buyer_stats.total_losses + $3,
			total_spots_claimed = buyer_stats.total_spots_claimed + 1,
			updated_at = $4
	`, instagramHandle, boolToInt(isWin), boolToInt(!isWin), time.Now())
	if err != nil {
		return fmt.Errorf("update buyer stats: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func ListWaffles(includeArchived bool) ([]models.Waffle, error) {
	var query string
	if includeArchived {
		query = `
			SELECT id, slug, title, description, image_url, total_spots, spot_price, payment_info, status, winning_spot_number, winning_instagram_handle, instagram_media_links, archived, created_at, completed_at, item_count, winning_spot_numbers, winning_instagram_handles
			FROM waffles WHERE archived = true ORDER BY created_at DESC
		`
	} else {
		query = `
			SELECT id, slug, title, description, image_url, total_spots, spot_price, payment_info, status, winning_spot_number, winning_instagram_handle, instagram_media_links, archived, created_at, completed_at, item_count, winning_spot_numbers, winning_instagram_handles
			FROM waffles WHERE archived = false ORDER BY created_at DESC
		`
	}

	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("list waffles: %w", err)
	}
	defer rows.Close()

	var waffles []models.Waffle
	for rows.Next() {
		var waffle models.Waffle
		err := rows.Scan(
			&waffle.ID, &waffle.Slug, &waffle.Title, &waffle.Description, &waffle.ImageURL,
			&waffle.TotalSpots, &waffle.SpotPrice, &waffle.PaymentInfo, &waffle.Status,
			&waffle.WinningSpotNumber, &waffle.WinningInstagramHandle, &waffle.InstagramMediaLinks, &waffle.Archived,
			&waffle.CreatedAt, &waffle.CompletedAt, &waffle.ItemCount, &waffle.WinningSpotNumbers, &waffle.WinningInstagramHandles,
		)
		if err != nil {
			return nil, fmt.Errorf("scan waffle: %w", err)
		}
		waffles = append(waffles, waffle)
	}

	return waffles, nil
}

func CountWaffles() (total int, active int, err error) {
	err = db.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM waffles`).Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("count total waffles: %w", err)
	}

	err = db.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM waffles WHERE status = $1 AND archived = false`, models.WaffleStatusActive).Scan(&active)
	if err != nil {
		return 0, 0, fmt.Errorf("count active waffles: %w", err)
	}

	return total, active, nil
}

func GetWaffleStats(waffleID uuid.UUID) (map[string]interface{}, error) {
	var totalSpots, availableSpots, pendingSpots, paidSpots int
	err := db.Pool.QueryRow(context.Background(), `
		SELECT 
			(SELECT total_spots FROM waffles WHERE id = $1),
			(SELECT COUNT(*) FROM spots WHERE waffle_id = $1 AND status = $2),
			(SELECT COUNT(*) FROM spots WHERE waffle_id = $1 AND status = $3),
			(SELECT COUNT(*) FROM spots WHERE waffle_id = $1 AND status = $4)
	`, waffleID, models.SpotStatusAvailable, models.SpotStatusPending, models.SpotStatusPaid).Scan(
		&totalSpots, &availableSpots, &pendingSpots, &paidSpots,
	)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	return map[string]interface{}{
		"total_spots":     totalSpots,
		"available":       availableSpots,
		"pending":         pendingSpots,
		"paid":            paidSpots,
		"spots_remaining": availableSpots,
	}, nil
}

func ClearWinner(waffleID uuid.UUID) error {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	var status string
	err = tx.QueryRow(context.Background(), `
		SELECT status FROM waffles WHERE id = $1 FOR UPDATE
	`, waffleID).Scan(&status)
	if err != nil {
		return fmt.Errorf("get waffle status: %w", err)
	}
	if status != string(models.WaffleStatusCompleted) {
		return fmt.Errorf("waffle is not completed")
	}

	_, err = tx.Exec(context.Background(), `
		UPDATE spots SET status = $1
		WHERE waffle_id = $2 AND status IN ($3, $4)
	`, models.SpotStatusPaid, waffleID, models.SpotStatusWinner, models.SpotStatusLoser)
	if err != nil {
		return fmt.Errorf("reset spots: %w", err)
	}

	_, err = tx.Exec(context.Background(), `
		UPDATE waffles SET status = $1, winning_spot_number = NULL, winning_instagram_handle = NULL, completed_at = NULL, winning_spot_numbers = '[]', winning_instagram_handles = '[]'
		WHERE id = $2
	`, models.WaffleStatusActive, waffleID)
	if err != nil {
		return fmt.Errorf("reset waffle: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	spots, err := GetSpotsByWaffleID(waffleID)
	if err != nil {
		slog.Error("Failed to get spots for buyer stats update", "error", err)
	} else {
		for _, spot := range spots {
			if spot.ClaimedByHandle != nil {
				isWin := spot.Status == models.SpotStatusWinner
				if err := UpdateBuyerStats(*spot.ClaimedByHandle, isWin); err != nil {
					slog.Error("Failed to update buyer stats", "handle", *spot.ClaimedByHandle, "error", err)
				}
			}
		}
	}

	return nil
}

func ChangeWinner(waffleID uuid.UUID, newWinningSpotNumbers []int) error {
	if len(newWinningSpotNumbers) == 0 {
		return fmt.Errorf("no winning spot numbers provided")
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	var status string
	var itemCount int
	err = tx.QueryRow(context.Background(), `
		SELECT status, item_count FROM waffles WHERE id = $1 FOR UPDATE
	`, waffleID).Scan(&status, &itemCount)
	if err != nil {
		return fmt.Errorf("get waffle details: %w", err)
	}
	if status != string(models.WaffleStatusCompleted) {
		return fmt.Errorf("waffle is not completed")
	}

	if len(newWinningSpotNumbers) != itemCount {
		return fmt.Errorf("must provide exactly %d winning spot numbers, got %d", itemCount, len(newWinningSpotNumbers))
	}

	// Reset all winner/loser spots back to paid temporarily, so we can re-evaluate them
	_, err = tx.Exec(context.Background(), `
		UPDATE spots SET status = $1
		WHERE waffle_id = $2 AND status IN ($3, $4)
	`, models.SpotStatusPaid, waffleID, models.SpotStatusWinner, models.SpotStatusLoser)
	if err != nil {
		return fmt.Errorf("reset spots: %w", err)
	}

	var winningSpotIDs []uuid.UUID
	var winningHandles []string
	for _, spotNum := range newWinningSpotNumbers {
		var spotID uuid.UUID
		var handle string
		err = tx.QueryRow(context.Background(), `
			SELECT id, claimed_by_handle FROM spots 
			WHERE waffle_id = $1 AND number = $2 AND status = $3
			FOR UPDATE
		`, waffleID, spotNum, models.SpotStatusPaid).Scan(&spotID, &handle)
		if err != nil {
			return fmt.Errorf("new winning spot #%d not found or not paid: %w", spotNum, err)
		}
		winningSpotIDs = append(winningSpotIDs, spotID)
		winningHandles = append(winningHandles, handle)
	}

	for _, spotID := range winningSpotIDs {
		_, err = tx.Exec(context.Background(), `
			UPDATE spots SET status = $1 WHERE id = $2
		`, models.SpotStatusWinner, spotID)
		if err != nil {
			return fmt.Errorf("promote new winner: %w", err)
		}
	}

	_, err = tx.Exec(context.Background(), `
		UPDATE spots SET status = $1 
		WHERE waffle_id = $2 AND status = $3
	`, models.SpotStatusLoser, waffleID, models.SpotStatusPaid)
	if err != nil {
		return fmt.Errorf("mark losers: %w", err)
	}

	firstWinningSpot := newWinningSpotNumbers[0]
	firstWinningHandle := winningHandles[0]

	_, err = tx.Exec(context.Background(), `
		UPDATE waffles SET winning_spot_number = $1, winning_instagram_handle = $2, winning_spot_numbers = $3, winning_instagram_handles = $4
		WHERE id = $5
	`, firstWinningSpot, firstWinningHandle, newWinningSpotNumbers, winningHandles, waffleID)
	if err != nil {
		return fmt.Errorf("update waffle winner: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	spots, err := GetSpotsByWaffleID(waffleID)
	if err != nil {
		slog.Error("Failed to get spots for buyer stats update", "error", err)
	} else {
		for _, spot := range spots {
			if spot.ClaimedByHandle != nil {
				isWin := spot.Status == models.SpotStatusWinner
				if err := UpdateBuyerStats(*spot.ClaimedByHandle, isWin); err != nil {
					slog.Error("Failed to update buyer stats", "handle", *spot.ClaimedByHandle, "error", err)
				}
			}
		}
	}

	return nil
}
