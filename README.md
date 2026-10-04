# Moda Style

[![CI](https://github.com/adibfahimi/moda-style/actions/workflows/ci.yml/badge.svg)](https://github.com/adibfahimi/moda-style/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Moda Style is a clothing e‑commerce platform built as a set of small, independent
Go microservices behind an Nginx gateway, with a SolidJS single‑page application
as the storefront and admin console.

The project intentionally favours **explicit, boring, standard building blocks**
(Fiber, GORM, PostgreSQL, JWT) so that each service stays small enough to reason
about in isolation.

---

## Table of contents

- [Architecture](#architecture)
- [Repository layout](#repository-layout)
- [Tech stack](#tech-stack)
- [Getting started](#getting-started)
- [Service catalogue](#service-catalogue)
- [API reference](#api-reference)
- [Environment variables](#environment-variables)
- [Testing](#testing)
- [Documentation](#documentation)
- [License](#license)

---

## Architecture

```
                          ┌──────────────────────────┐
                          │        Browser           │
                          │  SolidJS SPA (Vite)      │
                          └────────────┬─────────────┘
                                       │ HTTP
                          ┌────────────▼─────────────┐
                          │   Nginx  (gateway, :80)  │
                          │  routes /api/v1/* to the │
                          │  matching microservice   │
                          └────────────┬─────────────┘
        ┌───────────────┬──────────────┼───────────────┬───────────────┐
        │               │              │               │               │
┌───────▼──────┐ ┌──────▼───────┐ ┌────▼──────┐ ┌──────▼─────┐ ┌───────▼──────┐
│ auth-service │ │product-svc   │ │cart-svc   │ │order-svc   │ │admin-service │
│    :8001     │ │    :8002     │ │   :8003   │ │   :8005    │ │    :8004     │
└───────┬──────┘ └──────┬───────┘ └────┬──────┘ └──────┬─────┘ └───────┬──────┘
        │               │              │               │               │
        └───────────────┴──────────────┴───────────────┴───────────────┘
                                       │ GORM
                          ┌────────────▼─────────────┐
                          │  PostgreSQL 16 (shared)  │
                          │   database: moda_style   │
                          └──────────────────────────┘
```

Each service owns its handlers/models but they **share a single PostgreSQL
database**. Cross‑service reads (for example, the cart service reading product
rows) therefore happen through shared tables rather than network calls. This
keeps the deployment simple at the cost of tighter coupling — see
[`docs/architecture.md`](docs/architecture.md) for the trade‑offs and a roadmap.

Auth is stateless: a JWT issued by `auth-service` is validated by every service
via the shared `common.RequireAuth` middleware.

---

## Repository layout

```
.
├── common/                 # Shared Go module: JWT, DB connection, validation helpers
├── services/
│   ├── auth-service/       # Registration, login, profile, password reset
│   ├── product-service/    # Products, categories, reviews (public read API)
│   ├── cart-service/       # Cart + wishlist (authenticated)
│   ├── order-service/      # Orders + payment simulation (authenticated)
│   └── admin-service/      # Admin console API (admin only)
├── frontend/               # SolidJS + TypeScript + Vite SPA
├── nginx/                  # Gateway / reverse-proxy configuration
├── docs/                   # Architecture and operations documentation
├── docker-compose.yml      # Local/prod-like orchestration
├── go.work                 # Go workspace tying all modules together
└── Makefile                # Developer convenience targets
```

---

## Tech stack

| Layer          | Technology                                                        |
| -------------- | ----------------------------------------------------------------- |
| Backend        | Go 1.24, [Fiber v2](https://gofiber.io/)                          |
| ORM            | [GORM](https://gorm.io/) (`gorm.io/gorm`, postgres driver)        |
| Database       | PostgreSQL 16                                                     |
| Auth           | JWT (`github.com/golang-jwt/jwt/v5`), bcrypt password hashing     |
| Validation     | `github.com/go-playground/validator/v10`                          |
| Frontend       | SolidJS, TypeScript, [Vite 7](https://vitejs.dev/)                |
| Styling        | Tailwind CSS v4 + daisyUI 5                                       |
| Gateway        | Nginx (reverse proxy + SPA fallback)                              |
| Orchestration  | Docker / Docker Compose                                           |
| Tests          | Go `testing`, pure‑Go SQLite (`github.com/glebarez/sqlite`), Vitest |

---

## Getting started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose (recommended), **or**
- Go 1.24+, Node.js 20+, and a local PostgreSQL 16 instance for the manual path.

### Running with Docker Compose

```bash
# 1. Create your environment file from the template
cp .env.example .env
# 2. (Recommended) set a strong JWT secret in .env
#    JWT_SECRET=$(openssl rand -hex 32)
# 3. Build and start every service
docker compose up --build
```

Once the stack is healthy the application is available at:

| URL                         | Description            |
| --------------------------- | ---------------------- |
| http://localhost            | Storefront (via Nginx) |
| http://localhost/health     | Gateway health check   |
| http://localhost/api/v1/... | Versioned API          |

### Running locally for development

Backend services read their configuration from environment variables. The
quickest way to run a single service against a local database:

```bash
export JWT_SECRET=dev-secret
export DB_HOST=localhost DB_PORT=5432 DB_USER=postgres \
       DB_PASSWORD=postgres DB_NAME=moda_style

make test          # run all Go tests
make run-auth      # run the auth service locally (see Makefile for others)
```

Frontend (requires [Bun](https://bun.sh), the package manager used to produce
`frontend/bun.lock`):

```bash
cd frontend
bun install
bun run dev        # Vite dev server with HMR on http://localhost:5173
bun run test       # Vitest suite (single run)
bun run typecheck  # tsc --build
```

In development the frontend talks to the services directly on `localhost:800x`
(see `frontend/src/config/api.ts`).

---

## Service catalogue

| Service           | Port | Responsibility                                                       |
| ----------------- | ---- | -------------------------------------------------------------------- |
| `auth-service`    | 8001 | Register, login, JWT issuance, profile read/update, password resets  |
| `product-service` | 8002 | Product catalogue, categories (hierarchical), product reviews        |
| `cart-service`    | 8003 | Per‑user shopping cart and wishlist                                  |
| `admin-service`   | 8004 | Admin console: dashboard, users, products, categories, orders, audit |
| `order-service`   | 8005 | Order placement, simulated payments, order history, cancellation     |
| `frontend`        | 80   | SolidJS SPA (served by Nginx)                                        |
| `nginx`           | 80   | Public gateway, routes `/api/v1/*` to services                       |
| `postgres`        | 5432 | Shared PostgreSQL database                                           |

Every service exposes a `GET /health` endpoint returning `{"status":"ok"}`.


---

## API reference

All endpoints are namespaced under `/api/v1`. Authenticated endpoints require an
`Authorization: Bearer <token>` header.

### auth-service (8001)

| Method | Path                          | Auth | Description                    |
| ------ | ----------------------------- | ---- | ------------------------------ |
| POST   | `/api/v1/auth/register`       | –    | Create an account, returns JWT |
| POST   | `/api/v1/auth/login`          | –    | Authenticate, returns JWT      |
| GET    | `/api/v1/auth/profile`        | ✔    | Current user profile           |
| PATCH  | `/api/v1/auth/profile`        | ✔    | Update name/email              |
| POST   | `/api/v1/auth/reset-password` | –    | Request a password reset       |

### product-service (8002)

| Method | Path                           | Auth | Description                      |
| ------ | ------------------------------ | ---- | -------------------------------- |
| GET    | `/api/v1/products`             | –    | List/filter products (paginated) |
| GET    | `/api/v1/products/:id`         | –    | Product detail with sizes        |
| GET    | `/api/v1/categories`           | –    | Category tree                    |
| GET    | `/api/v1/products/:id/reviews` | –    | Product reviews (paginated)      |
| POST   | `/api/v1/products/:id/reviews` | ✔    | Submit a review                  |

### cart-service (8003)

| Method | Path                           | Auth | Description              |
| ------ | ------------------------------ | ---- | ------------------------ |
| GET    | `/api/v1/cart`                 | ✔    | Current cart with totals |
| POST   | `/api/v1/cart`                 | ✔    | Add an item              |
| PUT    | `/api/v1/cart/:id`             | ✔    | Update item quantity     |
| DELETE | `/api/v1/cart/:id`             | ✔    | Remove an item           |
| DELETE | `/api/v1/cart`                 | ✔    | Clear the cart           |
| GET    | `/api/v1/wishlist`             | ✔    | List wishlist items      |
| POST   | `/api/v1/wishlist/:product_id` | ✔    | Toggle a product         |

### order-service (8005)

| Method | Path                        | Auth | Description               |
| ------ | --------------------------- | ---- | ------------------------- |
| POST   | `/api/v1/orders`            | ✔    | Create an order from cart |
| POST   | `/api/v1/orders/:id/pay`    | ✔    | Pay for an order          |
| GET    | `/api/v1/orders/my-orders`  | ✔    | Order history             |
| GET    | `/api/v1/orders/:id`        | ✔    | Order detail              |
| POST   | `/api/v1/orders/:id/cancel` | ✔    | Cancel an order           |

### admin-service (8004)

Admin endpoints live under `/api/v1/admin` and require the `admin` role.

| Group         | Endpoints                                                                                            |
| ------------- | ---------------------------------------------------------------------------------------------------- |
| Dashboard     | `GET /dashboard/stats`, `/dashboard/activity`, `/dashboard/logs`                                     |
| Users         | `GET /users`, `/users/analytics`, `/users/:id`, `PUT/DELETE /users/:id`, `POST /users/:id/ban\|unban` |
| Products      | `GET /products`, `/products/images`, `POST /products`, `POST /products/upload-image`, `PUT/DELETE /products/:id` |
| Product sizes | `GET/POST /products/:id/sizes`, `PUT/DELETE /products/:id/sizes/:sizeId`                             |
| Categories    | `GET/POST /categories`, `PUT/DELETE /categories/:id`                                                 |
| Orders        | `GET /orders`, `/orders/analytics`, `/orders/:id`, `PUT/DELETE /orders/:id`                          |


---

## Environment variables

Configuration is supplied through environment variables (see `.env.example`).

| Variable      | Required | Default   | Description                          |
| ------------- | -------- | --------- | ------------------------------------ |
| `JWT_SECRET`  | ✔        | –         | HMAC secret used to sign/verify JWTs |
| `DB_HOST`     | ✔        | –         | PostgreSQL host                      |
| `DB_PORT`     |          | `5432`    | PostgreSQL port                      |
| `DB_USER`     | ✔        | –         | PostgreSQL user                      |
| `DB_PASSWORD` | ✔        | –         | PostgreSQL password                  |
| `DB_NAME`     | ✔        | –         | PostgreSQL database name             |
| `DB_SSL_MODE` |          | `disable` | GORM/postgres `sslmode`              |

Frontend build-time overrides (optional, see `frontend/src/config/api.ts`):

| Variable                   | Description                          |
| -------------------------- | ------------------------------------ |
| `VITE_AUTH_SERVICE_URL`    | Absolute URL for the auth service    |
| `VITE_PRODUCT_SERVICE_URL` | Absolute URL for the product service |
| `VITE_CART_SERVICE_URL`    | Absolute URL for the cart service    |
| `VITE_ADMIN_SERVICE_URL`   | Absolute URL for the admin service   |
| `VITE_ORDER_SERVICE_URL`   | Absolute URL for the order service   |
| `VITE_PAYMENT_SERVICE_URL` | Optional real payment service URL    |

> **Security note:** `JWT_SECRET` must be set for the services to start. In
> production generate a 256‑bit random value (e.g. `openssl rand -hex 32`) and
> keep it out of source control.

---

## Testing

```bash
make test               # all Go tests (common + every service)
make test-frontend      # Vitest suite for the SPA
make typecheck-frontend # tsc --build for the SPA
make test-cover         # Go coverage summary
make check              # everything CI runs
```

### Backend

Go tests run without a running database: the handler tests boot an in‑memory,
pure‑Go SQLite instance (`github.com/glebarez/sqlite`) so they are fast and
hermetic. `JWT_SECRET` is injected by the tests themselves. Every HTTP handler
is exercised end to end through `net/http/httptest`, including validation,
authentication, and error branches.

### Frontend

The SPA is tested with [Vitest](https://vitest.dev) in a
[jsdom](https://github.com/jsdom/jsdom) environment. Suites live next to the
code they cover (`*.test.ts` / `*.test.tsx`) and are configured in
`frontend/vite.config.ts`, with shared helpers in `frontend/src/test`:

| Suite                             | Covers                                              |
| --------------------------------- | --------------------------------------------------- |
| `src/config/api.test.ts`          | Service URL resolution (env override, dev, origin)  |
| `src/services/*.test.ts`          | Every service module: URLs, verbs, bodies, failures |
| `src/components/admin/*.test.tsx` | `StatsCard`, `Pagination`, `Modal` via Solid Testing Library |

Service modules are tested through a `fetch` stand-in (`installFetchMock`), so
assertions cover the request that would go over the wire — method, path, query
string, JSON body, bearer token — and the message surfaced to the UI when the
backend fails. There is no network access and no database involved.

---

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — deeper design notes, data
  model, request lifecycle, and roadmap.

---

## License

Released under the [MIT License](LICENSE).

Copyright © 2026 Adib Fahimi.

You are free to use, copy, modify, merge, publish, distribute, sublicense and
sell this software, including commercially, provided the copyright notice and
this permission notice stay with it. The software is provided "as is", without
warranty of any kind.

