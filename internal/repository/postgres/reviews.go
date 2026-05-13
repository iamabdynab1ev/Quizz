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
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, course_id, user_id::text, rating, text, status, created_at, moderated_at,
			COUNT(*) OVER() AS total_count
		FROM reviews
		WHERE 1 = 1
	`)

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

	query.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", position, position+1))
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres reviews list query: %w", err)
	}
	defer rows.Close()

	var total int
	reviews := make([]domain.Review, 0, filter.Limit)
	for rows.Next() {
		review, rowTotal, err := scanReviewRowWithTotal(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres reviews list scan: %w", err)
		}
		total = rowTotal
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
	var userID, text sql.NullString
	var status string
	var moderatedAt sql.NullTime

	if err := scanner.Scan(
		&review.ID, &review.CourseID, &userID, &review.Rating,
		&text, &status, &review.CreatedAt, &moderatedAt,
	); err != nil {
		return domain.Review{}, err
	}

	review.UserID = optionalString(userID)
	review.Text = optionalString(text)
	review.Status = domain.ReviewStatus(status)
	review.ModeratedAt = optionalTime(moderatedAt)
	return review, nil
}

func scanReviewRowWithTotal(scanner reviewRowScanner) (domain.Review, int, error) {
	var review domain.Review
	var userID, text sql.NullString
	var status string
	var moderatedAt sql.NullTime
	var total int

	if err := scanner.Scan(
		&review.ID, &review.CourseID, &userID, &review.Rating,
		&text, &status, &review.CreatedAt, &moderatedAt,
		&total,
	); err != nil {
		return domain.Review{}, 0, err
	}

	review.UserID = optionalString(userID)
	review.Text = optionalString(text)
	review.Status = domain.ReviewStatus(status)
	review.ModeratedAt = optionalTime(moderatedAt)
	return review, total, nil
}
