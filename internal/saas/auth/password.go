package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword returns a bcrypt hash suitable for User.PasswordHash.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", fmt.Errorf("password required")
	}
	if len(plain) < 6 {
		return "", fmt.Errorf("password must be at least 6 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword compares plain password against a stored bcrypt hash.
func CheckPassword(hash, plain string) bool {
	if hash == "" || plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
