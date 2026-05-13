package usecase

import (
	"context"
	"testing"

	"lms-arvand-backend/internal/domain"
)

type userUseCaseRepositoryStub struct {
	createParams domain.CreateUserParams
	updateParams domain.UpdateUserParams
	currentUser  domain.User
}

func (s *userUseCaseRepositoryStub) Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error) {
	s.createParams = params
	return domain.User{
		ID:                 "user-id",
		IsAdmin:            params.IsAdmin,
		IsSuperAdmin:       params.IsSuperAdmin,
		MustChangePassword: params.MustChangePassword,
	}, nil
}

func (s *userUseCaseRepositoryStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	if s.currentUser.ID == "" {
		s.currentUser.ID = userID
	}
	return s.currentUser, nil
}

func (s *userUseCaseRepositoryStub) List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int, error) {
	return nil, 0, nil
}

func (s *userUseCaseRepositoryStub) Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error) {
	s.updateParams = params
	return domain.User{ID: params.ID}, nil
}

func (s *userUseCaseRepositoryStub) Deactivate(ctx context.Context, userID string) error {
	return nil
}

func TestUserUseCaseCreateAdminMarksPasswordAsTemporary(t *testing.T) {
	t.Parallel()

	repository := &userUseCaseRepositoryStub{}
	useCase := NewUserUseCase(repository, 4)
	email := "admin@local.test"
	password := "Admin123!"

	if _, err := useCase.Create(context.Background(), domain.CreateUserParams{
		Email:     &email,
		Password:  &password,
		IsAdmin:   true,
		FirstName: "Admin",
		LastName:  "User",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if !repository.createParams.MustChangePassword {
		t.Fatal("Create() must_change_password = false, want true")
	}
}

func TestUserUseCaseCreateSuperAdminDoesNotMarkPasswordAsTemporary(t *testing.T) {
	t.Parallel()

	repository := &userUseCaseRepositoryStub{}
	useCase := NewUserUseCase(repository, 4)
	email := "super@local.test"
	password := "Admin123!"

	if _, err := useCase.Create(context.Background(), domain.CreateUserParams{
		Email:        &email,
		Password:     &password,
		IsAdmin:      true,
		IsSuperAdmin: true,
		FirstName:    "Super",
		LastName:     "Admin",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repository.createParams.MustChangePassword {
		t.Fatal("Create() must_change_password = true, want false")
	}
}

func TestUserUseCaseUpdateAdminPasswordMarksPasswordAsTemporary(t *testing.T) {
	t.Parallel()

	repository := &userUseCaseRepositoryStub{
		currentUser: domain.User{ID: "admin-id", IsAdmin: true},
	}
	useCase := NewUserUseCase(repository, 4)
	email := "admin@local.test"
	password := "Admin123!"

	if _, err := useCase.Update(context.Background(), domain.UpdateUserParams{
		ID:        "admin-id",
		Email:     &email,
		Password:  &password,
		IsAdmin:   true,
		FirstName: "Admin",
		LastName:  "User",
		IsActive:  true,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if repository.updateParams.MustChangePassword == nil || !*repository.updateParams.MustChangePassword {
		t.Fatalf("Update() must_change_password = %#v, want true", repository.updateParams.MustChangePassword)
	}
}
