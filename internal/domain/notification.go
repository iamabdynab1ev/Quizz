package domain

import "time"

type NotificationType string

const (
	NotificationTypeCoursePublished   NotificationType = "course.published"
	NotificationTypeCertificateIssued NotificationType = "certificate.issued"
	NotificationTypeReviewApproved    NotificationType = "review.approved"
	NotificationTypeEnrollmentCreated NotificationType = "enrollment.created"
	NotificationTypeSystem            NotificationType = "system"
)

func (t NotificationType) IsValid() bool {
	switch t {
	case NotificationTypeCoursePublished,
		NotificationTypeCertificateIssued,
		NotificationTypeReviewApproved,
		NotificationTypeEnrollmentCreated,
		NotificationTypeSystem:
		return true
	default:
		return false
	}
}

type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     MultiLangText    `json:"title"`
	Body      MultiLangText    `json:"body"`
	Link      *string          `json:"link,omitempty"`
	Read      bool             `json:"read"`
	CreatedAt time.Time        `json:"created_at"`
}

type CreateNotificationParams struct {
	UserID string           `json:"user_id"`
	Type   NotificationType `json:"type"`
	Title  MultiLangText    `json:"title"`
	Body   MultiLangText    `json:"body"`
	Link   *string          `json:"link,omitempty"`
}

type NotificationListFilter struct {
	UserID *string
	Type   *NotificationType
	Read   *bool
	Limit  int
	Offset int
}
