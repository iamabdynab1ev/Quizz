package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type courseRepository interface {
	Create(ctx context.Context, params domain.CreateCourseParams) (domain.Course, error)
	GetByID(ctx context.Context, courseID string) (domain.Course, error)
	List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int, error)
	Update(ctx context.Context, params domain.UpdateCourseParams) (domain.Course, error)
	Archive(ctx context.Context, courseID string) error
}

type CourseUseCase struct {
	repository courseRepository
	audit      *AuditLogger
}

func NewCourseUseCase(repository courseRepository) *CourseUseCase {
	return &CourseUseCase{repository: repository}
}

func (u *CourseUseCase) WithAudit(audit *AuditLogger) *CourseUseCase {
	u.audit = audit
	return u
}

func (u *CourseUseCase) Create(ctx context.Context, params domain.CreateCourseParams) (domain.Course, error) {
	normalized, err := normalizeCreateCourseParams(params)
	if err != nil {
		return domain.Course{}, fmt.Errorf("usecase courses create: %w", err)
	}

	course, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.Course{}, fmt.Errorf("usecase courses create: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventCourseCreated, map[string]any{
			"course_id": course.ID,
			"title":     course.Title,
			"status":    course.Status,
		})
	}

	return course, nil
}

func (u *CourseUseCase) GetByID(ctx context.Context, courseID string) (domain.Course, error) {
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return domain.Course{}, fmt.Errorf("usecase courses get by id: %w", domain.ErrValidation)
	}

	course, err := u.repository.GetByID(ctx, courseID)
	if err != nil {
		return domain.Course{}, fmt.Errorf("usecase courses get by id: %w", err)
	}

	return course, nil
}

func (u *CourseUseCase) List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int, error) {
	normalized, err := normalizeCourseListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase courses list: %w", err)
	}

	courses, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase courses list: %w", err)
	}

	return courses, total, nil
}

func (u *CourseUseCase) Update(ctx context.Context, params domain.UpdateCourseParams) (domain.Course, error) {
	normalized, err := normalizeUpdateCourseParams(params)
	if err != nil {
		return domain.Course{}, fmt.Errorf("usecase courses update: %w", err)
	}

	course, err := u.repository.Update(ctx, normalized)
	if err != nil {
		return domain.Course{}, fmt.Errorf("usecase courses update: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventCourseUpdated, map[string]any{
			"course_id": course.ID,
			"title":     course.Title,
			"status":    course.Status,
		})
	}

	return course, nil
}

func (u *CourseUseCase) Archive(ctx context.Context, courseID string) error {
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return fmt.Errorf("usecase courses archive: %w", domain.ErrValidation)
	}

	if err := u.repository.Archive(ctx, courseID); err != nil {
		return fmt.Errorf("usecase courses archive: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventCourseDeleted, map[string]any{
			"course_id": courseID,
		})
	}

	return nil
}

func normalizeCreateCourseParams(params domain.CreateCourseParams) (domain.CreateCourseParams, error) {
	params.CoverImageURL = normalizeOptionalString(params.CoverImageURL)
	params.Category = normalizeOptionalString(params.Category)

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.CreateCourseParams{}, fmt.Errorf("title is invalid: %w", domain.ErrValidation)
	}

	if params.Status == "" {
		params.Status = domain.CourseStatusDraft
	}

	if !params.Status.IsValid() {
		return domain.CreateCourseParams{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
	}

	if err := normalizePlatforms(&params.Platforms); err != nil {
		return domain.CreateCourseParams{}, fmt.Errorf("platforms are invalid: %w", err)
	}

	if params.EstimatedMinutes != nil && *params.EstimatedMinutes <= 0 {
		return domain.CreateCourseParams{}, fmt.Errorf("estimated_minutes must be greater than zero: %w", domain.ErrValidation)
	}

	if params.CertificatePassingScore < 0 || params.CertificatePassingScore > 100 {
		return domain.CreateCourseParams{}, fmt.Errorf("certificate_passing_score must be in range 0..100: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeUpdateCourseParams(params domain.UpdateCourseParams) (domain.UpdateCourseParams, error) {
	params.ID = strings.TrimSpace(params.ID)
	params.CoverImageURL = normalizeOptionalString(params.CoverImageURL)
	params.Category = normalizeOptionalString(params.Category)

	if params.ID == "" {
		return domain.UpdateCourseParams{}, fmt.Errorf("id is required: %w", domain.ErrValidation)
	}

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.UpdateCourseParams{}, fmt.Errorf("title is invalid: %w", domain.ErrValidation)
	}

	if !params.Status.IsValid() {
		return domain.UpdateCourseParams{}, fmt.Errorf("status is invalid: %w", domain.ErrValidation)
	}

	if err := normalizePlatforms(&params.Platforms); err != nil {
		return domain.UpdateCourseParams{}, fmt.Errorf("platforms are invalid: %w", err)
	}

	if params.EstimatedMinutes != nil && *params.EstimatedMinutes <= 0 {
		return domain.UpdateCourseParams{}, fmt.Errorf("estimated_minutes must be greater than zero: %w", domain.ErrValidation)
	}

	if params.CertificatePassingScore < 0 || params.CertificatePassingScore > 100 {
		return domain.UpdateCourseParams{}, fmt.Errorf("certificate_passing_score must be in range 0..100: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeCourseListFilter(filter domain.CourseListFilter) (domain.CourseListFilter, error) {
	filter.Search = strings.TrimSpace(filter.Search)

	if filter.Status != nil {
		status := domain.CourseStatus(strings.TrimSpace(string(*filter.Status)))
		if !status.IsValid() {
			return domain.CourseListFilter{}, fmt.Errorf("status filter is invalid: %w", domain.ErrValidation)
		}
		filter.Status = &status
	}

	if filter.Category != nil {
		filter.Category = normalizeOptionalString(filter.Category)
	}

	if filter.Platform != nil {
		platform := domain.Platform(strings.TrimSpace(string(*filter.Platform)))
		if !platform.IsValid() {
			return domain.CourseListFilter{}, fmt.Errorf("platform filter is invalid: %w", domain.ErrValidation)
		}
		filter.Platform = &platform
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Offset < 0 {
		return domain.CourseListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}

func normalizePlatforms(platforms *[]domain.Platform) error {
	if len(*platforms) == 0 {
		return nil
	}

	normalized := make([]domain.Platform, 0, len(*platforms))
	seen := make(map[domain.Platform]struct{}, len(*platforms))

	for _, platform := range *platforms {
		trimmed := domain.Platform(strings.TrimSpace(string(platform)))
		if !trimmed.IsValid() {
			return domain.ErrValidation
		}

		if _, exists := seen[trimmed]; exists {
			continue
		}

		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	*platforms = normalized
	return nil
}
