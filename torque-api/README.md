# torque-api

Go backend for the Torque CRM. Sprint **S01** skeleton — health probes, full
migration set (3 migrations covering organizations, users, team_members, leads,
pipes, tags, audit log, and tasks + `users.ui_mode_preference` per ADR-007),
Docker Compose, Makefile.

Subsequent sprints layer in auth (S02), security headers + observability
(S03), OpenAPI + WebSocket hub + async jobs (S04).

## Stack

- **Go 1.22+** · chi router, zerolog, pgx v5, google/uuid
- **PostgreSQL 15+** with gen_random_uuid + citext
- **Migrations** via `golang-migrate`
- **Runtime image** distroless `nonroot`

## Quickstart

```bash
cp .env.example .env
docker compose up            # Postgres + migrate + API in one shot
curl -s localhost:8080/healthz
curl -s localhost:8080/readyz
```

Or run on the host against Compose Postgres:

```bash
make docker-up-db            # just the database
make migrate-up              # apply migrations (needs `migrate` CLI on PATH)
make run                     # starts the API on :8080
```

## Layout

```
torque-api/
├── api/
│   └── openapi.yaml                  OpenAPI 3.1 — skeleton for S01 (probes only)
├── cmd/api/
│   └── main.go                       Bootstrap, graceful shutdown, router wiring
├── internal/
│   ├── config/                       Typed env config (fail-fast on required vars)
│   ├── db/                           pgx pool with sane defaults
│   ├── domain/                       (reserved — S02+)
│   ├── handler/
│   │   └── health/                   /healthz + /readyz
│   ├── httpx/middleware/             request_id, access log, recover, security headers,
│   │                                 strip_organization_id, CORS
│   └── repository/
│       └── task_concurrency_test.go  Integration test: proves the unique partial index
│                                     `uq_tasks_one_in_progress_per_assignee`.
├── migrations/
│   ├── 0001_organizations_users.{up,down}.sql
│   ├── 0002_team_members_leads_pipes.{up,down}.sql
│   └── 0003_tasks_and_ui_preferences.{up,down}.sql          (ADR-007)
├── Dockerfile                        Multi-stage build → distroless nonroot
├── docker-compose.yml                Postgres + migrate + API
├── Makefile                          Opinionated dev commands (`make help`)
└── README.md                         You are here.
```

## Config (env)

| Variable            | Required | Default         | Notes                                |
|---------------------|----------|-----------------|--------------------------------------|
| `DATABASE_URL`      | ✅       | —               | `postgres://user:pass@host:5432/db`  |
| `ENV`               |          | `dev`           | `dev` \| `staging` \| `prod`         |
| `HTTP_ADDR`         |          | `:8080`         | Listen address                       |
| `LOG_LEVEL`         |          | `info`          | `debug` \| `info` \| `warn` \| `error` |
| `SHUTDOWN_TIMEOUT`  |          | `15s`           | Graceful-drain ceiling               |
| `READ_TIMEOUT`      |          | `10s`           | HTTP read timeout                    |
| `WRITE_TIMEOUT`     |          | `15s`           | HTTP write timeout                   |
| `IDLE_TIMEOUT`      |          | `60s`           | HTTP idle timeout                    |
| `CORS_ORIGINS`      |          | (empty, closed) | Comma-separated allow-list           |

## Probes

- `GET /healthz` — liveness. Returns 200 while the process is up. Used by
  orchestrators to decide **restart**.
- `GET /readyz` — readiness. Pings the DB; returns 503 when unreachable. Used
  by orchestrators to decide **route traffic**.

## Migrations

`golang-migrate` is the tool. Each migration is a pair: `NNNN_name.up.sql` +
`NNNN_name.down.sql`. Migration `0003_tasks_and_ui_preferences` implements
ADR-007 (Task absorbs Follow-up, `users.ui_mode_preference`, partial unique
index for single in_progress per assignee).

```bash
make migrate-up       # apply all pending
make migrate-down     # roll back the last one
make migrate-create name=add_foo
```

## Multi-tenancy invariant

`organization_id` is **never** accepted in a request body. The
`StripOrganizationID` middleware rejects any POST/PATCH/PUT whose JSON body
contains that key with `400 TENANT_FIELD_FORBIDDEN`. The tenant is derived
exclusively from the JWT in later sprints; here it is the architectural floor.

## Tests

```bash
make test                 # unit (fast, no DB)
make test-integration     # requires DATABASE_URL; runs the concurrency test
```

## Dev seeds

A committed seed creates a deterministic admin so the frontend can log in
immediately after migrations:

```bash
make seed-dev             # idempotent; refuses when ENV != dev
```

Credentials:

| email                 | password | role  | organization |
|-----------------------|----------|-------|--------------|
| `marcelo@gmail.com`   | `admin`  | admin | Torque Dev   |

Source: `seeds/dev_admin.sql`. The bcrypt hash in that file is regenerable at
any time and is committed deliberately because it is a development credential,
not a production secret.

The concurrency test races 32 goroutines transitioning tasks to `in_progress`
and asserts that exactly one succeeds — the database enforces it via
`uq_tasks_one_in_progress_per_assignee`.

## Decisions referenced

- ADR-001 — OpenAPI snake_case wire / camelCase frontend
- ADR-002 — WebSocket hub per tenant (S04)
- ADR-003 — Auth via httpOnly cookies (S02)
- ADR-007 — UI modes + Task unification (this sprint: migration `0003`)
