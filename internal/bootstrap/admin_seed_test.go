package bootstrap

import (
	"context"
	"errors"
	"testing"

	"lms-arvand-backend/internal/config"
	"lms-arvand-backend/internal/domain"
)

type stubAdminLookup struct {
	user domain.User
	err  error
}

func (s stubAdminLookup) GetByLogin(ctx context.Context, identifier string) (domain.User, error) {
	return s.user, s.err
}

type stubAdminWriter struct {
	createFn func(ctx context.Context, params domain.CreateUserParams) (domain.User, error)
	updateFn func(ctx context.Context, params domain.UpdateUserParams) (domain.User, error)
}

func (s stubAdminWriter) Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error) {
	return s.createFn(ctx, params)
}

func (s stubAdminWriter) Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error) {
	return s.updateFn(ctx, params)
}

func TestAdminSeederSeedCreatesMissingAdmin(t *testing.T) {
	t.Parallel()

	seeder := NewAdminSeeder(
		stubAdminLookup{err: domain.ErrNotFound},
		stubAdminWriter{
			createFn: func(ctx context.Context, params domain.CreateUserParams) (domain.User, error) {
				if params.Role != domain.UserRoleAdmin {
					t.Fatalf("Create() role = %q, want %q", params.Role, domain.UserRoleAdmin)
				}

				if params.AdminInfo == nil || len(params.AdminInfo.Permissions) != 1 || params.AdminInfo.Permissions[0] != "*" {
					t.Fatalf("Create() permissions = %#v, want [*]", params.AdminInfo)
				}

				return domain.User{ID: "seeded-admin"}, nil
			},
			updateFn: func(ctx context.Context, params domain.UpdateUserParams) (domain.User, error) {
				t.Fatal("Update() should not be called")
				return domain.User{}, nil
			},
		},
	)

	user, err := seeder.Seed(context.Background(), config.SeedAdminConfig{
		Username:     "admin",
		Email:        "admin@local.test",
		Password:     "Admin123!",
		FirstName:    "System",
		LastName:     "Admin",
		IsSuperAdmin: true,
		Permissions:  []string{"*"},
	})
	if err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	if user.ID != "seeded-admin" {
		t.Fatalf("Seed() user.ID = %q, want %q", user.ID, "seeded-admin")
	}
}

func TestAdminSeederSeedUpdatesExistingAdmin(t *testing.T) {
	t.Parallel()

	seeder := NewAdminSeeder(
		stubAdminLookup{
			user: domain.User{
				ID:        "admin-id",
				Username:  "admin",
				Role:      domain.UserRoleAdmin,
				FirstName: "Old",
				LastName:  "Admin",
				Gender:    domain.GenderUnspecified,
			},
		},
		stubAdminWriter{
			createFn: func(ctx context.Context, params domain.CreateUserParams) (domain.User, error) {
				t.Fatal("Create() should not be called")
				return domain.User{}, nil
			},
			updateFn: func(ctx context.Context, params domain.UpdateUserParams) (domain.User, error) {
				if params.ID != "admin-id" {
					t.Fatalf("Update() id = %q, want %q", params.ID, "admin-id")
				}

				if params.Role != domain.UserRoleAdmin {
					t.Fatalf("Update() role = %q, want %q", params.Role, domain.UserRoleAdmin)
				}

				if !params.IsActive {
					t.Fatal("Update() is_active = false, want true")
				}

				return domain.User{ID: params.ID}, nil
			},
		},
	)

	user, err := seeder.Seed(context.Background(), config.SeedAdminConfig{
		Username:     "admin",
		Email:        "admin@local.test",
		Password:     "Admin123!",
		FirstName:    "System",
		LastName:     "Admin",
		IsSuperAdmin: true,
		Permissions:  []string{"*"},
	})
	if err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	if user.ID != "admin-id" {
		t.Fatalf("Seed() user.ID = %q, want %q", user.ID, "admin-id")
	}
}

func TestAdminSeederSeedFailsOnUnexpectedLookupError(t *testing.T) {
	t.Parallel()

	seeder := NewAdminSeeder(
		stubAdminLookup{err: errors.New("db failed")},
		stubAdminWriter{},
	)

	if _, err := seeder.Seed(context.Background(), config.SeedAdminConfig{
		Username:     "admin",
		Email:        "admin@local.test",
		Password:     "Admin123!",
		FirstName:    "System",
		LastName:     "Admin",
		IsSuperAdmin: true,
		Permissions:  []string{"*"},
	}); err == nil {
		t.Fatal("Seed() error = nil, want non-nil")
	}
}
