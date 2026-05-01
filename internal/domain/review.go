package domain

import "time"

type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusApproved ReviewStatus = "approved"
	ReviewStatusRejected ReviewStatus = "rejected"
)

func (s ReviewStatus) IsValid() bool {
	switch s {
	case ReviewStatusPending, ReviewStatusApproved, ReviewStatusRejected:
		return true
	default:
		return false
	}
}

type Review struct {
	ID          string       `json:"id"`
	CourseID    string       `json:"course_id"`
	UserID      *string      `json:"user_id,omitempty"`
	Rating      int16        `json:"rating"`
	Text        *string      `json:"text,omitempty"`
	Status      ReviewStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	ModeratedAt *time.Time   `json:"moderated_at,omitempty"`
}

type CreateReviewParams struct {
	CourseID string  `json:"course_id"`
	UserID   *string `json:"user_id,omitempty"`
	Rating   int16   `json:"rating"`
	Text     *string `json:"text,omitempty"`
}

type ModerateReviewParams struct {
	ID     string       `json:"id"`
	Status ReviewStatus `json:"status"`
}

type ReviewListFilter struct {
	CourseID *string
	UserID   *string
	Status   *ReviewStatus
	Limit    int
	Offset   int
}
