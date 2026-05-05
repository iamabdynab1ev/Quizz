package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CoursePackageRepository struct {
	pool *pgxpool.Pool
}

func NewCoursePackageRepository(pool *pgxpool.Pool) *CoursePackageRepository {
	return &CoursePackageRepository{pool: pool}
}

func (r *CoursePackageRepository) Create(ctx context.Context, params domain.CreateCoursePackageParams) (domain.CoursePackage, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.CoursePackage{}, fmt.Errorf("repository postgres course packages create begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	courseID, err := r.insertCourse(ctx, tx, params.Course)
	if err != nil {
		return domain.CoursePackage{}, err
	}

	quizID, err := r.insertQuiz(ctx, tx, params.Quiz)
	if err != nil {
		return domain.CoursePackage{}, err
	}

	quizRepository := &QuizRepository{pool: r.pool}
	if err := quizRepository.replaceQuestions(ctx, tx, quizID, params.Quiz.Questions); err != nil {
		return domain.CoursePackage{}, fmt.Errorf("repository postgres course packages create questions: %w", err)
	}

	courseTest, err := r.insertCourseTest(ctx, tx, courseID, quizID, params.LinkPosition)
	if err != nil {
		return domain.CoursePackage{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.CoursePackage{}, fmt.Errorf("repository postgres course packages create commit: %w", err)
	}

	course, err := NewCourseRepository(r.pool).GetByID(ctx, courseID)
	if err != nil {
		return domain.CoursePackage{}, fmt.Errorf("repository postgres course packages fetch course: %w", err)
	}

	quiz, err := NewQuizRepository(r.pool).GetByID(ctx, quizID)
	if err != nil {
		return domain.CoursePackage{}, fmt.Errorf("repository postgres course packages fetch quiz: %w", err)
	}

	return domain.CoursePackage{
		Course:     course,
		Quiz:       quiz,
		CourseTest: courseTest,
	}, nil
}

func (r *CoursePackageRepository) insertCourse(ctx context.Context, tx pgx.Tx, params domain.CreateCourseParams) (string, error) {
	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return "", fmt.Errorf("repository postgres course packages course title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return "", fmt.Errorf("repository postgres course packages course description value: %w", err)
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
			reviews_enabled
		) VALUES (
			$1::jsonb,
			$2::jsonb,
			$3,
			$4,
			$5,
			$6,
			$7::platform[],
			$8,
			$9,
			$10,
			$11
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
	).Scan(&courseID); err != nil {
		return "", wrapPGError("repository postgres course packages course insert", err)
	}

	return courseID, nil
}

func (r *CoursePackageRepository) insertQuiz(ctx context.Context, tx pgx.Tx, params domain.CreateQuizParams) (string, error) {
	titleValue, err := toJSONValue(params.Title)
	if err != nil {
		return "", fmt.Errorf("repository postgres course packages quiz title value: %w", err)
	}

	descriptionValue, err := multiLangValueOrNil(params.Description)
	if err != nil {
		return "", fmt.Errorf("repository postgres course packages quiz description value: %w", err)
	}

	var quizID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO quizzes (
			title,
			description,
			category,
			status,
			platforms,
			time_limit_minutes,
			passing_score,
			max_attempts,
			shuffle_questions,
			show_results,
			allow_retry
		) VALUES (
			$1::jsonb,
			$2::jsonb,
			$3,
			$4,
			$5::platform[],
			$6,
			$7,
			$8,
			$9,
			$10,
			$11
		)
		RETURNING id
	`,
		titleValue,
		descriptionValue,
		nullableStringPointerForWrite(params.Category),
		string(params.Status),
		platformsToStrings(params.Platforms),
		nullableIntPointerForWrite(params.TimeLimitMinutes),
		params.PassingScore,
		params.MaxAttempts,
		params.ShuffleQuestions,
		params.ShowResults,
		params.AllowRetry,
	).Scan(&quizID); err != nil {
		return "", wrapPGError("repository postgres course packages quiz insert", err)
	}

	return quizID, nil
}

func (r *CoursePackageRepository) insertCourseTest(ctx context.Context, tx pgx.Tx, courseID, quizID string, position int) (domain.CourseTest, error) {
	var courseTest domain.CourseTest
	var scannedCourseID sql.NullString
	var scannedModuleID sql.NullString

	if err := tx.QueryRow(ctx, `
		INSERT INTO course_tests (
			course_id,
			module_id,
			quiz_id,
			position
		) VALUES (
			$1::uuid,
			NULL,
			$2::uuid,
			$3
		)
		RETURNING id, course_id::text, module_id::text, quiz_id::text, position
	`, courseID, quizID, position).Scan(
		&courseTest.ID,
		&scannedCourseID,
		&scannedModuleID,
		&courseTest.QuizID,
		&courseTest.Position,
	); err != nil {
		return domain.CourseTest{}, wrapPGError("repository postgres course packages course test insert", err)
	}

	courseTest.CourseID = optionalString(scannedCourseID)
	courseTest.ModuleID = optionalString(scannedModuleID)
	return courseTest, nil
}
