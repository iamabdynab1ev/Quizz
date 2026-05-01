-- +goose Up
-- +goose StatementBegin

ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS webhook_processing_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS webhook_dispatched_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS webhook_dispatch_attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS webhook_next_dispatch_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS webhook_last_dispatch_error TEXT;

CREATE INDEX IF NOT EXISTS idx_audit_logs_webhook_pending
    ON audit_logs (webhook_next_dispatch_at, at)
    WHERE webhook_dispatched_at IS NULL;

-- +goose StatementEnd



-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_audit_logs_webhook_pending;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS webhook_last_dispatch_error,
    DROP COLUMN IF EXISTS webhook_next_dispatch_at,
    DROP COLUMN IF EXISTS webhook_dispatch_attempts,
    DROP COLUMN IF EXISTS webhook_dispatched_at,
    DROP COLUMN IF EXISTS webhook_processing_at;

-- +goose StatementEnd
