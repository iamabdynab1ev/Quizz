-- +goose Up
-- +goose StatementBegin
ALTER TABLE courses
    ALTER COLUMN certificate_enabled SET DEFAULT true;

UPDATE courses
SET certificate_enabled = true
WHERE certificate_enabled = false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE courses
    ALTER COLUMN certificate_enabled SET DEFAULT false;
-- +goose StatementEnd
