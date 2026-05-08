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
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create description value: %w", err)
	}

	var courseID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO courses (
			title,
			description,
			cover_image_url,
			video_url,
			category,
			status,
			platforms,
			estimated_minutes,
			certificate_enabled,
			certificate_passing_score,
			reviews_enabled,
			quiz_pass_percent,
			quiz_minutes,
			max_attempts,
			retake_cooldown_days,
			created_by_user_id,
			created_by_name
		) VALUES (
			$1::jsonb, $2::jsonb, $3, $4, $5, $6, $7::platform[],
			$8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		RETURNING id
	`,
		titleValue,
		descriptionValue,
		nullableStringPointerForWrite(params.CoverImageURL),
		nullableStringPointerForWrite(params.VideoURL),
		nullableStringPointerForWrite(params.Category),
		string(params.Status),
		platformsToStrings(params.Platforms),
		nullableIntPointerForWrite(params.EstimatedMinutes),
		boolPointerForWrite(params.CertificateEnabled),
		params.CertificatePassingScore,
		boolPointerForWrite(params.ReviewsEnabled),
		params.QuizPassPercent,
		params.QuizMinutes,
		params.MaxAttempts,
		params.RetakeCooldownDays,
		params.CreatedByUserID,
		params.CreatedByName,
	).Scan(&courseID); err != nil {
		return domain.Course{}, wrapPGError("repository postgres courses create", err)
	}

	if err := replaceQuestions(ctx, tx, courseID, params.Questions); err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create questions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses create commit: %w", err)
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
			id, title, description, cover_image_url, video_url, category,
			status, platforms, estimated_minutes, certificate_enabled,
			certificate_passing_score, reviews_enabled,
			quiz_pass_percent, quiz_minutes, max_attempts, retake_cooldown_days,
			created_by_user_id, created_by_name,
			created_at, updated_at
		FROM courses
		WHERE id = $1
	`, courseID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Course{}, fmt.Errorf("repository postgres courses get by id: %w", domain.ErrNotFound)
		}
		return domain.Course{}, fmt.Errorf("repository postgres courses get by id: %w", err)
	}

	questions, err := listQuestionsByCourseID(ctx, r.pool, courseID)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses get by id questions: %w", err)
	}
	course.Questions = questions

	return course, nil
}

func (r *CourseRepository) List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT
					id, title, description, cover_image_url, video_url, category,
					status, platforms, estimated_minutes, certificate_enabled,
					certificate_passing_score, reviews_enabled,
					quiz_pass_percent, quiz_minutes, max_attempts, retake_cooldown_days,
					created_by_user_id, created_by_name,
					created_at, updated_at
				FROM courses
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`SELECT COUNT(*) FROM courses WHERE 1 = 1`)
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
		} else if !filter.IncludeArchived {
			query.WriteString(" AND status <> 'archived'")
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
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update description value: %w", err)
	}

	var courseID string
	if err := tx.QueryRow(ctx, `
		UPDATE courses
		SET
			title = $2::jsonb,
			description = $3::jsonb,
			cover_image_url = $4,
			video_url = $5,
			category = $6,
			status = $7,
			platforms = $8::platform[],
			estimated_minutes = $9,
			certificate_enabled = $10,
			certificate_passing_score = $11,
			reviews_enabled = $12,
			quiz_pass_percent = $13,
			quiz_minutes = $14,
			max_attempts = $15,
			retake_cooldown_days = $16,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		titleValue,
		descriptionValue,
		nullableStringPointerForWrite(params.CoverImageURL),
		nullableStringPointerForWrite(params.VideoURL),
		nullableStringPointerForWrite(params.Category),
		string(params.Status),
		platformsToStrings(params.Platforms),
		nullableIntPointerForWrite(params.EstimatedMinutes),
		params.CertificateEnabled,
		params.CertificatePassingScore,
		params.ReviewsEnabled,
		params.QuizPassPercent,
		params.QuizMinutes,
		params.MaxAttempts,
		params.RetakeCooldownDays,
	).Scan(&courseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Course{}, fmt.Errorf("repository postgres courses update: %w", domain.ErrNotFound)
		}
		return domain.Course{}, wrapPGError("repository postgres courses update", err)
	}

	if err := replaceQuestions(ctx, tx, courseID, params.Questions); err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update questions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres courses update commit: %w", err)
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
		SET status = 'archived', updated_at = NOW()
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

func replaceQuestions(ctx context.Context, tx pgx.Tx, courseID string, questions []domain.QuestionPayload) error {
	if _, err := tx.Exec(ctx, `DELETE FROM questions WHERE course_id = $1`, courseID); err != nil {
		return fmt.Errorf("repository postgres courses replace questions delete: %w", err)
	}

	for _, question := range questions {
		promptValue, err := toJSONValue(question.Prompt)
		if err != nil {
			return fmt.Errorf("repository postgres courses question prompt value: %w", err)
		}

		configValue := question.Config
		if len(configValue) == 0 {
			configValue = []byte("{}")
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO questions (course_id, position, type, prompt, points, required, config)
			VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7::jsonb)
		`,
			courseID,
			question.Position,
			string(question.Type),
			promptValue,
			question.Points,
			question.Required,
			[]byte(configValue),
		); err != nil {
			return wrapPGError("repository postgres courses insert question", err)
		}
	}

	return nil
}

func listQuestionsByCourseID(ctx context.Context, pool *pgxpool.Pool, courseID string) ([]domain.Question, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, position, type, prompt, points, required, config, created_at
		FROM questions
		WHERE course_id = $1
		ORDER BY position ASC
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("repository postgres courses list questions query: %w", err)
	}
	defer rows.Close()

	questions := make([]domain.Question, 0)
	for rows.Next() {
		question, err := scanQuestionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("repository postgres courses list questions scan: %w", err)
		}
		questions = append(questions, question)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository postgres courses list questions rows: %w", err)
	}

	return questions, nil
}

func scanCourse(scanner courseRowScanner) (domain.Course, error) {
	var course domain.Course
	var titleBytes []byte
	var descriptionBytes []byte
	var coverImageURL sql.NullString
	var videoURL sql.NullString
	var category sql.NullString
	var status string
	var platforms []string
	var estimatedMinutes sql.NullInt32
	var createdByUserID sql.NullString
	var createdByName sql.NullString

	if err := scanner.Scan(
		&course.ID,
		&titleBytes,
		&descriptionBytes,
		&coverImageURL,
		&videoURL,
		&category,
		&status,
		&platforms,
		&estimatedMinutes,
		&course.CertificateEnabled,
		&course.CertificatePassingScore,
		&course.ReviewsEnabled,
		&course.QuizPassPercent,
		&course.QuizMinutes,
		&course.MaxAttempts,
		&course.RetakeCooldownDays,
		&createdByUserID,
		&createdByName,
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
	course.VideoURL = optionalString(videoURL)
	course.Category = optionalString(category)
	course.Status = domain.CourseStatus(status)
	course.Platforms = stringsToPlatforms(platforms)
	course.EstimatedMinutes = optionalInt(estimatedMinutes)
	if createdByUserID.Valid {
		course.CreatedByUserID = &createdByUserID.String
	}
	if createdByName.Valid {
		course.CreatedByName = createdByName.String
	}

	return course, nil
}

func scanQuestionRow(scanner courseRowScanner) (domain.Question, error) {
	var question domain.Question
	var promptBytes []byte
	var questionType string

	if err := scanner.Scan(
		&question.ID,
		&question.Position,
		&questionType,
		&promptBytes,
		&question.Points,
		&question.Required,
		&question.Config,
		&question.CreatedAt,
	); err != nil {
		return domain.Question{}, err
	}

	if err := question.Prompt.Scan(promptBytes); err != nil {
		return domain.Question{}, fmt.Errorf("repository postgres scan question prompt: %w", err)
	}

	question.Type = domain.QuestionType(questionType)

	return question, nil
}
