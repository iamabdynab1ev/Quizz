package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

const (
	defaultQuizPassingScore       = 80
	defaultQuizMaxAttempts        = 3
	defaultQuizRetakeCooldownDays = 0
	maxQuizRetakeCooldownDays     = 730
)

// normalizeQuestionPayloads validates and normalizes a slice of question payloads.
func normalizeQuestionPayloads(questions []domain.QuestionPayload) ([]domain.QuestionPayload, error) {
	if len(questions) == 0 {
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
			ID             string `json:"id"`
			IsCorrect      bool   `json:"is_correct"`
			IsCorrectCamel bool   `json:"isCorrect"`
		} `json:"options"`
		Correct              *bool    `json:"correct"`
		AcceptedAnswers      []string `json:"accepted_answers"`
		AcceptedAnswersCamel []string `json:"acceptedAnswers"`
	}

	if err := json.Unmarshal(config, &payload); err != nil {
		return domain.FieldValidationError("Проверьте поля формы",
			domain.ValidationField(field, "invalid_json", "Настройки вопроса должны быть корректным JSON"))
	}

	var validation fieldValidationBuilder
	switch questionType {
	case domain.QuestionTypeSingleChoice:
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
		acceptedAnswers := payload.AcceptedAnswers
		if len(acceptedAnswers) == 0 {
			acceptedAnswers = payload.AcceptedAnswersCamel
		}
		if len(normalizeStringSlice(acceptedAnswers)) == 0 {
			validation.add(field+".accepted_answers", "required", "Добавьте минимум один правильный текстовый ответ")
		}
	}

	return validation.err()
}

func validateChoiceOptions(options []struct {
	ID             string `json:"id"`
	IsCorrect      bool   `json:"is_correct"`
	IsCorrectCamel bool   `json:"isCorrect"`
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
		if option.IsCorrect || option.IsCorrectCamel {
			correctCount++
		}
	}

	return correctCount, true
}

func normalizeQuizScoringAndAttempts(
	passingScore int,
	quizMinutes int,
	maxAttempts int,
	retakeCooldownDays int,
	validation *fieldValidationBuilder,
) (int, int, int, int) {
	if passingScore <= 0 {
		passingScore = defaultQuizPassingScore
	}
	validation.addIntRange("quiz_pass_percent", passingScore, 0, 100, "Значение проходного процента")

	if quizMinutes < 0 {
		quizMinutes = 0
	}

	if maxAttempts <= 0 {
		maxAttempts = defaultQuizMaxAttempts
	}

	if retakeCooldownDays < 0 {
		retakeCooldownDays = defaultQuizRetakeCooldownDays
	}
	if retakeCooldownDays > maxQuizRetakeCooldownDays {
		validation.add("retake_cooldown_days", "out_of_range", "Пауза перед новой сдачей не может быть больше 730 дней")
	}

	return passingScore, quizMinutes, maxAttempts, retakeCooldownDays
}
