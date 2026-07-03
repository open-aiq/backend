#!/bin/sh
set -e

# Apply any pending database migrations before the server starts.
# Idempotent: a no-op when the database is already at the latest version.
# Atlas takes a lock, so concurrent boots don't race.
atlas migrate apply --url "$DATABASE_URL" --dir "file:///migrations"

# Hand off to the server as PID 1 so it receives signals for graceful shutdown.
exec server
