ENGINE := podman
CONTAINER := openaiq-postgres
CONTAINER_VOLUME := openaiq-pgdata
DB_VERSION := 18
BACKUP_DIR ?= backups
TOOLS_DIR    := $(CURDIR)/.tools
AQUA_VERSION := v2.66.0
export AQUA_ROOT_DIR := $(TOOLS_DIR)/aqua
export GOTOOLCHAIN   := go1.26.1
# A GOROOT exported by a version manager (e.g. mise) would override the pinned toolchain.
unexport GOROOT
export PATH          := $(AQUA_ROOT_DIR)/bin:$(TOOLS_DIR)/bin:$(PATH)
ATLAS := $(AQUA_ROOT_DIR)/bin/atlas
SWAG  := $(AQUA_ROOT_DIR)/bin/swag
AIR   := $(AQUA_ROOT_DIR)/bin/air
HL    := $(AQUA_ROOT_DIR)/bin/hl

# Load .env so DATABASE_URL is available to the db-* targets.
# DATABASE_URL has no fallback — it must be defined in .env (see .env.example).
-include .env

# Release configuration.
RELEASE_BRANCH ?= main
DIST           ?= dist
BINARY         ?= server
PLATFORMS      ?= linux/amd64 linux/arm64 darwin/arm64

.PHONY: help init dev dev\:raw format\:fix format\:check lint\:check lint\:fix test verify swagger\:gen build build\:run build\:clean orm\:gen migration\:gen migration\:apply require-database-url db\:seed db\:up db\:down db\:backup db\:backups\:clean db\:clean db\:logs db\:shell release

## help: Show available commands
help:
	@echo "Available commands:"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | awk '{ i = index($$0, ": "); if (i) printf "  %-20s %s\n", substr($$0, 1, i - 1), substr($$0, i + 2); else print "" }'

## init: install deps and setup the path, run this command before starting the work
init:
	@test -x "$(TOOLS_DIR)/bin/aqua" || GOTOOLCHAIN=auto GOBIN="$(TOOLS_DIR)/bin" go install github.com/aquaproj/aqua/v2/cmd/aqua@$(AQUA_VERSION)
	@"$(TOOLS_DIR)/bin/aqua" install
	@test -f .envrc || printf '%s\n' \
	  'PATH_add .tools/bin' \
	  'export AQUA_ROOT_DIR="$$PWD/.tools/aqua"' \
	  'PATH_add .tools/aqua/bin' \
	  'export GOTOOLCHAIN=go1.26.1' \
	  'dotenv_if_exists .env' > .envrc
	@echo "Tools ready. With direnv: run 'direnv allow' once."

## dev: Live reload with swagger regeneration; JSON logs are pretty-printed by hl
dev:
	@"$(AIR)" 2>&1 | "$(HL)" -P -L

## dev:raw: Same as dev, but prints the raw JSON log lines
dev\:raw:
	"$(AIR)"

## format:fix: Format Go sources with gofmt
format\:fix:
	gofmt -w cmd internal

## format:check: Check Go formatting without changing files
format\:check:
	@files="$$(gofmt -l cmd internal)"; [ -z "$$files" ] || { echo "not gofmt-formatted:"; echo "$$files"; exit 1; }

## lint:check: Run go vet
lint\:check:
	go vet ./...

## lint:fix: Apply go fix's automatic modernizations
lint\:fix:
	go fix ./...

## test: Run the Go tests
test:
	go test ./...

## verify: Run format:check, lint:check and test
verify:
	@$(MAKE) --no-print-directory format:check
	@$(MAKE) --no-print-directory lint:check
	@$(MAKE) --no-print-directory test

## swagger:gen: Generate swagger docs
swagger\:gen:
	"$(SWAG)" init -g cmd/server/main.go -o docs
	go run ./cmd/problemgen -out docs/problems.json


## build: Regenerate code, generate swagger, and build the binary
build:
	@$(MAKE) --no-print-directory orm:gen
	@$(MAKE) --no-print-directory swagger:gen
	go build -o bin/server ./cmd/server/

## build:run: Build and run the server
build\:run: build
	./bin/server

## build:clean: Remove build artifacts
build\:clean:
	rm -rf bin/ tmp/ $(DIST)/

## orm:gen: Regenerate the Ent client after editing a schema (internal/platform/ent/schema/)
orm\:gen:
	go generate ./internal/platform/ent

## migration:gen: Generate a versioned migration from schema changes. usage: make migration:gen name=<description>
# Atlas's docker:// driver always invokes a binary named "docker". The target
# creates a temporary PATH entry with that name pointing to ENGINE (Podman by
# default), then removes it when migration generation finishes.
migration\:gen:
	@test -n "$(name)" || { echo "usage: make migration:gen name=<description>"; exit 1; }
	@command -v "$(ENGINE)" >/dev/null || { echo "container engine '$(ENGINE)' is not installed"; exit 1; }
	@set -eu; \
		cli_dir="$$(mktemp -d)"; \
		trap 'rm -rf "$$cli_dir"' EXIT; \
		ln -s "$$(command -v "$(ENGINE)")" "$$cli_dir/docker"; \
		PATH="$$cli_dir:$$PATH" "$(ATLAS)" migrate diff "$(name)" --env defaultConfig

## migration:apply: Apply all pending migrations to DATABASE_URL
migration\:apply: require-database-url
	"$(ATLAS)" migrate apply --env defaultConfig --url "$(DATABASE_URL)"




# Fail fast if DATABASE_URL isn't defined (no fallback; see .env.example).
require-database-url:
	@if [ -z "$(DATABASE_URL)" ]; then \
	  echo "DATABASE_URL is not set. Define it in .env (see .env.example)."; \
	  exit 1; \
	fi

## db:seed: Seed mock readings and map-enable a device — usage: make db:seed device=dev_<id> [map=false]
db\:seed: require-database-url
	@test -n "$(device)" || { echo "usage: make db:seed device=dev_<id>"; exit 1; }
	@DEVICE_ID="$(device)" DATABASE_URL="$(DATABASE_URL)" ENGINE="$(ENGINE)" CONTAINER="$(CONTAINER)" MAP_VISIBLE="$(if $(map),$(map),true)" bash scripts/seed_readings.sh

## db:up: Start the PostgreSQL container (creds derived from DATABASE_URL)
db\:up: require-database-url
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
	  postgres:$(DB_VERSION)-alpine; \
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

## db:down: Stop and remove the PostgreSQL container (data volume is kept)
db\:down:
	$(ENGINE) rm -f $(CONTAINER)

## db:backup: Back up the container database to backups/ in PostgreSQL custom format
db\:backup:
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

## db:backups:clean: Delete all Open AIQ backup files after interactive confirmation
db\:backups\:clean:
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

## db:clean: Back up the database, recreate its container and volume, then start fresh
db\:clean: require-database-url
	@$(MAKE) db:backup
	@$(MAKE) db:down
	@$(ENGINE) volume rm $(CONTAINER_VOLUME)
	@$(MAKE) db:up
	@$(MAKE) migration:apply

## db:logs: Tail the PostgreSQL container logs
db\:logs:
	$(ENGINE) logs -f $(CONTAINER)

## db:shell: Open a psql shell in the PostgreSQL container
db\:shell: require-database-url
	@$(ENGINE) exec -it $(CONTAINER) psql -d "$(DATABASE_URL)"


##.
## release: Build, tag, and publish — make release [version=major|minor|patch|1.2.3|1.2.3-rc.1]
release:
	@RELEASE_BRANCH="$(RELEASE_BRANCH)" RELEASE_VERSION="$(version)" DIST="$(DIST)" BINARY="$(BINARY)" PLATFORMS="$(PLATFORMS)" \
		bash scripts/release.sh
