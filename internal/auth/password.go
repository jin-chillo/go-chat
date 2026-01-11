package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptCost is the cost factor for bcrypt hashing.
	// TRD.md specifies cost=12 for password hashing.
	bcryptCost = 12
)

// HashPassword generates a bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a password with a bcrypt hash.
// Returns nil if the password matches, or an error otherwise.
func CheckPassword(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("password mismatch: %w", err)
	}
	return nil
}
