-- +goose Up
-- +goose StatementBegin

ALTER TABLE course_tests
    ADD COLUMN IF NOT EXISTS id UUID;

UPDATE course_tests
SET id = COALESCE(id, gen_random_uuid())
WHERE id IS NULL;

ALTER TABLE course_tests
    ALTER COLUMN id SET DEFAULT gen_random_uuid(),
    ALTER COLUMN id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_course_tests_id
    ON course_tests(id);

-- +goose StatementEnd



-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_course_tests_id;

ALTER TABLE course_tests
    DROP COLUMN IF EXISTS id;

-- +goose StatementEnd
