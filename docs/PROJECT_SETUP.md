# Project Setup, Configuration and Connections

This file collects the runtime settings, external connections and startup modes for `request-system`.
Use it as the single place to prepare the backend for development, testing or deployment.

## 1. What Must Be Available

Minimum runtime dependencies:

- Go 1.25.x,
- PostgreSQL,
- Redis,
- a valid `.env` file or equivalent environment variables,
- TLS certificate and key for the HTTPS server.

Optional external dependencies:

- LDAP / Active Directory,
- Telegram bot token,
- 1C API key,
- online bank integration credentials,
- proxy settings for outbound HTTP access.

## 2. How the Application Starts

Main entrypoint: [app/main.go](../app/main.go)

Default command:

```powershell
go run ./app
```

Build:

```powershell
go build ./...
```

The server starts as HTTPS and runs migrations on boot.

If migrations fail, startup fails.

## 3. Core Configuration File

Configuration is loaded from environment variables by [pkg/config/config.go](../pkg/config/config.go).

The project currently expects the following groups of settings.

### 3.1 Application and HTTP

- `APP_NAME` - application name shown in logs.
- `SERVER_PORT` - HTTPS port.
- `SERVER_BASE_URL` - public server URL used in generated links.
- `ALLOWED_ORIGINS` - CORS allowlist, comma separated.
- `APP_TIMEZONE` - application timezone.
- `SSL_CERT_PATH` - TLS certificate path.
- `SSL_KEY_PATH` - TLS key path.
- `LOG_LEVEL` - logger level used by `app/main.go`.
- `HTTP_PROXY_URL` - optional outbound proxy.
- `NO_PROXY_LIST` - optional bypass list for proxy.

### 3.2 PostgreSQL

- `DATABASE_URL` - required PostgreSQL DSN.
- `DB_POOL_MAX_CONNS` - max pool size.
- `DB_POOL_MIN_CONNS` - min pool size.
- `DB_POOL_MAX_CONN_LIFETIME_MINUTES` - max connection lifetime in minutes.
- `DB_POOL_MAX_CONN_IDLE_MINUTES` - max idle time in minutes.
- `DB_POOL_HEALTH_CHECK_PERIOD_SECONDS` - pool health check period in seconds.

Example:

```env
DATABASE_URL=postgres://quiz_user:password@127.0.0.1:5432/request_system?sslmode=disable
```

### 3.3 Redis

- `REDIS_ADDRESS` - Redis host:port.
- `REDIS_PASSWORD` - Redis password, if any.

Redis is used for:

- auth lockout counters,
- refresh-session registry,
- permission caching,
- Telegram/internal state cache flows,
- other short-lived runtime keys.

### 3.4 JWT / Auth

- `JWT_SECRET_KEY` - required JWT signing secret.
- `AUTH_MAX_LOGIN_ATTEMPTS` - failed login threshold before lockout.
- `AUTH_LOCKOUT_DURATION` - how long the account stays locked.
- `AUTH_MAX_RESET_ATTEMPTS` - failed reset-code threshold.
- `AUTH_RESET_TOKEN_TTL` - password reset token TTL.
- `AUTH_VERIFICATION_CODE_TTL` - verification code TTL.

### 3.5 Seeder

- `SEED_ADMIN_EMAIL` - bootstrap/admin email.
- `SEED_ADMIN_PASSWORD` - bootstrap/admin password.

### 3.6 Integrations

- `INTEGRATION_ACTIVE_PROVIDER` - active integration provider name.
- `ONE_C_API_KEY` - 1C sync protection key.
- `DEFAULT_ROLES_FOR_1C_USERS` - comma-separated roles for 1C imported users.
- `ONLINEBANK_BASE_URL` - online bank base URL.
- `ONLINEBANK_USERNAME` - online bank username.
- `ONLINEBANK_PASSWORD` - online bank password.

### 3.7 Telegram

- `TELEGRAM_BOT_TOKEN` - Telegram bot token.
- `TELEGRAM_BOT_USERNAME` - bot username without `@`.
- `TELEGRAM_WEBHOOK_SECRET_TOKEN` - optional webhook secret validation token.
- `TELEGRAM_ADVANCED_MODE_ENABLED` - enables advanced Telegram UX behavior.

### 3.8 Frontend

- `FRONTEND_BASE_URL` - public frontend URL used in notification links and redirects.

### 3.9 LDAP / Active Directory

- `LDAP_ENABLED` - turn LDAP auth on or off.
- `LDAP_SEARCH_ENABLED` - enable AD search lookups.
- `LDAP_HOST` - LDAP host.
- `LDAP_PORT` - LDAP port.
- `LDAP_DOMAIN` - domain prefix used in bind.
- `LDAP_BIND_DN` - bind DN for searches.
- `LDAP_BIND_PASSWORD` - bind password.
- `LDAP_TIMEOUT_SECONDS` - LDAP timeout.
- `LDAP_SEARCH_BASE_DN` - base DN for searches.
- `LDAP_SEARCH_FILTER_PATTERN` - search filter pattern.
- `LDAP_SEARCH_ATTRIBUTES` - searchable LDAP attributes.
- `LDAP_SEARCH_ATTR_USERNAME` - username attribute.
- `LDAP_SEARCH_ATTR_FIO` - display-name/FIO attribute.

## 4. Typical `.env` Skeleton

```env
APP_NAME=QUIZ
APP_TIMEZONE=Asia/Tashkent
LOG_LEVEL=debug

SERVER_PORT=8091
SERVER_BASE_URL=https://localhost:8091
ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
SSL_CERT_PATH=./certs/server.crt
SSL_KEY_PATH=./certs/server.key

DATABASE_URL=postgres://quiz_user:password@127.0.0.1:5432/request_system?sslmode=disable
REDIS_ADDRESS=127.0.0.1:6379
REDIS_PASSWORD=

JWT_SECRET_KEY=change-me
AUTH_MAX_LOGIN_ATTEMPTS=5
AUTH_LOCKOUT_DURATION=15m
AUTH_MAX_RESET_ATTEMPTS=5
AUTH_RESET_TOKEN_TTL=15m
AUTH_VERIFICATION_CODE_TTL=15m

SEED_ADMIN_EMAIL=admin@local
SEED_ADMIN_PASSWORD=Admin123!

INTEGRATION_ACTIVE_PROVIDER=mock
ONE_C_API_KEY=
DEFAULT_ROLES_FOR_1C_USERS=USER
ONLINEBANK_BASE_URL=
ONLINEBANK_USERNAME=
ONLINEBANK_PASSWORD=

TELEGRAM_BOT_TOKEN=
TELEGRAM_BOT_USERNAME=
TELEGRAM_WEBHOOK_SECRET_TOKEN=
TELEGRAM_ADVANCED_MODE_ENABLED=false

FRONTEND_BASE_URL=http://localhost:3000

LDAP_ENABLED=false
LDAP_SEARCH_ENABLED=false
LDAP_HOST=ldap.local
LDAP_PORT=389
LDAP_DOMAIN=
LDAP_BIND_DN=
LDAP_BIND_PASSWORD=
LDAP_TIMEOUT_SECONDS=10
LDAP_SEARCH_BASE_DN=
LDAP_SEARCH_FILTER_PATTERN=(&(objectClass=person)(sAMAccountName=%s))
LDAP_SEARCH_ATTRIBUTES=sAMAccountName,displayName,mail
LDAP_SEARCH_ATTR_USERNAME=sAMAccountName
LDAP_SEARCH_ATTR_FIO=displayName
```

## 5. Startup Modes

### 5.1 Normal production-style boot

```powershell
go run ./app
```

This mode:

- loads config,
- runs migrations,
- starts HTTPS server,
- starts websocket hub,
- registers routes and listeners.

### 5.2 Seeder mode

```powershell
go run ./seeders/cmd/seed/main.go -core
go run ./seeders/cmd/seed/main.go -roles
go run ./seeders/cmd/seed/main.go -all
```

Seeder mode is used for:

- core dictionaries,
- role/admin bootstrap,
- bulk initialization.

### 5.3 Build mode

```powershell
go build -o bin/api.exe ./app
```

On Linux:

```bash
go build -o bin/api ./app
```

## 6. External Connections

### 6.1 PostgreSQL

The application opens a `pgxpool` and uses Goose migrations at startup.

Connection settings live in:

- `DATABASE_URL`
- `DB_POOL_*` variables

If the database is unreachable or the DSN is wrong, startup stops immediately.

### 6.2 Redis

Redis is required for:

- lockout,
- refresh-session registry,
- permission cache,
- notification and Telegram state flows.

If Redis is down, the backend may still start, but auth/session related flows degrade or fail depending on the code path.

### 6.3 File Storage

Files are stored on local disk under `uploads/`.

The app also serves static files from `/uploads`.

Current file flows include:

- request attachments,
- user profile photos,
- status icons,
- equipment-related file URLs.

### 6.4 Telegram

Telegram features depend on:

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_BOT_USERNAME`
- optional `TELEGRAM_WEBHOOK_SECRET_TOKEN`

Telegram links and notifications also use:

- `SERVER_BASE_URL`
- `FRONTEND_BASE_URL`

### 6.5 LDAP / AD

LDAP is optional and controlled by:

- `LDAP_ENABLED`
- `LDAP_SEARCH_ENABLED`

When enabled, auth can bind to AD and users can be searched in the directory.

### 6.6 1C

1C sync is enabled only when `ONE_C_API_KEY` is set.

The sync endpoint is protected by an API key middleware.

## 7. Deployment Patterns

### 7.1 Direct HTTPS backend

This is the current native runtime mode.

Requirements:

- valid certificate,
- valid private key,
- `SERVER_PORT`,
- `SERVER_BASE_URL` matching the public URL.

### 7.2 Reverse proxy in front of HTTPS backend

You can place Nginx/Caddy in front of the Go server.

In that case:

- frontend should call the proxy URL,
- `ALLOWED_ORIGINS` should include the frontend origin,
- `SERVER_BASE_URL` should still match the public URL seen by clients.

### 7.3 Local development

For local development:

- keep PostgreSQL and Redis running locally,
- set `SERVER_BASE_URL` and `FRONTEND_BASE_URL` to local URLs,
- use a local certificate or a trusted dev cert.

## 8. Startup Order Checklist

1. Start PostgreSQL.
2. Start Redis.
3. Make sure `.env` is present.
4. Verify `DATABASE_URL` and `JWT_SECRET_KEY`.
5. Verify `SERVER_PORT`, `SERVER_BASE_URL`, `ALLOWED_ORIGINS`.
6. Verify TLS certificate and key paths.
7. Run the backend.
8. Check `GET /ping`.
9. Check auth login.
10. Check one list endpoint and one file URL.

## 9. Common Failure Points

- wrong PostgreSQL password,
- wrong TLS certificate path,
- empty `JWT_SECRET_KEY`,
- missing Redis connection,
- `ALLOWED_ORIGINS` missing the frontend origin,
- incorrect `SERVER_BASE_URL`,
- missing `TELEGRAM_BOT_TOKEN` if Telegram features are used,
- LDAP enabled but not configured.

## 10. What Frontend and Ops Should Remember

- the backend is HTTPS-first,
- cookies are used for refresh tokens,
- CORS must include the browser origin,
- file URLs are served from `/uploads`,
- migrations run on startup,
- all runtime behavior is controlled by environment variables, not hardcoded constants.

