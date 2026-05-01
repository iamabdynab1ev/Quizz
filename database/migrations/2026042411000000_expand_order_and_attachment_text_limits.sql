-- +goose Up
-- +goose StatementBegin
SELECT 'up: expanding order title and attachment text limits';

ALTER TABLE public.orders
    ALTER COLUMN name TYPE VARCHAR(500);

ALTER TABLE public.attachments
    ALTER COLUMN file_name TYPE VARCHAR(500),
    ALTER COLUMN file_path TYPE TEXT,
    ALTER COLUMN file_type TYPE VARCHAR(100);

ALTER TABLE public.order_history
    ALTER COLUMN file_name TYPE VARCHAR(500),
    ALTER COLUMN file_path TYPE TEXT,
    ALTER COLUMN file_type TYPE VARCHAR(100);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down: restoring previous order title and attachment text limits';

ALTER TABLE public.order_history
    ALTER COLUMN file_type TYPE VARCHAR(50),
    ALTER COLUMN file_path TYPE VARCHAR(255),
    ALTER COLUMN file_name TYPE VARCHAR(255);

ALTER TABLE public.attachments
    ALTER COLUMN file_type TYPE VARCHAR(50),
    ALTER COLUMN file_path TYPE VARCHAR(255),
    ALTER COLUMN file_name TYPE VARCHAR(255);

ALTER TABLE public.orders
    ALTER COLUMN name TYPE VARCHAR(255);

-- +goose StatementEnd
