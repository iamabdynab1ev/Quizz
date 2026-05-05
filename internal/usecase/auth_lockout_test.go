package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
)

type loginAttemptRepositoryStub struct {
	count       int
	called      bool
	scope       string
	identifier  string
	ipAddress   *string
	recordCalls int
}

func (s *loginAttemptRepositoryStub) CountRecentFailed(ctx context.Context, identifier string, ipAddress *string, scope string, since time.Time) (int, error) {
	s.called = true
	s.scope = scope
	s.identifier = identifier
	s.ipAddress = ipAddress
	return s.count, nil
}

func (s *loginAttemptRepositoryStub) Record(ctx context.Context, identifier string, ipAddress *string, succeeded bool) error {
	s.recordCalls++
	return nil
}

func TestAuthUseCaseEnsureLoginAllowedDisabled(t *testing.T) {
	t.Parallel()

	repo := &loginAttemptRepositoryStub{count: 99}
	uc := &AuthUseCase{
		loginAttempts:       repo,
		loginLockoutEnabled: false,
		maxLoginAttempts:    5,
		loginAttemptWindow:  15 * time.Minute,
		loginLockoutScope:   "identifier_ip",
		now:                 time.Now,
	}

	if err := uc.ensureLoginAllowed(context.Background(), "admin@local.test", ptr("127.0.0.1")); err != nil {
		t.Fatalf("expected disabled lockout to allow login, got %v", err)
	}

	if repo.called {
		t.Fatal("expected disabled lockout to skip repository lookup")
	}

	succeeded := false
	uc.recordLoginAttempt(context.Background(), "admin@local.test", ptr("127.0.0.1"), &succeeded)
	if repo.recordCalls != 0 {
		t.Fatal("expected disabled lockout to skip login attempt recording")
	}
}

func TestAuthUseCaseEnsureLoginAllowedUsesConfiguredScope(t *testing.T) {
	t.Parallel()

	repo := &loginAttemptRepositoryStub{count: 4}
	uc := &AuthUseCase{
		loginAttempts:       repo,
		loginLockoutEnabled: true,
		maxLoginAttempts:    5,
		loginAttemptWindow:  15 * time.Minute,
		loginLockoutScope:   "ip",
		now:                 time.Now,
	}

	if err := uc.ensureLoginAllowed(context.Background(), "admin@local.test", ptr("127.0.0.1")); err != nil {
		t.Fatalf("expected login to be allowed below threshold, got %v", err)
	}

	if !repo.called {
		t.Fatal("expected repository lookup")
	}

	if repo.scope != "ip" {
		t.Fatalf("expected scope ip, got %q", repo.scope)
	}
}

func TestAuthUseCaseEnsureLoginAllowedReturnsTooManyRequests(t *testing.T) {
	t.Parallel()

	repo := &loginAttemptRepositoryStub{count: 5}
	uc := &AuthUseCase{
		loginAttempts:       repo,
		loginLockoutEnabled: true,
		maxLoginAttempts:    5,
		loginAttemptWindow:  15 * time.Minute,
		loginLockoutScope:   "identifier_ip",
		now:                 time.Now,
	}

	err := uc.ensureLoginAllowed(context.Background(), "admin@local.test", ptr("127.0.0.1"))
	if err == nil {
		t.Fatal("expected lockout error")
	}

	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}

	if appErr.Status != 429 || appErr.Code != "too_many_attempts" {
		t.Fatalf("expected too_many_attempts 429, got %s %d", appErr.Code, appErr.Status)
	}
}
