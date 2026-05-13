package usecase

import (
	"testing"

	"lms-arvand-backend/internal/domain"
)

func TestNormalizeQuestionPayloads_AllowsEmpty(t *testing.T) {
	result, err := normalizeQuestionPayloads(nil)
	if err != nil {
		t.Fatalf("expected nil questions to be allowed, got error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d", len(result))
	}
}

func TestNormalizeQuestionPayloads_DefaultsPosition(t *testing.T) {
	questions := []domain.QuestionPayload{
		{
			Type:   domain.QuestionTypeSingleChoice,
			Prompt: domain.MultiLangText{RU: "Вопрос", TJ: "Савол"},
			Config: mustJSON(t, map[string]any{
				"options": []map[string]any{
					{"id": "a", "is_correct": true},
					{"id": "b", "is_correct": false},
				},
			}),
		},
	}

	result, err := normalizeQuestionPayloads(questions)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 question, got %d", len(result))
	}
	if result[0].Position != 1 {
		t.Fatalf("expected position 1, got %d", result[0].Position)
	}
}

func TestNormalizeQuestionPayloads_DefaultsPoints(t *testing.T) {
	questions := []domain.QuestionPayload{
		{
			Position: 1,
			Type:     domain.QuestionTypeSingleChoice,
			Prompt:   domain.MultiLangText{RU: "Вопрос", TJ: "Савол"},
			Points:   0,
			Config: mustJSON(t, map[string]any{
				"options": []map[string]any{
					{"id": "a", "is_correct": true},
				},
			}),
		},
	}

	result, err := normalizeQuestionPayloads(questions)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if result[0].Points != 1 {
		t.Fatalf("expected default points 1, got %v", result[0].Points)
	}
}

func TestNormalizeQuestionPayloads_RejectsDuplicatePositions(t *testing.T) {
	questions := []domain.QuestionPayload{
		{
			Position: 1,
			Type:     domain.QuestionTypeSingleChoice,
			Prompt:   domain.MultiLangText{RU: "Q1", TJ: "Q1"},
			Config: mustJSON(t, map[string]any{
				"options": []map[string]any{{"id": "a", "is_correct": true}},
			}),
		},
		{
			Position: 1,
			Type:     domain.QuestionTypeSingleChoice,
			Prompt:   domain.MultiLangText{RU: "Q2", TJ: "Q2"},
			Config: mustJSON(t, map[string]any{
				"options": []map[string]any{{"id": "a", "is_correct": true}},
			}),
		},
	}

	_, err := normalizeQuestionPayloads(questions)
	if err == nil {
		t.Fatalf("expected error for duplicate positions")
	}
}

func TestNormalizeQuizScoringDefaults(t *testing.T) {
	score, minutes, maxAttempts, cooldown := normalizeQuizScoringAndAttempts(0, 0, 0, 0, &fieldValidationBuilder{})
	if score != 80 {
		t.Fatalf("expected default passing score 80, got %d", score)
	}
	if minutes != 0 {
		t.Fatalf("expected quiz minutes 0, got %d", minutes)
	}
	if maxAttempts != 3 {
		t.Fatalf("expected default max attempts 3, got %d", maxAttempts)
	}
	if cooldown != 0 {
		t.Fatalf("expected default retake cooldown 0, got %d", cooldown)
	}
}
