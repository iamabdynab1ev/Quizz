package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type contentBlockRepository interface {
	Create(ctx context.Context, params domain.CreateContentBlockParams) (domain.ContentBlock, error)
	GetByID(ctx context.Context, blockID string) (domain.ContentBlock, error)
	List(ctx context.Context, filter domain.ContentBlockListFilter) ([]domain.ContentBlock, int, error)
	Update(ctx context.Context, params domain.UpdateContentBlockParams) (domain.ContentBlock, error)
	Delete(ctx context.Context, blockID string) error
}

type ContentBlockUseCase struct {
	repository contentBlockRepository
}

func NewContentBlockUseCase(repository contentBlockRepository) *ContentBlockUseCase {
	return &ContentBlockUseCase{repository: repository}
}

func (u *ContentBlockUseCase) Create(ctx context.Context, params domain.CreateContentBlockParams) (domain.ContentBlock, error) {
	normalized, err := normalizeCreateContentBlockParams(params)
	if err != nil {
		return domain.ContentBlock{}, fmt.Errorf("usecase content blocks create: %w", err)
	}

	block, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.ContentBlock{}, fmt.Errorf("usecase content blocks create: %w", err)
	}

	return block, nil
}

func (u *ContentBlockUseCase) GetByID(ctx context.Context, blockID string) (domain.ContentBlock, error) {
	blockID = strings.TrimSpace(blockID)
	if blockID == "" {
		return domain.ContentBlock{}, fmt.Errorf("usecase content blocks get by id: %w", domain.ErrValidation)
	}

	block, err := u.repository.GetByID(ctx, blockID)
	if err != nil {
		return domain.ContentBlock{}, fmt.Errorf("usecase content blocks get by id: %w", err)
	}

	return block, nil
}

func (u *ContentBlockUseCase) List(ctx context.Context, filter domain.ContentBlockListFilter) ([]domain.ContentBlock, int, error) {
	normalized, err := normalizeContentBlockListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase content blocks list: %w", err)
	}

	blocks, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase content blocks list: %w", err)
	}

	return blocks, total, nil
}

func (u *ContentBlockUseCase) Update(ctx context.Context, params domain.UpdateContentBlockParams) (domain.ContentBlock, error) {
	normalized, err := normalizeUpdateContentBlockParams(params)
	if err != nil {
		return domain.ContentBlock{}, fmt.Errorf("usecase content blocks update: %w", err)
	}

	block, err := u.repository.Update(ctx, normalized)
	if err != nil {
		return domain.ContentBlock{}, fmt.Errorf("usecase content blocks update: %w", err)
	}

	return block, nil
}

func (u *ContentBlockUseCase) Delete(ctx context.Context, blockID string) error {
	blockID = strings.TrimSpace(blockID)
	if blockID == "" {
		return fmt.Errorf("usecase content blocks delete: %w", domain.ErrValidation)
	}

	if err := u.repository.Delete(ctx, blockID); err != nil {
		return fmt.Errorf("usecase content blocks delete: %w", err)
	}

	return nil
}

func normalizeCreateContentBlockParams(params domain.CreateContentBlockParams) (domain.CreateContentBlockParams, error) {
	if params.CourseID != nil {
		params.CourseID = normalizeOptionalString(params.CourseID)
	}

	if params.ModuleID != nil {
		params.ModuleID = normalizeOptionalString(params.ModuleID)
	}

	if (params.CourseID == nil && params.ModuleID == nil) || (params.CourseID != nil && params.ModuleID != nil) {
		return domain.CreateContentBlockParams{}, fmt.Errorf("exactly one of course_id or module_id is required: %w", domain.ErrValidation)
	}

	if params.Position <= 0 {
		return domain.CreateContentBlockParams{}, fmt.Errorf("position must be greater than zero: %w", domain.ErrValidation)
	}

	if !params.Type.IsValid() {
		return domain.CreateContentBlockParams{}, fmt.Errorf("type is invalid: %w", domain.ErrValidation)
	}

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.CreateContentBlockParams{}, fmt.Errorf("title is invalid: %w", domain.ValidationError("title requires both ru and tj variants"))
	}

	payload, err := normalizeJSONRawMessage(params.Payload, false)
	if err != nil {
		return domain.CreateContentBlockParams{}, fmt.Errorf("payload is invalid: %w", err)
	}
	params.Payload = payload

	if err := domain.ValidateContentBlockPayload(params.Type, params.Payload); err != nil {
		return domain.CreateContentBlockParams{}, fmt.Errorf("payload is invalid: %w", err)
	}

	return params, nil
}

func normalizeUpdateContentBlockParams(params domain.UpdateContentBlockParams) (domain.UpdateContentBlockParams, error) {
	params.ID = strings.TrimSpace(params.ID)

	if params.CourseID != nil {
		params.CourseID = normalizeOptionalString(params.CourseID)
	}

	if params.ModuleID != nil {
		params.ModuleID = normalizeOptionalString(params.ModuleID)
	}

	if params.ID == "" {
		return domain.UpdateContentBlockParams{}, fmt.Errorf("id is required: %w", domain.ErrValidation)
	}

	if (params.CourseID == nil && params.ModuleID == nil) || (params.CourseID != nil && params.ModuleID != nil) {
		return domain.UpdateContentBlockParams{}, fmt.Errorf("exactly one of course_id or module_id is required: %w", domain.ErrValidation)
	}

	if params.Position <= 0 {
		return domain.UpdateContentBlockParams{}, fmt.Errorf("position must be greater than zero: %w", domain.ErrValidation)
	}

	if !params.Type.IsValid() {
		return domain.UpdateContentBlockParams{}, fmt.Errorf("type is invalid: %w", domain.ErrValidation)
	}

	if err := params.Title.ValidateRequired(); err != nil {
		return domain.UpdateContentBlockParams{}, fmt.Errorf("title is invalid: %w", domain.ValidationError("title requires both ru and tj variants"))
	}

	payload, err := normalizeJSONRawMessage(params.Payload, false)
	if err != nil {
		return domain.UpdateContentBlockParams{}, fmt.Errorf("payload is invalid: %w", err)
	}
	params.Payload = payload

	if err := domain.ValidateContentBlockPayload(params.Type, params.Payload); err != nil {
		return domain.UpdateContentBlockParams{}, fmt.Errorf("payload is invalid: %w", err)
	}

	return params, nil
}

func normalizeContentBlockListFilter(filter domain.ContentBlockListFilter) (domain.ContentBlockListFilter, error) {
	if filter.CourseID != nil {
		filter.CourseID = normalizeOptionalString(filter.CourseID)
	}

	if filter.ModuleID != nil {
		filter.ModuleID = normalizeOptionalString(filter.ModuleID)
	}

	if filter.CourseID != nil && filter.ModuleID != nil {
		return domain.ContentBlockListFilter{}, fmt.Errorf("only one of course_id or module_id is allowed: %w", domain.ErrValidation)
	}

	if filter.CourseID == nil && filter.ModuleID == nil {
		return domain.ContentBlockListFilter{}, fmt.Errorf("course_id or module_id is required: %w", domain.ErrValidation)
	}

	return filter, nil
}
