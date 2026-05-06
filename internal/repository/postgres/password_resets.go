package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PasswordResetRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordResetRepository(pool *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{pool: pool}
}

func (r *PasswordResetRepository) Create(ctx context.Context, params domain.CreatePasswordResetTokenParams) (domain.PasswordResetToken, error) {
	token, err := scanPasswordResetToken(r.pool.QueryRow(ctx, `
		INSERT INTO password_reset_tokens (
			user_id,
			token_hash,
			expires_at
		) VALUES (
			$1, $2, $3
		)
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at
	`, params.UserID, params.TokenHash, params.ExpiresAt))
	if err != nil {
		return domain.PasswordResetToken{}, wrapPGError("repository postgres password resets create", err)
	}

	return token, nil
}

func (r *PasswordResetRepository) GetValidByTokenHash(ctx context.Context, tokenHash string, now time.Time) (domain.PasswordResetToken, error) {
	token, err := scanPasswordResetToken(r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
			AND used_at IS NULL
			AND expires_at > $2
		LIMIT 1
	`, tokenHash, now))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PasswordResetToken{}, fmt.Errorf("repository postgres password resets get valid: %w", domain.ErrNotFound)
		}
		return domain.PasswordResetToken{}, fmt.Errorf("repository postgres password resets get valid: %w", err)
	}

	return token, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, tokenID string) error {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE id = $1
			AND used_at IS NULL
	`, tokenID)
	if err != nil {
		return fmt.Errorf("repository postgres password resets mark used: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres password resets mark used: %w", domain.ErrNotFound)
	}

	return nil
}

func scanPasswordResetToken(scanner interface{ Scan(dest ...any) error }) (domain.PasswordResetToken, error) {
	var token domain.PasswordResetToken
	var usedAt sql.NullTime
	if err := scanner.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&usedAt,
		&token.CreatedAt,
	); err != nil {
		return domain.PasswordResetToken{}, err
	}

	token.UsedAt = optionalTime(usedAt)
	return token, nil
}
