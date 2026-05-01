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

type enrollmentUseCase interface {
	Create(ctx context.Context, params domain.CreateEnrollmentParams) (domain.Enrollment, error)
	GetByID(ctx context.Context, enrollmentID string) (domain.Enrollment, error)
	List(ctx context.Context, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int, error)
	Complete(ctx context.Context, enrollmentID string) (domain.Enrollment, error)
}

type EnrollmentsHandler struct {
	logger  *slog.Logger
	useCase enrollmentUseCase
}

type createEnrollmentRequest struct {
	CourseID string  `json:"course_id"`
	UserID   *string `json:"user_id,omitempty"`
}

func NewEnrollmentsHandler(logger *slog.Logger, useCase enrollmentUseCase) *EnrollmentsHandler {
	return &EnrollmentsHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *EnrollmentsHandler) CreateEnrollment(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request createEnrollmentRequest
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	resolvedUserID, err := resolveActorUserID(r.Context(), request.UserID)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "create enrollment authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	params := domain.CreateEnrollmentParams{
		CourseID: request.CourseID,
		UserID:   resolvedUserID,
	}

	enrollment, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create enrollment failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, enrollment); err != nil {
		h.logger.ErrorContext(r.Context(), "create enrollment response failed", slog.String("error", err.Error()))
	}
}

func (h *EnrollmentsHandler) GetEnrollmentByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	enrollment, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "enrollmentID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get enrollment failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := ensureOwnOrAdmin(r.Context(), enrollment.UserID); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "get enrollment authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, enrollment); err != nil {
		h.logger.ErrorContext(r.Context(), "get enrollment response failed", slog.String("error", err.Error()))
	}
}

func (h *EnrollmentsHandler) ListEnrollments(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseEnrollmentListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	filter.UserID, err = scopeUserID(r.Context(), filter.UserID)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "list enrollments authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	enrollments, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list enrollments failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, enrollments, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list enrollments response failed", slog.String("error", err.Error()))
	}
}

func (h *EnrollmentsHandler) CompleteEnrollment(w nethttp.ResponseWriter, r *nethttp.Request) {
	enrollment, err := h.useCase.Complete(r.Context(), chi.URLParam(r, "enrollmentID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "complete enrollment failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, enrollment); err != nil {
		h.logger.ErrorContext(r.Context(), "complete enrollment response failed", slog.String("error", err.Error()))
	}
}

func (h *EnrollmentsHandler) parseEnrollmentListFilter(r *nethttp.Request) (domain.EnrollmentListFilter, error) {
	query := r.URL.Query()

	filter := domain.EnrollmentListFilter{}

	if courseIDValue := query.Get("course_id"); courseIDValue != "" {
		filter.CourseID = &courseIDValue
	}

	if userIDValue := query.Get("user_id"); userIDValue != "" {
		filter.UserID = &userIDValue
	}

	if statusValue := query.Get("status"); statusValue != "" {
		status := domain.EnrollmentStatus(statusValue)
		filter.Status = &status
	}

	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.EnrollmentListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}

	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.EnrollmentListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
