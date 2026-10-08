package region

import (
	"path/filepath"
	"strings"
	"testing"
)

const validRegionsYAML = `
regions:
  us-east-1:
    postgres_host: pg-use1.internal
    postgres_port: 5432
    lakekeeper_url: https://lakekeeper.us-east-1.internal:8181
    starrocks_fe_host: sr-fe.us-east-1.internal
    debezium_connect_url: http://connect.us-east-1.internal:8083
`

func TestParseRegistryAcceptsValidConfig(t *testing.T) {
	reg, err := ParseRegistry([]byte(validRegionsYAML))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	ep, err := reg.Resolve("us-east-1")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if ep.PostgresHost != "pg-use1.internal" || ep.PostgresPort != 5432 {
		t.Fatalf("unexpected endpoints: %+v", ep)
	}
	if got := reg.Codes(); len(got) != 1 || got[0] != "us-east-1" {
		t.Fatalf("Codes = %v", got)
	}
}

func TestResolveRejectsUnknownRegionFailClosed(t *testing.T) {
	reg, err := ParseRegistry([]byte(validRegionsYAML))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	for _, code := range []string{"", "eu-west-1", "US-EAST-1", "us-east-1 ", "us-east-1;drop"} {
		if _, err := reg.Resolve(code); err == nil {
			t.Errorf("Resolve(%q) succeeded, want unknown-region error", code)
		}
	}
}

func TestParseRegistryRejectsInvalidConfig(t *testing.T) {
	// Each case replaces the us-east-1 block (or adds a bad one) and must fail.
	good := func(region string) string {
		return `
regions:
  ` + region + `:
    postgres_host: pg.internal
    postgres_port: 5432
    lakekeeper_url: https://lk.internal:8181
    starrocks_fe_host: sr.internal
    debezium_connect_url: http://connect.internal:8083
`
	}
	tests := []struct {
		name string
		yaml string
	}{
		{"empty document", ""},
		{"no regions key", "other: 1\n"},
		{"no regions", "regions: {}\n"},
		{"region code with uppercase", good("US-EAST-1")},
		{"region code with semicolon", good("us-east-1;drop")},
		{"region code with space", good("us east 1")},
		{"region code single segment", good("useast1")},
		{"region code unicode", good("us-éast-1")},
		{"region code overlong", good("us-" + strings.Repeat("a", 80) + "-1")},
		{"missing postgres_host", strings.Replace(good("us-east-1"), "postgres_host: pg.internal", "postgres_host: \"\"", 1)},
		{"postgres_port zero", strings.Replace(good("us-east-1"), "postgres_port: 5432", "postgres_port: 0", 1)},
		{"postgres_port too large", strings.Replace(good("us-east-1"), "postgres_port: 5432", "postgres_port: 70000", 1)},
		{"lakekeeper_url empty", strings.Replace(good("us-east-1"), "https://lk.internal:8181", "\"\"", 1)},
		{"lakekeeper_url wrong scheme", strings.Replace(good("us-east-1"), "https://lk.internal:8181", "ftp://lk.internal", 1)},
		{"lakekeeper_url no host", strings.Replace(good("us-east-1"), "https://lk.internal:8181", "https://", 1)},
		{"debezium_connect_url javascript scheme", strings.Replace(good("us-east-1"), "http://connect.internal:8083", "javascript:alert(1)", 1)},
		{"starrocks_fe_host empty", strings.Replace(good("us-east-1"), "starrocks_fe_host: sr.internal", "starrocks_fe_host: \"\"", 1)},
		{"malformed yaml", "regions: [\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseRegistry([]byte(tc.yaml)); err == nil {
				t.Fatalf("ParseRegistry accepted invalid config:\n%s", tc.yaml)
			}
		})
	}
}

func TestLoadRegistryMissingFileFails(t *testing.T) {
	if _, err := LoadRegistry(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Fatal("LoadRegistry accepted a missing file")
	}
}
