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
	GetByLogin(ctx context.Context, identifier string) (domain.User, error)
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

	existingUser, err := s.lookup.GetByLogin(ctx, normalized.Username)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, fmt.Errorf("bootstrap admin seed get by login: %w", err)
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
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Email = strings.TrimSpace(cfg.Email)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.FirstName = strings.TrimSpace(cfg.FirstName)
	cfg.LastName = strings.TrimSpace(cfg.LastName)
	cfg.Patronymic = strings.TrimSpace(cfg.Patronymic)
	cfg.Permissions = normalizePermissions(cfg.Permissions)

	if cfg.Username == "" {
		return config.SeedAdminConfig{}, fmt.Errorf("username is required: %w", domain.ErrValidation)
	}

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

	if len(cfg.Permissions) == 0 {
		cfg.Permissions = []string{"*"}
	}

	return cfg, nil
}

func buildCreateAdminParams(cfg config.SeedAdminConfig) domain.CreateUserParams {
	email := cfg.Email
	password := cfg.Password

	return domain.CreateUserParams{
		Username:   cfg.Username,
		Email:      &email,
		Password:   &password,
		Role:       domain.UserRoleAdmin,
		FirstName:  cfg.FirstName,
		LastName:   cfg.LastName,
		Patronymic: cfg.Patronymic,
		Gender:     domain.GenderUnspecified,
		AdminInfo: &domain.AdminInfo{
			IsSuperAdmin: cfg.IsSuperAdmin,
			Permissions:  append([]string(nil), cfg.Permissions...),
		},
	}
}

func buildUpdateAdminParams(existing domain.User, cfg config.SeedAdminConfig) domain.UpdateUserParams {
	email := cfg.Email
	password := cfg.Password
	gender := existing.Gender
	if !gender.IsValid() {
		gender = domain.GenderUnspecified
	}

	adminInfo := &domain.AdminInfo{
		IsSuperAdmin: cfg.IsSuperAdmin,
		Permissions:  append([]string(nil), cfg.Permissions...),
	}

	if existing.AdminInfo != nil {
		adminInfo.LastLoginAt = existing.AdminInfo.LastLoginAt
	}

	return domain.UpdateUserParams{
		ID:           existing.ID,
		Username:     cfg.Username,
		Email:        &email,
		GoogleID:     existing.GoogleID,
		Password:     &password,
		Role:         domain.UserRoleAdmin,
		FirstName:    cfg.FirstName,
		LastName:     cfg.LastName,
		Patronymic:   cfg.Patronymic,
		Phone:        existing.Phone,
		Gender:       gender,
		Address:      existing.Address,
		City:         existing.City,
		AvatarURL:    existing.AvatarURL,
		IsActive:     true,
		EmployeeInfo: nil,
		AdminInfo:    adminInfo,
		StudentInfo:  nil,
		GuestInfo:    nil,
	}
}

func normalizePermissions(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		if _, exists := seen[trimmed]; exists {
			continue
		}

		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}
