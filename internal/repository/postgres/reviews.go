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

type reviewRowScanner interface {
	Scan(dest ...any) error
}

type ReviewRepository struct {
	pool *pgxpool.Pool
}

func NewReviewRepository(pool *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{pool: pool}
}

func (r *ReviewRepository) Create(ctx context.Context, params domain.CreateReviewParams) (domain.Review, error) {
	review, err := scanReviewRow(r.pool.QueryRow(ctx, `
		INSERT INTO reviews (
			course_id,
			user_id,
			rating,
			text,
			status
		) VALUES (
			$1, $2::uuid, $3, $4, 'pending'
		)
		RETURNING id, course_id, user_id::text, rating, text, status, created_at, moderated_at
	`,
		params.CourseID,
		nullableStringPointerForWrite(params.UserID),
		params.Rating,
		nullableStringPointerForWrite(params.Text),
	))
	if err != nil {
		return domain.Review{}, wrapPGError("repository postgres reviews create", err)
	}

	return review, nil
}

func (r *ReviewRepository) GetByID(ctx context.Context, reviewID string) (domain.Review, error) {
	review, err := scanReviewRow(r.pool.QueryRow(ctx, `
		SELECT id, course_id, user_id::text, rating, text, status, created_at, moderated_at
		FROM reviews
		WHERE id = $1
	`, reviewID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Review{}, fmt.Errorf("repository postgres reviews get by id: %w", domain.ErrNotFound)
		}

		return domain.Review{}, fmt.Errorf("repository postgres reviews get by id: %w", err)
	}

	return review, nil
}

func (r *ReviewRepository) List(ctx context.Context, filter domain.ReviewListFilter) ([]domain.Review, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT id, course_id, user_id::text, rating, text, status, created_at, moderated_at
				FROM reviews
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM reviews
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 5)
		position := 1
		if filter.CourseID != nil {
			query.WriteString(fmt.Sprintf(" AND course_id = $%d::uuid", position))
			args = append(args, *filter.CourseID)
			position++
		}
		if filter.UserID != nil {
			query.WriteString(fmt.Sprintf(" AND user_id = $%d::uuid", position))
			args = append(args, *filter.UserID)
			position++
		}
		if filter.Status != nil {
			query.WriteString(fmt.Sprintf(" AND status = $%d", position))
			args = append(args, string(*filter.Status))
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
		return nil, 0, fmt.Errorf("repository postgres reviews list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres reviews list query: %w", err)
	}
	defer rows.Close()

	reviews := make([]domain.Review, 0, filter.Limit)
	for rows.Next() {
		review, err := scanReviewRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres reviews list scan: %w", err)
		}
		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres reviews list rows: %w", err)
	}

	return reviews, total, nil
}

func (r *ReviewRepository) Moderate(ctx context.Context, params domain.ModerateReviewParams) (domain.Review, error) {
	review, err := scanReviewRow(r.pool.QueryRow(ctx, `
		UPDATE reviews
		SET
			status = $2,
			moderated_at = NOW()
		WHERE id = $1
		RETURNING id, course_id, user_id::text, rating, text, status, created_at, moderated_at
	`,
		params.ID,
		string(params.Status),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Review{}, fmt.Errorf("repository postgres reviews moderate: %w", domain.ErrNotFound)
		}
		return domain.Review{}, fmt.Errorf("repository postgres reviews moderate: %w", err)
	}

	return review, nil
}

func scanReviewRow(scanner reviewRowScanner) (domain.Review, error) {
	var review domain.Review
	var userID sql.NullString
	var text sql.NullString
	var status string
	var moderatedAt sql.NullTime

	if err := scanner.Scan(
		&review.ID,
		&review.CourseID,
		&userID,
		&review.Rating,
		&text,
		&status,
		&review.CreatedAt,
		&moderatedAt,
	); err != nil {
		return domain.Review{}, err
	}

	review.UserID = optionalString(userID)
	review.Text = optionalString(text)
	review.Status = domain.ReviewStatus(status)
	review.ModeratedAt = optionalTime(moderatedAt)
	return review, nil
}
