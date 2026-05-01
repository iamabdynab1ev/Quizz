package http

import (
	"context"
	"log/slog"
	"net"
	nethttp "net/http"
	"strings"

	"lms-arvand-backend/internal/domain"
	"lms-arvand-backend/internal/handler/http/middleware"
)

type authUseCase interface {
	Login(ctx context.Context, params domain.LoginParams) (domain.LoginResult, error)
	LoginWithGoogle(ctx context.Context, params domain.GoogleLoginParams) (domain.LoginResult, error)
	Authenticate(ctx context.Context, token string) (domain.AuthIdentity, error)
	Logout(ctx context.Context, token string) error
}

type AuthHandler struct {
	logger  *slog.Logger
	useCase authUseCase
}

func NewAuthHandler(logger *slog.Logger, useCase authUseCase) *AuthHandler {
	return &AuthHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *AuthHandler) Login(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}

	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	result, err := h.useCase.Login(r.Context(), domain.LoginParams{
		Identifier: request.Identifier,
		Password:   request.Password,
		IPAddress:  requestIP(r),
		UserAgent:  normalizeUserAgent(r.UserAgent()),
	})
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "login failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, result); err != nil {
		h.logger.ErrorContext(r.Context(), "login response failed", slog.String("error", err.Error()))
	}
}

func (h *AuthHandler) LoginWithGoogle(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request struct {
		IDToken string `json:"id_token"`
	}

	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	result, err := h.useCase.LoginWithGoogle(r.Context(), domain.GoogleLoginParams{
		IDToken:   request.IDToken,
		IPAddress: requestIP(r),
		UserAgent: normalizeUserAgent(r.UserAgent()),
	})
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusUnauthorized || status == nethttp.StatusServiceUnavailable {
			h.logger.ErrorContext(r.Context(), "google login failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, result); err != nil {
		h.logger.ErrorContext(r.Context(), "google login response failed", slog.String("error", err.Error()))
	}
}

func (h *AuthHandler) Logout(w nethttp.ResponseWriter, r *nethttp.Request) {
	token, ok := middleware.CurrentSessionToken(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	if err := h.useCase.Logout(r.Context(), token); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "logout failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
	identity, ok := middleware.CurrentAuthIdentity(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	response := struct {
		User    domain.User    `json:"user"`
		Session domain.Session `json:"session"`
	}{
		User:    identity.User,
		Session: identity.Session,
	}

	if err := writeJSON(w, nethttp.StatusOK, response); err != nil {
		h.logger.ErrorContext(r.Context(), "me response failed", slog.String("error", err.Error()))
	}
}

func requestIP(r *nethttp.Request) *string {
	if r == nil {
		return nil
	}

	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return nil
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return nil
	}

	return &host
}

func normalizeUserAgent(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
