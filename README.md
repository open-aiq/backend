# AIQ Backend API

Air quality monitoring API built with Go, Gin, and Ent (PostgreSQL).

## Prerequisites

- Go 1.26+
- Docker (for PostgreSQL)
- [`air`](https://github.com/air-verse/air) and [`swag`](https://github.com/swaggo/swag) for live reload and Swagger generation (only needed for `make dev` / `make swagger`)

## Configuration

Configuration is read from environment variables, loaded from a `.env` file at the
project root if present. Real environment variables take precedence over `.env`,
and a missing `.env` is fine (values fall back to sensible defaults).

Get started by copying the template:

```
cp .env.example .env
```

See [`.env.example`](.env.example) for the full list of variables, their defaults,
and descriptions — it is the single source of truth for configuration.

> The `.env` file is git-ignored. Never commit real credentials — keep `.env.example` as the shared template.

## Database

Start a PostgreSQL container. Its user, password, database name, and port are
derived from `DATABASE_URL` in your `.env`, so there's nothing to configure twice:

```
make db-up
```

Other database helpers: `make db-down` (remove container, keep data), `make db-logs`,
and `make db-shell` (open a `psql` session). Run `make help` for the full list.

The schema is managed with [Ent](https://entgo.io). Tables are auto-migrated on
startup, so no manual migration step is needed in development. After editing a
schema in `internal/platform/ent/schema/`, regenerate the client:

```
go generate ./internal/platform/ent
```

## Running

This project uses a [`Makefile`](Makefile) as a command runner. Rather than
memorising long commands, you run short `make <target>` shortcuts. You don't need
to know `make` itself — list every available target, with descriptions, at any time:

```
make help
```

The usual one to start the server:

```
make run        # generate Swagger docs, build, and run
```

The [`Makefile`](Makefile) is the single source of truth for the available targets.

Prefer to skip the build tooling? Run directly with Go:

```
go run ./cmd/server
```

## API Documentation

Swagger UI: http://localhost:8080/swagger/index.html
