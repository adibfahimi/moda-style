# Architecture

This document describes how Moda Style is put together, the reasoning behind
the main design decisions, and where the seams are if you need to change them.

## Goals and non-goals

**Goals**

- Keep every service small enough that a new contributor can read it top‑to‑bottom.
- Reuse cross‑cutting concerns (auth, DB access, validation) through one module.
- Ship a realistic, end‑to‑end e‑commerce flow (browse → cart → checkout → order)
  without pulling in heavyweight infrastructure.

**Non-goals**

- Independent per‑service databases. Moda Style deliberately shares one
  PostgreSQL database (see [Shared database](#shared-database)).
- Event streaming / message queues. Services communicate through the database.
- Real payment processing. Payments are simulated behind a clear seam so a real
  provider (e.g. Stripe) can be dropped in later.

## Runtime topology

| Component         | Image / runtime         | Port | Notes                                      |
| ----------------- | ----------------------- | ---- | ------------------------------------------ |
| `postgres`        | `postgres:16-alpine`    | 5432 | Single source of truth for all data        |
| `auth-service`    | static Go binary        | 8001 | Stateless; the only JWT issuer             |
| `product-service` | static Go binary        | 8002 | Public read API                            |
| `cart-service`    | static Go binary        | 8003 | Authenticated                              |
| `admin-service`   | static Go binary        | 8004 | Serves `/uploads` for product images       |
| `order-service`   | static Go binary        | 8005 | Authenticated                              |
| `frontend`        | `nginx:alpine` + assets | 80   | SPA, history‑mode fallback to `index.html` |
| `nginx`           | `nginx:alpine`          | 80   | Public entrypoint, path‑based routing      |

Services are built from a shared Go **workspace** (`go.work`). Each service is a
separate Go module that depends on `github.com/adibfahimi/moda-style/common`.
The Dockerfiles build a static (`CGO_ENABLED=0`) binary and ship it in a
`scratch` image together with the CA bundle.

## Shared database

All services open a connection to the same PostgreSQL database using
`common.ConnectDatabase`. Each service runs `AutoMigrate` for the models it
owns at startup:

| Service           | Owned tables                                      |
| ----------------- | ------------------------------------------------- |
| `auth-service`    | `users`                                           |
| `product-service` | `categories`, `products`, `sizes`, `reviews`      |
| `cart-service`    | `cart_items`, `wishlist_items`                    |
| `order-service`   | `orders`, `order_items`, `payment_transactions`   |
| `admin-service`   | `activity_logs`, `orders`, `order_items`          |

**Why one database?** It removes a whole class of distributed‑transaction and
data‑consistency problems for a catalogue/cart/order domain that is naturally
tightly coupled, and it lets one service join another's tables (for example the
cart service enriching cart rows with product names and prices) with no network
hop.

**Consequences to be aware of**

- Handler code in one service sometimes queries another service's tables
  directly (e.g. `cart-service` reads `products` and `sizes`). This is a
  deliberate, documented coupling.
- Schema ownership is social, not enforced. When you change a model, grep for
  raw queries against its table across `services/`.
- The admin service and order service both migrate `orders`/`order_items`. Keep
  their model definitions in sync.

The `common` module is the right home for anything that must stay consistent
across services; the duplicated table definitions are a known trade‑off tracked
as future work (see [Roadmap](#roadmap)).

## Authentication and authorisation

1. A client registers or logs in against `auth-service`.
2. `auth-service` signs a **JWT** (HS256) whose claims include `user_id`,
   `email`, `user_name` and `role`, expiring after 24 hours.
3. The client sends `Authorization: Bearer <token>` on subsequent requests.
4. Every service protects routes with `common.RequireAuth`, which verifies the
   signature with the shared `JWT_SECRET` and copies the claims into Fiber
   request `Locals`.
5. Admin routes add `common.RequireAdmin`, which rejects callers whose `role`
   claim is not `admin`.

Because verification is stateless, the services never call each other to
authenticate a request — they only need the shared secret.

## Request lifecycle (example: place an order)

1. The SPA calls `POST /api/v1/orders` through Nginx.
2. Nginx routes `/api/v1/orders*` to `order-service:8005`.
3. `RequireAuth` validates the JWT and injects the user id.
4. `order-service` reads the user's `cart_items`, joins `products` and `sizes`,
   validates stock, builds `order_items`, and persists an `orders` row with
   status `pending`.
5. The SPA calls `POST /api/v1/orders/:id/pay`. `order-service` simulates the
   payment, records a `payment_transactions` row, marks the order `paid` /
   `processing`, decrements size stock, and clears the cart.
6. `GET /api/v1/orders/my-orders` returns the user's order history.

## Frontend

The SPA is SolidJS + TypeScript, bundled by Vite. Routing is handled by
`@solidjs/router`; the routes are declared centrally in `frontend/src/index.tsx`.

- `frontend/src/context/AuthContext.tsx` owns the logged‑in user and token.
- `frontend/src/context/CartContext.tsx` owns cart state and reacts to auth changes.
- `frontend/src/services/*` is the single place where HTTP calls live; UI
  components never call `fetch` directly.
- `frontend/src/config/api.ts` resolves each service URL: explicit
  `VITE_*_SERVICE_URL` override → `localhost:800x` in dev → same origin in
  production (so Nginx can proxy).

## Error handling conventions

- Handlers return JSON shaped as `{ "error": "message" }` on failure via
  `common.SendErrorResponse`, and success payloads as small `fiber.Map`s.
- Request bodies are parsed and validated with `common.ParseAndValidate`, which
  reads `validate:"..."` struct tags and returns human‑readable messages.
- The admin service installs a custom Fiber `ErrorHandler` so unexpected errors
  still return the standard error envelope.

## Testing strategy

- **Unit tests** cover pure logic (token generation, validation formatting,
  password hashing, stock calculation, slug/number generation).
- **Handler tests** boot an in‑memory pure‑Go SQLite database, assign it to the
  service's `database.DB` global, and drive handlers through `app.Test(...)`.
  This exercises routing, middleware, validation and SQL without any external
  services.
- **Frontend tests** (Vitest) cover pure client logic and services with a
  mocked `fetch`.

See the `Makefile` for the exact commands.

## Roadmap

Ideas that are intentionally left out of the current scope:

- Move the duplicated `orders`/`order_items` models into `common` to remove drift risk.
- Introduce optimistic concurrency / row locking around stock decrements.
- Replace simulated payments with a real provider behind the existing
  `paymentService` seam.
- Add structured logging and request IDs across services.
- Consider per‑service databases with an API (rather than direct table joins) if
  the domain boundaries harden.

