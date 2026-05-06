# Windows Test Deploy

Тестовая схема для Windows-сервера:

```text
Browser -> http://192.168.10.79:4041
             -> nginx for Windows
                  /api/v1/* -> Go backend 127.0.0.1:9000
                  /health   -> Go backend 127.0.0.1:9000
                  /*        -> React build
```

Снаружи один порт: `4041`.

Backend не открывается наружу, он слушает только:

```env
HTTP_ADDRESS=127.0.0.1:9000
```

## Что собирать

Backend:

```powershell
go build -o bin/api.exe ./cmd/api
```

Frontend:

```powershell
npm run build
```

Нужны два результата:

- `bin/api.exe`
- frontend папка `dist/`

## Куда положить на сервере

Пример:

```text
C:\projects\Quiz\bin\api.exe
C:\projects\Quiz\.env
C:\var\www\arvand\index.html
C:\var\www\arvand\assets\...
```

## Backend `.env` на сервере

```env
APP_NAME=QUIZ
APP_ENV=development
LOG_LEVEL=INFO

HTTP_ADDRESS=127.0.0.1:9000
HTTP_CORS_ALLOWED_ORIGINS=http://192.168.10.79:4041,http://localhost:4041

DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5433/lms_arvand?sslmode=disable

MIGRATE_RUN_ON_START=true
MIGRATIONS_DIR=migrations
SEED_RUN_ON_START=true

SEED_ADMIN_EMAIL=admin@local.test
SEED_ADMIN_PASSWORD=Admin123!
SEED_ADMIN_FIRST_NAME=System
SEED_ADMIN_LAST_NAME=Admin

AUTH_LOGIN_LOCKOUT_ENABLED=false
AUTH_PASSWORD_RESET_RETURN_TOKEN=true
AUTH_GOOGLE_DEFAULT_ROLE=student
GOOGLE_CLIENT_ID=

UPLOADS_DIR=uploads
UPLOAD_MAX_SIZE_MB=20
```

Если PostgreSQL на `5432`, поменять только `DATABASE_URL`.

## Запуск backend вручную

```powershell
cd C:\projects\Quiz
.\bin\api.exe
```

Проверить:

```powershell
Invoke-RestMethod http://127.0.0.1:9000/health
Invoke-RestMethod http://127.0.0.1:9000/api/v1/health
```

## Frontend build

Для одного origin через nginx:

```env
VITE_API_URL=/api/v1
```

Собрать:

```powershell
npm run build
```

Содержимое `dist/` скопировать в:

```text
C:\var\www\arvand
```

## Nginx for Windows

Пример конфига:

```nginx
server {
    listen 4041;
    server_name 192.168.10.79 localhost;

    root C:/var/www/arvand;
    index index.html;

    location /api/v1/ {
        proxy_pass http://127.0.0.1:9000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://127.0.0.1:9000;
    }

    location /uploads/ {
        proxy_pass http://127.0.0.1:9000;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

## Проверка после запуска

Открыть:

```text
http://192.168.10.79:4041
```

Проверить:

1. login `admin@local.test / Admin123!`;
2. `GET /api/v1/auth/me`;
3. список курсов;
4. создание курса;
5. создание теста;
6. сдача теста;
7. сертификат.

## После теста

Когда HTTP-схема проверена:

- добавить TLS certificate;
- перевести nginx на `listen 4041 ssl`;
- оставить backend на `127.0.0.1:9000`;
- выключить тестовые настройки:
  - `AUTH_LOGIN_LOCKOUT_ENABLED=true`;
  - `AUTH_PASSWORD_RESET_RETURN_TOKEN=false`;
  - сменить `SEED_ADMIN_PASSWORD`.
