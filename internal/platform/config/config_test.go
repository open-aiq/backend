package config

import (
	"strings"
	"testing"
	"time"
)

func validEnvironment() []string {
	return []string{
		"ENV=development", "HOST=", "PORT=8080",
		"DATABASE_URL=postgres://user:pass@localhost:5432/openaiq?sslmode=disable",
		"CORS_ALLOWED_ORIGINS= http://localhost:5173 , https://openaiq.org ",
		"CLERK_SECRET_KEY= secret ",
		"CLERK_AUTHORIZED_PARTIES=http://localhost:5173,https://openaiq.org",
		"DB_MAX_OPEN_CONNS=25", "DB_MAX_IDLE_CONNS=5", "DB_CONN_MAX_LIFETIME=5m",
	}
}

func TestParseTransformsTypedConfiguration(t *testing.T) {
	cfg, err := parse(validEnvironment())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr() != ":8080" {
		t.Fatalf("Addr() = %q", cfg.Addr())
	}
	if cfg.DatabaseURL.Scheme != "postgres" {
		t.Fatalf("scheme = %q", cfg.DatabaseURL.Scheme)
	}
	if cfg.DB.ConnMaxLifetime != 5*time.Minute {
		t.Fatalf("lifetime = %s", cfg.DB.ConnMaxLifetime)
	}
	if cfg.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Fatalf("origin was not trimmed: %q", cfg.CORSAllowedOrigins[0])
	}
	if cfg.ClerkSecretKey != "secret" {
		t.Fatalf("secret was not trimmed: %q", cfg.ClerkSecretKey)
	}
}

func TestParseRejectsInvalidConfiguration(t *testing.T) {
	tests := map[string]func([]string) []string{
		"missing values":          func(_ []string) []string { return nil },
		"invalid postgres scheme": func(env []string) []string { return replace(env, "DATABASE_URL", "mysql://localhost/db") },
		"wildcard with origin":    func(env []string) []string { return replace(env, "CORS_ALLOWED_ORIGINS", "*,https://openaiq.org") },
		"idle exceeds open":       func(env []string) []string { return replace(env, "DB_MAX_IDLE_CONNS", "26") },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parse(mutate(validEnvironment())); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseSupportsIPv6Host(t *testing.T) {
	cfg, err := parse(replace(validEnvironment(), "HOST", "::1"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr() != "[::1]:8080" {
		t.Fatalf("Addr() = %q", cfg.Addr())
	}
}

func replace(environment []string, key, value string) []string {
	out := append([]string(nil), environment...)
	prefix := key + "="
	for i := range out {
		if strings.HasPrefix(out[i], prefix) {
			out[i] = prefix + value
			return out
		}
	}
	return append(out, prefix+value)
}
