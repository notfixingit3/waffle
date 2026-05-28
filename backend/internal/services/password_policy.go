package services

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/syrup/backend/internal/db"
)

var commonPasswords = map[string]bool{
	"password":    true,
	"password1":   true,
	"password123": true,
	"12345678":    true,
	"123456789":   true,
	"qwerty":      true,
	"qwerty123":   true,
	"admin":       true,
	"admin123":    true,
	"letmein":     true,
	"welcome":     true,
	"welcome123":  true,
	"iloveyou":    true,
	"monkey":      true,
	"dragon":      true,
	"football":    true,
	"baseball":    true,
	"abc123":      true,
	"111111":      true,
	"00000000":    true,
}

func GetPasswordMinLength() int {
	if db.Pool == nil {
		return 8
	}
	value, err := GetSetting("password_min_length")
	if err != nil || value == "" {
		return 8
	}
	minLength, err := strconv.Atoi(value)
	if err != nil || minLength <= 0 {
		return 8
	}
	return minLength
}

func ValidatePassword(password string) error {
	minLength := GetPasswordMinLength()
	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters", minLength)
	}
	if commonPasswords[strings.ToLower(strings.TrimSpace(password))] {
		return fmt.Errorf("password is too common")
	}
	return nil
}
