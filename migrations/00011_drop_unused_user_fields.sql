-- +goose Up
-- +goose StatementBegin
UPDATE users
SET email = username || '@local.invalid'
WHERE email IS NULL
  AND NULLIF(username, '') IS NOT NULL;

UPDATE users
SET email = id::text || '@local.invalid'
WHERE email IS NULL;

ALTER TABLE users
    ALTER COLUMN email SET NOT NULL;

DROP TABLE IF EXISTS user_admin_info CASCADE;

ALTER TABLE users
    DROP COLUMN IF EXISTS username;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- This cleanup is intentionally irreversible: removed fields are not used by the application.
SELECT 1;
-- +goose StatementEnd
