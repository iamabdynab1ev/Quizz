package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type courseModuleRepository interface {
	Create(ctx context.Context, params domain.CreateCourseModuleParams) (domain.CourseModule, error)
	GetByID(ctx context.Context, moduleID string) (domain.CourseModule, error)
	List(ctx context.Context, filter domain.CourseModuleListFilter) ([]domain.CourseModule, int, error)
	Update(ctx context.Context, params domain.UpdateCourseModuleParams) (domain.CourseModule, error)
	Delete(ctx context.Context, moduleID string) error
}

type CourseModuleUseCase struct {
	repository courseModuleRepository
}

func NewCourseModuleUseCase(repository courseModuleRepository) *CourseModuleUseCase {
	return &CourseModuleUseCase{repository: repository}
}

func (u *CourseModuleUseCase) Create(ctx context.Context, params domain.CreateCourseModuleParams) (domain.CourseModule, error) {
	normalized, err := normalizeCreateCourseModuleParams(params)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("usecase course modules create: %w", err)
	}

	module, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("usecase course modules create: %w", err)
	}

	return module, nil
}

func (u *CourseModuleUseCase) GetByID(ctx context.Context, moduleID string) (domain.CourseModule, error) {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return domain.CourseModule{}, fmt.Errorf("usecase course modules get by id: %w", domain.ErrValidation)
	}

	module, err := u.repository.GetByID(ctx, moduleID)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("usecase course modules get by id: %w", err)
	}

	return module, nil
}

func (u *CourseModuleUseCase) List(ctx context.Context, filter domain.CourseModuleListFilter) ([]domain.CourseModule, int, error) {
	normalized, err := normalizeCourseModuleListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase course modules list: %w", err)
	}

	modules, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase course modules list: %w", err)
	}

	return modules, total, nil
}

func (u *CourseModuleUseCase) Update(ctx context.Context, params domain.UpdateCourseModuleParams) (domain.CourseModule, error) {
	normalized, err := normalizeUpdateCourseModuleParams(params)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("usecase course modules update: %w", err)
	}

	module, err := u.repository.Update(ctx, normalized)
	if err != nil {
		return domain.CourseModule{}, fmt.Errorf("usecase course modules update: %w", err)
	}

	return module, nil
}

func (u *CourseModuleUseCase) Delete(ctx context.Context, moduleID string) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return fmt.Errorf("usecase course modules delete: %w", domain.ErrValidation)
	}

	if err := u.repository.Delete(ctx, moduleID); err != nil {
		return fmt.Errorf("usecase course modules delete: %w", err)
	}

	return nil
}

func normalizeCreateCourseModuleParams(params domain.CreateCourseModuleParams) (domain.CreateCourseModuleParams, error) {
	params.CourseID = strings.TrimSpace(params.CourseID)

	if params.CourseID == "" {
		return domain.CreateCourseModuleParams{}, fmt.Errorf("course_id is required: %w", domain.ErrValidation)
	}

	if params.Position <= 0 {
		return domain.CreateCourseModuleParams{}, fmt.Errorf("position must be greater than zero: %w", domain.ErrValidation)
	}

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.CreateCourseModuleParams{}, fmt.Errorf("title is invalid: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeUpdateCourseModuleParams(params domain.UpdateCourseModuleParams) (domain.UpdateCourseModuleParams, error) {
	params.ID = strings.TrimSpace(params.ID)
	params.CourseID = strings.TrimSpace(params.CourseID)

	if params.ID == "" || params.CourseID == "" {
		return domain.UpdateCourseModuleParams{}, fmt.Errorf("id and course_id are required: %w", domain.ErrValidation)
	}

	if params.Position <= 0 {
		return domain.UpdateCourseModuleParams{}, fmt.Errorf("position must be greater than zero: %w", domain.ErrValidation)
	}

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.UpdateCourseModuleParams{}, fmt.Errorf("title is invalid: %w", domain.ErrValidation)
	}

	return params, nil
}

func normalizeCourseModuleListFilter(filter domain.CourseModuleListFilter) (domain.CourseModuleListFilter, error) {
	filter.CourseID = strings.TrimSpace(filter.CourseID)
	if filter.CourseID == "" {
		return domain.CourseModuleListFilter{}, fmt.Errorf("course_id is required: %w", domain.ErrValidation)
	}

	return filter, nil
}
