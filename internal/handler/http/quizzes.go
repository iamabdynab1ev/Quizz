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

type quizRequest struct {
	ID                      string                   `json:"id,omitempty"`
	CourseID                *string                  `json:"course_id,omitempty"`
	CourseIDCamel           *string                  `json:"courseId,omitempty"`
	Title                   domain.MultiLangText     `json:"title"`
	Description             domain.MultiLangText     `json:"description"`
	Category                *string                  `json:"category,omitempty"`
	Status                  domain.QuizStatus        `json:"status"`
	Platforms               []domain.Platform        `json:"platforms"`
	TimeLimitMinutes        *int                     `json:"time_limit_minutes,omitempty"`
	TimeLimitMinutesCamel   *int                     `json:"timeLimitMinutes,omitempty"`
	PassingScore            int                      `json:"passing_score"`
	PassingScoreCamel       int                      `json:"passingScore,omitempty"`
	PassingPoints           *float64                 `json:"passing_points,omitempty"`
	PassingPointsCamel      *float64                 `json:"passingPoints,omitempty"`
	MaxAttempts             int                      `json:"max_attempts"`
	MaxAttemptsCamel        int                      `json:"maxAttempts,omitempty"`
	RetakeCooldownDays      int                      `json:"retake_cooldown_days"`
	RetakeCooldownDaysCamel int                      `json:"retakeCooldownDays,omitempty"`
	ShuffleQuestions        bool                     `json:"shuffle_questions"`
	ShuffleQuestionsCamel   bool                     `json:"shuffleQuestions,omitempty"`
	ShowResults             bool                     `json:"show_results"`
	ShowResultsCamel        bool                     `json:"showResults,omitempty"`
	AllowRetry              bool                     `json:"allow_retry"`
	AllowRetryCamel         bool                     `json:"allowRetry,omitempty"`
	Questions               []domain.QuestionPayload `json:"questions"`
	CreatedAt               *time.Time               `json:"created_at,omitempty"`
	CreatedAtCamel          *time.Time               `json:"createdAt,omitempty"`
	UpdatedAt               *time.Time               `json:"updated_at,omitempty"`
	UpdatedAtCamel          *time.Time               `json:"updatedAt,omitempty"`
}

func NewQuizzesHandler(logger *slog.Logger, useCase quizUseCase) *QuizzesHandler {
	return &QuizzesHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *QuizzesHandler) CreateQuiz(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request quizRequest
	if err := decodeJSON(w, r, &request, 2<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params := request.toCreateParams()
	quiz, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, toQuizResponse(quiz)); err != nil {
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

	if err := writeJSON(w, nethttp.StatusOK, toQuizResponse(quiz)); err != nil {
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

	if err := writePagedJSON(w, nethttp.StatusOK, toQuizResponses(quizzes), total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list quizzes response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) UpdateQuiz(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request quizRequest
	if err := decodeJSON(w, r, &request, 2<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params := request.toUpdateParams()
	params.ID = chi.URLParam(r, "quizID")

	quiz, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toQuizResponse(quiz)); err != nil {
		h.logger.ErrorContext(r.Context(), "update quiz response failed", slog.String("error", err.Error()))
	}
}

func (r quizRequest) toCreateParams() domain.CreateQuizParams {
	passingScore, passingPoints, timeLimit, maxAttempts, cooldownDays := r.normalizedAliases()

	return domain.CreateQuizParams{
		Title:              r.Title,
		Description:        r.Description,
		Category:           r.Category,
		Status:             r.Status,
		Platforms:          r.Platforms,
		TimeLimitMinutes:   timeLimit,
		PassingScore:       passingScore,
		PassingPoints:      passingPoints,
		MaxAttempts:        maxAttempts,
		RetakeCooldownDays: cooldownDays,
		ShuffleQuestions:   r.ShuffleQuestions || r.ShuffleQuestionsCamel,
		ShowResults:        r.ShowResults || r.ShowResultsCamel,
		AllowRetry:         r.AllowRetry || r.AllowRetryCamel,
		Questions:          r.Questions,
	}
}

func (r quizRequest) toUpdateParams() domain.UpdateQuizParams {
	passingScore, passingPoints, timeLimit, maxAttempts, cooldownDays := r.normalizedAliases()

	return domain.UpdateQuizParams{
		Title:              r.Title,
		Description:        r.Description,
		Category:           r.Category,
		Status:             r.Status,
		Platforms:          r.Platforms,
		TimeLimitMinutes:   timeLimit,
		PassingScore:       passingScore,
		PassingPoints:      passingPoints,
		MaxAttempts:        maxAttempts,
		RetakeCooldownDays: cooldownDays,
		ShuffleQuestions:   r.ShuffleQuestions || r.ShuffleQuestionsCamel,
		ShowResults:        r.ShowResults || r.ShowResultsCamel,
		AllowRetry:         r.AllowRetry || r.AllowRetryCamel,
		Questions:          r.Questions,
	}
}

func (r quizRequest) normalizedAliases() (int, float64, *int, int, int) {
	passingScore := r.PassingScore
	if passingScore == 0 && r.PassingScoreCamel != 0 {
		passingScore = r.PassingScoreCamel
	}

	passingPoints := 0.0
	if r.PassingPoints != nil {
		passingPoints = *r.PassingPoints
	}
	if r.PassingPoints == nil && r.PassingPointsCamel != nil {
		passingPoints = *r.PassingPointsCamel
	}

	timeLimit := r.TimeLimitMinutes
	if timeLimit == nil && r.TimeLimitMinutesCamel != nil {
		timeLimit = r.TimeLimitMinutesCamel
	}

	maxAttempts := r.MaxAttempts
	if maxAttempts == 0 && r.MaxAttemptsCamel != 0 {
		maxAttempts = r.MaxAttemptsCamel
	}

	cooldownDays := r.RetakeCooldownDays
	if cooldownDays == 0 && r.RetakeCooldownDaysCamel != 0 {
		cooldownDays = r.RetakeCooldownDaysCamel
	}

	return passingScore, passingPoints, timeLimit, maxAttempts, cooldownDays
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
