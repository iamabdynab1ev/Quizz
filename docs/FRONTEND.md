# QUIZ Frontend Guide

Этот файл нужен фронтенд-команде как единый контракт по текущему backend.

## 1. Базовая информация

- Base API: `/api/v1`
- Public health: `/health` и `/api/v1/health`
- Авторизация: `Authorization: Bearer <token>`
- ID почти везде строка UUID
- Время в ответах: RFC3339 / ISO-8601
- Билингвальные поля передаются как объект:

```json
{
  "ru": "Русский текст",
  "tj": "Тоҷикӣ"
}
```

- Backend включает `DisallowUnknownFields`, поэтому лишние поля в JSON body вызывают ошибку
- Для большинства list endpoint-ов сейчас возвращается просто массив, без envelope `data/total`
- Для большинства list endpoint-ов используются `limit` и `offset`
- Ошибки приходят в формате:

```json
{
  "error": "validation_error",
  "message": "request validation failed"
}
```

## 2. Общие правила работы фронта

- После логина все запросы идут с `Authorization: Bearer <token>`
- Для `GET /auth/me` токен обязателен
- Если endpoint помечен как admin-only, обычный пользователь получит `403`
- Для user-scoped данных non-admin может видеть только свои записи
- Не отправляй в backend лишние поля в body
- Не рассчитывай на `total` в list responses, его сейчас нет

## 3. Матрица доступа

| Зона | Методы | Кто может |
| --- | --- | --- |
| Health | `GET /health`, `GET /api/v1/health` | Public |
| Auth | `POST /api/v1/auth/login`, `POST /api/v1/auth/google` | Public |
| Verify certificate | `GET /api/v1/certificates/verify/{verifyHash}` | Public |
| Auth session | `GET /api/v1/auth/me`, `POST /api/v1/auth/logout` | Любой авторизованный |
| Learning content | `courses`, `quizzes`, `course-modules`, `content-blocks`, `reviews`, `attempts`, `enrollments`, `certificates`, `notifications` | Авторизованный, часть действий только admin |
| Admin tools | `users`, `webhooks`, `audit-logs`, `course-tests`, moderation/review actions | Admin |

## 4. Все доступные API

### 4.1 Public

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/health` | Health check |
| GET | `/api/v1/health` | Health check через API |
| POST | `/api/v1/auth/login` | Логин по identifier/password |
| POST | `/api/v1/auth/google` | Логин через Google `id_token` |
| GET | `/api/v1/certificates/verify/{verifyHash}` | Публичная проверка сертификата |

### 4.2 Auth

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/api/v1/auth/me` | Возвращает текущего пользователя и сессию |
| POST | `/api/v1/auth/logout` | Завершает текущую сессию |

### 4.3 Users

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/users` | admin | Список пользователей |
| POST | `/api/v1/users` | admin | Создание пользователя |
| GET | `/api/v1/users/{userID}` | admin | Карточка пользователя |
| PUT | `/api/v1/users/{userID}` | admin | Обновление пользователя |
| DELETE | `/api/v1/users/{userID}` | admin | Деактивация пользователя |

### 4.4 Courses

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/courses` | auth | Список курсов |
| GET | `/api/v1/courses/{courseID}` | auth | Карточка курса |
| POST | `/api/v1/courses` | admin | Создание |
| PUT | `/api/v1/courses/{courseID}` | admin | Обновление |
| DELETE | `/api/v1/courses/{courseID}` | admin | Архивирование |

### 4.5 Quizzes

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/quizzes` | auth | Список тестов |
| GET | `/api/v1/quizzes/{quizID}` | auth | Карточка теста |
| POST | `/api/v1/quizzes` | admin | Создание |
| PUT | `/api/v1/quizzes/{quizID}` | admin | Обновление |
| DELETE | `/api/v1/quizzes/{quizID}` | admin | Архивирование |

### 4.6 Attempts

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/quizzes/{quizID}/attempts` | auth | Отправка попытки |
| GET | `/api/v1/attempts` | auth | Список попыток |
| GET | `/api/v1/attempts/{attemptID}` | auth | Карточка попытки |
| POST | `/api/v1/attempts/{attemptID}/review` | admin | Ручная проверка `needs_review`, поддерживает `passed/comment` и `scores[]` |

### 4.7 Enrollments

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/enrollments` | auth | Записать пользователя на курс |
| GET | `/api/v1/enrollments` | auth | Список записей |
| GET | `/api/v1/enrollments/{enrollmentID}` | auth | Карточка записи |
| POST | `/api/v1/enrollments/{enrollmentID}/complete` | admin | Завершить обучение, сертификат создаётся автоматически если условия выполнены |

### 4.8 Certificates

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/certificates` | auth | Список сертификатов |
| GET | `/api/v1/certificates/{certificateID}` | auth | Карточка сертификата |
| POST | `/api/v1/certificates` | admin | Ручная выдача сертификата |
| GET | `/api/v1/certificates/verify/{verifyHash}` | public | Проверка подлинности |

### 4.9 Course modules

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/course-modules` | auth | Список модулей |
| GET | `/api/v1/course-modules/{moduleID}` | auth | Карточка модуля |
| POST | `/api/v1/course-modules` | admin | Создание |
| PUT | `/api/v1/course-modules/{moduleID}` | admin | Обновление |
| DELETE | `/api/v1/course-modules/{moduleID}` | admin | Удаление |

### 4.10 Content blocks

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/content-blocks` | auth | Список блоков |
| GET | `/api/v1/content-blocks/{blockID}` | auth | Карточка блока |
| POST | `/api/v1/content-blocks` | admin | Создание |
| PUT | `/api/v1/content-blocks/{blockID}` | admin | Обновление |
| DELETE | `/api/v1/content-blocks/{blockID}` | admin | Удаление |

### 4.11 Course tests

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/course-tests` | admin | Список связей курс/тест |
| POST | `/api/v1/course-tests` | admin | Создание связи |
| DELETE | `/api/v1/course-tests/{courseTestID}` | admin | Основной путь удаления; старый query-param вариант оставлен как compatibility |

`DELETE /course-tests` работает так:

```text
DELETE /api/v1/course-tests/{courseTestID}
DELETE /api/v1/course-tests?course_id=...&quiz_id=...
DELETE /api/v1/course-tests?module_id=...&quiz_id=...
```

### 4.12 Reviews

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/reviews` | auth | Список отзывов |
| POST | `/api/v1/reviews` | auth | Создать отзыв |
| GET | `/api/v1/reviews/{reviewID}` | auth | Карточка отзыва |
| POST | `/api/v1/reviews/{reviewID}/moderate` | admin | Модерация |

### 4.13 Notifications

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/notifications` | auth | Список уведомлений |
| GET | `/api/v1/notifications/{notificationID}` | auth | Карточка уведомления |
| POST | `/api/v1/notifications/{notificationID}/read` | auth | Отметить прочитанным |
| POST | `/api/v1/notifications` | admin | Создание уведомления |

### 4.14 Webhooks

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/webhooks` | admin | Список webhooks |
| POST | `/api/v1/webhooks` | admin | Создание webhook |
| GET | `/api/v1/webhooks/{webhookID}` | admin | Карточка webhook |
| PUT | `/api/v1/webhooks/{webhookID}` | admin | Обновление webhook |
| DELETE | `/api/v1/webhooks/{webhookID}` | admin | Удаление webhook |

### 4.15 Audit logs

| Method | Path | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/audit-logs` | admin | Список audit log |
| GET | `/api/v1/audit-logs/{auditLogID}` | admin | Карточка audit log |

## 5. Схемы запросов, которые фронт должен знать

### 5.1 Login

```json
{
  "identifier": "admin",
  "password": "secret"
}
```

Ответ:

```json
{
  "token": "session-token",
  "expires_at": "2026-05-01T12:00:00Z",
  "user": { }
}
```

### 5.2 Google login

```json
{
  "id_token": "google-id-token"
}
```

Ответ тот же, что у `POST /auth/login`.

### 5.3 Submit attempt

```json
{
  "user_id": "optional-for-admin",
  "started_at": "2026-05-01T10:00:00Z",
  "answers": [
    {
      "question_id": "uuid",
      "selected_option_ids": ["uuid"]
    }
  ]
}
```

Правило:
- обычный пользователь не может подставить чужой `user_id`
- admin может передать `user_id`
- если `user_id` не передан, backend использует текущего пользователя

### 5.4 Review attempt

```json
{
  "passed": true,
  "comment": "Хороший ответ, но не полный"
}
```

### 5.5 Create enrollment

```json
{
  "course_id": "uuid",
  "user_id": "uuid"
}
```

### 5.6 Complete enrollment

```json
{}
```

Сертификат создаётся автоматически, если условия выполнены.

### 5.7 Create review

```json
{
  "course_id": "uuid",
  "user_id": "optional",
  "rating": 5,
  "text": "Очень полезный курс"
}
```

### 5.8 Moderate review

```json
{
  "id": "uuid",
  "status": "approved"
}
```

### 5.9 Create webhook

```json
{
  "name": "CRM",
  "url": "https://example.com/webhook",
  "events": ["course.created", "attempt.finished"],
  "secret": "shared-secret",
  "status": "active"
}
```

Важно:
- `secret` возвращается только в ответе на создание
- в `GET`/`PUT` секрет больше не должен показываться полностью

### 5.10 Create notification

```json
{
  "user_id": "uuid",
  "type": "system",
  "title": { "ru": "Заголовок", "tj": "Сарлавҳа" },
  "body": { "ru": "Текст", "tj": "Матн" },
  "link": "https://..."
}
```

## 6. Основные фильтры list endpoint-ов

### Users

- `search`
- `role`
- `is_active`
- `limit`
- `offset`

### Courses

- `search`
- `status`
- `category`
- `platform`
- `limit`
- `offset`

### Quizzes

- `search`
- `status`
- `category`
- `platform`
- `limit`
- `offset`

### Attempts

- `quiz_id`
- `user_id`
- `limit`
- `offset`

### Enrollments

- `course_id`
- `user_id`
- `status`
- `limit`
- `offset`

### Certificates

- `user_id`
- `course_id`
- `enrollment_id`
- `limit`
- `offset`

### Course modules

- `course_id`

### Content blocks

- `course_id`
- `module_id`

### Reviews

- `course_id`
- `user_id`
- `status`
- `limit`
- `offset`

### Notifications

- `user_id`
- `type`
- `read`
- `limit`
- `offset`

### Webhooks

- `status`
- `limit`
- `offset`

## 7. Что фронту важно помнить

- `POST /api/v1/uploads` реализован через admin API
- `POST /audit-logs` больше нет
- `GET /webhooks` и `GET /webhooks/{id}` не должны светить полный secret
- `DELETE /course-tests` использует query params
- `GET /content-blocks` обычно вызывается с `course_id` или `module_id`
- `GET /attempts`, `GET /enrollments`, `GET /certificates`, `GET /notifications` для non-admin всегда ограничиваются текущим пользователем
- `POST /attempts/{attemptID}/review` только для admin; при `scores[]` backend пересчитывает итоговые баллы
- `POST /enrollments/{enrollmentID}/complete` только для admin
- `POST /certificates` можно использовать как ручной override, но основной flow должен идти через complete

## 8. Текущий контракт списков

Сейчас list endpoint-ы возвращают массив напрямую:

```json
[
  {
    "id": "uuid"
  }
]
```

То есть:
- `data` envelope нет
- `total` envelope нет
- `has_more` envelope нет
- фронт должен сам управлять пагинацией через `limit` и `offset`

## 9. Что считать стабильным поведением backend

- login работает через session token
- Google login возвращает тот же `LoginResult`
- `auth/me` возвращает текущего пользователя и session
- сертификат можно проверить по `verifyHash`
- попытки с `needs_review=true` можно добрать через admin review endpoint, включая manual scoring для open-ended вопросов
- сертификат выдаётся автоматически при `enrollment complete`, если условия выполнены
