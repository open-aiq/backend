# AIQ Backend API

Air quality monitoring API built with Go, Gin, and Ent (PostgreSQL).

## Prerequisites

- Go 1.26+
- Docker (for PostgreSQL)
- [`air`](https://github.com/air-verse/air) and [`swag`](https://github.com/swaggo/swag) for live reload and Swagger generation (only needed for `make dev` / `make swagger`)

## Commands

This project uses a [`Makefile`](Makefile) as its command runner and single source
of truth for all operations.

```
make help
```

## Configuration

Configuration is read from environment variables, loaded from a `.env` file at the
project root if present. Real environment variables take precedence over `.env`.
Every variable is **required** — there are no defaults — so the app fails fast on
startup (listing what's missing) if any is unset.

Get started by copying the template:

```
cp .env.example .env
```

See [`.env.example`](.env.example) for the full list of variables, their defaults,
and descriptions — it is the single source of truth for configuration.

> The `.env` file is git-ignored. Never commit real credentials — keep `.env.example` as the shared template.

## Database

PostgreSQL runs in a Docker container whose user, password, database, and port are
derived from `DATABASE_URL` in your `.env`, so there's nothing to configure twice.
Start it and manage it with the `db-*` targets in `make help`.

The schema is managed with [Ent](https://entgo.io) and auto-migrated on startup, so
no manual migration step is needed in development. After editing a schema in
`internal/platform/ent/schema/`, regenerate the Ent client with `make generate`
(also run automatically by `make build`).

### Existing device ownership backfill

The initial Clerk ownership migration leaves `devices.owner_id` nullable so no
legacy device is assigned implicitly. Backfill each device with an explicit Clerk
user ID, for example:

```sql
UPDATE devices SET owner_id = 'user_...' WHERE device_id = 'dev_...';
SELECT device_id, name FROM devices WHERE owner_id IS NULL;
```

Only after the second query returns no rows should a separate migration make
`owner_id` non-null. Unowned devices are hidden from private APIs; public legacy
devices remain available through `/api/v1/public/devices`.

## Architecture

The backend follows a hybrid domain-driven + hexagonal design.

- **Domains** live in `internal/<domain>/` (e.g. `internal/airquality/`,
  `internal/device/`) and are self-contained. Each is split into:
  - `model.go` — domain types and request/response DTOs
  - `repository.go` — the `Repository` interface (hexagonal port) and its implementation
  - `service.go` — business logic
  - `handler.go` — Gin HTTP handlers with Swagger annotations
  - `routes.go` — route registration
- **Shared infrastructure** lives in `internal/platform/` — config, database,
  and the Ent client/schema.
- **Request flow:** `routes → handler → service → repository (interface) → DB`.
  Handlers never touch the DB directly; services depend on the repository
  *interface*, so implementations (Ent-backed or in-memory mock) are swappable.
  (Note: `airquality` currently uses a mock repository; `device` is Ent-backed.)
- **Entrypoint:** `cmd/server/main.go` wires each domain's repository → service
  → handler and registers its routes.

## Releases

Releases follow [SemVer](https://semver.org) with a `v` prefix (e.g. `v0.1.0`) and
are cut **only from `main`**. `dev` is for integration; promote `dev` → `main`, then
release from `main`.

```
make release
```

This prompts for the bump type (**major/minor/patch**), computes the next version
from the latest tag, and — after you confirm — builds version-stamped binaries for
`linux/amd64`, `linux/arm64`, and `darwin/arm64`, assembles a deploy bundle plus the
OpenAPI spec and `SHA256SUMS`, tags the commit, and publishes a GitHub release with
auto-generated notes and the artifacts attached.

It runs locally and requires an authenticated [`gh`](https://cli.github.com) CLI.
The logic lives in [`scripts/release.sh`](scripts/release.sh).

## API Documentation

Swagger UI: http://localhost:8080/swagger/index.html
