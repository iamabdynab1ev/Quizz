package http

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
)

func TestPublicResponsesDoNotExposeInternalFields(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	email := "student@local.test"

	courseBody := mustMarshalString(t, toCourseResponse(domain.Course{
		ID:                      "course-1",
		Title:                   domain.MultiLangText{RU: "Курс", TJ: "Курс"},
		Description:             domain.MultiLangText{RU: "Описание", TJ: "Тавсиф"},
		Status:                  domain.CourseStatusPublished,
		Platforms:               []domain.Platform{domain.PlatformWeb},
		CertificateEnabled:      true,
		CertificatePassingScore: 80,
		ReviewsEnabled:          true,
		CreatedAt:               now,
		UpdatedAt:               now,
	}))
	assertJSONDoesNotContain(t, courseBody,
		"status",
		"platforms",
		"certificate_enabled",
		"certificate_passing_score",
		"reviews_enabled",
	)

	quizBody := mustMarshalString(t, toQuizResponse(domain.Quiz{
		ID:               "quiz-1",
		Title:            domain.MultiLangText{RU: "Тест", TJ: "Тест"},
		Description:      domain.MultiLangText{RU: "Описание", TJ: "Тавсиф"},
		Status:           domain.QuizStatusPublished,
		Platforms:        []domain.Platform{domain.PlatformWeb},
		PassingScore:     70,
		MaxAttempts:      3,
		ShuffleQuestions: true,
		ShowResults:      true,
		AllowRetry:       true,
		Questions: []domain.Question{{
			ID:       "question-1",
			Position: 1,
			Type:     domain.QuestionTypeSingleChoice,
			Prompt:   domain.MultiLangText{RU: "Вопрос", TJ: "Савол"},
			Points:   1,
			Required: true,
			Config: json.RawMessage(`{
				"options":[
					{"id":"a","text":{"ru":"A","tj":"A"},"is_correct":true},
					{"id":"b","text":{"ru":"B","tj":"B"},"is_correct":false}
				],
				"correct": true,
				"accepted_answers": ["answer"]
			}`),
			CreatedAt: now,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}))
	assertJSONDoesNotContain(t, quizBody,
		"status",
		"platforms",
		"shuffle_questions",
		"show_results",
		"is_correct",
		"correct",
		"accepted_answers",
	)

	attemptBody := mustMarshalString(t, toAttemptResponse(domain.Attempt{
		ID:                "attempt-1",
		QuizID:            "quiz-1",
		StartedAt:         now,
		QuestionsSnapshot: json.RawMessage(`[{"config":{"options":[{"id":"a","is_correct":true}]}}]`),
		AnswersData:       json.RawMessage(`[{"question_id":"question-1"}]`),
		NeedsReview:       true,
	}))
	assertJSONDoesNotContain(t, attemptBody,
		"questions_snapshot",
		"answers_data",
		"needs_review",
		"is_correct",
	)

	userBody := mustMarshalString(t, toUserResponse(domain.User{
		ID:        "user-1",
		Email:     &email,
		Role:      domain.UserRoleStudent,
		FirstName: "Ali",
		LastName:  "Karimov",
		Gender:    domain.GenderMale,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}))
	assertJSONDoesNotContain(t, userBody, "is_active", "is_super_admin")

	superAdminBody := mustMarshalString(t, toUserResponse(domain.User{
		ID:           "super-admin-1",
		Email:        &email,
		Role:         domain.UserRoleAdmin,
		IsSuperAdmin: true,
		FirstName:    "System",
		LastName:     "Admin",
		Gender:       domain.GenderUnspecified,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}))
	if !strings.Contains(superAdminBody, `"is_super_admin":true`) {
		t.Fatalf("super admin response must expose is_super_admin=true, got %s", superAdminBody)
	}
}

func mustMarshalString(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	return string(body)
}

func assertJSONDoesNotContain(t *testing.T, body string, forbiddenKeys ...string) {
	t.Helper()
	for _, key := range forbiddenKeys {
		if strings.Contains(body, `"`+key+`"`) {
			t.Fatalf("response contains forbidden key %q: %s", key, body)
		}
	}
}
