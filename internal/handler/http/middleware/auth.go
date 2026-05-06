package middleware

import (
	"context"
	"fmt"
	"strings"

	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/appctx"
	"lms-arvand-backend/internal/domain"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (domain.AuthIdentity, error)
}

func RequireAuth(logger *slog.Logger, authenticator Authenticator) func(next nethttp.Handler) nethttp.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			token, ok := extractBearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeMiddlewareError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
				return
			}

			identity, err := authenticator.Authenticate(r.Context(), token)
			if err != nil {
				logger.WarnContext(r.Context(), "запрос отклонён middleware авторизации", slog.String("error", err.Error()))
				writeMiddlewareError(w, nethttp.StatusUnauthorized, "unauthorized", "Сессия недействительна или истекла")
				return
			}

			ctx := appctx.WithAuthIdentity(r.Context(), identity)
			ctx = appctx.WithSessionToken(ctx, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CurrentAuthIdentity(ctx context.Context) (domain.AuthIdentity, bool) {
	return appctx.CurrentAuthIdentity(ctx)
}

func CurrentSessionToken(ctx context.Context) (string, bool) {
	return appctx.CurrentSessionToken(ctx)
}

func RequireRoles(roles ...domain.UserRole) func(next nethttp.Handler) nethttp.Handler {
	allowed := make(map[domain.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			identity, ok := CurrentAuthIdentity(r.Context())
			if !ok {
				writeMiddlewareError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
				return
			}

			if _, exists := allowed[identity.User.Role]; !exists {
				writeMiddlewareError(w, nethttp.StatusForbidden, "forbidden", "Недостаточно прав")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireSuperAdmin() func(next nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			identity, ok := CurrentAuthIdentity(r.Context())
			if !ok {
				writeMiddlewareError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
				return
			}

			if identity.User.Role != domain.UserRoleAdmin || !identity.User.IsSuperAdmin {
				writeMiddlewareError(w, nethttp.StatusForbidden, "forbidden", "Недостаточно прав")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(value string) (string, bool) {
	if value == "" {
		return "", false
	}

	parts := strings.SplitN(strings.TrimSpace(value), " ", 2)
	if len(parts) != 2 {
		return "", false
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

func writeMiddlewareError(w nethttp.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintf(w, `{"error":%q,"message":%q}`, code, message)
}
