package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type enrollmentRepository interface {
	Create(ctx context.Context, params domain.CreateEnrollmentParams) (domain.Enrollment, error)
	GetByID(ctx context.Context, enrollmentID string) (domain.Enrollment, error)
	List(ctx context.Context, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int, error)
	Complete(ctx context.Context, enrollmentID string) (domain.Enrollment, error)
	HasEnrollment(ctx context.Context, courseID, userID string) (bool, error)
}

type enrollmentCertificateAutoIssuer interface {
	TryAutoIssueForEnrollment(ctx context.Context, enrollmentID string) (*domain.Certificate, error)
}

type EnrollmentUseCase struct {
	repository enrollmentRepository
	audit      *AuditLogger
	autoIssuer enrollmentCertificateAutoIssuer
}

func NewEnrollmentUseCase(repository enrollmentRepository) *EnrollmentUseCase {
	return &EnrollmentUseCase{repository: repository}
}

func (u *EnrollmentUseCase) WithAudit(audit *AuditLogger) *EnrollmentUseCase {
	u.audit = audit
	return u
}

func (u *EnrollmentUseCase) WithCertificateAutoIssuer(autoIssuer enrollmentCertificateAutoIssuer) *EnrollmentUseCase {
	u.autoIssuer = autoIssuer
	return u
}

func (u *EnrollmentUseCase) Create(ctx context.Context, params domain.CreateEnrollmentParams) (domain.Enrollment, error) {
	normalized, err := normalizeCreateEnrollmentParams(params)
	if err != nil {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments create: %w", err)
	}

	exists, err := u.repository.HasEnrollment(ctx, normalized.CourseID, normalized.UserID)
	if err != nil {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments create check duplicate: %w", err)
	}

	if exists {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments create duplicate: %w", domain.ErrConflict)
	}

	enrollment, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments create: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventEnrollmentCreated, map[string]any{
			"enrollment_id": enrollment.ID,
			"course_id":     enrollment.CourseID,
			"user_id":       enrollment.UserID,
			"status":        enrollment.Status,
		})
	}

	return enrollment, nil
}

func (u *EnrollmentUseCase) GetByID(ctx context.Context, enrollmentID string) (domain.Enrollment, error) {
	enrollmentID = strings.TrimSpace(enrollmentID)
	if enrollmentID == "" {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments get by id: %w", domain.ErrValidation)
	}

	enrollment, err := u.repository.GetByID(ctx, enrollmentID)
	if err != nil {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments get by id: %w", err)
	}

	return enrollment, nil
}

func (u *EnrollmentUseCase) List(ctx context.Context, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int, error) {
	normalized, err := normalizeEnrollmentListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase enrollments list: %w", err)
	}

	enrollments, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase enrollments list: %w", err)
	}

	return enrollments, total, nil
}

func (u *EnrollmentUseCase) Complete(ctx context.Context, enrollmentID string) (domain.Enrollment, error) {
	enrollmentID = strings.TrimSpace(enrollmentID)
	if enrollmentID == "" {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments complete: %w", domain.ErrValidation)
	}

	enrollment, err := u.repository.Complete(ctx, enrollmentID)
	if err != nil {
		return domain.Enrollment{}, fmt.Errorf("usecase enrollments complete: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventEnrollmentComplete, map[string]any{
			"enrollment_id": enrollment.ID,
			"course_id":     enrollment.CourseID,
			"user_id":       enrollment.UserID,
			"status":        enrollment.Status,
			"completed_at":  enrollment.CompletedAt,
		})
	}

	if u.autoIssuer != nil {
		certificate, err := u.autoIssuer.TryAutoIssueForEnrollment(ctx, enrollment.ID)
		if err != nil {
			slog.ErrorContext(ctx, "auto certificate issuance failed",
				slog.String("enrollment_id", enrollment.ID),
				slog.String("error", err.Error()),
			)
		} else if certificate != nil {
			slog.InfoContext(ctx, "auto certificate issued",
				slog.String("enrollment_id", enrollment.ID),
				slog.String("certificate_id", certificate.ID),
			)
		}
	}

	return enrollment, nil
}

func normalizeCreateEnrollmentParams(params domain.CreateEnrollmentParams) (domain.CreateEnrollmentParams, error) {
	params.CourseID = strings.TrimSpace(params.CourseID)
	params.UserID = strings.TrimSpace(params.UserID)

	if params.CourseID == "" {
		return domain.CreateEnrollmentParams{}, fmt.Errorf("course_id is required: %w", domain.ErrValidation)
	}

	if params.UserID == "" {
		return domain.CreateEnrollmentParams{}, fmt.Errorf("user_id is required: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeEnrollmentListFilter(filter domain.EnrollmentListFilter) (domain.EnrollmentListFilter, error) {
	if filter.CourseID != nil {
		filter.CourseID = normalizeOptionalString(filter.CourseID)
	}

	if filter.UserID != nil {
		filter.UserID = normalizeOptionalString(filter.UserID)
	}

	if filter.Status != nil {
		status := domain.EnrollmentStatus(strings.TrimSpace(string(*filter.Status)))
		if !status.IsValid() {
			return domain.EnrollmentListFilter{}, fmt.Errorf("status filter is invalid: %w", domain.ErrValidation)
		}
		filter.Status = &status
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Offset < 0 {
		return domain.EnrollmentListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}
