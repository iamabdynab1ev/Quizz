# QUIZ Frontend API Contract

Документ для frontend-разработчика. Описывает текущий API backend `QUIZ`, какие поля отправлять, какие поля получать и какие бизнес-правила учитывать.

## Base URL

Локально без nginx:

```text
http://localhost:9000/api/v1
```

Через Vite proxy или nginx на одном origin:

```text
/api/v1
```

Health:

```text
GET /health
GET /api/v1/health
```

Uploads:

```text
GET /uploads/{file}
```

## Авторизация

Backend использует session token, не JWT.

После login frontend сохраняет `token` и отправляет его во всех protected запросах:

```http
Authorization: Bearer <token>
```

Первый admin для входа берётся из `.env`:

```text
SEED_ADMIN_EMAIL=admin@local.test
SEED_ADMIN_PASSWORD=Admin123!
```

## Формат пользователя

Backend наружу отдаёт упрощённый user:

```json
{
  "id": "uuid",
  "email": "admin@local.test",
  "is_admin": true,
  "is_super_admin": true,
  "first_name": "System",
  "last_name": "Admin",
  "phone": "999999999",
  "is_male": false,
  "city": "Худжанд",
  "birth_date": "2001-07-12",
  "created_at": "2026-05-06T14:08:12Z",
  "updated_at": "2026-05-06T14:08:12Z"
}
```

Важно:

- `username` не использовать, его нет в response.
- `role` не использовать на frontend, вместо него `is_admin`.
- `is_super_admin` приходит только когда true.
- `birth_date` приходит только если заполнен.
- `is_active` не приходит.
- `is_male=true` значит мужчина, `false` значит женщина/не мужчина.

## Ошибки

Одна ошибка поля:

```json
{
  "field": "quiz_pass_percent",
  "code": "out_of_range",
  "message": "Значение проходного процента должно быть от 0 до 100"
}
```

Несколько ошибок формы:

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

Общая ошибка:

```json
{
  "error": "conflict",
  "message": "Лимит попыток исчерпан. Повторная сдача будет доступна после 05.06.2026."
}
```

Frontend должен показывать пользователю `message`. Если есть `field`, подсветить конкретный input. Если есть `fields`, подсветить все поля из массива.

## List responses

Большинство list endpoint-ов возвращает envelope:

```json
{
  "data": [],
  "total": 0,
  "limit": 20,
  "offset": 0
}
```

Frontend должен читать список из `data`, а не ожидать массив напрямую.

## Auth endpoints

### POST `/auth/register`

Public self-signup.

Body:

```json
{
  "email": "student@local.test",
  "password": "Student123!",
  "first_name": "Ali",
  "last_name": "Karimov",
  "phone": "900000000",
  "is_male": true,
  "city": "Худжанд",
  "birth_date": "2001-07-12"
}
```

Response: как login.

### POST `/auth/login`

Body:

```json
{
  "email": "admin@local.test",
  "password": "Admin123!"
}
```

`identifier` ещё принимается для совместимости, но новый frontend должен отправлять `email`.

Response:

```json
{
  "token": "session-token",
  "expires_at": "2026-05-06T22:11:02Z",
  "user": {
    "id": "uuid",
    "email": "admin@local.test",
    "is_admin": true,
    "is_super_admin": true,
    "first_name": "System",
    "last_name": "Admin",
    "is_male": false,
    "created_at": "2026-05-06T14:08:12Z",
    "updated_at": "2026-05-06T14:08:12Z"
  }
}
```

### GET `/auth/google/config`

Response:

```json
{
  "enabled": true,
  "client_id": "497804149487-...apps.googleusercontent.com"
}
```

Если `enabled=false`, кнопку Google лучше скрыть.

### POST `/auth/google`

Body:

```json
{
  "id_token": "google-id-token-from-frontend"
}
```

Response: такой же, как login.

Frontend должен получать `id_token` через Google SDK. Backend сам не открывает Google popup.

### GET `/auth/me`

Protected.

Response:

```json
{
  "user": {},
  "session": {
    "token": "session-token",
    "user_id": "uuid",
    "ip_address": "127.0.0.1",
    "user_agent": "browser",
    "created_at": "2026-05-06T14:11:02Z",
    "expires_at": "2026-05-06T22:11:02Z"
  }
}
```

### PUT `/auth/me`

Protected. Пользователь редактирует только свой профиль.

Body:

```json
{
  "first_name": "Ali",
  "last_name": "Karimov",
  "phone": "900000000",
  "is_male": true,
  "city": "Худжанд",
  "birth_date": "2001-07-12",
  "avatar_url": "/uploads/avatar.png"
}
```

### POST `/auth/password/change`

Protected.

Body:

```json
{
  "current_password": "Old123!",
  "new_password": "New123!"
}
```

### POST `/auth/password/forgot`

Public.

Body:

```json
{
  "email": "student@local.test"
}
```

Local testing response may include `reset_token` if `AUTH_PASSWORD_RESET_RETURN_TOKEN=true`.

### POST `/auth/password/reset`

Public.

Body:

```json
{
  "token": "reset-token",
  "new_password": "New123!"
}
```

### POST `/auth/logout`

Protected. Response `204`.

## Courses

### GET `/courses`

Public. Токен не нужен: frontend может показывать каталог до входа пользователя.

Query:

- `search`
- `status`
- `category`
- `platform`
- `limit`
- `offset`

Response item:

```json
{
  "id": "uuid",
  "title": { "ru": "Язык Go", "tj": "Забони Go" },
  "description": { "ru": "Описание", "tj": "Тавсиф" },
  "cover_image_url": "/uploads/cover.png",
  "video_url": "https://youtube.com/watch?v=...",
  "category": "programming",
  "estimated_minutes": 40,
  "quiz_pass_percent": 80,
  "quiz_minutes": 30,
  "max_attempts": 3,
  "retake_cooldown_days": 0,
  "certificate_enabled": true,
  "created_at": "2026-05-06T14:08:12Z",
  "updated_at": "2026-05-06T14:08:12Z"
}
```

Скрыты: `status`, `platforms`, `certificate_passing_score`, `reviews_enabled`.

### GET `/courses/{courseID}`

Public. Токен не нужен: frontend может открыть детальную страницу курса до входа пользователя.

Response такой же, как item в `GET /courses`.

### POST `/courses`

Admin.

Body:

```json
{
  "title": { "ru": "Язык Go", "tj": "Забони Go" },
  "description": { "ru": "Описание", "tj": "Тавсиф" },
  "cover_image_url": "/uploads/cover.png",
  "video_url": "https://youtube.com/watch?v=w_PlmLNWuSU",
  "category": "programming",
  "estimated_minutes": 40
}
```

### PUT `/courses/{courseID}`

Admin. Body такой же, как create.

### DELETE `/courses/{courseID}`

Admin. Текущая реализация делает hard delete. Связанные questions/attempts/certificates удаляются каскадом после миграции `00019_hard_delete_cascade.sql`.

## Quizzes

### GET `/quizzes`

Protected. Это compatibility endpoint: отдельной таблицы `quizzes` больше нет, backend возвращает курсы в формате quiz. Для публичной страницы курса можно использовать `course.id` как `quizID`.

Response item:

```json
{
  "id": "uuid",
  "title": { "ru": "Тест Go", "tj": "Тести Go" },
  "description": { "ru": "Описание", "tj": "Тавсиф" },
  "course_id": "uuid",
  "time_limit_minutes": 30,
  "passing_score": 80,
  "max_attempts": 3,
  "retake_cooldown_days": 0,
  "created_at": "2026-05-06T14:08:12Z",
  "updated_at": "2026-05-06T14:08:12Z"
}
```

### GET `/quizzes/{quizID}`

Public. Токен не нужен: frontend может показать тест до авторизации. Авторизация нужна только при отправке попытки.

Возвращает quiz + `questions`.

В `questions[].config` правильные ответы скрыты. Frontend не должен ожидать `is_correct`, `correct`, `accepted_answers`.

### POST `/quizzes`

Admin. Использовать snake_case поля текущего backend.

Body:

```json
{
  "title": { "ru": "Тест Go", "tj": "Тести Go" },
  "description": { "ru": "Описание", "tj": "Тавсиф" },
  "passing_score": 80,
  "max_attempts": 3,
  "retake_cooldown_days": 0,
  "questions": [
    {
      "position": 1,
      "type": "single_choice",
      "prompt": { "ru": "Go это язык?", "tj": "Go забон аст?" },
      "points": 1,
      "required": true,
      "config": {
        "options": [
          { "id": "yes", "text": { "ru": "Да", "tj": "Ҳа" }, "isCorrect": true },
          { "id": "no", "text": { "ru": "Нет", "tj": "Не" }, "isCorrect": false }
        ]
      }
    }
  ]
}
```

Правило прохождения:

- backend считает максимум как сумму `questions[].points`;
- backend считает `score_percent = total_earned / total_max * 100`;
- попытка успешна, если `score_percent >= passing_score`;
- `passing_score` должен быть от 1 до 100.

### PUT `/quizzes/{quizID}`

Admin. Body как create.

### DELETE `/quizzes/{quizID}`

Admin. Archive.

## Question config

`single_choice` / `image_choice`:

```json
{
  "options": [
    { "id": "a", "text": { "ru": "A", "tj": "A" }, "isCorrect": true },
    { "id": "b", "text": { "ru": "B", "tj": "B" }, "isCorrect": false }
  ]
}
```

`multiple_choice`:

```json
{
  "options": [
    { "id": "a", "text": { "ru": "A", "tj": "A" }, "isCorrect": true },
    { "id": "b", "text": { "ru": "B", "tj": "B" }, "isCorrect": true }
  ]
}
```

`true_false`:

```json
{
  "correct": true
}
```

`short_answer` / `fill_blank`:

```json
{
  "acceptedAnswers": ["go", "golang"]
}
```

Manual review types:

- `long_text`
- `matching`
- `ordering`
- `audio`
- `video`
- `code`

## Attempts

### POST `/courses/{courseID}/attempts`

Protected.

Body:

```json
{
  "started_at": "2026-05-06T14:00:00Z",
  "answers": [
    {
      "question_id": "uuid",
      "selected_option_ids": ["yes"]
    }
  ]
}
```

Admin может передать `user_id`, обычный пользователь не может сдавать за другого.

Response:

```json
{
  "id": "uuid",
  "course_id": "uuid",
  "user_id": "uuid",
  "started_at": "2026-05-06T14:00:00Z",
  "finished_at": "2026-05-06T14:10:00Z",
  "total_earned": 8,
  "total_max": 10,
  "score_percent": 80,
  "passed": true
}
```

Скрыты: `questions_snapshot`, `answers_data`.

Бизнес-правила:

- сертификат уже выдан -> тест повторно сдавать нельзя;
- `score_percent >= passing_score` -> попытка пройдена;
- `score_percent < passing_score` -> попытка не пройдена;
- после `max_attempts` неудачных/любых попыток сдача закрыта до `retake_cooldown_days`;
- после cooldown попытки снова доступны.

### GET `/attempts`

Query:

- `course_id`
- `user_id`
- `limit`
- `offset`

Обычный пользователь видит только свои attempts.

Manual review endpoint сейчас не зарегистрирован в router. Вопросы manual-типов можно хранить, но полноценный review flow требует отдельной доработки backend и UI.

## Enrollments

### POST `/enrollments`

```json
{
  "course_id": "uuid"
}
```

Admin может передать `user_id`, обычный пользователь записывает только себя.

### GET `/enrollments`

Query:

- `course_id`
- `user_id`
- `status`
- `limit`
- `offset`

### POST `/enrollments/{enrollmentID}/complete`

Admin. Завершает курс и запускает auto certificate issue, если есть passed attempt.

## Certificates

### GET `/certificates`

Обычный пользователь видит только свои сертификаты.

Query:

- `user_id`
- `course_id`
- `enrollment_id`
- `limit`
- `offset`

### GET `/certificates/{certificateID}`

Public. Токен не нужен: можно открыть сертификат по прямой ссылке.

Также поддерживается совместимый singular route:

```text
GET /certificate/{certificateID}
```

### GET `/certificates/verify/{verifyHash}`

Public.

### POST `/certificates`

Admin override.

```json
{
  "enrollment_id": "uuid",
  "attempt_id": "uuid",
  "pdf_url": "/uploads/cert.pdf"
}
```

## Course packages

### POST `/course-packages`

Admin. Создаёт курс с quiz-настройками и вопросами одним запросом.

```json
{
  "course": {
    "title": { "ru": "Go", "tj": "Go" },
    "description": { "ru": "Описание", "tj": "Тавсиф" },
    "video_url": "https://youtube.com/watch?v=w_PlmLNWuSU",
    "quiz_pass_percent": 80,
    "max_attempts": 3,
    "retake_cooldown_days": 0,
    "questions": []
  }
}
```

Старый route управления связями курс-тест удалён из router и frontend API.

## Course modules

- `GET /course-modules?course_id=uuid`
- `GET /course-modules/{moduleID}`
- `POST /course-modules` admin
- `PUT /course-modules/{moduleID}` admin
- `DELETE /course-modules/{moduleID}` admin

## Content blocks

- `GET /content-blocks?course_id=uuid`
- `GET /content-blocks?module_id=uuid`
- `GET /content-blocks/{blockID}`
- `POST /content-blocks` admin
- `PUT /content-blocks/{blockID}` admin
- `DELETE /content-blocks/{blockID}` admin

`content-blocks` требует ровно один target: `course_id` или `module_id`.

Types:

- `text`
- `url`
- `video`
- `photo`
- `file`

## Reviews

- `GET /reviews`
- `POST /reviews`
- `GET /reviews/{reviewID}`
- `POST /reviews/{reviewID}/moderate` admin

Create:

```json
{
  "course_id": "uuid",
  "rating": 5,
  "text": "Хороший курс"
}
```

Moderate:

```json
{
  "status": "approved"
}
```

## Notifications

- `GET /notifications`
- `GET /notifications/{notificationID}`
- `POST /notifications/{notificationID}/read`
- `POST /notifications` admin

Обычный пользователь видит только свои notifications.

## Uploads

### POST `/uploads`

Admin.

`multipart/form-data`:

- `file`: файл
- `type`: `image | video | file | avatar`

Response:

```json
{
  "url": "/uploads/2026/05/file.png",
  "filename": "file.png",
  "size_bytes": 12345
}
```

## Webhooks

Admin only:

- `GET /webhooks`
- `POST /webhooks`
- `GET /webhooks/{webhookID}`
- `PUT /webhooks/{webhookID}`
- `DELETE /webhooks/{webhookID}`

`secret` возвращается только в response на create. В list/get/update его нет.

## Audit logs

Admin only:

- `GET /audit-logs`
- `GET /audit-logs/{auditLogID}`

`POST /audit-logs` нет.

## Access summary

- Public: health, register, login, Google config/login, forgot/reset password, courses list/detail, quiz detail, certificate by id/verify, uploads files.
- Auth user: profile, quizzes list, attempt submit, attempts, enrollments, own certificates list, reviews, notifications.
- Admin: content management, tests, attempts review, completion, certificates override, uploads, webhooks, audit logs.
- Super admin: users management.

## Frontend checklist

- Использовать `email`, не `username`.
- Хранить и отправлять Bearer token.
- Для lists читать `response.data`.
- Показывать `message` из ошибок.
- Если есть `field`, подсветить один input.
- Если есть `fields`, подсветить все поля.
- Не ожидать правильные ответы в quiz response.
- Не показывать тест повторно после успешного сертификата.
- Для Google брать `client_id` из `/auth/google/config`.
- Для локального сброса пароля token может вернуться из `/auth/password/forgot`, только если backend настроен на тестовый режим.
