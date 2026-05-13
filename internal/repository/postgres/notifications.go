package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type notificationRowScanner interface {
	Scan(dest ...any) error
}

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Create(ctx context.Context, params domain.CreateNotificationParams) (domain.Notification, error) {
	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("repository postgres notifications create title value: %w", err)
	}

	bodyValue, err := multiLangValueOrNil(params.Body)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("repository postgres notifications create body value: %w", err)
	}

	notification, err := scanNotificationRow(r.pool.QueryRow(ctx, `
		INSERT INTO notifications (
			user_id,
			type,
			title,
			body,
			link
		) VALUES (
			$1, $2, $3::jsonb, $4::jsonb, $5
		)
		RETURNING id, user_id, type, title, body, link, read, created_at
	`,
		params.UserID,
		string(params.Type),
		titleValue,
		bodyValue,
		nullableStringPointerForWrite(params.Link),
	))
	if err != nil {
		return domain.Notification{}, wrapPGError("repository postgres notifications create", err)
	}

	return notification, nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, notificationID string) (domain.Notification, error) {
	notification, err := scanNotificationRow(r.pool.QueryRow(ctx, `
		SELECT id, user_id, type, title, body, link, read, created_at
		FROM notifications
		WHERE id = $1
	`, notificationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Notification{}, fmt.Errorf("repository postgres notifications get by id: %w", domain.ErrNotFound)
		}

		return domain.Notification{}, fmt.Errorf("repository postgres notifications get by id: %w", err)
	}

	return notification, nil
}

func (r *NotificationRepository) List(ctx context.Context, filter domain.NotificationListFilter) ([]domain.Notification, int, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, user_id, type, title, body, link, read, created_at,
			COUNT(*) OVER() AS total_count
		FROM notifications
		WHERE 1 = 1
	`)

	args := make([]any, 0, 5)
	position := 1

	if filter.UserID != nil {
		query.WriteString(fmt.Sprintf(" AND user_id = $%d::uuid", position))
		args = append(args, *filter.UserID)
		position++
	}

	if filter.Type != nil {
		query.WriteString(fmt.Sprintf(" AND type = $%d", position))
		args = append(args, string(*filter.Type))
		position++
	}

	if filter.Read != nil {
		query.WriteString(fmt.Sprintf(" AND read = $%d", position))
		args = append(args, *filter.Read)
		position++
	}

	query.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", position, position+1))
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres notifications list query: %w", err)
	}
	defer rows.Close()

	var total int
	notifications := make([]domain.Notification, 0, filter.Limit)
	for rows.Next() {
		notification, rowTotal, err := scanNotificationRowWithTotal(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres notifications list scan: %w", err)
		}
		total = rowTotal
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres notifications list rows: %w", err)
	}

	return notifications, total, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, notificationID string) (domain.Notification, error) {
	notification, err := scanNotificationRow(r.pool.QueryRow(ctx, `
		UPDATE notifications
		SET read = true
		WHERE id = $1
		RETURNING id, user_id, type, title, body, link, read, created_at
	`, notificationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Notification{}, fmt.Errorf("repository postgres notifications mark read: %w", domain.ErrNotFound)
		}

		return domain.Notification{}, fmt.Errorf("repository postgres notifications mark read: %w", err)
	}

	return notification, nil
}

func scanNotificationRow(scanner notificationRowScanner) (domain.Notification, error) {
	var notification domain.Notification
	var notificationType string
	var titleBytes, bodyBytes []byte
	var link sql.NullString

	if err := scanner.Scan(
		&notification.ID, &notification.UserID, &notificationType,
		&titleBytes, &bodyBytes, &link, &notification.Read, &notification.CreatedAt,
	); err != nil {
		return domain.Notification{}, err
	}

	if err := notification.Title.Scan(titleBytes); err != nil {
		return domain.Notification{}, fmt.Errorf("repository postgres scan notification title: %w", err)
	}
	if len(bodyBytes) > 0 {
		if err := notification.Body.Scan(bodyBytes); err != nil {
			return domain.Notification{}, fmt.Errorf("repository postgres scan notification body: %w", err)
		}
	}
	notification.Type = domain.NotificationType(notificationType)
	notification.Link = optionalString(link)
	return notification, nil
}

func scanNotificationRowWithTotal(scanner notificationRowScanner) (domain.Notification, int, error) {
	var notification domain.Notification
	var notificationType string
	var titleBytes, bodyBytes []byte
	var link sql.NullString
	var total int

	if err := scanner.Scan(
		&notification.ID, &notification.UserID, &notificationType,
		&titleBytes, &bodyBytes, &link, &notification.Read, &notification.CreatedAt,
		&total,
	); err != nil {
		return domain.Notification{}, 0, err
	}

	if err := notification.Title.Scan(titleBytes); err != nil {
		return domain.Notification{}, 0, fmt.Errorf("repository postgres scan notification title: %w", err)
	}
	if len(bodyBytes) > 0 {
		if err := notification.Body.Scan(bodyBytes); err != nil {
			return domain.Notification{}, 0, fmt.Errorf("repository postgres scan notification body: %w", err)
		}
	}
	notification.Type = domain.NotificationType(notificationType)
	notification.Link = optionalString(link)
	return notification, total, nil
}
