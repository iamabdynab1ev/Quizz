# QUIZ Backend Current State

Документ описывает фактическое состояние backend проекта `QUIZ` по текущему коду. Это не roadmap и не желаемая архитектура, а рабочая реализация на сейчас.

## Назначение

Backend реализует LMS/QUIZ платформу:

- регистрация, вход по email/password и вход через Google;
- opaque session auth без JWT;
- управление пользователями, курсами, видео, тестами и вопросами;
- прохождение тестов с баллами, лимитом попыток и паузой перед новой сдачей;
- автоматическая и ручная выдача сертификатов;
- профили пользователей и смена/сброс пароля;
- загрузка файлов на локальный диск;
- отзывы, уведомления, dashboard;
- audit logs и webhooks.

Проект является Go-монолитом с одной PostgreSQL базой. Redis, message broker, S3/MinIO и JWT не обязательны и сейчас не используются.

## Технологии

- Go `1.24`
- HTTP: `net/http`
- Router: `github.com/go-chi/chi/v5`
- PostgreSQL: `github.com/jackc/pgx/v5/pgxpool`
- Password hashing: `bcrypt`
- Logging: `log/slog`
- Config: `.env` + переменные окружения
- Миграции: внутренний bootstrap migrator
- Upload storage: локальная папка `UPLOADS_DIR`
- Google Sign-In: проверка `id_token` через Google `tokeninfo`

ORM не используется. SQL написан вручную в `internal/repository/postgres`.

## Архитектура

Слои выдержаны так:

```text
Router -> Handler -> UseCase -> Repository -> PostgreSQL
```

Основные директории:

- `cmd/api` - основной entrypoint API.
- `cmd/dbcreate` - вспомогательная команда для создания БД.
- `cmd/dbseed` - вспомогательная команда seed.
- `internal/config` - загрузка `.env` и env-конфига.
- `internal/bootstrap` - миграции и bootstrap admin.
- `internal/domain` - доменные модели, enum-ы, ошибки.
- `internal/handler/http` - router, handlers, response helpers, middleware.
- `internal/usecase` - бизнес-логика.
- `internal/repository/postgres` - SQL-доступ.
- `internal/cache` - in-memory session cache.
- `internal/storage` - локальное файловое хранилище.
- `migrations` - SQL schema и изменения.
- `docs` - документация.

## Runtime

Запуск:

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

Флаги:

- `-migrate` - применить миграции и выйти.
- `-seed-admin` - создать/обновить bootstrap admin и выйти.
- `-all` - миграции + seed и выйти.

При обычном запуске backend:

1. загружает `.env`;
2. читает конфиг;
3. подключается к PostgreSQL;
4. применяет миграции, если `MIGRATE_RUN_ON_START=true`;
5. создаёт/обновляет bootstrap admin, если `SEED_RUN_ON_START=true`;
6. собирает repositories/usecases/handlers;
7. запускает HTTP server на `HTTP_ADDRESS`;
8. раздаёт `/uploads/*` из `UPLOADS_DIR`.

## Конфигурация

Файл `.env.example` является основным шаблоном.

Ключевые настройки:

- `APP_NAME=QUIZ`
- `HTTP_ADDRESS=127.0.0.1:9000`
- `HTTP_CORS_ALLOWED_ORIGINS=http://localhost:4041`
- `DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5433/lms_arvand?sslmode=disable`
- `AUTH_SESSION_TTL=8h`
- `AUTH_SESSION_CACHE_TTL=5m`
- `AUTH_LOGIN_LOCKOUT_ENABLED=false|true`
- `AUTH_PASSWORD_RESET_TOKEN_TTL=15m`
- `AUTH_PASSWORD_RESET_RETURN_TOKEN=false|true`
- `GOOGLE_CLIENT_ID=...apps.googleusercontent.com`
- `AUTH_GOOGLE_DEFAULT_ROLE=student`
- `UPLOADS_DIR=uploads`
- `UPLOAD_MAX_SIZE_MB=20`
- `MIGRATE_RUN_ON_START=true`
- `SEED_RUN_ON_START=true`
- `SEED_ADMIN_EMAIL=admin@local.test`
- `SEED_ADMIN_PASSWORD=Admin123!`

`.env` загружается из:

1. `ENV_FILE`, если указан;
2. `.env` рядом с exe;
3. `.env` в родительской папке exe;
4. `.env` в текущей рабочей директории.

## Пользователи и права

Внутренняя роль в БД остаётся enum:

- `admin`
- `employee`
- `student`
- `guest`

В public API роль упрощена:

- `is_admin: true` значит пользователь админ.
- отсутствие `is_super_admin` значит обычный пользователь или обычный админ.
- `is_super_admin: true` возвращается только bootstrap super admin.
- `username` наружу больше не отдаётся.
- `is_active` наружу в user response не отдаётся.
- `gender` наружу упрощён до `is_male`.
- `birth_date` возвращается только если есть значение.

Super admin:

- создаётся/поддерживается из `.env` через seed admin;
- только он может управлять пользователями и создавать других админов;
- обычный `is_admin=true` нужен для управления учебным контентом.

Обычный пользователь может редактировать свой профиль через `PUT /api/v1/auth/me`: имя, фамилию, телефон, город, дату рождения и похожие профильные поля. Управление ролью, активностью и админством ему недоступно.

## Auth

Поддерживается:

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/google/config`
- `POST /api/v1/auth/google`
- `POST /api/v1/auth/password/forgot`
- `POST /api/v1/auth/password/reset`
- `GET /api/v1/auth/me`
- `PUT /api/v1/auth/me`
- `POST /api/v1/auth/password/change`
- `POST /api/v1/auth/logout`

Вход:

- основной логин: email + password;
- `identifier` ещё принимается как compatibility;
- после успешного входа создаётся opaque session token;
- token нужно отправлять в `Authorization: Bearer <token>`.

Сессии:

- хранятся в PostgreSQL;
- hot path использует `sessions.GetByTokenWithUser`, то есть один SQL JOIN вместо двух отдельных запросов;
- дополнительно есть in-memory cache на `AUTH_SESSION_CACHE_TTL`.

Login lockout:

- настраивается через `.env`;
- можно отключить для локального тестирования;
- scope: `identifier`, `ip`, `identifier_ip`.

Password reset:

- `forgot` создаёт reset token;
- `reset` меняет пароль по token;
- для локального теста можно включить `AUTH_PASSWORD_RESET_RETURN_TOKEN=true`, тогда token возвращается прямо в response;
- для production это должно быть `false`, token нужно доставлять через email/SMS/другой доверенный канал.

Google login:

- frontend получает Google ID token через Google SDK;
- backend проверяет token через Google;
- новый пользователь получает роль из `AUTH_GOOGLE_DEFAULT_ROLE`, по умолчанию `student`;
- нужен реальный `GOOGLE_CLIENT_ID`.

## Ошибки API

Для общей ошибки:

```json
{
  "error": "conflict",
  "message": "Лимит попыток исчерпан. Повторная сдача будет доступна после 05.06.2026."
}
```

Для одной ошибки конкретного поля backend отдаёт плоский формат:

```json
{
  "field": "passing_points",
  "code": "too_high",
  "message": "Баллы для прохождения не могут быть больше максимального балла теста (10.00)"
}
```

Для нескольких ошибок формы backend отдаёт:

```json
{
  "error": "validation_error",
  "message": "Проверьте поля формы",
  "fields": [
    {
      "field": "title.ru",
      "code": "required",
      "message": "Название обязательно"
    }
  ]
}
```

HTTP mapping:

- `400 validation_error`
- `401 unauthorized`
- `403 forbidden`
- `404 not_found`
- `409 conflict`
- `429 too_many_requests`
- `503 service_unavailable`
- `500 internal_error`

## Курсы

Курс содержит:

- `title`
- `description`
- `cover_image_url`
- `video_url`
- `quiz_id`
- `category`
- `estimated_minutes`
- timestamps

В public response скрыты внутренние поля:

- `status`
- `platforms`
- `certificate_enabled`
- `certificate_passing_score`
- `reviews_enabled`

Удаление курса - soft delete через `status=archived`. Архивные курсы не должны попадать во frontend list.

## Тесты и вопросы

Quiz содержит:

- `title`
- `description`
- `course_id`
- `category`
- `time_limit_minutes`
- `passing_score` legacy процент
- `passing_points` основной порог в баллах
- `max_attempts`
- `retake_cooldown_days`
- `allow_retry`
- `questions`

В public response скрыты:

- `status`
- `platforms`
- `shuffle_questions`
- `show_results`

Backend принимает и snake_case, и frontend camelCase:

- `passing_points` и `passingPoints`
- `max_attempts` и `maxAttempts`
- `retake_cooldown_days` и `retakeCooldownDays`
- `is_correct` и `isCorrect`
- `accepted_answers` и `acceptedAnswers`

Поддерживаемые типы вопросов:

- `single_choice`
- `multiple_choice`
- `true_false`
- `short_answer`
- `long_text`
- `matching`
- `ordering`
- `fill_blank`
- `image_choice`
- `audio`
- `video`
- `code`

Для frontend правильные ответы не отдаются. Из `config` удаляются answer keys:

- `is_correct`, `isCorrect`
- `correct`
- `accepted_answers`, `acceptedAnswers`
- `correct_answer`, `correctAnswers`
- `answer_key`
- похожие ключи правильных ответов.

## Проходной балл, попытки и сертификаты

Новая логика:

- максимум теста считается динамически как сумма `points` всех вопросов;
- `passing_points` не может быть больше этого максимума;
- попытка считается успешной, если `total_earned >= passing_points`;
- если `passing_points` не задан, backend fallback-ит его из legacy `passing_score`;
- если пользователь уже получил сертификат по курсу, повторная сдача теста закрыта;
- видео/курс остаются доступными после сертификата;
- если попытки исчерпаны, новая сдача открывается после `retake_cooldown_days`;
- после окончания cooldown попытки считаются заново.

Пример:

- в тесте 10 вопросов по 1 баллу, максимум `10`;
- админ ставит `passingPoints=8`;
- студент набрал `8`, `9` или `10` - `passed=true`, сертификат может быть выдан;
- студент набрал `7` - `passed=false`, сертификат не выдаётся;
- если админ поставит `passingPoints=11`, backend вернёт ошибку по `passing_points`.

Если в тесте 5 вопросов по 1 баллу, максимум будет `5`, и `passingPoints=6` будет ошибкой. Значение не захардкожено.

## Attempts

В БД attempt хранит:

- `questions_snapshot`
- `answers_data`
- `total_earned`
- `total_max`
- `score_percent`
- `passed`
- `needs_review`
- review fields

Во frontend response скрыты:

- `questions_snapshot`
- `answers_data`
- `needs_review`

Автооценка работает для:

- `single_choice`
- `image_choice`
- `multiple_choice`
- `true_false`
- `short_answer`
- `fill_blank`

Manual review нужен для:

- `long_text`
- `matching`
- `ordering`
- `audio`
- `video`
- `code`

Admin endpoint `POST /api/v1/attempts/{attemptID}/review` пересчитывает ручные баллы и снова применяет `passing_points`.

## Enrollment и certificate

Enrollment:

- `active`
- `completed`
- `dropped`

`POST /api/v1/enrollments/{id}/complete` завершает enrollment и пытается автоматически выпустить сертификат, если есть успешная попытка.

Certificate создаётся только если:

- курс разрешает сертификаты;
- attempt связан с этим курсом через `course_tests`;
- attempt принадлежит тому же user;
- attempt `passed=true`;
- сертификат ещё не был выдан.

`POST /api/v1/certificates` оставлен как admin override, но основной flow должен идти через успешную попытку и completion.

## Uploads

`POST /api/v1/uploads`:

- доступ: admin;
- `multipart/form-data`;
- fields: `file`, `type`;
- type: `image`, `video`, `file`, `avatar`;
- возвращает `url`, `filename`, `size_bytes`;
- файлы физически сохраняются в `UPLOADS_DIR`;
- раздаются через `/uploads/*`.

## Audit logs и webhooks

`POST /audit-logs` отсутствует. Audit записи создаются внутренним `AuditLogger` как side effect usecase-ов.

Webhooks:

- CRUD реализован;
- `secret` возвращается только при создании;
- read/update/list secret не раскрывают;
- delivery идёт через audit outbox worker;
- подпись HMAC-SHA256 в `X-LMS-Signature`;
- обновляются `deliveries`, `failures`, `last_status_code`, `last_error`, `last_triggered_at`.

Ограничение: worker живёт внутри API процесса, отдельного worker service пока нет.

## Реальные маршруты

Public:

- `GET /health`
- `GET /api/v1/health`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/google/config`
- `POST /api/v1/auth/google`
- `POST /api/v1/auth/password/forgot`
- `POST /api/v1/auth/password/reset`
- `GET /api/v1/certificates/verify/{verifyHash}`
- `GET /uploads/*`

Protected:

- `GET /api/v1/auth/me`
- `PUT /api/v1/auth/me`
- `POST /api/v1/auth/password/change`
- `POST /api/v1/auth/logout`
- `GET /api/v1/dashboard`
- `GET /api/v1/dashboard/me`
- `GET /api/v1/courses`
- `GET /api/v1/courses/{courseID}`
- `GET /api/v1/quizzes`
- `GET /api/v1/quizzes/{quizID}`
- `POST /api/v1/quizzes/{quizID}/attempts`
- `GET /api/v1/attempts`
- `GET /api/v1/attempts/{attemptID}`
- `GET /api/v1/enrollments`
- `POST /api/v1/enrollments`
- `GET /api/v1/enrollments/{enrollmentID}`
- `GET /api/v1/certificates`
- `GET /api/v1/certificates/{certificateID}`
- `GET /api/v1/course-modules`
- `GET /api/v1/course-modules/{moduleID}`
- `GET /api/v1/content-blocks`
- `GET /api/v1/content-blocks/{blockID}`
- `GET /api/v1/reviews`
- `POST /api/v1/reviews`
- `GET /api/v1/reviews/{reviewID}`
- `GET /api/v1/notifications`
- `GET /api/v1/notifications/{notificationID}`
- `POST /api/v1/notifications/{notificationID}/read`

Admin:

- `GET /api/v1/dashboard/admin`
- `POST /api/v1/courses`
- `PUT /api/v1/courses/{courseID}`
- `DELETE /api/v1/courses/{courseID}`
- `POST /api/v1/course-packages`
- `POST /api/v1/quizzes`
- `PUT /api/v1/quizzes/{quizID}`
- `DELETE /api/v1/quizzes/{quizID}`
- `POST /api/v1/attempts/{attemptID}/review`
- `POST /api/v1/enrollments/{enrollmentID}/complete`
- `POST /api/v1/certificates`
- mutations for course modules/content blocks/reviews/notifications
- `GET/POST/DELETE /api/v1/course-tests`
- `GET/POST/PUT/DELETE /api/v1/webhooks`
- `GET /api/v1/audit-logs`
- `POST /api/v1/uploads`

Super admin only:

- `GET /api/v1/users`
- `POST /api/v1/users`
- `GET /api/v1/users/{userID}`
- `PUT /api/v1/users/{userID}`
- `DELETE /api/v1/users/{userID}`

## Текущие сильные стороны

- слоистая архитектура читается и поддерживается;
- SQL изолирован в repository;
- session auth проще и надёжнее JWT для монолита;
- Google login встроен в ту же session модель;
- correct answers скрыты из frontend responses;
- list endpoints возвращают `data/total/limit/offset`;
- upload API уже есть;
- webhooks не раскрывают secret;
- audit create endpoint убран;
- password reset и password change реализованы;
- super admin отделён от обычного admin;
- новая логика сертификатов работает по реальным баллам.

## Оставшиеся ограничения

- нет Redis/distributed cache для multi-node;
- password reset пока не доставляет token через email/SMS, если `AUTH_PASSWORD_RESET_RETURN_TOKEN=false`;
- webhook worker живёт внутри API процесса;
- часть логов в коде ещё требует нормализации кодировки;
- `course.certificate_passing_score` остался legacy полем курса и больше не должен быть главным источником логики сертификата;
- manual review flow минимальный, без отдельного UI breakdown API.
