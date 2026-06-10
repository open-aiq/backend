package config

// Config holds the application configuration.
type Config struct {
	Port string
}

// Load returns the application configuration.
func Load() *Config {
	return &Config{
		Port: ":8080",
	}
}
