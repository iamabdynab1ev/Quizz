-- +goose Up
-- +goose StatementBegin

ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS video_url TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE courses
    DROP COLUMN IF EXISTS video_url;

-- +goose StatementEnd
