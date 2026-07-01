# Project Rules

## General
- Do NOT run commands (build, test, make, etc.) unless explicitly asked
- Do NOT commit or push unless explicitly asked
- Do NOT create files unless absolutely necessary
- Ask before making destructive changes

## Project Structure
- Entrypoint: `cmd/server/main.go`
- Domain code lives in `internal/<domain>/` (e.g., `internal/airquality/`)
- Each domain has: `model.go`, `repository.go`, `service.go`, `handler.go`, `routes.go`
- Shared infrastructure lives in `internal/platform/` (config, database, middleware)
- Build output goes to `bin/`
- Swagger docs are in `docs/` and auto-generated via `swag init`

## Architecture
- Hybrid domain-driven + hexagonal approach
- Each domain defines a Repository interface (from hexagonal)
- Flow: routes → handler → service → repository (interface) → DB
- Keep domains self-contained

## Build & Run
- `make dev` — live reload with air
- `make build` — swagger + build
- `make run` — swagger + build + run
- `make swagger` — regenerate swagger docs only
- `make clean` — remove build artifacts
- `go run ./cmd/server/` — quick run without swagger

## Code Style
- Go 1.22+ features allowed (e.g., range over int)
- Keep it simple, avoid over-engineering
- No premature abstractions
