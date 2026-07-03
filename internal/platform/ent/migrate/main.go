//go:build ignore

// Command main generates a versioned migration file from the current Ent schema.
//
// It diffs the schema against the existing migration history (replay mode) and
// writes a new pair of golang-migrate .up.sql / .down.sql files into the
// migrations directory. The diff is computed against an ephemeral dev database
// (Docker by default), so Docker must be running.
//
// Run via `make migration name=<description>`.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/jackc/pgx/v5/stdlib"

	"go-aiq-backend/internal/platform/ent/migrate"
)

// Atlas opens the dev database with sql.Open("postgres", …). We don't depend on
// lib/pq, so register the pgx driver the rest of the app already uses under that
// name instead of pulling in a second Postgres driver.
func init() {
	sql.Register("postgres", stdlib.GetDefaultDriver())
}

// migrationsDir is where versioned migration files are written and embedded from.
const migrationsDir = "internal/platform/migration/migrations"

// defaultDevURL points at a local Postgres used only to compute the diff. The
// pure-Go generator needs a real, reachable dev database (the bare docker://
// scheme is an Atlas CLI feature), so `make migration` starts a throwaway
// Postgres and passes its address via ATLAS_DEV_URL — see
// scripts/generate-migration.sh. This constant is only a fallback for manual
// runs against an already-running Postgres.
const defaultDevURL = "postgres://dev:dev@127.0.0.1:5432/dev?sslmode=disable&search_path=public"

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("migration name is required — run: make migration name=<description>")
	}
	name := os.Args[1]

	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		log.Fatalf("create migrations dir: %v", err)
	}

	dir, err := sqltool.NewGolangMigrateDir(migrationsDir)
	if err != nil {
		log.Fatalf("open migrations dir: %v", err)
	}

	devURL := os.Getenv("ATLAS_DEV_URL")
	if devURL == "" {
		devURL = defaultDevURL
	}

	opts := []schema.MigrateOption{
		schema.WithDir(dir),
		schema.WithMigrationMode(schema.ModeReplay),
		schema.WithDialect(dialect.Postgres),
		schema.WithFormatter(sqltool.GolangMigrateFormatter),
	}

	if err := migrate.NamedDiff(context.Background(), devURL, name, opts...); err != nil {
		log.Fatalf("generate migration: %v", err)
	}
}
