package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"time"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type webhookUseCase interface {
	Create(ctx context.Context, params domain.CreateWebhookParams) (domain.Webhook, error)
	GetByID(ctx context.Context, webhookID string) (domain.Webhook, error)
	List(ctx context.Context, filter domain.WebhookListFilter) ([]domain.Webhook, int, error)
	Update(ctx context.Context, params domain.UpdateWebhookParams) (domain.Webhook, error)
	Delete(ctx context.Context, webhookID string) error
}

type WebhooksHandler struct {
	logger  *slog.Logger
	useCase webhookUseCase
}

type webhookResponse struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	URL             string                `json:"url"`
	Events          []domain.AppEventType `json:"events"`
	Secret          *string               `json:"secret,omitempty"`
	Status          domain.WebhookStatus  `json:"status"`
	LastTriggeredAt *time.Time            `json:"last_triggered_at,omitempty"`
	LastStatusCode  *int                  `json:"last_status_code,omitempty"`
	LastError       *string               `json:"last_error,omitempty"`
	Deliveries      int64                 `json:"deliveries"`
	Failures        int64                 `json:"failures"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

func NewWebhooksHandler(logger *slog.Logger, useCase webhookUseCase) *WebhooksHandler {
	return &WebhooksHandler{logger: logger, useCase: useCase}
}

func toWebhookResponse(webhook domain.Webhook, includeSecret bool) webhookResponse {
	response := webhookResponse{
		ID:              webhook.ID,
		Name:            webhook.Name,
		URL:             webhook.URL,
		Events:          append([]domain.AppEventType(nil), webhook.Events...),
		Status:          webhook.Status,
		LastTriggeredAt: webhook.LastTriggeredAt,
		LastStatusCode:  webhook.LastStatusCode,
		LastError:       webhook.LastError,
		Deliveries:      webhook.Deliveries,
		Failures:        webhook.Failures,
		CreatedAt:       webhook.CreatedAt,
		UpdatedAt:       webhook.UpdatedAt,
	}

	if includeSecret {
		secret := webhook.Secret
		response.Secret = &secret
	}

	return response
}

func (h *WebhooksHandler) CreateWebhook(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateWebhookParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	webhook, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create webhook failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, toWebhookResponse(webhook, true)); err != nil {
		h.logger.ErrorContext(r.Context(), "create webhook response failed", slog.String("error", err.Error()))
	}
}

func (h *WebhooksHandler) GetWebhookByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	webhook, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "webhookID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get webhook failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toWebhookResponse(webhook, false)); err != nil {
		h.logger.ErrorContext(r.Context(), "get webhook response failed", slog.String("error", err.Error()))
	}
}

func (h *WebhooksHandler) ListWebhooks(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseWebhookListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	webhooks, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list webhooks failed", slog.String("error", err.Error()))
		}
		return
	}

	response := make([]webhookResponse, 0, len(webhooks))
	for _, webhook := range webhooks {
		response = append(response, toWebhookResponse(webhook, false))
	}

	if err := writePagedJSON(w, nethttp.StatusOK, response, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list webhooks response failed", slog.String("error", err.Error()))
	}
}

func (h *WebhooksHandler) UpdateWebhook(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.UpdateWebhookParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	params.ID = chi.URLParam(r, "webhookID")

	webhook, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update webhook failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toWebhookResponse(webhook, false)); err != nil {
		h.logger.ErrorContext(r.Context(), "update webhook response failed", slog.String("error", err.Error()))
	}
}

func (h *WebhooksHandler) DeleteWebhook(w nethttp.ResponseWriter, r *nethttp.Request) {
	if err := h.useCase.Delete(r.Context(), chi.URLParam(r, "webhookID")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "delete webhook failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *WebhooksHandler) parseWebhookListFilter(r *nethttp.Request) (domain.WebhookListFilter, error) {
	query := r.URL.Query()
	filter := domain.WebhookListFilter{}

	if statusValue := query.Get("status"); statusValue != "" {
		status := domain.WebhookStatus(statusValue)
		filter.Status = &status
	}
	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.WebhookListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}
	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.WebhookListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
