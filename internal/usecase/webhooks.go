package usecase

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type webhookRepository interface {
	Create(ctx context.Context, params domain.CreateWebhookParams) (domain.Webhook, error)
	GetByID(ctx context.Context, webhookID string) (domain.Webhook, error)
	List(ctx context.Context, filter domain.WebhookListFilter) ([]domain.Webhook, int, error)
	Update(ctx context.Context, params domain.UpdateWebhookParams) (domain.Webhook, error)
	Delete(ctx context.Context, webhookID string) error
}

type WebhookUseCase struct {
	repository webhookRepository
}

func NewWebhookUseCase(repository webhookRepository) *WebhookUseCase {
	return &WebhookUseCase{repository: repository}
}

func (u *WebhookUseCase) Create(ctx context.Context, params domain.CreateWebhookParams) (domain.Webhook, error) {
	normalized, err := normalizeCreateWebhookParams(params)
	if err != nil {
		return domain.Webhook{}, fmt.Errorf("usecase webhooks create: %w", err)
	}

	webhook, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.Webhook{}, fmt.Errorf("usecase webhooks create: %w", err)
	}

	return webhook, nil
}

func (u *WebhookUseCase) GetByID(ctx context.Context, webhookID string) (domain.Webhook, error) {
	webhookID = strings.TrimSpace(webhookID)
	if webhookID == "" {
		return domain.Webhook{}, fmt.Errorf("usecase webhooks get by id: %w", domain.ErrValidation)
	}

	webhook, err := u.repository.GetByID(ctx, webhookID)
	if err != nil {
		return domain.Webhook{}, fmt.Errorf("usecase webhooks get by id: %w", err)
	}

	return webhook, nil
}

func (u *WebhookUseCase) List(ctx context.Context, filter domain.WebhookListFilter) ([]domain.Webhook, int, error) {
	normalized, err := normalizeWebhookListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase webhooks list: %w", err)
	}

	webhooks, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase webhooks list: %w", err)
	}

	return webhooks, total, nil
}

func (u *WebhookUseCase) Update(ctx context.Context, params domain.UpdateWebhookParams) (domain.Webhook, error) {
	normalized, err := normalizeUpdateWebhookParams(params)
	if err != nil {
		return domain.Webhook{}, fmt.Errorf("usecase webhooks update: %w", err)
	}

	webhook, err := u.repository.Update(ctx, normalized)
	if err != nil {
		return domain.Webhook{}, fmt.Errorf("usecase webhooks update: %w", err)
	}

	return webhook, nil
}

func (u *WebhookUseCase) Delete(ctx context.Context, webhookID string) error {
	webhookID = strings.TrimSpace(webhookID)
	if webhookID == "" {
		return fmt.Errorf("usecase webhooks delete: %w", domain.ErrValidation)
	}

	if err := u.repository.Delete(ctx, webhookID); err != nil {
		return fmt.Errorf("usecase webhooks delete: %w", err)
	}

	return nil
}

func normalizeCreateWebhookParams(params domain.CreateWebhookParams) (domain.CreateWebhookParams, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.URL = strings.TrimSpace(params.URL)
	params.Secret = strings.TrimSpace(params.Secret)

	if params.Name == "" || params.URL == "" || params.Secret == "" {
		return domain.CreateWebhookParams{}, fmt.Errorf("name, url, and secret are required: %w", domain.ErrValidation)
	}

	if _, err := url.ParseRequestURI(params.URL); err != nil {
		return domain.CreateWebhookParams{}, fmt.Errorf("url is invalid: %w", domain.ErrValidation)
	}

	if params.Status == "" {
		params.Status = domain.WebhookStatusActive
	}
	if !params.Status.IsValid() {
		return domain.CreateWebhookParams{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
	}

	events, err := normalizeAppEventTypes(params.Events)
	if err != nil {
		return domain.CreateWebhookParams{}, fmt.Errorf("events are invalid: %w", err)
	}
	params.Events = events

	return params, nil
}

func normalizeUpdateWebhookParams(params domain.UpdateWebhookParams) (domain.UpdateWebhookParams, error) {
	params.ID = strings.TrimSpace(params.ID)
	params.Name = strings.TrimSpace(params.Name)
	params.URL = strings.TrimSpace(params.URL)
	params.Secret = strings.TrimSpace(params.Secret)

	if params.ID == "" || params.Name == "" || params.URL == "" || params.Secret == "" {
		return domain.UpdateWebhookParams{}, fmt.Errorf("id, name, url, and secret are required: %w", domain.ErrValidation)
	}

	if _, err := url.ParseRequestURI(params.URL); err != nil {
		return domain.UpdateWebhookParams{}, fmt.Errorf("url is invalid: %w", domain.ErrValidation)
	}

	if !params.Status.IsValid() {
		return domain.UpdateWebhookParams{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
	}

	events, err := normalizeAppEventTypes(params.Events)
	if err != nil {
		return domain.UpdateWebhookParams{}, fmt.Errorf("events are invalid: %w", err)
	}
	params.Events = events

	return params, nil
}

func normalizeWebhookListFilter(filter domain.WebhookListFilter) (domain.WebhookListFilter, error) {
	if filter.Status != nil {
		status := domain.WebhookStatus(strings.TrimSpace(string(*filter.Status)))
		if !status.IsValid() {
			return domain.WebhookListFilter{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
		}
		filter.Status = &status
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		return domain.WebhookListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}
