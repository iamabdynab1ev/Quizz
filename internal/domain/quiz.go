package domain

import (
	"encoding/json"
	"time"
)

type QuizStatus string

const (
	QuizStatusDraft     QuizStatus = "draft"
	QuizStatusPublished QuizStatus = "published"
	QuizStatusArchived  QuizStatus = "archived"
)

func (s QuizStatus) IsValid() bool {
	switch s {
	case QuizStatusDraft, QuizStatusPublished, QuizStatusArchived:
		return true
	default:
		return false
	}
}

type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "single_choice"
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeTrueFalse      QuestionType = "true_false"
	QuestionTypeShortAnswer    QuestionType = "short_answer"
	QuestionTypeLongText       QuestionType = "long_text"
	QuestionTypeMatching       QuestionType = "matching"
	QuestionTypeOrdering       QuestionType = "ordering"
	QuestionTypeFillBlank      QuestionType = "fill_blank"
	QuestionTypeImageChoice    QuestionType = "image_choice"
	QuestionTypeAudio          QuestionType = "audio"
	QuestionTypeVideo          QuestionType = "video"
	QuestionTypeCode           QuestionType = "code"
)

func (t QuestionType) IsValid() bool {
	switch t {
	case QuestionTypeSingleChoice,
		QuestionTypeMultipleChoice,
		QuestionTypeTrueFalse,
		QuestionTypeShortAnswer,
		QuestionTypeLongText,
		QuestionTypeMatching,
		QuestionTypeOrdering,
		QuestionTypeFillBlank,
		QuestionTypeImageChoice,
		QuestionTypeAudio,
		QuestionTypeVideo,
		QuestionTypeCode:
		return true
	default:
		return false
	}
}

type Question struct {
	ID          string          `json:"id"`
	Position    int             `json:"position"`
	Type        QuestionType    `json:"type"`
	Prompt      MultiLangText   `json:"prompt"`
	Explanation MultiLangText   `json:"explanation"`
	Points      float64         `json:"points"`
	Required    bool            `json:"required"`
	Config      json.RawMessage `json:"config"`
	CreatedAt   time.Time       `json:"created_at"`
}

type QuestionPayload struct {
	Position    int             `json:"position"`
	Type        QuestionType    `json:"type"`
	Prompt      MultiLangText   `json:"prompt"`
	Explanation MultiLangText   `json:"explanation"`
	Points      float64         `json:"points"`
	Required    bool            `json:"required"`
	Config      json.RawMessage `json:"config"`
}

type Quiz struct {
	ID               string        `json:"id"`
	Title            MultiLangText `json:"title"`
	Description      MultiLangText `json:"description"`
	Category         *string       `json:"category,omitempty"`
	Status           QuizStatus    `json:"status"`
	Platforms        []Platform    `json:"platforms"`
	TimeLimitMinutes *int          `json:"time_limit_minutes,omitempty"`
	PassingScore     int           `json:"passing_score"`
	MaxAttempts      int           `json:"max_attempts"`
	ShuffleQuestions bool          `json:"shuffle_questions"`
	ShowResults      bool          `json:"show_results"`
	AllowRetry       bool          `json:"allow_retry"`
	Questions        []Question    `json:"questions,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type CreateQuizParams struct {
	Title            MultiLangText     `json:"title"`
	Description      MultiLangText     `json:"description"`
	Category         *string           `json:"category,omitempty"`
	Status           QuizStatus        `json:"status"`
	Platforms        []Platform        `json:"platforms"`
	TimeLimitMinutes *int              `json:"time_limit_minutes,omitempty"`
	PassingScore     int               `json:"passing_score"`
	MaxAttempts      int               `json:"max_attempts"`
	ShuffleQuestions bool              `json:"shuffle_questions"`
	ShowResults      bool              `json:"show_results"`
	AllowRetry       bool              `json:"allow_retry"`
	Questions        []QuestionPayload `json:"questions"`
}

type UpdateQuizParams struct {
	ID               string            `json:"id"`
	Title            MultiLangText     `json:"title"`
	Description      MultiLangText     `json:"description"`
	Category         *string           `json:"category,omitempty"`
	Status           QuizStatus        `json:"status"`
	Platforms        []Platform        `json:"platforms"`
	TimeLimitMinutes *int              `json:"time_limit_minutes,omitempty"`
	PassingScore     int               `json:"passing_score"`
	MaxAttempts      int               `json:"max_attempts"`
	ShuffleQuestions bool              `json:"shuffle_questions"`
	ShowResults      bool              `json:"show_results"`
	AllowRetry       bool              `json:"allow_retry"`
	Questions        []QuestionPayload `json:"questions"`
}

type QuizListFilter struct {
	Search   string
	Status   *QuizStatus
	Category *string
	Platform *Platform
	Limit    int
	Offset   int
}
