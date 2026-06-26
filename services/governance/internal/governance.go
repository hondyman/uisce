package config

import (
	"fmt"
	"time"
)

// GovernanceConfig holds configuration for the governance system
type GovernanceConfig struct {
	// Database settings
	DatabaseURL      string        `yaml:"database_url" env:"DATABASE_URL"`
	DBMaxConnections int           `yaml:"db_max_connections" default:"10"`
	DBTimeout        time.Duration `yaml:"db_timeout" default:"10s"`

	// Cache settings
	CacheEnabled bool          `yaml:"cache_enabled" default:"true"`
	CacheTTL     time.Duration `yaml:"cache_ttl" default:"5m"`
	RedisURL     string        `yaml:"redis_url" env:"REDIS_URL"`

	// Policy settings
	DefaultDenyAll    bool     `yaml:"default_deny_all" default:"false"`
	AllowedTenants    []string `yaml:"allowed_tenants"`
	RestrictedActions []string `yaml:"restricted_actions"`

	// Performance settings
	MaxConcurrentEvaluations int           `yaml:"max_concurrent_evaluations" default:"100"`
	EvaluationTimeout        time.Duration `yaml:"evaluation_timeout" default:"30s"`

	// Security settings
	EnableAuditLog     bool `yaml:"enable_audit_log" default:"true"`
	EnableRequestID    bool `yaml:"enable_request_id" default:"true"`
	EnableRateLimiting bool `yaml:"enable_rate_limiting" default:"false"`
	RateLimitPerMinute int  `yaml:"rate_limit_per_minute" default:"1000"`

	// Feature flags
	EnableSemanticPlanner bool `yaml:"enable_semantic_planner" default:"true"`
	EnablePolicyCaching   bool `yaml:"enable_policy_caching" default:"true"`

	// Monitoring settings
	MetricsEnabled bool   `yaml:"metrics_enabled" default:"true"`
	TracingEnabled bool   `yaml:"tracing_enabled" default:"false"`
	JaegerEndpoint string `yaml:"jaeger_endpoint" env:"JAEGER_ENDPOINT"`

	// Server settings
	HTTPPort    string `yaml:"http_port" default:"8080"`
	HTTPSPort   string `yaml:"https_port" default:"8443"`
	EnableHTTPS bool   `yaml:"enable_https" default:"false"`
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file"`

	// CORS settings
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"`

	// External integrations
	EnableWebhookNotifications bool     `yaml:"enable_webhook_notifications" default:"false"`
	WebhookEndpoints           []string `yaml:"webhook_endpoints"`

	// Query rewrite settings
	EnableQueryRewrite    bool          `yaml:"enable_query_rewrite" default:"true"`
	RewriteAuditEnabled   bool          `yaml:"rewrite_audit_enabled" default:"true"`
	MaxRewriteSuggestions int           `yaml:"max_rewrite_suggestions" default:"5"`
	RewriteTimeout        time.Duration `yaml:"rewrite_timeout" default:"10s"`
}

// LoadDefaultConfig returns a configuration with sensible defaults
func LoadDefaultConfig() *GovernanceConfig {
	return &GovernanceConfig{
		DatabaseURL:                "postgres://100.84.126.19:5432/governance?sslmode=disable",
		DBMaxConnections:           10,
		DBTimeout:                  10 * time.Second,
		CacheEnabled:               true,
		CacheTTL:                   5 * time.Minute,
		RedisURL:                   "redis://100.84.126.19:6379",
		DefaultDenyAll:             false,
		MaxConcurrentEvaluations:   100,
		EvaluationTimeout:          30 * time.Second,
		EnableAuditLog:             true,
		EnableRequestID:            true,
		EnableRateLimiting:         false,
		RateLimitPerMinute:         1000,
		EnableSemanticPlanner:      true,
		EnablePolicyCaching:        true,
		MetricsEnabled:             true,
		TracingEnabled:             false,
		JaegerEndpoint:             "http://localhost:14268/api/traces",
		HTTPPort:                   "8080",
		HTTPSPort:                  "8443",
		EnableHTTPS:                false,
		CORSAllowedOrigins:         []string{"*"},
		EnableWebhookNotifications: false,
		EnableQueryRewrite:         true,
		RewriteAuditEnabled:        true,
		MaxRewriteSuggestions:      5,
		RewriteTimeout:             10 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *GovernanceConfig) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}
	if c.CacheTTL < 0 {
		return fmt.Errorf("cache_ttl must be positive")
	}
	if c.MaxConcurrentEvaluations < 1 {
		return fmt.Errorf("max_concurrent_evaluations must be at least 1")
	}
	if c.EvaluationTimeout < 0 {
		return fmt.Errorf("evaluation_timeout must be positive")
	}
	if c.RateLimitPerMinute < 0 {
		return fmt.Errorf("rate_limit_per_minute must be non-negative")
	}
	if c.DBMaxConnections < 1 {
		return fmt.Errorf("db_max_connections must be at least 1")
	}
	if c.DBTimeout < 0 {
		return fmt.Errorf("db_timeout must be positive")
	}
	if c.HTTPPort == "" {
		return fmt.Errorf("http_port is required")
	}
	if c.EnableHTTPS {
		if c.TLSCertFile == "" || c.TLSKeyFile == "" {
			return fmt.Errorf("tls_cert_file and tls_key_file are required when enable_https is true")
		}
	}
	return nil
}
