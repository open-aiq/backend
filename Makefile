ENGINE := podman
CONTAINER := openaiq-postgres
CONTAINER_VOLUME := openaiq-pgdata
DB_VERION := 18
BACKUP_DIR ?= backups

# Load .env so DATABASE_URL is available to the db-* targets.
# DATABASE_URL has no fallback — it must be defined in .env (see .env.example).
-include .env

# Release configuration.
RELEASE_BRANCH ?= main
DIST           ?= dist
BINARY         ?= server
PLATFORMS      ?= linux/amd64 linux/arm64 darwin/arm64

.PHONY: help dev generate build run swagger clean release migration migrate-up seed require-database-url db-up db-down db-backup db-backups-clean db-clean db-logs db-shell

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


## build: Regenerate code, generate swagger, and build the binary
build: generate swagger
	go build -o bin/server ./cmd/server/

## build-run: Build and run the server
build-run: build
	./bin/server

## build-clean: Remove build artifacts
build-clean:
	rm -rf bin/ tmp/ $(DIST)/

## orm-gen: Regenerate the Ent client after editing a schema (internal/platform/ent/schema/)
orm-gen:
	go generate ./internal/platform/ent

## migration-gen: Generate a versioned migration from schema changes — usage: make migration name=<description> (needs Docker)
migration-gen:
	@test -n "$(name)" || { echo "usage: make migration name=<description>"; exit 1; }
	atlas migrate diff "$(name)" --env defaultConfig

## migration-apply: Apply all pending migrations to DATABASE_URL
migration-apply: require-database-url
	atlas migrate apply --env defaultConfig --url "$(DATABASE_URL)"




# Fail fast if DATABASE_URL isn't defined (no fallback; see .env.example).
require-database-url:
	@if [ -z "$(DATABASE_URL)" ]; then \
	  echo "DATABASE_URL is not set. Define it in .env (see .env.example)."; \
	  exit 1; \
	fi

## seed: Seed mock readings and map-enable a device — usage: make seed device=dev_<id> [map=false]
db-seed: require-database-url
	@test -n "$(device)" || { echo "usage: make seed device=dev_<id>"; exit 1; }
	@DEVICE_ID="$(device)" DATABASE_URL="$(DATABASE_URL)" ENGINE="$(ENGINE)" CONTAINER="$(CONTAINER)" MAP_VISIBLE="$(if $(map),$(map),true)" bash scripts/seed_readings.sh

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
	echo "Starting $(CONTAINER) (db=$$db, port=$$port)..."; \
	$(ENGINE) run -d \
	  --name $(CONTAINER) \
	  --restart unless-stopped \
	  -e POSTGRES_USER="$$user" \
	  -e POSTGRES_PASSWORD="$$pass" \
	  -e POSTGRES_DB="$$db" \
	  -e PGDATA=/var/lib/postgresql/data/pgdata \
	  -p 127.0.0.1:$$port:5432 \
	  -v $(CONTAINER_VOLUME):/var/lib/postgresql/data \
	  --shm-size=256m \
	  --health-cmd="pg_isready -U $$user -d $$db" \
	  --health-interval=10s --health-timeout=5s --health-retries=5 \
	  postgres:$(DB_VERION); \
	echo "Waiting for $(CONTAINER) to become healthy..."; \
	attempt=0; \
	while :; do \
	  health="$$( $(ENGINE) inspect --format '{{.State.Health.Status}}' $(CONTAINER) 2>/dev/null || true )"; \
	  [ "$$health" = "healthy" ] && break; \
	  if [ "$$health" = "unhealthy" ]; then \
	    echo "$(CONTAINER) failed its health check."; \
	    $(ENGINE) logs $(CONTAINER); \
	    exit 1; \
	  fi; \
	  attempt=$$((attempt + 1)); \
	  if [ "$$attempt" -ge 60 ]; then \
	    echo "Timed out waiting for $(CONTAINER) to become healthy."; \
	    $(ENGINE) logs $(CONTAINER); \
	    exit 1; \
	  fi; \
	  sleep 1; \
	done; \
	echo "$(CONTAINER) is healthy."

## db-down: Stop and remove the PostgreSQL container (data volume is kept)
db-down:
	$(ENGINE) rm -f $(CONTAINER)

## db-backup: Back up the container database to backups/ in PostgreSQL custom format
db-backup:
	@set -eu; \
		mkdir -p "$(BACKUP_DIR)"; \
		backup="$(BACKUP_DIR)/openaiq-backup-$$(date -u +%Y-%m-%d_%H-%M-%S_UTC).dump"; \
		temporary="$$backup.tmp"; \
		trap 'rm -f "$$temporary"' EXIT; \
		test ! -e "$$backup"; \
		$(ENGINE) exec $(CONTAINER) sh -c 'exec pg_dump --format=custom --no-owner --no-privileges --username="$$POSTGRES_USER" --dbname="$$POSTGRES_DB"' > "$$temporary"; \
		mv "$$temporary" "$$backup"; \
		trap - EXIT; \
		echo "Database backup created: $$backup"

## db-backups-clean: Delete all Open AIQ backup files after interactive confirmation
db-backups-clean:
	@set -eu; \
		if [ ! -d "$(BACKUP_DIR)" ]; then \
			echo "No backup directory found: $(BACKUP_DIR)"; \
			exit 0; \
		fi; \
		files="$$(find "$(BACKUP_DIR)" -maxdepth 1 -type f -name 'openaiq-*.dump' -print)"; \
		if [ -z "$$files" ]; then \
			echo "No Open AIQ backups found in $(BACKUP_DIR)."; \
			exit 0; \
		fi; \
		echo "The following backups will be permanently deleted:"; \
		printf '  %s\n' $$files; \
		printf 'Continue? [y/N] '; \
		read -r answer; \
		case "$$answer" in \
			y|Y|yes|YES) ;; \
			*) echo "Cancelled; no backups were deleted."; exit 0 ;; \
		esac; \
		find "$(BACKUP_DIR)" -maxdepth 1 -type f -name 'openaiq-*.dump' -delete; \
		echo "Open AIQ backups deleted."

## db-clean: Back up the database, recreate its container and volume, then start fresh
db-clean: require-database-url
	@$(MAKE) db-backup
	@$(MAKE) db-down
	@$(ENGINE) volume rm $(CONTAINER_VOLUME)
	@$(MAKE) db-up
	@$(MAKE) migrate-up

## db-logs: Tail the PostgreSQL container logs
db-logs:
	$(ENGINE) logs -f $(CONTAINER)

## db-shell: Open a psql shell in the PostgreSQL container
db-shell: require-database-url
	@$(ENGINE) exec -it $(CONTAINER) psql -d "$(DATABASE_URL)"


##.    
## release: Bump version (prompts major/minor/patch), build artifacts, tag, and publish a GitHub release
release:
	@RELEASE_BRANCH="$(RELEASE_BRANCH)" DIST="$(DIST)" BINARY="$(BINARY)" PLATFORMS="$(PLATFORMS)" \
		bash scripts/release.sh