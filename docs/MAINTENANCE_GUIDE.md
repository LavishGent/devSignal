# DevSignal — Maintenance & Technical Guide

Technical documentation for developers who build, run, test, extend, and deploy
DevSignal. For end-user instructions, see [USER_GUIDE.md](USER_GUIDE.md); for
the domain model, see [UML.md](UML.md).

## 1. Architecture overview

DevSignal is a two-tier full-stack application:

| Layer    | Technology                         | Location    |
|----------|------------------------------------|-------------|
| Frontend | React 19 + TypeScript (Vite)       | `frontend/` |
| Backend  | Go (standard `net/http`, `pgx`)    | `backend/`  |
| Database | PostgreSQL 16                      | migrations in `backend/migrations/` |

The backend exposes a JSON REST API under `/api/v1`. In production, the same Go
binary also serves the compiled React app as static files (see `Dockerfile`),
so the whole product ships as one small container.

### Backend package layout (`backend/internal/`)
| Package | Responsibility |
|---------|----------------|
| `domain` | Entities and their validation rules (OOP core) |
| `validation` | Reusable field-level validators |
| `repo` | Persistence **interfaces**; `postgres/` and `memory/` implementations |
| `auth` | Password hashing (bcrypt) and session management |
| `httpapi` | HTTP server, middleware, and per-module handlers |
| `snapshot` | Loads a user's full data set for reporting/recommendations |
| `reports` | Builds titled, tabular, CSV-exportable reports |
| `recommend` | Rule-based recommendation engine |
| `search` | Cross-module search service |
| `dashboard` | Dashboard summary computation |
| `config` | Environment-based configuration |
| `database` | Connection pool and migration runner |

## 2. Prerequisites

- Go 1.25+
- Node.js 20+ (Docker image uses Node 24)
- Docker + Docker Compose (for PostgreSQL and container builds)

## 3. Local development

The `Makefile` wraps the common commands.

```bash
make db-up        # start PostgreSQL 16 on localhost:5432 (docker compose)
make api          # run the Go API on :8080 (runs DB migrations at startup)
make web          # run the Vite dev server on :5173 (proxies /api to :8080)
```

Then open `http://localhost:5173`. Configuration comes from environment
variables with working development defaults; copy `.env.example` to `.env` to
override. Key variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | API listen port |
| `DATABASE_URL` | local compose DSN | PostgreSQL connection string |
| `APP_ENV` | `development` | `production` enables Secure cookies + HSTS |
| `SESSION_TTL_HOURS` | `72` | Login session lifetime |
| `BCRYPT_COST` | `12` | Password hash work factor (10–16) |
| `STATIC_DIR` | *(empty)* | Directory of built frontend to serve |

Secrets (database credentials, production `DATABASE_URL`) are never committed;
set them via the environment or `fly secrets set` in production.

## 4. Database migrations

SQL migrations live in `backend/migrations/` (`00001_users.sql` …
`00010_goals.sql`) and are embedded into the binary (`embed.go`). They run
automatically at API startup, in order. **To change the schema, add a new
numbered migration file** — never edit an already-applied migration. Keep the
filename prefix sequential.

## 5. How to update the code

### Add a new field to an existing entity
1. Add the field to the struct in `backend/internal/domain/<entity>.go` and add
   any validation in its `Validate()` method.
2. Add a new migration in `backend/migrations/` that alters the table.
3. Update the entity's PostgreSQL repository in
   `backend/internal/repo/postgres/<entity>.go` (INSERT, SELECT, UPDATE column
   lists) and the matching in-memory repository in `repo/memory/`.
4. Surface the field in the frontend form/table under `frontend/src/pages/`.
5. Add or update tests (see Section 6), then run `make test`.

### Add a whole new module (record type)
1. Create `backend/internal/domain/<entity>.go` with a struct that embeds
   `BaseRecord` and implements `Validate()`. Register it in the compile-time
   `Validator` assertion in `domain/base.go`.
2. Add a migration for its table.
3. Define its repository interface in `backend/internal/repo/repo.go` and
   implement it in both `repo/postgres/` and `repo/memory/`.
4. Add a handler in `backend/internal/httpapi/<entity>_handlers.go` that embeds
   `BaseHandler` and implements `Register(mux, protected)`.
5. Wire it in `backend/cmd/api/modules.go` (the single place modules are
   registered) — add it to the snapshot `Loader` and `search.Service` too if it
   should appear in reports/search.
6. Add a page under `frontend/src/pages/` and a route in `frontend/src/App.tsx`.
7. Add tests and run `make test`.

### Add a recommendation rule
Create a struct embedding `BaseRule`, implement `Evaluate()`, and append it to
the slice in `recommend.NewEngine()`. The engine needs no other change. Keep
numeric thresholds as unexported package constants and cover them with a test.

### Add a report
Add a `Build…` function in `backend/internal/reports/builders.go`, register it
in `Catalog()` and `ByID()`, and add a test asserting its columns and rows.

## 6. Testing

```bash
make test         # go vet + unit/handler tests (with coverage) + frontend build
make integration  # full-stack tests against real PostgreSQL (needs `make db-up`)
make lint         # golangci-lint + oxlint
```

- Unit and handler tests use the in-memory repositories, so they are fast and
  need no database.
- The integration test (`internal/integration/`, build tag `integration`) runs
  the real server against PostgreSQL via `TEST_DATABASE_URL`.
- See [TEST_PLAN.md](TEST_PLAN.md) and [TEST_RESULTS.md](TEST_RESULTS.md).

## 7. Security model (summary)

- Passwords hashed with **bcrypt**; the hash is unexported and never serialized.
- Sessions are 256-bit random tokens delivered in an **HttpOnly, SameSite=Strict**
  cookie; only the SHA-256 hash of the token is stored.
- `Secure` cookie flag and **HSTS** are enabled when `APP_ENV=production`.
- All database access uses **parameterized queries** (no string-built SQL).
- Every query is **user-scoped**, and handlers re-check ownership.
- **Rate limiting** protects the register/login endpoints (10 req/min/IP).
- **Security headers** (CSP, `X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy`) are set on every response.
- CSV exports neutralize **spreadsheet formula injection**.

## 8. Build & deploy

```bash
make build        # production build of both halves
make docker       # build the production container image
```

- **Container:** the multi-stage `Dockerfile` builds the SPA and the Go binary,
  then ships Alpine running as a non-root user. `docker compose up --build`
  runs the whole stack at `http://localhost:8080`.
- **Cloud (Fly.io):** `fly.toml` defines the app, a health check on
  `/api/v1/health`, and forced HTTPS. Set the production database with
  `fly secrets set DATABASE_URL=…`, then `fly deploy`. The GitLab CI pipeline
  (`.gitlab-ci.yml`) can deploy from `main` on a manual trigger.

## 9. Continuous integration

`.gitlab-ci.yml` runs on every push:
1. `backend-tests` — `go vet`, unit tests with coverage, and the integration
   suite against a PostgreSQL service.
2. `backend-lint` — `golangci-lint`.
3. `frontend-build` — `npm ci`, `npm run build`, `oxlint`.
4. `docker-image` — builds the production image on `main`.
5. `deploy-fly` — manual deploy to Fly.io from `main`.
