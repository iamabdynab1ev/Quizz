-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT false;

UPDATE users
SET must_change_password = false
WHERE is_super_admin = true;

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS must_change_password;
