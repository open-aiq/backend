# AIQ Backend API

Air quality monitoring API built with Go, Gin, and Ent (PostgreSQL).

> **Documentation:** See the [unified Open AIQ documentation](https://open-aiq.github.io/docs/) for system architecture, device guides, shared contracts, and the generated API reference. This README remains the source of truth for backend setup and commands.

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

Create a PostgreSQL custom-format backup with `make db-backup`. Backups use
readable UTC filenames such as
`backups/openaiq-backup-2026-09-07_04-30-00_UTC.dump`. To remove all Open AIQ
backups, run `make db-backups-clean`; it lists the files and asks for confirmation
before deleting anything.

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

API failures use [RFC 9457 Problem Details](https://www.rfc-editor.org/rfc/rfc9457)
with the `application/problem+json` media type. The canonical error catalog is
generated from `internal/platform/problem` by `make swagger` and published at
[docs.air-iq.net/reference/errors](https://docs.air-iq.net/reference/errors/).

## Public map and location privacy

`GET /api/v1/public/map/devices` returns one latest complete reading per map-visible
device for clustering and marker rendering. A device is included only when its
owner enables both `is_public` and `is_location_public`. Location sharing exposes
the exact coordinates from the latest telemetry sample; making a device private
automatically revokes that consent. Existing and newly migrated devices default
to private location.

For a map-ready local dataset, choose an existing device ID from the dashboard
and run `make seed device=dev_<id>`. This inserts chart history plus a current
Karachi location and enables both public visibility flags. Use
`make seed device=dev_<id> map=false` when readings should be added without
changing that device's visibility.
