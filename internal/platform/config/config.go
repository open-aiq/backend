package config

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	goenv "github.com/caarlos0/env/v11"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	Env                    string   `env:"ENV,required" mod:"trim" validate:"required,oneof=development test production"`
	Host                   string   `env:"HOST,required" mod:"trim" validate:"hostname_portless"`
	Port                   uint16   `env:"PORT,required" validate:"required"`
	DatabaseURL            url.URL  `env:"DATABASE_URL,required" validate:"postgres_url"`
	CORSAllowedOrigins     []string `env:"CORS_ALLOWED_ORIGINS,required" envSeparator:"," mod:"dive,trim" validate:"required,min=1,dive,http_origin"`
	ClerkSecretKey         string   `env:"CLERK_SECRET_KEY,required" mod:"trim" validate:"required"`
	ClerkAuthorizedParties []string `env:"CLERK_AUTHORIZED_PARTIES,required" envSeparator:"," mod:"dive,trim" validate:"required,min=1,dive,http_origin"`
	DB                     DBConfig
}

type DBConfig struct {
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS,required" validate:"gte=1"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS,required" validate:"gte=0,ltefield=MaxOpenConns"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME,required" validate:"gte=0"`
}

func (c *Config) Addr() string { return net.JoinHostPort(c.Host, strconv.Itoa(int(c.Port))) }

func (c *Config) IsProduction() bool { return c.Env == "production" }

func Load() (*Config, error) {
	_ = godotenv.Load()
	return parse(os.Environ())
}

func parse(environment []string) (*Config, error) {
	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	cfg, err := goenv.ParseAsWithOptions[Config](goenv.Options{Environment: values})
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	if err := modifiers.New().Struct(context.Background(), &cfg); err != nil {
		return nil, fmt.Errorf("normalize configuration: %w", err)
	}

	v := validator.New(validator.WithRequiredStructEnabled())
	_ = v.RegisterValidation("postgres_url", validatePostgresURL)
	_ = v.RegisterValidation("http_origin", validateHTTPOrigin)
	_ = v.RegisterValidation("hostname_portless", validateHost)
	if err := v.Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	if err := validateOriginSets(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return &cfg, nil
}

func validatePostgresURL(fl validator.FieldLevel) bool {
	u, ok := fl.Field().Interface().(url.URL)
	return ok && (u.Scheme == "postgres" || u.Scheme == "postgresql") && u.Host != ""
}

func validateHTTPOrigin(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "*" {
		return true
	}
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}

func validateHost(fl validator.FieldLevel) bool {
	host := fl.Field().String()
	return host == "" || net.ParseIP(host) != nil || (!strings.Contains(host, ":") && validHostname(host))
}

func validHostname(host string) bool {
	if len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}

func validateOriginSets(cfg *Config) error {
	var errs []error
	if len(cfg.CORSAllowedOrigins) > 1 {
		for _, origin := range cfg.CORSAllowedOrigins {
			if origin == "*" {
				errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS must not combine * with explicit origins"))
				break
			}
		}
	}
	for _, party := range cfg.ClerkAuthorizedParties {
		if party == "*" {
			errs = append(errs, errors.New("CLERK_AUTHORIZED_PARTIES must not contain *"))
		}
	}
	return errors.Join(errs...)
}
