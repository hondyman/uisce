package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds configuration settings loaded from config.yaml.
type Config struct {
	// Core settings
	YAMLDir string `yaml:"yaml_dir" json:"yaml_dir"`
	Driver  string `yaml:"driver" json:"driver"` // "snowflake", "postgres", or "mssql"
	DSN     string `yaml:"dsn" json:"dsn"`
	Port    string `yaml:"port" json:"port"`       // HTTP port, default ":8080"
	PGPort  string `yaml:"pg_port" json:"pg_port"` // PostgreSQL wire port, default ":5432"

	// Aggregates database (e.g., northwinds) for multi-datasource support
	AggregatesDSN string `yaml:"aggregates_dsn" json:"aggregates_dsn"`

	// Enhanced settings
	RedisAddr      string `yaml:"redis_addr" json:"redis_addr"` // Redis address for caching
	JWTSecret      string `yaml:"jwt_secret" json:"jwt_secret"` // Secret for JWT signing/verification
	GraphQLURL     string `yaml:"graphql_url" json:"graphql_url"`
	Environment    string `yaml:"environment" json:"environment"`         // dev, staging, prod
	LogLevel       string `yaml:"log_level" json:"log_level"`             // debug, info, warn, error
	EnableMetrics  bool   `yaml:"enable_metrics" json:"enable_metrics"`   // Enable metrics collection
	EnableCaching  bool   `yaml:"enable_caching" json:"enable_caching"`   // Enable advanced caching
	EnableSecurity bool   `yaml:"enable_security" json:"enable_security"` // Enable security features

	// Database settings
	DBMaxOpenConns int           `yaml:"db_max_open_conns" json:"db_max_open_conns"`
	DBMaxIdleConns int           `yaml:"db_max_idle_conns" json:"db_max_idle_conns"`
	DBMaxLifetime  time.Duration `yaml:"db_max_lifetime" json:"db_max_lifetime"`

	// Cache settings
	CacheNumShards       int           `yaml:"cache_num_shards" json:"cache_num_shards"`
	CacheMaxSizePerShard int64         `yaml:"cache_max_size_per_shard" json:"cache_max_size_per_shard"`
	CacheDefaultTTL      time.Duration `yaml:"cache_default_ttl" json:"cache_default_ttl"`

	// Security settings
	SecurityRateLimitEnabled  bool          `yaml:"security_rate_limit_enabled" json:"security_rate_limit_enabled"`
	SecurityRateLimitRequests int64         `yaml:"security_rate_limit_requests" json:"security_rate_limit_requests"`
	SecurityRateLimitWindow   time.Duration `yaml:"security_rate_limit_window" json:"security_rate_limit_window"`
}

// LoadConfig reads the configuration from the given file path.
func LoadConfig(path string) (*Config, error) {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()

	// Start with defaults
	cfg := getDefaults()

	// Load from file if provided
	if path != "" {
		if err := loadFromFile(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	loadFromEnv(cfg)

	// Normalize DSN formats for supported drivers (e.g., postgres vs postgresql URI prefixes)
	if strings.EqualFold(cfg.Driver, "postgres") {
		cfg.DSN = normalizePostgresDSN(cfg.DSN)
	}

	// Apply environment-specific defaults
	applyEnvironmentDefaults(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// getDefaults returns a configuration with sensible defaults
func getDefaults() *Config {
	return &Config{
		YAMLDir:                   "./config",
		Driver:                    "postgres",
		Port:                      ":8080",
		PGPort:                    ":5432",
		Environment:               "development",
		LogLevel:                  "info",
		EnableMetrics:             true,
		EnableCaching:             true,
		EnableSecurity:            true,
		DBMaxOpenConns:            25,
		DBMaxIdleConns:            5,
		DBMaxLifetime:             5 * time.Minute,
		CacheNumShards:            16,
		CacheMaxSizePerShard:      10000,
		CacheDefaultTTL:           30 * time.Minute,
		SecurityRateLimitEnabled:  true,
		SecurityRateLimitRequests: 100,
		SecurityRateLimitWindow:   1 * time.Minute,
		JWTSecret:                 "your-super-secret-jwt-key-change-in-production",
	}
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(path string, cfg *Config) error {
	configData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	err = yaml.Unmarshal(configData, cfg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %w", path, err)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(cfg *Config) {
	// Core settings
	if val := os.Getenv("YAML_DIR"); val != "" {
		cfg.YAMLDir = val
	}
	if val := os.Getenv("DRIVER"); val != "" {
		cfg.Driver = val
	}
	if val := os.Getenv("DSN"); val != "" {
		cfg.DSN = val
	} else if val := os.Getenv("DATABASE_URL"); val != "" {
		cfg.DSN = val
	}
	if val := os.Getenv("PORT"); val != "" {
		cfg.Port = val
	}
	if val := os.Getenv("PG_PORT"); val != "" {
		cfg.PGPort = val
	}

	// Enhanced settings
	if val := os.Getenv("REDIS_ADDR"); val != "" {
		cfg.RedisAddr = val
	}
	if val := os.Getenv("JWT_SECRET"); val != "" {
		cfg.JWTSecret = val
	}
	if val := os.Getenv("GRAPHQL_URL"); val != "" {
		cfg.GraphQLURL = val
	}
	if val := os.Getenv("ENVIRONMENT"); val != "" {
		cfg.Environment = val
	}
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}

	// Feature flags
	if val := os.Getenv("ENABLE_METRICS"); val != "" {
		cfg.EnableMetrics = parseBool(val)
	}
	if val := os.Getenv("ENABLE_CACHING"); val != "" {
		cfg.EnableCaching = parseBool(val)
	}
	if val := os.Getenv("ENABLE_SECURITY"); val != "" {
		cfg.EnableSecurity = parseBool(val)
	}

	// Database settings
	if val := os.Getenv("DB_MAX_OPEN_CONNS"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			cfg.DBMaxOpenConns = v
		}
	}
	if val := os.Getenv("DB_MAX_IDLE_CONNS"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			cfg.DBMaxIdleConns = v
		}
	}

	// Cache settings
	if val := os.Getenv("CACHE_NUM_SHARDS"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			cfg.CacheNumShards = v
		}
	}
	if val := os.Getenv("CACHE_MAX_SIZE_PER_SHARD"); val != "" {
		if v, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.CacheMaxSizePerShard = v
		}
	}

	// Security settings
	if val := os.Getenv("RATE_LIMIT_ENABLED"); val != "" {
		cfg.SecurityRateLimitEnabled = parseBool(val)
	}
	if val := os.Getenv("RATE_LIMIT_REQUESTS"); val != "" {
		if v, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.SecurityRateLimitRequests = v
		}
	}
}

func normalizePostgresDSN(dsn string) string {
	if dsn == "" {
		return dsn
	}

	const (
		postgresPrefix   = "postgres://"
		postgresqlPrefix = "postgresql://"
	)

	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, postgresqlPrefix):
		return postgresPrefix + dsn[len(postgresqlPrefix):]
	case strings.HasPrefix(lower, postgresPrefix) && !strings.HasPrefix(dsn, postgresPrefix):
		// Normalize the prefix casing to lower-case postgres://
		return postgresPrefix + dsn[len(postgresPrefix):]
	default:
		return dsn
	}
}

// applyEnvironmentDefaults applies environment-specific defaults
func applyEnvironmentDefaults(cfg *Config) {
	switch strings.ToLower(cfg.Environment) {
	case "production", "prod":
		applyProductionDefaults(cfg)
	case "staging":
		applyStagingDefaults(cfg)
	case "development", "dev":
		applyDevelopmentDefaults(cfg)
	}
}

// applyProductionDefaults applies production-specific settings
func applyProductionDefaults(cfg *Config) {
	cfg.LogLevel = "warn"
	cfg.DBMaxOpenConns = 50
	cfg.DBMaxIdleConns = 10
	cfg.CacheMaxSizePerShard = 50000
	cfg.SecurityRateLimitRequests = 1000
	if cfg.JWTSecret == "your-super-secret-jwt-key-change-in-production" {
		// Force production to set a proper secret
		cfg.JWTSecret = ""
	}
}

// applyStagingDefaults applies staging-specific settings
func applyStagingDefaults(cfg *Config) {
	cfg.LogLevel = "info"
	cfg.DBMaxOpenConns = 30
	cfg.DBMaxIdleConns = 8
	cfg.CacheMaxSizePerShard = 25000
	cfg.SecurityRateLimitRequests = 500
}

// applyDevelopmentDefaults applies development-specific settings
func applyDevelopmentDefaults(cfg *Config) {
	cfg.LogLevel = "debug"
	cfg.DBMaxOpenConns = 10
	cfg.DBMaxIdleConns = 2
	cfg.CacheMaxSizePerShard = 5000
	cfg.SecurityRateLimitEnabled = false
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.YAMLDir == "" {
		return fmt.Errorf("yaml_dir is required")
	}
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}
	if c.DSN == "" {
		return fmt.Errorf("dsn is required")
	}

	// Validate DSN format for PostgreSQL
	if c.Driver == "postgres" {
		hasDBName := false
		// Check for space-separated format (e.g., "host=localhost dbname=alpha")
		if strings.Contains(c.DSN, "dbname=") {
			hasDBName = true
		} else if strings.HasPrefix(strings.ToLower(c.DSN), "postgres://") {
			// Handle URI format (e.g., "postgres://user:password@host:port/dbname?params")
			trimmed := c.DSN[len("postgres://"):]
			parts := strings.SplitN(trimmed, "/", 2)
			if len(parts) == 2 {
				dbPath := strings.SplitN(parts[1], "?", 2)[0]
				if dbPath != "" {
					hasDBName = true
				}
			}
		}
		if !hasDBName {
			return fmt.Errorf("DSN must include dbname for postgres driver")
		}
	}

	if c.EnableSecurity && c.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required when security is enabled")
	}

	if c.CacheNumShards < 1 {
		return fmt.Errorf("cache_num_shards must be at least 1")
	}

	return nil
}

// PrintConfig prints the current configuration (with sensitive data masked)
func (c *Config) PrintConfig() {
	configCopy := *c
	if configCopy.JWTSecret != "" {
		configCopy.JWTSecret = "***masked***"
	}
	if strings.Contains(configCopy.DSN, "password=") {
		re := regexp.MustCompile(`password=[^\s?&]+`)
		configCopy.DSN = re.ReplaceAllString(configCopy.DSN, "password=***masked***")
	}

	fmt.Println("=== Configuration ===")
	fmt.Printf("Environment: %s\n", configCopy.Environment)
	fmt.Printf("Driver: %s\n", configCopy.Driver)
	fmt.Printf("Port: %s\n", configCopy.Port)
	fmt.Printf("YAML Dir: %s\n", configCopy.YAMLDir)
	fmt.Printf("Log Level: %s\n", configCopy.LogLevel)
	fmt.Printf("Enable Metrics: %t\n", configCopy.EnableMetrics)
	fmt.Printf("Enable Caching: %t\n", configCopy.EnableCaching)
	fmt.Printf("Enable Security: %t\n", configCopy.EnableSecurity)
	fmt.Printf("Database Max Open Conns: %d\n", configCopy.DBMaxOpenConns)
	fmt.Printf("Cache Shards: %d\n", configCopy.CacheNumShards)
	fmt.Printf("Rate Limiting: %t (%d requests per %v)\n",
		configCopy.SecurityRateLimitEnabled,
		configCopy.SecurityRateLimitRequests,
		configCopy.SecurityRateLimitWindow)

	// Log DSN with password redacted
	re := regexp.MustCompile(`password=[^\s?&]+|:[^@]+@`)
	redactedDSN := re.ReplaceAllString(configCopy.DSN, ":********@")
	fmt.Printf("DSN: %s\n", redactedDSN)
}

// IsProduction returns true if running in production
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production" || strings.ToLower(c.Environment) == "prod"
}

// IsDevelopment returns true if running in development
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development" || strings.ToLower(c.Environment) == "dev"
}

// parseBool parses a string to boolean
func parseBool(s string) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return false
	}
}
