package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Env         string
	Host        string
	Port        string
	DatabaseURL string
	DB          DBConfig
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

// Load reads configuration from the environment, applies defaults, and validates it.
// It returns an error if any value is missing or invalid so the caller can fail fast.
func Load() (*Config, error) {
	cfg := &Config{
		Env:         getenv("ENV", "development"),
		Host:        getenv("HOST", ""), // empty = all interfaces (0.0.0.0)
		Port:        normalizePort(getenv("PORT", "8080")),
		DatabaseURL: getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/openaiq?sslmode=disable"),
		DB: DBConfig{
			MaxOpenConns:    getenvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getenvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getenvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
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

// getenv returns the trimmed value of the environment variable named by key,
// or fallback if the variable is unset or empty.
func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// getenvInt parses an integer environment variable, falling back on unset/invalid input.
func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// getenvDuration parses a duration environment variable (e.g. "5m", "30s"),
// falling back on unset/invalid input.
func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
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
