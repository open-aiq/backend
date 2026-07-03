# Load .env so DATABASE_URL is available to the db-* targets.
# DATABASE_URL has no fallback — it must be defined in .env (see .env.example).
-include .env

# Release configuration.
RELEASE_BRANCH ?= main
DIST           ?= dist
BINARY         ?= server
PLATFORMS      ?= linux/amd64 linux/arm64 darwin/arm64

.PHONY: help dev generate build run swagger clean release migration migrate-up require-database-url db-up db-down db-logs db-shell

## help: Show available commands
help:
	@echo "Available commands:"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

## dev: Live reload with swagger regeneration
dev:
	air

## swagger: Generate swagger docs
swagger:
	swag init -g cmd/server/main.go -o docs

## generate: Regenerate the Ent client after editing a schema (internal/platform/ent/schema/)
generate:
	go generate ./internal/platform/ent

## migration: Generate a versioned migration from schema changes — usage: make migration name=<description> (needs Docker)
migration:
	@test -n "$(name)" || { echo "usage: make migration name=<description>"; exit 1; }
	atlas migrate diff "$(name)" --env defaultConfig

## migrate-up: Apply all pending migrations to DATABASE_URL
migrate-up: require-database-url
	atlas migrate apply --env defaultConfig --url "$(DATABASE_URL)"

## build: Regenerate code, generate swagger, and build the binary
build: generate swagger
	go build -o bin/server ./cmd/server/

## run: Build and run the server
run: build
	./bin/server

## clean: Remove build artifacts
clean:
	rm -rf bin/ tmp/ $(DIST)/

## release: Bump version (prompts major/minor/patch), build artifacts, tag, and publish a GitHub release
release:
	@RELEASE_BRANCH="$(RELEASE_BRANCH)" DIST="$(DIST)" BINARY="$(BINARY)" PLATFORMS="$(PLATFORMS)" \
		bash scripts/release.sh

# Fail fast if DATABASE_URL isn't defined (no fallback; see .env.example).
require-database-url:
	@if [ -z "$(DATABASE_URL)" ]; then \
	  echo "DATABASE_URL is not set. Define it in .env (see .env.example)."; \
	  exit 1; \
	fi

## db-up: Start the PostgreSQL container (creds derived from DATABASE_URL)
db-up: require-database-url
	@set -e; \
	url="$(DATABASE_URL)"; \
	creds="$${url#*://}"; \
	user="$${creds%%:*}"; \
	rest="$${creds#*:}"; \
	pass="$${rest%%@*}"; \
	hostpart="$${rest#*@}"; \
	hostport="$${hostpart%%/*}"; \
	port="$${hostport##*:}"; \
	[ "$$port" = "$${hostport%%:*}" ] && port=5432; \
	dbq="$${hostpart#*/}"; \
	db="$${dbq%%\?*}"; \
	echo "Starting openaiq-postgres (db=$$db, port=$$port)..."; \
	docker run -d \
	  --name openaiq-postgres \
	  --restart unless-stopped \
	  -e POSTGRES_USER="$$user" \
	  -e POSTGRES_PASSWORD="$$pass" \
	  -e POSTGRES_DB="$$db" \
	  -e PGDATA=/var/lib/postgresql/data/pgdata \
	  -p 127.0.0.1:$$port:5432 \
	  -v openaiq-pgdata:/var/lib/postgresql/data \
	  --shm-size=256m \
	  --health-cmd="pg_isready -U $$user -d $$db" \
	  --health-interval=10s --health-timeout=5s --health-retries=5 \
	  postgres:16

## db-down: Stop and remove the PostgreSQL container (data volume is kept)
db-down:
	docker rm -f openaiq-postgres

## db-logs: Tail the PostgreSQL container logs
db-logs:
	docker logs -f openaiq-postgres

## db-shell: Open a psql shell in the PostgreSQL container
db-shell: require-database-url
	@docker exec -it openaiq-postgres psql -U "$$(url='$(DATABASE_URL)'; creds="$${url#*://}"; echo "$${creds%%:*}")" -d "$$(url='$(DATABASE_URL)'; rest="$${url##*/}"; echo "$${rest%%\?*}")"
