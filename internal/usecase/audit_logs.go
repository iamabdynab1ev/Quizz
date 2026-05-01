package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type auditLogRepository interface {
	Create(ctx context.Context, params domain.CreateAuditLogParams) (domain.AuditLog, error)
	GetByID(ctx context.Context, auditLogID string) (domain.AuditLog, error)
	List(ctx context.Context, filter domain.AuditLogListFilter) ([]domain.AuditLog, int, error)
}

type AuditLogUseCase struct {
	repository auditLogRepository
}

func NewAuditLogUseCase(repository auditLogRepository) *AuditLogUseCase {
	return &AuditLogUseCase{repository: repository}
}

func (u *AuditLogUseCase) Create(ctx context.Context, params domain.CreateAuditLogParams) (domain.AuditLog, error) {
	normalized, err := normalizeCreateAuditLogParams(params)
	if err != nil {
		return domain.AuditLog{}, fmt.Errorf("usecase audit logs create: %w", err)
	}

	auditLog, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.AuditLog{}, fmt.Errorf("usecase audit logs create: %w", err)
	}

	return auditLog, nil
}

func (u *AuditLogUseCase) GetByID(ctx context.Context, auditLogID string) (domain.AuditLog, error) {
	auditLogID = strings.TrimSpace(auditLogID)
	if auditLogID == "" {
		return domain.AuditLog{}, fmt.Errorf("usecase audit logs get by id: %w", domain.ErrValidation)
	}

	auditLog, err := u.repository.GetByID(ctx, auditLogID)
	if err != nil {
		return domain.AuditLog{}, fmt.Errorf("usecase audit logs get by id: %w", err)
	}

	return auditLog, nil
}

func (u *AuditLogUseCase) List(ctx context.Context, filter domain.AuditLogListFilter) ([]domain.AuditLog, int, error) {
	normalized, err := normalizeAuditLogListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase audit logs list: %w", err)
	}

	auditLogs, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase audit logs list: %w", err)
	}

	return auditLogs, total, nil
}

func normalizeCreateAuditLogParams(params domain.CreateAuditLogParams) (domain.CreateAuditLogParams, error) {
	params.ActorID = normalizeOptionalString(params.ActorID)

	if !params.Type.IsValid() {
		return domain.CreateAuditLogParams{}, fmt.Errorf("type is invalid: %w", domain.ErrValidation)
	}

	payload, err := normalizeJSONRawMessage(params.Payload, false)
	if err != nil {
		return domain.CreateAuditLogParams{}, fmt.Errorf("payload is invalid: %w", err)
	}
	params.Payload = payload

	return params, nil
}

func normalizeAuditLogListFilter(filter domain.AuditLogListFilter) (domain.AuditLogListFilter, error) {
	if filter.Type != nil {
		eventType := domain.AppEventType(strings.TrimSpace(string(*filter.Type)))
		if !eventType.IsValid() {
			return domain.AuditLogListFilter{}, fmt.Errorf("type is invalid: %w", domain.ErrValidation)
		}
		filter.Type = &eventType
	}

	if filter.ActorID != nil {
		filter.ActorID = normalizeOptionalString(filter.ActorID)
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		return domain.AuditLogListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}
