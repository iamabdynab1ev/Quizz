# LMS Arvand — Полная документация

Система дистанционного обучения: видеокурсы + тесты + сертификаты.
Два отдельных приложения — Go-бэкенд и React-фронтенд.

---

## Содержание

1. [Общая архитектура](#1-общая-архитектура)
2. [Бэкенд (Go)](#2-бэкенд-go)
   - [Стек и запуск](#21-стек-и-запуск)
   - [Переменные окружения](#22-переменные-окружения)
   - [Структура директорий](#23-структура-директорий)
   - [База данных — схема таблиц](#24-база-данных--схема-таблиц)
   - [Миграции](#25-миграции)
   - [Domain-модели](#26-domain-модели)
   - [Usecase-слой (бизнес-логика)](#27-usecase-слой-бизнес-логика)
   - [Repository-слой (PostgreSQL)](#28-repository-слой-postgresql)
   - [HTTP-маршруты](#29-http-маршруты)
   - [Аутентификация и авторизация](#210-аутентификация-и-авторизация)
3. [Фронтенд (React)](#3-фронтенд-react)
   - [Стек и запуск](#31-стек-и-запуск)
   - [Структура директорий](#32-структура-директорий)
   - [Роутинг (SPA без React Router)](#33-роутинг-spa-без-react-router)
   - [API-слой](#34-api-слой)
   - [Маппинг данных](#35-маппинг-данных)
   - [AuthContext — управление сессией](#36-authcontext--управление-сессией)
   - [Компоненты и страницы](#37-компоненты-и-страницы)
   - [DashboardPage — оркестратор](#38-dashboardpage--оркестратор)
   - [Двуязычность (ru / tj)](#39-двуязычность-ru--tj)
4. [Ключевые сценарии](#4-ключевые-сценарии)
   - [Регистрация и вход](#41-регистрация-и-вход)
   - [Прохождение теста](#42-прохождение-теста)
   - [Выдача сертификата](#43-выдача-сертификата)
   - [Создание курса и вопросов (admin)](#44-создание-курса-и-вопросов-admin)
5. [Важные технические детали](#5-важные-технические-детали)

---

## 1. Общая архитектура

```
Браузер (React 19, Vite)
        │  HTTP/JSON
        ▼
Go HTTP API  (chi, :9000)
        │
        ├── usecase (бизнес-логика)
        │       │
        └── repository
                │
           PostgreSQL (:5433, база lms_arvand)
```

**Фронтенд** — SPA, работает на порту `4041`, все запросы `/api/*` проксируются на `9000`.

**Бэкенд** — REST API на Go, запускается из `cmd/api/main.go`. Миграции и seed admin применяются автоматически при старте.

---

## 2. Бэкенд (Go)

### 2.1 Стек и запуск

| Компонент | Решение |
|---|---|
| Язык | Go 1.24 |
| HTTP-роутер | `github.com/go-chi/chi/v5` |
| DB-драйвер | `github.com/jackc/pgx/v5` |
| Пароли | `golang.org/x/crypto/bcrypt` |
| UUID | `github.com/google/uuid` |
| Миграции | Встроенный migrator (goose-совместимый SQL) |

```bash
cd c:\projects\go\src\myApps\quiz
# Запуск (читает .env, применяет миграции, создаёт admin)
go run ./cmd/api
```

Три команды в `cmd/`:

| Команда | Назначение |
|---|---|
| `cmd/api` | Основной HTTP-сервер |
| `cmd/dbcreate` | Создание базы данных |
| `cmd/dbseed` | Только seed данных |

---

### 2.2 Переменные окружения

Файл `.env` в корне проекта.

```ini
# Приложение
APP_NAME=QUIZ
APP_ENV=development          # development | production
LOG_LEVEL=DEBUG              # DEBUG | INFO | WARN | ERROR

# HTTP
HTTP_ADDRESS=127.0.0.1:9000
HTTP_CORS_ALLOWED_ORIGINS=http://localhost:4041
HTTP_READ_TIMEOUT=5s
HTTP_WRITE_TIMEOUT=15s
HTTP_SHUTDOWN_TIMEOUT=15s

# Auth
AUTH_SESSION_TTL=8h
AUTH_BCRYPT_COST=12
AUTH_LOGIN_LOCKOUT_ENABLED=false
AUTH_LOGIN_MAX_ATTEMPTS=5
AUTH_LOGIN_ATTEMPT_WINDOW=1m
AUTH_LOGIN_LOCKOUT_SCOPE=identifier_ip  # identifier | ip | identifier_ip
AUTH_PASSWORD_RESET_RETURN_TOKEN=true   # true = возвращает токен в ответе (только для разработки)

# Google OAuth
GOOGLE_CLIENT_ID=<client-id>.apps.googleusercontent.com
AUTH_GOOGLE_DEFAULT_ROLE=student

# Файлы
UPLOADS_DIR=uploads
UPLOAD_MAX_SIZE_MB=20

# PostgreSQL
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5433/lms_arvand?sslmode=disable
PGX_MAX_CONNS=20
PGX_MIN_CONNS=2
PGX_MAX_CONN_LIFETIME=30m
PGX_MAX_CONN_IDLE_TIME=5m

# Миграции и seed
MIGRATE_RUN_ON_START=true
MIGRATIONS_DIR=migrations
SEED_RUN_ON_START=true
SEED_ADMIN_EMAIL=admin@local.test
SEED_ADMIN_PASSWORD=Admin123!
SEED_ADMIN_FIRST_NAME=System
SEED_ADMIN_LAST_NAME=Admin
SEED_ADMIN_IS_SUPER_ADMIN=true
```

---

### 2.3 Структура директорий

```
quiz/
├── cmd/
│   ├── api/           — HTTP-сервер (main.go + log_translate.go)
│   ├── dbcreate/      — создание БД
│   └── dbseed/        — только seed
├── internal/
│   ├── appctx/        — контекст запроса (auth identity)
│   ├── bootstrap/     — migrator, admin seed
│   ├── cache/         — in-memory кэш сессий
│   ├── config/        — загрузка .env
│   ├── domain/        — типы и интерфейсы (entities)
│   ├── handler/http/  — HTTP-хендлеры и роутер
│   │   └── middleware/ — auth, cors, request logger
│   ├── repository/postgres/ — SQL-запросы
│   ├── storage/       — работа с файлами (local)
│   └── usecase/       — бизнес-логика
├── migrations/        — SQL-миграции (00001..00019)
├── uploads/           — загруженные файлы
├── .env
└── go.mod
```

---

### 2.4 База данных — схема таблиц

#### Пользователи

```sql
users
  id UUID PK, email UNIQUE, google_id UNIQUE, password_hash,
  is_admin BOOLEAN, is_super_admin BOOLEAN, is_male BOOLEAN,
  first_name, last_name, patronymic, phone, birth_date DATE,
  city, avatar_url, is_active BOOLEAN, created_at, updated_at

sessions
  token TEXT PK, user_id → users CASCADE,
  ip_address, user_agent, created_at, expires_at

login_attempts
  id PK, identifier TEXT, ip TEXT,
  attempted_at TIMESTAMPTZ, succeeded BOOLEAN

password_reset_tokens
  id PK, user_id → users CASCADE,
  token_hash TEXT UNIQUE, expires_at, used_at, created_at
```

#### Курсы и вопросы

```sql
courses
  id UUID PK,
  title JSONB {ru, tj},           -- обязательно
  description JSONB {ru, tj},
  cover_image_url TEXT,
  video_url TEXT,                  -- YouTube ID или URL
  category TEXT,
  status course_status             -- draft | published | archived
  platforms platform[]             -- web | mobile | telegram
  estimated_minutes INT,
  certificate_enabled BOOLEAN,     -- выдавать сертификат?
  certificate_passing_score INT,
  reviews_enabled BOOLEAN,
  quiz_pass_percent INT,           -- % для прохождения теста (напр. 80)
  quiz_minutes INT,                -- лимит времени теста (0 = без лимита)
  max_attempts INT,                -- макс. попыток (напр. 3)
  retake_cooldown_days INT,        -- дней между пересдачами (0 = без паузы)
  created_by_user_id UUID,
  created_by_name TEXT,
  created_at, updated_at

questions
  id UUID PK,
  course_id → courses CASCADE,
  position INT,
  type question_type               -- single_choice | multiple_choice | true_false |
                                   -- short_answer | long_text | fill_blank | ...
  prompt JSONB {ru, tj},           -- текст вопроса
  points NUMERIC,                  -- баллы за правильный ответ
  required BOOLEAN,
  config JSONB,                    -- варианты ответов:
                                   -- { options: [{id, text:{ru,tj}, is_correct}] }
  created_at

-- Вспомогательные (не используются активно в quiz-фронте)
course_modules  — разбивка на модули
content_blocks  — материалы (text/url/video/photo/file)
```

#### Прохождение курса

```sql
enrollments
  id UUID PK,
  course_id → courses CASCADE,
  user_id → users SET NULL,
  status enrollment_status         -- active | completed | dropped
  enrolled_at, completed_at

enrollment_completed_blocks
  enrollment_id → enrollments CASCADE,
  block_id → content_blocks CASCADE,
  completed_at

attempts
  id UUID PK,
  course_id → courses CASCADE,     -- ON DELETE CASCADE (миграция 00019)
  user_id → users SET NULL,
  started_at, finished_at,
  questions_snapshot JSONB,        -- снимок вопросов на момент сдачи
  answers_data JSONB,              -- ответы студента
  total_earned NUMERIC,            -- набранные баллы
  total_max NUMERIC,               -- максимум баллов
  score_percent NUMERIC,           -- процент
  passed BOOLEAN,
  needs_review BOOLEAN

certificates
  id UUID PK,
  enrollment_id → enrollments CASCADE,
  user_id → users CASCADE,
  course_id → courses CASCADE,     -- ON DELETE CASCADE (миграция 00019)
  attempt_id → attempts CASCADE,   -- ON DELETE CASCADE (миграция 00019)
  serial_number TEXT UNIQUE,       -- формат "000-000-000"
  verify_hash TEXT UNIQUE,         -- для QR-кода
  issued_at, pdf_url
```

#### Экосистема

```sql
reviews        — отзывы (rating 1-5, status: pending|approved|rejected)
notifications  — уведомления пользователей
webhooks       — вебхуки на события платформы
audit_logs     — журнал всех событий (actor_id, payload JSONB)
goose_db_version — таблица версий миграций (не трогать!)
```

---

### 2.5 Миграции

| Файл | Изменение |
|---|---|
| `00001_init_schema.sql` | Все основные таблицы |
| `00002_courses_add_fields.sql` | Дополнительные поля курсов |
| `00003_login_attempts.sql` | Таблица блокировки входа |
| `00004_certificate_fields.sql` | Поля сертификата |
| `00005_audit_webhook_outbox.sql` | Вебхук-очередь в audit_logs |
| `00006_course_tests_add_id.sql` | ID для course_tests |
| `00007_*` — `00012_*` | Поэтапное добавление полей |
| `00013_password_reset_tokens.sql` | Сброс пароля |
| `00014_*` | Дополнения |
| `00015_refactor_roles_gender.sql` | Замена role/gender enum на is_admin/is_male boolean; удалены user_*_info таблицы |
| `00016_simplify_course_quiz.sql` | Квиз встроен в курс: удалены таблицы `quizzes` и `course_tests`, questions переехали под course_id |
| `00017_*` — `00018_*` | Мелкие правки |
| `00019_hard_delete_cascade.sql` | FK attempts/certificates → CASCADE для полного удаления курсов |

---

### 2.6 Domain-модели

Файлы в `internal/domain/`. Ключевые типы:

```go
// Многоязычный текст
type MultiLangText struct {
    RU string `json:"ru"`
    TJ string `json:"tj"`
}

// Статусы
type CourseStatus  string  // draft | published | archived
type Platform      string  // web | mobile | telegram
type QuestionType  string  // single_choice | multiple_choice | ...

// Курс
type Course struct {
    ID, Title, Description, VideoURL, CoverImageURL, Category
    Status, Platforms, CertificateEnabled
    QuizPassPercent, QuizMinutes, MaxAttempts, RetakeCooldownDays
    Questions []Question
    CreatedAt, UpdatedAt
}

// Вопрос
type Question struct {
    ID, Position, Type, Prompt, Points, Required
    Config json.RawMessage  // { options: [{id, text, is_correct}] }
    CreatedAt
}

// Попытка
type Attempt struct {
    ID, CourseID, UserID
    TotalEarned, TotalMax, ScorePercent float64
    Passed bool
    StartedAt, FinishedAt
}

// Сертификат
type Certificate struct {
    ID, EnrollmentID, UserID, CourseID, AttemptID
    SerialNumber, VerifyHash
    IssuedAt
}

// Идентификатор запроса (из JWT)
type AuthIdentity struct {
    User    SessionUser
    Session Session
}
```

---

### 2.7 Usecase-слой (бизнес-логика)

Каждый usecase принимает repository-интерфейсы через конструктор. Зависимости инжектируются в `cmd/api/main.go`.

| Usecase | Методы |
|---|---|
| `AuthUseCase` | Register, Login, GoogleLogin, Me, Logout, UpdateProfile, ChangePassword, ForgotPassword, ResetPassword |
| `UserUseCase` | Create, GetByID, List, Update, Deactivate (→ hard DELETE) |
| `CourseUseCase` | Create, GetByID, List, Update, Archive (→ hard DELETE) |
| `AttemptUseCase` | Submit, GetByID, List + авто-выдача сертификата через tryAutoIssueCertificate |
| `CertificateUseCase` | Create, GetByID, GetByVerifyHash, List, TryAutoIssueForEnrollment |
| `EnrollmentUseCase` | Create, GetByID, List, Complete |
| `DashboardUseCase` | GetPublic, GetForUser, GetAdmin |
| `ReviewUseCase` | Create, GetByID, List, Moderate |
| `NotificationUseCase` | Create, GetByID, List, MarkRead |
| `UploadUseCase` | Upload (валидация mime-type и размера) |
| `WebhookUseCase` | Create, GetByID, List, Update, Delete |
| `AuditLogUseCase` | List, GetByID |
| `CourseModuleUseCase` | Create, GetByID, List, Update, Delete |
| `ContentBlockUseCase` | Create, GetByID, List, Update, Delete |
| `AuditWebhookOutboxWorker` | Фоновый процесс: доставка вебхуков из audit_logs |

**Цепочка автовыдачи сертификата:**

```
Submit attempt
  → evaluateAttempt (score)
  → CreateAttempt (save)
  → tryAutoIssueCertificate
      → enrollmentLookup.GetLatestByCourseAndUser
      → TryAutoIssueForEnrollment
          → FindAutoIssueCandidate (passed attempt + certificate_enabled=true)
          → CertificateUseCase.Create (serial number + verify hash)
```

---

### 2.8 Repository-слой (PostgreSQL)

Все репозитории в `internal/repository/postgres/`. Используют `pgxpool.Pool`.

**Ключевые особенности:**

- `replaceQuestions()` — при каждом обновлении курса: DELETE все вопросы + INSERT заново (UUID вопросов меняются!)
- `scanCourse()` / `scanQuestionRow()` — row scanners для pgx
- `sanitizeQuestionConfig()` — срезает `is_correct` из config для публичного API
- Session cache (`internal/cache/`) — in-memory кэш на `AUTH_SESSION_CACHE_TTL` чтобы не ходить в БД на каждый запрос

---

### 2.9 HTTP-маршруты

Все маршруты в `internal/handler/http/router.go`. Базовый путь `/api/v1`.

#### Публичные (без авторизации)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/health` | Healthcheck |
| POST | `/auth/register` | Регистрация |
| POST | `/auth/login` | Вход |
| GET | `/auth/google/config` | Google OAuth конфиг |
| POST | `/auth/google` | Вход через Google |
| POST | `/auth/password/forgot` | Запрос сброса пароля |
| POST | `/auth/password/reset` | Сброс пароля по токену |
| GET | `/courses` | Список опубликованных курсов |
| GET | `/courses/{courseID}` | Детали курса |
| GET | `/quizzes/{quizID}` | Тест без ответов |
| GET | `/certificates/verify/{verifyHash}` | Верификация сертификата |
| GET | `/certificates/{certificateID}` | Публичный сертификат |

#### Защищённые (требуют Bearer token)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/auth/me` | Текущий пользователь |
| PUT | `/auth/me` | Обновить профиль |
| POST | `/auth/password/change` | Сменить пароль |
| POST | `/auth/logout` | Выход |
| GET | `/quizzes` | Список тестов |
| GET | `/quizzes/{quizID}` | Тест |
| GET | `/attempts` | История попыток |
| GET | `/attempts/{attemptID}` | Попытка по ID |
| POST | `/courses/{courseID}/attempts` | Отправить ответы теста |
| GET | `/enrollments` | Записи на курсы |
| POST | `/enrollments` | Записаться на курс |
| GET | `/enrollments/{enrollmentID}` | Запись |
| GET | `/certificates` | Сертификаты пользователя |
| GET | `/notifications` | Уведомления |
| POST | `/notifications/{id}/read` | Прочитать уведомление |

#### Только для Admin (`RequireAdmin`)

| Метод | Путь | Описание |
|---|---|---|
| POST | `/courses` | Создать курс |
| PUT | `/courses/{courseID}` | Обновить курс |
| DELETE | `/courses/{courseID}` | Удалить курс (hard delete) |
| POST | `/quizzes` | Создать тест |
| PUT | `/quizzes/{quizID}` | Обновить тест |
| DELETE | `/quizzes/{quizID}` | Удалить тест |
| GET | `/quizzes/{quizID}/answers` | Тест С ответами |
| POST | `/uploads` | Загрузить файл |
| GET/POST | `/webhooks` | Вебхуки |
| PUT/DELETE | `/webhooks/{id}` | Управление вебхуком |
| GET | `/audit-logs` | Журнал событий |
| GET | `/dashboard/admin` | Статистика для админа |
| POST/PUT/DELETE | `/course-modules` | Модули |
| POST/PUT/DELETE | `/content-blocks` | Контент-блоки |
| POST | `/enrollments/{id}/complete` | Завершить запись |
| POST | `/certificates` | Выдать сертификат вручную |
| POST | `/reviews/{id}/moderate` | Модерировать отзыв |
| POST | `/notifications` | Создать уведомление |

#### Только для Super Admin (`RequireSuperAdmin`)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/users` | Список пользователей |
| POST | `/users` | Создать пользователя |
| GET | `/users/{userID}` | Пользователь по ID |
| PUT | `/users/{userID}` | Обновить пользователя |
| DELETE | `/users/{userID}` | Удалить пользователя (hard delete) |

---

### 2.10 Аутентификация и авторизация

**Сессионные токены** (не JWT) — случайный token хранится в таблице `sessions`.

```
POST /auth/login
  → проверка пароля (bcrypt)
  → лимит попыток (login_attempts)
  → создание session record
  → возврат { token, expires_at, user }
```

**Bearer token** в заголовке: `Authorization: Bearer <token>`

**Уровни доступа:**
- `guest` — публичные маршруты
- `authenticated` — авторизованные маршруты (любая роль)
- `admin` (`is_admin = true`) — управление контентом
- `super_admin` (`is_super_admin = true`) — управление пользователями

**Password reset flow:**
```
POST /auth/password/forgot { email }
  → генерация token, запись в password_reset_tokens
  → (в dev) возвращает token в ответе (AUTH_PASSWORD_RESET_RETURN_TOKEN=true)
  → (в prod) отправляет email (не реализовано — только в БД)

POST /auth/password/reset { token, password }
  → проверка token_hash + expires_at
  → UPDATE users SET password_hash
  → пометить token как использованный
```

---

## 3. Фронтенд (React)

### 3.1 Стек и запуск

| Компонент | Решение |
|---|---|
| Фреймворк | React 19.2.5 |
| Сборка | Vite 8.0.10 |
| Компилятор | babel-plugin-react-compiler |
| Иконки | lucide-react |
| PDF | jspdf + html2canvas |
| QR-коды | qrcode.react |

```bash
cd c:\projects\go\src\myApps\videocourses
npm install
npm run dev    # порт 4041
```

`vite.config.js` — proxy:
```js
proxy: {
  '/api': 'http://127.0.0.1:9000',
  '/uploads': 'http://127.0.0.1:9000',
}
```

---

### 3.2 Структура директорий

```
videocourses/src/
├── api/
│   ├── client.js       — HTTP-клиент (fetch + auth + error handling)
│   ├── index.js        — все API-функции (authApi, coursesApi, ...)
│   └── mappers.js      — нормализация ответов бэкенда
├── context/
│   └── AuthContext.jsx — сессия, signIn/signOut/signUp/updateProfile
├── hooks/
│   ├── useAuth.js      — доступ к AuthContext
│   ├── usePathname.js  — SPA-роутинг через pushState + custom event
│   └── useClickOutside.js
├── components/
│   ├── app/
│   │   └── AppContent.jsx  — главный роутер
│   ├── auth/
│   │   ├── LoginForm.jsx
│   │   ├── SignupStub.jsx
│   │   ├── GoogleSignInButton.jsx
│   │   ├── ForgotPasswordForm.jsx
│   │   └── ResetPasswordForm.jsx
│   ├── dashboard/
│   │   ├── AdminOverview.jsx  — админ-панель (CRUD курсов и вопросов)
│   │   ├── UserOverview.jsx
│   │   └── RoleHero.jsx
│   ├── layout/
│   │   ├── PublicLayout.jsx   — шапка/подвал для гостей
│   │   ├── ProtectedLayout.jsx — шапка/подвал для авторизованных
│   │   └── ProfileMenu.jsx
│   ├── certificate/
│   │   └── CertificateQr.jsx — QR-код на странице сертификата
│   └── icons/
├── pages/
│   ├── DashboardPage.jsx   — ОРКЕСТРАТОР (загрузка данных + роутинг страниц)
│   ├── MainPage.jsx        — главная (каталог курсов + история попыток)
│   ├── CourseDetailPage.jsx — страница курса (видео + инфо о тесте)
│   ├── TestPage.jsx        — прохождение теста (вопрос за вопросом)
│   ├── TestResultPage.jsx  — результат теста
│   ├── ProfilePage.jsx     — профиль пользователя
│   ├── CertificatePage.jsx — сертификат с PDF-экспортом
│   └── AuthPage.jsx        — страница входа/регистрации
└── data/
    ├── translations.js     — все строки интерфейса (ru + tj)
    └── tajikistanLocations.js — города/районы для выбора города
```

---

### 3.3 Роутинг (SPA без React Router)

Роутинг реализован вручную через `window.history.pushState` + `CustomEvent`:

```js
// usePathname.js — useSyncExternalStore подписывается на pathname
const navigateTo = (path, options) => {
  window.history.pushState({}, '', path)
  window.dispatchEvent(new CustomEvent('pathname-change'))
}
```

`DashboardPage` получает текущий `pathname` и рендерит нужную страницу через `if/else if`.

**Таблица маршрутов:**

| URL | Компонент | Условие |
|---|---|---|
| `/` | MainPage | default |
| `/profile` | ProfilePage | isAuthenticated |
| `/admin` | AdminOverview | isAuthenticated + admin |
| `/courses/:id` | CourseDetailPage | курс найден |
| `/courses/:id/test` | TestPage | курс найден + authenticated |
| `/courses/:id/test/result` | TestResultPage | есть результат |
| `/courses/:id/certificate/:hash` | CertificatePage | есть сертификат |
| `/login` | AuthPage | не authenticated (в AppContent) |
| `/forgot-password` | ForgotPasswordForm | (в AppContent) |
| `/reset-password` | ResetPasswordForm | (в AppContent) |

---

### 3.4 API-слой

`src/api/client.js` — базовый fetch-клиент:
```js
apiRequest(path, { method, body, query, auth, signal })
  → добавляет Authorization: Bearer <token> если auth=true (дефолт)
  → при 401 → clearSession() (автологаут)
  → при ошибке → бросает { status, payload } объект
```

`src/api/index.js` — функции по сущностям:

```js
authApi.login(identifier, password)
authApi.register(payload)         // → auto-login
authApi.googleLogin(idToken)
authApi.me()
authApi.logout()
authApi.updateMe(payload)
authApi.changePassword(payload)
authApi.forgotPassword({ email })
authApi.resetPassword({ token, password })

usersApi.list(params)             // super admin only
usersApi.get(id) / create / update / remove

coursesApi.list(params, options)
coursesApi.get(id, options)       // options.auth=false для публичного
coursesApi.create / update / remove

quizzesApi.list / get / create / update / remove
quizzesApi.getWithAnswers(id)     // GET /quizzes/:id/answers — только admin

attemptsApi.list(params)
attemptsApi.get(id)
attemptsApi.submit(courseId, { started_at, answers })

enrollmentsApi.list / get / create / complete

certificatesApi.list / get / create
certificatesApi.verify(verifyHash) // публичный, auth: false

dashboardApi.getAdminStats()
```

---

### 3.5 Маппинг данных

`src/api/mappers.js` нормализует ответы бэкенда в единый фронтенд-формат.

```js
mapUser(rawUser) → { id, name, firstName, lastName, patronymic, email,
                     role, gender, birthday, phone, city, isActive }

mapSessionUser(rawUser) → { ...mapUser, isSuperAdmin, hasPassword }

mapCourse(rawCourse) → { id, title:{ru,tj}, descriptionText:{ru,tj},
                         videoId, coverImageUrl, status, platforms,
                         quizPassPercent, maxAttempts, quizId, ... }

mapQuiz(rawQuiz) → { id, title, testItems: [mapQuizQuestion()] }

mapQuizQuestion(rawQuestion) → {
  id,
  correctOption: findIndex(o => o.is_correct),  // -1 если sanitized (студент)
  optionIds: options.map(o => o.id),             // нужны для submit
  options: { ru: [...], tj: [...] },             // тексты вариантов
  question: {ru, tj}                             // текст вопроса
}

mapAttempt(rawAttempt) → { totalEarned, totalMax, scorePercent, passed,
                            courseId, certificateId, tryNumber, ... }

mapCertificate(rawCertificate) → { id, courseId, userName,
                                    certificateNumber, verifyHash, issuedDate }
```

---

### 3.6 AuthContext — управление сессией

```jsx
// src/context/AuthContext.jsx
const AuthContext = {
  isReady,       // false пока идёт проверка сессии при монтировании
  session,       // null | SessionUser
  signIn,        // email + password → устанавливает session
  signInWithGoogle,
  signOut,
  signUp,        // регистрация → устанавливает session
  updateProfile, // PUT /auth/me → обновляет session
  changePassword,
}
```

**LocalStorage ключи:**
- `videocourses-auth-token` — Bearer-токен
- `videocourses-session` — JSON объект пользователя

**При монтировании:**
1. Читает token из localStorage
2. Вызывает `authApi.me()` для верификации
3. Устанавливает `session` или `null`
4. Устанавливает `isReady = true`

---

### 3.7 Компоненты и страницы

#### AppContent.jsx
Верхний роутер. До того как DashboardPage решит что рендерить, AppContent обрабатывает:
- `/login` → `<AuthPage>` (если не авторизован)
- `/forgot-password` → `<ForgotPasswordForm>`
- `/reset-password?token=...` → `<ResetPasswordForm>`
- Всё остальное → `<DashboardPage>`

#### DashboardPage.jsx (2000+ строк)
Главный оркестратор — подробно в разделе 3.8.

#### MainPage.jsx
- Список доступных курсов (`availableCourseViewModels` — активные без сертификата)
- Список пройденных курсов (`certificateEntries`)
- История попыток с пагинацией (`pagedTryEntries`, ATTEMPTS_PER_PAGE=10)

#### CourseDetailPage.jsx
- YouTube embed (парсит ID из любого формата ссылки)
- Инфо о тесте: кол-во вопросов, варианты, процент прохождения, макс. попыток
- Кнопка «Пройти тест» → `onOpenTest(id)`
- Если уже есть сертификат → кнопка «Показать сертификат»

#### TestPage.jsx
- Вопросы по одному (`currentIndex`)
- Прогресс-бар (currentIndex+1 / total)
- Выбор варианта ответа (single choice, radio-подобно)
- На последнем вопросе кнопка «Завершить» → `onComplete(payload)`
- `isSubmitting` flag предотвращает двойную отправку

**Формат ответов при submit:**
```js
answers: course.testItems
  .filter(item => answers[item.id] !== undefined)
  .map(item => ({
    question_id: item.id,                              // UUID вопроса из БД
    selected_option_ids: [item.optionIds[answers[item.id]]]  // ID варианта из config
  }))
```

#### TestResultPage.jsx
- `totalEarned / totalQuestions`, `scorePercent%`
- Если `passed` → «Получить сертификат» + «На главную»
- Если `failed` → «Повторить тест» + «К курсу» + «На главную»

#### ProfilePage.jsx
- Просмотр/редактирование: имя, фамилия, отчество, телефон, город, пол, дата рождения
- Email — только чтение (нельзя менять через профиль)
- Смена пароля: current_password + new_password + confirm

#### CertificatePage.jsx
- Отображение сертификата (имя, курс, дата, серийный номер)
- QR-код со ссылкой `/api/v1/certificates/verify/:verifyHash`
- Кнопка «Скачать PDF» (jspdf + html2canvas)

#### AdminOverview.jsx
- Таблица курсов с CRUD
- Для каждого курса — список вопросов
- Форма создания/редактирования вопроса:
  - Текст (ru + tj)
  - 3 варианта ответа (ru + tj)
  - Radio — какой вариант правильный (`correctOption: '0'|'1'|'2'`)

---

### 3.8 DashboardPage — оркестратор

#### Состояния

```js
const [courses, setCourses]           // []Course
const [quizzes, setQuizzes]           // []Quiz (с testItems)
const [attempts, setAttempts]         // []Attempt
const [certificates, setCertificates] // []Certificate
const [enrollments, setEnrollments]   // []Enrollment
const [submittedResults, setSubmittedResults] // {[courseId]: Attempt}
const [isLoading, setIsLoading]
const [loadError, setLoadError]
const [isOpeningCertificate, setIsOpeningCertificate]
```

#### Refs (синхронный доступ в callbacks)
```js
coursesRef, quizzesRef, enrollmentsRef
```

#### Computed values
```js
courseViewModels   — Course + Quiz → объединённая view-модель
coursesById        — Map<id, Course>
quizzesById        — Map<id, Quiz>
activeCoursesById  — только published
userAttempts       — попытки текущего пользователя
latestResultsByCourse — последний attempt по каждому курсу
selectedResult     — submittedResults[courseId] || latestResultsByCourse[courseId]
certificateEntries — список { course, issuedDate, certificateNumber, verifyHash }
passedCourseIds    — Set<courseId> с сертификатом
availableCourseViewModels — активные курсы без сертификата
```

#### Загрузка данных

**Основной эффект** (зависит от pathname-флагов `needsQuizzes`, `needsAttempts`, и т.д.):
- `Promise.allSettled([courses, quizzes, attempts, certificates])`
- Для admin: `quizzesApi.getWithAnswers()` (с `is_correct`)
- Для студента: `quizzesApi.get()` (sanitized)
- `hydrateQuizDetails()` — загружает полные данные для каждого quiz ID

**`loadCourseQuiz` эффект** (зависит от `courseId`):
- Запасной загрузчик если тест не попал в основной эффект
- Для admin: `getWithAnswers`, для студента: `get`
- Не перезаписывает уже загруженный quiz

**`refreshLearningContent()`** — вызывается после CRUD операций, перезагружает курсы и тесты с правильным fetcher'ом.

#### Ключевые хендлеры

```js
handleOpenTest(id)
  → если не авторизован: сохраняет путь, редирект на /login
  → если есть сертификат: редирект на /courses/:id
  → enrollmentsApi.create (409 = уже записан, ок)
  → navigateTo(/courses/:id/test)

handleTestComplete(courseId, payload)
  → attemptsApi.submit(courseId, payload)
  → flushSync: setSubmittedResults + setAttempts (ВАЖНО: до navigateTo!)
  → если passed: ищет сертификат через enrollment
  → navigateTo(/courses/:id/test/result)
  → 409: navigateTo result (показывает старый результат)

handleOpenCertificate(courseId)
  → ищет enrollment
  → certificatesApi.list({enrollment_id, course_id, user_id})
  → navigateTo(/courses/:id/certificate/:verifyHash)

handleCreateQuestion / handleUpdateQuestion / handleDeleteQuestion
  → buildQuizPayload(course, nextQuiz)
  → quizzesApi.update(quizId, payload)
  → refreshLearningContent()
```

#### buildQuizQuestionPayload

```js
// Формирует payload вопроса для сохранения в БД
{
  position, type, prompt, points, required,
  config: {
    options: [
      { id: question.optionIds[i] || `${baseId}-option-${i+1}`,
        text: { ru, tj },
        is_correct: Number(question.correctOption) === i }
    ]
  }
}
```

---

### 3.9 Двуязычность (ru / tj)

Все строки интерфейса в `src/data/translations.js`:
```js
const translations = {
  ru: { auth: { login: 'Войти', ... }, courses: { title: 'Курсы', ... }, ... },
  tj: { auth: { login: 'Даромадан', ... }, ... }
}
```

Текущий язык хранится в `useState('ru')` в `AppContent` / `DashboardPage` и передаётся через props.

Все многоязычные данные с бэкенда — `{ ru: '...', tj: '...' }`, рендерятся как `text[language]`.

---

## 4. Ключевые сценарии

### 4.1 Регистрация и вход

```
[Пользователь] → /login → AuthPage
  → SignupStub (многошаговая форма)
  → authApi.register({ first_name, last_name, email, password, ... })
  → POST /auth/register
  → бэкенд создаёт user + session
  → AuthContext.signUp → сохраняет token + session в localStorage
  → AppContent видит session → редирект на сохранённый путь или /
```

### 4.2 Прохождение теста

```
[Пользователь] → CourseDetailPage → «Пройти тест»
  → handleOpenTest(courseId)
  → POST /enrollments { course_id }   (или 409 если уже записан)
  → navigateTo(/courses/:id/test)

[DashboardPage рендерит TestPage]
  → quiz уже в state (загружен hydrateQuizDetails или loadCourseQuiz)
  → quiz НЕ содержит is_correct (sanitized для студентов)
  → TestPage показывает вопросы

[Студент отвечает] → handleNext() на последнем
  → onComplete({ startedAt, answers: [{question_id, selected_option_ids}] })

[DashboardPage handleTestComplete]
  → POST /courses/:id/attempts { started_at, answers }
  → Бэкенд: evaluateAttempt (сравнивает option_ids с is_correct в config)
  → Бэкенд: CreateAttempt
  → Бэкенд: tryAutoIssueCertificate (если passed + enrolled + certificate_enabled)
  → Ответ: { total_earned, total_max, score_percent, passed }
  → flushSync → setSubmittedResults → setAttempts
  → navigateTo(/courses/:id/test/result)
```

### 4.3 Выдача сертификата

```
[TestResultPage — passed] → «Получить сертификат» → handleOpenCertificate

[handleOpenCertificate]
  → findEnrollmentForCourse (из ref или GET /enrollments)
  → GET /certificates?enrollment_id=...&course_id=...&limit=1
  → Сертификат уже создан бэкендом в tryAutoIssueCertificate
  → setCertificates (кэш)
  → navigateTo(/courses/:id/certificate/:verifyHash)

[CertificatePage]
  → Отображает имя, курс, дату, серийный номер
  → QR-код → /api/v1/certificates/verify/:verifyHash
  → «Скачать PDF» → html2canvas → jspdf
```

### 4.4 Создание курса и вопросов (admin)

```
[Admin] → /admin → AdminOverview
  → Форма нового курса: заголовок (ru+tj), описание, видео URL, настройки теста
  → handleCreateCourse → POST /courses { title, description, ... }
  → refreshLearningContent

[Admin] → «Добавить вопрос»
  → форма: текст вопроса (ru+tj), 3 варианта (ru+tj), radio правильного
  → handleCreateQuestion
  → buildQuizPayload (все вопросы курса → config с is_correct)
  → PUT /quizzes/:id { questions: [...] }
  → бэкенд: replaceQuestions (DELETE + INSERT, новые UUID!)
  → refreshLearningContent → getWithAnswers → корректный correctOption в state
```

---

## 5. Важные технические детали

### Проблема race condition (исправлена)

`loadCourseQuiz` и основной `loadData` запускаются одновременно при монтировании. `loadCourseQuiz` раньше всегда использовал `get()` (sanitized), что перезаписывало admin-версию с `is_correct`. Теперь для admin используется `getWithAnswers`.

### `certificate_enabled` (исправлено)

`quizRequest.toUpdateParams()` не устанавливал `CertificateEnabled`, Go-дефолт `false` записывался в БД при каждом обновлении теста. Теперь явно `CertificateEnabled: true`.

### `replaceQuestions` — UUIDs меняются

При каждом PUT на тест все вопросы пересоздаются с новыми UUID. Поэтому `optionIds` хранятся внутри `config.options[].id` (а не как внешние ключи) — они не зависят от UUID вопроса.

### `flushSync` перед navigateTo

`usePathname` использует `useSyncExternalStore`, который срабатывает синхронно при `pushState`. Если просто вызвать `setState` + `navigateTo`, React батчит состояние и перерисовка происходит уже после навигации с пустым `selectedResult`. `flushSync` форсирует синхронный flush состояния до навигации.

### Удаление (hard delete)

После миграции `00019`:
- Удаление курса → каскадно удаляет вопросы, enrollments, attempts, certificates
- Удаление пользователя → каскадно удаляет сессии, сертификаты, уведомления

### Очистка БД (разработка)

```sql
TRUNCATE TABLE
  audit_logs, webhooks, notifications, reviews, certificates, attempts,
  enrollment_completed_blocks, enrollments, questions,
  content_blocks, course_modules, courses,
  sessions, users, login_attempts, password_reset_tokens
RESTART IDENTITY CASCADE;
-- Перезапустить бэкенд → admin пересоздастся через seed
```
