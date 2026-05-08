package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/config"
	"lms-arvand-backend/internal/domain"
)

type adminUserLookup interface {
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}

type adminUserWriter interface {
	Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error)
	Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error)
}

type AdminSeeder struct {
	lookup adminUserLookup
	writer adminUserWriter
}

func NewAdminSeeder(lookup adminUserLookup, writer adminUserWriter) *AdminSeeder {
	return &AdminSeeder{
		lookup: lookup,
		writer: writer,
	}
}

func (s *AdminSeeder) Seed(ctx context.Context, cfg config.SeedAdminConfig) (domain.User, error) {
	normalized, err := normalizeAdminSeedConfig(cfg)
	if err != nil {
		return domain.User{}, fmt.Errorf("bootstrap admin seed normalize config: %w", err)
	}

	existingUser, err := s.lookup.GetByEmail(ctx, normalized.Email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, fmt.Errorf("bootstrap admin seed get by email: %w", err)
	}

	if errors.Is(err, domain.ErrNotFound) {
		user, createErr := s.writer.Create(ctx, buildCreateAdminParams(normalized))
		if createErr != nil {
			return domain.User{}, fmt.Errorf("bootstrap admin seed create: %w", createErr)
		}

		return user, nil
	}

	user, updateErr := s.writer.Update(ctx, buildUpdateAdminParams(existingUser, normalized))
	if updateErr != nil {
		return domain.User{}, fmt.Errorf("bootstrap admin seed update: %w", updateErr)
	}

	return user, nil
}

func normalizeAdminSeedConfig(cfg config.SeedAdminConfig) (config.SeedAdminConfig, error) {
	cfg.Email = strings.TrimSpace(strings.ToLower(cfg.Email))
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.FirstName = strings.TrimSpace(cfg.FirstName)
	cfg.LastName = strings.TrimSpace(cfg.LastName)
	cfg.Patronymic = strings.TrimSpace(cfg.Patronymic)

	if cfg.Password == "" {
		return config.SeedAdminConfig{}, fmt.Errorf("password is required: %w", domain.ErrValidation)
	}

	if cfg.Email == "" {
		return config.SeedAdminConfig{}, fmt.Errorf("email is required: %w", domain.ErrValidation)
	}

	if cfg.FirstName == "" {
		return config.SeedAdminConfig{}, fmt.Errorf("first name is required: %w", domain.ErrValidation)
	}

	if cfg.LastName == "" {
		return config.SeedAdminConfig{}, fmt.Errorf("last name is required: %w", domain.ErrValidation)
	}

	return cfg, nil
}

func buildCreateAdminParams(cfg config.SeedAdminConfig) domain.CreateUserParams {
	email := cfg.Email
	password := cfg.Password

	return domain.CreateUserParams{
		Email:        &email,
		Password:     &password,
		IsAdmin:      true,
		IsSuperAdmin: true,
		FirstName:    cfg.FirstName,
		LastName:     cfg.LastName,
		Patronymic:   cfg.Patronymic,
	}
}

func buildUpdateAdminParams(existing domain.User, cfg config.SeedAdminConfig) domain.UpdateUserParams {
	email := cfg.Email
	password := cfg.Password
	isSuperAdmin := true

	return domain.UpdateUserParams{
		ID:           existing.ID,
		Email:        &email,
		GoogleID:     existing.GoogleID,
		Password:     &password,
		IsAdmin:      true,
		IsSuperAdmin: &isSuperAdmin,
		FirstName:    cfg.FirstName,
		LastName:     cfg.LastName,
		Patronymic:   cfg.Patronymic,
		Phone:        existing.Phone,
		IsMale:       existing.IsMale,
		BirthDate:    existing.BirthDate,
		Address:      existing.Address,
		City:         existing.City,
		AvatarURL:    existing.AvatarURL,
		IsActive:     true,
	}
}
