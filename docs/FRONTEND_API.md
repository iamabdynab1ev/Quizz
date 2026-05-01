# Frontend API Guide

This document is the frontend-facing contract for the current `request-system` backend.
It answers:

- which endpoints exist,
- how auth works,
- how list filtering/pagination works,
- how files and WebSocket URLs should be handled,
- which frontend assumptions are safe.

Base API prefix: `/api`

Health endpoint outside the API prefix: `GET /ping`

## 1. Response Shape

The backend uses a consistent JSON envelope:

```json
{
  "status": true,
  "message": "OK",
  "body": {}
}
```

Error responses look like:

```json
{
  "status": false,
  "message": "Readable error message"
}
```

For paginated lists, the backend currently wraps the body like this:

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

If pagination is disabled for a request, `body` is usually the raw list/object.

## 2. Auth Rules

### Access token

- Sent in `Authorization: Bearer <access_token>`.
- Used for all protected API calls.

### Refresh token

- Stored in an HttpOnly cookie named `refreshToken`.
- Frontend should not read or store it in local storage.
- Refresh flow relies on the cookie being sent with the request.

### Logout

- Logout clears the refresh cookie and invalidates the server-side refresh session.

### Lockout

- Login is rate-protected and can return an account-lock error after repeated failures.

## 3. Query Parameters

Most list endpoints support these query params:

- `search` - text search,
- `limit` - page size,
- `offset` - offset-based pagination,
- `page` - page number,
- `withPagination=true|false` - enable or disable pagination envelope,
- `sort[field]=asc|desc` - sort by field,
- `filter[field]=value` - exact or structured filter depending on endpoint,
- `fields=id,name,...` - selective projection for some heavy list endpoints.

### Important note about `fields`

Selective `fields=` is supported only on some heavy lookup-style list endpoints.
At the moment it is intended for:

- `users`
- `positions`
- `equipment`

For those endpoints:

- `id` is always included,
- unknown fields are rejected,
- use it only when the UI needs a lightweight selector/lookup response.

Example:

```text
GET /api/users?fields=id,fio,phone_number&search=ali&limit=20
GET /api/position?fields=id,name,type&limit=20
GET /api/equipment?fields=id,name&search=printer&limit=20
```

## 4. File URLs

The backend serves files under `/uploads/...`.

Frontend rules:

- use the returned `url` as-is,
- do not manually rewrite or double-prefix the path,
- if the backend returns a full URL, open it directly,
- if it returns a relative `/uploads/...` path, use it as the browser URL.

## 5. WebSocket

- Endpoint: `GET /api/ws`
- Authentication:
  - `Authorization: Bearer <token>`, or
  - `Sec-WebSocket-Protocol: bearer, <token>`

Use WebSocket for realtime updates if the frontend needs live notifications.

## 6. Auth Endpoints

- `POST /api/auth/login`
- `POST /api/auth/refresh_token`
- `GET /api/auth/me`
- `POST /api/auth/logout`
- `PUT /api/auth/me`
- `POST /api/auth/password/request`
- `POST /api/auth/password/verify_phone`
- `POST /api/auth/password/reset`

### Login payload

```json
{
  "login": "user@example.com",
  "password": "secret",
  "remember_me": true
}
```

### Password reset flow

1. `POST /api/auth/password/request`
2. `POST /api/auth/password/verify_phone`
3. `POST /api/auth/password/reset`

## 7. Users and AD

- `GET /api/users`
- `POST /api/users`
- `GET /api/users/:id`
- `PUT /api/users/:id`
- `DELETE /api/users/:id`
- `GET /api/users/permission/:id`
- `PUT /api/users/permission/:id`
- `GET /api/ad-users`
- `POST /api/users/bind-ad-usernames`

Use cases:

- admin user management,
- permission management,
- AD search and username binding.

## 8. Orders / Requests

- `POST /api/order`
- `GET /api/order`
- `GET /api/order/:id`
- `PUT /api/order/:id`
- `DELETE /api/order/:id`
- `GET /api/order/:orderID/history`
- `GET /api/order-comments`
- `GET /api/order-comment/:id`
- `POST /api/order-comment`
- `PUT /api/order-comment/:id`
- `DELETE /api/order-comment/:id`
- `GET /api/attachment`
- `DELETE /api/attachment/:id`

### Frontend notes

- order creation and update can include file data,
- history is important for timeline rendering,
- attachment URLs come from backend and should be used directly.

## 9. Dashboard and Reports

- `GET /api/dashboard`
- `GET /api/report`
- `GET /api/main`

Frontend should request only the widgets/period/scope it needs.

## 10. Equipment and Equipment Types

- `GET /api/equipment`
- `POST /api/equipment`
- `GET /api/equipment/:id`
- `PUT /api/equipment/:id`
- `DELETE /api/equipment/:id`
- `POST /api/equipment-import`
- `GET /api/equipment-types`
- `POST /api/equipment-types`
- `GET /api/equipment-types/:id`
- `PUT /api/equipment-types/:id`
- `DELETE /api/equipment-types/:id`

## 11. Dictionaries / Structure

Use the exact backend paths below. Some resources are singular in the API even if they represent collections.

- Branches: `/api/branch`
- Departments: `/api/department`
- Otdels: `/api/otdel`
- Offices: `/api/office`
- Positions: `/api/position`
- Priorities: `/api/priority`
- Statuses: `/api/status`
- Roles: `/api/role`
- Permissions: `/api/permission`
- Role permissions: `/api/role_permission`
- Order types: `/api/order_type`
- Order routing rules: `/api/order_rule`

Standard CRUD methods:

- `GET <base>` - list,
- `POST <base>` - create,
- `GET <base>/:id` - get by ID,
- `PUT <base>/:id` - update,
- `DELETE <base>/:id` - delete.

### Extra endpoints

- `GET /api/position/types` - position types list
- `GET /api/order_type/:id/config` - order type config for create/update UI

## 12. Telegram Integration Endpoints

These are mostly current-user and webhook endpoints:

- `GET /api/profile/telegram`
- `DELETE /api/profile/telegram`
- `POST /api/profile/telegram/generate-token`
- `POST /api/webhooks/telegram` - backend webhook endpoint used by Telegram, not a normal UI action.

Frontend usually needs only:

- link status,
- generate token,
- unlink state.

## 13. 1C Sync

- `POST /api/sync/1c` - backend sync endpoint, not a frontend user-flow call.

Notes:

- protected by API key middleware,
- not a frontend user-flow endpoint,
- used by external 1C sync processes.

## 14. Frontend Behavior Rules

Frontend should:

- always send the access token in `Authorization`,
- keep the refresh token in the HttpOnly cookie only,
- use `withCredentials: true` if API is cross-origin,
- not assume every list response has the same shape,
- treat `/uploads/...` links as backend-owned URLs,
- prefer `fields=` for heavy selector data where supported,
- use `withPagination=true` when the UI needs page counts,
- not manually concatenate backend host names to file URLs unless the backend returns a relative path.

## 15. Example Calls

```text
GET /api/users?search=ali&limit=20&withPagination=true&fields=id,fio,phone_number
GET /api/order?limit=20&page=1&filter[status_id]=3
GET /api/dashboard?period=month
GET /api/profile/telegram
GET /api/ws
```

## 16. What Frontend Should Not Expect

- There is no fully standardized universal `data/total` envelope across all endpoints yet.
- There is no generic public upload route wired into the router at the moment.
- Some admin-only routes are intentionally behind permissions and will return `403`.

## 17. Practical Frontend File Rules

If the UI renders:

- profile photos,
- attachments,
- status icons,
- equipment icons,

it should keep the URL returned by backend and let the browser fetch it directly.
