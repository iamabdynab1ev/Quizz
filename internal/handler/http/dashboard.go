package http

import (
	"context"
	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"
	"lms-arvand-backend/internal/handler/http/middleware"
)

type dashboardUseCase interface {
	GetStudentDashboard(ctx context.Context, userID string) (domain.StudentDashboard, error)
	GetAdminDashboard(ctx context.Context) (domain.AdminDashboard, error)
}

type DashboardHandler struct {
	logger  *slog.Logger
	useCase dashboardUseCase
}

func NewDashboardHandler(logger *slog.Logger, useCase dashboardUseCase) *DashboardHandler {
	return &DashboardHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *DashboardHandler) GetDashboard(w nethttp.ResponseWriter, r *nethttp.Request) {
	identity, ok := middleware.CurrentAuthIdentity(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	if identity.User.IsAdmin || identity.User.IsSuperAdmin {
		h.GetAdminDashboard(w, r)
		return
	}

	h.GetMyDashboard(w, r)
}

func (h *DashboardHandler) GetMyDashboard(w nethttp.ResponseWriter, r *nethttp.Request) {
	identity, ok := middleware.CurrentAuthIdentity(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	dashboard, err := h.useCase.GetStudentDashboard(r.Context(), identity.User.ID)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "дашборд пользователя не загружен", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, dashboard); err != nil {
		h.logger.ErrorContext(r.Context(), "ответ дашборда пользователя не отправлен", slog.String("error", err.Error()))
	}
}

func (h *DashboardHandler) GetAdminDashboard(w nethttp.ResponseWriter, r *nethttp.Request) {
	dashboard, err := h.useCase.GetAdminDashboard(r.Context())
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "админский дашборд не загружен", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, dashboard); err != nil {
		h.logger.ErrorContext(r.Context(), "ответ админского дашборда не отправлен", slog.String("error", err.Error()))
	}
}
