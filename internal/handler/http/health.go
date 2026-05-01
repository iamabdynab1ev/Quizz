package http

import (
	"context"
	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/usecase"
)

type healthChecker interface {
	Check(ctx context.Context) (usecase.HealthStatus, error)
}

type HealthHandler struct {
	logger  *slog.Logger
	useCase healthChecker
}

func NewHealthHandler(logger *slog.Logger, useCase healthChecker) *HealthHandler {
	return &HealthHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *HealthHandler) GetHealth(w nethttp.ResponseWriter, r *nethttp.Request) {
	status, err := h.useCase.Check(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "health check failed", slog.String("error", err.Error()))
		writeError(w, nethttp.StatusServiceUnavailable, "service_unavailable", "service is unavailable")
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, status); err != nil {
		h.logger.ErrorContext(r.Context(), "write health response failed", slog.String("error", err.Error()))
	}
}
