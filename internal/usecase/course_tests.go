package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type courseTestRepository interface {
	Create(ctx context.Context, params domain.CreateCourseTestParams) (domain.CourseTest, error)
	List(ctx context.Context, filter domain.CourseTestListFilter) ([]domain.CourseTest, int, error)
	Delete(ctx context.Context, courseID, moduleID, quizID string) error
	DeleteByID(ctx context.Context, courseTestID string) error
}

type CourseTestUseCase struct {
	repository courseTestRepository
}

func NewCourseTestUseCase(repository courseTestRepository) *CourseTestUseCase {
	return &CourseTestUseCase{repository: repository}
}

func (u *CourseTestUseCase) Create(ctx context.Context, params domain.CreateCourseTestParams) (domain.CourseTest, error) {
	normalized, err := normalizeCreateCourseTestParams(params)
	if err != nil {
		return domain.CourseTest{}, fmt.Errorf("usecase course tests create: %w", err)
	}

	courseTest, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.CourseTest{}, fmt.Errorf("usecase course tests create: %w", err)
	}

	return courseTest, nil
}

func (u *CourseTestUseCase) List(ctx context.Context, filter domain.CourseTestListFilter) ([]domain.CourseTest, int, error) {
	normalized, err := normalizeCourseTestListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase course tests list: %w", err)
	}

	courseTests, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase course tests list: %w", err)
	}

	return courseTests, total, nil
}

func (u *CourseTestUseCase) Delete(ctx context.Context, courseID, moduleID, quizID string) error {
	courseID = strings.TrimSpace(courseID)
	moduleID = strings.TrimSpace(moduleID)
	quizID = strings.TrimSpace(quizID)

	if (courseID == "" && moduleID == "") || (courseID != "" && moduleID != "") {
		return fmt.Errorf("usecase course tests delete target: %w", domain.ErrValidation)
	}

	if quizID == "" {
		return fmt.Errorf("usecase course tests delete quiz id: %w", domain.ErrValidation)
	}

	if err := u.repository.Delete(ctx, courseID, moduleID, quizID); err != nil {
		return fmt.Errorf("usecase course tests delete: %w", err)
	}

	return nil
}

func (u *CourseTestUseCase) DeleteByID(ctx context.Context, courseTestID string) error {
	courseTestID = strings.TrimSpace(courseTestID)
	if courseTestID == "" {
		return fmt.Errorf("usecase course tests delete by id: %w", domain.ErrValidation)
	}

	if err := u.repository.DeleteByID(ctx, courseTestID); err != nil {
		return fmt.Errorf("usecase course tests delete by id: %w", err)
	}

	return nil
}

func normalizeCreateCourseTestParams(params domain.CreateCourseTestParams) (domain.CreateCourseTestParams, error) {
	if params.CourseID != nil {
		params.CourseID = normalizeOptionalString(params.CourseID)
	}

	if params.ModuleID != nil {
		params.ModuleID = normalizeOptionalString(params.ModuleID)
	}

	params.QuizID = strings.TrimSpace(params.QuizID)

	if (params.CourseID == nil && params.ModuleID == nil) || (params.CourseID != nil && params.ModuleID != nil) {
		return domain.CreateCourseTestParams{}, fmt.Errorf("exactly one of course_id or module_id is required: %w", domain.ErrValidation)
	}

	if params.QuizID == "" {
		return domain.CreateCourseTestParams{}, fmt.Errorf("quiz_id is required: %w", domain.ErrValidation)
	}

	if params.Position <= 0 {
		params.Position = 1
	}

	return params, nil
}

func normalizeCourseTestListFilter(filter domain.CourseTestListFilter) (domain.CourseTestListFilter, error) {
	if filter.CourseID != nil {
		filter.CourseID = normalizeOptionalString(filter.CourseID)
	}

	if filter.ModuleID != nil {
		filter.ModuleID = normalizeOptionalString(filter.ModuleID)
	}

	if filter.CourseID != nil && filter.ModuleID != nil {
		return domain.CourseTestListFilter{}, fmt.Errorf("only one of course_id or module_id is allowed: %w", domain.ErrValidation)
	}

	return filter, nil
}
