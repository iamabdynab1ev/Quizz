# QUIZ Frontend Guide

Короткая инструкция для подключения frontend к backend. Полный контракт endpoint-ов см. в [FRONTEND_API.md](FRONTEND_API.md).

## Локальная схема

Backend:

```text
http://127.0.0.1:9000
```

Frontend Vite:

```text
http://localhost:4041
```

Рекомендуемый `.env` backend:

```env
HTTP_ADDRESS=127.0.0.1:9000
HTTP_CORS_ALLOWED_ORIGINS=http://localhost:4041
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5433/lms_arvand?sslmode=disable
MIGRATE_RUN_ON_START=true
SEED_RUN_ON_START=true
AUTH_LOGIN_LOCKOUT_ENABLED=false
AUTH_PASSWORD_RESET_RETURN_TOKEN=true
```

Рекомендуемый `.env` frontend:

```env
VITE_API_URL=http://localhost:9000/api/v1
```

Если frontend настроен через proxy или nginx, можно использовать:

```env
VITE_API_URL=/api/v1
```

## Вход

Первый вход:

```json
{
  "email": "admin@local.test",
  "password": "Admin123!"
}
```

Endpoint:

```text
POST /api/v1/auth/login
```

После входа сохранить `token` и отправлять:

```http
Authorization: Bearer <token>
```

`username` на frontend больше не использовать.

## Google login

Frontend сначала вызывает:

```text
GET /api/v1/auth/google/config
```

Если ответ:

```json
{
  "enabled": true,
  "client_id": "....apps.googleusercontent.com"
}
```

то показывать кнопку Google и использовать `client_id` для Google SDK.

После Google SDK отправить:

```text
POST /api/v1/auth/google
```

```json
{
  "id_token": "google-id-token"
}
```

## Профиль

Текущий пользователь:

```text
GET /api/v1/auth/me
```

Редактировать свой профиль:

```text
PUT /api/v1/auth/me
```

```json
{
  "first_name": "Ali",
  "last_name": "Karimov",
  "phone": "900000000",
  "is_male": true,
  "city": "Худжанд",
  "birth_date": "2001-07-12"
}
```

## Роли на frontend

Использовать:

- `user.is_admin === true` - админские экраны контента.
- `user.is_super_admin === true` - управление пользователями.

Не использовать:

- `role`
- `username`
- `admin_info`
- `is_active`

## Списки

Большинство списков приходит так:

```json
{
  "data": [],
  "total": 15,
  "limit": 20,
  "offset": 0
}
```

Читать элементы нужно из `data`.

## Ошибки

Если пришло:

```json
{
  "field": "passing_points",
  "code": "too_high",
  "message": "Баллы для прохождения не могут быть больше максимального балла теста (10.00)"
}
```

Нужно показать message и подсветить input `passing_points` / `passingPoints`.

Если пришло:

```json
{
  "error": "conflict",
  "message": "Лимит попыток исчерпан. Повторная сдача будет доступна после 05.06.2026."
}
```

Показать `message` как общую ошибку.

## Курсы и тесты

Курс может иметь `video_url` и `quiz_id`.

Тест создаётся с:

```json
{
  "passingPoints": 8,
  "maxAttempts": 3,
  "retakeCooldownDays": 30,
  "questions": []
}
```

Backend сам считает максимум баллов по вопросам:

- 5 вопросов по 1 баллу -> максимум 5;
- 10 вопросов по 1 баллу -> максимум 10;
- вопросы 2 + 3 + 5 -> максимум 10.

`passingPoints` нельзя ставить больше максимума.

## Правильные ответы

Backend не отдаёт правильные ответы на frontend.

В `GET /quizzes/{id}` скрыты:

- `is_correct`
- `isCorrect`
- `correct`
- `accepted_answers`
- `acceptedAnswers`
- другие answer keys.

Admin при создании теста может отправлять эти поля, но студент при чтении теста их не получает.

## Попытки и сертификаты

Submit:

```text
POST /api/v1/quizzes/{quizID}/attempts
```

Сертификат выдаётся, если:

- попытка `passed=true`;
- `total_earned >= passing_points`;
- тест связан с курсом;
- сертификат ещё не выдавался.

После получения сертификата:

- видео остаётся доступным;
- повторная сдача теста закрывается.

Если попытки закончились:

- backend возвращает `409 conflict`;
- в message будет дата, когда можно сдавать снова.

## Uploads

Загрузка только admin:

```text
POST /api/v1/uploads
```

`multipart/form-data`:

- `file`
- `type=image|video|file|avatar`

Ответ:

```json
{
  "url": "/uploads/...",
  "filename": "file.png",
  "size_bytes": 12345
}
```

Этот `url` можно сохранять в `cover_image_url`, `video_url`, `avatar_url` или content block payload.

## Что обязательно проверить с frontend

1. Login по email/password.
2. `auth/me` после refresh страницы.
3. Создание курса с video_url.
4. Список курсов после создания/удаления.
5. Создание теста с вопросами.
6. Ошибка, если `passingPoints` больше суммы баллов вопросов.
7. Сдача теста ниже проходного балла.
8. Сдача теста выше проходного балла.
9. Получение сертификата.
10. Запрет повторной сдачи после сертификата.
11. Блокировка попыток до cooldown.
12. Редактирование профиля и `birth_date`.
13. Сброс пароля.
14. Google login, если есть `GOOGLE_CLIENT_ID`.
