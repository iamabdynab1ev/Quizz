package usecase

import (
	"context"
	"fmt"

	"lms-arvand-backend/internal/domain"
)

type coursePackageRepository interface {
	Create(ctx context.Context, params domain.CreateCoursePackageParams) (domain.CoursePackage, error)
}

type CoursePackageUseCase struct {
	repository coursePackageRepository
	audit      *AuditLogger
}

func NewCoursePackageUseCase(repository coursePackageRepository) *CoursePackageUseCase {
	return &CoursePackageUseCase{repository: repository}
}

func (u *CoursePackageUseCase) WithAudit(audit *AuditLogger) *CoursePackageUseCase {
	u.audit = audit
	return u
}

func (u *CoursePackageUseCase) Create(ctx context.Context, params domain.CreateCoursePackageParams) (domain.CoursePackage, error) {
	normalized, err := normalizeCreateCoursePackageParams(params)
	if err != nil {
		return domain.CoursePackage{}, fmt.Errorf("usecase course packages create: %w", err)
	}

	created, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.CoursePackage{}, fmt.Errorf("usecase course packages create: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventCourseCreated, map[string]any{
			"course_id": created.Course.ID,
			"quiz_id":   created.Quiz.ID,
			"title":     created.Course.Title,
			"status":    created.Course.Status,
		})
		u.audit.Log(ctx, domain.AppEventTestCreated, map[string]any{
			"course_id": created.Course.ID,
			"quiz_id":   created.Quiz.ID,
			"title":     created.Quiz.Title,
			"status":    created.Quiz.Status,
		})
	}

	return created, nil
}

func normalizeCreateCoursePackageParams(params domain.CreateCoursePackageParams) (domain.CreateCoursePackageParams, error) {
	course, err := normalizeCreateCourseParams(params.Course)
	if err != nil {
		return domain.CreateCoursePackageParams{}, err
	}

	if params.Quiz.Title.IsZero() {
		params.Quiz.Title = course.Title
	}
	if params.Quiz.Description.IsZero() {
		params.Quiz.Description = course.Description
	}
	if params.Quiz.Status == "" {
		params.Quiz.Status = domain.QuizStatusDraft
	}

	quiz, err := normalizeCreateQuizParams(params.Quiz)
	if err != nil {
		return domain.CreateCoursePackageParams{}, err
	}

	if params.LinkPosition <= 0 {
		params.LinkPosition = 1
	}

	return domain.CreateCoursePackageParams{
		Course:       course,
		Quiz:         quiz,
		LinkPosition: params.LinkPosition,
	}, nil
}
