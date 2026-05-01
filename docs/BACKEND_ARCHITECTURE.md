# Backend Architecture

## 1. Что это за проект

`request-system` это backend системы заявок / helpdesk-платформы.

По коду видно, что проект решает сразу несколько задач:

- хранение и обработка заявок;
- маршрутизация заявок по оргструктуре;
- управление пользователями, ролями и правами;
- хранение справочников: статусы, приоритеты, должности, филиалы, отделы, офисы, оборудование;
- история изменений заявки;
- вложения и загрузка файлов;
- Telegram-бот для просмотра и изменения заявок;
- WebSocket-уведомления для web-клиента;
- dashboard и отчетность;
- синхронизация справочников и пользователей из 1С;
- поиск пользователей в Active Directory.

Проект работает как HTTPS API-сервер на Echo, использует PostgreSQL как основную БД и Redis для кеша, токенов, Telegram-state и служебных ключей.

## 2. Технологический стек

Основной стек:

- Go `1.25.5`
- Echo v4
- PostgreSQL через `pgx/v5`
- Redis через `go-redis/v8`
- JWT через `golang-jwt/jwt/v5`
- Goose для миграций
- Zap для логирования
- Validator v10 для валидации DTO
- Gorilla WebSocket
- LDAP client для Active Directory
- Excelize для импорта оборудования из Excel

Ключевые внешние библиотеки:

- `github.com/labstack/echo/v4` - HTTP framework
- `github.com/jackc/pgx/v5` - PostgreSQL pool и доступ к БД
- `github.com/go-redis/redis/v8` - Redis
- `github.com/golang-jwt/jwt/v5` - access/refresh JWT
- `github.com/go-playground/validator/v10` - валидация
- `github.com/go-ldap/ldap/v3` - LDAP / AD
- `github.com/gorilla/websocket` - realtime websocket
- `github.com/pressly/goose/v3` - миграции
- `github.com/xuri/excelize/v2` - импорт Excel
- `go.uber.org/zap` - structured logging

## 3. Как запускается приложение

Точка входа: `app/main.go`.

Что происходит на старте:

1. Загружается конфиг через `pkg/config/config.go`.
2. Устанавливается timezone приложения.
3. Опционально прокидываются `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`.
4. Обрабатываются CLI-режимы сидов и Excel-импорта.
5. Создаются логгеры.
6. Запускаются Goose-миграции из `database/migrations/`.
7. Создается Echo.
8. Включаются middleware:
   - `Recover`
   - `Gzip`
   - `CORS`
9. Подключается PostgreSQL pool.
10. Подключается Redis.
11. Создаются JWT service, permission cache service, event bus, websocket hub.
12. Создаются notification listener, AD service, Telegram service.
13. Собираются repositories, services, controllers и routes.
14. Поднимается HTTPS server через `ListenAndServeTLS`.
15. Параллельно запускается WebSocket hub.

Важные runtime-факты:

- сервер работает по HTTPS;
- статические файлы отдаются по `/uploads`;
- `GET /ping` используется как health endpoint;
- миграции выполняются автоматически при старте;
- если миграции не проходят, сервер не поднимается.

## 4. Режимы запуска

Приложение умеет работать не только как API-сервер.

Через флаги можно запускать:

- `-core` - заполнение базовых справочников;
- `-roles` - создание ролей и root/admin данных;
- `-all` - все сидеры сразу;
- `-import-atms` - импорт банкоматов из Excel;
- `-import-terms` - импорт терминалов из Excel;
- `-import-pos` - импорт POS-терминалов из Excel.

То есть `app/main.go` это одновременно:

- production entrypoint;
- seeding launcher;
- data import launcher.

## 5. Основные директории проекта

### `app/`

Entrypoint приложения.

### `config/`

Локальные runtime-конфиги, например `config/upload.go`.

### `database/migrations/`

Goose-миграции схемы БД.

Сейчас в каталоге `70` migration-файлов. Они покрывают:

- базовую схему;
- справочники;
- пользователей и роли;
- заявки и историю;
- KPI-поля;
- order routing rules;
- интеграционные поля;
- Telegram;
- dashboard performance indexes;
- recent text/file limit changes.

### `internal/controllers/`

HTTP-обработчики.

Отвечают за:

- bind/query parsing;
- вызов service-слоя;
- возврат `SuccessResponse` / `ErrorResponse`.

### `internal/controllers/telegram/`

Отдельный слой для логики Telegram-бота:

- команды;
- callback actions;
- экранное состояние;
- карточка заявки;
- дедупликация кликов;
- выбор исполнителя;
- workflow делегирования и смены статуса.

### `internal/services/`

Главная бизнес-логика проекта.

Тут находятся:

- auth;
- orders;
- order history;
- routing;
- dashboard;
- sync;
- notifications;
- AD;
- dictionary services;
- телеграм-интеграция.

### `internal/repositories/`

Доступ к PostgreSQL и Redis.

### `internal/routes/`

DI/wiring и регистрация маршрутов.

### `internal/entities/`

DB entity-слой.

### `pkg/`

Общие инфраструктурные пакеты:

- config
- errors
- middleware
- service
- eventbus
- filestorage
- websocket
- validation
- utils
- constants
- contextkeys
- database
- telegram
- types

### `seeders/`

Инициализация базовых данных.

### `tools/`

Служебные инструменты:

- backend load test
- telegram hotpaths / loadtest

### `docs/`

Операционные и API-документы.

## 6. Общая архитектура backend

Проект построен по классической многослойной схеме:

`Route -> Controller -> Service -> Repository -> PostgreSQL/Redis`

Дополнительные поперечные слои:

- `middleware` для auth и доступа;
- `eventbus` для внутренних событий;
- `listeners` для реакции на события;
- `websocket` для realtime уведомлений;
- `telegram` для chat UX;
- `filestorage` для файлов;
- `validation` и `errors` для общих правил ответа.

### Типичный request flow

1. Echo принимает HTTP request.
2. `AuthMiddleware` валидирует JWT и кладет `user_id`, `role_id`, `permissions`, `permissionsMap` в context.
3. Route может дополнительно навесить `AuthorizeAny(...)` или `AuthorizeAll(...)`.
4. Controller:
   - парсит `query/body/form-data`;
   - валидирует DTO;
   - формирует `types.Filter` через `pkg/utils/http_helpers.go`.
5. Service:
   - проверяет бизнес-права;
   - выполняет валидацию;
   - запускает транзакции;
   - вызывает repositories;
   - при необходимости публикует событие в event bus.
6. Repository делает SQL к PostgreSQL или операции к Redis.
7. Listener может отправить Telegram/WebSocket уведомления.
8. Controller возвращает унифицированный JSON через `SuccessResponse` / `ErrorResponse`.

## 7. Конфигурация

Главный конфиг: `pkg/config/config.go`.

Он агрегирует:

- `Server`
- `Postgres`
- `Redis`
- `JWT`
- `Auth`
- `Integrations`
- `Telegram`
- `Frontend`
- `LDAP`
- `Seeder`

### ServerConfig

- `SERVER_PORT`
- `SERVER_BASE_URL`
- `ALLOWED_ORIGINS`
- `SSL_CERT_PATH`
- `SSL_KEY_PATH`
- `APP_TIMEZONE`

### PostgresConfig

- `DATABASE_URL`

### RedisConfig

- `REDIS_ADDRESS`
- `REDIS_PASSWORD`

### JWTConfig

- secret key
- access TTL
- refresh TTL

### AuthConfig

Содержит:

- reset token TTL
- verification code TTL
- max reset attempts
- max login attempts
- lockout duration
- system root login

Важно: в конфиге уже есть `MaxLoginAttempts` и `LockoutDuration`, а в `pkg/constants/constants.go` уже есть Redis-ключи `CacheKeyLockout` и `CacheKeyLoginAttempts`. Реальная блокировка после нескольких неудачных логинов внедрена в `AuthService.Login`.

### IntegrationsConfig

- `INTEGRATION_ACTIVE_PROVIDER`
- `ONE_C_API_KEY`
- default roles for 1C users
- OnlineBank settings

### TelegramConfig

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_BOT_USERNAME`
- `TELEGRAM_WEBHOOK_SECRET_TOKEN`
- `TELEGRAM_ADVANCED_MODE_ENABLED`

### LDAPConfig

- `LDAP_ENABLED`
- `LDAP_SEARCH_ENABLED`
- host, port, domain
- bind DN / password
- timeout
- base DN
- search pattern
- search attributes
- username attribute
- FIO attribute

## 8. База данных и миграции

PostgreSQL подключается через `pkg/database/postgresql/connection.go`.

Что делает connection layer:

- парсит DSN;
- маскирует пароль в логах;
- настраивает pool через env:
  - `DB_POOL_MAX_CONNS`
  - `DB_POOL_MIN_CONNS`
  - `DB_POOL_MAX_CONN_LIFETIME_MINUTES`
  - `DB_POOL_MAX_CONN_IDLE_MINUTES`
  - `DB_POOL_HEALTH_CHECK_PERIOD_SECONDS`
- делает `Ping`;
- логирует параметры пула.

### Транзакции

Транзакционный helper: `internal/repositories/transaction.go`.

Он:

- открывает `pgx.Tx`;
- запускает callback;
- делает rollback на panic/error;
- делает commit на success;
- оборачивает DB ошибки через `apperrors.WrapDBError`.

### Эволюция схемы

Схема выросла от базовых справочников до полноценной service-desk модели.

Основные классы миграций:

- core dictionaries;
- users / roles / permissions;
- orders / comments / documents / attachments / history;
- user_roles / user_permissions / denials;
- org hierarchy changes;
- integration external ids;
- Telegram chat id;
- KPI and dashboard indexes;
- text/file limits.

## 9. Основные домены данных

По `internal/entities` и миграциям backend оперирует такими доменами:

- `User`
- `Role`
- `Permission`
- `RolePermission`
- `Status`
- `Priority`
- `Position`
- `Department`
- `Otdel`
- `Branch`
- `Office`
- `EquipmentType`
- `Equipment`
- `Order`
- `Attachment`
- `OrderComment`
- `OrderDelegation`
- `OrderHistory`
- `OrderType`
- `OrderRoutingRule`
- report entities

### Ключевая сущность: Order

`internal/entities/order-entity.go`

Поля заявки включают:

- `id`
- `name`
- оргструктуру:
  - `department_id`
  - `otdel_id`
  - `branch_id`
  - `office_id`
- `status_id`
- `priority_id`
- `order_type_id`
- `creator_id` (`user_id` в таблице)
- `executor_id`
- `equipment_id`
- `equipment_type_id`
- `address`
- `duration`
- `created_at`
- `updated_at`
- `completed_at`

Также у заявки есть KPI-поля:

- `first_response_time_seconds`
- `resolution_time_seconds`
- `is_first_contact_resolution`

## 10. HTTP API

Полная карта API уже собрана в `docs/API.md`.

Ниже архитектурная группировка.

### Auth

- login
- refresh token
- me
- logout
- update profile
- password reset flow

### Users

- CRUD пользователей
- AD search
- массовая привязка AD usernames
- просмотр и изменение индивидуальных прав пользователя

### Orders

- create
- list
- get by id
- update
- delete
- comments
- attachments
- history

### Dashboard / Report

- dashboard widgets с period/granularity/widgets
- report endpoints

### Org dictionaries

- branches
- departments
- otdels
- offices
- positions

### Service dictionaries

- statuses
- priorities
- order types
- routing rules
- equipment types
- equipment

### Integrations

- 1C sync webhook
- Telegram webhook
- WebSocket endpoint

## 11. Унификация ответов и ошибок

Главный helper: `pkg/utils/http_helpers.go`.

Стандартный success response:

```json
{
  "status": true,
  "message": "...",
  "body": ...
}
```

Стандартный error response:

```json
{
  "status": false,
  "message": "..."
}
```

Если есть детали, они кладутся в `body`.

### Error layer

`pkg/errors/errors.go`

Что умеет:

- базовые `HttpError`;
- typed errors для auth, validation, forbidden, not found и т.д.;
- map PostgreSQL constraint errors в понятные сообщения;
- fallback parsing DB text errors;
- обработку `varchar` overflow с выводом лимита, если его можно извлечь из текста ошибки.

Это важно для UX:

- user-friendly ответы приходят не только из controller/service;
- база тоже оборачивается в более понятный message.

## 12. Auth и безопасность

### JWT

`pkg/service/jwt.go`

Используется:

- `HS512`
- custom claims:
  - `userID`
  - `roleID`
  - `IsRefreshToken`

Есть:

- генерация access/refresh токенов;
- валидация token signature;
- проверка refresh token отдельно.

### Auth middleware

`pkg/middleware/auth.go`

Что делает:

- читает `Authorization: Bearer <token>`;
- валидирует JWT;
- отклоняет refresh token в protected API;
- загружает все права пользователя через `AuthPermissionService`;
- складывает в context:
  - `UserIDKey`
  - `UserRoleIDKey`
  - `UserPermissionsKey`
  - `UserPermissionsMapKey`

### RBAC

Проект использует permission-based модель, а не только роль.

Проверки делаются:

- route-level через `AuthorizeAny/AuthorizeAll`;
- service-level через `internal/authz`;
- field-level в order service.

Особенно детально это сделано для заявок:

- разные права на create/update отдельных полей;
- разные права на назначение исполнителя;
- scope-права для dashboard;
- отдельные права на upload attachment.

### Важный нюанс

Логика блокировки после нескольких неверных логинов пока не доведена до runtime, хотя конфиг и Redis keys уже существуют.

## 13. AuthService

`internal/services/auth.go`

Функции:

- `Login`
- `GetUserByID`
- `RequestPasswordReset`
- `VerifyResetCode`
- `ResetPassword`
- `UpdateMyProfile`

### Login flow

1. Поиск пользователя по email/login.
2. Проверка, что пользователь активен.
3. Если это system root login, пароль сравнивается локально.
4. Если LDAP включен:
   - берется AD username;
   - делается bind в LDAP.
5. Если LDAP выключен:
   - используется локальное сравнение hash password.
6. Если `MustChangePassword = true`, выдается специальный reset token через Redis.

### Password reset

Сейчас reset flow завязан на Telegram:

- генерируется 4-значный код;
- код кладется в Redis;
- если у пользователя привязан Telegram chat id, код отправляется ботом.

## 14. Работа с пользователями

Главная логика: `internal/services/user-service.go`.

Функциональность:

- CRUD пользователей;
- выдача permission details;
- update user permissions;
- генерация Telegram link token;
- привязка Telegram аккаунта;
- batch AD username binding;
- projected list mode для облегченных выборок.

### Projected fields

Сейчас `fields=` поддерживается не для всего проекта, а точечно.

По текущему controller-коду:

- `GET /api/users` поддерживает `fields=...` через whitelist в `internal/queryfields/dictionary_fields.go`;
- `position` и `equipment` fields-path принудительно отключены на controller-уровне.

## 15. Заявки: центральный домен проекта

Главный сервис: `internal/services/order.go` плюс разнесенные файлы:

- `order_service_create.go`
- `order_service_update.go`
- `order_service_delete.go`
- `order_service_list.go`
- `order_service_history.go`
- `order_service_metrics.go`
- `order_service_query_helpers.go`
- `order_service_validation.go`

### Что делает OrderService

- создает заявку;
- ищет исполнителя по routing rules или hierarchy fallback;
- валидирует обязательные поля по типу заявки;
- валидирует права на отдельные поля;
- работает с историей изменений;
- работает с attachment;
- считает и обновляет KPI;
- публикует события в event bus;
- дергает notification service;
- работает с cache invalidation для dashboard.

### Валидация по типу заявки

Сейчас в коде есть registry-правила.

Например для `EQUIPMENT` обязательны:

- `equipment_id`
- `equipment_type_id`
- `priority_id`

Для не-`EQUIPMENT` заявок при create/update требуется комментарий.

### Лимиты

Актуальные бизнес-лимиты из кода:

- `order.name` max `500`
- attachment file name max `500`
- upload `order_document` max `30 MB`
- comment явного max в коде сейчас не имеет

### Файлы

Валидация файлов идет через:

- `config/upload.go`
- `pkg/validation/file.go`

Поддерживаются:

- изображения;
- `pdf`;
- `doc`;
- `docx`;
- `odt`;
- `odp`;
- `ods`

Word/OpenDocument файлы нормализуются корректно, чтобы `docx` не падал как `application/zip`.

## 16. Маршрутизация исполнителя

Главный механизм: `internal/services/rule_engine_service.go`.

Есть два пути:

### 1. Явный routing rule

По таблице `order_routing_rules` ищется правило по:

- `order_type_id`
- `department_id`
- `otdel_id`
- `branch_id`
- `office_id`

Если правило найдено, система пытается найти человека по целевой позиции.

### 2. Hierarchy fallback

Если явного правила нет, либо правило есть, но человек по позиции не найден, включается fallback по оргструктуре.

Идея waterfall:

- department head / deputy
- otdel head / deputy / manager
- branch director / deputy
- office head / deputy

Если ответственный не найден, backend возвращает управляемую business error с предложением выбрать исполнителя вручную или настроить маршрутизацию.

### Ручной исполнитель

Если исполнитель выбран вручную, сервис проверяет:

- что пользователь существует;
- что он активен;
- что он относится к структуре заявки.

## 17. История заявки

История изменений вынесена в отдельный домен:

- repository `order_history-repository.go`
- service `order_history.go`
- controller `order_history.go`
- event `internal/events/order_events.go`

История используется не только как audit trail, но и как:

- источник уведомлений;
- источник dashboard last activity;
- источник метрик и итоговых KPI;
- источник timeline в UI;
- основа для attachment notifications.

## 18. Вложения и файловое хранилище

### Хранилище

`pkg/filestorage/local_filestorage.go`

Файлы сохраняются локально:

- base path: `uploads`
- структура: `prefix/YYYY/MM/DD/<date>-<uuid>.<ext>`

Примеры prefixes:

- `avatars`
- `orders`
- `icons/small`
- `icons/big`

### URL-логика

`pkg/utils/upload_urls.go`

Backend умеет:

- нормализовать `file_path`;
- собирать полный публичный URL через `SERVER_BASE_URL`.

Это нужно, потому что фронт и backend могут жить на разных origins, и относительный `/uploads/...` не всегда открывается корректно.

## 19. Dashboard

Главный сервис: `internal/services/dashboard_service.go`.

### Особенности

- dashboard разбит на widget-модель;
- frontend может запрашивать конкретные widgets;
- если widgets не переданы, backend грузит весь набор по умолчанию;
- widgets исполняются параллельно через `errgroup`;
- кешируются через Redis;
- используется `singleflight`, чтобы не дублировать одинаковые тяжелые вычисления.

### Поддерживаемые widget-группы

- alerts
- kpis
- sla
- weekly_volume
- time_by_priority
- time_by_order_type
- count_by_status
- count_by_executor
- top_categories
- departments
- branches
- last_activity

### Security scope

Dashboard scope вычисляется из RBAC:

- all
- department
- branch
- otdel
- office
- own

### Кеш

Dashboard кеш использует:

- user scope;
- permissions;
- actor structure;
- widgets;
- date range;
- version keys.

## 20. Кеш и Redis-слой

Redis repository: `internal/repositories/redis_cache_repository.go`.

Redis используется для:

- reset tokens и verification codes;
- Telegram state;
- Telegram screen invalidation;
- auth permission cache;
- dashboard cache;
- versioned dictionary caches;
- telegram link tokens;
- force password change tokens;
- прочих служебных lock/protection keys.

### Versioned list cache

`internal/services/versioned_list_cache.go`

Используется для маленьких справочников, чтобы:

- не чистить все ключи вручную;
- инвалидация шла через bump version key.

Сейчас это уже видно в сервисах:

- status
- priority
- permission

## 21. Notifications, EventBus и realtime

### Event bus

`pkg/eventbus/eventbus.go`

Это простой in-process event bus:

- subscribe by event name;
- publish async;
- каждый listener вызывается в goroutine;
- на listener навешивается timeout 1 минута;
- ошибки listener-ов логируются.

### Событие

Сейчас ключевое событие:

- `order.history.created`

### NotificationListener

`internal/listeners/notification_listener.go`

Что делает:

- группирует несколько history events одной заявки по `order_id + txid`;
- дает небольшое окно агрегации;
- определяет recipients:
  - creator
  - executor
  - участники истории
  - старый исполнитель при делегировании
- исключает самого actor;
- отправляет:
  - Telegram notification
  - WebSocket notification

### WebSocket

Слой realtime:

- controller: `internal/controllers/websocket_controller.go`
- package: `pkg/websocket`

Поддержка auth:

- `Authorization: Bearer ...`
- или `Sec-WebSocket-Protocol: bearer, <token>`

Hub хранит:

- все клиенты;
- клиентов по `userID`;
- умеет слать адресные сообщения пользователю.

## 22. Telegram subsystem

Это один из самых развитых и нетривиальных слоев проекта.

Основные файлы:

- `internal/controllers/telegram/controller.go`
- `commands.go`
- `actions_core.go`
- `actions_flow.go`
- `order_details.go`
- `executor_candidates.go`
- `screen.go`
- `screen_helpers.go`
- `request_deduplicator.go`

### Что умеет Telegram backend

- привязка аккаунта через deep-link token;
- webhook endpoint;
- меню и навигация;
- список заявок;
- карточка заявки;
- делегирование;
- смена статуса;
- текстовый поиск;
- защита от повторных кликов;
- автоочистка устаревших меню;
- информирование пользователя при задержке обработки;
- fallback в понятный экран ошибки.

### Как Telegram хранит состояние

Через Redis:

- ключ состояния пользователя;
- screen message id;
- deep-link token data;

### Stability-механизмы

В коде уже есть защита от типичных Telegram-проблем:

- дедупликация команд;
- cooldown на callback/menu;
- ограничение concurrent requests;
- background cleanup;
- timeout на goroutines;
- `Обрабатываю...` при долгом callback;
- удаление устаревших сообщений/экранов;
- invalidation экрана после уведомлений.

### Подбор сотрудников в Telegram

Для делегирования система использует структуру самой заявки, а не структуру текущего пользователя.

Это важно, потому что Telegram-UX завязан на актуальную структуру заявки.

## 23. Telegram integration service

`internal/services/telegram_integration_service.go`

Отвечает за:

- включенность Telegram по bot token;
- построение deep link;
- проверку secret token вебхука;
- регистрацию webhook;
- запрос `getWebhookInfo`.

Webhook может регистрироваться только если `SERVER_BASE_URL` начинается с `https://`.

## 24. AD / LDAP интеграция

`internal/services/ad_service.go`

Поддерживает:

- поиск пользователей в AD;
- поиск exact usernames батчами;
- bind через service account;
- configurable search filter;
- configurable attributes.

Используется для:

- поиска AD пользователей;
- массовой привязки AD usernames локальным пользователям;
- auth, если включен LDAP login.

## 25. 1С sync

### Route

`/api/sync/1c`

### Ограничения

- endpoint активен только если `ONE_C_API_KEY` задан;
- защищен `KeyAuth`;
- одновременный sync не допускается.

### SyncService

`internal/services/sync_service.go`

Использует:

- atomic guard `running`;
- background processing;
- structured logs;
- handler layer `internal/sync`.

### Какие данные приходят из 1С

По DTO и service-коду:

- departments
- otdels
- branches
- offices
- positions
- users

То есть 1С здесь выступает источником оргструктуры и части кадровых данных.

## 26. Upload и validation

### Upload contexts

`config/upload.go`

Контексты:

- `profile_photo`
- `order_document`
- `icon_small`
- `icon_big`

### Validation

`pkg/validation`

Используется для:

- проверки mime type;
- проверки размера;
- проверки image-constraints;
- корректной обработки office document mime detection.

## 27. Query, filters, pagination

Главный helper: `pkg/utils/http_helpers.go`.

Поддерживаемый query-формат:

- `search=...`
- `fields=a,b,c`
- `limit=...`
- `page=...`
- `offset=...`
- `withPagination=true|false`
- `include_attachments=true`
- `sort[field]=asc|desc`
- `filter[field]=value`

### Важно

Сейчас глобальные значения:

- `DefaultLimit = 2000`
- `MaxLimit = 2000`

Это исторически очень большой дефолт и одна из причин тяжелых list-ответов. Архитектурно это место требует осторожности при дальнейшем тюнинге, потому что изменение дефолта может затронуть старые экраны.

## 28. Справочники и performance-паттерны

В коде уже видны прикладные оптимизации для справочников:

- `gzip` на уровне HTTP;
- versioned list cache для маленьких справочников;
- projected queries для users;
- поля `fields=` только по whitelist;
- у `position` и `equipment` fields-path отключен;
- у Telegram есть облегченные candidate-path и дополнительные кеши состояния;
- dashboard идет через parallel widget execution и Redis cache.

## 29. Role/permission модель

Судя по структуре миграций и сервисов, проект использует смешанную модель:

- roles
- role_permissions
- user_permissions
- user_permission_denials
- user_roles

То есть backend умеет:

- базовые права через роль;
- индивидуальные grant для пользователя;
- индивидуальные deny.

Это более гибкая RBAC/ABAC-гибридная модель, чем простое `user has one role`.

## 30. Сильные стороны архитектуры

- Четкое разделение по слоям.
- Хорошо разведен core service слой.
- Есть event-driven кусок для уведомлений.
- Богатая Telegram-логика без выноса в отдельный сервис.
- Реальный RBAC на уровне полей заявки.
- Dashboard не монолитный, а widget-based.
- 1С sync отделен и защищен API key.
- Есть Redis caching и dictionary cache versioning.
- Ошибки БД мапятся в человеческие сообщения.
- Автоматические миграции на старте упрощают rollout.

## 31. Технические компромиссы и особенности

### 1. Один бинарник делает многое

Плюс:

- простой деплой.

Минус:

- entrypoint разрастается;
- API, seeders, imports и webhook wiring живут рядом.

### 2. Большой `DefaultLimit`

Исторически удобно, но плохо для performance.

### 3. Telegram-контроллер очень толстый

Это рабочее решение, но Telegram-подсистема уже достаточно большая, чтобы со временем проситься в более явный application layer / state machine.

### 4. Auth lockout завершен

Конфиг, Redis-ключи и enforcement уже есть: `AuthService.Login` блокирует аккаунт после превышения лимита неудачных попыток.

### 5. Часть runtime-поведения завязана на env и naming conventions

Например:

- head branch names;
- webhook behavior;
- LDAP search patterns;
- dashboard worker count.

## 32. Что уже важно знать новому backend-разработчику

Если новый человек заходит в проект, ему надо в первую очередь прочитать:

1. `app/main.go`
2. `internal/routes/routes.go`
3. `internal/routes/wiring.go`
4. `docs/API.md`
5. `internal/services/order.go` и соседние `order_service_*`
6. `internal/services/rule_engine_service.go`
7. `internal/services/auth.go`
8. `internal/services/dashboard_service.go`
9. `internal/controllers/telegram/controller.go`
10. `internal/listeners/notification_listener.go`

После этого уже читать:

- `pkg/middleware/auth.go`
- `pkg/errors/errors.go`
- `pkg/utils/http_helpers.go`
- `config/upload.go`
- `internal/queryfields/dictionary_fields.go`

## 33. Как я бы описал backend в одном абзаце

Это монолитный Go backend для service-desk / request-management системы, где вокруг домена заявок построены RBAC, оргструктура, routing rules, история изменений, вложения, Telegram-бот, WebSocket-уведомления, dashboard-аналитика и 1С/AD интеграции. Архитектура слоистая, с PostgreSQL как source of truth, Redis как служебным кешем и state-store, а центральная бизнес-логика сосредоточена в `services` вокруг заявок, пользователей, маршрутизации и realtime-уведомлений.

## 34. Где смотреть дальше

Если нужен следующий уровень детализации:

- точные endpoint contracts: `docs/API.md`
- детали rollout и smoke-check: `docs/deployment-checklist.md`
- dashboard/load profiling: `docs/loadtest-checklist.md`
- ручная регрессия: `docs/manual-regression-checklist.md`
- HTTPS по IP и AD CS: `docs/ad-ip-certificate-rollout.md`

## 35. Короткий итог

Backend уже не маленький CRUD-сервис. Это зрелый монолит с несколькими подсистемами:

- core business layer вокруг заявок;
- rights and identity layer;
- realtime notifications;
- Telegram application layer;
- analytics layer;
- integration layer.

Главное в этом проекте это не отдельный endpoint, а связка:

`Order -> Routing -> History -> Notifications -> Telegram/WebSocket -> Dashboard -> Integration`

Именно эта цепочка лучше всего описывает, как backend реально живет в проде.
