package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	// Server
	ServerPort  string
	Environment string
	LogLevel    string
	CORSOrigins []string

	// Database
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	// Hasura
	HasuraEndpoint    string
	HasuraAdminSecret string
	HasuraJWTSecret   string

	// Temporal
	TemporalHostPort  string
	TemporalNamespace string
	TemporalTaskQueue string

	// Redpanda
	RedpandaBrokers       []string
	RedpandaCDCTopic      string
	RedpandaConsumerGroup string

	// Redis
	RedisURL      string
	RedisCacheTTL time.Duration
	RedisPrefix   string
	CacheEnabled  bool

	// Global Distribution
	WorkerRegions       []string
	DefaultRegion       string
	DataResidencyPolicy string

	// Job Priority
	PriorityQueues       []string
	DefaultPriority      int
	CriticalQueueWorkers int
	StandardQueueWorkers int
	BulkQueueWorkers     int

	// AI
	AIKey     string
	AIURL     string
	AIModel   string
	AITimeout time.Duration

	// Security
	JWTSigningKey string
	EncryptionKey string

	// Monitoring
	PrometheusPort       string
	OTELExporterEndpoint string
	MetricsEnabled       bool

	// Legacy (for compatibility)
	CacheTTLMinutes int
	EnableCDC       bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		// Server
		ServerPort:  getEnv("SERVER_PORT", "8081"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		CORSOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),

		// Database
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       getEnv("POSTGRES_DB", "calendar_db"),
		PostgresSSLMode:  getEnv("POSTGRES_SSL_MODE", "disable"),

		// Hasura
		HasuraEndpoint:    getEnv("HASURA_ENDPOINT", "http://localhost:8080/v1/graphql"),
		HasuraAdminSecret: getEnv("HASURA_ADMIN_SECRET", "myadminsecret"),
		HasuraJWTSecret:   getEnv("HASURA_JWT_SECRET", "your-secret-key"),

		// Temporal
		TemporalHostPort:  getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		TemporalTaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "calendar-task-queue"),

		// Redpanda
		RedpandaBrokers:       getEnvSlice("REDPANDA_BROKERS", []string{"localhost:9092"}),
		RedpandaCDCTopic:      getEnv("REDPANDA_CDC_TOPIC", "cdc_holidays"),
		RedpandaConsumerGroup: getEnv("REDPANDA_CONSUMER_GROUP", "calendar-cdc-group"),

		// Redis
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisCacheTTL: getEnvDuration("REDIS_CACHE_TTL", 3600),
		RedisPrefix:   getEnv("REDIS_PREFIX", "calendar"),
		CacheEnabled:  getEnvBool("CACHE_ENABLED", true),

		// Global Distribution
		WorkerRegions:       getEnvSlice("WORKER_REGIONS", []string{"us-east-1"}),
		DefaultRegion:       getEnv("DEFAULT_REGION", "us-east-1"),
		DataResidencyPolicy: getEnv("DATA_RESIDENCY_POLICY", "strict"),

		// Job Priority
		PriorityQueues:       getEnvSlice("PRIORITY_QUEUES", []string{"critical", "standard", "bulk"}),
		DefaultPriority:      getEnvInt("DEFAULT_PRIORITY", 5),
		CriticalQueueWorkers: getEnvInt("CRITICAL_QUEUE_WORKERS", 3),
		StandardQueueWorkers: getEnvInt("STANDARD_QUEUE_WORKERS", 2),
		BulkQueueWorkers:     getEnvInt("BULK_QUEUE_WORKERS", 1),

		// AI
		AIKey:     getEnv("AI_API_KEY", ""),
		AIURL:     getEnv("AI_API_URL", "https://api.openai.com/v1/chat/completions"),
		AIModel:   getEnv("AI_MODEL", "gpt-4"),
		AITimeout: getEnvDuration("AI_TIMEOUT", 30),

		// Security
		JWTSigningKey: getEnv("JWT_SIGNING_KEY", "your-jwt-signing-key"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "your-32-byte-encryption-key"),

		// Monitoring
		PrometheusPort:       getEnv("PROMETHEUS_PORT", "9090"),
		OTELExporterEndpoint: getEnv("OTEL_EXPORTER_ENDPOINT", "http://localhost:4317"),
		MetricsEnabled:       getEnvBool("METRICS_ENABLED", true),

		// Legacy
		CacheTTLMinutes: 60,
		EnableCDC:       getEnvBool("ENABLE_CDC", true),
	}
}

// Load is an alias for LoadConfig (for compatibility)
func Load() *Config {
	return LoadConfig()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		return strings.Split(v, ",")
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		_, err := fmt.Sscanf(v, "%d", &i)
		if err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallbackSeconds int) time.Duration {
	if v := os.Getenv(key); v != "" {
		var i int
		_, err := fmt.Sscanf(v, "%d", &i)
		if err == nil {
			return time.Duration(i) * time.Second
		}
	}
	return time.Duration(fallbackSeconds) * time.Second
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return fallback
}
