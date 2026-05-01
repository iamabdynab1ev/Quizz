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
- Configuration is loaded from `.env`, `ENV_FILE`, exe folder, or parent folder
