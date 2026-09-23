package config

import (
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "5s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "10s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "60s")
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/dbname")
	t.Setenv("JWT_SECRET", "this-is-a-test-secret-key-with-32-characters")
	t.Setenv("DB_MAX_CONNS", "10")
	t.Setenv("DB_MIN_CONNS", "1")
	t.Setenv("DB_MAX_CONN_LIFETIME", "1h")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "30m")
	t.Setenv("DB_HEALTH_CHECK_PERIOD", "1m")
	t.Setenv("HTTP_READ_TIMEOUT", "15s")

	expectedDatabaseURL := "postgres://user:password@localhost:5432/dbname"

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Env != "test" {
		t.Errorf("Expected Env to be 'test', got %s", cfg.Env)
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("Expected HTTP Port to be 8080, got %d", cfg.HTTP.Port)
	}
	if cfg.HTTP.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("Expected ReadHeaderTimeout to be 5s, got %v", cfg.HTTP.ReadHeaderTimeout)
	}
	if cfg.HTTP.WriteTimeout != 10*time.Second {
		t.Errorf("Expected WriteTimeout to be 10s, got %v", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.IdleTimeout != 60*time.Second {
		t.Errorf("Expected IdleTimeout to be 60s, got %v", cfg.HTTP.IdleTimeout)
	}
	if cfg.Database.URL != expectedDatabaseURL {
		t.Errorf("Expected Database URL to be '%s', got %s", expectedDatabaseURL, cfg.Database.URL)
	}
	if cfg.Database.MaxConns != 10 {
		t.Errorf("expected DB MaxConns 10, got %d", cfg.Database.MaxConns)
	}

	if cfg.Database.MinConns != 1 {
		t.Errorf("expected DB MinConns 1, got %d", cfg.Database.MinConns)
	}

	if cfg.Database.MaxConnLifetime != time.Hour {
		t.Errorf(
			"expected DB MaxConnLifetime %v, got %v",
			time.Hour,
			cfg.Database.MaxConnLifetime,
		)
	}

	if cfg.Database.MaxConnIdleTime != 30*time.Minute {
		t.Errorf(
			"expected DB MaxConnIdleTime %v, got %v",
			30*time.Minute,
			cfg.Database.MaxConnIdleTime,
		)
	}

	if cfg.Database.HealthCheckPeriod != time.Minute {
		t.Errorf(
			"expected DB HealthCheckPeriod %v, got %v",
			time.Minute,
			cfg.Database.HealthCheckPeriod,
		)
	}
	if cfg.JWT == nil {
		t.Fatal("expected JWT config, got nil")
	}

	if cfg.JWT.Secret != "this-is-a-test-secret-key-with-32-characters" {
		t.Errorf(
			"unexpected JWT secret: %q",
			cfg.JWT.Secret,
		)
	}

	if cfg.JWT.TTL != 24*time.Hour {
		t.Errorf(
			"expected JWT TTL %v, got %v",
			24*time.Hour,
			cfg.JWT.TTL,
		)
	}

	if cfg.HTTP.ReadTimeout != 15*time.Second {
		t.Errorf(
			"Expected ReadTimeout to be 15s, got %v",
			cfg.HTTP.ReadTimeout,
		)
	}
}

func TestLoadConfigInvalidPort(t *testing.T) {
	setRequiredEnv(t)
	tests := []struct {
		name string
		port string
	}{
		{
			name: "invalid port",
			port: "invalid",
		},
		{
			name: "port out of range",
			port: "70000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HTTP_PORT", tt.port)
			_, err := Load()
			if err == nil {
				t.Fatalf("expected error for port %s, got nil", tt.port)
			}
		})
	}
}

func TestLoadConfigInvalidTimeout(t *testing.T) {
	setRequiredEnv(t)
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{
			name:  "invalid read header timeout",
			env:   "HTTP_READ_HEADER_TIMEOUT",
			value: "invalid",
		},
		{
			name:  "invalid write timeout",
			env:   "HTTP_WRITE_TIMEOUT",
			value: "invalid",
		},
		{
			name:  "invalid idle timeout",
			env:   "HTTP_IDLE_TIMEOUT",
			value: "invalid",
		},
		{
			name:  "invalid read timeout",
			env:   "HTTP_READ_TIMEOUT",
			value: "invalid",
		},
		{
			name:  "zero read timeout",
			env:   "HTTP_READ_TIMEOUT",
			value: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.env, tt.value)
			_, err := Load()
			if err == nil {
				t.Fatalf("expected error for %s-%s, got nil", tt.env, tt.value)
			}
		})
	}
}

func TestLoadConfigEmptyDatabaseURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()

	if err == nil {
		t.Fatalf("Expected error for empty DATABASE_URL, got nil")
	}
}

func TestLoadConfigInvalidJWTSecret(t *testing.T) {
	t.Setenv(
		"DATABASE_URL",
		"postgres://test:test@localhost:5432/test",
	)

	t.Setenv(
		"JWT_SECRET",
		"short",
	)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadConfigInvalidJWTTTL(t *testing.T) {
	setRequiredEnv(t)

	t.Setenv(
		"JWT_TTL",
		"invalid",
	)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadConfigNegativeJWTTTL(t *testing.T) {
	setRequiredEnv(t)

	t.Setenv(
		"JWT_TTL",
		"-1h",
	)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv(
		"DATABASE_URL",
		"postgres://test:test@localhost:5432/test",
	)

	t.Setenv(
		"JWT_SECRET",
		"this-is-a-test-secret-key-with-32-characters",
	)

	t.Setenv("DB_MAX_CONNS", "10")
	t.Setenv("DB_MIN_CONNS", "1")
	t.Setenv("DB_MAX_CONN_LIFETIME", "1h")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "30m")
	t.Setenv("DB_HEALTH_CHECK_PERIOD", "1m")
}

func TestLoadConfigDatabasePoolDefaults(t *testing.T) {
	setRequiredEnv(t)

	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_MIN_CONNS", "")
	t.Setenv("DB_MAX_CONN_LIFETIME", "")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "")
	t.Setenv("DB_HEALTH_CHECK_PERIOD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Database.MaxConns != 10 {
		t.Errorf("expected default MaxConns 10, got %d", cfg.Database.MaxConns)
	}

	if cfg.Database.MinConns != 1 {
		t.Errorf("expected default MinConns 1, got %d", cfg.Database.MinConns)
	}

	if cfg.Database.MaxConnLifetime != time.Hour {
		t.Errorf(
			"expected default MaxConnLifetime %v, got %v",
			time.Hour,
			cfg.Database.MaxConnLifetime,
		)
	}

	if cfg.Database.MaxConnIdleTime != 30*time.Minute {
		t.Errorf(
			"expected default MaxConnIdleTime %v, got %v",
			30*time.Minute,
			cfg.Database.MaxConnIdleTime,
		)
	}

	if cfg.Database.HealthCheckPeriod != time.Minute {
		t.Errorf(
			"expected default HealthCheckPeriod %v, got %v",
			time.Minute,
			cfg.Database.HealthCheckPeriod,
		)
	}
}

func TestLoadConfigInvalidDatabasePoolConfig(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{
			name:  "invalid max connections",
			env:   "DB_MAX_CONNS",
			value: "invalid",
		},
		{
			name:  "zero max connections",
			env:   "DB_MAX_CONNS",
			value: "0",
		},
		{
			name:  "negative min connections",
			env:   "DB_MIN_CONNS",
			value: "-1",
		},
		{
			name:  "invalid max connection lifetime",
			env:   "DB_MAX_CONN_LIFETIME",
			value: "invalid",
		},
		{
			name:  "zero max connection lifetime",
			env:   "DB_MAX_CONN_LIFETIME",
			value: "0s",
		},
		{
			name:  "invalid max idle time",
			env:   "DB_MAX_CONN_IDLE_TIME",
			value: "invalid",
		},
		{
			name:  "zero max idle time",
			env:   "DB_MAX_CONN_IDLE_TIME",
			value: "0s",
		},
		{
			name:  "invalid health check period",
			env:   "DB_HEALTH_CHECK_PERIOD",
			value: "invalid",
		},
		{
			name:  "zero health check period",
			env:   "DB_HEALTH_CHECK_PERIOD",
			value: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tt.env, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatalf(
					"expected error for %s=%q, got nil",
					tt.env,
					tt.value,
				)
			}
		})
	}
}

func TestLoadConfigMinConnsGreaterThanMaxConns(t *testing.T) {
	setRequiredEnv(t)

	t.Setenv("DB_MAX_CONNS", "5")
	t.Setenv("DB_MIN_CONNS", "6")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DB_MIN_CONNS is greater than DB_MAX_CONNS")
	}
}

func TestLoadConfigRejectsExampleJWTSecretInProduction(t *testing.T) {
	setRequiredEnv(t)

	t.Setenv(
		"APP_ENV",
		"production",
	)

	t.Setenv(
		"JWT_SECRET",
		insecureExampleJWTSecret,
	)

	_, err := Load()
	if err == nil {
		t.Fatal(
			"expected error for example JWT secret in production",
		)
	}
}

func TestLoadConfigAllowsCustomJWTSecretInProduction(t *testing.T) {
	setRequiredEnv(t)

	t.Setenv(
		"APP_ENV",
		"production",
	)

	t.Setenv(
		"JWT_SECRET",
		"gN7vP2xQ9mK4sR8wT3yL6cF1hJ5dB0zA",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if cfg.Env != "production" {
		t.Errorf(
			"expected production environment, got %q",
			cfg.Env,
		)
	}
}
