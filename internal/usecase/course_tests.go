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

	var validation fieldValidationBuilder
	if (courseID == "" && moduleID == "") || (courseID != "" && moduleID != "") {
		validation.add("course_id", "invalid_target", "Передайте course_id или module_id")
		validation.add("module_id", "invalid_target", "Передайте course_id или module_id")
	}

	if quizID == "" {
		validation.add("quiz_id", "required", "ID теста обязателен")
	}

	if err := validation.err(); err != nil {
		return fmt.Errorf("usecase course tests delete target: %w", err)
	}

	if err := u.repository.Delete(ctx, courseID, moduleID, quizID); err != nil {
		return fmt.Errorf("usecase course tests delete: %w", err)
	}

	return nil
}

func (u *CourseTestUseCase) DeleteByID(ctx context.Context, courseTestID string) error {
	courseTestID = strings.TrimSpace(courseTestID)
	if courseTestID == "" {
		return fmt.Errorf("usecase course tests delete by id: %w", domain.FieldValidationError("Проверьте поля формы",
			domain.ValidationField("id", "required", "ID связи курса и теста обязателен")))
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

	var validation fieldValidationBuilder
	if (params.CourseID == nil && params.ModuleID == nil) || (params.CourseID != nil && params.ModuleID != nil) {
		validation.add("course_id", "invalid_target", "Передайте course_id или module_id")
		validation.add("module_id", "invalid_target", "Передайте course_id или module_id")
	}

	if params.QuizID == "" {
		validation.add("quiz_id", "required", "ID теста обязателен")
	}

	if params.Position <= 0 {
		params.Position = 1
	}

	if err := validation.err(); err != nil {
		return domain.CreateCourseTestParams{}, err
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
		return domain.CourseTestListFilter{}, domain.FieldValidationError("Проверьте параметры запроса",
			domain.ValidationField("course_id", "mutually_exclusive", "Передайте только course_id или только module_id"),
			domain.ValidationField("module_id", "mutually_exclusive", "Передайте только course_id или только module_id"))
	}

	return filter, nil
}
