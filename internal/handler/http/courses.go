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

type courseUseCase interface {
	Create(ctx context.Context, params domain.CreateCourseParams) (domain.Course, error)
	GetByID(ctx context.Context, courseID string) (domain.Course, error)
	List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int, error)
	Update(ctx context.Context, params domain.UpdateCourseParams) (domain.Course, error)
	Archive(ctx context.Context, courseID string) error
}

type CoursesHandler struct {
	logger  *slog.Logger
	useCase courseUseCase
}

func NewCoursesHandler(logger *slog.Logger, useCase courseUseCase) *CoursesHandler {
	return &CoursesHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *CoursesHandler) CreateCourse(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateCourseParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	course, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create course failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, course); err != nil {
		h.logger.ErrorContext(r.Context(), "create course response failed", slog.String("error", err.Error()))
	}
}

func (h *CoursesHandler) GetCourseByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	course, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "courseID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get course failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, course); err != nil {
		h.logger.ErrorContext(r.Context(), "get course response failed", slog.String("error", err.Error()))
	}
}

func (h *CoursesHandler) ListCourses(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseCourseListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	courses, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list courses failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, courses, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list courses response failed", slog.String("error", err.Error()))
	}
}

func (h *CoursesHandler) UpdateCourse(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.UpdateCourseParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params.ID = chi.URLParam(r, "courseID")

	course, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update course failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, course); err != nil {
		h.logger.ErrorContext(r.Context(), "update course response failed", slog.String("error", err.Error()))
	}
}

func (h *CoursesHandler) ArchiveCourse(w nethttp.ResponseWriter, r *nethttp.Request) {
	if err := h.useCase.Archive(r.Context(), chi.URLParam(r, "courseID")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "archive course failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *CoursesHandler) parseCourseListFilter(r *nethttp.Request) (domain.CourseListFilter, error) {
	query := r.URL.Query()

	filter := domain.CourseListFilter{
		Search: query.Get("search"),
	}

	if statusValue := query.Get("status"); statusValue != "" {
		status := domain.CourseStatus(statusValue)
		filter.Status = &status
	}

	if categoryValue := query.Get("category"); categoryValue != "" {
		filter.Category = &categoryValue
	}

	if platformValue := query.Get("platform"); platformValue != "" {
		platform := domain.Platform(platformValue)
		filter.Platform = &platform
	}

	if includeArchivedValue := query.Get("include_archived"); includeArchivedValue != "" {
		parsed, err := strconv.ParseBool(includeArchivedValue)
		if err != nil {
			return domain.CourseListFilter{}, fmt.Errorf("include_archived must be boolean")
		}
		filter.IncludeArchived = parsed
	}

	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.CourseListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}

	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.CourseListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
