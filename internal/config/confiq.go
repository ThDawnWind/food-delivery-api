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
}

type HTTPConfig struct {
	Port              int
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL string
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

	return Config{
		Env: env,
		HTTP: HTTPConfig{
			Port:              portInt,
			ReadHeaderTimeout: readHeaderTimeoutParseDuration,
			WriteTimeout:      writeTimeoutParseDuration,
			IdleTimeout:       idleTimeoutParseDuration,
		},
		Database: DatabaseConfig{
			URL: databaseURL,
		},
		JWT: &JWTConfig{
			Secret: jwtSecret,
			TTL:    jwtTTL,
		},
	}, nil
}
