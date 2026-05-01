# Backend Current State

This document describes the current, real backend implementation of `request-system` as it exists in this repository.
It is meant to answer:

- what the project does,
- how requests flow through the system,
- what is already implemented,
- what is intentionally not implemented,
- and where the important operational settings live.

For a deeper architectural decomposition, see `docs/BACKEND_ARCHITECTURE.md`.

## 1. What This Project Is

`request-system` is a production Go backend for a helpdesk / service-request platform.
It is not a toy CRUD API. The codebase already covers:

- request creation and processing,
- routing and executor assignment,
- request history and audit-like tracking,
- file attachments,
- users, roles, permissions and organizational structure,
- dashboards and reports,
- Telegram-based request management,
- WebSocket notifications,
- Active Directory / LDAP integration,
- 1C synchronization,
- local file storage for uploads.

The runtime is an HTTPS Echo server with PostgreSQL as the main database and Redis as the shared cache/session store.

## 2. Stack

The backend currently uses:

- Go 1.25.x,
- Echo v4,
- PostgreSQL via `pgx/v5`,
- Redis via `go-redis/v8`,
- Goose for migrations,
- Zap for structured logging,
- JWT for auth tokens,
- LDAP for AD/organizational lookup,
- Gorilla WebSocket for realtime notifications,
- local disk storage for files under `uploads/`.

## 3. Main Entry Flow

The application entrypoint is [app/main.go](../app/main.go).

On startup the backend:

1. loads configuration from `.env` and environment variables,
2. sets application timezone,
3. applies proxy environment overrides if present,
4. supports CLI modes for seeders/imports,
5. creates loggers,
6. runs Goose migrations,
7. creates the Echo app,
8. enables Recover, Gzip and CORS middleware,
9. connects to PostgreSQL and Redis,
10. creates JWT, auth-permission, event bus and websocket services,
11. wires repositories, services and controllers,
12. starts the HTTPS server,
13. runs the websocket hub in the background,
14. blocks until shutdown signal.

Important runtime facts:

- the server is HTTPS-first and uses `ListenAndServeTLS`,
- static files are served from `/uploads`,
- `GET /ping` is the simplest health check,
- migrations run at startup; if they fail, the server does not continue.

## 4. High-Level Architecture

The backend follows a clean layered flow:

`Route -> Controller -> Service -> Repository -> PostgreSQL / Redis`

Cross-cutting layers:

- `pkg/middleware` for authentication and authorization,
- `pkg/eventbus` and `internal/listeners` for internal events,
- `pkg/websocket` for realtime pushes,
- `pkg/filestorage` for file saving,
- `pkg/utils` for response helpers, filtering and upload URL normalization,
- `pkg/errors` for a consistent HTTP error model.

### Typical request flow

1. Echo receives the request.
2. Auth middleware validates the JWT access token.
3. Middleware places user identity, role and permissions into request context.
4. Controller parses query/body/form data and converts it into DTOs and `types.Filter`.
5. Service layer applies business rules and runs transactions if needed.
6. Repository layer executes SQL against PostgreSQL or operations against Redis.
7. Internal listeners may publish Telegram/WebSocket side effects.
8. Controller returns a standardized JSON response.

## 5. Current Domain Areas

### 5.1 Auth

Auth is built around access/refresh JWT plus a server-side refresh-session registry.

Current behavior:

- login validates credentials against local password hash or LDAP/AD depending on configuration,
- login protection includes failed-attempt tracking and lockout,
- refresh tokens carry a `sessionID`,
- refresh sessions are stored server-side and can be invalidated,
- logout revokes the current refresh session,
- password reset uses a code/token flow,
- `/auth/me` returns current profile data,
- profile update supports photo upload.

Important files:

- [internal/controllers/auth.go](../internal/controllers/auth.go)
- [internal/services/auth.go](../internal/services/auth.go)
- [pkg/service/jwt.go](../pkg/service/jwt.go)
- [pkg/middleware/auth.go](../pkg/middleware/auth.go)

### 5.2 Requests / Orders

The request lifecycle includes:

- create,
- update,
- delete,
- assignment / delegation,
- status changes,
- priority changes,
- attachment handling,
- history generation,
- Telegram notifications.

Routing is not a simple static lookup.
It supports:

- explicit rules,
- partial rules with nullable fields as wildcards,
- hierarchy fallback if a rule does not produce an executor,
- special handling for organizational edge cases such as `HEAD_BRANCH_NAMES`.

Important files:

- [internal/services/order_service_create.go](../internal/services/order_service_create.go)
- [internal/services/order_service_update.go](../internal/services/order_service_update.go)
- [internal/services/order_service_history.go](../internal/services/order_service_history.go)
- [internal/services/order_routing_rule_service.go](../internal/services/order_routing_rule_service.go)
- [internal/services/rule_engine_service.go](../internal/services/rule_engine_service.go)
- [internal/controllers/order.go](../internal/controllers/order.go)

### 5.3 Order History

`order_history` is central in this project.
It is used as:

- timeline / audit trail for requests,
- source for notification side effects,
- data source for dashboard and activity views,
- source for attachment-related URLs in notifications.

Important file:

- [internal/services/order_history.go](../internal/services/order_history.go)

### 5.4 Files / Attachments

The backend stores files on local disk in the `uploads/` directory.

Current behavior:

- profile photos can be uploaded through existing profile flows,
- request attachments are stored and returned as URLs under `/uploads/...`,
- status icons and other image-based dictionary fields also use the same storage model,
- the application serves `uploads` statically from the backend process.

Important note:

- a generic `UploadController` exists in code, but it is not mounted in the router at the moment,
- so the practical file-upload paths are the embedded flows already wired into business endpoints.

Important files:

- [pkg/filestorage/local_filestorage.go](../pkg/filestorage/local_filestorage.go)
- [pkg/utils/upload_urls.go](../pkg/utils/upload_urls.go)
- [internal/controllers/upload_controller.go](../internal/controllers/upload_controller.go)

### 5.5 Telegram

Telegram is a first-class user channel, not an afterthought.

Current behavior:

- current-user Telegram binding status,
- Telegram link token generation,
- unlink flow,
- webhook handling,
- request card views,
- executor selection,
- callback actions,
- stale screen protection,
- state cleanup after notifications,
- request delegation and status changes from Telegram.

Important files:

- [internal/controllers/telegram](../internal/controllers/telegram)
- [internal/routes/telegram_router.go](../internal/routes/telegram_router.go)
- [internal/listeners/notification_listener.go](../internal/listeners/notification_listener.go)

### 5.6 Dashboard / Reports

The dashboard is a separate production path.
It uses cache-aware and parallelized data retrieval, not a single giant query.

Important files:

- [internal/services/dashboard_service.go](../internal/services/dashboard_service.go)
- [internal/controllers/dashboard.go](../internal/controllers/dashboard.go)
- [internal/services/report_service.go](../internal/services/report_service.go)

### 5.7 Users, Org Structure and Permissions

The project contains the full internal structure model:

- users,
- roles,
- permissions,
- role-permission bindings,
- branches,
- departments,
- otdels,
- offices,
- positions,
- priorities,
- statuses,
- order types,
- order routing rules,
- equipment and equipment types.

Many of these lists support selective lightweight responses or fields projection for performance-sensitive UI lookup paths.

Important files:

- [internal/controllers/user.go](../internal/controllers/user.go)
- [internal/controllers/position.go](../internal/controllers/position.go)
- [internal/controllers/equipment.go](../internal/controllers/equipment.go)
- [internal/queryfields/dictionary_fields.go](../internal/queryfields/dictionary_fields.go)

### 5.8 Integrations

The backend already includes production integration surfaces:

- LDAP / AD search and auth support,
- 1C sync endpoint,
- optional online-bank integration config,
- websocket notifications,
- Telegram notification delivery.

Important files:

- [internal/services/auth.go](../internal/services/auth.go)
- [internal/services/sync_service.go](../internal/services/sync_service.go)
- [internal/routes/sync_router.go](../internal/routes/sync_router.go)
- [pkg/websocket](../pkg/websocket)
- [pkg/eventbus](../pkg/eventbus)

## 6. Response Contract

The backend uses a consistent response envelope from `pkg/utils/http_helpers.go`.

### Success

```json
{
  "status": true,
  "message": "OK",
  "body": {}
}
```

When pagination is enabled, list responses are wrapped like this:

```json
{
  "status": true,
  "message": "OK",
  "body": {
    "list": [],
    "pagination": {
      "total_count": 0,
      "page": 1,
      "limit": 20,
      "total_pages": 0
    }
  }
}
```

### Error

```json
{
  "status": false,
  "message": "human readable error"
}
```

Some validation and database errors map into user-friendly messages in `pkg/errors`.

## 7. Current Backend Optimizations

The codebase already contains several performance-oriented changes:

- gzip response middleware,
- selective `fields=` support for heavy lookup lists,
- projected repository reads for users, positions and equipment,
- Redis-backed cache for session and auth state,
- login lockout,
- refresh-session registry and invalidation,
- query helpers for normalized upload URLs,
- cache-aware dashboard/list flows,
- in-process event bus to reduce direct coupling between request flows and notification delivery.

## 8. Current Gaps and Tradeoffs

The backend is functional and production-oriented, but it still has tradeoffs:

- it is still a monolith,
- notification delivery is event-driven but still process-bound,
- some list contracts are still legacy-style and not yet standardized to a single universal envelope across every endpoint,
- the generic upload controller is not routed yet,
- the codebase contains several large services and controllers because it evolved in production, not from a blank architecture exercise.

These are not blockers for local deployment or frontend integration, but they are the right places for future cleanup.

## 9. Key Files to Read First

If you want to understand the backend quickly, start here:

- [app/main.go](../app/main.go)
- [internal/routes/routes.go](../internal/routes/routes.go)
- [pkg/config/config.go](../pkg/config/config.go)
- [pkg/utils/http_helpers.go](../pkg/utils/http_helpers.go)
- [pkg/errors/errors.go](../pkg/errors/errors.go)
- [internal/controllers/auth.go](../internal/controllers/auth.go)
- [internal/services/auth.go](../internal/services/auth.go)
- [internal/services/order_service_create.go](../internal/services/order_service_create.go)
- [internal/services/order_routing_rule_service.go](../internal/services/order_routing_rule_service.go)
- [internal/controllers/telegram](../internal/controllers/telegram)

