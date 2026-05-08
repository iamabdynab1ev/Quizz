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
	email := "client@local.test"

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
		"certificate_passing_score",
		"reviews_enabled",
	)

	quizBody := mustMarshalString(t, toCourseAsQuizResponse(domain.Course{
		ID:                 "quiz-1",
		Title:              domain.MultiLangText{RU: "Тест", TJ: "Тест"},
		Description:        domain.MultiLangText{RU: "Описание", TJ: "Тавсиф"},
		Status:             domain.CourseStatusPublished,
		Platforms:          []domain.Platform{domain.PlatformWeb},
		QuizPassPercent:    70,
		MaxAttempts:        3,
		RetakeCooldownDays: 30,
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
	}, false))
	assertJSONDoesNotContain(t, quizBody,
		"status",
		"platforms",
		"passing_points",
		"shuffle_questions",
		"show_results",
		"is_correct",
		"correct",
		"accepted_answers",
	)

	attemptBody := mustMarshalString(t, toAttemptResponse(domain.Attempt{
		ID:                "attempt-1",
		CourseID:          "course-1",
		QuestionsSnapshot: json.RawMessage(`[{"config":{"options":[{"id":"a","is_correct":true}]}}]`),
		AnswersData:       json.RawMessage(`[{"question_id":"question-1"}]`),
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
		IsAdmin:   false,
		IsMale:    func() *bool { v := true; return &v }(),
		FirstName: "Ali",
		LastName:  "Karimov",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}))
	assertJSONDoesNotContain(t, userBody, "is_active", "is_super_admin")

	superAdminBody := mustMarshalString(t, toUserResponse(domain.User{
		ID:           "super-admin-1",
		Email:        &email,
		IsAdmin:      true,
		IsSuperAdmin: true,
		FirstName:    "System",
		LastName:     "Admin",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}))
	if !strings.Contains(superAdminBody, `"is_super_admin":true`) {
		t.Fatalf("super admin response must expose is_super_admin=true, got %s", superAdminBody)
	}
}

func TestAdminQuizResponseIncludesAnswerKeys(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	body := mustMarshalString(t, toCourseAsQuizResponse(domain.Course{
		ID:              "quiz-1",
		Title:           domain.MultiLangText{RU: "Test", TJ: "Test"},
		Description:     domain.MultiLangText{RU: "Description", TJ: "Description"},
		QuizPassPercent: 70,
		MaxAttempts:     3,
		Questions: []domain.Question{{
			ID:       "question-1",
			Position: 1,
			Type:     domain.QuestionTypeSingleChoice,
			Prompt:   domain.MultiLangText{RU: "Question", TJ: "Question"},
			Points:   1,
			Required: true,
			Config: json.RawMessage(`{
				"options":[
					{"id":"a","text":{"ru":"A","tj":"A"},"is_correct":true},
					{"id":"b","text":{"ru":"B","tj":"B"},"is_correct":false}
				]
			}`),
			CreatedAt: now,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}, true))

	if !strings.Contains(body, `"is_correct":true`) {
		t.Fatalf("admin quiz response must include answer keys, got %s", body)
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
