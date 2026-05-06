package usecase

import (
	"errors"
	"testing"

	"lms-arvand-backend/internal/domain"
)

func TestNormalizeCreateQuizParams_AllowsEmptyQuestions(t *testing.T) {
	params := domain.CreateQuizParams{
		Title:        domain.MultiLangText{RU: "Тест", TJ: "Санҷиш"},
		Description:  domain.MultiLangText{RU: "Описание", TJ: "Тавсиф"},
		PassingScore: 60,
	}

	normalized, err := normalizeCreateQuizParams(params)
	if err != nil {
		t.Fatalf("expected empty questions to be allowed, got error: %v", err)
	}

	if normalized.Status != domain.QuizStatusDraft {
		t.Fatalf("expected draft status, got %s", normalized.Status)
	}

	if normalized.MaxAttempts != 3 {
		t.Fatalf("expected default max attempts 3, got %d", normalized.MaxAttempts)
	}
	if normalized.RetakeCooldownDays != 30 {
		t.Fatalf("expected default retake cooldown 30 days, got %d", normalized.RetakeCooldownDays)
	}

	if len(normalized.Questions) != 0 {
		t.Fatalf("expected empty questions, got %d", len(normalized.Questions))
	}
}

func TestNormalizeCreateQuizParamsRejectsPassingPointsAboveTotal(t *testing.T) {
	params := domain.CreateQuizParams{
		Title:         domain.MultiLangText{RU: "Тест", TJ: "Санҷиш"},
		Description:   domain.MultiLangText{RU: "Описание", TJ: "Тавсиф"},
		PassingPoints: 11,
		Questions: []domain.QuestionPayload{
			{
				Position: 1,
				Type:     domain.QuestionTypeSingleChoice,
				Prompt:   domain.MultiLangText{RU: "Вопрос", TJ: "Савол"},
				Points:   10,
				Config: mustJSON(t, map[string]any{
					"options": []map[string]any{
						{"id": "a", "is_correct": true},
						{"id": "b", "is_correct": false},
					},
				}),
			},
		},
	}

	_, err := normalizeCreateQuizParams(params)
	if err == nil {
		t.Fatalf("expected passing points validation error")
	}

	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if len(appErr.Fields) == 0 || appErr.Fields[0].Field != "passing_points" {
		t.Fatalf("expected passing_points field error, got %#v", appErr.Fields)
	}
}

func TestNormalizeCreateQuizParamsComputesPassingPointsFromLegacyPercent(t *testing.T) {
	params := domain.CreateQuizParams{
		Title:        domain.MultiLangText{RU: "Тест", TJ: "Санҷиш"},
		Description:  domain.MultiLangText{RU: "Описание", TJ: "Тавсиф"},
		PassingScore: 80,
		Questions: []domain.QuestionPayload{
			{
				Position: 1,
				Type:     domain.QuestionTypeSingleChoice,
				Prompt:   domain.MultiLangText{RU: "Вопрос", TJ: "Савол"},
				Points:   10,
				Config: mustJSON(t, map[string]any{
					"options": []map[string]any{
						{"id": "a", "is_correct": true},
						{"id": "b", "is_correct": false},
					},
				}),
			},
		},
	}

	normalized, err := normalizeCreateQuizParams(params)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if normalized.PassingPoints != 8 {
		t.Fatalf("expected passing points 8, got %v", normalized.PassingPoints)
	}
}
