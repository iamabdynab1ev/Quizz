package domain

import "time"

type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusDisabled WebhookStatus = "disabled"
)

func (s WebhookStatus) IsValid() bool {
	switch s {
	case WebhookStatusActive, WebhookStatusDisabled:
		return true
	default:
		return false
	}
}

type AppEventType string

const (
	AppEventCourseCreated      AppEventType = "course.created"
	AppEventCourseUpdated      AppEventType = "course.updated"
	AppEventCourseDeleted      AppEventType = "course.deleted"
	AppEventCoursePublished    AppEventType = "course.published"
	AppEventTestCreated        AppEventType = "test.created"
	AppEventTestUpdated        AppEventType = "test.updated"
	AppEventTestDeleted        AppEventType = "test.deleted"
	AppEventUserCreated        AppEventType = "user.created"
	AppEventUserUpdated        AppEventType = "user.updated"
	AppEventUserDeleted        AppEventType = "user.deleted"
	AppEventEnrollmentCreated  AppEventType = "enrollment.created"
	AppEventEnrollmentComplete AppEventType = "enrollment.completed"
	AppEventAttemptFinished    AppEventType = "attempt.finished"
	AppEventAttemptPassed      AppEventType = "attempt.passed"
	AppEventAttemptFailed      AppEventType = "attempt.failed"
	AppEventCertificateIssued  AppEventType = "certificate.issued"
	AppEventCertificateRevoked AppEventType = "certificate.revoked"
	AppEventReviewCreated      AppEventType = "review.created"
	AppEventReviewApproved     AppEventType = "review.approved"
	AppEventReviewRejected     AppEventType = "review.rejected"
)

func (t AppEventType) IsValid() bool {
	switch t {
	case AppEventCourseCreated,
		AppEventCourseUpdated,
		AppEventCourseDeleted,
		AppEventCoursePublished,
		AppEventTestCreated,
		AppEventTestUpdated,
		AppEventTestDeleted,
		AppEventUserCreated,
		AppEventUserUpdated,
		AppEventUserDeleted,
		AppEventEnrollmentCreated,
		AppEventEnrollmentComplete,
		AppEventAttemptFinished,
		AppEventAttemptPassed,
		AppEventAttemptFailed,
		AppEventCertificateIssued,
		AppEventCertificateRevoked,
		AppEventReviewCreated,
		AppEventReviewApproved,
		AppEventReviewRejected:
		return true
	default:
		return false
	}
}

type Webhook struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	URL             string         `json:"url"`
	Events          []AppEventType `json:"events"`
	Secret          string         `json:"secret"`
	Status          WebhookStatus  `json:"status"`
	LastTriggeredAt *time.Time     `json:"last_triggered_at,omitempty"`
	LastStatusCode  *int           `json:"last_status_code,omitempty"`
	LastError       *string        `json:"last_error,omitempty"`
	Deliveries      int64          `json:"deliveries"`
	Failures        int64          `json:"failures"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type CreateWebhookParams struct {
	Name   string         `json:"name"`
	URL    string         `json:"url"`
	Events []AppEventType `json:"events"`
	Secret string         `json:"secret"`
	Status WebhookStatus  `json:"status"`
}

type UpdateWebhookParams struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	URL    string         `json:"url"`
	Events []AppEventType `json:"events"`
	Secret string         `json:"secret"`
	Status WebhookStatus  `json:"status"`
}

type WebhookListFilter struct {
	Status *WebhookStatus
	Limit  int
	Offset int
}
