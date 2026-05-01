-- +goose Up
-- +goose StatementBegin

ALTER TABLE attempts
    ADD COLUMN IF NOT EXISTS review_scores JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE attempts
SET review_scores = '[]'::jsonb
WHERE review_scores IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE attempts
    DROP COLUMN IF EXISTS review_scores;

-- +goose StatementEnd

