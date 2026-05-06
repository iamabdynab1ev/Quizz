-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_super_admin BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX IF NOT EXISTS users_single_super_admin_idx
    ON users(is_super_admin)
    WHERE is_super_admin = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS users_single_super_admin_idx;

ALTER TABLE users
    DROP COLUMN IF EXISTS is_super_admin;
-- +goose StatementEnd
