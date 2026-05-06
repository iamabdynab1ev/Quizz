# QUIZ Project Setup

Практическая инструкция по запуску backend локально и на тестовом сервере.

## Что нужно

- Go `1.24+`
- PostgreSQL
- `.env`
- доступная база `lms_arvand`

Не нужно для старта:

- Redis
- RabbitMQ/Kafka
- S3/MinIO
- Docker, если PostgreSQL уже установлен локально

## Быстрый запуск

```powershell
cd C:\projects\go\src\myApps\quiz
go run ./cmd/api
```

Если backend стартовал, в логах будет:

```text
HTTP сервер запускается
address=127.0.0.1:9000
```

Проверка:

```powershell
Invoke-RestMethod http://127.0.0.1:9000/health
Invoke-RestMethod http://127.0.0.1:9000/api/v1/health
```

## Сборка

Windows:

```powershell
go build -o bin/api.exe ./cmd/api
```

Linux:

```bash
go build -o bin/api ./cmd/api
```

## Проверка качества

```powershell
go test ./...
go vet ./...
```

## Как backend ищет `.env`

При запуске `cmd/api` загружает `.env` из нескольких мест:

1. путь из `ENV_FILE`, если переменная задана;
2. `.env` рядом с exe;
3. `.env` в родительской папке exe;
4. `.env` в текущей рабочей директории.

Для разработки проще держать `.env` в корне проекта:

```text
C:\projects\go\src\myApps\quiz\.env
```

## Минимальный `.env`

```env
APP_NAME=QUIZ
APP_ENV=development
LOG_LEVEL=INFO

HTTP_ADDRESS=127.0.0.1:9000
HTTP_CORS_ALLOWED_ORIGINS=http://localhost:4041
HTTP_READ_TIMEOUT=5s
HTTP_READ_HEADER_TIMEOUT=2s
HTTP_WRITE_TIMEOUT=15s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=15s

AUTH_SESSION_TTL=8h
AUTH_SESSION_CACHE_TTL=5m
AUTH_BCRYPT_COST=12
AUTH_LOGIN_LOCKOUT_ENABLED=false
AUTH_LOGIN_MAX_ATTEMPTS=5
AUTH_LOGIN_ATTEMPT_WINDOW=15m
AUTH_LOGIN_LOCKOUT_SCOPE=identifier_ip
AUTH_PASSWORD_RESET_TOKEN_TTL=15m
AUTH_PASSWORD_RESET_RETURN_TOKEN=true

GOOGLE_CLIENT_ID=
AUTH_GOOGLE_DEFAULT_ROLE=student

UPLOADS_DIR=uploads
UPLOAD_MAX_SIZE_MB=20

DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5433/lms_arvand?sslmode=disable
PGX_MAX_CONNS=20
PGX_MIN_CONNS=2
PGX_MAX_CONN_LIFETIME=30m
PGX_MAX_CONN_IDLE_TIME=5m
PGX_HEALTH_CHECK_PERIOD=1m

MIGRATE_RUN_ON_START=true
MIGRATIONS_DIR=migrations

SEED_RUN_ON_START=true
SEED_ADMIN_EMAIL=admin@local.test
SEED_ADMIN_PASSWORD=Admin123!
SEED_ADMIN_FIRST_NAME=System
SEED_ADMIN_LAST_NAME=Admin
SEED_ADMIN_PATRONYMIC=
```

## Важные настройки

`HTTP_ADDRESS`

- `127.0.0.1:9000` - backend доступен только локально.
- `0.0.0.0:9000` - backend доступен из сети, обычно лучше не делать без proxy.

`HTTP_CORS_ALLOWED_ORIGINS`

- для Vite: `http://localhost:4041`;
- можно несколько через запятую.

`DATABASE_URL`

Формат:

```text
postgres://USER:PASSWORD@HOST:PORT/DB?sslmode=disable
```

Пример для Docker/Postgres на порту `5433`:

```env
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5433/lms_arvand?sslmode=disable
```

Пример для обычного PostgreSQL на порту `5432`:

```env
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/lms_arvand?sslmode=disable
```

`AUTH_LOGIN_LOCKOUT_ENABLED`

- `false` удобно локально, чтобы не заблокировать себя при тестах;
- `true` лучше на сервере.

`AUTH_PASSWORD_RESET_RETURN_TOKEN`

- `true` только локально и для Postman;
- `false` на production, потому что token должен доставляться через безопасный канал.

`GOOGLE_CLIENT_ID`

- OAuth Web Client ID из Google Cloud Console;
- если пустой, Google login можно скрыть на frontend.

## PostgreSQL

Через pgAdmin нужно проверить:

- host: `127.0.0.1`
- port: `5432` или `5433`
- user: `postgres`
- database: `lms_arvand`
- password совпадает с `DATABASE_URL`

Создать базу вручную:

```sql
CREATE DATABASE lms_arvand;
```

Миграции создадут таблицы автоматически при `MIGRATE_RUN_ON_START=true`.

## Bootstrap admin

Первый пользователь создаётся из `.env`:

```env
SEED_ADMIN_EMAIL=admin@local.test
SEED_ADMIN_PASSWORD=Admin123!
```

Он является единственным `is_super_admin=true`.

Первый вход во frontend:

```json
{
  "email": "admin@local.test",
  "password": "Admin123!"
}
```

## Frontend подключение

Если backend отдельно:

```env
VITE_API_URL=http://localhost:9000/api/v1
```

Если всё через nginx на одном порту:

```env
VITE_API_URL=/api/v1
```

Backend CORS должен разрешать origin frontend:

```env
HTTP_CORS_ALLOWED_ORIGINS=http://localhost:4041
```

## Типовые ошибки

`DATABASE_URL is empty`

- `.env` не найден;
- файл назван `.env.txt`;
- запускается exe из другой папки.

`password authentication failed for user "postgres"`

- пароль в `DATABASE_URL` не совпадает с PostgreSQL.

`connection refused`

- PostgreSQL не запущен;
- указан неправильный port.

`bind: Only one usage of each socket address`

- порт backend уже занят;
- нужно закрыть старый `api.exe` или поменять `HTTP_ADDRESS`.

`too many login attempts`

- включён login lockout;
- для локального теста поставить `AUTH_LOGIN_LOCKOUT_ENABLED=false`;
- перезапустить backend.

## Порядок чистого запуска

1. Запустить PostgreSQL.
2. Проверить host/port/user/password/database.
3. Проверить `.env`.
4. Запустить backend.
5. Проверить `/health`.
6. Войти через `SEED_ADMIN_EMAIL`.
7. Запустить frontend.
8. Проверить курсы, тесты, попытки, сертификаты.
