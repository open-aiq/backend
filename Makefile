# Load .env so DATABASE_URL is available to the db-* targets.
# DATABASE_URL is the single source of truth for DB config; this is the fresh-setup fallback.
-include .env
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/openaiq?sslmode=disable

.PHONY: help dev build run swagger clean db-up db-down db-logs db-shell

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

## build: Build the binary (runs swagger first)
build: swagger
	go build -o bin/server ./cmd/server/

## run: Build and run the server
run: build
	./bin/server

## clean: Remove build artifacts
clean:
	rm -rf bin/ tmp/

## db-up: Start the PostgreSQL container (creds derived from DATABASE_URL)
db-up:
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
db-shell:
	@docker exec -it openaiq-postgres psql -U "$$(url='$(DATABASE_URL)'; creds="$${url#*://}"; echo "$${creds%%:*}")" -d "$$(url='$(DATABASE_URL)'; rest="$${url##*/}"; echo "$${rest%%\?*}")"
