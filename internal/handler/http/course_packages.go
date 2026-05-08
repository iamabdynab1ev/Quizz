package http

import (
	"context"
	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"
)

type coursePackageUseCase interface {
	Create(ctx context.Context, params domain.CreateCoursePackageParams) (domain.Course, error)
}

type CoursePackagesHandler struct {
	logger  *slog.Logger
	useCase coursePackageUseCase
}

func NewCoursePackagesHandler(logger *slog.Logger, useCase coursePackageUseCase) *CoursePackagesHandler {
	return &CoursePackagesHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *CoursePackagesHandler) CreateCoursePackage(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateCoursePackageParams
	if err := decodeJSON(w, r, &params, 4<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	created, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create course package failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, created); err != nil {
		h.logger.ErrorContext(r.Context(), "create course package response failed", slog.String("error", err.Error()))
	}
}
