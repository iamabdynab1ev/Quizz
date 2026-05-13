-- +goose Up
-- +goose StatementBegin

-- Course certificates are part of the current product flow. Older course updates
-- could accidentally store certificate_enabled=false because the update DTO used
-- a non-pointer boolean and missing JSON fields decoded to false.
ALTER TABLE courses
    ALTER COLUMN certificate_enabled SET DEFAULT true;

UPDATE courses
SET certificate_enabled = true
WHERE certificate_enabled = false;

WITH best_attempts AS (
    SELECT DISTINCT ON (e.id)
        e.id AS enrollment_id,
        e.user_id,
        e.course_id,
        a.id AS attempt_id
    FROM enrollments e
    JOIN attempts a
        ON a.course_id = e.course_id
       AND a.user_id = e.user_id
       AND a.passed = true
    WHERE e.status IN ('active', 'completed')
      AND NOT EXISTS (
          SELECT 1
          FROM certificates cert
          WHERE cert.enrollment_id = e.id
      )
    ORDER BY e.id, a.score_percent DESC, a.finished_at DESC NULLS LAST, a.started_at DESC
),
numbered AS (
    SELECT
        best_attempts.*,
        ROW_NUMBER() OVER (ORDER BY best_attempts.enrollment_id)
            + (SELECT COUNT(*) FROM certificates) AS serial_seq
    FROM best_attempts
)
INSERT INTO certificates (
    enrollment_id,
    user_id,
    course_id,
    attempt_id,
    serial_number,
    verify_hash
)
SELECT
    enrollment_id,
    user_id,
    course_id,
    attempt_id,
    CONCAT(
        LPAD((((900000000 + serial_seq) / 1000000) % 1000)::text, 3, '0'),
        '-',
        LPAD((((900000000 + serial_seq) / 1000) % 1000)::text, 3, '0'),
        '-',
        LPAD(((900000000 + serial_seq) % 1000)::text, 3, '0')
    ),
    encode(gen_random_bytes(32), 'hex')
FROM numbered
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Data-only repair; do not delete issued certificates on rollback.
SELECT 1;

-- +goose StatementEnd
