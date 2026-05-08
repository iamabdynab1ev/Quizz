package domain

import (
	"encoding/json"
	"time"
)

type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "single_choice"
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeTrueFalse      QuestionType = "true_false"
	QuestionTypeShortAnswer    QuestionType = "short_answer"
	QuestionTypeFillBlank      QuestionType = "fill_blank"
)

func (t QuestionType) IsValid() bool {
	switch t {
	case QuestionTypeSingleChoice,
		QuestionTypeMultipleChoice,
		QuestionTypeTrueFalse,
		QuestionTypeShortAnswer,
		QuestionTypeFillBlank:
		return true
	default:
		return false
	}
}

type Question struct {
	ID        string          `json:"id"`
	Position  int             `json:"position"`
	Type      QuestionType    `json:"type"`
	Prompt    MultiLangText   `json:"prompt"`
	Points    float64         `json:"points"`
	Required  bool            `json:"required"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
}

type QuestionPayload struct {
	Position int             `json:"position"`
	Type     QuestionType    `json:"type"`
	Prompt   MultiLangText   `json:"prompt"`
	Points   float64         `json:"points"`
	Required bool            `json:"required"`
	Config   json.RawMessage `json:"config"`
}
