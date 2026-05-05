package usecase

import (
	"testing"

	"lms-arvand-backend/internal/domain"
)

func TestNormalizeCreateQuizParams_AllowsEmptyQuestions(t *testing.T) {
	params := domain.CreateQuizParams{
		Title:       domain.MultiLangText{RU: "Тест", TJ: "Санҷиш"},
		Description: domain.MultiLangText{RU: "Описание", TJ: "Тавсиф"},
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

	if len(normalized.Questions) != 0 {
		t.Fatalf("expected empty questions, got %d", len(normalized.Questions))
	}
}
