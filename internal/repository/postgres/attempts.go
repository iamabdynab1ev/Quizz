package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type attemptRowScanner interface {
	Scan(dest ...any) error
}

type AttemptRepository struct {
	pool *pgxpool.Pool
}

func NewAttemptRepository(pool *pgxpool.Pool) *AttemptRepository {
	return &AttemptRepository{pool: pool}
}

func (r *AttemptRepository) GetCourseForAttempt(ctx context.Context, courseID string) (domain.Course, error) {
	courseRepository := NewCourseRepository(r.pool)
	course, err := courseRepository.GetByID(ctx, courseID)
	if err != nil {
		return domain.Course{}, fmt.Errorf("repository postgres attempts get course for attempt: %w", err)
	}
	return course, nil
}

func (r *AttemptRepository) GetUserCourseAttemptWindow(ctx context.Context, courseID, userID string, since *time.Time) (domain.AttemptWindow, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT COUNT(*), MIN(started_at)
		FROM attempts
		WHERE course_id = $1 AND user_id = $2
	`)

	args := []any{courseID, userID}
	if since != nil {
		query.WriteString(" AND started_at > $3")
		args = append(args, *since)
	}

	var count int
	var earliestStartedAt sql.NullTime
	if err := r.pool.QueryRow(ctx, query.String(), args...).Scan(&count, &earliestStartedAt); err != nil {
		return domain.AttemptWindow{}, fmt.Errorf("repository postgres attempts get user course attempt window: %w", err)
	}

	return domain.AttemptWindow{
		Count:             count,
		EarliestStartedAt: optionalTime(earliestStartedAt),
	}, nil
}

func (r *AttemptRepository) UserHasCourseCertificate(ctx context.Context, courseID, userID string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM certificates
			WHERE course_id = $1 AND user_id = $2
		)
	`, courseID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("repository postgres attempts user has course certificate: %w", err)
	}
	return exists, nil
}

func (r *AttemptRepository) CreateAttempt(ctx context.Context, params domain.CreateAttemptRecordParams) (domain.Attempt, error) {
	var attemptID string
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO attempts (
			course_id, user_id, started_at, finished_at,
			questions_snapshot, answers_data,
			total_earned, total_max, score_percent, passed
		) VALUES (
			$1, $2, $3, $4,
			$5::jsonb, $6::jsonb,
			$7, $8, $9, $10
		)
		RETURNING id
	`,
		params.CourseID,
		params.UserID,
		params.StartedAt,
		params.FinishedAt,
		[]byte(params.QuestionsSnapshot),
		[]byte(params.AnswersData),
		params.TotalEarned,
		params.TotalMax,
		params.ScorePercent,
		params.Passed,
	).Scan(&attemptID); err != nil {
		return domain.Attempt{}, wrapPGError("repository postgres attempts create", err)
	}

	attempt, err := r.GetAttemptByID(ctx, attemptID)
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("repository postgres attempts create fetch by id: %w", err)
	}

	return attempt, nil
}

func (r *AttemptRepository) GetAttemptByID(ctx context.Context, attemptID string) (domain.Attempt, error) {
	attempt, err := scanAttemptRow(r.pool.QueryRow(ctx, `
		SELECT
			id, course_id, user_id, started_at, finished_at,
			questions_snapshot, answers_data,
			total_earned, total_max, score_percent, passed
		FROM attempts
		WHERE id = $1
	`, attemptID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Attempt{}, fmt.Errorf("repository postgres attempts get by id: %w", domain.ErrNotFound)
		}
		return domain.Attempt{}, fmt.Errorf("repository postgres attempts get by id: %w", err)
	}
	return attempt, nil
}

func (r *AttemptRepository) ListAttempts(ctx context.Context, filter domain.AttemptListFilter) ([]domain.Attempt, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT
					id, course_id, user_id, started_at, finished_at,
					questions_snapshot, answers_data,
					total_earned, total_max, score_percent, passed
				FROM attempts
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`SELECT COUNT(*) FROM attempts WHERE 1 = 1`)
		}

		args := make([]any, 0, 4)
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

		if includePagination {
			query.WriteString(fmt.Sprintf(" ORDER BY started_at DESC LIMIT $%d OFFSET $%d", position, position+1))
			args = append(args, filter.Limit, filter.Offset)
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres attempts list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres attempts list query: %w", err)
	}
	defer rows.Close()

	attempts := make([]domain.Attempt, 0, filter.Limit)
	for rows.Next() {
		attempt, err := scanAttemptRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres attempts list scan: %w", err)
		}
		attempts = append(attempts, attempt)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres attempts list rows: %w", err)
	}

	return attempts, total, nil
}

func scanAttemptRow(scanner attemptRowScanner) (domain.Attempt, error) {
	var attempt domain.Attempt
	var userID sql.NullString
	var finishedAt sql.NullTime

	if err := scanner.Scan(
		&attempt.ID,
		&attempt.CourseID,
		&userID,
		&attempt.StartedAt,
		&finishedAt,
		&attempt.QuestionsSnapshot,
		&attempt.AnswersData,
		&attempt.TotalEarned,
		&attempt.TotalMax,
		&attempt.ScorePercent,
		&attempt.Passed,
	); err != nil {
		return domain.Attempt{}, err
	}

	attempt.UserID = optionalString(userID)
	attempt.FinishedAt = optionalTime(finishedAt)

	return attempt, nil
}
