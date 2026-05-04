# LMS Arvand Backend Current State

Документ описывает текущее состояние backend проекта `lms-arvand-backend` по исходному коду репозитория. Это не roadmap и не желаемая архитектура, а фактическая реализация на данный момент.

Для тестового запуска фронта и backend см. [DEPLOY_WINDOWS_TEST.md](DEPLOY_WINDOWS_TEST.md).

## 1. Назначение проекта

Проект реализует backend LMS-платформы для:

- обучения сотрудников, студентов, гостей и других категорий пользователей;
- публикации курсов и тестов;
- прохождения тестов с ограничением по количеству попыток;
- хранения snapshots попыток;
- ручной выдачи сертификатов;
- работы с bilingual-контентом (`ru` / `tj`);
- отзывов о курсах;
- пользовательских уведомлений;{
  "identifier": "admin",
  "password": "Admin123!"
}
- административного управления пользователями и учебным контентом;
- webhooks как админской сущности и audit logs как читаемого журнала.

Система не построена вокруг микросервисов. Это один Go-монолит с HTTP API и PostgreSQL.

## 2. Технологический стек

Текущий стек:

- Go `1.24`
- HTTP server: `net/http`
- Router: `github.com/go-chi/chi/v5`
- Middleware: `chi/middleware`
- Logging: `log/slog`
- PostgreSQL driver/pool: `github.com/jackc/pgx/v5/pgxpool`
- Password hashing: `golang.org/x/crypto/bcrypt`
- Google token verification: `https://oauth2.googleapis.com/tokeninfo`
- SQL access: raw SQL, без ORM

Что не используется:

- `gorm`
- `ent`
- `sqlx`
- separate external message broker / queue
- external S3/MinIO object storage

Для browser frontend сейчас уже есть CORS middleware с конфигурацией через `HTTP_CORS_ALLOWED_ORIGINS`.

## 3. Общая архитектура

Проект построен слоисто:

`Router -> Handler -> UseCase -> Repository -> PostgreSQL`

### 3.1 Router

Маршруты собираются в [router.go](C:/projects/go/src/myApps/quiz/internal/handler/http/router.go:1).

Router отвечает за:

- разбиение на `public`, `protected`, `admin`;
- подключение middleware;
- wiring HTTP endpoint-ов.

### 3.2 Handler layer

Handlers живут в `internal/handler/http`.

Задачи handler-слоя:

- читать path/query параметры;
- декодировать JSON body;
- навесить transport-level валидацию;
- вызвать usecase;
- отдать JSON или mapped HTTP error.

Backend использует helper [decodeJSON](C:/projects/go/src/myApps/quiz/internal/handler/http/request.go:9):

- `MaxBytesReader` ограничивает размер body;
- `json.Decoder.DisallowUnknownFields()` запрещает неизвестные поля;
- body должен содержать ровно один JSON объект.

JSON response helper now supports paged responses via [writePagedJSON](C:/projects/go/src/myApps/quiz/internal/handler/http/response.go:24). List responses are wrapped in `data/total/limit/offset`.

### 3.3 UseCase layer

UseCase-слой находится в `internal/usecase`.

Он отвечает за:

- нормализацию входных данных;
- доменную валидацию;
- бизнес-правила;
- orchestration нескольких repository вызовов;
- принятие решений по доступу и конфликтам на уровне домена.

Примеры:

- [auth.go](C:/projects/go/src/myApps/quiz/internal/usecase/auth.go:32)
- [attempts.go](C:/projects/go/src/myApps/quiz/internal/usecase/attempts.go:19)
- [certificates.go](C:/projects/go/src/myApps/quiz/internal/usecase/certificates.go:18)
- [content_blocks.go](C:/projects/go/src/myApps/quiz/internal/usecase/content_blocks.go:11)

### 3.4 Repository layer

Repository-слой находится в `internal/repository/postgres`.

Он отвечает за:

- SQL;
- scan rows;
- insert/update/select/delete;
- преобразование DB rows в `domain` модели.

Repository-слой не содержит HTTP-логики.

## 4. Структура репозитория

Основные entrypoint-ы:

- [cmd/api/main.go](C:/projects/go/src/myApps/quiz/cmd/api/main.go:1)
- [cmd/dbcreate/main.go](C:/projects/go/src/myApps/quiz/cmd/dbcreate/main.go:1)
- [cmd/dbseed/main.go](C:/projects/go/src/myApps/quiz/cmd/dbseed/main.go:1)

Основные пакеты:

- `internal/config` — чтение `.env`, конфиг приложения;
- `internal/bootstrap` — миграции и bootstrap seed;
- `internal/domain` — доменные сущности, enum-ы, ошибки;
- `internal/handler/http` — HTTP handlers, router, auth middleware, helpers;
- `internal/usecase` — бизнес-логика;
- `internal/repository/postgres` — SQL-доступ к PostgreSQL;
- `migrations` — SQL schema;
- `docs` — документация.

## 5. Runtime и запуск

### 5.1 `.env` и конфиг

Конфиг читается через:

- [envfile.go](C:/projects/go/src/myApps/quiz/internal/config/envfile.go:1)
- [config.go](C:/projects/go/src/myApps/quiz/internal/config/config.go:1)

Ключевые env:

- `APP_NAME`
- `APP_ENV`
- `LOG_LEVEL`
- `HTTP_ADDRESS`
- `HTTP_READ_TIMEOUT`
- `HTTP_READ_HEADER_TIMEOUT`
- `HTTP_WRITE_TIMEOUT`
- `HTTP_IDLE_TIMEOUT`
- `HTTP_SHUTDOWN_TIMEOUT`
- `AUTH_SESSION_TTL`
- `AUTH_SESSION_CACHE_TTL`
- `AUTH_BCRYPT_COST`
- `GOOGLE_CLIENT_ID`
- `DATABASE_URL`
- `PGX_MAX_CONNS`
- `PGX_MIN_CONNS`
- `PGX_MAX_CONN_LIFETIME`
- `PGX_MAX_CONN_IDLE_TIME`
- `PGX_HEALTH_CHECK_PERIOD`
- `MIGRATE_RUN_ON_START`
- `MIGRATIONS_DIR`
- `SEED_RUN_ON_START`
- `SEED_ADMIN_*`

### 5.2 HTTP server

`cmd/api/main.go` делает:

- загрузку `.env`;
- загрузку конфига;
- инициализацию `pgxpool`;
- опциональный запуск миграций;
- опциональный bootstrap admin seed;
- сборку repositories/usecases/handlers;
- запуск `http.Server`;
- graceful shutdown по `SIGTERM`/`Ctrl+C`.

### 5.3 Флаги запуска

Поддерживаются:

- `go run ./cmd/api`
- `go run ./cmd/api -migrate`
- `go run ./cmd/api -seed-admin`
- `go run ./cmd/api -all`

Смысл:

- `-migrate` — применить миграции и выйти;
- `-seed-admin` — создать/обновить bootstrap admin и выйти;
- `-all` — миграции + сид и выйти;
- без флагов — обычный запуск API.

### 5.4 Миграции

Миграции лежат в:

- [00001_init_schema.sql](C:/projects/go/src/myApps/quiz/migrations/00001_init_schema.sql:1)
- [00002_users_google_id_unique_index.sql](C:/projects/go/src/myApps/quiz/migrations/00002_users_google_id_unique_index.sql:1)

Применение идет через bootstrap migrator, а не через отдельную внешнюю CLI-команду.

### 5.5 Bootstrap admin seed

Bootstrap admin создается через:

- [admin_seed.go](C:/projects/go/src/myApps/quiz/internal/bootstrap/admin_seed.go:1)

Поведение:

- если admin не найден — создается;
- если найден — обновляется;
- пользователь принудительно остается `is_active = true`.

## 6. Middleware и HTTP pipeline

На router навешаны:

- `RealIP`
- `RequestID`
- `Recoverer`
- собственный `RequestLogger`

Авторизация подключается через:

- [RequireAuth](C:/projects/go/src/myApps/quiz/internal/handler/http/middleware/auth.go:22)
- [RequireRoles](C:/projects/go/src/myApps/quiz/internal/handler/http/middleware/auth.go:52)

Логика доступа к своим данным реализована helper-ами:

- [scopeUserID](C:/projects/go/src/myApps/quiz/internal/handler/http/authz.go:19)
- [resolveActorUserID](C:/projects/go/src/myApps/quiz/internal/handler/http/authz.go:35)
- [ensureOwnOrAdmin](C:/projects/go/src/myApps/quiz/internal/handler/http/authz.go:53)

Это означает:

- admin может читать/создавать ресурсы за других пользователей;
- non-admin ограничен своим `user_id`;
- попытки, enrollment-ы, сертификаты, уведомления и некоторые чтения защищены ownership-check логикой.

## 7. Модель ошибок

Доменная ошибка описана в [errors.go](C:/projects/go/src/myApps/quiz/internal/domain/errors.go:1).

Базовые sentinel errors:

- `ErrValidation`
- `ErrUnauthorized`
- `ErrForbidden`
- `ErrNotFound`
- `ErrConflict`
- `ErrUnavailable`

Есть и richer-форма `AppError`:

- `Code`
- `Message`
- `Status`
- `Err`

HTTP mapping выполняется в [error_map.go](C:/projects/go/src/myApps/quiz/internal/handler/http/error_map.go:10).

Текущий JSON error contract:

```json
{
  "error": "validation_error",
  "message": "request validation failed"
}
```

## 8. Auth: как реализовано

### 8.1 Что используется

Текущая auth-модель:

- логин по `username` или `email`;
- bcrypt password check;
- opaque session token;
- таблица `sessions`;
- Google Sign-In как альтернативный public login flow.

Backend не использует JWT.

### 8.2 Login flow

Основная логика:

- [AuthUseCase.Login](C:/projects/go/src/myApps/quiz/internal/usecase/auth.go:48)

Шаги:

1. нормализовать login params;
2. проверить недавние неудачные попытки логина;
3. найти пользователя по `identifier`;
4. проверить `is_active`;
5. проверить `password_hash`;
6. сравнить пароль через bcrypt;
7. записать результат login attempt;
8. сгенерировать случайный session token;
9. создать session в БД;
10. вернуть `LoginResult`.

Текущее защитное поведение:

- backend ведет таблицу `login_attempts`
- при превышении лимита возвращается `429 too_many_attempts`
- окно и лимит задаются через auth config

### 8.3 Сессии

Session repository:

- [sessions.go](C:/projects/go/src/myApps/quiz/internal/repository/postgres/sessions.go:1)

Сессия содержит:

- `token`
- `user_id`
- `ip_address`
- `user_agent`
- `created_at`
- `expires_at`

### 8.4 Authenticate на каждом запросе

Текущее поведение в [AuthUseCase.Authenticate](C:/projects/go/src/myApps/quiz/internal/usecase/auth.go:137):

1. взять token из Bearer header;
2. сделать `sessions.GetByTokenWithUser`;
3. проверить expiration;
4. проверить `is_active`;
5. собрать `AuthIdentity`.

Следствие:

- auth hot path уже ужат до одного SQL lookup с `JOIN sessions + users`

### 8.5 Google Sign-In

Google login уже реализован:

- [POST /auth/google](C:/projects/go/src/myApps/quiz/internal/handler/http/router.go:48)
- [LoginWithGoogle](C:/projects/go/src/myApps/quiz/internal/usecase/auth.go:87)
- [Google verifier](C:/projects/go/src/myApps/quiz/internal/usecase/auth_google.go:28)

Логика:

1. frontend передает `id_token`;
2. backend вызывает `tokeninfo`;
3. проверяет `aud`, `iss`, `sub`, `email`, `email_verified`, `exp`;
4. ищет пользователя по `google_id`;
5. если не находит, ищет по `email` и привязывает `google_id`;
6. если не находит совсем, создает нового пользователя.

Текущее поведение:

- новый Google-пользователь создается с ролью `guest`, а не `student` [auth.go](C:/projects/go/src/myApps/quiz/internal/usecase/auth.go:216)

### 8.6 Текущие security gaps

Для single-node local run дополнительных dependencies не требуется.

Сейчас отсутствует только:

- Redis-backed distributed session cache / revocation layer для multi-node deployment

## 9. Доменная модель

Основные сущности:

- `User`
- `Session`
- `Course`
- `Quiz`
- `Question`
- `Attempt`
- `Enrollment`
- `Certificate`
- `CourseModule`
- `ContentBlock`
- `CourseTest`
- `Review`
- `Notification`
- `Webhook`
- `AuditLog`

Дополнительные core-типы:

- `MultiLangText`
- `AppError`
- enum-ы статусов и ролей

### 9.1 MultiLangText

`MultiLangText` реализует:

- `sql.Scanner`
- `driver.Valuer`

Формат:

```json
{
  "ru": "Русский текст",
  "tj": "Матни тоҷикӣ"
}
```

Проверка обязательных локалей:

- [ValidateRequired](C:/projects/go/src/myApps/quiz/internal/domain/multilang_text.go:45)

## 10. Users

Сущность пользователя описана в [user.go](C:/projects/go/src/myApps/quiz/internal/domain/user.go:1).

Поддерживаются роли:

- `admin`
- `employee`
- `student`
- `guest`

Профиль пользователя может содержать один из вложенных блоков:

- `employee_info`
- `admin_info`
- `student_info`
- `guest_info`

Текущие user-фичи:

- create
- get by id
- list
- update
- deactivate
- lookup by login/email/google id
- link google account

DELETE пользователя это не hard delete, а деактивация.

## 11. Courses

Course-модель описана в [course.go](C:/projects/go/src/myApps/quiz/internal/domain/course.go:1).

Полезные поля:

- `title`
- `description`
- `cover_image_url`
- `category`
- `status`
- `platforms`
- `estimated_minutes`
- `certificate_enabled`
- `certificate_passing_score`
- `reviews_enabled`

Текущая валидация:

- `title` требует `ru` и `tj`
- `status` валидируется
- `platforms` нормализуются и дедуплицируются
- `estimated_minutes`, если задан, должен быть `> 0`
- `certificate_passing_score` должен быть в диапазоне `0..100`

Удаление курса реализовано как archive через `status = archived`.

## 12. Quizzes и Questions

Quiz-модель описана в [quiz.go](C:/projects/go/src/myApps/quiz/internal/domain/quiz.go:1).

Поддерживаемые question types:

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

Текущая логика create/update quiz:

- `title` требует `ru` и `tj`
- `status` валидируется
- `platforms` валидируются
- `time_limit_minutes`, если задан, должен быть `> 0`
- `passing_score` должен быть `0..100`
- `max_attempts <= 0` на create заменяется на `3`
- хотя бы один вопрос обязателен
- позиции вопросов должны быть уникальны
- если `question.points <= 0`, backend ставит `1`
- если `question.config` пустой, backend подставляет `{}`

Repository работает транзакционно для quiz + questions:

- [quizzes.go](C:/projects/go/src/myApps/quiz/internal/repository/postgres/quizzes.go:24)

## 13. Attempts

Attempt-домен описан в [attempt.go](C:/projects/go/src/myApps/quiz/internal/domain/attempt.go:1).

### 13.1 Что сохраняется

В attempt хранятся:

- `quiz_id`
- `user_id`
- `started_at`
- `finished_at`
- `questions_snapshot`
- `answers_data`
- `total_earned`
- `total_max`
- `score_percent`
- `passed`
- `needs_review`

То есть backend хранит snapshot структуры теста и пользовательских ответов на момент попытки.

### 13.2 Submit flow

Основная логика:

- [Submit](C:/projects/go/src/myApps/quiz/internal/usecase/attempts.go:28)

Шаги:

1. валидировать входные ответы;
2. загрузить quiz;
3. посчитать число прошлых попыток пользователя;
4. сравнить с `quiz.max_attempts`;
5. вычислить score;
6. определить `needs_review`;
7. вычислить `passed`;
8. сохранить attempt.

### 13.3 Автооценка

Сейчас автооценка поддерживает:

- `single_choice`
- `image_choice`
- `multiple_choice`
- `true_false`
- `short_answer`
- `fill_blank`

Оценка реализована в:

- [evaluateAttempt](C:/projects/go/src/myApps/quiz/internal/usecase/attempts.go:187)
- [evaluateQuestion](C:/projects/go/src/myApps/quiz/internal/usecase/attempts.go:213)

### 13.4 Когда выставляется `needs_review`

Если тип вопроса не покрыт автооценкой, код уходит в default branch и считает, что нужна ручная проверка.

Сейчас это касается:

- `long_text`
- `matching`
- `ordering`
- `audio`
- `video`
- `code`

### 13.5 Как считается `passed`

```text
passed = !needsReview && scorePercent >= quiz.PassingScore
```

То есть:

- если `needs_review = true`, попытка автоматически не считается passed;
- итог зависит от `quiz.passing_score`, а не от курса.

### 13.6 Текущий review flow

Для reviewed attempts теперь есть admin endpoint `POST /attempts/{id}/review`.

Текущее поведение:

- review работает только для попыток с `needs_review = true`
- admin может выставить `passed`, `comment` или передать `scores[]`
- backend сохраняет `reviewed_at`, `reviewer_id`, `review_comment`, `manual_passed`, `review_scores`
- `needs_review` снимается

Ограничение:

- `scores[]` нужен только для manual-вопросов

## 14. Enrollments

Enrollment-модель описана в [enrollment.go](C:/projects/go/src/myApps/quiz/internal/domain/enrollment.go:1).

Статусы:

- `active`
- `completed`
- `dropped`

Текущие операции:

- create
- get by id
- list
- complete

Create flow:

- enrollment создается по `course_id + user_id`
- duplicate enrollment запрещен

Complete flow:

- [Complete](C:/projects/go/src/myApps/quiz/internal/usecase/enrollments.go:78)

Текущее поведение:

- endpoint переводит enrollment в completed-состояние;
- после completion backend пытается автоматически выпустить сертификат по course threshold / passed attempt;
- admin по-прежнему может вручную вызвать `POST /certificates` как override.

## 15. Certificates

Certificate-модель описана в [certificate.go](C:/projects/go/src/myApps/quiz/internal/domain/certificate.go:1).

### 15.1 Что делает create certificate

Основная логика:

- [Create](C:/projects/go/src/myApps/quiz/internal/usecase/certificates.go:31)

Backend проверяет:

- нет ли уже сертификата для этой пары `enrollment_id + attempt_id`;
- существует ли issuance context;
- enrollment должен быть `completed`;
- у курса `certificate_enabled = true`;
- attempt должен быть `passed`;
- user enrollment-а и user attempt-а должны совпадать;
- quiz attempt-а должен быть связан с курсом через `course_tests`.

После этого:

- генерируется `serial_number`;
- генерируется `verify_hash`;
- создается certificate record.

### 15.2 verify endpoint

Публичный verify endpoint:

- `GET /api/v1/certificates/verify/{verifyHash}`

### 15.3 Текущий конфликт логики

В `Course` есть `certificate_passing_score`, и именно он используется как порог выдачи сертификата, если задан. Если порог не задан (`0`), backend fallback-ится на `attempt.passed`, который считается от `quiz.passing_score`.

Следствие:

- `course.certificate_passing_score` теперь является source of truth для выдачи сертификата, когда задан.

### 15.4 Текущий процесс выдачи

Сейчас есть оба сценария:

- основной flow автоматически пытается выпустить сертификат при `POST /enrollments/{id}/complete`;
- admin всё ещё может вручную вызвать `POST /certificates` как override для edge cases.

## 16. Course Modules

CourseModule-модель описана в [course_module.go](C:/projects/go/src/myApps/quiz/internal/domain/course_module.go:1).

Текущие операции:

- create
- get by id
- list by `course_id`
- update
- delete

Сущность простая:

- `course_id`
- `position`
- `title`
- `description`

## 17. Content Blocks

ContentBlock-модель описана в [content_block.go](C:/projects/go/src/myApps/quiz/internal/domain/content_block.go:1).

Block может принадлежать либо:

- курсу (`course_id`)
- модулю (`module_id`)

Одновременно оба поля использовать нельзя.

### 17.1 Типы content blocks

- `text`
- `url`
- `video`
- `photo`
- `file`

### 17.2 Payload validation

Payload schema валидируется централизованно:

- [ValidateContentBlockPayload](C:/projects/go/src/myApps/quiz/internal/domain/content_block_payload.go:44)

Текущие правила:

- `text` требует `content.ru` и `content.tj`
- `url` требует абсолютный `http/https` URL и bilingual `label`
- `video` требует URL и `provider in {direct,youtube}`
- `photo` требует URL, `caption` опционален, но если есть — должны быть обе локали
- `file` требует URL и `filename`

### 17.3 Бизнес-правила

При create/update:

- должен быть ровно один из `course_id` или `module_id`
- `position > 0`
- `title` требует `ru` и `tj`
- payload нормализуется как JSON и валидируется по типу

## 18. Course Tests

Course tests это link-таблица между:

- курсом или модулем
- и quiz

Модель:

- [course_test_link.go](C:/projects/go/src/myApps/quiz/internal/domain/course_test_link.go:1)

Текущее поведение:

- можно привязать quiz либо к курсу, либо к модулю;
- одновременно оба target-а передать нельзя;
- delete сейчас реализован через query params.

Текущий контракт delete:

```text
DELETE /course-tests?course_id=...&quiz_id=...
DELETE /course-tests?module_id=...&quiz_id=...
```

Это не лучший REST-контракт, но это текущая рабочая реализация.

## 19. Reviews

Review-модель описана в [review.go](C:/projects/go/src/myApps/quiz/internal/domain/review.go:1).

Это отзывы на курсы, а не review flow для попыток.

Поддерживаются статусы:

- `pending`
- `approved`
- `rejected`

Текущие операции:

- create review;
- get by id;
- list reviews;
- moderate review.

Правила:

- `course_id` обязателен;
- `rating` должен быть в диапазоне `1..5`;
- admin может перевести review только в `approved` или `rejected`.

## 20. Notifications

Notification-модель описана в [notification.go](C:/projects/go/src/myApps/quiz/internal/domain/notification.go:1).

Поддерживаемые notification types:

- `course.published`
- `certificate.issued`
- `review.approved`
- `enrollment.created`
- `system`

Текущие операции:

- create notification;
- list notifications;
- get by id;
- mark read.

Логика доступа:

- non-admin видит только свои уведомления;
- admin может создавать уведомления и читать любые.

Текущая модель проста:

- никакой async delivery нет;
- это просто БД-уведомления с флагом `read`.

## 21. Webhooks

Webhook-модель описана в [webhook.go](C:/projects/go/src/myApps/quiz/internal/domain/webhook.go:1).

### 21.1 Что уже есть

Реализован CRUD:

- create
- get by id
- list
- update
- delete

Есть поля:

- `events`
- `secret`
- `status`
- `last_triggered_at`
- `last_status_code`
- `last_error`
- `deliveries`
- `failures`

### 21.2 Что уже исправлено

Repository по-прежнему читает `secret` из БД, но HTTP layer теперь маскирует контракт:

- `POST /webhooks` возвращает `secret` один раз в create response
- `GET /webhooks`
- `GET /webhooks/{id}`
- `PUT /webhooks/{id}`

во всех этих read/update ответах `secret` наружу больше не отдается.

### 21.3 Что теперь есть

Теперь после части audit/app events backend пытается доставлять webhook-события:

- ищет активные webhook-и по `event`
- отправляет `POST` на настроенный URL
- добавляет заголовки:
  - `X-LMS-Event`
  - `X-LMS-Delivery-ID`
  - `X-LMS-Timestamp`
  - `X-LMS-Signature`
- подписывает payload через `HMAC-SHA256`
- обновляет `deliveries`, `failures`, `last_triggered_at`, `last_status_code`, `last_error`
- делает retry с коротким exponential backoff

### 21.4 Что все еще отсутствует

Сейчас delivery идёт через durable outbox, который хранится в `audit_logs` и обрабатывается background worker-ом внутри `api` процесса.

Остаётся ограничение:

- worker не вынесен в отдельный сервис/процесс;
- доставка всё ещё зависит от доступности текущего `api` процесса во время выполнения dispatch.

## 22. Audit Logs

AuditLog-модель описана в [audit_log.go](C:/projects/go/src/myApps/quiz/internal/domain/audit_log.go:1).

Текущие операции:

- get by id
- list

Внешний create route убран.

Route открыт для admin:

- `GET /audit-logs`
- `GET /audit-logs/{id}`

Текущее поведение:

- публичного `POST /audit-logs` больше нет
- записи теперь создаются через внутренний `AuditLogger`
- audit уже подключен как side effect к части ключевых мутаций:
  - users
  - courses
  - quizzes
  - attempts
  - enrollments
  - certificates
  - reviews

Следствие:

- audit trail стал заметно надежнее, но еще не покрывает абсолютно все mutation usecase-ы проекта

## 23. База данных

Основные таблицы по текущей схеме:

- `users`
- `sessions`
- `courses`
- `quizzes`
- `questions`
- `course_modules`
- `content_blocks`
- `course_tests`
- `enrollments`
- `attempts`
- `certificates`
- `reviews`
- `notifications`
- `webhooks`
- `audit_logs`

База данных одна: PostgreSQL.

Доступ к БД в runtime идет через `pgxpool`.

## 24. Реальные маршруты

### Public

- `GET /health`
- `GET /api/v1/health`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/google`
- `GET /api/v1/certificates/verify/{verifyHash}`

### Protected

- `GET /auth/me`
- `POST /auth/logout`
- `GET /courses`
- `GET /courses/{courseID}`
- `GET /quizzes`
- `GET /quizzes/{quizID}`
- `POST /quizzes/{quizID}/attempts`
- `GET /attempts`
- `GET /attempts/{attemptID}`
- `GET /enrollments`
- `POST /enrollments`
- `GET /enrollments/{enrollmentID}`
- `GET /certificates`
- `GET /certificates/{certificateID}`
- `GET /course-modules`
- `GET /course-modules/{moduleID}`
- `GET /content-blocks`
- `GET /content-blocks/{blockID}`
- `GET /reviews`
- `POST /reviews`
- `GET /reviews/{reviewID}`
- `GET /notifications`
- `GET /notifications/{notificationID}`
- `POST /notifications/{notificationID}/read`

### Admin only

- все `/users`
- mutation routes для `/courses`
- mutation routes для `/quizzes`
- `POST /enrollments/{enrollmentID}/complete`
- `POST /certificates`
- mutation routes для `/course-modules`
- mutation routes для `/content-blocks`
- все `/course-tests`
- `POST /reviews/{reviewID}/moderate`
- `POST /notifications`
- все `/webhooks`
- все `/audit-logs`

## 25. Что уже реализовано хорошо

- слоистая архитектура без грубого смешения HTTP/SQL;
- raw SQL репозитории читаемы и локализованы по доменам;
- auth и роли уже разведены через middleware;
- Google Sign-In уже существует и встроен в session model;
- snapshots попыток сохраняются;
- content block payload валидируется по типам;
- graceful shutdown и bootstrap flow оформлены аккуратно;
- проект собирается и проходит `go test ./...`.

## 26. Что сейчас реально болит

### 26.1 Security

- нет lockout после серии неверных входов;
- на `/auth/login` уже есть throttling по числу неудачных попыток, но это еще не полноценный distributed rate limit;
- webhook secrets через API read/update response больше не утекают;
- публичный `POST /audit-logs` убран.

### 26.2 Незавершенные product flows

- upload API реализован через `POST /api/v1/uploads` и локальное файловое хранилище;
- review flow для attempts есть, но пока минимальный и без ручного по-вопросного scoring.
- auto certificate issuance уже есть в `enrollments.Complete`, но вручную выдать сертификат через admin API тоже все еще можно;
- audit webhook delivery now uses a durable outbox worker backed by `audit_logs`.

### 26.3 Контрактные и архитектурные долги

- list endpoint-ы возвращают envelope `data/total/limit/offset`;
- `DELETE /course-tests/{courseTestID}` теперь есть, а старый query-param delete ещё оставлен как compatibility;
- `course.certificate_passing_score` теперь используется как source of truth для порога сертификата;
- auth по-прежнему stateful и DB-backed, но уже не делает 2 отдельных lookup на каждый защищенный запрос.

## 27. Чего в проекте сейчас нет

Нет следующих вещей:

- Redis cache
- JWT
- websocket layer

## 28. Практический вывод

Это уже рабочий LMS backend с понятной слоистой структурой и закрытым базовым функционалом:

- auth;
- users;
- courses;
- quizzes;
- attempts;
- enrollments;
- certificates;
- modules;
- content blocks;
- reviews;
- notifications.

Но часть функций пока реализована только до CRUD-уровня и не доведена до полного production-flow:

- webhooks;
- audit logs;
- attempt review.

Если смотреть на проект как на production backend, главные текущие риски такие:

1. неполное покрытие audit side effects по всем мутациям;
2. отсутствие полноценного lockout / distributed rate limiting в auth;
3. review flow без ручного по-вопросного scoring;
4. `DELETE /course-tests` всё ещё использует query params вместо нормального REST path.
