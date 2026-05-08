package domain

import (
	"encoding/json"
	"time"
)

type AttemptAnswer struct {
	QuestionID        string            `json:"question_id"`
	SelectedOptionIDs []string          `json:"selected_option_ids,omitempty"`
	TextAnswer        *string           `json:"text_answer,omitempty"`
	BooleanAnswer     *bool             `json:"boolean_answer,omitempty"`
}

type SubmitAttemptParams struct {
	CourseID  string          `json:"course_id"`
	UserID    string          `json:"user_id"`
	StartedAt *time.Time      `json:"started_at,omitempty"`
	Answers   []AttemptAnswer `json:"answers"`
}

type CreateAttemptRecordParams struct {
	CourseID          string
	UserID            string
	StartedAt         time.Time
	FinishedAt        time.Time
	QuestionsSnapshot json.RawMessage
	AnswersData       json.RawMessage
	TotalEarned       float64
	TotalMax          float64
	ScorePercent      float64
	Passed            bool
}

type AttemptWindow struct {
	Count             int
	EarliestStartedAt *time.Time
}

type Attempt struct {
	ID                string          `json:"id"`
	CourseID          string          `json:"course_id"`
	UserID            *string         `json:"user_id,omitempty"`
	StartedAt         time.Time       `json:"started_at"`
	FinishedAt        *time.Time      `json:"finished_at,omitempty"`
	QuestionsSnapshot json.RawMessage `json:"questions_snapshot"`
	AnswersData       json.RawMessage `json:"answers_data"`
	TotalEarned       float64         `json:"total_earned"`
	TotalMax          float64         `json:"total_max"`
	ScorePercent      float64         `json:"score_percent"`
	Passed            bool            `json:"passed"`
}

type AttemptListFilter struct {
	CourseID *string
	UserID   *string
	Limit    int
	Offset   int
}
