package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type certificateRepository interface {
	Create(ctx context.Context, params domain.CreateCertificateRecordParams) (domain.Certificate, error)
	GetByID(ctx context.Context, certificateID string) (domain.Certificate, error)
	GetByVerifyHash(ctx context.Context, verifyHash string) (domain.Certificate, error)
	List(ctx context.Context, filter domain.CertificateListFilter) ([]domain.Certificate, int, error)
	ExistsByEnrollmentAttempt(ctx context.Context, enrollmentID, attemptID string) (bool, error)
	ExistsByEnrollment(ctx context.Context, enrollmentID string) (bool, error)
	GetIssuanceContext(ctx context.Context, enrollmentID, attemptID string) (domain.CertificateIssuanceContext, error)
	FindAutoIssueCandidate(ctx context.Context, enrollmentID string) (domain.CertificateAutoIssueCandidate, error)
}

type CertificateUseCase struct {
	repository certificateRepository
	audit      *AuditLogger
}

func NewCertificateUseCase(repository certificateRepository) *CertificateUseCase {
	return &CertificateUseCase{repository: repository}
}

func (u *CertificateUseCase) WithAudit(audit *AuditLogger) *CertificateUseCase {
	u.audit = audit
	return u
}

func (u *CertificateUseCase) Create(ctx context.Context, params domain.CreateCertificateParams) (domain.Certificate, error) {
	normalized, err := normalizeCreateCertificateParams(params)
	if err != nil {
		return domain.Certificate{}, fmt.Errorf("usecase certificates create: %w", err)
	}

	exists, err := u.repository.ExistsByEnrollmentAttempt(ctx, normalized.EnrollmentID, normalized.AttemptID)
	if err != nil {
		return domain.Certificate{}, fmt.Errorf("usecase certificates create check duplicate: %w", err)
	}

	if exists {
		return domain.Certificate{}, fmt.Errorf("usecase certificates create duplicate: %w", domain.ErrConflict)
	}

	issuanceContext, err := u.repository.GetIssuanceContext(ctx, normalized.EnrollmentID, normalized.AttemptID)
	if err != nil {
		return domain.Certificate{}, fmt.Errorf("usecase certificates create load context: %w", err)
	}

	if issuanceContext.EnrollmentStatus != domain.EnrollmentStatusActive && issuanceContext.EnrollmentStatus != domain.EnrollmentStatusCompleted {
		return domain.Certificate{}, fmt.Errorf("enrollment is not active or completed: %w", domain.ErrValidation)
	}

	if !issuanceContext.CertificateEnabled {
		return domain.Certificate{}, fmt.Errorf("course certificate is disabled: %w", domain.ErrConflict)
	}

	if issuanceContext.CertificatePassingScore > 0 {
		if issuanceContext.AttemptScorePercent < float64(issuanceContext.CertificatePassingScore) {
			return domain.Certificate{}, fmt.Errorf("attempt score is below certificate passing score: %w", domain.ErrValidation)
		}
	} else if !issuanceContext.AttemptPassed {
		return domain.Certificate{}, fmt.Errorf("attempt is not passed: %w", domain.ErrValidation)
	}

	if issuanceContext.EnrollmentUserID != issuanceContext.AttemptUserID {
		return domain.Certificate{}, fmt.Errorf("attempt user mismatch: %w", domain.ErrConflict)
	}

	var lastErr error
	for range 5 {
		serialNumber, err := generateCertificateSerialNumber()
		if err != nil {
			return domain.Certificate{}, fmt.Errorf("usecase certificates create serial number: %w", err)
		}

		verifyHash, err := generateVerifyHash()
		if err != nil {
			return domain.Certificate{}, fmt.Errorf("usecase certificates create verify hash: %w", err)
		}

		certificate, err := u.repository.Create(ctx, domain.CreateCertificateRecordParams{
			EnrollmentID: normalized.EnrollmentID,
			UserID:       issuanceContext.EnrollmentUserID,
			CourseID:     issuanceContext.CourseID,
			AttemptID:    normalized.AttemptID,
			SerialNumber: serialNumber,
			VerifyHash:   verifyHash,
			PDFURL:       normalized.PDFURL,
		})
		if err == nil {
			if u.audit != nil {
				u.audit.Log(ctx, domain.AppEventCertificateIssued, map[string]any{
					"certificate_id": certificate.ID,
					"enrollment_id":  certificate.EnrollmentID,
					"attempt_id":     certificate.AttemptID,
					"user_id":        certificate.UserID,
					"course_id":      certificate.CourseID,
					"serial_number":  certificate.SerialNumber,
				})
			}
			return certificate, nil
		}

		lastErr = err
		if !errors.Is(err, domain.ErrConflict) {
			return domain.Certificate{}, fmt.Errorf("usecase certificates create insert: %w", err)
		}
	}

	return domain.Certificate{}, fmt.Errorf("usecase certificates create exhausted retries: %w", lastErr)
}

func (u *CertificateUseCase) GetByID(ctx context.Context, certificateID string) (domain.Certificate, error) {
	certificateID = strings.TrimSpace(certificateID)
	if certificateID == "" {
		return domain.Certificate{}, fmt.Errorf("usecase certificates get by id: %w", domain.ErrValidation)
	}

	certificate, err := u.repository.GetByID(ctx, certificateID)
	if err != nil {
		return domain.Certificate{}, fmt.Errorf("usecase certificates get by id: %w", err)
	}

	return certificate, nil
}

func (u *CertificateUseCase) GetByVerifyHash(ctx context.Context, verifyHash string) (domain.Certificate, error) {
	verifyHash = strings.TrimSpace(verifyHash)
	if verifyHash == "" {
		return domain.Certificate{}, fmt.Errorf("usecase certificates get by verify hash: %w", domain.ErrValidation)
	}

	certificate, err := u.repository.GetByVerifyHash(ctx, verifyHash)
	if err != nil {
		return domain.Certificate{}, fmt.Errorf("usecase certificates get by verify hash: %w", err)
	}

	return certificate, nil
}

func (u *CertificateUseCase) List(ctx context.Context, filter domain.CertificateListFilter) ([]domain.Certificate, int, error) {
	normalized, err := normalizeCertificateListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase certificates list: %w", err)
	}

	certificates, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase certificates list: %w", err)
	}

	return certificates, total, nil
}

func (u *CertificateUseCase) TryAutoIssueForEnrollment(ctx context.Context, enrollmentID string) (*domain.Certificate, error) {
	enrollmentID = strings.TrimSpace(enrollmentID)
	if enrollmentID == "" {
		return nil, fmt.Errorf("usecase certificates auto issue: %w", domain.ErrValidation)
	}

	exists, err := u.repository.ExistsByEnrollment(ctx, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("usecase certificates auto issue check existing: %w", err)
	}
	if exists {
		return nil, nil
	}

	candidate, err := u.repository.FindAutoIssueCandidate(ctx, enrollmentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("usecase certificates auto issue find candidate: %w", err)
	}

	if !candidate.CertificateEnabled {
		return nil, nil
	}

	if candidate.CertificatePassingScore > 0 && candidate.AttemptScorePercent < float64(candidate.CertificatePassingScore) {
		return nil, nil
	}

	certificate, err := u.Create(ctx, domain.CreateCertificateParams{
		EnrollmentID: enrollmentID,
		AttemptID:    candidate.AttemptID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, nil
		}

		return nil, fmt.Errorf("usecase certificates auto issue create: %w", err)
	}

	return &certificate, nil
}

func normalizeCreateCertificateParams(params domain.CreateCertificateParams) (domain.CreateCertificateParams, error) {
	params.EnrollmentID = strings.TrimSpace(params.EnrollmentID)
	params.AttemptID = strings.TrimSpace(params.AttemptID)
	params.PDFURL = normalizeOptionalString(params.PDFURL)

	if params.EnrollmentID == "" {
		return domain.CreateCertificateParams{}, fmt.Errorf("enrollment_id is required: %w", domain.ErrValidation)
	}

	if params.AttemptID == "" {
		return domain.CreateCertificateParams{}, fmt.Errorf("attempt_id is required: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeCertificateListFilter(filter domain.CertificateListFilter) (domain.CertificateListFilter, error) {
	if filter.UserID != nil {
		filter.UserID = normalizeOptionalString(filter.UserID)
	}

	if filter.CourseID != nil {
		filter.CourseID = normalizeOptionalString(filter.CourseID)
	}

	if filter.EnrollmentID != nil {
		filter.EnrollmentID = normalizeOptionalString(filter.EnrollmentID)
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Offset < 0 {
		return domain.CertificateListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}

func generateCertificateSerialNumber() (string, error) {
	buffer := make([]byte, 5)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("usecase certificates generate serial random: %w", err)
	}

	number := int(buffer[0])<<16 | int(buffer[1])<<8 | int(buffer[2])
	number = number % 1000000000

	return fmt.Sprintf("%03d-%03d-%03d", number/1000000, (number/1000)%1000, number%1000), nil
}

func generateVerifyHash() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("usecase certificates generate verify hash: %w", err)
	}

	return hex.EncodeToString(buffer), nil
}
