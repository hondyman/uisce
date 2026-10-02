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

	// DataFusion configuration (cold tier / embedded Iceberg execution)
	DataFusion *DataFusionConfig `yaml:"datafusion" json:"datafusion"`

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

// DataFusionConfig configures Apache DataFusion (cold tier / Iceberg query engine)
type DataFusionConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"` // Default: 8555
	Endpoint string        `yaml:"endpoint" json:"endpoint"` // e.g. "http://localhost:8555"
	Catalog  string        `yaml:"catalog" json:"catalog"`   // e.g., "iceberg"
	Schema   string        `yaml:"schema" json:"schema"`     // e.g., "wealth"
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
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

	// DataFusion configuration
	if endpoint := os.Getenv("DATAFUSION_ENDPOINT"); endpoint != "" {
		cfg.DataFusion = &DataFusionConfig{
			Endpoint: endpoint,
			Catalog:  getEnvOr("DATAFUSION_CATALOG", "iceberg"),
			Schema:   getEnvOr("DATAFUSION_SCHEMA", "wealth"),
			Timeout:  parseDurationOr(os.Getenv("DATAFUSION_TIMEOUT"), 5*time.Minute),
		}
	} else if host := os.Getenv("DATAFUSION_HOST"); host != "" {
		port := parseIntOr(os.Getenv("DATAFUSION_PORT"), 8555)
		cfg.DataFusion = &DataFusionConfig{
			Host:     host,
			Port:     port,
			Endpoint: fmt.Sprintf("http://%s:%d", host, port),
			Catalog:  getEnvOr("DATAFUSION_CATALOG", "iceberg"),
			Schema:   getEnvOr("DATAFUSION_SCHEMA", "wealth"),
			Timeout:  parseDurationOr(os.Getenv("DATAFUSION_TIMEOUT"), 5*time.Minute),
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

	if c.StarRocks == nil && c.DataFusion == nil {
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

	if c.DataFusion != nil {
		if c.DataFusion.Endpoint == "" && c.DataFusion.Host == "" {
			return fmt.Errorf("datafusion endpoint or host required")
		}
		if c.DataFusion.Port == 0 && c.DataFusion.Host != "" {
			c.DataFusion.Port = 8555
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

// GetEndpoint returns the Apache DataFusion connection endpoint
func (c *DataFusionConfig) GetEndpoint() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
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
