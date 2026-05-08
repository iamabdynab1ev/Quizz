-- +goose Up
-- +goose StatementBegin

-- 1. Add quiz settings columns to courses
ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS quiz_pass_percent   INT NOT NULL DEFAULT 80,
    ADD COLUMN IF NOT EXISTS quiz_minutes        INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_attempts        INT NOT NULL DEFAULT 3,
    ADD COLUMN IF NOT EXISTS retake_cooldown_days INT NOT NULL DEFAULT 30;

-- 2. Copy quiz settings from quizzes into courses via course_tests
UPDATE courses c
SET
    quiz_pass_percent    = q.passing_score,
    quiz_minutes         = COALESCE(q.time_limit_minutes, 0),
    max_attempts         = q.max_attempts,
    retake_cooldown_days = q.retake_cooldown_days
FROM course_tests ct
JOIN quizzes q ON q.id = ct.quiz_id
WHERE ct.course_id = c.id;

-- 3. Add course_id to questions (nullable for migration)
ALTER TABLE questions ADD COLUMN IF NOT EXISTS course_id UUID REFERENCES courses(id) ON DELETE CASCADE;

UPDATE questions q
SET course_id = ct.course_id
FROM course_tests ct
WHERE ct.quiz_id = q.quiz_id;

-- Replace unique constraint to use course_id
ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_quiz_id_position_key;
ALTER TABLE questions ADD CONSTRAINT questions_course_id_position_key UNIQUE (course_id, position);

-- Remove questions not linked to any course (orphaned)
DELETE FROM questions WHERE course_id IS NULL;

ALTER TABLE questions
    DROP COLUMN IF EXISTS quiz_id CASCADE,
    ALTER COLUMN course_id SET NOT NULL;

-- 4. Add course_id to attempts (nullable for migration)
ALTER TABLE attempts ADD COLUMN IF NOT EXISTS course_id UUID REFERENCES courses(id) ON DELETE RESTRICT;

UPDATE attempts a
SET course_id = ct.course_id
FROM course_tests ct
WHERE ct.quiz_id = a.quiz_id;

-- Remove certificates for orphaned attempts before deleting them
DELETE FROM certificates
WHERE attempt_id IN (SELECT id FROM attempts WHERE course_id IS NULL);

DELETE FROM attempts WHERE course_id IS NULL;

DROP INDEX IF EXISTS idx_attempts_quiz_user_started_at;
CREATE INDEX idx_attempts_course_user_started_at ON attempts (course_id, user_id, started_at DESC);

ALTER TABLE attempts
    DROP COLUMN IF EXISTS quiz_id CASCADE,
    DROP COLUMN IF EXISTS needs_review,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewer_id,
    DROP COLUMN IF EXISTS review_comment,
    DROP COLUMN IF EXISTS manual_passed,
    DROP COLUMN IF EXISTS review_scores,
    ALTER COLUMN course_id SET NOT NULL;

-- 5. Drop the now-unused join table and quizzes table
DROP TABLE IF EXISTS course_tests CASCADE;
DROP TABLE IF EXISTS quizzes CASCADE;
DROP TYPE IF EXISTS quiz_status;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

CREATE TYPE IF NOT EXISTS quiz_status AS ENUM ('draft', 'published', 'archived');

CREATE TABLE IF NOT EXISTS quizzes (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                JSONB NOT NULL,
    description          JSONB,
    category             TEXT,
    status               quiz_status NOT NULL DEFAULT 'draft',
    platforms            platform[] NOT NULL DEFAULT '{}',
    time_limit_minutes   INT,
    passing_score        INT NOT NULL DEFAULT 80,
    passing_points       NUMERIC(8,2) NOT NULL DEFAULT 0,
    max_attempts         INT NOT NULL DEFAULT 3,
    retake_cooldown_days INT NOT NULL DEFAULT 30,
    shuffle_questions    BOOLEAN NOT NULL DEFAULT false,
    show_results         BOOLEAN NOT NULL DEFAULT true,
    allow_retry          BOOLEAN NOT NULL DEFAULT true,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS course_tests (
    course_id UUID REFERENCES courses(id) ON DELETE CASCADE,
    module_id UUID REFERENCES course_modules(id) ON DELETE CASCADE,
    quiz_id   UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    position  INT NOT NULL,
    CHECK ((course_id IS NOT NULL) <> (module_id IS NOT NULL))
);

ALTER TABLE questions
    ADD COLUMN IF NOT EXISTS quiz_id UUID REFERENCES quizzes(id) ON DELETE CASCADE,
    DROP CONSTRAINT IF EXISTS questions_course_id_position_key,
    DROP COLUMN IF EXISTS course_id;

ALTER TABLE attempts
    ADD COLUMN IF NOT EXISTS quiz_id       UUID,
    ADD COLUMN IF NOT EXISTS needs_review  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS reviewed_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewer_id   UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS review_comment TEXT,
    ADD COLUMN IF NOT EXISTS manual_passed  BOOLEAN,
    ADD COLUMN IF NOT EXISTS review_scores  JSONB,
    DROP COLUMN IF EXISTS course_id;

DROP INDEX IF EXISTS idx_attempts_course_user_started_at;

ALTER TABLE courses
    DROP COLUMN IF EXISTS quiz_pass_percent,
    DROP COLUMN IF EXISTS quiz_minutes,
    DROP COLUMN IF EXISTS max_attempts,
    DROP COLUMN IF EXISTS retake_cooldown_days;

-- +goose StatementEnd
