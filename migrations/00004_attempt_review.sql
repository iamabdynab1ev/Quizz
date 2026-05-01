-- +goose Up
-- +goose StatementBegin

ALTER TABLE attempts
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS review_comment TEXT,
    ADD COLUMN IF NOT EXISTS manual_passed BOOLEAN;

CREATE INDEX IF NOT EXISTS idx_attempts_needs_review
    ON attempts (needs_review)
    WHERE needs_review = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_attempts_needs_review;

ALTER TABLE attempts
    DROP COLUMN IF EXISTS manual_passed,
    DROP COLUMN IF EXISTS review_comment,
    DROP COLUMN IF EXISTS reviewer_id,
    DROP COLUMN IF EXISTS reviewed_at;

-- +goose StatementEnd
