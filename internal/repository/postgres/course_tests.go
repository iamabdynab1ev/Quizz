package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseTestRepository struct {
	pool *pgxpool.Pool
}

func NewCourseTestRepository(pool *pgxpool.Pool) *CourseTestRepository {
	return &CourseTestRepository{pool: pool}
}

func (r *CourseTestRepository) Create(ctx context.Context, params domain.CreateCourseTestParams) (domain.CourseTest, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO course_tests (
			course_id,
			module_id,
			quiz_id,
			position
		) VALUES (
			$1::uuid, $2::uuid, $3, $4
		)
		RETURNING id, course_id::text, module_id::text, quiz_id::text, position
	`,
		nullableStringPointerForWrite(params.CourseID),
		nullableStringPointerForWrite(params.ModuleID),
		params.QuizID,
		params.Position,
	)

	var courseTest domain.CourseTest
	var courseID sql.NullString
	var moduleID sql.NullString
	if err := row.Scan(&courseTest.ID, &courseID, &moduleID, &courseTest.QuizID, &courseTest.Position); err != nil {
		return domain.CourseTest{}, wrapPGError("repository postgres course tests create", err)
	}

	courseTest.CourseID = optionalString(courseID)
	courseTest.ModuleID = optionalString(moduleID)
	return courseTest, nil
}

func (r *CourseTestRepository) List(ctx context.Context, filter domain.CourseTestListFilter) ([]domain.CourseTest, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
			SELECT id, course_id::text, module_id::text, quiz_id::text, position
				FROM course_tests
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM course_tests
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 2)
		position := 1

		if filter.CourseID != nil {
			query.WriteString(fmt.Sprintf(" AND course_id = $%d::uuid", position))
			args = append(args, *filter.CourseID)
			position++
		}

		if filter.ModuleID != nil {
			query.WriteString(fmt.Sprintf(" AND module_id = $%d::uuid", position))
			args = append(args, *filter.ModuleID)
			position++
		}

		if includePagination {
			query.WriteString(" ORDER BY position ASC")
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres course tests list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres course tests list query: %w", err)
	}
	defer rows.Close()

	courseTests := make([]domain.CourseTest, 0)
	for rows.Next() {
		var courseTest domain.CourseTest
		var courseID sql.NullString
		var moduleID sql.NullString
		if err := rows.Scan(&courseTest.ID, &courseID, &moduleID, &courseTest.QuizID, &courseTest.Position); err != nil {
			return nil, 0, fmt.Errorf("repository postgres course tests list scan: %w", err)
		}

		courseTest.CourseID = optionalString(courseID)
		courseTest.ModuleID = optionalString(moduleID)
		courseTests = append(courseTests, courseTest)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres course tests list rows: %w", err)
	}

	return courseTests, total, nil
}

func (r *CourseTestRepository) Delete(ctx context.Context, courseID, moduleID, quizID string) error {
	query := `
		DELETE FROM course_tests
		WHERE quiz_id = $1 AND course_id = $2::uuid
	`
	args := []any{quizID, courseID}

	if moduleID != "" {
		query = `
			DELETE FROM course_tests
			WHERE quiz_id = $1 AND module_id = $2::uuid
		`
		args = []any{quizID, moduleID}
	}

	commandTag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("repository postgres course tests delete: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres course tests delete: %w", domain.ErrNotFound)
	}

	return nil
}

func (r *CourseTestRepository) DeleteByID(ctx context.Context, courseTestID string) error {
	commandTag, err := r.pool.Exec(ctx, `
		DELETE FROM course_tests
		WHERE id = $1
	`, courseTestID)
	if err != nil {
		return fmt.Errorf("repository postgres course tests delete by id: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres course tests delete by id: %w", domain.ErrNotFound)
	}

	return nil
}
