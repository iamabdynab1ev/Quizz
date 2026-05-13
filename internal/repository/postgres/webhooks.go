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

type webhookRowScanner interface {
	Scan(dest ...any) error
}

type WebhookRepository struct {
	pool *pgxpool.Pool
}

func NewWebhookRepository(pool *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{pool: pool}
}

func (r *WebhookRepository) Create(ctx context.Context, params domain.CreateWebhookParams) (domain.Webhook, error) {
	webhook, err := scanWebhookRow(r.pool.QueryRow(ctx, `
		INSERT INTO webhooks (
			name,
			url,
			events,
			secret,
			status
		) VALUES (
			$1, $2, $3::app_event_type[], $4, $5
		)
		RETURNING id, name, url, events, secret, status, last_triggered_at, last_status_code, last_error, deliveries, failures, created_at, updated_at
	`,
		params.Name,
		params.URL,
		appEventTypesToStrings(params.Events),
		params.Secret,
		string(params.Status),
	))
	if err != nil {
		return domain.Webhook{}, wrapPGError("repository postgres webhooks create", err)
	}

	return webhook, nil
}

func (r *WebhookRepository) GetByID(ctx context.Context, webhookID string) (domain.Webhook, error) {
	webhook, err := scanWebhookRow(r.pool.QueryRow(ctx, `
		SELECT id, name, url, events, secret, status, last_triggered_at, last_status_code, last_error, deliveries, failures, created_at, updated_at
		FROM webhooks
		WHERE id = $1
	`, webhookID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Webhook{}, fmt.Errorf("repository postgres webhooks get by id: %w", domain.ErrNotFound)
		}

		return domain.Webhook{}, fmt.Errorf("repository postgres webhooks get by id: %w", err)
	}

	return webhook, nil
}

func (r *WebhookRepository) List(ctx context.Context, filter domain.WebhookListFilter) ([]domain.Webhook, int, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, name, url, events, secret, status, last_triggered_at, last_status_code, last_error, deliveries, failures, created_at, updated_at,
			COUNT(*) OVER() AS total_count
		FROM webhooks
		WHERE 1 = 1
	`)

	args := make([]any, 0, 3)
	position := 1

	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", position))
		args = append(args, string(*filter.Status))
		position++
	}

	query.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", position, position+1))
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres webhooks list query: %w", err)
	}
	defer rows.Close()

	var total int
	webhooks := make([]domain.Webhook, 0, filter.Limit)
	for rows.Next() {
		webhook, rowTotal, err := scanWebhookRowWithTotal(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres webhooks list scan: %w", err)
		}
		total = rowTotal
		webhooks = append(webhooks, webhook)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres webhooks list rows: %w", err)
	}

	return webhooks, total, nil
}

func (r *WebhookRepository) Update(ctx context.Context, params domain.UpdateWebhookParams) (domain.Webhook, error) {
	webhook, err := scanWebhookRow(r.pool.QueryRow(ctx, `
		UPDATE webhooks
		SET
			name = $2,
			url = $3,
			events = $4::app_event_type[],
			secret = $5,
			status = $6,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, url, events, secret, status, last_triggered_at, last_status_code, last_error, deliveries, failures, created_at, updated_at
	`,
		params.ID,
		params.Name,
		params.URL,
		appEventTypesToStrings(params.Events),
		params.Secret,
		string(params.Status),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Webhook{}, fmt.Errorf("repository postgres webhooks update: %w", domain.ErrNotFound)
		}

		return domain.Webhook{}, wrapPGError("repository postgres webhooks update", err)
	}

	return webhook, nil
}

func (r *WebhookRepository) Delete(ctx context.Context, webhookID string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM webhooks WHERE id = $1`, webhookID)
	if err != nil {
		return fmt.Errorf("repository postgres webhooks delete: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres webhooks delete: %w", domain.ErrNotFound)
	}

	return nil
}

func (r *WebhookRepository) ListActiveByEvent(ctx context.Context, eventType domain.AppEventType) ([]domain.Webhook, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, url, events, secret, status, last_triggered_at, last_status_code, last_error, deliveries, failures, created_at, updated_at
		FROM webhooks
		WHERE status = $1
			AND $2 = ANY(events)
		ORDER BY created_at DESC
	`, string(domain.WebhookStatusActive), string(eventType))
	if err != nil {
		return nil, fmt.Errorf("repository postgres webhooks list active by event query: %w", err)
	}
	defer rows.Close()

	webhooks := make([]domain.Webhook, 0)
	for rows.Next() {
		webhook, err := scanWebhookRow(rows)
		if err != nil {
			return nil, fmt.Errorf("repository postgres webhooks list active by event scan: %w", err)
		}
		webhooks = append(webhooks, webhook)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository postgres webhooks list active by event rows: %w", err)
	}

	return webhooks, nil
}

func (r *WebhookRepository) RecordDeliveryResult(ctx context.Context, webhookID string, attemptedAt time.Time, statusCode *int, lastError *string, delivered bool) error {
	var deliveriesDelta int
	var failuresDelta int
	if delivered {
		deliveriesDelta = 1
	} else {
		failuresDelta = 1
	}

	commandTag, err := r.pool.Exec(ctx, `
		UPDATE webhooks
		SET
			last_triggered_at = $2,
			last_status_code = $3,
			last_error = $4,
			deliveries = deliveries + $5,
			failures = failures + $6,
			updated_at = NOW()
		WHERE id = $1
	`, webhookID, attemptedAt, nullableIntPointerForWrite(statusCode), nullableStringPointerForWrite(lastError), deliveriesDelta, failuresDelta)
	if err != nil {
		return fmt.Errorf("repository postgres webhooks record delivery result: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres webhooks record delivery result: %w", domain.ErrNotFound)
	}

	return nil
}

func scanWebhookRowWithTotal(scanner webhookRowScanner) (domain.Webhook, int, error) {
	var webhook domain.Webhook
	var events []string
	var status string
	var lastTriggeredAt sql.NullTime
	var lastStatusCode sql.NullInt32
	var lastError sql.NullString
	var total int

	if err := scanner.Scan(
		&webhook.ID,
		&webhook.Name,
		&webhook.URL,
		&events,
		&webhook.Secret,
		&status,
		&lastTriggeredAt,
		&lastStatusCode,
		&lastError,
		&webhook.Deliveries,
		&webhook.Failures,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
		&total,
	); err != nil {
		return domain.Webhook{}, 0, err
	}

	webhook.Events = stringsToAppEventTypes(events)
	webhook.Status = domain.WebhookStatus(status)
	webhook.LastTriggeredAt = optionalTime(lastTriggeredAt)
	webhook.LastStatusCode = optionalInt(lastStatusCode)
	webhook.LastError = optionalString(lastError)

	return webhook, total, nil
}

func scanWebhookRow(scanner webhookRowScanner) (domain.Webhook, error) {
	var webhook domain.Webhook
	var events []string
	var status string
	var lastTriggeredAt sql.NullTime
	var lastStatusCode sql.NullInt32
	var lastError sql.NullString

	if err := scanner.Scan(
		&webhook.ID,
		&webhook.Name,
		&webhook.URL,
		&events,
		&webhook.Secret,
		&status,
		&lastTriggeredAt,
		&lastStatusCode,
		&lastError,
		&webhook.Deliveries,
		&webhook.Failures,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	); err != nil {
		return domain.Webhook{}, err
	}

	webhook.Events = stringsToAppEventTypes(events)
	webhook.Status = domain.WebhookStatus(status)
	webhook.LastTriggeredAt = optionalTime(lastTriggeredAt)
	webhook.LastStatusCode = optionalInt(lastStatusCode)
	webhook.LastError = optionalString(lastError)

	return webhook, nil
}
