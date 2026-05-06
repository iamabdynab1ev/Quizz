-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS birth_date DATE;

UPDATE users u
SET birth_date = stu.birth_date
FROM user_student_info stu
WHERE stu.user_id = u.id
  AND u.birth_date IS NULL
  AND stu.birth_date IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
    DROP COLUMN IF EXISTS birth_date;
-- +goose StatementEnd
