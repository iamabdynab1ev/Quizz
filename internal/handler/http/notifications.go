package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type notificationUseCase interface {
	Create(ctx context.Context, params domain.CreateNotificationParams) (domain.Notification, error)
	GetByID(ctx context.Context, notificationID string) (domain.Notification, error)
	List(ctx context.Context, filter domain.NotificationListFilter) ([]domain.Notification, int, error)
	MarkRead(ctx context.Context, notificationID string) (domain.Notification, error)
}

type NotificationsHandler struct {
	logger  *slog.Logger
	useCase notificationUseCase
}

func NewNotificationsHandler(logger *slog.Logger, useCase notificationUseCase) *NotificationsHandler {
	return &NotificationsHandler{logger: logger, useCase: useCase}
}

func (h *NotificationsHandler) CreateNotification(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateNotificationParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	notification, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create notification failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, notification); err != nil {
		h.logger.ErrorContext(r.Context(), "create notification response failed", slog.String("error", err.Error()))
	}
}

func (h *NotificationsHandler) GetNotificationByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	notification, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "notificationID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get notification failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := ensureOwnOrAdmin(r.Context(), &notification.UserID); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "get notification authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, notification); err != nil {
		h.logger.ErrorContext(r.Context(), "get notification response failed", slog.String("error", err.Error()))
	}
}

func (h *NotificationsHandler) ListNotifications(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseNotificationListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	filter.UserID, err = scopeUserID(r.Context(), filter.UserID)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "list notifications authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	notifications, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list notifications failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, notifications, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list notifications response failed", slog.String("error", err.Error()))
	}
}

func (h *NotificationsHandler) MarkRead(w nethttp.ResponseWriter, r *nethttp.Request) {
	current, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "notificationID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "mark notification read preload failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := ensureOwnOrAdmin(r.Context(), &current.UserID); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "mark notification read authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	notification, err := h.useCase.MarkRead(r.Context(), chi.URLParam(r, "notificationID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "mark notification read failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, notification); err != nil {
		h.logger.ErrorContext(r.Context(), "mark notification read response failed", slog.String("error", err.Error()))
	}
}

func (h *NotificationsHandler) parseNotificationListFilter(r *nethttp.Request) (domain.NotificationListFilter, error) {
	query := r.URL.Query()
	filter := domain.NotificationListFilter{}

	if userIDValue := query.Get("user_id"); userIDValue != "" {
		filter.UserID = &userIDValue
	}
	if typeValue := query.Get("type"); typeValue != "" {
		typ := domain.NotificationType(typeValue)
		filter.Type = &typ
	}
	if readValue := query.Get("read"); readValue != "" {
		parsed, err := strconv.ParseBool(readValue)
		if err != nil {
			return domain.NotificationListFilter{}, fmt.Errorf("read must be boolean")
		}
		filter.Read = &parsed
	}
	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.NotificationListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}
	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.NotificationListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
