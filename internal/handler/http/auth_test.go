package http

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
)

type authHandlerUseCaseStub struct {
	loginFn func(ctx context.Context, params domain.LoginParams) (domain.LoginResult, error)
}

func (s *authHandlerUseCaseStub) Login(ctx context.Context, params domain.LoginParams) (domain.LoginResult, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, params)
	}
	return domain.LoginResult{}, nil
}

func (s *authHandlerUseCaseStub) LoginWithGoogle(context.Context, domain.GoogleLoginParams) (domain.LoginResult, error) {
	return domain.LoginResult{}, nil
}

func (s *authHandlerUseCaseStub) Authenticate(context.Context, string) (domain.AuthIdentity, error) {
	return domain.AuthIdentity{}, nil
}

func (s *authHandlerUseCaseStub) Logout(context.Context, string) error {
	return nil
}

func TestAuthHandlerLoginUsesEmailFallback(t *testing.T) {
	t.Parallel()

	called := false
	stub := &authHandlerUseCaseStub{
		loginFn: func(ctx context.Context, params domain.LoginParams) (domain.LoginResult, error) {
			called = true
			if params.Identifier != "admin@local.test" {
				t.Fatalf("expected identifier to use email, got %q", params.Identifier)
			}
			if params.Password != "Admin123!" {
				t.Fatalf("expected password to pass through unchanged, got %q", params.Password)
			}
			return domain.LoginResult{
				Token:     "session-token",
				ExpiresAt: func() *time.Time { ts := time.Now().Add(time.Hour); return &ts }(),
				User:      domain.User{ID: "user-id", Username: "admin", Role: domain.UserRoleAdmin},
			}, nil
		},
	}

	handler := NewAuthHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), stub)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"admin@local.test","password":"Admin123!"}`))
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if !called {
		t.Fatal("expected login use case to be called")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if !bytes.Contains(rr.Body.Bytes(), []byte(`"token":"session-token"`)) {
		t.Fatalf("expected response to contain token, got %s", rr.Body.String())
	}
}
