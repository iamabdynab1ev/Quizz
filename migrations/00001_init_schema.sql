-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =======================================================
-- 1. ТИПЫ ДАННЫХ (Enums)
-- =======================================================
CREATE TYPE user_role          AS ENUM ('admin', 'employee', 'student', 'guest');
CREATE TYPE gender             AS ENUM ('male', 'female', 'other', 'unspecified');
CREATE TYPE quiz_status        AS ENUM ('draft', 'published', 'archived');
CREATE TYPE course_status      AS ENUM ('draft', 'published', 'archived');
CREATE TYPE platform           AS ENUM ('web', 'mobile', 'telegram');
CREATE TYPE enrollment_status  AS ENUM ('active', 'completed', 'dropped');
CREATE TYPE review_status      AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE webhook_status     AS ENUM ('active', 'disabled');

CREATE TYPE question_type      AS ENUM (
    'single_choice','multiple_choice','true_false','short_answer','long_text',
    'matching','ordering','fill_blank','image_choice','audio','video','code'
);
CREATE TYPE content_block_type AS ENUM ('text','url','video','photo','file');
CREATE TYPE notification_type  AS ENUM (
    'course.published','certificate.issued','review.approved','enrollment.created','system'
);
CREATE TYPE app_event_type     AS ENUM (
    'course.created','course.updated','course.deleted','course.published',
    'test.created','test.updated','test.deleted',
    'user.created','user.updated','user.deleted',
    'enrollment.created','enrollment.completed',
    'attempt.finished','attempt.passed','attempt.failed',
    'certificate.issued','certificate.revoked',
    'review.created','review.approved','review.rejected'
);

-- =======================================================
-- 2. ПОЛЬЗОВАТЕЛИ
-- =======================================================
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    google_id     TEXT UNIQUE,          -- Интеграция с OAuth
    password_hash TEXT,                 -- Может быть NULL если зашел через Google
    role          user_role NOT NULL DEFAULT 'student',
    first_name    TEXT,
    last_name     TEXT,
    patronymic    TEXT,
    phone         TEXT,
    gender        gender DEFAULT 'unspecified',
    birth_date    DATE,
    address       TEXT,
    city          TEXT,                 -- Выбор города/района из макета
    avatar_url    TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT true, -- Soft-блокировка сотрудника
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_role ON users(role);

CREATE TABLE user_employee_info (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    branch      TEXT, office TEXT, position TEXT, department TEXT,
    employee_id TEXT, hire_date DATE, notes TEXT
);

CREATE TABLE user_student_info (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    student_id       TEXT,
    group_name       TEXT,
    education_level  TEXT,
    birth_date       DATE
);

CREATE TABLE user_guest_info (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    source      TEXT,
    invited_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address TEXT,          -- Секьюрность банка
    user_agent TEXT,          -- Устройство
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

-- =======================================================
-- 3. КУРСЫ (Мультиязычность JSONB)
-- =======================================================
CREATE TABLE courses (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                      JSONB NOT NULL,
    description                JSONB,
    cover_image_url            TEXT,
    category                   TEXT,
    status                     course_status NOT NULL DEFAULT 'draft',
    platforms                  platform[] NOT NULL DEFAULT '{}',
    estimated_minutes          INT,
    certificate_enabled        BOOLEAN NOT NULL DEFAULT false,
    certificate_passing_score  INT NOT NULL DEFAULT 0,
    reviews_enabled            BOOLEAN NOT NULL DEFAULT true,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_courses_status ON courses(status);

CREATE TABLE course_modules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    position    INT  NOT NULL,
    title       JSONB NOT NULL,
    description JSONB,
    UNIQUE (course_id, position)
);

CREATE TABLE content_blocks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   UUID REFERENCES courses(id) ON DELETE CASCADE,
    module_id   UUID REFERENCES course_modules(id) ON DELETE CASCADE,
    position    INT  NOT NULL,
    type        content_block_type NOT NULL,
    title       JSONB,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    CHECK ((course_id IS NOT NULL) <> (module_id IS NOT NULL))
);

-- =======================================================
-- 4. ТЕСТЫ И ВОПРОСЫ
-- =======================================================
CREATE TABLE quizzes (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title              JSONB NOT NULL,
    description        JSONB,
    category           TEXT,
    status             quiz_status NOT NULL DEFAULT 'draft',
    platforms          platform[] NOT NULL DEFAULT '{}',
    time_limit_minutes INT,
    passing_score      INT NOT NULL DEFAULT 80, -- Процент для успешной сдачи
    max_attempts       INT NOT NULL DEFAULT 3,  -- Требование по попыткам
    shuffle_questions  BOOLEAN NOT NULL DEFAULT false,
    show_results       BOOLEAN NOT NULL DEFAULT true,
    allow_retry        BOOLEAN NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE course_tests (
    course_id UUID REFERENCES courses(id)        ON DELETE CASCADE,
    module_id UUID REFERENCES course_modules(id) ON DELETE CASCADE,
    quiz_id   UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    position  INT  NOT NULL,
    CHECK ((course_id IS NOT NULL) <> (module_id IS NOT NULL))
);

CREATE TABLE questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id     UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    position    INT  NOT NULL,
    type        question_type NOT NULL,
    prompt      JSONB NOT NULL, -- "Вопрос"
    explanation JSONB,          -- "Подсказка после"
    points      NUMERIC(6,2) NOT NULL DEFAULT 1,
    required    BOOLEAN NOT NULL DEFAULT true,
    config      JSONB NOT NULL DEFAULT '{}'::jsonb, -- Ответы
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (quiz_id, position)
);

-- =======================================================
-- 5. РЕЗУЛЬТАТЫ СТУДЕНТА (Attempts)
-- =======================================================
CREATE TABLE enrollments (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id              UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id                UUID REFERENCES users(id) ON DELETE SET NULL,
    status                 enrollment_status NOT NULL DEFAULT 'active',
    enrolled_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at           TIMESTAMPTZ
);

CREATE TABLE enrollment_completed_blocks (
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    block_id      UUID NOT NULL REFERENCES content_blocks(id) ON DELETE CASCADE,
    completed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (enrollment_id, block_id)
);

CREATE TABLE attempts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id             UUID NOT NULL REFERENCES quizzes(id) ON DELETE RESTRICT,
    user_id             UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at         TIMESTAMPTZ,
    questions_snapshot  JSONB NOT NULL,
    answers_data        JSONB,          -- JSON с ответами 
    total_earned        NUMERIC(8,2) NOT NULL DEFAULT 0,
    total_max           NUMERIC(8,2) NOT NULL DEFAULT 0,
    score_percent       NUMERIC(5,2) NOT NULL DEFAULT 0,
    passed              BOOLEAN NOT NULL DEFAULT false,
    needs_review        BOOLEAN NOT NULL DEFAULT false
);

-- =======================================================
-- 6. СЕРТИФИКАТЫ (Важнейший банковский документ)
-- =======================================================
CREATE TABLE certificates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id  UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id      UUID NOT NULL REFERENCES courses(id),
    attempt_id     UUID NOT NULL REFERENCES attempts(id),
    serial_number  TEXT UNIQUE NOT NULL, -- Наш формат "123-456-789"
    verify_hash    TEXT UNIQUE NOT NULL, -- QR код
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    pdf_url        TEXT
);

-- =======================================================
-- 7. ЭКОСИСТЕМА (Вебхуки, логи, отзывы)
-- =======================================================
CREATE TABLE reviews (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id    UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    rating       SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    text         TEXT,
    status       review_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    moderated_at TIMESTAMPTZ
);

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       notification_type NOT NULL,
    title      JSONB NOT NULL,
    body       JSONB,
    link       TEXT,
    read       BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE webhooks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name              TEXT NOT NULL,
    url               TEXT NOT NULL,
    events            app_event_type[] NOT NULL DEFAULT '{}',
    secret            TEXT NOT NULL,
    status            webhook_status NOT NULL DEFAULT 'active',
    last_triggered_at TIMESTAMPTZ,
    last_status_code  INT,
    last_error        TEXT,
    deliveries        BIGINT NOT NULL DEFAULT 0,
    failures          BIGINT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       app_event_type NOT NULL,
    at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    payload    JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX idx_audit_at_desc ON audit_logs(at DESC);

-- +goose StatementEnd



-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;
DROP TABLE IF EXISTS certificates CASCADE;
DROP TABLE IF EXISTS attempts CASCADE;
DROP TABLE IF EXISTS enrollment_completed_blocks CASCADE;
DROP TABLE IF EXISTS enrollments CASCADE;
DROP TABLE IF EXISTS questions CASCADE;
DROP TABLE IF EXISTS course_tests CASCADE;
DROP TABLE IF EXISTS quizzes CASCADE;
DROP TABLE IF EXISTS content_blocks CASCADE;
DROP TABLE IF EXISTS course_modules CASCADE;
DROP TABLE IF EXISTS courses CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS user_guest_info CASCADE;
DROP TABLE IF EXISTS user_student_info CASCADE;
DROP TABLE IF EXISTS user_employee_info CASCADE;
DROP TABLE IF EXISTS users CASCADE;

DROP TYPE IF EXISTS app_event_type;
DROP TYPE IF EXISTS notification_type;
DROP TYPE IF EXISTS content_block_type;
DROP TYPE IF EXISTS question_type;
DROP TYPE IF EXISTS webhook_status;
DROP TYPE IF EXISTS review_status;
DROP TYPE IF EXISTS enrollment_status;
DROP TYPE IF EXISTS platform;
DROP TYPE IF EXISTS course_status;
DROP TYPE IF EXISTS quiz_status;
DROP TYPE IF EXISTS gender;
DROP TYPE IF EXISTS user_role;

-- +goose StatementEnd
