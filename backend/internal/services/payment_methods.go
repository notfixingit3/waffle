package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func CreatePaymentMethod(req models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	pm := &models.PaymentMethod{
		ID:          uuid.New(),
		Type:        req.Type,
		DisplayName: req.DisplayName,
		HandleOrURL: req.HandleOrURL,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO payment_methods (id, type, display_name, handle_or_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, pm.ID, pm.Type, pm.DisplayName, pm.HandleOrURL, pm.IsActive, pm.CreatedAt, pm.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create payment method: %w", err)
	}

	return pm, nil
}

func ListPaymentMethods() ([]models.PaymentMethod, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT id, type, display_name, handle_or_url, is_active, created_at, updated_at
		FROM payment_methods
		WHERE is_active = true
		ORDER BY display_name
	`)
	if err != nil {
		return nil, fmt.Errorf("list payment methods: %w", err)
	}
	defer rows.Close()

	var methods []models.PaymentMethod
	for rows.Next() {
		var m models.PaymentMethod
		err := rows.Scan(
			&m.ID, &m.Type, &m.DisplayName, &m.HandleOrURL, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan payment method: %w", err)
		}
		methods = append(methods, m)
	}

	return methods, nil
}

func ListAllPaymentMethods() ([]models.PaymentMethod, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT id, type, display_name, handle_or_url, is_active, created_at, updated_at
		FROM payment_methods
		ORDER BY display_name
	`)
	if err != nil {
		return nil, fmt.Errorf("list all payment methods: %w", err)
	}
	defer rows.Close()

	var methods []models.PaymentMethod
	for rows.Next() {
		var m models.PaymentMethod
		err := rows.Scan(
			&m.ID, &m.Type, &m.DisplayName, &m.HandleOrURL, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan payment method: %w", err)
		}
		methods = append(methods, m)
	}

	return methods, nil
}

func GetPaymentMethodByID(id uuid.UUID) (*models.PaymentMethod, error) {
	pm := &models.PaymentMethod{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, type, display_name, handle_or_url, is_active, created_at, updated_at
		FROM payment_methods
		WHERE id = $1
	`, id).Scan(
		&pm.ID, &pm.Type, &pm.DisplayName, &pm.HandleOrURL, &pm.IsActive, &pm.CreatedAt, &pm.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get payment method: %w", err)
	}
	return pm, nil
}

func UpdatePaymentMethod(id uuid.UUID, req models.UpdatePaymentMethodRequest) (*models.PaymentMethod, error) {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE payment_methods SET
			display_name = COALESCE($1, display_name),
			handle_or_url = COALESCE($2, handle_or_url),
			updated_at = $3
		WHERE id = $4
	`, req.DisplayName, req.HandleOrURL, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("update payment method: %w", err)
	}
	return GetPaymentMethodByID(id)
}

func DeactivatePaymentMethod(id uuid.UUID) error {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE payment_methods SET is_active = false, updated_at = $1 WHERE id = $2
	`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("deactivate payment method: %w", err)
	}
	return nil
}

func GetPaymentMethodsForWaffle(waffleID uuid.UUID) ([]models.PaymentMethod, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT pm.id, pm.type, pm.display_name, pm.handle_or_url, pm.is_active, pm.created_at, pm.updated_at
		FROM payment_methods pm
		JOIN waffle_payment_methods wpm ON wpm.payment_method_id = pm.id
		WHERE wpm.waffle_id = $1
		ORDER BY pm.display_name
	`, waffleID)
	if err != nil {
		return nil, fmt.Errorf("get payment methods for waffle: %w", err)
	}
	defer rows.Close()

	var methods []models.PaymentMethod
	for rows.Next() {
		var m models.PaymentMethod
		err := rows.Scan(
			&m.ID, &m.Type, &m.DisplayName, &m.HandleOrURL, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan payment method: %w", err)
		}
		methods = append(methods, m)
	}

	return methods, nil
}

func SetWafflePaymentMethods(waffleID uuid.UUID, methodIDs []uuid.UUID) error {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `
		DELETE FROM waffle_payment_methods WHERE waffle_id = $1
	`, waffleID)
	if err != nil {
		return fmt.Errorf("delete existing payment methods: %w", err)
	}

	for _, id := range methodIDs {
		_, err = tx.Exec(context.Background(), `
			INSERT INTO waffle_payment_methods (waffle_id, payment_method_id)
			VALUES ($1, $2)
		`, waffleID, id)
		if err != nil {
			return fmt.Errorf("insert payment method %s: %w", id, err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func GetPaymentMethodUsageCount(id uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(DISTINCT wpm.waffle_id)
		FROM waffle_payment_methods wpm
		JOIN waffles w ON w.id = wpm.waffle_id
		WHERE wpm.payment_method_id = $1 AND w.status = $2
	`, id, models.WaffleStatusActive).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get payment method usage count: %w", err)
	}
	return count, nil
}
