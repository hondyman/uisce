package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// CalcEngineConfig configures the multi-source calculation engine
type CalcEngineConfig struct {
	// StarRocks configuration (hot tier for real-time analytics)
	StarRocks *StarRocksConfig `yaml:"starrocks" json:"starrocks"`

	// Trino configuration (cold tier for historical data)
	Trino *TrinoConfig `yaml:"trino" json:"trino"`

	// CubeJS configuration (semantic layer bridge)
	Cube *CubeConfig `yaml:"cube" json:"cube"`

	// Hot/Cold tier boundary in days (default: 90)
	HotColdBoundaryDays int `yaml:"hot_cold_boundary_days" json:"hot_cold_boundary_days"`

	// Whether to enable the multi-source engine (default: false, uses PostgreSQL)
	EnableMultiSource bool `yaml:"enable_multi_source" json:"enable_multi_source"`

	// Whether to force StarRocks for all queries regardless of date
	ForceHotTier bool `yaml:"force_hot_tier" json:"force_hot_tier"`
}

// StarRocksConfig configures StarRocks (hot tier)
type StarRocksConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"` // Default: 9030
	User     string        `yaml:"user" json:"user"`
	Password string        `yaml:"password" json:"password"`
	Database string        `yaml:"database" json:"database"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	MaxConns int           `yaml:"max_conns" json:"max_conns"`
}

// TrinoConfig configures Trino (cold tier)
type TrinoConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"` // Default: 8090
	User     string        `yaml:"user" json:"user"`
	Password string        `yaml:"password" json:"password"`
	Catalog  string        `yaml:"catalog" json:"catalog"` // e.g., "iceberg"
	Schema   string        `yaml:"schema" json:"schema"`   // e.g., "wealth"
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
}

// CubeConfig configures the Cube.js semantic layer bridge
type CubeConfig struct {
	APIURL     string `yaml:"api_url" json:"api_url"`         // e.g., "http://localhost:4000"
	Enabled    bool   `yaml:"enabled" json:"enabled"`         // Enable Cube bridge
	JWTSecret  string `yaml:"jwt_secret" json:"jwt_secret"`   // For authenticated requests
	RollupOnly bool   `yaml:"rollup_only" json:"rollup_only"` // Force CUBEJS_ROLLUP_ONLY
}

// NewCalcEngineConfigFromEnv creates CalcEngineConfig from environment variables
func NewCalcEngineConfigFromEnv() *CalcEngineConfig {
	cfg := &CalcEngineConfig{
		HotColdBoundaryDays: 90,
		EnableMultiSource:   parseBool(os.Getenv("CALC_ENGINE_MULTI_SOURCE")),
		ForceHotTier:        parseBool(os.Getenv("CALC_ENGINE_FORCE_HOT")),
	}

	// StarRocks configuration
	if host := os.Getenv("STARROCKS_HOST"); host != "" {
		cfg.StarRocks = &StarRocksConfig{
			Host:     host,
			Port:     parseIntOr(os.Getenv("STARROCKS_PORT"), 9030),
			User:     getEnvOr("STARROCKS_USER", "root"),
			Password: os.Getenv("STARROCKS_PASSWORD"),
			Database: getEnvOr("STARROCKS_DATABASE", "semantic_layer"),
			Timeout:  parseDurationOr(os.Getenv("STARROCKS_TIMEOUT"), 30*time.Second),
			MaxConns: parseIntOr(os.Getenv("STARROCKS_MAX_CONNS"), 20),
		}
	}

	// Trino configuration
	if host := os.Getenv("TRINO_HOST"); host != "" {
		cfg.Trino = &TrinoConfig{
			Host:     host,
			Port:     parseIntOr(os.Getenv("TRINO_PORT"), 8090),
			User:     getEnvOr("TRINO_USER", "admin"),
			Password: os.Getenv("TRINO_PASSWORD"),
			Catalog:  getEnvOr("TRINO_CATALOG", "iceberg"),
			Schema:   getEnvOr("TRINO_SCHEMA", "wealth"),
			Timeout:  parseDurationOr(os.Getenv("TRINO_TIMEOUT"), 5*time.Minute),
		}
	}

	// Cube configuration
	if apiURL := os.Getenv("CUBEJS_API_URL"); apiURL != "" {
		cfg.Cube = &CubeConfig{
			APIURL:     apiURL,
			Enabled:    parseBool(getEnvOr("CUBEJS_ENABLED", "true")),
			JWTSecret:  os.Getenv("CUBEJS_JWT_SECRET"),
			RollupOnly: parseBool(os.Getenv("CUBEJS_ROLLUP_ONLY")),
		}
	}

	// Hot/Cold boundary
	if days := os.Getenv("CALC_ENGINE_HOT_COLD_DAYS"); days != "" {
		cfg.HotColdBoundaryDays = parseIntOr(days, 90)
	}

	return cfg
}

// Validate validates the CalcEngineConfig
func (c *CalcEngineConfig) Validate() error {
	if !c.EnableMultiSource {
		return nil // Nothing to validate if multi-source is disabled
	}

	if c.StarRocks == nil && c.Trino == nil {
		return fmt.Errorf("multi-source engine enabled but no data sources configured")
	}

	if c.StarRocks != nil {
		if c.StarRocks.Host == "" {
			return fmt.Errorf("starrocks host required")
		}
		if c.StarRocks.Port == 0 {
			c.StarRocks.Port = 9030
		}
	}

	if c.Trino != nil {
		if c.Trino.Host == "" {
			return fmt.Errorf("trino host required")
		}
		if c.Trino.Port == 0 {
			c.Trino.Port = 8090
		}
	}

	if c.HotColdBoundaryDays <= 0 {
		c.HotColdBoundaryDays = 90
	}

	return nil
}

// GetDSN returns a MySQL-compatible DSN for StarRocks
func (c *StarRocksConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s&parseTime=true",
		c.User, c.Password, c.Host, c.Port, c.Database,
		c.Timeout.String())
}

// GetURI returns a Trino connection URI
func (c *TrinoConfig) GetURI() string {
	if c.Password != "" {
		return fmt.Sprintf("trino://%s:%s@%s:%d/%s/%s",
			c.User, c.Password, c.Host, c.Port, c.Catalog, c.Schema)
	}
	return fmt.Sprintf("trino://%s@%s:%d/%s/%s",
		c.User, c.Host, c.Port, c.Catalog, c.Schema)
}

// Helper functions for parsing environment variables

func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func parseDurationOr(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func getEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
