package http

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type courseResponse struct {
	ID               string               `json:"id"`
	Title            domain.MultiLangText `json:"title"`
	Description      domain.MultiLangText `json:"description"`
	CoverImageURL    *string              `json:"cover_image_url,omitempty"`
	VideoURL         *string              `json:"video_url,omitempty"`
	QuizID           *string              `json:"quiz_id,omitempty"`
	Category         *string              `json:"category,omitempty"`
	EstimatedMinutes *int                 `json:"estimated_minutes,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

type quizResponse struct {
	ID                 string               `json:"id"`
	Title              domain.MultiLangText `json:"title"`
	Description        domain.MultiLangText `json:"description"`
	CourseID           *string              `json:"course_id,omitempty"`
	Category           *string              `json:"category,omitempty"`
	TimeLimitMinutes   *int                 `json:"time_limit_minutes,omitempty"`
	PassingScore       int                  `json:"passing_score"`
	PassingPoints      float64              `json:"passing_points"`
	MaxAttempts        int                  `json:"max_attempts"`
	RetakeCooldownDays int                  `json:"retake_cooldown_days"`
	AllowRetry         bool                 `json:"allow_retry"`
	Questions          []questionResponse   `json:"questions,omitempty"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type questionResponse struct {
	ID        string               `json:"id"`
	Position  int                  `json:"position"`
	Type      domain.QuestionType  `json:"type"`
	Prompt    domain.MultiLangText `json:"prompt"`
	Points    float64              `json:"points"`
	Required  bool                 `json:"required"`
	Config    json.RawMessage      `json:"config"`
	CreatedAt time.Time            `json:"created_at"`
}

type attemptResponse struct {
	ID            string                      `json:"id"`
	QuizID        string                      `json:"quiz_id"`
	UserID        *string                     `json:"user_id,omitempty"`
	StartedAt     time.Time                   `json:"started_at"`
	FinishedAt    *time.Time                  `json:"finished_at,omitempty"`
	TotalEarned   float64                     `json:"total_earned"`
	TotalMax      float64                     `json:"total_max"`
	ScorePercent  float64                     `json:"score_percent"`
	Passed        bool                        `json:"passed"`
	ReviewedAt    *time.Time                  `json:"reviewed_at,omitempty"`
	ReviewerID    *string                     `json:"reviewer_id,omitempty"`
	ReviewComment *string                     `json:"review_comment,omitempty"`
	ManualPassed  *bool                       `json:"manual_passed,omitempty"`
	ReviewScores  []domain.AttemptReviewScore `json:"review_scores,omitempty"`
}

func toCourseResponse(course domain.Course) courseResponse {
	return courseResponse{
		ID:               course.ID,
		Title:            course.Title,
		Description:      course.Description,
		CoverImageURL:    course.CoverImageURL,
		VideoURL:         course.VideoURL,
		QuizID:           course.QuizID,
		Category:         course.Category,
		EstimatedMinutes: course.EstimatedMinutes,
		CreatedAt:        course.CreatedAt,
		UpdatedAt:        course.UpdatedAt,
	}
}

func toCourseResponses(courses []domain.Course) []courseResponse {
	responses := make([]courseResponse, 0, len(courses))
	for _, course := range courses {
		responses = append(responses, toCourseResponse(course))
	}
	return responses
}

func toQuizResponse(quiz domain.Quiz) quizResponse {
	return quizResponse{
		ID:                 quiz.ID,
		Title:              quiz.Title,
		Description:        quiz.Description,
		CourseID:           quiz.CourseID,
		Category:           quiz.Category,
		TimeLimitMinutes:   quiz.TimeLimitMinutes,
		PassingScore:       quiz.PassingScore,
		PassingPoints:      quiz.PassingPoints,
		MaxAttempts:        quiz.MaxAttempts,
		RetakeCooldownDays: quiz.RetakeCooldownDays,
		AllowRetry:         quiz.AllowRetry,
		Questions:          toQuestionResponses(quiz.Questions),
		CreatedAt:          quiz.CreatedAt,
		UpdatedAt:          quiz.UpdatedAt,
	}
}

func toQuizResponses(quizzes []domain.Quiz) []quizResponse {
	responses := make([]quizResponse, 0, len(quizzes))
	for _, quiz := range quizzes {
		responses = append(responses, toQuizResponse(quiz))
	}
	return responses
}

func toQuestionResponses(questions []domain.Question) []questionResponse {
	if len(questions) == 0 {
		return nil
	}

	responses := make([]questionResponse, 0, len(questions))
	for _, question := range questions {
		responses = append(responses, questionResponse{
			ID:        question.ID,
			Position:  question.Position,
			Type:      question.Type,
			Prompt:    question.Prompt,
			Points:    question.Points,
			Required:  question.Required,
			Config:    sanitizeQuestionConfig(question.Config),
			CreatedAt: question.CreatedAt,
		})
	}
	return responses
}

func toAttemptResponse(attempt domain.Attempt) attemptResponse {
	return attemptResponse{
		ID:            attempt.ID,
		QuizID:        attempt.QuizID,
		UserID:        attempt.UserID,
		StartedAt:     attempt.StartedAt,
		FinishedAt:    attempt.FinishedAt,
		TotalEarned:   attempt.TotalEarned,
		TotalMax:      attempt.TotalMax,
		ScorePercent:  attempt.ScorePercent,
		Passed:        attempt.Passed,
		ReviewedAt:    attempt.ReviewedAt,
		ReviewerID:    attempt.ReviewerID,
		ReviewComment: attempt.ReviewComment,
		ManualPassed:  attempt.ManualPassed,
		ReviewScores:  attempt.ReviewScores,
	}
}

func toAttemptResponses(attempts []domain.Attempt) []attemptResponse {
	responses := make([]attemptResponse, 0, len(attempts))
	for _, attempt := range attempts {
		responses = append(responses, toAttemptResponse(attempt))
	}
	return responses
}

func sanitizeQuestionConfig(raw json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`)
	}

	var payload any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return json.RawMessage(`{}`)
	}

	if payload == nil {
		return json.RawMessage(`{}`)
	}

	removeAnswerKeys(payload)

	sanitized, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return sanitized
}

func removeAnswerKeys(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isAnswerKey(key) {
				delete(typed, key)
				continue
			}
			removeAnswerKeys(child)
		}
	case []any:
		for _, child := range typed {
			removeAnswerKeys(child)
		}
	}
}

func isAnswerKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "is_correct",
		"iscorrect",
		"correct",
		"accepted_answer",
		"accepted_answers",
		"acceptedanswer",
		"acceptedanswers",
		"correct_answer",
		"correct_answers",
		"correctanswer",
		"correctanswers",
		"correct_id",
		"correct_ids",
		"correctid",
		"correctids",
		"correct_option_id",
		"correct_option_ids",
		"correctoptionid",
		"correctoptionids",
		"answer_key",
		"answer_keys",
		"answerkey",
		"answerkeys",
		"expected_answer",
		"expected_answers",
		"expectedanswer",
		"expectedanswers":
		return true
	default:
		return false
	}
}
