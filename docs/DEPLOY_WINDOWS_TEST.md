# Windows Test Deploy

Тестовый запуск на Windows-сервере:

- frontend: `http://192.168.10.79:4041`
- backend: `127.0.0.1:8080`
- nginx for Windows слушает один порт `4041`

## 1. Что собирать

Нужны два артефакта:

- Go backend binary для Windows: `api.exe`
- React build: папка `dist/`

## 2. Backend

Собирай Windows binary на своей машине или на сервере:

```powershell
go build -o bin/api.exe ./cmd/api
```

Потом положи на сервер в, например:

```text
C:\projects\Quiz\bin\api.exe
```

Используй env из [deploy/windows-test/backend.env.example](../deploy/windows-test/backend.env.example).

Ключевые значения:

- `HTTP_ADDRESS=127.0.0.1:8080`
- `HTTP_CORS_ALLOWED_ORIGINS=*`
- `AUTH_SESSION_CACHE_TTL=5m`
- `MIGRATE_RUN_ON_START=true`
- `SEED_RUN_ON_START=true`

Важно:

- backend читает настройки только из env-файла или переменных окружения
- если меняется Postgres, порт, пароль или адрес сервера, правишь только `.env`
- после изменения `.env` нужен перезапуск `api.exe`
- runtime hot-reload env сейчас не делается

## 3. Как запустить backend на Windows

Самый простой вариант для теста:

```powershell
cd C:\projects\Quiz
.\bin\api.exe
```

Если хочешь без ручного поиска `.env`, используй [deploy/windows-test/run-api.ps1](../deploy/windows-test/run-api.ps1):

```powershell
powershell -ExecutionPolicy Bypass -File .\deploy\windows-test\run-api.ps1 -Root C:\projects\Quiz -ExePath C:\projects\Quiz\bin\api.exe
```

Лучше как service:

- поставить `NSSM`
- создать сервис на `C:\projects\Quiz\bin\api.exe`
- рабочая папка: `C:\projects\Quiz`
- env подать через `.env` файл рядом с бинарником

Если `.env` не подхватывается, проверь:

- файл действительно называется `.env`, а не `.env.txt`
- `ENV_FILE` не задан на старое значение
- бинарник пересобран после фикса поиска env

Пример переменной подключения к PostgreSQL:

```env
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@127.0.0.1:5432/lms_arvand?sslmode=disable
```

## 4. Nginx for Windows

Положи [deploy/windows-test/nginx.conf](../deploy/windows-test/nginx.conf) в nginx-конфиг.

Важно:

- `root` должен указывать на папку с React build, например `C:/var/www/arvand`
- nginx слушает `4041`
- backend живет только на `127.0.0.1:8080`

## 5. Frontend

Для React в тесте можно использовать относительный API path:

```env
VITE_API_URL=/api/v1
```

Собери фронт:

```bash
npm run build
```

Потом скопируй `dist/*` в папку, которую читает nginx, например:

```text
C:\var\www\arvand
```

## 6. Итоговая схема

```text
Browser
  -> http://192.168.10.79:4041
       -> nginx for Windows
            /api/v1/* -> Go backend 127.0.0.1:8080
            /health   -> Go backend 127.0.0.1:8080
            /*        -> React static files
```

## 7. Что делать после теста

Когда всё проверишь на HTTP:

- добавить HTTPS certificate в nginx
- заменить `listen 4041` на `listen 4041 ssl`
- оставить backend без публичного доступа
