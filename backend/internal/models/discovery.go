package models

import "time"

// FeatureCandidate represents a discovered feature that could be added to the catalog
type FeatureCandidate struct {
	ID              string    // Unique identifier
	Name            string    // Feature name (e.g., "http_request_duration_p99")
	SourceDatabase  string    // Where it came from: postgres, logs, prometheus, trino, s3, derived
	SourceSchema    string    // Schema/database name
	SourceTable     string    // Table name (if applicable)
	SourceField     string    // Original field name
	DataType        string    // float, string, integer, boolean, categorical, timestamp
	Description     string    // Human-readable description
	Completeness    float64   // 0-1: % of non-null values
	Cardinality     int64     // Number of distinct values (-1 if unknown)
	BusinessValue   float64   // 0-1: how useful for business metrics
	TechnicalScore  float64   // 0-1: technical quality/signal strength
	DiscoveredAt    time.Time // When this candidate was discovered
	Status          string    // candidate, approved, rejected, deployed
	ApprovedBy      string    // User who approved
	ApprovedAt      time.Time // When approved
	RejectionReason string    // Why was it rejected
	Notes           string    // Additional context
}

// FeatureVersion represents a version of a feature in the catalog
type FeatureVersion struct {
	ID             string // Unique version ID
	FeatureName    string // Which feature this version belongs to
	VersionNumber  int    // 1, 2, 3, ...
	ComputeLogic   string // SQL/Python code to compute this feature
	SchemaLocation string // Where the feature is stored (table.column)
	CreatedAt      time.Time
	DeployedAt     time.Time
	Status         string                 // active, deprecated, testing
	Performance    map[string]interface{} // Metrics like avg_importance, cardinality, etc
}

// FeatureDiscoveryLog represents activity in the discovery process
type FeatureDiscoveryLog struct {
	ID        string    `db:"id"`
	Timestamp time.Time `db:"timestamp"`
	Action    string    `db:"action"` // "scan_start", "scan_complete", "candidate_found", etc
	Details   string    `db:"details"`
	Status    string    `db:"status"`
}

// FeatureCatalog represents the master feature catalog
type FeatureCatalog struct {
	ID            string
	Name          string
	Description   string
	Version       string
	LocalFeatures []string // Feature names in this catalog
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DiscoveryConfig holds configuration for feature discovery
type DiscoveryConfig struct {
	ScanInterval        time.Duration      // How often to scan for new features
	PostgresDatabases   []string           // Postgres DBs to scan
	TrinoDatabases      []string           // Trino warehouses to scan
	S3Buckets           []string           // S3 buckets with data
	PrometheusURL       string             // Prometheus endpoint
	MinCardinalityScore float64            // Minimum cardinality score to consider
	MinCompletenessRate float64            // Minimum completeness %
	ScoringWeights      map[string]float64 // Feature scoring weights
}

// DiscoveryResult represents output of a discovery run
type DiscoveryResult struct {
	ID              string
	RunID           string // Batch identifier
	StartTime       time.Time
	EndTime         time.Time
	SourcesScanned  []string           // What was scanned
	CandidatesFound int                // Total candidates
	Candidates      []FeatureCandidate // The actual candidates
	Stats           map[string]interface{}
	RunStatus       string // success, partial, failed
	ErrorMessage    string
}
