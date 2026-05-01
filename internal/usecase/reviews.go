package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type reviewRepository interface {
	Create(ctx context.Context, params domain.CreateReviewParams) (domain.Review, error)
	GetByID(ctx context.Context, reviewID string) (domain.Review, error)
	List(ctx context.Context, filter domain.ReviewListFilter) ([]domain.Review, int, error)
	Moderate(ctx context.Context, params domain.ModerateReviewParams) (domain.Review, error)
}

type ReviewUseCase struct {
	repository reviewRepository
	audit      *AuditLogger
}

func NewReviewUseCase(repository reviewRepository) *ReviewUseCase {
	return &ReviewUseCase{repository: repository}
}

func (u *ReviewUseCase) WithAudit(audit *AuditLogger) *ReviewUseCase {
	u.audit = audit
	return u
}

func (u *ReviewUseCase) Create(ctx context.Context, params domain.CreateReviewParams) (domain.Review, error) {
	normalized, err := normalizeCreateReviewParams(params)
	if err != nil {
		return domain.Review{}, fmt.Errorf("usecase reviews create: %w", err)
	}

	review, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.Review{}, fmt.Errorf("usecase reviews create: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventReviewCreated, map[string]any{
			"review_id":  review.ID,
			"course_id":  review.CourseID,
			"user_id":    review.UserID,
			"rating":     review.Rating,
			"status":     review.Status,
			"created_at": review.CreatedAt,
		})
	}

	return review, nil
}

func (u *ReviewUseCase) GetByID(ctx context.Context, reviewID string) (domain.Review, error) {
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" {
		return domain.Review{}, fmt.Errorf("usecase reviews get by id: %w", domain.ErrValidation)
	}

	review, err := u.repository.GetByID(ctx, reviewID)
	if err != nil {
		return domain.Review{}, fmt.Errorf("usecase reviews get by id: %w", err)
	}

	return review, nil
}

func (u *ReviewUseCase) List(ctx context.Context, filter domain.ReviewListFilter) ([]domain.Review, int, error) {
	normalized, err := normalizeReviewListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase reviews list: %w", err)
	}

	reviews, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase reviews list: %w", err)
	}

	return reviews, total, nil
}

func (u *ReviewUseCase) Moderate(ctx context.Context, params domain.ModerateReviewParams) (domain.Review, error) {
	normalized, err := normalizeModerateReviewParams(params)
	if err != nil {
		return domain.Review{}, fmt.Errorf("usecase reviews moderate: %w", err)
	}

	review, err := u.repository.Moderate(ctx, normalized)
	if err != nil {
		return domain.Review{}, fmt.Errorf("usecase reviews moderate: %w", err)
	}

	if u.audit != nil {
		eventType := domain.AppEventReviewRejected
		if review.Status == domain.ReviewStatusApproved {
			eventType = domain.AppEventReviewApproved
		}

		u.audit.Log(ctx, eventType, map[string]any{
			"review_id":    review.ID,
			"course_id":    review.CourseID,
			"user_id":      review.UserID,
			"status":       review.Status,
			"moderated_at": review.ModeratedAt,
		})
	}

	return review, nil
}

func normalizeCreateReviewParams(params domain.CreateReviewParams) (domain.CreateReviewParams, error) {
	params.CourseID = strings.TrimSpace(params.CourseID)
	params.UserID = normalizeOptionalString(params.UserID)
	params.Text = normalizeOptionalString(params.Text)

	if params.CourseID == "" {
		return domain.CreateReviewParams{}, fmt.Errorf("course_id is required: %w", domain.ErrValidation)
	}

	if params.Rating < 1 || params.Rating > 5 {
		return domain.CreateReviewParams{}, fmt.Errorf("rating must be in range 1..5: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeModerateReviewParams(params domain.ModerateReviewParams) (domain.ModerateReviewParams, error) {
	params.ID = strings.TrimSpace(params.ID)

	if params.ID == "" {
		return domain.ModerateReviewParams{}, fmt.Errorf("id is required: %w", domain.ErrValidation)
	}

	if !params.Status.IsValid() || params.Status == domain.ReviewStatusPending {
		return domain.ModerateReviewParams{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeReviewListFilter(filter domain.ReviewListFilter) (domain.ReviewListFilter, error) {
	if filter.CourseID != nil {
		filter.CourseID = normalizeOptionalString(filter.CourseID)
	}

	if filter.UserID != nil {
		filter.UserID = normalizeOptionalString(filter.UserID)
	}

	if filter.Status != nil {
		status := domain.ReviewStatus(strings.TrimSpace(string(*filter.Status)))
		if !status.IsValid() {
			return domain.ReviewListFilter{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
		}
		filter.Status = &status
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		return domain.ReviewListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}
