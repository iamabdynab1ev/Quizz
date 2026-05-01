package domain

import "time"

type Certificate struct {
	ID            string        `json:"id"`
	EnrollmentID  string        `json:"enrollment_id"`
	UserID        string        `json:"user_id"`
	CourseID      string        `json:"course_id"`
	AttemptID     string        `json:"attempt_id"`
	SerialNumber  string        `json:"serial_number"`
	VerifyHash    string        `json:"verify_hash"`
	IssuedAt      time.Time     `json:"issued_at"`
	PDFURL        *string       `json:"pdf_url,omitempty"`
	UserFirstName string        `json:"user_first_name,omitempty"`
	UserLastName  string        `json:"user_last_name,omitempty"`
	Patronymic    string        `json:"patronymic,omitempty"`
	CourseTitle   MultiLangText `json:"course_title"`
}

type CreateCertificateParams struct {
	EnrollmentID string  `json:"enrollment_id"`
	AttemptID    string  `json:"attempt_id"`
	PDFURL       *string `json:"pdf_url,omitempty"`
}

type CreateCertificateRecordParams struct {
	EnrollmentID string
	UserID       string
	CourseID     string
	AttemptID    string
	SerialNumber string
	VerifyHash   string
	PDFURL       *string
}

type CertificateListFilter struct {
	UserID       *string
	CourseID     *string
	EnrollmentID *string
	Limit        int
	Offset       int
}

type CertificateIssuanceContext struct {
	EnrollmentID            string
	EnrollmentStatus        EnrollmentStatus
	EnrollmentUserID        string
	CourseID                string
	CourseTitle             MultiLangText
	CertificateEnabled      bool
	CertificatePassingScore int
	AttemptID               string
	AttemptUserID           string
	AttemptPassed           bool
	AttemptScorePercent     float64
	UserFirstName           string
	UserLastName            string
	Patronymic              string
}

type CertificateAutoIssueCandidate struct {
	EnrollmentID            string
	CourseID                string
	CertificateEnabled      bool
	CertificatePassingScore int
	AttemptID               string
	AttemptScorePercent     float64
}
