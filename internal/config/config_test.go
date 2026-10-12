package config

import (
	"testing"
)

func TestParseFlags_DefaultWhenEmpty(t *testing.T) {
	t.Setenv(EnvJWTSecret, "secret")
	t.Setenv(EnvRunAddress, "")
	t.Setenv(EnvAccrualAddress, "")

	cfg, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != DefaultRunAddress {
		t.Fatalf("want %q, got %q", DefaultRunAddress, cfg.RunAddress)
	}
	if cfg.AccrualAddress != DefaultAccrualAddr {
		t.Fatalf("want %q, got %q", DefaultAccrualAddr, cfg.AccrualAddress)
	}
}

func TestParseFlags_FlagOverridesDefault(t *testing.T) {
	t.Setenv(EnvJWTSecret, "secret")

	cfg, err := parseFlags([]string{"-a", ":9999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != ":9999" {
		t.Fatalf("want :9999, got %q", cfg.RunAddress)
	}
}

func TestParseFlags_EnvOverridesFlag(t *testing.T) {
	t.Setenv(EnvJWTSecret, "secret")
	t.Setenv(EnvRunAddress, ":8888")

	cfg, err := parseFlags([]string{"-a", ":9999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunAddress != ":8888" {
		t.Fatalf("want :8888 (env wins), got %q", cfg.RunAddress)
	}
}

func TestParseFlags_GeneratesSecretWhenMissing(t *testing.T) {
	t.Setenv(EnvJWTSecret, "")

	cfg, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.JWTSecret == "" {
		t.Fatal("expected non-empty generated JWT secret")
	}

	// Два вызова должны дать разные секреты — иначе это дефолт, а не рандом.
	cfg2, _ := parseFlags(nil)
	if cfg.JWTSecret == cfg2.JWTSecret {
		t.Fatal("expected different secrets on each run, got identical")
	}
}
