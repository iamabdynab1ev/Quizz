-- +goose Up
-- +goose StatementBegin

-- Allow courses to be fully deleted even when attempts/certificates exist
ALTER TABLE attempts
    DROP CONSTRAINT IF EXISTS attempts_course_id_fkey,
    ADD CONSTRAINT attempts_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE;

ALTER TABLE certificates
    DROP CONSTRAINT IF EXISTS certificates_course_id_fkey,
    ADD CONSTRAINT certificates_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE;

ALTER TABLE certificates
    DROP CONSTRAINT IF EXISTS certificates_attempt_id_fkey,
    ADD CONSTRAINT certificates_attempt_id_fkey
        FOREIGN KEY (attempt_id) REFERENCES attempts(id) ON DELETE CASCADE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE certificates
    DROP CONSTRAINT IF EXISTS certificates_attempt_id_fkey,
    ADD CONSTRAINT certificates_attempt_id_fkey
        FOREIGN KEY (attempt_id) REFERENCES attempts(id);

ALTER TABLE certificates
    DROP CONSTRAINT IF EXISTS certificates_course_id_fkey,
    ADD CONSTRAINT certificates_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id);

ALTER TABLE attempts
    DROP CONSTRAINT IF EXISTS attempts_course_id_fkey,
    ADD CONSTRAINT attempts_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE RESTRICT;

-- +goose StatementEnd
