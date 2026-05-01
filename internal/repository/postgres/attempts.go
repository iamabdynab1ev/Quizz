package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

func (r *AttemptRepository) GetQuizForAttempt(ctx context.Context, quizID string) (domain.Quiz, error) {
	quizRepository := NewQuizRepository(r.pool)
	quiz, err := quizRepository.GetByID(ctx, quizID)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("repository postgres attempts get quiz for attempt: %w", err)
	}

	return quiz, nil
}

func (r *AttemptRepository) CountUserQuizAttempts(ctx context.Context, quizID, userID string) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM attempts
		WHERE quiz_id = $1 AND user_id = $2
	`, quizID, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("repository postgres attempts count user quiz attempts: %w", err)
	}

	return count, nil
}

func (r *AttemptRepository) CreateAttempt(ctx context.Context, params domain.CreateAttemptRecordParams) (domain.Attempt, error) {
	var attemptID string
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO attempts (
			quiz_id,
			user_id,
			started_at,
			finished_at,
			questions_snapshot,
			answers_data,
			total_earned,
			total_max,
			score_percent,
			passed,
			needs_review
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5::jsonb,
			$6::jsonb,
			$7,
			$8,
			$9,
			$10,
			$11
		)
		RETURNING id
	`,
		params.QuizID,
		params.UserID,
		params.StartedAt,
		params.FinishedAt,
		[]byte(params.QuestionsSnapshot),
		[]byte(params.AnswersData),
		params.TotalEarned,
		params.TotalMax,
		params.ScorePercent,
		params.Passed,
		params.NeedsReview,
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
			id,
			quiz_id,
			user_id,
			started_at,
			finished_at,
			questions_snapshot,
			answers_data,
			total_earned,
			total_max,
			score_percent,
			passed,
			needs_review,
			reviewed_at,
			reviewer_id::text,
			review_comment,
			manual_passed,
			review_scores
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
					id,
					quiz_id,
					user_id,
					started_at,
					finished_at,
					questions_snapshot,
					answers_data,
					total_earned,
					total_max,
					score_percent,
					passed,
					needs_review,
					reviewed_at,
					reviewer_id::text,
					review_comment,
					manual_passed,
					review_scores
				FROM attempts
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM attempts
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 4)
		position := 1

		if filter.QuizID != nil {
			query.WriteString(fmt.Sprintf(" AND quiz_id = $%d", position))
			args = append(args, *filter.QuizID)
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

func (r *AttemptRepository) UpdateReview(ctx context.Context, params domain.ReviewAttemptParams) (domain.Attempt, error) {
	attempt, err := scanAttemptRow(r.pool.QueryRow(ctx, `
		UPDATE attempts
		SET
			passed = $2,
			needs_review = false,
			reviewed_at = NOW(),
			reviewer_id = $3::uuid,
			review_comment = $4,
			manual_passed = $2,
			total_earned = $5,
			score_percent = $6,
			review_scores = $7::jsonb
		WHERE id = $1
			AND needs_review = true
		RETURNING
			id,
			quiz_id,
			user_id,
			started_at,
			finished_at,
			questions_snapshot,
			answers_data,
			total_earned,
			total_max,
			score_percent,
			passed,
			needs_review,
			reviewed_at,
			reviewer_id::text,
			review_comment,
			manual_passed,
			review_scores
	`, params.AttemptID, params.Passed, params.ReviewerID, nullableStringPointerForWrite(params.Comment), params.TotalEarned, params.ScorePercent, []byte(params.ReviewScores)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Attempt{}, fmt.Errorf("repository postgres attempts update review: %w", domain.ErrNotFound)
		}

		return domain.Attempt{}, fmt.Errorf("repository postgres attempts update review: %w", err)
	}

	return attempt, nil
}

func scanAttemptRow(scanner attemptRowScanner) (domain.Attempt, error) {
	var attempt domain.Attempt
	var userID sql.NullString
	var finishedAt sql.NullTime
	var reviewedAt sql.NullTime
	var reviewerID sql.NullString
	var reviewComment sql.NullString
	var manualPassed sql.NullBool
	var reviewScores json.RawMessage

	if err := scanner.Scan(
		&attempt.ID,
		&attempt.QuizID,
		&userID,
		&attempt.StartedAt,
		&finishedAt,
		&attempt.QuestionsSnapshot,
		&attempt.AnswersData,
		&attempt.TotalEarned,
		&attempt.TotalMax,
		&attempt.ScorePercent,
		&attempt.Passed,
		&attempt.NeedsReview,
		&reviewedAt,
		&reviewerID,
		&reviewComment,
		&manualPassed,
		&reviewScores,
	); err != nil {
		return domain.Attempt{}, err
	}

	attempt.UserID = optionalString(userID)
	attempt.FinishedAt = optionalTime(finishedAt)
	attempt.ReviewedAt = optionalTime(reviewedAt)
	attempt.ReviewerID = optionalString(reviewerID)
	attempt.ReviewComment = optionalString(reviewComment)
	attempt.ManualPassed = optionalBool(manualPassed)
	if len(reviewScores) > 0 {
		if err := json.Unmarshal(reviewScores, &attempt.ReviewScores); err != nil {
			return domain.Attempt{}, fmt.Errorf("scan attempt review scores: %w", err)
		}
	}

	return attempt, nil
}
