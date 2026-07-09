package config

import "os"

const (
	defaultHost        = "0.0.0.0"
	defaultPort        = "8080"
	defaultDatabaseURL = "postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable"
)

// Config contains process-level settings that are safe to read from the
// environment at startup. Auth settings are intentionally absent until that
// scoped issue adds them.
type Config struct {
	Host        string
	Port        string
	DatabaseURL string
}

// Load reads API host, port, and database URL from the environment, falling
// back to local development defaults.
func Load() Config {
	return Config{
		Host:        getEnv("BACKEND_HOST", defaultHost),
		Port:        getEnv("BACKEND_PORT", defaultPort),
		DatabaseURL: getEnv("DATABASE_URL", defaultDatabaseURL),
	}
}

// Addr returns the listen address accepted by net/http.
func (c Config) Addr() string {
	return c.Host + ":" + c.Port
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
