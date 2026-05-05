# Project Setup and Configuration

This file is the operational setup guide for the `quiz` repository.
Use it when you need to run the backend locally, on Windows, or behind a reverse proxy.

## 1. What the backend needs

Required:

- Go 1.24+
- PostgreSQL
- `.env` file or environment variables

Used by the backend:

- PostgreSQL for the main data store
- in-memory cache for auth/session and short-lived runtime state
- local disk for uploads
- Google token verification for Google login

The backend does **not** require Redis, a message broker, S3, or MinIO to start.

## 2. How configuration is loaded

Startup uses `cmd/api/main.go`.

Environment files are loaded in this order:

1. `ENV_FILE` if it is set
2. `.env` in the current working directory
3. `.env` next to the compiled exe
4. `.env` in the parent directory of the exe

`.env` values override existing process environment variables, so the local project file is the default source of truth during development.

That means you can run the binary from a build folder as long as one of those locations contains `.env`.

## 3. Main env variables

The backend reads these groups of variables from `internal/config/config.go` and `.env.example`.

### App and HTTP

- `APP_NAME`
- `APP_ENV`
- `LOG_LEVEL`
- `HTTP_ADDRESS`
- `HTTP_CORS_ALLOWED_ORIGINS`
- `HTTP_READ_TIMEOUT`
- `HTTP_READ_HEADER_TIMEOUT`
- `HTTP_WRITE_TIMEOUT`
- `HTTP_IDLE_TIMEOUT`
- `HTTP_SHUTDOWN_TIMEOUT`

### Auth

- `AUTH_SESSION_TTL`
- `AUTH_SESSION_CACHE_TTL`
- `AUTH_BCRYPT_COST`
- `AUTH_LOGIN_MAX_ATTEMPTS`
- `AUTH_LOGIN_ATTEMPT_WINDOW`

### Google login

- `GOOGLE_CLIENT_ID`

### Uploads

- `UPLOADS_DIR`
- `UPLOAD_MAX_SIZE_MB`

### Database

- `DATABASE_URL`
- `PGX_MAX_CONNS`
- `PGX_MIN_CONNS`
- `PGX_MAX_CONN_LIFETIME`
- `PGX_MAX_CONN_IDLE_TIME`
- `PGX_HEALTH_CHECK_PERIOD`

### Migrations

- `MIGRATE_RUN_ON_START`
- `MIGRATIONS_DIR`

### Seed admin

- `SEED_RUN_ON_START`
- `SEED_ADMIN_USERNAME`
- `SEED_ADMIN_EMAIL`
- `SEED_ADMIN_PASSWORD`
- `SEED_ADMIN_FIRST_NAME`
- `SEED_ADMIN_LAST_NAME`
- `SEED_ADMIN_PATRONYMIC`
- `SEED_ADMIN_IS_SUPER_ADMIN`
- `SEED_ADMIN_PERMISSIONS`

## 4. Minimal `.env.example`

Use `/.env.example` as the template. A typical local setup looks like this:

```env
APP_NAME=QUIZ
APP_ENV=development
LOG_LEVEL=INFO

HTTP_ADDRESS=127.0.0.1:9000
HTTP_CORS_ALLOWED_ORIGINS=*
HTTP_READ_TIMEOUT=5s
HTTP_READ_HEADER_TIMEOUT=2s
HTTP_WRITE_TIMEOUT=15s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=15s

AUTH_SESSION_TTL=24h
AUTH_SESSION_CACHE_TTL=5m
AUTH_BCRYPT_COST=12
AUTH_LOGIN_MAX_ATTEMPTS=5
AUTH_LOGIN_ATTEMPT_WINDOW=15m
GOOGLE_CLIENT_ID=
UPLOADS_DIR=uploads
UPLOAD_MAX_SIZE_MB=20

DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/lms_arvand?sslmode=disable
PGX_MAX_CONNS=20
PGX_MIN_CONNS=2
PGX_MAX_CONN_LIFETIME=30m
PGX_MAX_CONN_IDLE_TIME=5m
PGX_HEALTH_CHECK_PERIOD=1m

MIGRATE_RUN_ON_START=true
MIGRATIONS_DIR=migrations

SEED_RUN_ON_START=true
SEED_ADMIN_USERNAME=admin
SEED_ADMIN_EMAIL=admin@local.test
SEED_ADMIN_PASSWORD=Admin123!
SEED_ADMIN_FIRST_NAME=System
SEED_ADMIN_LAST_NAME=Admin
SEED_ADMIN_PATRONYMIC=
SEED_ADMIN_IS_SUPER_ADMIN=true
SEED_ADMIN_PERMISSIONS=*
```

The first login on the frontend should use `SEED_ADMIN_EMAIL` as the email input and `SEED_ADMIN_PASSWORD` as the password.

## 5. Run and build

Run locally:

```powershell
go run ./cmd/api
```

Build for Windows:

```powershell
go build -o bin/api.exe ./cmd/api
```

Build for Linux:

```bash
go build -o bin/api ./cmd/api
```

Tests:

```powershell
go test ./...
```

## 6. What starts at runtime

On startup the backend:

1. loads env files,
2. loads config,
3. creates logger,
4. prepares uploads directory,
5. connects to PostgreSQL,
6. runs migrations if enabled,
7. seeds admin if enabled,
8. wires repositories, usecases and handlers,
9. starts HTTP server on `HTTP_ADDRESS`,
10. serves `/uploads/*` from local disk.

## 7. Deployment notes

- The backend is plain HTTP on the configured address. TLS is usually handled by Nginx or another reverse proxy.
- Frontend origin must be allowed in `HTTP_CORS_ALLOWED_ORIGINS` if frontend and backend are on different origins.
- The `uploads/` directory must exist and be writable by the backend process.
- If migrations are enabled, startup fails when DB migration fails.

## 8. Common mistakes

- wrong PostgreSQL password in `DATABASE_URL`
- wrong port in `HTTP_ADDRESS`
- missing `GOOGLE_CLIENT_ID` when Google login is used
- missing write permissions for `UPLOADS_DIR`
- missing frontend origin in `HTTP_CORS_ALLOWED_ORIGINS`
- running the binary without a reachable PostgreSQL server

## 9. Recommended order for a clean startup

1. Start PostgreSQL.
2. Put `.env` in place.
3. Check `DATABASE_URL`.
4. Check `HTTP_ADDRESS` and `HTTP_CORS_ALLOWED_ORIGINS`.
5. Create or verify the uploads directory.
6. Run the backend.
7. Check `/health`.
8. Check `/api/v1/health`.
9. Check login.
10. Check one list endpoint and one upload URL.
