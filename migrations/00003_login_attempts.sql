-- +goose Up
-- +goose StatementBegin

CREATE TABLE login_attempts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier   TEXT NOT NULL,
    ip_address   TEXT,
    succeeded    BOOLEAN NOT NULL DEFAULT false,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_login_attempts_identifier_time
    ON login_attempts (identifier, attempted_at DESC);

CREATE INDEX idx_login_attempts_ip_time
    ON login_attempts (ip_address, attempted_at DESC)
    WHERE ip_address IS NOT NULL;

CREATE INDEX idx_login_attempts_failed_identifier_time
    ON login_attempts (identifier, attempted_at DESC)
    WHERE succeeded = false;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS login_attempts;

-- +goose StatementEnd
