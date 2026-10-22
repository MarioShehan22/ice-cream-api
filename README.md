# Ice Cream Shop API (Go + Fiber)

Backend for an ice cream shop: JWT authentication, role-based access control (Owner / Manager / Cashier), and CRUD APIs for **inventory items** and **customers**.

**Stack:** Go 1.21+, Fiber v2, GORM + SQLite (pure-Go driver, no CGO), JWT (HS256), bcrypt, go-playground/validator.

## Quick start

```bash
git clone <your-repo-url> && cd icecream-api
cp .env.example .env          # on Windows: copy .env.example .env
go mod tidy
go run ./cmd/server
```

The API listens on `http://localhost:8080`. On first start it creates `ice_cream.db`, runs migrations, and seeds data (idempotent).

Run the tests (in-memory SQLite, no setup needed):

```bash
go test ./... -v
```

## Seed users

| Role    | Email                   | Password     |
|---------|-------------------------|--------------|
| Owner   | owner@icecream.test     | Owner@123    |
| Manager | manager@icecream.test   | Manager@123  |
| Cashier | cashier@icecream.test   | Cashier@123  |

Seed data also includes categories (Chocolate, Vanilla, Strawberry), 6 inventory items, and 3 customers. Set `SEED_ON_START=false` to disable.

## Role permissions

| Action                  | Owner | Manager | Cashier |
|-------------------------|:-----:|:-------:|:-------:|
| Items - view            | yes   | yes     | yes     |
| Items - add             | yes   | yes     | **no**  |
| Items - edit            | yes   | yes     | yes     |
| Items - delete          | yes   | **no**  | **no**  |
| Customers - view        | yes   | yes     | yes     |
| Customers - add         | yes   | yes     | yes     |
| Customers - edit        | yes   | yes     | yes     |
| Customers - delete      | yes   | **no**  | **no**  |

Enforced server-side in `internal/server/server.go` via `middleware.RequireRoles(...)`. Unauthenticated requests get `401`, authenticated-but-forbidden requests get `403`.

## Endpoints

| Method        | Path                  | Notes |
|---------------|-----------------------|-------|
| POST          | `/api/auth/login`     | `{ "email", "password" }` -> JWT |
| POST          | `/api/auth/logout`    | revokes the current token (stored by its `jti` until expiry) |
| GET           | `/api/auth/me`        | current user |
| GET           | `/api/categories`     | flavours (read-only) - use for `category_id` |
| GET           | `/api/items`          | `?search=<name>&category_id=<id>&page=1&limit=20` |
| GET           | `/api/items/{id}`     | |
| POST          | `/api/items`          | owner, manager |
| PUT / PATCH   | `/api/items/{id}`     | partial update - only sent fields change |
| DELETE        | `/api/items/{id}`     | owner |
| GET           | `/api/customers`      | `?search=<name/email/phone>&page=1&limit=20` |
| GET           | `/api/customers/{id}` | |
| POST          | `/api/customers`      | |
| PUT / PATCH   | `/api/customers/{id}` | partial update |
| DELETE        | `/api/customers/{id}` | owner |

Send the token as `Authorization: Bearer <token>`.

**Response shape**

```jsonc
// success
{ "data": { ... } }
{ "data": [ ... ], "meta": { "page": 1, "limit": 20, "total": 6, "total_pages": 1 } }
// error
{ "error": { "message": "validation failed", "details": { "price": "must be greater than or equal to 0" } } }
```

Status codes: `200`, `201`, `204` (delete), `400` (bad id/JSON), `401`, `403`, `404`, `409` (duplicate customer email), `422` (validation), `429` (login rate limit).

**Example**

```bash
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"owner@icecream.test","password":"Owner@123"}' | jq -r .data.token)

curl -H "Authorization: Bearer $TOKEN" "localhost:8080/api/items?search=choc"

curl -X POST localhost:8080/api/items -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Mint Chip","category_id":1,"price":3.5,"quantity":20}'
```

## Database design / ER diagram

Source: [`docs/er-diagram.mmd`](docs/er-diagram.mmd) (Mermaid - also rendered below on GitHub). Export to PNG/PDF at <https://mermaid.live> for the submission.

```mermaid
erDiagram
    USERS {
        uint id PK
        string name
        string email UK
        string password_hash
        string role "owner | manager | cashier"
    }
    CATEGORIES {
        uint id PK
        string name UK
    }
    ITEMS {
        uint id PK
        string name
        string description
        uint category_id FK
        float price
        int quantity
    }
    CUSTOMERS {
        uint id PK
        string name
        string email UK
        string phone
        string address
    }
    REVOKED_TOKENS {
        string id PK "JWT jti"
        datetime expires_at
    }
    CATEGORIES ||--o{ ITEMS : "has many"
```

Design notes:
- A **category** is an ice cream flavour; each **item** belongs to exactly one category (`ON DELETE RESTRICT`).
- `users.role` is constrained to the three roles in code; users are created by the seeder (no registration endpoint, per the brief).
- `revoked_tokens` makes logout real for stateless JWTs; expired rows are purged on each logout.
- Role is read from the DB on every request, so role changes/deleted users take effect immediately.

## Project structure

```
cmd/server/main.go          entrypoint, graceful shutdown
internal/config             env config
internal/database           connect, migrate, seed
internal/models             GORM models
internal/auth               JWT generation/parsing
internal/middleware         Authenticate + RequireRoles
internal/handlers           auth, items, customers, categories
internal/response           response envelope + global error handler
internal/validation         validator wrapper (JSON field names)
internal/server             route table + RBAC wiring (also used by tests)
docs/er-diagram.mmd         ER diagram source
postman/                    Postman collection
```

## Postman

Import `postman/IceCreamShop.postman_collection.json`, run one of the **Login** requests (it saves the token to a collection variable), then call any endpoint.

## Configuration

See `.env.example`: `PORT`, `DATABASE_DSN`, `JWT_SECRET` (min 16 chars), `JWT_TTL_HOURS`, `SEED_ON_START`.
