package usecase

import (
	"context"
	"fmt"

	"lms-arvand-backend/internal/domain"
)

type coursePackageRepository interface {
	Create(ctx context.Context, params domain.CreateCoursePackageParams) (domain.Course, error)
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

func (u *CoursePackageUseCase) Create(ctx context.Context, params domain.CreateCoursePackageParams) (domain.Course, error) {
	course, err := normalizeCreateCourseParams(params.Course)
	if err != nil {
		return domain.Course{}, fmt.Errorf("usecase course packages create: %w", err)
	}

	params.Course = course

	created, err := u.repository.Create(ctx, params)
	if err != nil {
		return domain.Course{}, fmt.Errorf("usecase course packages create: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventCourseCreated, map[string]any{
			"course_id": created.ID,
			"title":     created.Title,
			"status":    created.Status,
		})
	}

	return created, nil
}
