package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type courseTestUseCase interface {
	Create(ctx context.Context, params domain.CreateCourseTestParams) (domain.CourseTest, error)
	List(ctx context.Context, filter domain.CourseTestListFilter) ([]domain.CourseTest, int, error)
	Delete(ctx context.Context, courseID, moduleID, quizID string) error
	DeleteByID(ctx context.Context, courseTestID string) error
}

type CourseTestsHandler struct {
	logger  *slog.Logger
	useCase courseTestUseCase
}

func NewCourseTestsHandler(logger *slog.Logger, useCase courseTestUseCase) *CourseTestsHandler {
	return &CourseTestsHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *CourseTestsHandler) CreateCourseTest(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateCourseTestParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	courseTest, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create course test failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, courseTest); err != nil {
		h.logger.ErrorContext(r.Context(), "create course test response failed", slog.String("error", err.Error()))
	}
}

func (h *CourseTestsHandler) ListCourseTests(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseCourseTestListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	courseTests, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list course tests failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, courseTests, total, 0, 0); err != nil {
		h.logger.ErrorContext(r.Context(), "list course tests response failed", slog.String("error", err.Error()))
	}
}

func (h *CourseTestsHandler) DeleteCourseTest(w nethttp.ResponseWriter, r *nethttp.Request) {
	query := r.URL.Query()

	if err := h.useCase.Delete(r.Context(), query.Get("course_id"), query.Get("module_id"), query.Get("quiz_id")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "delete course test failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *CourseTestsHandler) DeleteCourseTestByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	if err := h.useCase.DeleteByID(r.Context(), chi.URLParam(r, "courseTestID")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "delete course test by id failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *CourseTestsHandler) parseCourseTestListFilter(r *nethttp.Request) (domain.CourseTestListFilter, error) {
	query := r.URL.Query()
	filter := domain.CourseTestListFilter{}

	if courseIDValue := query.Get("course_id"); courseIDValue != "" {
		filter.CourseID = &courseIDValue
	}

	if moduleIDValue := query.Get("module_id"); moduleIDValue != "" {
		filter.ModuleID = &moduleIDValue
	}

	if filter.CourseID != nil && filter.ModuleID != nil {
		return domain.CourseTestListFilter{}, fmt.Errorf("only one of course_id or module_id is allowed")
	}

	return filter, nil
}
