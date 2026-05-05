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

type quizUseCase interface {
	Create(ctx context.Context, params domain.CreateQuizParams) (domain.Quiz, error)
	GetByID(ctx context.Context, quizID string) (domain.Quiz, error)
	List(ctx context.Context, filter domain.QuizListFilter) ([]domain.Quiz, int, error)
	Update(ctx context.Context, params domain.UpdateQuizParams) (domain.Quiz, error)
	Archive(ctx context.Context, quizID string) error
}

type QuizzesHandler struct {
	logger  *slog.Logger
	useCase quizUseCase
}

func NewQuizzesHandler(logger *slog.Logger, useCase quizUseCase) *QuizzesHandler {
	return &QuizzesHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *QuizzesHandler) CreateQuiz(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateQuizParams
	if err := decodeJSON(w, r, &params, 2<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	quiz, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, quiz); err != nil {
		h.logger.ErrorContext(r.Context(), "create quiz response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) GetQuizByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	quiz, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "quizID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, quiz); err != nil {
		h.logger.ErrorContext(r.Context(), "get quiz response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) ListQuizzes(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseQuizListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	quizzes, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list quizzes failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, quizzes, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list quizzes response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) UpdateQuiz(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.UpdateQuizParams
	if err := decodeJSON(w, r, &params, 2<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params.ID = chi.URLParam(r, "quizID")

	quiz, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, quiz); err != nil {
		h.logger.ErrorContext(r.Context(), "update quiz response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) ArchiveQuiz(w nethttp.ResponseWriter, r *nethttp.Request) {
	if err := h.useCase.Archive(r.Context(), chi.URLParam(r, "quizID")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "archive quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *QuizzesHandler) parseQuizListFilter(r *nethttp.Request) (domain.QuizListFilter, error) {
	query := r.URL.Query()

	filter := domain.QuizListFilter{
		Search: query.Get("search"),
	}

	if statusValue := query.Get("status"); statusValue != "" {
		status := domain.QuizStatus(statusValue)
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
			return domain.QuizListFilter{}, fmt.Errorf("include_archived must be boolean")
		}
		filter.IncludeArchived = parsed
	}

	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.QuizListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}

	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.QuizListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
