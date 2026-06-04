package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func GetOrCreateUser(instagramHandle string) (*models.User, error) {
	handle := NormalizeInstagramHandle(instagramHandle)

	if handle == "" {
		return nil, fmt.Errorf("get or create user: instagram handle is empty")
	}

	user := &models.User{
		ID:              uuid.New(),
		InstagramHandle: handle,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO users (id, instagram_handle, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (instagram_handle) DO NOTHING
	`, user.ID, user.InstagramHandle, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get or create user: %w", err)
	}

	existing := &models.User{}
	err = db.Pool.QueryRow(context.Background(), `
		SELECT id, instagram_handle, created_at, updated_at
		FROM users WHERE instagram_handle = $1
	`, handle).Scan(
		&existing.ID, &existing.InstagramHandle, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user after upsert: %w", err)
	}
	return existing, nil
}

func ListUsers(search string, limit, offset int) ([]models.User, int, error) {
	var total int
	countArgs := []interface{}{}
	countQuery := `SELECT COUNT(*) FROM users`

	if search != "" {
		countQuery = `SELECT COUNT(*) FROM users WHERE instagram_handle ILIKE '%' || $1 || '%'`
		countArgs = append(countArgs, search)
	}

	err := db.Pool.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	dataQuery := `SELECT id, instagram_handle, created_at, updated_at FROM users`
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		dataQuery += fmt.Sprintf(` WHERE instagram_handle ILIKE '%%' || $%d || '%%'`, argIdx)
		args = append(args, search)
		argIdx++
	}

	dataQuery += ` ORDER BY instagram_handle ASC`

	if limit > 0 {
		dataQuery += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, limit)
		argIdx++
	}

	if offset > 0 {
		dataQuery += fmt.Sprintf(` OFFSET $%d`, argIdx)
		args = append(args, offset)
		argIdx++
	}

	rows, err := db.Pool.Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.InstagramHandle, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

func BackfillUsers() (int, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT DISTINCT claimed_by_handle
		FROM spots
		WHERE claimed_by_handle IS NOT NULL AND claimed_by_handle != ''
	`)
	if err != nil {
		return 0, fmt.Errorf("backfill users: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	created := 0

	for rows.Next() {
		var handle string
		if err := rows.Scan(&handle); err != nil {
			return 0, fmt.Errorf("scan handle: %w", err)
		}

		handle = NormalizeInstagramHandle(handle)

		tag, err := db.Pool.Exec(context.Background(), `
			INSERT INTO users (id, instagram_handle, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (instagram_handle) DO NOTHING
		`, uuid.New(), handle, now, now)
		if err != nil {
			return 0, fmt.Errorf("backfill insert: %w", err)
		}

		created += int(tag.RowsAffected())
	}

	return created, nil
}
