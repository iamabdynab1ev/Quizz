-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_enrollments_user_id   ON enrollments(user_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_course_id ON enrollments(course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_course_user ON enrollments(course_id, user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, read) WHERE read = false;

CREATE INDEX IF NOT EXISTS idx_reviews_course_id ON reviews(course_id);
CREATE INDEX IF NOT EXISTS idx_reviews_user_id   ON reviews(user_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_type ON audit_logs(type, at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_enrollments_user_id;
DROP INDEX IF EXISTS idx_enrollments_course_id;
DROP INDEX IF EXISTS idx_enrollments_course_user;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_notifications_user_read;
DROP INDEX IF EXISTS idx_reviews_course_id;
DROP INDEX IF EXISTS idx_reviews_user_id;
DROP INDEX IF EXISTS idx_audit_logs_type;

-- +goose StatementEnd
