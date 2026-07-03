package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	Env                string
	Host               string
	Port               string
	DatabaseURL        string
	CORSAllowedOrigins []string
	DB                 DBConfig
}

// Addr returns the host:port address the HTTP server should bind to.
// An empty Host means all interfaces (0.0.0.0), reachable from other devices.
func (c *Config) Addr() string {
	return c.Host + c.Port
}

// DBConfig holds connection-pool tuning for the database.
type DBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// IsProduction reports whether the app is running in a production environment.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

// Load reads configuration from the environment and validates it. Every variable
// is required — there are no defaults — so the environment (or .env) is the single
// source of truth. Missing or invalid values are collected and returned together so
// the caller can fail fast.
func Load() (*Config, error) {
	// Load .env if present. Real environment variables take precedence over
	// .env values, and a missing file is not an error (e.g. in production where
	// the variables are set directly).
	_ = godotenv.Load()

	var errs []error

	cfg := &Config{
		Env:                requireString("ENV", &errs),
		Host:               requirePresent("HOST", &errs), // may be empty = all interfaces (0.0.0.0)
		Port:               normalizePort(requireString("PORT", &errs)),
		DatabaseURL:        requireString("DATABASE_URL", &errs),
		CORSAllowedOrigins: requireStringSlice("CORS_ALLOWED_ORIGINS", &errs),
		DB: DBConfig{
			MaxOpenConns:    requireInt("DB_MAX_OPEN_CONNS", &errs),
			MaxIdleConns:    requireInt("DB_MAX_IDLE_CONNS", &errs),
			ConnMaxLifetime: requireDuration("DB_CONN_MAX_LIFETIME", &errs),
		},
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate checks that the configuration values are coherent.
func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	u, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is not a valid URL: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL must use the postgres scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("DATABASE_URL must include a host")
	}

	if c.DB.MaxOpenConns < 1 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be at least 1, got %d", c.DB.MaxOpenConns)
	}
	if c.DB.MaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must not be negative, got %d", c.DB.MaxIdleConns)
	}
	if c.DB.MaxIdleConns > c.DB.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS (%d) must not exceed DB_MAX_OPEN_CONNS (%d)",
			c.DB.MaxIdleConns, c.DB.MaxOpenConns)
	}
	if c.DB.ConnMaxLifetime < 0 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME must not be negative, got %s", c.DB.ConnMaxLifetime)
	}

	return nil
}

// requireString returns the trimmed value of a required environment variable,
// recording an error if it is unset or empty.
func requireString(key string, errs *[]error) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
		return ""
	}
	return v
}

// requireStringSlice parses a required comma-separated environment variable into a
// slice of trimmed, non-empty values, recording an error if it is unset or empty.
func requireStringSlice(key string, errs *[]error) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
		return nil
	}

	var out []string
	for _, part := range strings.Split(v, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// requirePresent returns the trimmed value of a variable that must be present but
// may be empty (e.g. HOST, where empty means "all interfaces"). It records an error
// only when the variable is entirely unset.
func requirePresent(key string, errs *[]error) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		*errs = append(*errs, fmt.Errorf("%s is required (set it empty for all interfaces)", key))
		return ""
	}
	return strings.TrimSpace(v)
}

// requireInt parses a required integer environment variable, recording an error if
// it is unset, empty, or not a valid integer.
func requireInt(key string, errs *[]error) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s must be an integer, got %q", key, v))
		return 0
	}
	return n
}

// requireDuration parses a required duration environment variable (e.g. "5m",
// "30s"), recording an error if it is unset, empty, or not a valid duration.
func requireDuration(key string, errs *[]error) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
		return 0
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s must be a duration (e.g. 5m, 30s), got %q", key, v))
		return 0
	}
	return d
}

// normalizePort ensures the port string is in the ":<port>" form expected by net/http.
func normalizePort(port string) string {
	if !strings.HasPrefix(port, ":") {
		return ":" + port
	}
	return port
}
