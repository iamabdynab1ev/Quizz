package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"

	"lms-arvand-backend/internal/domain"
	"lms-arvand-backend/internal/handler/http/middleware"

	"github.com/go-chi/chi/v5"
)

// quizUseCase is the course use case — courses now embed quiz settings and questions.
type quizUseCase interface {
	Create(ctx context.Context, params domain.CreateCourseParams) (domain.Course, error)
	GetByID(ctx context.Context, courseID string) (domain.Course, error)
	List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int, error)
	Update(ctx context.Context, params domain.UpdateCourseParams) (domain.Course, error)
	Archive(ctx context.Context, courseID string) error
}

type QuizzesHandler struct {
	logger  *slog.Logger
	useCase quizUseCase
}

func NewQuizzesHandler(logger *slog.Logger, useCase quizUseCase) *QuizzesHandler {
	return &QuizzesHandler{logger: logger, useCase: useCase}
}

// quizRequest is the inbound JSON for create/update quiz (= course with quiz fields).
type quizRequest struct {
	Title              domain.MultiLangText     `json:"title"`
	Description        domain.MultiLangText     `json:"description"`
	Category           *string                  `json:"category,omitempty"`
	Status             domain.CourseStatus      `json:"status"`
	Platforms          []domain.Platform        `json:"platforms"`
	VideoURL           *string                  `json:"video_url,omitempty"`
	CoverImageURL      *string                  `json:"cover_image_url,omitempty"`
	TimeLimitMinutes   *int                     `json:"time_limit_minutes,omitempty"`
	PassingScore       int                      `json:"passing_score"`
	MaxAttempts        int                      `json:"max_attempts"`
	RetakeCooldownDays int                      `json:"retake_cooldown_days"`
	Questions          []domain.QuestionPayload `json:"questions"`
}

func (r quizRequest) toCreateParams() domain.CreateCourseParams {
	return domain.CreateCourseParams{
		Title:              r.Title,
		Description:        r.Description,
		Category:           r.Category,
		Status:             r.Status,
		Platforms:          r.Platforms,
		VideoURL:           r.VideoURL,
		CoverImageURL:      r.CoverImageURL,
		QuizPassPercent:    r.PassingScore,
		QuizMinutes:        ptrIntOrZero(r.TimeLimitMinutes),
		MaxAttempts:        r.MaxAttempts,
		RetakeCooldownDays: r.RetakeCooldownDays,
		Questions:          r.Questions,
	}
}

func (r quizRequest) toUpdateParams(id string) domain.UpdateCourseParams {
	return domain.UpdateCourseParams{
		ID:                 id,
		Title:              r.Title,
		Description:        r.Description,
		Category:           r.Category,
		Status:             r.Status,
		Platforms:          r.Platforms,
		VideoURL:           r.VideoURL,
		CoverImageURL:      r.CoverImageURL,
		QuizPassPercent:    r.PassingScore,
		QuizMinutes:        ptrIntOrZero(r.TimeLimitMinutes),
		MaxAttempts:        r.MaxAttempts,
		RetakeCooldownDays: r.RetakeCooldownDays,
		Questions:          r.Questions,
	}
}

func ptrIntOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func (h *QuizzesHandler) GetQuizByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	course, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "quizID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toCourseAsQuizResponse(course, false)); err != nil {
		h.logger.ErrorContext(r.Context(), "get quiz response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) GetQuizByIDWithAnswers(w nethttp.ResponseWriter, r *nethttp.Request) {
	course, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "quizID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get quiz with answers failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toCourseAsQuizResponse(course, true)); err != nil {
		h.logger.ErrorContext(r.Context(), "get quiz with answers response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) ListQuizzes(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseQuizListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	courses, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list quizzes failed", slog.String("error", err.Error()))
		}
		return
	}

	responses := make([]quizResponse, 0, len(courses))
	for _, c := range courses {
		responses = append(responses, toCourseAsQuizResponse(c, false))
	}

	if err := writePagedJSON(w, nethttp.StatusOK, responses, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list quizzes response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) CreateQuiz(w nethttp.ResponseWriter, r *nethttp.Request) {
	identity, ok := middleware.CurrentAuthIdentity(r.Context())

	var request quizRequest
	if err := decodeJSON(w, r, &request, 2<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params := request.toCreateParams()
	if ok {
		params.CreatedByUserID = &identity.User.ID
		name := identity.User.FirstName
		if identity.User.LastName != "" {
			if name != "" {
				name += " "
			}
			name += identity.User.LastName
		}
		params.CreatedByName = name
	}

	course, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, toCourseAsQuizResponse(course, false)); err != nil {
		h.logger.ErrorContext(r.Context(), "create quiz response failed", slog.String("error", err.Error()))
	}
}

func (h *QuizzesHandler) UpdateQuiz(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request quizRequest
	if err := decodeJSON(w, r, &request, 2<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	course, err := h.useCase.Update(r.Context(), request.toUpdateParams(chi.URLParam(r, "quizID")))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update quiz failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toCourseAsQuizResponse(course, false)); err != nil {
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

func (h *QuizzesHandler) parseQuizListFilter(r *nethttp.Request) (domain.CourseListFilter, error) {
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
