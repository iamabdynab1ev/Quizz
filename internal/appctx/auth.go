package appctx

import (
	"context"

	"lms-arvand-backend/internal/domain"
)

type key string

const (
	authIdentityKey key = "auth_identity"
	authTokenKey    key = "auth_token"
)

func WithAuthIdentity(ctx context.Context, identity domain.AuthIdentity) context.Context {
	return context.WithValue(ctx, authIdentityKey, identity)
}

func WithSessionToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey, token)
}

func CurrentAuthIdentity(ctx context.Context) (domain.AuthIdentity, bool) {
	identity, ok := ctx.Value(authIdentityKey).(domain.AuthIdentity)
	return identity, ok
}

func CurrentSessionToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(authTokenKey).(string)
	return token, ok
}
