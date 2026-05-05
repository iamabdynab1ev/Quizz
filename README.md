# QUIZ / LMS Arvand Backend

Backend for the QUIZ / LMS Arvand platform.

## Main docs

- [Backend current state](docs/BACKEND_CURRENT_STATE.md)
- [Frontend guide](docs/FRONTEND.md)
- [Project setup and configuration](docs/PROJECT_SETUP.md)

## Quick start

```powershell
go run ./cmd/api
```

Build:

```powershell
go build -o bin/api.exe ./cmd/api
```

Tests:

```powershell
go test ./...
```

## Runtime notes

- Base API prefix: `/api/v1`
- Health checks: `/health` and `/api/v1/health`
- Static uploads are served from `/uploads/*`
- Google Sign-In config is public at `/api/v1/auth/google/config`; set `GOOGLE_CLIENT_ID` in `.env` to enable it.
- Login lockout is controlled by `.env`: `AUTH_LOGIN_LOCKOUT_ENABLED`, `AUTH_LOGIN_MAX_ATTEMPTS`, `AUTH_LOGIN_ATTEMPT_WINDOW`, `AUTH_LOGIN_LOCKOUT_SCOPE`.
- Configuration is loaded from `.env` files with this priority:
  1. `ENV_FILE` if set
  2. `.env` in the current working directory
  3. `.env` next to the compiled exe
  4. `.env` in the parent directory of the exe
