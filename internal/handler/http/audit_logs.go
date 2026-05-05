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

type auditLogUseCase interface {
	Create(ctx context.Context, params domain.CreateAuditLogParams) (domain.AuditLog, error)
	GetByID(ctx context.Context, auditLogID string) (domain.AuditLog, error)
	List(ctx context.Context, filter domain.AuditLogListFilter) ([]domain.AuditLog, int, error)
}

type AuditLogsHandler struct {
	logger  *slog.Logger
	useCase auditLogUseCase
}

func NewAuditLogsHandler(logger *slog.Logger, useCase auditLogUseCase) *AuditLogsHandler {
	return &AuditLogsHandler{logger: logger, useCase: useCase}
}

func (h *AuditLogsHandler) CreateAuditLog(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateAuditLogParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	auditLog, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create audit log failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, auditLog); err != nil {
		h.logger.ErrorContext(r.Context(), "create audit log response failed", slog.String("error", err.Error()))
	}
}

func (h *AuditLogsHandler) GetAuditLogByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	auditLog, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "auditLogID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get audit log failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, auditLog); err != nil {
		h.logger.ErrorContext(r.Context(), "get audit log response failed", slog.String("error", err.Error()))
	}
}

func (h *AuditLogsHandler) ListAuditLogs(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseAuditLogListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	auditLogs, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list audit logs failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, auditLogs, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list audit logs response failed", slog.String("error", err.Error()))
	}
}

func (h *AuditLogsHandler) parseAuditLogListFilter(r *nethttp.Request) (domain.AuditLogListFilter, error) {
	query := r.URL.Query()
	filter := domain.AuditLogListFilter{}

	if typeValue := query.Get("type"); typeValue != "" {
		typ := domain.AppEventType(typeValue)
		filter.Type = &typ
	}
	if actorIDValue := query.Get("actor_id"); actorIDValue != "" {
		filter.ActorID = &actorIDValue
	}
	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.AuditLogListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}
	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.AuditLogListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
