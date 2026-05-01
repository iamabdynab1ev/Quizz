-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS users_google_id_unique_idx
ON users (google_id)
WHERE google_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS users_google_id_unique_idx;
