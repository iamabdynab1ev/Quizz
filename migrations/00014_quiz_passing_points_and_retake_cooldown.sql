-- +goose Up
-- +goose StatementBegin

ALTER TABLE quizzes
    ADD COLUMN IF NOT EXISTS passing_points NUMERIC(8,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS retake_cooldown_days INT NOT NULL DEFAULT 30;

CREATE INDEX IF NOT EXISTS idx_attempts_quiz_user_started_at
    ON attempts (quiz_id, user_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_certificates_course_user
    ON certificates (course_id, user_id);

UPDATE quizzes q
SET passing_points = ROUND((COALESCE(points.total_points, 0) * q.passing_score / 100.0)::numeric, 2)
FROM (
    SELECT quiz_id, SUM(points)::numeric AS total_points
    FROM questions
    GROUP BY quiz_id
) points
WHERE points.quiz_id = q.id
  AND q.passing_points = 0
  AND q.passing_score > 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE quizzes
    DROP COLUMN IF EXISTS retake_cooldown_days,
    DROP COLUMN IF EXISTS passing_points;

DROP INDEX IF EXISTS idx_certificates_course_user;
DROP INDEX IF EXISTS idx_attempts_quiz_user_started_at;

-- +goose StatementEnd
