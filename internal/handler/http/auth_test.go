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

func (s *authHandlerUseCaseStub) Register(context.Context, domain.RegisterParams) (domain.LoginResult, error) {
	return domain.LoginResult{}, nil
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

func (s *authHandlerUseCaseStub) UpdateProfile(context.Context, domain.UpdateProfileParams) (domain.User, error) {
	return domain.User{}, nil
}

func (s *authHandlerUseCaseStub) ChangePassword(context.Context, domain.ChangePasswordParams) error {
	return nil
}

func (s *authHandlerUseCaseStub) ForgotPassword(context.Context, domain.ForgotPasswordParams) (domain.ForgotPasswordResult, error) {
	return domain.ForgotPasswordResult{}, nil
}

func (s *authHandlerUseCaseStub) ResetPassword(context.Context, domain.ResetPasswordParams) error {
	return nil
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
				User:      domain.User{ID: "user-id", Role: domain.UserRoleAdmin},
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
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"is_admin":true`)) {
		t.Fatalf("expected response to expose is_admin, got %s", rr.Body.String())
	}
	for _, forbidden := range []string{`"username"`, `"role"`, `"gender"`} {
		if bytes.Contains(rr.Body.Bytes(), []byte(forbidden)) {
			t.Fatalf("expected response not to expose %s, got %s", forbidden, rr.Body.String())
		}
	}
}

func TestAuthHandlerGoogleConfig(t *testing.T) {
	t.Parallel()

	handler := NewAuthHandler(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&authHandlerUseCaseStub{},
		"google-client-id.apps.googleusercontent.com",
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/config", nil)
	rr := httptest.NewRecorder()

	handler.GoogleConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if !bytes.Contains(rr.Body.Bytes(), []byte(`"enabled":true`)) {
		t.Fatalf("expected google config to be enabled, got %s", rr.Body.String())
	}

	if !bytes.Contains(rr.Body.Bytes(), []byte(`"client_id":"google-client-id.apps.googleusercontent.com"`)) {
		t.Fatalf("expected google client id in response, got %s", rr.Body.String())
	}
}
