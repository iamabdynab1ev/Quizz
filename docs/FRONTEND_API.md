# LMS Arvand Frontend API

Документ описывает текущий HTTP API проекта `lms-arvand-backend` по состоянию исходного кода. Это не желаемая архитектура, а фактическое поведение backend.

Для тестового запуска фронта и backend см. [DEPLOY_WINDOWS_TEST.md](DEPLOY_WINDOWS_TEST.md).

## 1. Базовые данные

- Base URL: `http://localhost:8080/api/v1`
- Health URL: `http://localhost:8080/health`
- Public verify URL: `http://localhost:8080/api/v1/certificates/verify/{verifyHash}`
- Dev admin login:
  - `username`: `admin`
  - `password`: `Admin123!`

## 2. Общие правила API

### 2.1 Авторизация

Backend использует не JWT, а opaque session token.

Во все защищенные endpoint-ы frontend должен отправлять:

```http
Authorization: Bearer <token>
```

Для браузерного frontend backend теперь отдает CORS headers. Разрешенные origins управляются через `HTTP_CORS_ALLOWED_ORIGINS`.

### 2.2 Формат времени

Все даты и время возвращаются в ISO-8601 / RFC3339 строках.

### 2.3 ID

Все идентификаторы сущностей это UUID-строки.

Пример:

```text
b045dd03-214c-446e-964e-2e9aed06b1b4
```

### 2.4 MultiLangText

Bilingual-поля передаются как JSON-объект:

```json
{
  "ru": "Русский текст",
  "tj": "Матни тоҷикӣ"
}
```

Для большинства create/update сценариев backend ожидает обе локали: `ru` и `tj`.

### 2.5 Текущий формат list-ответов

Сейчас list endpoint-ы возвращают envelope вида `data/total/limit/offset`.

Пример:

```json
{
  "data": [
    {
      "id": "..."
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

Для `course-modules`, `content-blocks` и `course-tests` пагинация остаётся упрощенной по контракту самого endpoint-а, но ответ всё равно завернут в envelope.

### 2.6 Пагинация

Для большинства list endpoint-ов поддерживаются query-параметры:

- `limit`
- `offset`

Текущее поведение в usecase-ах:

- если `limit <= 0`, backend подставляет `20`
- максимум `limit` обычно `100`
- `offset < 0` считается ошибкой валидации

Есть исключения:

- `course-modules`
- `content-blocks`
- `course-tests`

Эти endpoint-ы не используют стандартный `limit/offset` list-flow.

### 2.7 Ошибки

Единый формат ошибки:

```json
{
  "error": "validation_error",
  "message": "request validation failed"
}
```

Основные error codes:

- `validation_error`
- `unauthorized`
- `forbidden`
- `not_found`
- `conflict`
- `internal_error`
- `service_unavailable`

Важно:

- многие бизнес-ошибки мапятся в общие сообщения вроде `resource conflict` или `request validation failed`
- часть endpoint-ов возвращает более точный `message`, если ошибка оформлена как `AppError`

### 2.8 JSON-правила

Backend включает `DisallowUnknownFields()`. Это значит:

- лишние неизвестные поля в JSON body считаются ошибкой
- body должен быть одним JSON-объектом, без нескольких JSON подряд

## 3. Матрица доступа

### Public

- `GET /health`
- `GET /api/v1/health`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/google`
- `GET /api/v1/certificates/verify/{verifyHash}`

### Любой авторизованный пользователь

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

### Только admin

- все `/users`
- `POST /courses`
- `PUT /courses/{courseID}`
- `DELETE /courses/{courseID}`
- `POST /quizzes`
- `PUT /quizzes/{quizID}`
- `DELETE /quizzes/{quizID}`
- `POST /enrollments/{enrollmentID}/complete`
- `POST /certificates`
- `POST /course-modules`
- `PUT /course-modules/{moduleID}`
- `DELETE /course-modules/{moduleID}`
- `POST /content-blocks`
- `PUT /content-blocks/{blockID}`
- `DELETE /content-blocks/{blockID}`
- `GET /course-tests`
- `POST /course-tests`
- `DELETE /course-tests`
- `POST /reviews/{reviewID}/moderate`
- `POST /notifications`
- все `/webhooks`
- все `/audit-logs`

### User-scoped endpoint-ы

Для non-admin backend сам ограничивает доступ к своим данным. Это касается:

- `attempts`
- `enrollments`
- `certificates`
- `notifications`

Следствия для frontend:

- обычный пользователь не может читать чужие записи даже если подставит чужой `user_id`
- admin может читать и фильтровать по любому `user_id`

## 4. Основные схемы объектов

### 4.1 User

```json
{
  "id": "uuid",
  "username": "admin",
  "email": "admin@local.test",
  "google_id": "optional-google-subject",
  "role": "admin",
  "first_name": "System",
  "last_name": "Admin",
  "patronymic": "",
  "phone": "+992...",
  "gender": "unspecified",
  "address": "optional",
  "city": "optional",
  "avatar_url": "optional",
  "is_active": true,
  "created_at": "2026-04-24T09:18:48Z",
  "updated_at": "2026-04-24T09:34:14Z",
  "employee_info": {},
  "admin_info": {},
  "student_info": {},
  "guest_info": {}
}
```

Возможные `role`:

- `admin`
- `employee`
- `student`
- `guest`

Возможные `gender`:

- `male`
- `female`
- `other`
- `unspecified`

### 4.2 Course

```json
{
  "id": "uuid",
  "title": { "ru": "Курс", "tj": "Курс" },
  "description": { "ru": "Описание", "tj": "Тавсиф" },
  "cover_image_url": "https://...",
  "category": "security",
  "status": "draft",
  "platforms": ["web", "mobile"],
  "estimated_minutes": 90,
  "certificate_enabled": true,
  "certificate_passing_score": 80,
  "reviews_enabled": true,
  "created_at": "2026-04-24T09:18:48Z",
  "updated_at": "2026-04-24T09:34:14Z"
}
```

Возможные `status`:

- `draft`
- `published`
- `archived`

Возможные `platforms`:

- `web`
- `mobile`
- `telegram`

### 4.3 Quiz

```json
{
  "id": "uuid",
  "title": { "ru": "Тест", "tj": "Тест" },
  "description": { "ru": "Описание", "tj": "Тавсиф" },
  "category": "security",
  "status": "draft",
  "platforms": ["web"],
  "time_limit_minutes": 30,
  "passing_score": 70,
  "max_attempts": 3,
  "shuffle_questions": false,
  "show_results": true,
  "allow_retry": true,
  "questions": [],
  "created_at": "2026-04-24T09:18:48Z",
  "updated_at": "2026-04-24T09:34:14Z"
}
```

Возможные `status`:

- `draft`
- `published`
- `archived`

### 4.4 Question

```json
{
  "id": "uuid",
  "position": 1,
  "type": "single_choice",
  "prompt": { "ru": "Вопрос", "tj": "Савол" },
  "explanation": { "ru": "Пояснение", "tj": "Шарҳ" },
  "points": 1,
  "required": true,
  "config": {},
  "created_at": "2026-04-24T09:18:48Z"
}
```

Возможные `type`:

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

### 4.5 Attempt

```json
{
  "id": "uuid",
  "quiz_id": "uuid",
  "user_id": "uuid",
  "started_at": "2026-04-24T09:18:48Z",
  "finished_at": "2026-04-24T09:21:10Z",
  "questions_snapshot": [],
  "answers_data": [],
  "total_earned": 8,
  "total_max": 10,
  "score_percent": 80,
  "passed": true,
  "needs_review": false
}
```

### 4.6 AttemptAnswer

```json
{
  "question_id": "uuid",
  "selected_option_ids": ["uuid"],
  "text_answer": "answer",
  "boolean_answer": true,
  "ordered_option_ids": ["uuid1", "uuid2"],
  "matched_pairs": {
    "left1": "right2"
  }
}
```

### 4.7 Enrollment

```json
{
  "id": "uuid",
  "course_id": "uuid",
  "user_id": "uuid",
  "status": "active",
  "enrolled_at": "2026-04-24T09:18:48Z",
  "completed_at": "2026-04-24T10:00:00Z"
}
```

Возможные `status`:

- `active`
- `completed`
- `dropped`

### 4.8 Certificate

```json
{
  "id": "uuid",
  "enrollment_id": "uuid",
  "user_id": "uuid",
  "course_id": "uuid",
  "attempt_id": "uuid",
  "serial_number": "123-456-789",
  "verify_hash": "long-hex-string",
  "issued_at": "2026-04-24T10:00:00Z",
  "pdf_url": "https://...",
  "user_first_name": "John",
  "user_last_name": "Doe",
  "patronymic": "",
  "course_title": { "ru": "Курс", "tj": "Курс" }
}
```

### 4.9 CourseModule

```json
{
  "id": "uuid",
  "course_id": "uuid",
  "position": 1,
  "title": { "ru": "Модуль", "tj": "Модул" },
  "description": { "ru": "Описание", "tj": "Тавсиф" }
}
```

### 4.10 ContentBlock

```json
{
  "id": "uuid",
  "course_id": "uuid",
  "module_id": null,
  "position": 1,
  "type": "text",
  "title": { "ru": "Блок", "tj": "Блок" },
  "payload": {}
}
```

Возможные `type`:

- `text`
- `url`
- `video`
- `photo`
- `file`

### 4.11 ContentBlock payload shapes

`text`:

```json
{
  "content": {
    "ru": "Текст",
    "tj": "Матн"
  }
}
```

`url`:

```json
{
  "url": "https://example.com",
  "label": {
    "ru": "Ссылка",
    "tj": "Пайванд"
  }
}
```

`video`:

```json
{
  "url": "https://example.com/video.mp4",
  "provider": "direct",
  "duration_seconds": 120
}
```

Возможные `provider`:

- `direct`
- `youtube`

`photo`:

```json
{
  "url": "https://example.com/image.jpg",
  "caption": {
    "ru": "Подпись",
    "tj": "Тавзеҳ"
  }
}
```

`file`:

```json
{
  "url": "https://example.com/file.pdf",
  "filename": "manual.pdf",
  "size_bytes": 123456
}
```

### 4.12 CourseTest

```json
{
  "course_id": "uuid",
  "module_id": null,
  "quiz_id": "uuid",
  "position": 1
}
```

### 4.13 Review

```json
{
  "id": "uuid",
  "course_id": "uuid",
  "user_id": "uuid",
  "rating": 5,
  "text": "Отличный курс",
  "status": "pending",
  "created_at": "2026-04-24T09:18:48Z",
  "moderated_at": "2026-04-24T10:00:00Z"
}
```

Возможные `status`:

- `pending`
- `approved`
- `rejected`

### 4.14 Notification

```json
{
  "id": "uuid",
  "user_id": "uuid",
  "type": "system",
  "title": { "ru": "Заголовок", "tj": "Сарлавҳа" },
  "body": { "ru": "Текст", "tj": "Матн" },
  "link": "https://...",
  "read": false,
  "created_at": "2026-04-24T09:18:48Z"
}
```

Возможные `type`:

- `course.published`
- `certificate.issued`
- `review.approved`
- `enrollment.created`
- `system`

### 4.15 Webhook

```json
{
  "id": "uuid",
  "name": "CRM webhook",
  "url": "https://example.com/hook",
  "events": ["course.created", "attempt.passed"],
  "secret": "plain-secret",
  "status": "active",
  "last_triggered_at": "2026-04-24T09:18:48Z",
  "last_status_code": 200,
  "last_error": "optional",
  "deliveries": 10,
  "failures": 2,
  "created_at": "2026-04-24T09:18:48Z",
  "updated_at": "2026-04-24T09:34:14Z"
}
```

Важно:

- сейчас backend действительно возвращает `secret` в API-ответах
- frontend не должен логировать или показывать это поле без необходимости

### 4.16 AuditLog

```json
{
  "id": "uuid",
  "type": "course.created",
  "at": "2026-04-24T09:18:48Z",
  "actor_id": "uuid",
  "payload": {}
}
```

## 5. Auth API

### 5.1 `POST /auth/login`

Request:

```json
{
  "identifier": "admin",
  "password": "Admin123!"
}
```

Response `200 OK`:

```json
{
  "token": "opaque-session-token",
  "expires_at": "2026-04-25T09:41:46Z",
  "user": {
    "id": "uuid",
    "username": "admin",
    "role": "admin"
  }
}
```

Логика:

- `identifier` это username или email
- token не JWT
- session хранится на backend в таблице `sessions`
- при слишком большом числе неудачных попыток backend может вернуть `429 Too Many Requests`

Пример ошибки throttling:

```json
{
  "error": {
    "code": "too_many_attempts",
    "message": "too many login attempts, try again later"
  }
}
```

### 5.2 `POST /auth/google`

Request:

```json
{
  "id_token": "google-id-token-from-frontend"
}
```

Response `200 OK`:

- тот же `LoginResult`, что и у `POST /auth/login`

Текущее поведение backend:

- frontend сам получает `id_token` через Google SDK
- backend валидирует `id_token` через Google tokeninfo
- если пользователь найден по `google_id`, он логинится
- если пользователь найден по `email`, backend привязывает `google_id`
- если пользователя нет, backend создает нового пользователя с ролью `guest`

### 5.3 `GET /auth/me`

Response:

```json
{
  "user": {},
  "session": {}
}
```

### 5.4 `POST /auth/logout`

Response:

- `204 No Content`

## 6. Users API

### 6.1 `GET /users`

Admin only.

Query:

- `search`
- `role`
- `is_active`
- `limit`
- `offset`

### 6.2 `POST /users`

Admin only.

Body: `CreateUserParams`

Основные поля:

- `username`
- `email`
- `google_id`
- `password`
- `role`
- `first_name`
- `last_name`
- `patronymic`
- `phone`
- `gender`
- `address`
- `city`
- `avatar_url`
- `employee_info`
- `admin_info`
- `student_info`
- `guest_info`

### 6.3 `GET /users/{userID}`

Admin only.

### 6.4 `PUT /users/{userID}`

Admin only.

Body: `UpdateUserParams`

### 6.5 `DELETE /users/{userID}`

Admin only.

Важно:

- это soft delete через деактивацию пользователя
- endpoint возвращает `204 No Content`

## 7. Courses API

### 7.1 `GET /courses`

Query:

- `search`
- `status`
- `category`
- `platform`
- `limit`
- `offset`

### 7.2 `GET /courses/{courseID}`

### 7.3 `POST /courses`

Admin only.

Body: `CreateCourseParams`

### 7.4 `PUT /courses/{courseID}`

Admin only.

Body: `UpdateCourseParams`

### 7.5 `DELETE /courses/{courseID}`

Admin only.

Важно:

- курс не удаляется физически
- backend архивирует его через `status = archived`

## 8. Quizzes API

### 8.1 `GET /quizzes`

Query:

- `search`
- `status`
- `category`
- `platform`
- `limit`
- `offset`

### 8.2 `GET /quizzes/{quizID}`

Возвращает quiz вместе с `questions`.

### 8.3 `POST /quizzes`

Admin only.

Body: `CreateQuizParams`

Важно:

- нужен хотя бы один вопрос
- если `max_attempts <= 0`, backend ставит `3`
- если `points <= 0`, для вопроса backend ставит `1`

### 8.4 `PUT /quizzes/{quizID}`

Admin only.

Body: `UpdateQuizParams`

### 8.5 `DELETE /quizzes/{quizID}`

Admin only.

Важно:

- quiz не удаляется физически
- backend архивирует его через `status = archived`

## 9. Attempts API

### 9.1 `POST /quizzes/{quizID}/attempts`

Body:

```json
{
  "user_id": "optional-admin-only",
  "started_at": "2026-04-24T09:18:48Z",
  "answers": [
    {
      "question_id": "uuid",
      "selected_option_ids": ["uuid"],
      "text_answer": "answer",
      "boolean_answer": true,
      "ordered_option_ids": ["uuid1", "uuid2"],
      "matched_pairs": {
        "left1": "right2"
      }
    }
  ]
}
```

Логика:

- non-admin не может отправить попытку за другого пользователя
- backend считает число предыдущих попыток
- если `attemptCount >= quiz.max_attempts`, вернет conflict
- сохраняется snapshot вопросов и answers_data

Автооценка сейчас работает для:

- `single_choice`
- `image_choice`
- `multiple_choice`
- `true_false`
- `short_answer`
- `fill_blank`

Для остальных типов backend ставит `needs_review = true`:

- `long_text`
- `matching`
- `ordering`
- `audio`
- `video`
- `code`

Текущий review flow:

- для попыток с `needs_review = true` теперь есть admin endpoint `POST /api/v1/attempts/{attemptID}/review`
- review поддерживает legacy `passed/comment` и manual scoring через `scores[]`

### 9.2 `GET /attempts`

Query:

- `quiz_id`
- `user_id`
- `limit`
- `offset`

Non-admin видит только свои попытки.

### 9.3 `GET /attempts/{attemptID}`

Non-admin может читать только свою попытку.

### 9.4 `POST /attempts/{attemptID}/review`

Admin only.

Body:

```json
{
  "passed": true,
  "comment": "Проверено вручную",
  "scores": [
    {
      "question_id": "uuid",
      "points": 8,
      "comment": "Хороший ответ"
    }
  ]
}
```

`scores[]` нужен только для вопросов, которые backend не может оценить автоматически:

- `long_text`
- `matching`
- `ordering`
- `audio`
- `video`
- `code`

Если `scores[]` передан, backend:

- проверяет, что для всех manual-вопросов есть оценка;
- пересчитывает `total_earned`;
- пересчитывает `score_percent`;
- выставляет `passed` по `quiz.passing_score`;
- сохраняет breakdown в `review_scores`.

Текущее поведение:

- работает только для попыток с `needs_review = true`
- сохраняет `reviewed_at`, `reviewer_id`, `review_comment`, `manual_passed`, `review_scores`
- снимает `needs_review`
- если передан `scores[]`, backend пересчитывает итоговые баллы и pass/fail

## 10. Enrollments API

### 10.1 `POST /enrollments`

Body:

```json
{
  "course_id": "uuid",
  "user_id": "optional-admin-only"
}
```

Non-admin создает enrollment только для себя.

### 10.2 `GET /enrollments`

Query:

- `course_id`
- `user_id`
- `status`
- `limit`
- `offset`

Non-admin видит только свои enrollment-ы.

### 10.3 `GET /enrollments/{enrollmentID}`

### 10.4 `POST /enrollments/{enrollmentID}/complete`

Admin only.

Важно:

- endpoint завершает enrollment
- после completion backend пытается автоматически выпустить сертификат, если:
  - у курса `certificate_enabled = true`
  - у пользователя есть подходящая passed attempt по этому курсу
  - `course.certificate_passing_score` либо `0`, либо candidate attempt набрала не меньше этого порога; если порог задан, именно он является source of truth
- если подходящего attempt нет, enrollment все равно успешно завершается

## 11. Certificates API

### 11.1 `GET /certificates/verify/{verifyHash}`

Public endpoint.

Используется для внешней проверки сертификата.

### 11.2 `GET /certificates`

Query:

- `user_id`
- `course_id`
- `enrollment_id`
- `limit`
- `offset`

Non-admin видит только свои сертификаты.

### 11.3 `GET /certificates/{certificateID}`

### 11.4 `POST /certificates`

Admin only.

Body:

```json
{
  "enrollment_id": "uuid",
  "attempt_id": "uuid",
  "pdf_url": "https://..."
}
```

Сертификат создается только если:

- enrollment существует
- enrollment имеет статус `completed`
- у курса `certificate_enabled = true`
- attempt имеет `passed = true`
- attempt принадлежит тому же пользователю
- quiz attempt-а действительно привязан к этому курсу через `course_tests`

Важно:

- выдача сертификата сейчас ручная
- `course.certificate_passing_score` используется как source of truth для порога сертификата, а если он равен `0`, backend fallback-ится на `attempt.passed`

## 12. Course Modules API

### 12.1 `GET /course-modules`

Query:

- `course_id`

### 12.2 `GET /course-modules/{moduleID}`

### 12.3 `POST /course-modules`

Admin only.

Body: `CreateCourseModuleParams`

### 12.4 `PUT /course-modules/{moduleID}`

Admin only.

Body: `UpdateCourseModuleParams`

### 12.5 `DELETE /course-modules/{moduleID}`

Admin only.

## 13. Content Blocks API

### 13.1 `GET /content-blocks`

Query:

- `course_id`
- `module_id`

Важно:

- можно передать только один из них
- если не передать ни одного, backend вернет validation error

### 13.2 `GET /content-blocks/{blockID}`

### 13.3 `POST /content-blocks`

Admin only.

Body: `CreateContentBlockParams`

Важно:

- нужно передать ровно один из `course_id` или `module_id`
- `position` должен быть больше `0`
- `title` требует обе локали
- `payload` валидируется в зависимости от `type`

### 13.4 `PUT /content-blocks/{blockID}`

Admin only.

Body: `UpdateContentBlockParams`

### 13.5 `DELETE /content-blocks/{blockID}`

Admin only.

## 14. Course Tests API

### 14.1 `GET /course-tests`

Admin only.

Query:

- `course_id`
- `module_id`

Важно:

- можно передать только один из них

### 14.2 `POST /course-tests`

Admin only.

Body:

```json
{
  "course_id": "uuid",
  "module_id": null,
  "quiz_id": "uuid",
  "position": 1
}
```

Важно:

- нужно передать ровно один из `course_id` или `module_id`

### 14.3 `DELETE /course-tests/{courseTestID}`

Admin only.

Новый основной контракт - удаление по `id` через path:

```text
DELETE /api/v1/course-tests/{courseTestID}
```

Для compatibility старый query-param вариант пока тоже оставлен:

```text
DELETE /api/v1/course-tests?course_id=...&quiz_id=...
DELETE /api/v1/course-tests?module_id=...&quiz_id=...
```

Это текущий контракт backend.

## 15. Reviews API

### 15.1 `GET /reviews`

Query:

- `course_id`
- `user_id`
- `status`
- `limit`
- `offset`

### 15.2 `POST /reviews`

Body:

```json
{
  "course_id": "uuid",
  "user_id": "optional",
  "rating": 5,
  "text": "Отличный курс"
}
```

Важно:

- это отзывы на курсы
- это не review endpoint для попыток

### 15.3 `GET /reviews/{reviewID}`

### 15.4 `POST /reviews/{reviewID}/moderate`

Admin only.

Body:

```json
{
  "status": "approved"
}
```

Допустимые значения:

- `approved`
- `rejected`

## 16. Notifications API

### 16.1 `GET /notifications`

Query:

- `user_id`
- `type`
- `read`
- `limit`
- `offset`

Non-admin видит только свои уведомления.

### 16.2 `GET /notifications/{notificationID}`

### 16.3 `POST /notifications`

Admin only.

Body: `CreateNotificationParams`

### 16.4 `POST /notifications/{notificationID}/read`

Помечает уведомление прочитанным и возвращает обновленный объект.

## 17. Webhooks API

### 17.1 `GET /webhooks`

Admin only.

Query:

- `status`
- `limit`
- `offset`

Response:

- возвращает webhook-объекты без поля `secret`

### 17.2 `POST /webhooks`

Admin only.

Body:

```json
{
  "name": "CRM",
  "url": "https://example.com/hook",
  "events": ["course.created", "attempt.passed"],
  "secret": "plain-secret",
  "status": "active"
}
```

Response:

- backend возвращает `secret` только в ответе на create

### 17.3 `GET /webhooks/{webhookID}`

Admin only.

Response:

- возвращает webhook без `secret`

### 17.4 `PUT /webhooks/{webhookID}`

Admin only.

Body: `UpdateWebhookParams`

Response:

- возвращает webhook без `secret`

### 17.5 `DELETE /webhooks/{webhookID}`

Admin only.

Важно:

- webhook CRUD реализован
- `secret` больше не отдается в `GET /webhooks`, `GET /webhooks/{id}` и `PUT /webhooks/{id}`
- `secret` приходит только один раз в ответе на `POST /webhooks`
- backend теперь пытается доставлять webhook-события по audit/app events
- используется HMAC подпись в заголовке `X-LMS-Signature`
- заголовки доставки:
  - `X-LMS-Event`
  - `X-LMS-Delivery-ID`
  - `X-LMS-Timestamp`
  - `X-LMS-Signature`
- поля `deliveries`, `failures`, `last_triggered_at`, `last_status_code`, `last_error` теперь обновляются при доставке
- delivery для audit-webhook теперь идёт через durable outbox worker, а не только best-effort

## 18. Audit Logs API

### 18.1 `GET /audit-logs`

Admin only.

Query:

- `type`
- `actor_id`
- `limit`
- `offset`

### 18.2 `GET /audit-logs/{auditLogID}`

Admin only.

Важно:

- публичного `POST /audit-logs` больше нет
- audit logs теперь читаются через API, а новые записи создаются backend-ом как внутренний side effect части мутаций

## 19. Что важно знать frontend-команде

### 19.1 Сейчас уже реализовано

- логин по username/email
- Google Sign-In
- opaque session auth
- курсы, тесты, попытки, enrollment-ы, сертификаты
- модули и контентные блоки
- отзывы и уведомления
- CRUD для webhooks
- read API для audit logs + внутреннее audit logging части мутаций

### 19.2 Сейчас не реализовано

- websocket layer
- Redis-backed distributed session cache

### 19.3 Upload API

- `POST /api/v1/uploads`
- доступ: `admin`
- content-type: `multipart/form-data`
- form fields:
  - `type`: `image | video | file | avatar`
  - `file`: сам файл
- response:
  - `url`: публичный путь файла вида `/uploads/...`
  - `filename`: исходное имя файла
  - `size_bytes`: размер файла
- backend сохраняет файлы в локальное файловое хранилище и раздает их через `/uploads/*`

### 19.4 Что считать текущими архитектурными особенностями

- list endpoint-ы возвращают envelope `data/total/limit/offset`
- `DELETE /course-tests` использует query params
- `webhook.secret` возвращается только один раз при create
- login endpoint теперь может отвечать `429 too_many_attempts`
- backend поддерживает CORS через `HTTP_CORS_ALLOWED_ORIGINS`
- `course.certificate_passing_score` используется как source of truth для порога сертификата
