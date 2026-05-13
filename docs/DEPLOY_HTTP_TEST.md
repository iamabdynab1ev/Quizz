# HTTP Test Deploy

Тестовый запуск без HTTPS. Цель - один внешний порт для frontend и backend:

```text
Browser -> http://192.168.10.79:4041
             -> Nginx
                  /api/v1/* -> Go backend 127.0.0.1:9000
                  /health   -> Go backend 127.0.0.1:9000
                  /uploads/* -> Go backend 127.0.0.1:9000
                  /*        -> React build
```

## Backend

Сборка:

```bash
go build -o bin/api ./cmd/api
```

Запуск:

```bash
./bin/api
```

Минимальный `.env`:

```env
APP_NAME=QUIZ
APP_ENV=development
LOG_LEVEL=INFO

HTTP_ADDRESS=127.0.0.1:9000
HTTP_CORS_ALLOWED_ORIGINS=http://192.168.10.79:4041

DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/lms_arvand?sslmode=disable

MIGRATE_RUN_ON_START=true
MIGRATIONS_DIR=migrations
SEED_RUN_ON_START=true

SEED_ADMIN_EMAIL=admin@local.test
SEED_ADMIN_PASSWORD=Admin123!
SEED_ADMIN_FIRST_NAME=System
SEED_ADMIN_LAST_NAME=Admin

AUTH_LOGIN_LOCKOUT_ENABLED=false
AUTH_PASSWORD_RESET_RETURN_TOKEN=true
GOOGLE_CLIENT_ID=

UPLOADS_DIR=uploads
UPLOAD_MAX_SIZE_MB=20
```

## Frontend

Frontend должен обращаться к backend через тот же origin:

```env
VITE_API_URL=/api/v1
```

Сборка:

```bash
npm run build
```

Скопировать `dist/*` в папку nginx, например:

```text
/var/www/arvand
```

## Nginx

Пример:

```nginx
server {
    listen 4041;
    server_name 192.168.10.79;

    root /var/www/arvand;
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

Проверка:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## Проверка

```text
http://192.168.10.79:4041
http://192.168.10.79:4041/health
http://192.168.10.79:4041/api/v1/health
```

Первый вход:

```text
admin@local.test
Admin123!
```

## После успешного HTTP-теста

Для production:

- добавить HTTPS;
- поставить `AUTH_LOGIN_LOCKOUT_ENABLED=true`;
- поставить `AUTH_PASSWORD_RESET_RETURN_TOKEN=false`;
- сменить пароль seed admin;
- указать точный `HTTP_CORS_ALLOWED_ORIGINS`;
- оставить backend только на `127.0.0.1:9000`.
