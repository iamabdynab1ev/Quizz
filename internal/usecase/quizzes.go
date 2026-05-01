package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type quizRepository interface {
	Create(ctx context.Context, params domain.CreateQuizParams) (domain.Quiz, error)
	GetByID(ctx context.Context, quizID string) (domain.Quiz, error)
	List(ctx context.Context, filter domain.QuizListFilter) ([]domain.Quiz, int, error)
	Update(ctx context.Context, params domain.UpdateQuizParams) (domain.Quiz, error)
	Archive(ctx context.Context, quizID string) error
}

type QuizUseCase struct {
	repository quizRepository
	audit      *AuditLogger
}

func NewQuizUseCase(repository quizRepository) *QuizUseCase {
	return &QuizUseCase{repository: repository}
}

func (u *QuizUseCase) WithAudit(audit *AuditLogger) *QuizUseCase {
	u.audit = audit
	return u
}

func (u *QuizUseCase) Create(ctx context.Context, params domain.CreateQuizParams) (domain.Quiz, error) {
	normalized, err := normalizeCreateQuizParams(params)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("usecase quizzes create: %w", err)
	}

	quiz, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("usecase quizzes create: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventTestCreated, map[string]any{
			"quiz_id": quiz.ID,
			"title":   quiz.Title,
			"status":  quiz.Status,
		})
	}

	return quiz, nil
}

func (u *QuizUseCase) GetByID(ctx context.Context, quizID string) (domain.Quiz, error) {
	quizID = strings.TrimSpace(quizID)
	if quizID == "" {
		return domain.Quiz{}, fmt.Errorf("usecase quizzes get by id: %w", domain.ErrValidation)
	}

	quiz, err := u.repository.GetByID(ctx, quizID)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("usecase quizzes get by id: %w", err)
	}

	return quiz, nil
}

func (u *QuizUseCase) List(ctx context.Context, filter domain.QuizListFilter) ([]domain.Quiz, int, error) {
	normalized, err := normalizeQuizListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase quizzes list: %w", err)
	}

	quizzes, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase quizzes list: %w", err)
	}

	return quizzes, total, nil
}

func (u *QuizUseCase) Update(ctx context.Context, params domain.UpdateQuizParams) (domain.Quiz, error) {
	normalized, err := normalizeUpdateQuizParams(params)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("usecase quizzes update: %w", err)
	}

	quiz, err := u.repository.Update(ctx, normalized)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("usecase quizzes update: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventTestUpdated, map[string]any{
			"quiz_id": quiz.ID,
			"title":   quiz.Title,
			"status":  quiz.Status,
		})
	}

	return quiz, nil
}

func (u *QuizUseCase) Archive(ctx context.Context, quizID string) error {
	quizID = strings.TrimSpace(quizID)
	if quizID == "" {
		return fmt.Errorf("usecase quizzes archive: %w", domain.ErrValidation)
	}

	if err := u.repository.Archive(ctx, quizID); err != nil {
		return fmt.Errorf("usecase quizzes archive: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventTestDeleted, map[string]any{
			"quiz_id": quizID,
		})
	}

	return nil
}

func normalizeCreateQuizParams(params domain.CreateQuizParams) (domain.CreateQuizParams, error) {
	params.Category = normalizeOptionalString(params.Category)

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.CreateQuizParams{}, fmt.Errorf("title is invalid: %w", domain.ErrValidation)
	}

	if params.Status == "" {
		params.Status = domain.QuizStatusDraft
	}

	if !params.Status.IsValid() {
		return domain.CreateQuizParams{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
	}

	if err := normalizePlatforms(&params.Platforms); err != nil {
		return domain.CreateQuizParams{}, fmt.Errorf("platforms are invalid: %w", err)
	}

	if params.TimeLimitMinutes != nil && *params.TimeLimitMinutes <= 0 {
		return domain.CreateQuizParams{}, fmt.Errorf("time_limit_minutes must be greater than zero: %w", domain.ErrValidation)
	}

	if params.PassingScore < 0 || params.PassingScore > 100 {
		return domain.CreateQuizParams{}, fmt.Errorf("passing_score must be in range 0..100: %w", domain.ErrValidation)
	}

	if params.MaxAttempts <= 0 {
		params.MaxAttempts = 3
	}

	questions, err := normalizeQuestionPayloads(params.Questions)
	if err != nil {
		return domain.CreateQuizParams{}, fmt.Errorf("questions are invalid: %w", err)
	}
	params.Questions = questions

	return params, nil
}

func normalizeUpdateQuizParams(params domain.UpdateQuizParams) (domain.UpdateQuizParams, error) {
	params.ID = strings.TrimSpace(params.ID)
	params.Category = normalizeOptionalString(params.Category)

	if params.ID == "" {
		return domain.UpdateQuizParams{}, fmt.Errorf("id is required: %w", domain.ErrValidation)
	}

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.UpdateQuizParams{}, fmt.Errorf("title is invalid: %w", domain.ErrValidation)
	}

	if !params.Status.IsValid() {
		return domain.UpdateQuizParams{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
	}

	if err := normalizePlatforms(&params.Platforms); err != nil {
		return domain.UpdateQuizParams{}, fmt.Errorf("platforms are invalid: %w", err)
	}

	if params.TimeLimitMinutes != nil && *params.TimeLimitMinutes <= 0 {
		return domain.UpdateQuizParams{}, fmt.Errorf("time_limit_minutes must be greater than zero: %w", domain.ErrValidation)
	}

	if params.PassingScore < 0 || params.PassingScore > 100 {
		return domain.UpdateQuizParams{}, fmt.Errorf("passing_score must be in range 0..100: %w", domain.ErrValidation)
	}

	if params.MaxAttempts <= 0 {
		return domain.UpdateQuizParams{}, fmt.Errorf("max_attempts must be greater than zero: %w", domain.ErrValidation)
	}

	questions, err := normalizeQuestionPayloads(params.Questions)
	if err != nil {
		return domain.UpdateQuizParams{}, fmt.Errorf("questions are invalid: %w", err)
	}
	params.Questions = questions

	return params, nil
}

func normalizeQuizListFilter(filter domain.QuizListFilter) (domain.QuizListFilter, error) {
	filter.Search = strings.TrimSpace(filter.Search)

	if filter.Status != nil {
		status := domain.QuizStatus(strings.TrimSpace(string(*filter.Status)))
		if !status.IsValid() {
			return domain.QuizListFilter{}, fmt.Errorf("status filter is invalid: %w", domain.ErrValidation)
		}
		filter.Status = &status
	}

	if filter.Category != nil {
		filter.Category = normalizeOptionalString(filter.Category)
	}

	if filter.Platform != nil {
		platform := domain.Platform(strings.TrimSpace(string(*filter.Platform)))
		if !platform.IsValid() {
			return domain.QuizListFilter{}, fmt.Errorf("platform filter is invalid: %w", domain.ErrValidation)
		}
		filter.Platform = &platform
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Offset < 0 {
		return domain.QuizListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}

func normalizeQuestionPayloads(questions []domain.QuestionPayload) ([]domain.QuestionPayload, error) {
	if len(questions) == 0 {
		return nil, fmt.Errorf("at least one question is required: %w", domain.ErrValidation)
	}

	normalized := make([]domain.QuestionPayload, 0, len(questions))
	positions := make(map[int]struct{}, len(questions))

	for index, question := range questions {
		if question.Position <= 0 {
			question.Position = index + 1
		}

		if _, exists := positions[question.Position]; exists {
			return nil, fmt.Errorf("duplicate question position: %w", domain.ErrValidation)
		}
		positions[question.Position] = struct{}{}

		if !question.Type.IsValid() {
			return nil, fmt.Errorf("question type is invalid: %w", domain.ErrValidation)
		}

		if err := question.Prompt.ValidateRequired(); err != nil {
			return nil, fmt.Errorf("question prompt is invalid: %w", domain.ErrValidation)
		}

		if question.Points <= 0 {
			question.Points = 1
		}

		if len(question.Config) == 0 {
			question.Config = []byte("{}")
		}

		normalized = append(normalized, question)
	}

	return normalized, nil
}
