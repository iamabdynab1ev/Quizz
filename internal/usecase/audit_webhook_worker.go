package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lms-arvand-backend/internal/domain"
)

type auditWebhookOutboxRepository interface {
	ClaimWebhookDispatchBatch(ctx context.Context, limit int) ([]domain.AuditLog, error)
	MarkWebhookDispatchSuccess(ctx context.Context, auditLogID string) error
	MarkWebhookDispatchFailure(ctx context.Context, auditLogID string, nextAttemptAt time.Time, lastError string) error
}

type AuditWebhookOutboxWorker struct {
	logger     *slog.Logger
	repository auditWebhookOutboxRepository
	dispatcher auditEventDispatcher
	interval   time.Duration
	batchSize  int
}

func NewAuditWebhookOutboxWorker(logger *slog.Logger, repository auditWebhookOutboxRepository, dispatcher auditEventDispatcher, interval time.Duration, batchSize int) *AuditWebhookOutboxWorker {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 10
	}

	return &AuditWebhookOutboxWorker{
		logger:     logger,
		repository: repository,
		dispatcher: dispatcher,
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (w *AuditWebhookOutboxWorker) Run(ctx context.Context) {
	if w == nil || w.repository == nil || w.dispatcher == nil {
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.drainOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.drainOnce(ctx)
		}
	}
}

func (w *AuditWebhookOutboxWorker) drainOnce(ctx context.Context) {
	for {
		batch, err := w.repository.ClaimWebhookDispatchBatch(ctx, w.batchSize)
		if err != nil {
			w.logger.ErrorContext(ctx, "audit webhook outbox claim failed", slog.String("error", err.Error()))
			return
		}

		if len(batch) == 0 {
			return
		}

		for _, auditLog := range batch {
			w.dispatchOne(ctx, auditLog)
		}

		if len(batch) < w.batchSize {
			return
		}
	}
}

func (w *AuditWebhookOutboxWorker) dispatchOne(ctx context.Context, auditLog domain.AuditLog) {
	dispatchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if err := w.dispatcher.Dispatch(dispatchCtx, auditLog); err != nil {
		nextAttemptAt := time.Now().UTC().Add(dispatchBackoff(auditLog.WebhookDispatchAttempts))
		if markErr := w.repository.MarkWebhookDispatchFailure(ctx, auditLog.ID, nextAttemptAt, err.Error()); markErr != nil {
			w.logger.ErrorContext(ctx, "audit webhook outbox mark failure failed",
				slog.String("audit_log_id", auditLog.ID),
				slog.String("error", markErr.Error()),
			)
		}

		w.logger.ErrorContext(ctx, "audit webhook outbox dispatch failed",
			slog.String("audit_log_id", auditLog.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	if err := w.repository.MarkWebhookDispatchSuccess(ctx, auditLog.ID); err != nil {
		w.logger.ErrorContext(ctx, "audit webhook outbox mark success failed",
			slog.String("audit_log_id", auditLog.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	w.logger.InfoContext(ctx, "audit webhook outbox dispatched",
		slog.String("audit_log_id", auditLog.ID),
		slog.String("event_type", string(auditLog.Type)),
	)
}

func dispatchBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}

	return time.Duration(1<<(attempts-1)) * time.Second
}

func (w *AuditWebhookOutboxWorker) String() string {
	return fmt.Sprintf("AuditWebhookOutboxWorker{interval=%s,batchSize=%d}", w.interval, w.batchSize)
}
