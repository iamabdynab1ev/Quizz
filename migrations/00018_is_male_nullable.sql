-- +goose Up
ALTER TABLE users ALTER COLUMN is_male DROP NOT NULL;
ALTER TABLE users ALTER COLUMN is_male SET DEFAULT NULL;

-- +goose Down
UPDATE users SET is_male = true WHERE is_male IS NULL;
ALTER TABLE users ALTER COLUMN is_male SET NOT NULL;
ALTER TABLE users ALTER COLUMN is_male SET DEFAULT true;
