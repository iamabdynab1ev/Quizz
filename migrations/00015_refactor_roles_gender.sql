-- +goose Up
ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT false;
UPDATE users SET is_admin = true WHERE role = 'admin';

ALTER TABLE users ADD COLUMN is_male BOOLEAN NOT NULL DEFAULT true;
UPDATE users SET is_male = (gender = 'male');

DROP TABLE IF EXISTS user_guest_info;
DROP TABLE IF EXISTS user_student_info;
DROP TABLE IF EXISTS user_employee_info;

ALTER TABLE users DROP COLUMN role;
ALTER TABLE users DROP COLUMN gender;

DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS gender;

-- +goose Down
CREATE TYPE user_role AS ENUM ('admin', 'employee', 'student', 'guest');
CREATE TYPE gender AS ENUM ('male', 'female', 'other', 'unspecified');

ALTER TABLE users ADD COLUMN role user_role NOT NULL DEFAULT 'student';
ALTER TABLE users ADD COLUMN gender gender NOT NULL DEFAULT 'unspecified';

UPDATE users SET role = CASE WHEN is_admin THEN 'admin'::user_role ELSE 'student'::user_role END;
UPDATE users SET gender = CASE WHEN is_male THEN 'male'::gender ELSE 'female'::gender END;

CREATE TABLE user_employee_info (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    branch TEXT,
    office TEXT,
    position TEXT,
    department TEXT,
    employee_id TEXT,
    hire_date DATE,
    notes TEXT
);

CREATE TABLE user_student_info (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    student_id TEXT,
    group_name TEXT,
    education_level TEXT,
    birth_date DATE
);

CREATE TABLE user_guest_info (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    source TEXT,
    invited_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ
);

ALTER TABLE users DROP COLUMN is_admin;
ALTER TABLE users DROP COLUMN is_male;
