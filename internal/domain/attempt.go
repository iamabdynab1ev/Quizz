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
	OrderedOptionIDs  []string          `json:"ordered_option_ids,omitempty"`
	MatchedPairs      map[string]string `json:"matched_pairs,omitempty"`
}

type AttemptReviewScore struct {
	QuestionID string  `json:"question_id"`
	Points     float64 `json:"points"`
	Comment    *string `json:"comment,omitempty"`
}

type SubmitAttemptParams struct {
	QuizID    string          `json:"quiz_id"`
	UserID    string          `json:"user_id"`
	StartedAt *time.Time      `json:"started_at,omitempty"`
	Answers   []AttemptAnswer `json:"answers"`
}

type CreateAttemptRecordParams struct {
	QuizID            string
	UserID            string
	StartedAt         time.Time
	FinishedAt        time.Time
	QuestionsSnapshot json.RawMessage
	AnswersData       json.RawMessage
	TotalEarned       float64
	TotalMax          float64
	ScorePercent      float64
	Passed            bool
	NeedsReview       bool
}

type AttemptWindow struct {
	Count             int
	EarliestStartedAt *time.Time
}

type Attempt struct {
	ID                string               `json:"id"`
	QuizID            string               `json:"quiz_id"`
	UserID            *string              `json:"user_id,omitempty"`
	StartedAt         time.Time            `json:"started_at"`
	FinishedAt        *time.Time           `json:"finished_at,omitempty"`
	QuestionsSnapshot json.RawMessage      `json:"questions_snapshot"`
	AnswersData       json.RawMessage      `json:"answers_data"`
	TotalEarned       float64              `json:"total_earned"`
	TotalMax          float64              `json:"total_max"`
	ScorePercent      float64              `json:"score_percent"`
	Passed            bool                 `json:"passed"`
	NeedsReview       bool                 `json:"needs_review"`
	ReviewedAt        *time.Time           `json:"reviewed_at,omitempty"`
	ReviewerID        *string              `json:"reviewer_id,omitempty"`
	ReviewComment     *string              `json:"review_comment,omitempty"`
	ManualPassed      *bool                `json:"manual_passed,omitempty"`
	ReviewScores      []AttemptReviewScore `json:"review_scores,omitempty"`
}

type AttemptListFilter struct {
	QuizID *string
	UserID *string
	Limit  int
	Offset int
}

type ReviewAttemptParams struct {
	AttemptID  string               `json:"attempt_id"`
	ReviewerID string               `json:"reviewer_id"`
	Passed     bool                 `json:"passed"`
	Comment    *string              `json:"comment,omitempty"`
	Scores     []AttemptReviewScore `json:"scores,omitempty"`

	TotalEarned  float64         `json:"-"`
	ScorePercent float64         `json:"-"`
	ReviewScores json.RawMessage `json:"-"`
}
