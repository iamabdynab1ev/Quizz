package usecase

import (
	"context"
	"encoding/json"
	"log/slog"

	"lms-arvand-backend/internal/appctx"
	"lms-arvand-backend/internal/domain"
)

type auditLogWriter interface {
	Create(ctx context.Context, params domain.CreateAuditLogParams) (domain.AuditLog, error)
}

type AuditLogger struct {
	logger *slog.Logger
	writer auditLogWriter
}

type auditEventDispatcher interface {
	Dispatch(ctx context.Context, auditLog domain.AuditLog) error
}

func NewAuditLogger(logger *slog.Logger, writer auditLogWriter) *AuditLogger {
	if logger == nil {
		logger = slog.Default()
	}

	return &AuditLogger{
		logger: logger,
		writer: writer,
	}
}

func (a *AuditLogger) Log(ctx context.Context, eventType domain.AppEventType, payload any) {
	if a == nil || a.writer == nil {
		return
	}

	var actorID *string
	if identity, ok := appctx.CurrentAuthIdentity(ctx); ok {
		actorID = &identity.User.ID
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		a.logger.ErrorContext(ctx, "audit log marshal payload failed", slog.String("event_type", string(eventType)), slog.String("error", err.Error()))
		return
	}

	_, createErr := a.writer.Create(ctx, domain.CreateAuditLogParams{
		Type:    eventType,
		ActorID: actorID,
		Payload: payloadBytes,
	})
	if createErr != nil {
		a.logger.ErrorContext(ctx, "audit log write failed", slog.String("event_type", string(eventType)), slog.String("error", createErr.Error()))
		return
	}
}
