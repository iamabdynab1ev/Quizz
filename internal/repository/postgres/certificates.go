package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type certificateRowScanner interface {
	Scan(dest ...any) error
}

type CertificateRepository struct {
	pool *pgxpool.Pool
}

func NewCertificateRepository(pool *pgxpool.Pool) *CertificateRepository {
	return &CertificateRepository{pool: pool}
}

func (r *CertificateRepository) Create(ctx context.Context, params domain.CreateCertificateRecordParams) (domain.Certificate, error) {
	var certificateID string
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO certificates (
			enrollment_id,
			user_id,
			course_id,
			attempt_id,
			serial_number,
			verify_hash,
			pdf_url
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		RETURNING id
	`,
		params.EnrollmentID,
		params.UserID,
		params.CourseID,
		params.AttemptID,
		params.SerialNumber,
		params.VerifyHash,
		nullableStringPointerForWrite(params.PDFURL),
	).Scan(&certificateID); err != nil {
		return domain.Certificate{}, wrapPGError("repository postgres certificates create", err)
	}

	certificate, err := r.GetByID(ctx, certificateID)
	if err != nil {
		return domain.Certificate{}, fmt.Errorf("repository postgres certificates create fetch by id: %w", err)
	}

	return certificate, nil
}

func (r *CertificateRepository) GetByID(ctx context.Context, certificateID string) (domain.Certificate, error) {
	certificate, err := scanCertificateRow(r.pool.QueryRow(ctx, certificateSelectQuery(`
		WHERE cert.id = $1
	`), certificateID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Certificate{}, fmt.Errorf("repository postgres certificates get by id: %w", domain.ErrNotFound)
		}

		return domain.Certificate{}, fmt.Errorf("repository postgres certificates get by id: %w", err)
	}

	return certificate, nil
}

func (r *CertificateRepository) GetByVerifyHash(ctx context.Context, verifyHash string) (domain.Certificate, error) {
	certificate, err := scanCertificateRow(r.pool.QueryRow(ctx, certificateSelectQuery(`
		WHERE cert.verify_hash = $1
	`), verifyHash))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Certificate{}, fmt.Errorf("repository postgres certificates get by verify hash: %w", domain.ErrNotFound)
		}

		return domain.Certificate{}, fmt.Errorf("repository postgres certificates get by verify hash: %w", err)
	}

	return certificate, nil
}

func (r *CertificateRepository) List(ctx context.Context, filter domain.CertificateListFilter) ([]domain.Certificate, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(certificateSelectQuery(" WHERE 1 = 1 "))
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM certificates cert
				JOIN users u ON u.id = cert.user_id
				JOIN courses c ON c.id = cert.course_id
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 5)
		position := 1

		if filter.UserID != nil {
			query.WriteString(fmt.Sprintf(" AND cert.user_id = $%d", position))
			args = append(args, *filter.UserID)
			position++
		}

		if filter.CourseID != nil {
			query.WriteString(fmt.Sprintf(" AND cert.course_id = $%d", position))
			args = append(args, *filter.CourseID)
			position++
		}

		if filter.EnrollmentID != nil {
			query.WriteString(fmt.Sprintf(" AND cert.enrollment_id = $%d", position))
			args = append(args, *filter.EnrollmentID)
			position++
		}

		if includePagination {
			query.WriteString(fmt.Sprintf(" ORDER BY cert.issued_at DESC LIMIT $%d OFFSET $%d", position, position+1))
			args = append(args, filter.Limit, filter.Offset)
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres certificates list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres certificates list query: %w", err)
	}
	defer rows.Close()

	certificates := make([]domain.Certificate, 0, filter.Limit)
	for rows.Next() {
		certificate, err := scanCertificateRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres certificates list scan: %w", err)
		}

		certificates = append(certificates, certificate)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres certificates list rows: %w", err)
	}

	return certificates, total, nil
}

func (r *CertificateRepository) ExistsByEnrollmentAttempt(ctx context.Context, enrollmentID, attemptID string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM certificates
			WHERE enrollment_id = $1 AND attempt_id = $2
		)
	`, enrollmentID, attemptID).Scan(&exists); err != nil {
		return false, fmt.Errorf("repository postgres certificates exists by enrollment attempt: %w", err)
	}

	return exists, nil
}

func (r *CertificateRepository) ExistsByEnrollment(ctx context.Context, enrollmentID string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM certificates
			WHERE enrollment_id = $1
		)
	`, enrollmentID).Scan(&exists); err != nil {
		return false, fmt.Errorf("repository postgres certificates exists by enrollment: %w", err)
	}

	return exists, nil
}

func (r *CertificateRepository) GetIssuanceContext(ctx context.Context, enrollmentID, attemptID string) (domain.CertificateIssuanceContext, error) {
	var result domain.CertificateIssuanceContext
	var enrollmentStatus string
	var courseTitleBytes []byte

	if err := r.pool.QueryRow(ctx, `
		SELECT
			e.id,
			e.status,
			COALESCE(e.user_id::text, ''),
			e.course_id::text,
			c.title,
			c.certificate_enabled,
			c.certificate_passing_score,
			a.id::text,
			COALESCE(a.user_id::text, ''),
			a.passed,
			a.score_percent,
			u.first_name,
			u.last_name,
			COALESCE(u.patronymic, '')
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		JOIN attempts a ON a.id = $2
		JOIN users u ON u.id = e.user_id
		WHERE e.id = $1
			AND EXISTS (
				SELECT 1
				FROM course_tests ct
				LEFT JOIN course_modules cm ON cm.id = ct.module_id
				WHERE ct.quiz_id = a.quiz_id
					AND (ct.course_id = e.course_id OR cm.course_id = e.course_id)
			)
	`,
		enrollmentID,
		attemptID,
	).Scan(
		&result.EnrollmentID,
		&enrollmentStatus,
		&result.EnrollmentUserID,
		&result.CourseID,
		&courseTitleBytes,
		&result.CertificateEnabled,
		&result.CertificatePassingScore,
		&result.AttemptID,
		&result.AttemptUserID,
		&result.AttemptPassed,
		&result.AttemptScorePercent,
		&result.UserFirstName,
		&result.UserLastName,
		&result.Patronymic,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CertificateIssuanceContext{}, fmt.Errorf("repository postgres certificates get issuance context: %w", domain.ErrNotFound)
		}

		return domain.CertificateIssuanceContext{}, fmt.Errorf("repository postgres certificates get issuance context: %w", err)
	}

	if err := result.CourseTitle.Scan(courseTitleBytes); err != nil {
		return domain.CertificateIssuanceContext{}, fmt.Errorf("repository postgres certificates scan course title: %w", err)
	}

	result.EnrollmentStatus = domain.EnrollmentStatus(enrollmentStatus)
	return result, nil
}

func (r *CertificateRepository) FindAutoIssueCandidate(ctx context.Context, enrollmentID string) (domain.CertificateAutoIssueCandidate, error) {
	var candidate domain.CertificateAutoIssueCandidate

	if err := r.pool.QueryRow(ctx, `
		SELECT
			e.id::text,
			e.course_id::text,
			c.certificate_enabled,
			c.certificate_passing_score,
			a.id::text,
			a.score_percent
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		JOIN attempts a ON a.user_id = e.user_id AND a.passed = true
		WHERE e.id = $1
			AND EXISTS (
				SELECT 1
				FROM course_tests ct
				LEFT JOIN course_modules cm ON cm.id = ct.module_id
				WHERE ct.quiz_id = a.quiz_id
					AND (ct.course_id = e.course_id OR cm.course_id = e.course_id)
			)
		ORDER BY a.score_percent DESC, a.finished_at DESC NULLS LAST, a.started_at DESC
		LIMIT 1
	`, enrollmentID).Scan(
		&candidate.EnrollmentID,
		&candidate.CourseID,
		&candidate.CertificateEnabled,
		&candidate.CertificatePassingScore,
		&candidate.AttemptID,
		&candidate.AttemptScorePercent,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CertificateAutoIssueCandidate{}, fmt.Errorf("repository postgres certificates find auto issue candidate: %w", domain.ErrNotFound)
		}

		return domain.CertificateAutoIssueCandidate{}, fmt.Errorf("repository postgres certificates find auto issue candidate: %w", err)
	}

	return candidate, nil
}

func certificateSelectQuery(whereClause string) string {
	return `
		SELECT
			cert.id,
			cert.enrollment_id,
			cert.user_id,
			cert.course_id,
			cert.attempt_id,
			cert.serial_number,
			cert.verify_hash,
			cert.issued_at,
			cert.pdf_url,
			COALESCE(u.first_name, ''),
			COALESCE(u.last_name, ''),
			COALESCE(u.patronymic, ''),
			c.title
		FROM certificates cert
		JOIN users u ON u.id = cert.user_id
		JOIN courses c ON c.id = cert.course_id
	` + whereClause
}

func scanCertificateRow(scanner certificateRowScanner) (domain.Certificate, error) {
	var certificate domain.Certificate
	var pdfURL sql.NullString
	var courseTitleBytes []byte

	if err := scanner.Scan(
		&certificate.ID,
		&certificate.EnrollmentID,
		&certificate.UserID,
		&certificate.CourseID,
		&certificate.AttemptID,
		&certificate.SerialNumber,
		&certificate.VerifyHash,
		&certificate.IssuedAt,
		&pdfURL,
		&certificate.UserFirstName,
		&certificate.UserLastName,
		&certificate.Patronymic,
		&courseTitleBytes,
	); err != nil {
		return domain.Certificate{}, err
	}

	if err := certificate.CourseTitle.Scan(courseTitleBytes); err != nil {
		return domain.Certificate{}, fmt.Errorf("repository postgres scan certificate course title: %w", err)
	}

	certificate.PDFURL = optionalString(pdfURL)

	return certificate, nil
}
