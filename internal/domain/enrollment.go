package domain

import "time"

type EnrollmentStatus string

const (
	EnrollmentStatusActive    EnrollmentStatus = "active"
	EnrollmentStatusCompleted EnrollmentStatus = "completed"
	EnrollmentStatusDropped   EnrollmentStatus = "dropped"
)

func (s EnrollmentStatus) IsValid() bool {
	switch s {
	case EnrollmentStatusActive, EnrollmentStatusCompleted, EnrollmentStatusDropped:
		return true
	default:
		return false
	}
}

type Enrollment struct {
	ID          string           `json:"id"`
	CourseID    string           `json:"course_id"`
	UserID      *string          `json:"user_id,omitempty"`
	Status      EnrollmentStatus `json:"status"`
	EnrolledAt  time.Time        `json:"enrolled_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
}

type CreateEnrollmentParams struct {
	CourseID string `json:"course_id"`
	UserID   string `json:"user_id"`
}

type EnrollmentListFilter struct {
	CourseID *string
	UserID   *string
	Status   *EnrollmentStatus
	Limit    int
	Offset   int
}
