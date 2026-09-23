package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	HTTP     HTTPConfig
	Database DatabaseConfig
	JWT      *JWTConfig
	Timezone string
}

type HTTPConfig struct {
	Port              int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type JWTConfig struct {
	Secret string
	TTL    time.Duration
}

func Load() (Config, error) {
	env, exists := os.LookupEnv("APP_ENV")
	if !exists {
		env = "local"
	}

	port, exists := os.LookupEnv("HTTP_PORT")
	if !exists {
		port = "8080"
	}

	readTimeout, exists := os.LookupEnv("HTTP_READ_TIMEOUT")
	if !exists {
		readTimeout = "15s"
	}

	readTimeoutParseDuration, err := time.ParseDuration(readTimeout)
	if err != nil {
		return Config{}, fmt.Errorf(
			"Invalid HTTP_READ_TIMEOUT value: %w",
			err,
		)
	}

	if readTimeoutParseDuration <= 0 {
		return Config{}, fmt.Errorf(
			"HTTP_READ_TIMEOUT must be greater than zero",
		)
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return Config{}, fmt.Errorf("Invalid HTTP_PORT value: %w", err)
	}

	if portInt < 1 || portInt > 65535 {
		return Config{}, fmt.Errorf("HTTP_PORT must be between 1 and 65535, got %d", portInt)
	}

	readHeaderTimeout, exists := os.LookupEnv("HTTP_READ_HEADER_TIMEOUT")
	if !exists {
		readHeaderTimeout = "5s"
	}

	readHeaderTimeoutParseDuration, err := time.ParseDuration(readHeaderTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("Invalid HTTP_READ_HEADER_TIMEOUT value: %w", err)
	}

	if readHeaderTimeoutParseDuration <= 0 {
		return Config{}, fmt.Errorf("HTTP_READ_HEADER_TIMEOUT must be greater than or equal to 0")
	}

	readWriteTimeout, exists := os.LookupEnv("HTTP_WRITE_TIMEOUT")
	if !exists {
		readWriteTimeout = "10s"
	}

	writeTimeoutParseDuration, err := time.ParseDuration(readWriteTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("Invalid HTTP_WRITE_TIMEOUT value: %w", err)
	}

	if writeTimeoutParseDuration < 0 {
		return Config{}, fmt.Errorf("HTTP_WRITE_TIMEOUT must be greater than or equal to 0")
	}

	idleTimeout, exists := os.LookupEnv("HTTP_IDLE_TIMEOUT")
	if !exists {
		idleTimeout = "60s"
	}

	idleTimeoutParseDuration, err := time.ParseDuration(idleTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("Invalid HTTP_IDLE_TIMEOUT value: %w", err)
	}

	if idleTimeoutParseDuration <= 0 {
		return Config{}, fmt.Errorf("HTTP_IDLE_TIMEOUT must be greater than or equal to 0")
	}

	databaseURL, exists := os.LookupEnv("DATABASE_URL")
	if !exists || databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	dbMaxConns, err := parseInt32Env("DB_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}

	if dbMaxConns <= 0 {
		return Config{}, errors.New("DB_MAX_CONNS must be greater than zero")
	}

	dbMinConns, err := parseInt32Env("DB_MIN_CONNS", 1)
	if err != nil {
		return Config{}, err
	}

	if dbMinConns < 0 {
		return Config{}, errors.New("DB_MIN_CONNS must be greater than or equal to zero")
	}

	if dbMinConns > dbMaxConns {
		return Config{}, errors.New("DB_MIN_CONNS must not be greater than DB_MAX_CONNS")
	}

	dbMaxConnLifetime, err := parseDurationEnv("DB_MAX_CONN_LIFETIME", "1h")
	if err != nil {
		return Config{}, err
	}

	if dbMaxConnLifetime <= 0 {
		return Config{}, errors.New("DB_MAX_CONN_LIFETIME must be greater than zero")
	}

	dbMaxConnIdleTime, err := parseDurationEnv("DB_MAX_CONN_IDLE_TIME", "30m")
	if err != nil {
		return Config{}, err
	}

	if dbMaxConnIdleTime <= 0 {
		return Config{}, errors.New("DB_MAX_CONN_IDLE_TIME must be greater than zero")
	}

	dbHealthCheckPeriod, err := parseDurationEnv("DB_HEALTH_CHECK_PERIOD", "1m")
	if err != nil {
		return Config{}, err
	}

	if dbHealthCheckPeriod <= 0 {
		return Config{}, errors.New("DB_HEALTH_CHECK_PERIOD must be greater than zero")
	}

	jwtSecret := strings.TrimSpace(
		os.Getenv("JWT_SECRET"),
	)

	if len(jwtSecret) < 32 {
		return Config{}, errors.New(
			"JWT_SECRET must be at least 32 characters",
		)
	}

	jwtTTLRaw := os.Getenv("JWT_TTL")
	if jwtTTLRaw == "" {
		jwtTTLRaw = "24h"
	}

	jwtTTL, err := time.ParseDuration(jwtTTLRaw)
	if err != nil {
		return Config{}, fmt.Errorf(
			"invalid JWT_TTL: %w",
			err,
		)
	}

	if jwtTTL <= 0 {
		return Config{}, errors.New(
			"JWT_TTL must be greater than zero",
		)
	}

	timezone := os.Getenv("APP_TIMEZONE")
	if timezone == "" {
		timezone = "UTC"
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return Config{}, fmt.Errorf(
			"invalid APP_TIMEZONE %q: %w",
			timezone,
			err,
		)
	}

	return Config{
		Env: env,
		HTTP: HTTPConfig{
			Port:              portInt,
			ReadTimeout:       readTimeoutParseDuration,
			ReadHeaderTimeout: readHeaderTimeoutParseDuration,
			WriteTimeout:      writeTimeoutParseDuration,
			IdleTimeout:       idleTimeoutParseDuration,
		},
		Database: DatabaseConfig{
			URL:               databaseURL,
			MaxConns:          dbMaxConns,
			MinConns:          dbMinConns,
			MaxConnLifetime:   dbMaxConnLifetime,
			MaxConnIdleTime:   dbMaxConnIdleTime,
			HealthCheckPeriod: dbHealthCheckPeriod,
		},
		JWT: &JWTConfig{
			Secret: jwtSecret,
			TTL:    jwtTTL,
		},
		Timezone: timezone,
	}, nil
}

func parseInt32Env(name string, defaultValue int32) (int32, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value: %w", name, err)
	}

	return int32(value), nil
}

func parseDurationEnv(name string, defaultValue string) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		raw = defaultValue
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value: %w", name, err)
	}

	return value, nil
}
