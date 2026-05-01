package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LoginAttemptRepository struct {
	pool *pgxpool.Pool
}

func NewLoginAttemptRepository(pool *pgxpool.Pool) *LoginAttemptRepository {
	return &LoginAttemptRepository{pool: pool}
}

func (r *LoginAttemptRepository) CountRecentFailed(ctx context.Context, identifier string, ipAddress *string, since time.Time) (int, error) {
	var count int

	if ipAddress != nil && *ipAddress != "" {
		if err := r.pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM login_attempts
			WHERE succeeded = false
				AND attempted_at >= $3
				AND (identifier = $1 OR ip_address = $2)
		`, identifier, *ipAddress, since).Scan(&count); err != nil {
			return 0, fmt.Errorf("repository postgres login attempts count recent failed: %w", err)
		}

		return count, nil
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM login_attempts
		WHERE succeeded = false
			AND attempted_at >= $2
			AND identifier = $1
	`, identifier, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("repository postgres login attempts count recent failed: %w", err)
	}

	return count, nil
}

func (r *LoginAttemptRepository) Record(ctx context.Context, identifier string, ipAddress *string, succeeded bool) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO login_attempts (identifier, ip_address, succeeded)
		VALUES ($1, $2, $3)
	`, identifier, nullableStringPointerForWrite(ipAddress), succeeded); err != nil {
		return fmt.Errorf("repository postgres login attempts record: %w", err)
	}

	return nil
}
