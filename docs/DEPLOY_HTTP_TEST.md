# HTTP Test Deploy

Тестовый запуск проекта без HTTPS:

- frontend доступен на `http://192.168.10.79:4041`
- backend слушает `127.0.0.1:9000`
- nginx разруливает один origin и один порт

## 1. Backend

Используй env из [deploy/http-test/backend.env.example](../deploy/http-test/backend.env.example).

Ключевые значения:

- `HTTP_ADDRESS=127.0.0.1:9000`
- `HTTP_CORS_ALLOWED_ORIGINS=*`
- `MIGRATE_RUN_ON_START=true`
- `SEED_RUN_ON_START=true`

Сборка и запуск:

```bash
go build -o bin/api ./cmd/api
./bin/api
```

Если backend запускать как systemd service, используй [deploy/http-test/arvand-api.service](../deploy/http-test/arvand-api.service).

## 2. Nginx

Положи [deploy/http-test/nginx.conf](../deploy/http-test/nginx.conf) в `/etc/nginx/sites-available/arvand`.

Проверь:

```bash
sudo ln -sf /etc/nginx/sites-available/arvand /etc/nginx/sites-enabled/arvand
sudo nginx -t
sudo systemctl reload nginx
```

Конфиг делает следующее:

- `/api/v1/*` -> `127.0.0.1:9000`
- `/health` -> `127.0.0.1:9000`
- `/` -> React build из `/var/www/arvand`

## 3. Frontend

React build должен обращаться к API через тот же origin:

```env
VITE_API_URL=/api/v1
```

Если frontend по какой-то причине не поддерживает относительный base path, используй:

```env
VITE_API_URL=http://192.168.10.79:4041/api/v1
```

После сборки фронта скопируй `dist/*` в `/var/www/arvand`.

## 4. Итоговая схема

```text
Browser
  -> http://192.168.10.79:4041
       -> Nginx
            /api/v1/* -> Go :9000
            /health   -> Go :9000
            /*        -> React static files
```

## 5. После теста

Когда будете готовы к HTTPS:

- добавить TLS certificate в nginx
- заменить `listen 4041` на `listen 4041 ssl`
- включить `ssl_certificate`
- оставить backend без публичного порта наружу
