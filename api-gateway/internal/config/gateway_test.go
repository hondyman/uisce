package config

import (
	"strings"
	"testing"
)

// clearGatewayEnv makes every variable LoadGatewayConfig reads empty for the test.
func clearGatewayEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ENV", "ENVIRONMENT", "LOG_LEVEL", "PORT", "BACKEND_URL", "AUTH_SERVICE_URL", "REVOCATION_REDIS_ADDR"} {
		t.Setenv(k, "")
	}
}

func TestLoadGatewayConfig_DefaultsMatchTheOriginalContract(t *testing.T) {
	clearGatewayEnv(t)
	cfg, err := LoadGatewayConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := GatewayConfig{
		Env: "development", LogLevel: "info", Port: "8001",
		BackendURL: "http://localhost:8080", AuthServiceURL: "http://auth-service:8001",
		RevocationRedisAddr: "", // empty selects the in-memory revocation store
	}
	if *cfg != want {
		t.Fatalf("defaults = %+v, want %+v", *cfg, want)
	}
}

func TestLoadGatewayConfig_ReadsEnvironment(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv("ENV", "production")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("PORT", "9100")
	t.Setenv("BACKEND_URL", "https://backend.internal:8443")
	t.Setenv("AUTH_SERVICE_URL", "http://auth:9000")
	t.Setenv("REVOCATION_REDIS_ADDR", "redis:6379")
	cfg, err := LoadGatewayConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != "production" || cfg.LogLevel != "warn" || cfg.Port != "9100" ||
		cfg.BackendURL != "https://backend.internal:8443" || cfg.AuthServiceURL != "http://auth:9000" ||
		cfg.RevocationRedisAddr != "redis:6379" {
		t.Fatalf("env not applied: %+v", *cfg)
	}
}

func TestLoadGatewayConfig_RevocationUsesItsOwnVariable(t *testing.T) {
	// REDIS_ADDR is the GraphQL gateway's caching address (config.go); it must
	// NOT switch token revocation to Redis.
	clearGatewayEnv(t)
	t.Setenv("REDIS_ADDR", "redis:6379")
	cfg, err := LoadGatewayConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RevocationRedisAddr != "" {
		t.Fatalf("REDIS_ADDR must not enable Redis revocation, got %q", cfg.RevocationRedisAddr)
	}
}

func TestLoadGatewayConfig_EnvFallsBackToENVIRONMENT(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv("ENVIRONMENT", "staging")
	if cfg, _ := LoadGatewayConfig(); cfg == nil || cfg.Env != "staging" {
		t.Fatalf("Env should fall back to ENVIRONMENT, got %+v", cfg)
	}
	t.Setenv("ENV", "production") // ENV wins when both are set
	if cfg, _ := LoadGatewayConfig(); cfg == nil || cfg.Env != "production" {
		t.Fatalf("ENV should take precedence, got %+v", cfg)
	}
}

func TestLoadGatewayConfig_PortLeadingColonIsTolerated(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv("PORT", ":8081") // main.go prepends ":", so a leading one must not double up
	cfg, err := LoadGatewayConfig()
	if err != nil || cfg.Port != "8081" {
		t.Fatalf("port = %v err = %v, want 8081", cfg, err)
	}
}

func TestLoadGatewayConfig_RejectsBadValues(t *testing.T) {
	for _, tc := range []struct{ name, key, val, wantErr string }{
		{"non-numeric port", "PORT", "abc", "PORT"},
		{"port zero", "PORT", "0", "PORT"},
		{"port too high", "PORT", "70000", "PORT"},
		{"backend without scheme", "BACKEND_URL", "localhost:8080", "BACKEND_URL"},
		{"backend wrong scheme", "BACKEND_URL", "ftp://host", "BACKEND_URL"},
		{"backend without host", "BACKEND_URL", "http://", "BACKEND_URL"},
		{"auth url garbage", "AUTH_SERVICE_URL", "not a url", "AUTH_SERVICE_URL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearGatewayEnv(t)
			t.Setenv(tc.key, tc.val)
			cfg, err := LoadGatewayConfig()
			if err == nil || cfg != nil {
				t.Fatalf("expected an error, got cfg=%+v err=%v", cfg, err)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q should name %s", err, tc.wantErr)
			}
		})
	}
}
