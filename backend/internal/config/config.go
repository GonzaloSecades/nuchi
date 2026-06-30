package config

import "os"

const (
	defaultHost = "0.0.0.0"
	defaultPort = "8080"
)

// Config contains process-level settings that are safe to read from the
// environment at startup. Database and auth settings are intentionally absent
// until those scoped issues add them.
type Config struct {
	Host string
	Port string
}

// Load reads API host and port from the environment, falling back to local
// development defaults.
func Load() Config {
	return Config{
		Host: getEnv("BACKEND_HOST", defaultHost),
		Port: getEnv("BACKEND_PORT", defaultPort),
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
