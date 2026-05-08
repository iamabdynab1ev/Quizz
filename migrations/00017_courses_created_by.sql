-- +goose Up
ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS created_by_name    TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE courses
    DROP COLUMN IF EXISTS created_by_name,
    DROP COLUMN IF EXISTS created_by_user_id;
