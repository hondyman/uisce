package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// GatewayConfig is the API gateway's runtime configuration, read from the
// environment. It is deliberately separate from Config (the GraphQL/YAML
// gateway settings in config.go).
//
// The variable names and defaults are the ones main.go read directly before
// 48d8e904d introduced this type (PORT, BACKEND_URL, AUTH_SERVICE_URL and
// REVOCATION_REDIS_ADDR); that commit referenced GatewayConfig/LoadGatewayConfig
// but the definition was never committed, so the gateway did not compile.
type GatewayConfig struct {
	// Env is the deployment environment, only reported in the startup log.
	Env string
	// LogLevel is only reported in the startup log; the logger itself is
	// initialized by logging.InitGlobalLogger before this is loaded.
	LogLevel string
	// Port is the HTTP listen port WITHOUT a leading colon (main.go adds it).
	Port string
	// BackendURL is the backend service requests are proxied to.
	BackendURL string
	// AuthServiceURL is the auth service used for token operations.
	AuthServiceURL string
	// RevocationRedisAddr, when set, makes token revocation Redis-backed
	// (shared across gateway replicas). Empty means the in-memory store.
	RevocationRedisAddr string
}

// LoadGatewayConfig reads GatewayConfig from the environment and validates it.
func LoadGatewayConfig() (*GatewayConfig, error) {
	cfg := &GatewayConfig{
		Env:                 firstEnv("development", "ENV", "ENVIRONMENT"),
		LogLevel:            firstEnv("info", "LOG_LEVEL"),
		Port:                strings.TrimPrefix(firstEnv("8001", "PORT"), ":"),
		BackendURL:          firstEnv("http://localhost:8080", "BACKEND_URL"),
		AuthServiceURL:      firstEnv("http://auth-service:8001", "AUTH_SERVICE_URL"),
		RevocationRedisAddr: firstEnv("", "REVOCATION_REDIS_ADDR"),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate rejects values the gateway cannot run with, so a typo fails at
// startup instead of surfacing as a confusing proxy error later.
func (c *GatewayConfig) Validate() error {
	if n, err := strconv.Atoi(c.Port); err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("invalid PORT %q: must be a number from 1 to 65535", c.Port)
	}
	for name, v := range map[string]string{"BACKEND_URL": c.BackendURL, "AUTH_SERVICE_URL": c.AuthServiceURL} {
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("invalid %s %q: must be an absolute http(s) URL", name, v)
		}
	}
	return nil
}

// firstEnv returns the first non-empty environment variable among names, or def.
func firstEnv(def string, names ...string) string {
	for _, n := range names {
		if v := strings.TrimSpace(os.Getenv(n)); v != "" {
			return v
		}
	}
	return def
}
