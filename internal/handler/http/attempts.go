package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"time"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type attemptUseCase interface {
	Submit(ctx context.Context, params domain.SubmitAttemptParams) (domain.Attempt, error)
	GetByID(ctx context.Context, attemptID string) (domain.Attempt, error)
	List(ctx context.Context, filter domain.AttemptListFilter) ([]domain.Attempt, int, error)
	Review(ctx context.Context, params domain.ReviewAttemptParams) (domain.Attempt, error)
}

type AttemptsHandler struct {
	logger  *slog.Logger
	useCase attemptUseCase
}

type submitAttemptRequest struct {
	UserID    *string                `json:"user_id,omitempty"`
	StartedAt *time.Time             `json:"started_at,omitempty"`
	Answers   []domain.AttemptAnswer `json:"answers"`
}

func NewAttemptsHandler(logger *slog.Logger, useCase attemptUseCase) *AttemptsHandler {
	return &AttemptsHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *AttemptsHandler) SubmitAttempt(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request submitAttemptRequest
	if err := decodeJSON(w, r, &request, 2<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	resolvedUserID, err := resolveActorUserID(r.Context(), request.UserID)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "submit attempt authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	params := domain.SubmitAttemptParams{
		QuizID:    chi.URLParam(r, "quizID"),
		UserID:    resolvedUserID,
		StartedAt: request.StartedAt,
		Answers:   request.Answers,
	}

	attempt, err := h.useCase.Submit(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "submit attempt failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, toAttemptResponse(attempt)); err != nil {
		h.logger.ErrorContext(r.Context(), "submit attempt response failed", slog.String("error", err.Error()))
	}
}

func (h *AttemptsHandler) GetAttemptByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	attempt, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "attemptID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get attempt failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := ensureOwnOrAdmin(r.Context(), attempt.UserID); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "get attempt authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toAttemptResponse(attempt)); err != nil {
		h.logger.ErrorContext(r.Context(), "get attempt response failed", slog.String("error", err.Error()))
	}
}

func (h *AttemptsHandler) ListAttempts(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseAttemptListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	filter.UserID, err = scopeUserID(r.Context(), filter.UserID)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "list attempts authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	attempts, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list attempts failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, toAttemptResponses(attempts), total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list attempts response failed", slog.String("error", err.Error()))
	}
}

func (h *AttemptsHandler) ReviewAttempt(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request struct {
		Passed  bool                        `json:"passed"`
		Comment *string                     `json:"comment,omitempty"`
		Scores  []domain.AttemptReviewScore `json:"scores,omitempty"`
	}
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	identity, err := currentAuthIdentity(r.Context())
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "review attempt authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	attempt, err := h.useCase.Review(r.Context(), domain.ReviewAttemptParams{
		AttemptID:  chi.URLParam(r, "attemptID"),
		ReviewerID: identity.User.ID,
		Passed:     request.Passed,
		Comment:    request.Comment,
		Scores:     request.Scores,
	})
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "review attempt failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toAttemptResponse(attempt)); err != nil {
		h.logger.ErrorContext(r.Context(), "review attempt response failed", slog.String("error", err.Error()))
	}
}

func (h *AttemptsHandler) parseAttemptListFilter(r *nethttp.Request) (domain.AttemptListFilter, error) {
	query := r.URL.Query()

	filter := domain.AttemptListFilter{}

	if quizIDValue := query.Get("quiz_id"); quizIDValue != "" {
		filter.QuizID = &quizIDValue
	}

	if userIDValue := query.Get("user_id"); userIDValue != "" {
		filter.UserID = &userIDValue
	}

	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.AttemptListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}

	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.AttemptListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
