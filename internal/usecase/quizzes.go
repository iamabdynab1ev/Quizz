package usecase

import (
	"context"
	"encoding/json"
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

	var validation fieldValidationBuilder
	validation.addRequiredMultiLang("title", params.Title, "Название теста")

	if params.Status == "" {
		params.Status = domain.QuizStatusDraft
	}

	if !params.Status.IsValid() {
		validation.add("status", "invalid_enum", "Статус теста должен быть draft, published или archived")
	}

	if err := normalizePlatforms(&params.Platforms); err != nil {
		validation.add("platforms", "invalid_enum", "Платформа должна быть web, mobile или telegram")
	}

	if params.TimeLimitMinutes != nil && *params.TimeLimitMinutes <= 0 {
		validation.add("time_limit_minutes", "must_be_positive", "Лимит времени должен быть больше 0")
	}

	validation.addIntRange("passing_score", params.PassingScore, 0, 100, "Процент прохождения")

	if params.MaxAttempts <= 0 {
		params.MaxAttempts = 3
	}

	questions, err := normalizeQuestionPayloads(params.Questions)
	if err != nil {
		return domain.CreateQuizParams{}, err
	}

	if err := validation.err(); err != nil {
		return domain.CreateQuizParams{}, err
	}
	params.Questions = questions

	return params, nil
}

func normalizeUpdateQuizParams(params domain.UpdateQuizParams) (domain.UpdateQuizParams, error) {
	params.ID = strings.TrimSpace(params.ID)
	params.Category = normalizeOptionalString(params.Category)

	var validation fieldValidationBuilder
	if params.ID == "" {
		validation.add("id", "required", "ID теста обязателен")
	}
	validation.addRequiredMultiLang("title", params.Title, "Название теста")

	if !params.Status.IsValid() {
		validation.add("status", "invalid_enum", "Статус теста должен быть draft, published или archived")
	}

	if err := normalizePlatforms(&params.Platforms); err != nil {
		validation.add("platforms", "invalid_enum", "Платформа должна быть web, mobile или telegram")
	}

	if params.TimeLimitMinutes != nil && *params.TimeLimitMinutes <= 0 {
		validation.add("time_limit_minutes", "must_be_positive", "Лимит времени должен быть больше 0")
	}

	validation.addIntRange("passing_score", params.PassingScore, 0, 100, "Процент прохождения")

	if params.MaxAttempts <= 0 {
		validation.add("max_attempts", "must_be_positive", "Количество попыток должно быть больше 0")
	}

	questions, err := normalizeQuestionPayloads(params.Questions)
	if err != nil {
		return domain.UpdateQuizParams{}, err
	}

	if err := validation.err(); err != nil {
		return domain.UpdateQuizParams{}, err
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
		// Draft quizzes can be created empty and filled later from the admin UI.
		return nil, nil
	}

	normalized := make([]domain.QuestionPayload, 0, len(questions))
	positions := make(map[int]struct{}, len(questions))

	for index, question := range questions {
		fieldPrefix := fmt.Sprintf("questions.%d", index)
		if question.Position <= 0 {
			question.Position = index + 1
		}

		if _, exists := positions[question.Position]; exists {
			return nil, domain.FieldValidationError("Проверьте поля формы",
				domain.ValidationField(fieldPrefix+".position", "duplicate", "Позиция вопроса не должна повторяться"))
		}
		positions[question.Position] = struct{}{}

		if !question.Type.IsValid() {
			return nil, domain.FieldValidationError("Проверьте поля формы",
				domain.ValidationField(fieldPrefix+".type", "invalid_enum", "Тип вопроса не поддерживается"))
		}

		var validation fieldValidationBuilder
		validation.addRequiredMultiLang(fieldPrefix+".prompt", question.Prompt, "Текст вопроса")
		if err := validation.err(); err != nil {
			return nil, err
		}

		if question.Points <= 0 {
			question.Points = 1
		}

		if len(question.Config) == 0 {
			question.Config = []byte("{}")
		}

		if err := validateQuestionConfig(fieldPrefix+".config", question.Type, question.Config); err != nil {
			return nil, err
		}

		normalized = append(normalized, question)
	}

	return normalized, nil
}

func validateQuestionConfig(field string, questionType domain.QuestionType, config []byte) error {
	var payload struct {
		Options []struct {
			ID        string `json:"id"`
			IsCorrect bool   `json:"is_correct"`
		} `json:"options"`
		Correct         *bool    `json:"correct"`
		AcceptedAnswers []string `json:"accepted_answers"`
	}

	if err := json.Unmarshal(config, &payload); err != nil {
		return domain.FieldValidationError("Проверьте поля формы",
			domain.ValidationField(field, "invalid_json", "Настройки вопроса должны быть корректным JSON"))
	}

	var validation fieldValidationBuilder
	switch questionType {
	case domain.QuestionTypeSingleChoice, domain.QuestionTypeImageChoice:
		correctCount, optionsOK := validateChoiceOptions(payload.Options)
		if !optionsOK {
			validation.add(field+".options", "required", "Добавьте варианты ответа с ID")
		}
		if correctCount != 1 {
			validation.add(field+".options", "invalid_correct_count", "Для этого типа вопроса должен быть ровно один правильный ответ")
		}
	case domain.QuestionTypeMultipleChoice:
		correctCount, optionsOK := validateChoiceOptions(payload.Options)
		if !optionsOK {
			validation.add(field+".options", "required", "Добавьте варианты ответа с ID")
		}
		if correctCount < 1 {
			validation.add(field+".options", "invalid_correct_count", "Отметьте минимум один правильный ответ")
		}
	case domain.QuestionTypeTrueFalse:
		if payload.Correct == nil {
			validation.add(field+".correct", "required", "Укажите правильное значение true или false")
		}
	case domain.QuestionTypeShortAnswer, domain.QuestionTypeFillBlank:
		if len(normalizeStringSlice(payload.AcceptedAnswers)) == 0 {
			validation.add(field+".accepted_answers", "required", "Добавьте минимум один правильный текстовый ответ")
		}
	}

	return validation.err()
}

func validateChoiceOptions(options []struct {
	ID        string `json:"id"`
	IsCorrect bool   `json:"is_correct"`
}) (int, bool) {
	if len(options) == 0 {
		return 0, false
	}

	correctCount := 0
	seenIDs := make(map[string]struct{}, len(options))
	for _, option := range options {
		id := strings.TrimSpace(option.ID)
		if id == "" {
			return correctCount, false
		}
		if _, exists := seenIDs[id]; exists {
			return correctCount, false
		}
		seenIDs[id] = struct{}{}
		if option.IsCorrect {
			correctCount++
		}
	}

	return correctCount, true
}
