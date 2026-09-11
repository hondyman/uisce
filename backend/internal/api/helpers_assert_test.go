package api

// White-box tests for AssertProductionConfig.
// Package api (not api_test) so we can manipulate env without export,
// and call the unexported getEnv path directly to confirm behavior.

import (
	"os"
	"strings"
	"testing"
)

// setEnvForTest sets and then restores environment variables for a single test.
func setEnvForTest(t *testing.T, pairs ...string) {
	t.Helper()
	if len(pairs)%2 != 0 {
		t.Fatal("setEnvForTest: pairs must be even (key, value, key, value, ...)")
	}
	for i := 0; i < len(pairs); i += 2 {
		key, val := pairs[i], pairs[i+1]
		prev, had := os.LookupEnv(key)
		if val == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, val)
		}
		t.Cleanup(func() {
			if had {
				os.Setenv(key, prev)
			} else {
				os.Unsetenv(key)
			}
		})
	}
}

func unsetForTest(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		prev, had := os.LookupEnv(key)
		os.Unsetenv(key)
		t.Cleanup(func() {
			if had {
				os.Setenv(key, prev)
			} else {
				os.Unsetenv(key)
			}
		})
	}
}

// ── Clean production ─────────────────────────────────────────────────────────

func TestAssertProductionConfig_productionClean(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "production",
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "false",
	)
	if err := AssertProductionConfig(); err != nil {
		t.Fatalf("expected nil for clean production config, got: %v", err)
	}
}

// ── Production + unsafe flags ─────────────────────────────────────────────────

func TestAssertProductionConfig_productionHeaderFallback(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "production",
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "false",
	)
	err := AssertProductionConfig()
	if err == nil {
		t.Fatal("expected error for ALLOW_CLIENT_TENANT_HEADER_FALLBACK=true in production, got nil")
	}
	if !strings.Contains(err.Error(), "ALLOW_CLIENT_TENANT_HEADER_FALLBACK") {
		t.Errorf("error message should mention the flag; got: %v", err)
	}
}

func TestAssertProductionConfig_productionDevKey(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "production",
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "true",
	)
	err := AssertProductionConfig()
	if err == nil {
		t.Fatal("expected error for API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK=true in production, got nil")
	}
	if !strings.Contains(err.Error(), "API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK") {
		t.Errorf("error message should mention the flag; got: %v", err)
	}
}

// ── UNSET ENVIRONMENT — the critical fail-closed case ─────────────────────────
// This is the most dangerous misconfiguration: a server deployed without
// ENVIRONMENT set, combined with a dev flag still true, must be rejected.
// If this test passes with err==nil, the control is fail-open (broken).

func TestAssertProductionConfig_unsetEnvironment_headerFallback(t *testing.T) {
	unsetForTest(t, "ENVIRONMENT")
	setEnvForTest(t,
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "false",
	)
	err := AssertProductionConfig()
	if err == nil {
		t.Fatal("CRITICAL: unset ENVIRONMENT with ALLOW_CLIENT_TENANT_HEADER_FALLBACK=true must be rejected (fail-closed), got nil")
	}
}

func TestAssertProductionConfig_unsetEnvironment_devKey(t *testing.T) {
	unsetForTest(t, "ENVIRONMENT")
	setEnvForTest(t,
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "true",
	)
	err := AssertProductionConfig()
	if err == nil {
		t.Fatal("CRITICAL: unset ENVIRONMENT with API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK=true must be rejected (fail-closed), got nil")
	}
}

func TestAssertProductionConfig_unsetEnvironment_bothFlagsOff(t *testing.T) {
	unsetForTest(t, "ENVIRONMENT")
	setEnvForTest(t,
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "false",
	)
	// Both flags off — even unset ENVIRONMENT is safe, no unsafe flags active.
	if err := AssertProductionConfig(); err != nil {
		t.Fatalf("expected nil when both flags are false (even with unset ENVIRONMENT), got: %v", err)
	}
}

// ── Typo / abbreviated ENVIRONMENT ──────────────────────────────────────────

func TestAssertProductionConfig_typoEnvironment(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "prod", // common abbreviation, NOT in safe list
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true",
	)
	err := AssertProductionConfig()
	if err == nil {
		t.Fatal(`ENVIRONMENT="prod" is not in the safe list and must be treated as production, got nil`)
	}
}

// ── Safe environments — flags permitted ───────────────────────────────────────

func TestAssertProductionConfig_developmentPermissive(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "development",
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "true",
	)
	if err := AssertProductionConfig(); err != nil {
		t.Fatalf("expected nil for development environment with dev flags, got: %v", err)
	}
}

func TestAssertProductionConfig_localPermissive(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "local",
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "true",
	)
	if err := AssertProductionConfig(); err != nil {
		t.Fatalf("expected nil for local environment with dev flags, got: %v", err)
	}
}

func TestAssertProductionConfig_testPermissive(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "test",
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true",
		"API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "true",
	)
	if err := AssertProductionConfig(); err != nil {
		t.Fatalf("expected nil for test environment with dev flags, got: %v", err)
	}
}

// Case-insensitive: "Development" must also be safe.
func TestAssertProductionConfig_caseInsensitive(t *testing.T) {
	setEnvForTest(t,
		"ENVIRONMENT", "Development",
		"ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true",
	)
	if err := AssertProductionConfig(); err != nil {
		t.Fatalf(`expected nil for ENVIRONMENT="Development" (case-insensitive match), got: %v`, err)
	}
}
