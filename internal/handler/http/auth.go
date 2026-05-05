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
	Register(ctx context.Context, params domain.RegisterParams) (domain.LoginResult, error)
	Login(ctx context.Context, params domain.LoginParams) (domain.LoginResult, error)
	LoginWithGoogle(ctx context.Context, params domain.GoogleLoginParams) (domain.LoginResult, error)
	Authenticate(ctx context.Context, token string) (domain.AuthIdentity, error)
	UpdateProfile(ctx context.Context, params domain.UpdateProfileParams) (domain.User, error)
	Logout(ctx context.Context, token string) error
}

type AuthHandler struct {
	logger         *slog.Logger
	useCase        authUseCase
	googleClientID string
}

func NewAuthHandler(logger *slog.Logger, useCase authUseCase, googleClientID ...string) *AuthHandler {
	clientID := ""
	if len(googleClientID) > 0 {
		clientID = strings.TrimSpace(googleClientID[0])
	}

	return &AuthHandler{
		logger:         logger,
		useCase:        useCase,
		googleClientID: clientID,
	}
}

func (h *AuthHandler) Register(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request domain.RegisterParams
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	request.IPAddress = requestIP(r)
	request.UserAgent = normalizeUserAgent(r.UserAgent())

	result, err := h.useCase.Register(r.Context(), request)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "регистрация пользователя не выполнена", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, result); err != nil {
		h.logger.ErrorContext(r.Context(), "ответ регистрации пользователя не отправлен", slog.String("error", err.Error()))
	}
}

func (h *AuthHandler) Login(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request struct {
		Identifier string `json:"identifier"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}

	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	identifier := strings.TrimSpace(request.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(request.Email)
	}

	result, err := h.useCase.Login(r.Context(), domain.LoginParams{
		Identifier: identifier,
		Password:   request.Password,
		IPAddress:  requestIP(r),
		UserAgent:  normalizeUserAgent(r.UserAgent()),
	})
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "вход пользователя не выполнен", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, result); err != nil {
		h.logger.ErrorContext(r.Context(), "ответ входа пользователя не отправлен", slog.String("error", err.Error()))
	}
}

func (h *AuthHandler) LoginWithGoogle(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request struct {
		IDToken string `json:"id_token"`
	}

	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
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
			h.logger.ErrorContext(r.Context(), "вход через Google не выполнен", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, result); err != nil {
		h.logger.ErrorContext(r.Context(), "ответ входа через Google не отправлен", slog.String("error", err.Error()))
	}
}

func (h *AuthHandler) GoogleConfig(w nethttp.ResponseWriter, r *nethttp.Request) {
	response := struct {
		Enabled  bool   `json:"enabled"`
		ClientID string `json:"client_id,omitempty"`
	}{
		Enabled:  h.googleClientID != "",
		ClientID: h.googleClientID,
	}

	if err := writeJSON(w, nethttp.StatusOK, response); err != nil {
		h.logger.ErrorContext(r.Context(), "ответ настроек Google не отправлен", slog.String("error", err.Error()))
	}
}

func (h *AuthHandler) Logout(w nethttp.ResponseWriter, r *nethttp.Request) {
	token, ok := middleware.CurrentSessionToken(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	if err := h.useCase.Logout(r.Context(), token); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "выход пользователя не выполнен", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *AuthHandler) UpdateMe(w nethttp.ResponseWriter, r *nethttp.Request) {
	identity, ok := middleware.CurrentAuthIdentity(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	sessionToken, _ := middleware.CurrentSessionToken(r.Context())

	var request domain.UpdateProfileParams
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	request.UserID = identity.User.ID
	request.SessionToken = sessionToken

	user, err := h.useCase.UpdateProfile(r.Context(), request)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "профиль пользователя не обновлён", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, user); err != nil {
		h.logger.ErrorContext(r.Context(), "ответ профиля пользователя не отправлен", slog.String("error", err.Error()))
	}
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
	identity, ok := middleware.CurrentAuthIdentity(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
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
		h.logger.ErrorContext(r.Context(), "ответ текущего пользователя не отправлен", slog.String("error", err.Error()))
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
