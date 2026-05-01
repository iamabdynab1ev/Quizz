package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type notificationRepository interface {
	Create(ctx context.Context, params domain.CreateNotificationParams) (domain.Notification, error)
	GetByID(ctx context.Context, notificationID string) (domain.Notification, error)
	List(ctx context.Context, filter domain.NotificationListFilter) ([]domain.Notification, int, error)
	MarkRead(ctx context.Context, notificationID string) (domain.Notification, error)
}

type NotificationUseCase struct {
	repository notificationRepository
}

func NewNotificationUseCase(repository notificationRepository) *NotificationUseCase {
	return &NotificationUseCase{repository: repository}
}

func (u *NotificationUseCase) Create(ctx context.Context, params domain.CreateNotificationParams) (domain.Notification, error) {
	normalized, err := normalizeCreateNotificationParams(params)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("usecase notifications create: %w", err)
	}

	notification, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("usecase notifications create: %w", err)
	}

	return notification, nil
}

func (u *NotificationUseCase) GetByID(ctx context.Context, notificationID string) (domain.Notification, error) {
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return domain.Notification{}, fmt.Errorf("usecase notifications get by id: %w", domain.ErrValidation)
	}

	notification, err := u.repository.GetByID(ctx, notificationID)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("usecase notifications get by id: %w", err)
	}

	return notification, nil
}

func (u *NotificationUseCase) List(ctx context.Context, filter domain.NotificationListFilter) ([]domain.Notification, int, error) {
	normalized, err := normalizeNotificationListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase notifications list: %w", err)
	}

	notifications, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase notifications list: %w", err)
	}

	return notifications, total, nil
}

func (u *NotificationUseCase) MarkRead(ctx context.Context, notificationID string) (domain.Notification, error) {
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return domain.Notification{}, fmt.Errorf("usecase notifications mark read: %w", domain.ErrValidation)
	}

	notification, err := u.repository.MarkRead(ctx, notificationID)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("usecase notifications mark read: %w", err)
	}

	return notification, nil
}

func normalizeCreateNotificationParams(params domain.CreateNotificationParams) (domain.CreateNotificationParams, error) {
	params.UserID = strings.TrimSpace(params.UserID)
	params.Link = normalizeOptionalString(params.Link)

	if params.UserID == "" {
		return domain.CreateNotificationParams{}, fmt.Errorf("user_id is required: %w", domain.ErrValidation)
	}

	if !params.Type.IsValid() {
		return domain.CreateNotificationParams{}, fmt.Errorf("type is invalid: %w", domain.ErrValidation)
	}

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.CreateNotificationParams{}, fmt.Errorf("title is invalid: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeNotificationListFilter(filter domain.NotificationListFilter) (domain.NotificationListFilter, error) {
	if filter.UserID != nil {
		filter.UserID = normalizeOptionalString(filter.UserID)
	}

	if filter.Type != nil {
		typ := domain.NotificationType(strings.TrimSpace(string(*filter.Type)))
		if !typ.IsValid() {
			return domain.NotificationListFilter{}, fmt.Errorf("type is invalid: %w", domain.ErrValidation)
		}
		filter.Type = &typ
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		return domain.NotificationListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}
