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

type enrollmentRowScanner interface {
	Scan(dest ...any) error
}

type EnrollmentRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) *EnrollmentRepository {
	return &EnrollmentRepository{pool: pool}
}

func (r *EnrollmentRepository) Create(ctx context.Context, params domain.CreateEnrollmentParams) (domain.Enrollment, error) {
	enrollment, err := scanEnrollmentRow(r.pool.QueryRow(ctx, `
		INSERT INTO enrollments (
			course_id,
			user_id,
			status
		) VALUES (
			$1, $2, 'active'
		)
		RETURNING id, course_id, user_id, status, enrolled_at, completed_at
	`,
		params.CourseID,
		params.UserID,
	))
	if err != nil {
		return domain.Enrollment{}, wrapPGError("repository postgres enrollments create", err)
	}

	return enrollment, nil
}

func (r *EnrollmentRepository) GetByID(ctx context.Context, enrollmentID string) (domain.Enrollment, error) {
	enrollment, err := scanEnrollmentRow(r.pool.QueryRow(ctx, `
		SELECT id, course_id, user_id, status, enrolled_at, completed_at
		FROM enrollments
		WHERE id = $1
	`, enrollmentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Enrollment{}, fmt.Errorf("repository postgres enrollments get by id: %w", domain.ErrNotFound)
		}

		return domain.Enrollment{}, fmt.Errorf("repository postgres enrollments get by id: %w", err)
	}

	return enrollment, nil
}

func (r *EnrollmentRepository) List(ctx context.Context, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT id, course_id, user_id, status, enrolled_at, completed_at
				FROM enrollments
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM enrollments
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 5)
		position := 1

		if filter.CourseID != nil {
			query.WriteString(fmt.Sprintf(" AND course_id = $%d", position))
			args = append(args, *filter.CourseID)
			position++
		}

		if filter.UserID != nil {
			query.WriteString(fmt.Sprintf(" AND user_id = $%d", position))
			args = append(args, *filter.UserID)
			position++
		}

		if filter.Status != nil {
			query.WriteString(fmt.Sprintf(" AND status = $%d", position))
			args = append(args, string(*filter.Status))
			position++
		}

		if includePagination {
			query.WriteString(fmt.Sprintf(" ORDER BY enrolled_at DESC LIMIT $%d OFFSET $%d", position, position+1))
			args = append(args, filter.Limit, filter.Offset)
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres enrollments list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres enrollments list query: %w", err)
	}
	defer rows.Close()

	enrollments := make([]domain.Enrollment, 0, filter.Limit)
	for rows.Next() {
		enrollment, err := scanEnrollmentRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres enrollments list scan: %w", err)
		}

		enrollments = append(enrollments, enrollment)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres enrollments list rows: %w", err)
	}

	return enrollments, total, nil
}

func (r *EnrollmentRepository) Complete(ctx context.Context, enrollmentID string) (domain.Enrollment, error) {
	enrollment, err := scanEnrollmentRow(r.pool.QueryRow(ctx, `
		UPDATE enrollments
		SET
			status = 'completed',
			completed_at = COALESCE(completed_at, NOW())
		WHERE id = $1
			AND status <> 'dropped'
		RETURNING id, course_id, user_id, status, enrolled_at, completed_at
	`, enrollmentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Enrollment{}, fmt.Errorf("repository postgres enrollments complete: %w", domain.ErrNotFound)
		}

		return domain.Enrollment{}, fmt.Errorf("repository postgres enrollments complete: %w", err)
	}

	return enrollment, nil
}

func (r *EnrollmentRepository) HasEnrollment(ctx context.Context, courseID, userID string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments
			WHERE course_id = $1
				AND user_id = $2
				AND status IN ('active', 'completed')
		)
	`, courseID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("repository postgres enrollments has enrollment: %w", err)
	}

	return exists, nil
}

func scanEnrollmentRow(scanner enrollmentRowScanner) (domain.Enrollment, error) {
	var enrollment domain.Enrollment
	var userID sql.NullString
	var status string
	var completedAt sql.NullTime

	if err := scanner.Scan(
		&enrollment.ID,
		&enrollment.CourseID,
		&userID,
		&status,
		&enrollment.EnrolledAt,
		&completedAt,
	); err != nil {
		return domain.Enrollment{}, err
	}

	enrollment.UserID = optionalString(userID)
	enrollment.Status = domain.EnrollmentStatus(status)
	enrollment.CompletedAt = optionalTime(completedAt)

	return enrollment, nil
}
