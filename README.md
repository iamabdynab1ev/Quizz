# QUIZ Backend

Go backend для LMS/QUIZ платформы.

## Документы

- [Backend current state](docs/BACKEND_CURRENT_STATE.md)
- [Frontend API contract](docs/FRONTEND_API.md)
- [Frontend guide](docs/FRONTEND.md)
- [Project setup](docs/PROJECT_SETUP.md)
- [Windows test deploy](docs/DEPLOY_WINDOWS_TEST.md)
- [HTTP test deploy](docs/DEPLOY_HTTP_TEST.md)

## Быстрый запуск

```powershell
go run ./cmd/api
```

Сборка:

```powershell
go build -o bin/api.exe ./cmd/api
```

Проверка:

```powershell
go test ./...
go vet ./...
```

## Основные runtime notes

- API prefix: `/api/v1`
- Health: `/health` и `/api/v1/health`
- Uploads: `/uploads/*`
- Auth: opaque session token через `Authorization: Bearer <token>`
- Login: email/password или Google ID token
- Публично без токена: `GET /api/v1/courses`, `GET /api/v1/courses/{id}`, `GET /api/v1/quizzes/{id}`, `GET /api/v1/certificates/{id}`
- Первый admin берётся из `.env`: `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`
- Только bootstrap user получает `is_super_admin=true`
- Frontend должен использовать `is_admin`, а не `role`
- Списки возвращают `data/total/limit/offset`
- Правильные ответы в quiz response скрыты
- Сдача теста идёт через `POST /api/v1/courses/{courseID}/attempts`
- Сертификат выдаётся после `passed=true`, где `score_percent >= quiz_pass_percent`
