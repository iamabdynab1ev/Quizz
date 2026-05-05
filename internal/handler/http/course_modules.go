package http

import (
	"context"
	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type courseModuleUseCase interface {
	Create(ctx context.Context, params domain.CreateCourseModuleParams) (domain.CourseModule, error)
	GetByID(ctx context.Context, moduleID string) (domain.CourseModule, error)
	List(ctx context.Context, filter domain.CourseModuleListFilter) ([]domain.CourseModule, int, error)
	Update(ctx context.Context, params domain.UpdateCourseModuleParams) (domain.CourseModule, error)
	Delete(ctx context.Context, moduleID string) error
}

type CourseModulesHandler struct {
	logger  *slog.Logger
	useCase courseModuleUseCase
}

func NewCourseModulesHandler(logger *slog.Logger, useCase courseModuleUseCase) *CourseModulesHandler {
	return &CourseModulesHandler{logger: logger, useCase: useCase}
}

func (h *CourseModulesHandler) CreateCourseModule(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateCourseModuleParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	module, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create course module failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, module); err != nil {
		h.logger.ErrorContext(r.Context(), "create course module response failed", slog.String("error", err.Error()))
	}
}

func (h *CourseModulesHandler) GetCourseModuleByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	module, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "moduleID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get course module failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, module); err != nil {
		h.logger.ErrorContext(r.Context(), "get course module response failed", slog.String("error", err.Error()))
	}
}

func (h *CourseModulesHandler) ListCourseModules(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter := domain.CourseModuleListFilter{CourseID: r.URL.Query().Get("course_id")}

	modules, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list course modules failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, modules, total, 0, 0); err != nil {
		h.logger.ErrorContext(r.Context(), "list course modules response failed", slog.String("error", err.Error()))
	}
}

func (h *CourseModulesHandler) UpdateCourseModule(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.UpdateCourseModuleParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params.ID = chi.URLParam(r, "moduleID")

	module, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update course module failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, module); err != nil {
		h.logger.ErrorContext(r.Context(), "update course module response failed", slog.String("error", err.Error()))
	}
}

func (h *CourseModulesHandler) DeleteCourseModule(w nethttp.ResponseWriter, r *nethttp.Request) {
	if err := h.useCase.Delete(r.Context(), chi.URLParam(r, "moduleID")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "delete course module failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}
