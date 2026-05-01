package domain

import "encoding/json"

type ContentBlockType string

const (
	ContentBlockTypeText  ContentBlockType = "text"
	ContentBlockTypeURL   ContentBlockType = "url"
	ContentBlockTypeVideo ContentBlockType = "video"
	ContentBlockTypePhoto ContentBlockType = "photo"
	ContentBlockTypeFile  ContentBlockType = "file"
)

func (t ContentBlockType) IsValid() bool {
	switch t {
	case ContentBlockTypeText, ContentBlockTypeURL, ContentBlockTypeVideo, ContentBlockTypePhoto, ContentBlockTypeFile:
		return true
	default:
		return false
	}
}

type ContentBlock struct {
	ID       string           `json:"id"`
	CourseID *string          `json:"course_id,omitempty"`
	ModuleID *string          `json:"module_id,omitempty"`
	Position int              `json:"position"`
	Type     ContentBlockType `json:"type"`
	Title    MultiLangText    `json:"title"`
	Payload  json.RawMessage  `json:"payload"`
}

type CreateContentBlockParams struct {
	CourseID *string          `json:"course_id,omitempty"`
	ModuleID *string          `json:"module_id,omitempty"`
	Position int              `json:"position"`
	Type     ContentBlockType `json:"type"`
	Title    MultiLangText    `json:"title"`
	Payload  json.RawMessage  `json:"payload"`
}

type UpdateContentBlockParams struct {
	ID       string           `json:"id"`
	CourseID *string          `json:"course_id,omitempty"`
	ModuleID *string          `json:"module_id,omitempty"`
	Position int              `json:"position"`
	Type     ContentBlockType `json:"type"`
	Title    MultiLangText    `json:"title"`
	Payload  json.RawMessage  `json:"payload"`
}

type ContentBlockListFilter struct {
	CourseID *string
	ModuleID *string
}
