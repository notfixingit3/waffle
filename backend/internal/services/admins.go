package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func CreateAdmin(req models.CreateAdminRequest) (*models.Admin, error) {
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	admin := &models.Admin{
		ID:          uuid.New(),
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		Timezone:    "UTC",
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, display_name, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, admin.ID, admin.Username, admin.Email, hash, admin.DisplayName, admin.Role, admin.Active, admin.Timezone, admin.CreatedAt, admin.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert admin: %w", err)
	}

	return admin, nil
}

func GetAdminByUsername(username string) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, email, display_name, role, active, last_login_at, created_at, updated_at
		FROM admins WHERE username = $1
	`, username).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.Role,
		&admin.Active, &admin.LastLoginAt, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin: %w", err)
	}
	return admin, nil
}

func GetAdminByID(id uuid.UUID) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, email, display_name, role, active, last_login_at, created_at, updated_at
		FROM admins WHERE id = $1
	`, id).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.Role,
		&admin.Active, &admin.LastLoginAt, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin: %w", err)
	}
	return admin, nil
}

func GetAdminByEmail(email string) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, email, display_name, role, active, last_login_at, created_at, updated_at
		FROM admins WHERE email = $1
	`, email).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.Role,
		&admin.Active, &admin.LastLoginAt, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin: %w", err)
	}
	return admin, nil
}

func ListAdmins() ([]models.Admin, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT id, username, email, display_name, role, active, last_login_at, created_at, updated_at
		FROM admins ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	var admins []models.Admin
	for rows.Next() {
		var a models.Admin
		err := rows.Scan(
			&a.ID, &a.Username, &a.Email, &a.DisplayName, &a.Role,
			&a.Active, &a.LastLoginAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan admin: %w", err)
		}
		admins = append(admins, a)
	}
	return admins, nil
}

func UpdateAdmin(id uuid.UUID, req models.UpdateAdminRequest) (*models.Admin, error) {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE admins SET
			display_name = COALESCE($1, display_name),
			role = COALESCE($2, role),
			active = COALESCE($3, active),
			updated_at = $4
		WHERE id = $5
	`, req.DisplayName, req.Role, req.Active, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("update admin: %w", err)
	}
	return GetAdminByID(id)
}

func UpdateAdminPassword(id uuid.UUID, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = db.Pool.Exec(context.Background(), `
		UPDATE admins SET password_hash = $1, updated_at = $2 WHERE id = $3
	`, hash, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func RecordLogin(id uuid.UUID) error {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE admins SET last_login_at = $1 WHERE id = $2
	`, time.Now(), id)
	return err
}

func AuthenticateAdmin(username, password string) (*models.Admin, error) {
	var hash string
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, email, display_name, role, active, last_login_at, created_at, updated_at, password_hash
		FROM admins WHERE username = $1
	`, username).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.Role,
		&admin.Active, &admin.LastLoginAt, &admin.CreatedAt, &admin.UpdatedAt, &hash,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !admin.Active {
		return nil, fmt.Errorf("account deactivated")
	}

	if !CheckPassword(password, hash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	return admin, nil
}

func GenerateAdminToken(admin *models.Admin) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin_id": admin.ID.String(),
		"username": admin.Username,
		"role":     admin.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(GetJWTSecret()))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tokenString, nil
}

func CreatePasswordResetToken(adminID uuid.UUID) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(bytes)
	tokenHash := hashToken(token)

	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO password_reset_tokens (admin_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, adminID, tokenHash, time.Now().Add(time.Hour*1))
	if err != nil {
		return "", fmt.Errorf("save token: %w", err)
	}

	return token, nil
}

func ValidatePasswordResetToken(token string) (uuid.UUID, error) {
	tokenHash := hashToken(token)

	var adminID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time
	err := db.Pool.QueryRow(context.Background(), `
		SELECT admin_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1
	`, tokenHash).Scan(&adminID, &expiresAt, &usedAt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token")
	}

	if usedAt != nil {
		return uuid.Nil, fmt.Errorf("token already used")
	}

	if time.Now().After(expiresAt) {
		return uuid.Nil, fmt.Errorf("token expired")
	}

	return adminID, nil
}

func MarkResetTokenUsed(token string) error {
	tokenHash := hashToken(token)
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE password_reset_tokens SET used_at = $1 WHERE token_hash = $2
	`, time.Now(), tokenHash)
	return err
}

func hashToken(token string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	return string(hash)
}

func CompareTokenHash(token, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
	return err == nil
}

func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}