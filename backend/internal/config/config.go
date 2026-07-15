package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultHost        = "0.0.0.0"
	defaultPort        = "8080"
	defaultDatabaseURL = "postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable"

	// defaultAccessTokenTTL is the dev default lifetime of a signed JWT
	// access token (spec: "Initial dev access-token lifetime is 30 minutes
	// and configurable").
	defaultAccessTokenTTL = 30 * time.Minute
	// defaultRefreshTokenTTL is the dev default lifetime of a refresh token
	// (30 days).
	defaultRefreshTokenTTL = 720 * time.Hour
	// defaultCookieSecure is false so the refresh cookie works over plain
	// HTTP in local development; every deployed environment must override
	// this to true.
	defaultCookieSecure = false

	// minJWTSecretBytes is the minimum acceptable length (in bytes, of the
	// raw environment value — not decoded) for AUTH_JWT_SECRET. There is no
	// default: an HS256 signing secret must never ship with a checked-in
	// fallback.
	minJWTSecretBytes = 32
)

// Config contains process-level settings that are safe to read from the
// environment at startup.
type Config struct {
	Host        string
	Port        string
	DatabaseURL string

	// JWTSecret is the HMAC key used to sign and verify access tokens
	// (AUTH_JWT_SECRET). Required; Load fails fast if it is missing or too
	// short, the same fail-fast philosophy as the database ping in main.
	JWTSecret []byte
	// AccessTokenTTL is how long a signed access token remains valid
	// (AUTH_ACCESS_TOKEN_TTL).
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is how long a refresh token remains valid
	// (AUTH_REFRESH_TOKEN_TTL).
	RefreshTokenTTL time.Duration
	// CookieSecure controls the Secure attribute on the refresh-token
	// cookie (AUTH_COOKIE_SECURE). Must be true in any deployed
	// environment; false is only safe for local HTTP development.
	CookieSecure bool
}

// Load reads process configuration from the environment, falling back to
// local development defaults where a default is safe. It fails fast (a
// non-nil error) when a required or malformed value would otherwise cause
// silent misconfiguration — most importantly, a missing or too-short
// AUTH_JWT_SECRET, which must never be assumed.
func Load() (Config, error) {
	jwtSecret := os.Getenv("AUTH_JWT_SECRET")
	if len(jwtSecret) < minJWTSecretBytes {
		return Config{}, fmt.Errorf(
			"config: AUTH_JWT_SECRET must be set to at least %d bytes (got %d); generate one with `openssl rand -base64 48`",
			minJWTSecretBytes, len(jwtSecret),
		)
	}

	accessTokenTTL, err := getEnvDuration("AUTH_ACCESS_TOKEN_TTL", defaultAccessTokenTTL)
	if err != nil {
		return Config{}, err
	}
	// time.ParseDuration happily returns zero and negative values, which
	// would make every issued token/cookie already expired — auth would be
	// completely offline behind a valid-looking config. Both lifetimes are
	// truncated to whole seconds downstream (expiresIn, cookie Max-Age), so
	// anything below one second is equally broken. Fail fast instead.
	if accessTokenTTL < time.Second {
		return Config{}, fmt.Errorf("config: AUTH_ACCESS_TOKEN_TTL must be at least 1s, got %v", accessTokenTTL)
	}

	refreshTokenTTL, err := getEnvDuration("AUTH_REFRESH_TOKEN_TTL", defaultRefreshTokenTTL)
	if err != nil {
		return Config{}, err
	}
	if refreshTokenTTL < time.Second {
		return Config{}, fmt.Errorf("config: AUTH_REFRESH_TOKEN_TTL must be at least 1s, got %v", refreshTokenTTL)
	}

	cookieSecure, err := getEnvBool("AUTH_COOKIE_SECURE", defaultCookieSecure)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Host:        getEnv("BACKEND_HOST", defaultHost),
		Port:        getEnv("BACKEND_PORT", defaultPort),
		DatabaseURL: getEnv("DATABASE_URL", defaultDatabaseURL),

		JWTSecret:       []byte(jwtSecret),
		AccessTokenTTL:  accessTokenTTL,
		RefreshTokenTTL: refreshTokenTTL,
		CookieSecure:    cookieSecure,
	}, nil
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

// getEnvDuration parses key as a Go duration (e.g. "30m", "720h"), falling
// back to fallback when unset. A value that is set but fails to parse is a
// configuration error, not silently ignored.
func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("config: %s: invalid duration %q: %w", key, value, err)
	}
	return parsed, nil
}

// getEnvBool parses key as a bool (strconv.ParseBool: "1", "t", "true",
// "0", "f", "false", case-insensitive, and a few more), falling back to
// fallback when unset. A value that is set but fails to parse is a
// configuration error, not silently ignored.
func getEnvBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("config: %s: invalid bool %q: %w", key, value, err)
	}
	return parsed, nil
}
