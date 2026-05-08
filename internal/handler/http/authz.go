package http

import (
	"context"
	"fmt"

	"lms-arvand-backend/internal/domain"
	"lms-arvand-backend/internal/handler/http/middleware"
)

func currentAuthIdentity(ctx context.Context) (domain.AuthIdentity, error) {
	identity, ok := middleware.CurrentAuthIdentity(ctx)
	if !ok {
		return domain.AuthIdentity{}, fmt.Errorf("missing auth identity: %w", domain.UnauthorizedError("authentication required"))
	}

	return identity, nil
}

func scopeUserID(ctx context.Context, requested *string) (*string, error) {
	identity, err := currentAuthIdentity(ctx)
	if err != nil {
		return nil, err
	}

	if identity.User.IsAdmin || identity.User.IsSuperAdmin {
		return requested, nil
	}

	if requested != nil && *requested != identity.User.ID {
		return nil, fmt.Errorf("user scope mismatch: %w", domain.ForbiddenError("access denied for requested user"))
	}

	userID := identity.User.ID
	return &userID, nil
}

func resolveActorUserID(ctx context.Context, requested *string) (string, error) {
	identity, err := currentAuthIdentity(ctx)
	if err != nil {
		return "", err
	}

	if !identity.User.IsAdmin && !identity.User.IsSuperAdmin {
		if requested != nil && *requested != identity.User.ID {
			return "", fmt.Errorf("user scope mismatch: %w", domain.ForbiddenError("access denied for requested user"))
		}

		return identity.User.ID, nil
	}

	if requested != nil && *requested != "" {
		return *requested, nil
	}

	return identity.User.ID, nil
}

func ensureOwnOrAdmin(ctx context.Context, ownerUserID *string) error {
	identity, err := currentAuthIdentity(ctx)
	if err != nil {
		return err
	}

	if identity.User.IsAdmin || identity.User.IsSuperAdmin {
		return nil
	}

	if ownerUserID == nil || *ownerUserID != identity.User.ID {
		return fmt.Errorf("resource ownership mismatch: %w", domain.ForbiddenError("access denied for requested resource"))
	}

	return nil
}
