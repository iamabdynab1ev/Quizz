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

type courseRowScanner interface {
	Scan(dest ...any) error
}

type CourseRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) *CourseRepository {
	return &CourseRepository{pool: pool}
}

func (r *CourseRepository) Create(ctx context.Context, params domain.CreateCourseParams) (domain.Course, error) {
	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create description value: %w", err)
	}

	var courseID string
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO courses (
			title,
			description,
			cover_image_url,
			category,
			status,
			platforms,
			estimated_minutes,
			certificate_enabled,
			certificate_passing_score,
			reviews_enabled
		) VALUES (
			$1::jsonb,
			$2::jsonb,
			$3,
			$4,
			$5,
			$6::platform[],
			$7,
			$8,
			$9,
			$10
		)
		RETURNING id
	`,
		titleValue,
		descriptionValue,
		nullableStringPointerForWrite(params.CoverImageURL),
		nullableStringPointerForWrite(params.Category),
		string(params.Status),
		platformsToStrings(params.Platforms),
		nullableIntPointerForWrite(params.EstimatedMinutes),
		params.CertificateEnabled,
		params.CertificatePassingScore,
		params.ReviewsEnabled,
	).Scan(&courseID); err != nil {
		return domain.Course{}, wrapPGError("repository postgres courses create", err)
	}

	course, err := r.GetByID(ctx, courseID)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create fetch by id: %w", err)
	}

	return course, nil
}

func (r *CourseRepository) GetByID(ctx context.Context, courseID string) (domain.Course, error) {
	course, err := scanCourse(r.pool.QueryRow(ctx, `
		SELECT
			id,
			title,
			description,
			cover_image_url,
			category,
			status,
			platforms,
			estimated_minutes,
			certificate_enabled,
			certificate_passing_score,
			reviews_enabled,
			created_at,
			updated_at
		FROM courses
		WHERE id = $1
	`, courseID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Course{}, fmt.Errorf("repository postgres courses get by id: %w", domain.ErrNotFound)
		}

		return domain.Course{}, fmt.Errorf("repository postgres courses get by id: %w", err)
	}

	return course, nil
}

func (r *CourseRepository) List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT
					id,
					title,
					description,
					cover_image_url,
					category,
					status,
					platforms,
					estimated_minutes,
					certificate_enabled,
					certificate_passing_score,
					reviews_enabled,
					created_at,
					updated_at
				FROM courses
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM courses
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 6)
		position := 1

		if filter.Search != "" {
			query.WriteString(fmt.Sprintf(`
				AND (
					title->>'ru' ILIKE $%d OR
					title->>'tj' ILIKE $%d OR
					COALESCE(category, '') ILIKE $%d
				)
			`, position, position, position))
			args = append(args, "%"+filter.Search+"%")
			position++
		}

		if filter.Status != nil {
			query.WriteString(fmt.Sprintf(" AND status = $%d", position))
			args = append(args, string(*filter.Status))
			position++
		}

		if filter.Category != nil {
			query.WriteString(fmt.Sprintf(" AND category = $%d", position))
			args = append(args, *filter.Category)
			position++
		}

		if filter.Platform != nil {
			query.WriteString(fmt.Sprintf(" AND $%d = ANY(platforms)", position))
			args = append(args, string(*filter.Platform))
			position++
		}

		if includePagination {
			query.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", position, position+1))
			args = append(args, filter.Limit, filter.Offset)
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres courses list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres courses list query: %w", err)
	}
	defer rows.Close()

	courses := make([]domain.Course, 0, filter.Limit)
	for rows.Next() {
		course, err := scanCourse(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres courses list scan: %w", err)
		}

		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres courses list rows: %w", err)
	}

	return courses, total, nil
}

func (r *CourseRepository) Update(ctx context.Context, params domain.UpdateCourseParams) (domain.Course, error) {
	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update description value: %w", err)
	}

	var courseID string
	if err := r.pool.QueryRow(ctx, `
		UPDATE courses
		SET
			title = $2::jsonb,
			description = $3::jsonb,
			cover_image_url = $4,
			category = $5,
			status = $6,
			platforms = $7::platform[],
			estimated_minutes = $8,
			certificate_enabled = $9,
			certificate_passing_score = $10,
			reviews_enabled = $11,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		titleValue,
		descriptionValue,
		nullableStringPointerForWrite(params.CoverImageURL),
		nullableStringPointerForWrite(params.Category),
		string(params.Status),
		platformsToStrings(params.Platforms),
		nullableIntPointerForWrite(params.EstimatedMinutes),
		params.CertificateEnabled,
		params.CertificatePassingScore,
		params.ReviewsEnabled,
	).Scan(&courseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Course{}, fmt.Errorf("repository postgres courses update: %w", domain.ErrNotFound)
		}

		return domain.Course{}, wrapPGError("repository postgres courses update", err)
	}

	course, err := r.GetByID(ctx, courseID)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update fetch by id: %w", err)
	}

	return course, nil
}

func (r *CourseRepository) Archive(ctx context.Context, courseID string) error {
	var returnedID string
	if err := r.pool.QueryRow(ctx, `
		UPDATE courses
		SET
			status = 'archived',
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, courseID).Scan(&returnedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("repository postgres courses archive: %w", domain.ErrNotFound)
		}

		return fmt.Errorf("repository postgres courses archive: %w", err)
	}

	return nil
}

func scanCourse(scanner courseRowScanner) (domain.Course, error) {
	var course domain.Course
	var titleBytes []byte
	var descriptionBytes []byte
	var coverImageURL sql.NullString
	var category sql.NullString
	var status string
	var platforms []string
	var estimatedMinutes sql.NullInt32

	if err := scanner.Scan(
		&course.ID,
		&titleBytes,
		&descriptionBytes,
		&coverImageURL,
		&category,
		&status,
		&platforms,
		&estimatedMinutes,
		&course.CertificateEnabled,
		&course.CertificatePassingScore,
		&course.ReviewsEnabled,
		&course.CreatedAt,
		&course.UpdatedAt,
	); err != nil {
		return domain.Course{}, err
	}

	if err := course.Title.Scan(titleBytes); err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres scan course title: %w", err)
	}

	if len(descriptionBytes) > 0 {
		if err := course.Description.Scan(descriptionBytes); err != nil {
			return domain.Course{}, fmt.Errorf("repository postgres scan course description: %w", err)
		}
	}

	course.CoverImageURL = optionalString(coverImageURL)
	course.Category = optionalString(category)
	course.Status = domain.CourseStatus(status)
	course.Platforms = stringsToPlatforms(platforms)
	course.EstimatedMinutes = optionalInt(estimatedMinutes)

	return course, nil
}
