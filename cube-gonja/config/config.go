package config

import (
	"os"
	"strings"
)

type Config struct {
	TemplateDir       string
	OutputDir         string
	CubeReloadURL     string
	AllowedDataSource map[string]struct{}
	// Optional: External validator command, e.g., "npx cubejs-cli validate"
	ExternalValidator []string
	// Multi-tenant configuration
	EnableMultiTenant bool
	TenantBaseDir     string
	DefaultTenant     string
	// Git integration
	GitEnabled   bool
	GitRemoteURL string
	GitBranch    string
	// Authentication
	RequireAuth bool
	APIKeys     map[string]string // tenant -> api_key
	// Database configuration for catalog updates
	DatabaseDSN      string
	DatabaseHost     string
	DatabasePort     string
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSSLMode  string
}

func FromEnv() Config {
	allowed := map[string]struct{}{}
	// Comma-separated list, e.g., "default,sales_db,analytics_db"
	list := os.Getenv("ALLOWED_DATA_SOURCES")
	if list == "" {
		list = "default" // sane default; override in prod
	}
	for _, s := range splitCSV(list) {
		allowed[s] = struct{}{}
	}

	// Parse API keys (format: tenant1:key1,tenant2:key2)
	apiKeys := make(map[string]string)
	if keys := os.Getenv("TENANT_API_KEYS"); keys != "" {
		for _, pair := range splitCSV(keys) {
			if parts := strings.SplitN(pair, ":", 2); len(parts) == 2 {
				apiKeys[parts[0]] = parts[1]
			}
		}
	}

	return Config{
		TemplateDir:       getenv("TEMPLATE_DIR", "templates"),
		OutputDir:         getenv("OUTPUT_DIR", "model-out"),
		CubeReloadURL:     getenv("CUBE_RELOAD_URL", "http://cube:4000/cubejs-api/v1/reload"),
		AllowedDataSource: allowed,
		ExternalValidator: splitCSV(os.Getenv("EXTERNAL_VALIDATOR_CMD")), // optional
		// Multi-tenant settings
		EnableMultiTenant: getenv("ENABLE_MULTI_TENANT", "false") == "true",
		TenantBaseDir:     getenv("TENANT_BASE_DIR", "tenants"),
		DefaultTenant:     getenv("DEFAULT_TENANT", "default"),
		// Git settings
		GitEnabled:   getenv("GIT_ENABLED", "false") == "true",
		GitRemoteURL: os.Getenv("GIT_REMOTE_URL"),
		GitBranch:    getenv("GIT_BRANCH", "main"),
		// Auth settings
		RequireAuth: getenv("REQUIRE_AUTH", "false") == "true",
		APIKeys:     apiKeys,
		// Database settings
		DatabaseDSN:      os.Getenv("DATABASE_DSN"),
		DatabaseHost:     getenv("DATABASE_HOST", "localhost"),
		DatabasePort:     getenv("DATABASE_PORT", "5432"),
		DatabaseName:     getenv("DATABASE_NAME", "semlayer"),
		DatabaseUser:     getenv("DATABASE_USER", "postgres"),
		DatabasePassword: os.Getenv("DATABASE_PASSWORD"),
		DatabaseSSLMode:  getenv("DATABASE_SSL_MODE", "disable"),
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, trim(cur))
				cur = ""
			}
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, trim(cur))
	}
	return out
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
