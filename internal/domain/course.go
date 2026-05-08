package domain

import "time"

type CourseStatus string

const (
	CourseStatusDraft     CourseStatus = "draft"
	CourseStatusPublished CourseStatus = "published"
	CourseStatusArchived  CourseStatus = "archived"
)

func (s CourseStatus) IsValid() bool {
	switch s {
	case CourseStatusDraft, CourseStatusPublished, CourseStatusArchived:
		return true
	default:
		return false
	}
}

type Platform string

const (
	PlatformWeb      Platform = "web"
	PlatformMobile   Platform = "mobile"
	PlatformTelegram Platform = "telegram"
)

func (p Platform) IsValid() bool {
	switch p {
	case PlatformWeb, PlatformMobile, PlatformTelegram:
		return true
	default:
		return false
	}
}

type Course struct {
	ID                      string        `json:"id"`
	Title                   MultiLangText `json:"title"`
	Description             MultiLangText `json:"description"`
	CoverImageURL           *string       `json:"cover_image_url,omitempty"`
	VideoURL                *string       `json:"video_url,omitempty"`
	Category                *string       `json:"category,omitempty"`
	Status                  CourseStatus  `json:"status"`
	Platforms               []Platform    `json:"platforms"`
	EstimatedMinutes        *int          `json:"estimated_minutes,omitempty"`
	CertificateEnabled      bool          `json:"certificate_enabled"`
	CertificatePassingScore int           `json:"certificate_passing_score"`
	ReviewsEnabled          bool          `json:"reviews_enabled"`
	QuizPassPercent         int           `json:"quiz_pass_percent"`
	QuizMinutes             int           `json:"quiz_minutes"`
	MaxAttempts             int           `json:"max_attempts"`
	RetakeCooldownDays      int           `json:"retake_cooldown_days"`
	CreatedByUserID         *string       `json:"created_by_user_id,omitempty"`
	CreatedByName           string        `json:"created_by_name,omitempty"`
	Questions               []Question    `json:"questions,omitempty"`
	CreatedAt               time.Time     `json:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at"`
}

type CreateCourseParams struct {
	Title                   MultiLangText     `json:"title"`
	Description             MultiLangText     `json:"description"`
	CoverImageURL           *string           `json:"cover_image_url,omitempty"`
	VideoURL                *string           `json:"video_url,omitempty"`
	Category                *string           `json:"category,omitempty"`
	Status                  CourseStatus      `json:"status"`
	Platforms               []Platform        `json:"platforms"`
	EstimatedMinutes        *int              `json:"estimated_minutes,omitempty"`
	CertificateEnabled      *bool             `json:"certificate_enabled,omitempty"`
	CertificatePassingScore int               `json:"certificate_passing_score"`
	ReviewsEnabled          *bool             `json:"reviews_enabled,omitempty"`
	QuizPassPercent         int               `json:"quiz_pass_percent"`
	QuizMinutes             int               `json:"quiz_minutes"`
	MaxAttempts             int               `json:"max_attempts"`
	RetakeCooldownDays      int               `json:"retake_cooldown_days"`
	Questions               []QuestionPayload `json:"questions"`
	CreatedByUserID         *string           `json:"-"`
	CreatedByName           string            `json:"-"`
}

type UpdateCourseParams struct {
	ID                      string            `json:"id"`
	Title                   MultiLangText     `json:"title"`
	Description             MultiLangText     `json:"description"`
	CoverImageURL           *string           `json:"cover_image_url,omitempty"`
	VideoURL                *string           `json:"video_url,omitempty"`
	Category                *string           `json:"category,omitempty"`
	Status                  CourseStatus      `json:"status"`
	Platforms               []Platform        `json:"platforms"`
	EstimatedMinutes        *int              `json:"estimated_minutes,omitempty"`
	CertificateEnabled      bool              `json:"certificate_enabled"`
	CertificatePassingScore int               `json:"certificate_passing_score"`
	ReviewsEnabled          bool              `json:"reviews_enabled"`
	QuizPassPercent         int               `json:"quiz_pass_percent"`
	QuizMinutes             int               `json:"quiz_minutes"`
	MaxAttempts             int               `json:"max_attempts"`
	RetakeCooldownDays      int               `json:"retake_cooldown_days"`
	Questions               []QuestionPayload `json:"questions"`
}

type CourseListFilter struct {
	Search          string
	Status          *CourseStatus
	Category        *string
	Platform        *Platform
	IncludeArchived bool
	Limit           int
	Offset          int
}
