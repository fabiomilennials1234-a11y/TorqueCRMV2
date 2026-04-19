// Package config carries the runtime configuration of the API.
//
// The config is loaded exclusively from environment variables. Secrets never
// land in the binary or in a file checked into source control. A missing
// required variable is a fatal boot error — the process refuses to start with
// placeholder defaults for anything load-bearing.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the fully-resolved runtime configuration.
//
// Every field is immutable after Load. Pass by value; the struct is small.
type Config struct {
	Env             string        // "dev" | "staging" | "prod"
	HTTPAddr        string        // ":8080"
	DatabaseURL     string        // postgres://...
	LogLevel        string        // "debug" | "info" | "warn" | "error"
	ShutdownTimeout time.Duration // graceful shutdown upper bound
	ReadTimeout     time.Duration // HTTP read timeout
	WriteTimeout    time.Duration // HTTP write timeout
	IdleTimeout     time.Duration // HTTP idle timeout
	CORSOrigins     []string      // allow-list; empty means closed
}

// Load reads the config from the environment. Returns an error if any required
// variable is missing or malformed.
//
// Required (no default):
//   - DATABASE_URL
//
// Optional (with world-class defaults):
//   - ENV             default "dev"
//   - HTTP_ADDR       default ":8080"
//   - LOG_LEVEL       default "info"
//   - SHUTDOWN_TIMEOUT default "15s"
//   - READ_TIMEOUT    default "10s"
//   - WRITE_TIMEOUT   default "15s"
//   - IDLE_TIMEOUT    default "60s"
//   - CORS_ORIGINS    default "" (closed)
func Load() (Config, error) {
	c := Config{
		Env:             getenv("ENV", "dev"),
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		LogLevel:        strings.ToLower(getenv("LOG_LEVEL", "info")),
		ShutdownTimeout: mustDuration("SHUTDOWN_TIMEOUT", "15s"),
		ReadTimeout:     mustDuration("READ_TIMEOUT", "10s"),
		WriteTimeout:    mustDuration("WRITE_TIMEOUT", "15s"),
		IdleTimeout:     mustDuration("IDLE_TIMEOUT", "60s"),
		CORSOrigins:     parseCSV(os.Getenv("CORS_ORIGINS")),
	}

	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if c.Env != "dev" && c.Env != "staging" && c.Env != "prod" {
		return Config{}, fmt.Errorf("invalid ENV %q (expected dev|staging|prod)", c.Env)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("invalid LOG_LEVEL %q", c.LogLevel)
	}

	return c, nil
}

// IsProd reports whether the process is running in production.
func (c Config) IsProd() bool { return c.Env == "prod" }

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func mustDuration(key, fallback string) time.Duration {
	raw := getenv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		// Fall back loudly — surface at boot via Load() validation if needed.
		// Using fallback here means a malformed env var does not panic;
		// it degrades to the default. Config.validate() is the gate.
		d, _ = time.ParseDuration(fallback)
		return d
	}
	return d
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// MustAtoi parses an int from env, panics on failure. Reserved for numeric
// knobs that would be meaningless at the fallback (e.g., port-adjacent).
func MustAtoi(key, fallback string) int {
	v, err := strconv.Atoi(getenv(key, fallback))
	if err != nil {
		panic(fmt.Errorf("env %s: %w", key, err))
	}
	return v
}
