package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
)

func CreateAdmin(req models.CreateAdminRequest) (*models.Admin, error) {
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var email string
	if req.Email != nil {
		email = *req.Email
	}

	admin := &models.Admin{
		ID:          uuid.New(),
		Username:    req.Username,
		Email:       email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		Timezone:    "UTC",
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO admins (id, username, email, password_hash, first_name, last_name, display_name, role, active, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, admin.ID, admin.Username, admin.Email, hash, admin.FirstName, admin.LastName, admin.DisplayName, admin.Role, admin.Active, admin.Timezone, admin.CreatedAt, admin.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert admin: %w", err)
	}

	return admin, nil
}

func GetAdminByUsername(username string) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, COALESCE(email, '') as email, display_name, first_name, last_name, social_links, role, timezone, active, last_login_at, last_login_ip, created_at, updated_at
		FROM admins WHERE username = $1
	`, username).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.FirstName, &admin.LastName, &admin.SocialLinks, &admin.Role, &admin.Timezone,
		&admin.Active, &admin.LastLoginAt, &admin.LastLoginIP, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin: %w", err)
	}
	return admin, nil
}

func GetAdminByID(id uuid.UUID) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, COALESCE(email, '') as email, display_name, first_name, last_name, social_links, role, timezone, active, last_login_at, last_login_ip, created_at, updated_at
		FROM admins WHERE id = $1
	`, id).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.FirstName, &admin.LastName, &admin.SocialLinks, &admin.Role, &admin.Timezone,
		&admin.Active, &admin.LastLoginAt, &admin.LastLoginIP, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin: %w", err)
	}
	return admin, nil
}

func GetAdminByEmail(email string) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, COALESCE(email, '') as email, display_name, first_name, last_name, social_links, role, timezone, active, last_login_at, last_login_ip, created_at, updated_at
		FROM admins WHERE email = $1
	`, email).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.FirstName, &admin.LastName, &admin.SocialLinks, &admin.Role, &admin.Timezone,
		&admin.Active, &admin.LastLoginAt, &admin.LastLoginIP, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin: %w", err)
	}
	return admin, nil
}

func ListAdmins() ([]models.Admin, error) {
	rows, err := db.Pool.Query(context.Background(), `
		SELECT id, username, COALESCE(email, '') as email, display_name, first_name, last_name, social_links, role, timezone, active, last_login_at, last_login_ip, created_at, updated_at
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
			&a.ID, &a.Username, &a.Email, &a.DisplayName, &a.FirstName, &a.LastName, &a.SocialLinks, &a.Role, &a.Timezone,
			&a.Active, &a.LastLoginAt, &a.LastLoginIP, &a.CreatedAt, &a.UpdatedAt,
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

func UpdateAdminTimezone(id uuid.UUID, timezone string) (*models.Admin, error) {
	_, err := db.Pool.Exec(context.Background(), `
		UPDATE admins SET timezone = $1, updated_at = $2 WHERE id = $3
	`, timezone, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("update admin timezone: %w", err)
	}
	return GetAdminByID(id)
}

func AuthenticateAdmin(username, password string) (*models.Admin, error) {
	var hash string
	admin := &models.Admin{}
	err := db.Pool.QueryRow(context.Background(), `
		SELECT id, username, COALESCE(email, '') as email, display_name, first_name, last_name, social_links, role, timezone, active, last_login_at, last_login_ip, created_at, updated_at, password_hash
		FROM admins WHERE username = $1
	`, username).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.DisplayName, &admin.FirstName, &admin.LastName, &admin.SocialLinks, &admin.Role, &admin.Timezone,
		&admin.Active, &admin.LastLoginAt, &admin.LastLoginIP, &admin.CreatedAt, &admin.UpdatedAt, &hash,
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

func GetJWTExpirationHours() int {
	if db.Pool == nil {
		return 24
	}
	value, err := GetSetting("jwt_expiration_hours")
	if err != nil || value == "" {
		return 24
	}
	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		return 24
	}
	return hours
}

func GenerateAdminToken(admin *models.Admin) (string, error) {
	expiresIn := time.Duration(GetJWTExpirationHours()) * time.Hour
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin_id": admin.ID.String(),
		"username": admin.Username,
		"role":     admin.Role,
		"exp":      time.Now().Add(expiresIn).Unix(),
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

func GetAdminPasswordHash(id uuid.UUID) (string, error) {
	var hash string
	err := db.Pool.QueryRow(context.Background(), `
		SELECT password_hash FROM admins WHERE id = $1
	`, id).Scan(&hash)
	if err != nil {
		return "", fmt.Errorf("get admin password hash: %w", err)
	}
	return hash, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CompareTokenHash(token, hash string) bool {
	return ConstantTimeCompare(hashToken(token), hash)
}

func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func ValidateSocialLinks(links []models.SocialLink) []string {
	var errors []string
	if len(links) > 10 {
		errors = append(errors, "maximum 10 social links allowed")
	}

	seenPlatforms := make(map[string]bool)
	for i, link := range links {
		validPlatform := false
		for _, platform := range models.ValidSocialPlatforms {
			if link.Platform == platform {
				validPlatform = true
				break
			}
		}
		if !validPlatform {
			errors = append(errors, fmt.Sprintf("social link %d: invalid platform '%s'", i+1, link.Platform))
		}

		if seenPlatforms[link.Platform] {
			errors = append(errors, fmt.Sprintf("social link %d: duplicate platform '%s'", i+1, link.Platform))
		}
		seenPlatforms[link.Platform] = true

		if len(link.Handle) > 100 {
			errors = append(errors, fmt.Sprintf("social link %d: handle exceeds 100 characters", i+1))
		}
	}

	return errors
}

func UpdateAdminProfile(id uuid.UUID, req models.UpdateAdminProfileRequest) (*models.Admin, error) {
	if req.SocialLinks != nil {
		if validationErrors := ValidateSocialLinks(req.SocialLinks); len(validationErrors) > 0 {
			return nil, fmt.Errorf("validation failed: %s", validationErrors[0])
		}
	}

	if req.Email != nil && *req.Email != "" {
		existing, err := GetAdminByEmail(*req.Email)
		if err == nil && existing.ID != id {
			return nil, fmt.Errorf("email already in use")
		}
	}

	var updates []string
	var args []interface{}
	argIndex := 1

	if req.FirstName != nil {
		updates = append(updates, fmt.Sprintf("first_name = $%d", argIndex))
		args = append(args, *req.FirstName)
		argIndex++
	}

	if req.LastName != nil {
		updates = append(updates, fmt.Sprintf("last_name = $%d", argIndex))
		args = append(args, *req.LastName)
		argIndex++
	}

	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argIndex))
		if *req.Email == "" {
			args = append(args, nil)
		} else {
			args = append(args, *req.Email)
		}
		argIndex++
	}

	if req.SocialLinks != nil {
		updates = append(updates, fmt.Sprintf("social_links = $%d", argIndex))
		args = append(args, req.SocialLinks)
		argIndex++
	}

	if len(updates) == 0 {
		return GetAdminByID(id)
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	query := fmt.Sprintf("UPDATE admins SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)
	args = append(args, id)

	_, err := db.Pool.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("update admin profile: %w", err)
	}

	return GetAdminByID(id)
}
