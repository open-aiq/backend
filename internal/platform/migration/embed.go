// Package migration embeds the versioned SQL migration files so they can be
// applied from a self-contained binary (see cmd/migrate) without shipping the
// migrations directory alongside it.
package migration

import "embed"

// FS holds the versioned golang-migrate migration files. New migrations are
// generated with `make migration name=<description>`.
//
//go:embed migrations/*.sql
var FS embed.FS
