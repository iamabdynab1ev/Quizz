package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditLogRowScanner interface {
	Scan(dest ...any) error
}

type AuditLogRepository struct {
	pool *pgxpool.Pool
}

func NewAuditLogRepository(pool *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{pool: pool}
}

func (r *AuditLogRepository) Create(ctx context.Context, params domain.CreateAuditLogParams) (domain.AuditLog, error) {
	auditLog, err := scanAuditLogRow(r.pool.QueryRow(ctx, `
		INSERT INTO audit_logs (
			type,
			actor_id,
			payload
		) VALUES (
			$1, $2::uuid, $3::jsonb
		)
		RETURNING id, type, at, actor_id::text, payload
	`,
		string(params.Type),
		nullableStringPointerForWrite(params.ActorID),
		[]byte(params.Payload),
	))
	if err != nil {
		return domain.AuditLog{}, wrapPGError("repository postgres audit logs create", err)
	}

	return auditLog, nil
}

func (r *AuditLogRepository) GetByID(ctx context.Context, auditLogID string) (domain.AuditLog, error) {
	auditLog, err := scanAuditLogRow(r.pool.QueryRow(ctx, `
		SELECT id, type, at, actor_id::text, payload
		FROM audit_logs
		WHERE id = $1
	`, auditLogID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AuditLog{}, fmt.Errorf("repository postgres audit logs get by id: %w", domain.ErrNotFound)
		}

		return domain.AuditLog{}, fmt.Errorf("repository postgres audit logs get by id: %w", err)
	}

	return auditLog, nil
}

func (r *AuditLogRepository) List(ctx context.Context, filter domain.AuditLogListFilter) ([]domain.AuditLog, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT id, type, at, actor_id::text, payload
				FROM audit_logs
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM audit_logs
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 4)
		position := 1

		if filter.Type != nil {
			query.WriteString(fmt.Sprintf(" AND type = $%d", position))
			args = append(args, string(*filter.Type))
			position++
		}

		if filter.ActorID != nil {
			query.WriteString(fmt.Sprintf(" AND actor_id = $%d::uuid", position))
			args = append(args, *filter.ActorID)
			position++
		}

		if includePagination {
			query.WriteString(fmt.Sprintf(" ORDER BY at DESC LIMIT $%d OFFSET $%d", position, position+1))
			args = append(args, filter.Limit, filter.Offset)
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres audit logs list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres audit logs list query: %w", err)
	}
	defer rows.Close()

	auditLogs := make([]domain.AuditLog, 0, filter.Limit)
	for rows.Next() {
		auditLog, err := scanAuditLogRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres audit logs list scan: %w", err)
		}
		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres audit logs list rows: %w", err)
	}

	return auditLogs, total, nil
}

func (r *AuditLogRepository) ClaimWebhookDispatchBatch(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 {
		limit = 1
	}

	rows, err := r.pool.Query(ctx, `
		WITH picked AS (
			SELECT id
			FROM audit_logs
			WHERE webhook_dispatched_at IS NULL
				AND webhook_next_dispatch_at <= NOW()
				AND (webhook_processing_at IS NULL OR webhook_processing_at < NOW() - INTERVAL '2 minutes')
			ORDER BY at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE audit_logs a
		SET
			webhook_processing_at = NOW(),
			webhook_dispatch_attempts = webhook_dispatch_attempts + 1
		FROM picked
		WHERE a.id = picked.id
		RETURNING
			a.id,
			a.type,
			a.at,
			a.actor_id::text,
			a.payload,
			a.webhook_dispatch_attempts,
			a.webhook_next_dispatch_at
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("repository postgres audit logs claim webhook dispatch batch query: %w", err)
	}
	defer rows.Close()

	auditLogs := make([]domain.AuditLog, 0, limit)
	for rows.Next() {
		auditLog, err := scanAuditLogDispatchRow(rows)
		if err != nil {
			return nil, fmt.Errorf("repository postgres audit logs claim webhook dispatch batch scan: %w", err)
		}
		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository postgres audit logs claim webhook dispatch batch rows: %w", err)
	}

	return auditLogs, nil
}

func (r *AuditLogRepository) MarkWebhookDispatchSuccess(ctx context.Context, auditLogID string) error {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE audit_logs
		SET
			webhook_dispatched_at = NOW(),
			webhook_processing_at = NULL,
			webhook_last_dispatch_error = NULL
		WHERE id = $1
	`, auditLogID)
	if err != nil {
		return fmt.Errorf("repository postgres audit logs mark webhook dispatch success: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres audit logs mark webhook dispatch success: %w", domain.ErrNotFound)
	}

	return nil
}

func (r *AuditLogRepository) MarkWebhookDispatchFailure(ctx context.Context, auditLogID string, nextAttemptAt time.Time, lastError string) error {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE audit_logs
		SET
			webhook_processing_at = NULL,
			webhook_next_dispatch_at = $2,
			webhook_last_dispatch_error = $3
		WHERE id = $1
	`, auditLogID, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("repository postgres audit logs mark webhook dispatch failure: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres audit logs mark webhook dispatch failure: %w", domain.ErrNotFound)
	}

	return nil
}

func scanAuditLogRow(scanner auditLogRowScanner) (domain.AuditLog, error) {
	var auditLog domain.AuditLog
	var eventType string
	var actorID sql.NullString

	if err := scanner.Scan(
		&auditLog.ID,
		&eventType,
		&auditLog.At,
		&actorID,
		&auditLog.Payload,
	); err != nil {
		return domain.AuditLog{}, err
	}

	auditLog.Type = domain.AppEventType(eventType)
	auditLog.ActorID = optionalString(actorID)
	return auditLog, nil
}

func scanAuditLogDispatchRow(scanner auditLogRowScanner) (domain.AuditLog, error) {
	var auditLog domain.AuditLog
	var eventType string
	var actorID sql.NullString
	var attempts int
	var nextDispatchAt sql.NullTime

	if err := scanner.Scan(
		&auditLog.ID,
		&eventType,
		&auditLog.At,
		&actorID,
		&auditLog.Payload,
		&attempts,
		&nextDispatchAt,
	); err != nil {
		return domain.AuditLog{}, err
	}

	auditLog.Type = domain.AppEventType(eventType)
	auditLog.ActorID = optionalString(actorID)
	auditLog.WebhookDispatchAttempts = attempts
	auditLog.WebhookNextDispatchAt = optionalTime(nextDispatchAt)
	return auditLog, nil
}
