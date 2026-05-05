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

type reviewUseCase interface {
	Create(ctx context.Context, params domain.CreateReviewParams) (domain.Review, error)
	GetByID(ctx context.Context, reviewID string) (domain.Review, error)
	List(ctx context.Context, filter domain.ReviewListFilter) ([]domain.Review, int, error)
	Moderate(ctx context.Context, params domain.ModerateReviewParams) (domain.Review, error)
}

type ReviewsHandler struct {
	logger  *slog.Logger
	useCase reviewUseCase
}

func NewReviewsHandler(logger *slog.Logger, useCase reviewUseCase) *ReviewsHandler {
	return &ReviewsHandler{logger: logger, useCase: useCase}
}

func (h *ReviewsHandler) CreateReview(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateReviewParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	review, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create review failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, review); err != nil {
		h.logger.ErrorContext(r.Context(), "create review response failed", slog.String("error", err.Error()))
	}
}

func (h *ReviewsHandler) GetReviewByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	review, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "reviewID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get review failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, review); err != nil {
		h.logger.ErrorContext(r.Context(), "get review response failed", slog.String("error", err.Error()))
	}
}

func (h *ReviewsHandler) ListReviews(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseReviewListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	reviews, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list reviews failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, reviews, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list reviews response failed", slog.String("error", err.Error()))
	}
}

func (h *ReviewsHandler) ModerateReview(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request struct {
		Status domain.ReviewStatus `json:"status"`
	}
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	review, err := h.useCase.Moderate(r.Context(), domain.ModerateReviewParams{
		ID:     chi.URLParam(r, "reviewID"),
		Status: request.Status,
	})
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "moderate review failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, review); err != nil {
		h.logger.ErrorContext(r.Context(), "moderate review response failed", slog.String("error", err.Error()))
	}
}

func (h *ReviewsHandler) parseReviewListFilter(r *nethttp.Request) (domain.ReviewListFilter, error) {
	query := r.URL.Query()
	filter := domain.ReviewListFilter{}

	if courseIDValue := query.Get("course_id"); courseIDValue != "" {
		filter.CourseID = &courseIDValue
	}
	if userIDValue := query.Get("user_id"); userIDValue != "" {
		filter.UserID = &userIDValue
	}
	if statusValue := query.Get("status"); statusValue != "" {
		status := domain.ReviewStatus(statusValue)
		filter.Status = &status
	}
	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.ReviewListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}
	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.ReviewListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
