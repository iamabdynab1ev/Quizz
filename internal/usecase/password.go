package usecase

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string, cost int) (string, error) {
	trimmed := strings.TrimSpace(password)
	if trimmed == "" {
		return "", fmt.Errorf("usecase password hash: empty password")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(trimmed), cost)
	if err != nil {
		return "", fmt.Errorf("usecase password hash: %w", err)
	}

	return string(hashed), nil
}

func comparePasswordHash(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("usecase password compare: %w", err)
	}

	return nil
}
