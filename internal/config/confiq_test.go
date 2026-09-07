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
}

func TestLoadConfigInvalidPort(t *testing.T) {
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
	t.Setenv("DATABASE_URL", "")

	_, err := Load()

	if err == nil {
		t.Fatalf("Expected error for empty DATABASE_URL, got nil")
	}
}
