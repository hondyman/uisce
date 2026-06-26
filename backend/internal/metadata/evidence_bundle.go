package metadata

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EvidenceBundle represents a complete audit trail for an upgrade lifecycle
type EvidenceBundle struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	UpgradeRequestID uuid.UUID       `json:"upgrade_request_id" db:"upgrade_request_id"`
	OldVersion       string          `json:"old_version" db:"old_version"`
	NewVersion       string          `json:"new_version" db:"new_version"`
	Status           BundleStatus    `json:"status" db:"status"`
	Stages           []StageEvidence `json:"stages,omitempty" db:"-"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
}

// BundleStatus represents the lifecycle state of an evidence bundle
type BundleStatus string

const (
	BundleStatusInProgress BundleStatus = "in_progress"
	BundleStatusCompleted  BundleStatus = "completed"
	BundleStatusFailed     BundleStatus = "failed"
	BundleStatusRolledBack BundleStatus = "rolled_back"
)

// StageEvidence represents evidence collected from a single upgrade stage
type StageEvidence struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	BundleID    uuid.UUID       `json:"bundle_id" db:"bundle_id"`
	StageName   StageName       `json:"stage_name" db:"stage_name"`
	Status      StageStatus     `json:"status" db:"status"`
	Artifacts   []Artifact      `json:"artifacts" db:"artifacts"`
	StartedAt   time.Time       `json:"started_at" db:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	ActorID     string          `json:"actor_id,omitempty" db:"actor_id"`
}

// StageName represents the type of upgrade stage
type StageName string

const (
	StageDiff     StageName = "diff"
	StageRebase   StageName = "rebase"
	StageTest     StageName = "test"
	StageApproval StageName = "approval"
	StageDeploy   StageName = "deploy"
	StageRollback StageName = "rollback"
	StageAudit    StageName = "audit"
)

// StageStatus represents the execution status of a stage
type StageStatus string

const (
	StageStatusPending  StageStatus = "pending"
	StageStatusRunning  StageStatus = "running"
	StageStatusSuccess  StageStatus = "success"
	StageStatusFailed   StageStatus = "failed"
	StageStatusSkipped  StageStatus = "skipped"
)

// Artifact represents an immutable evidence artifact
type Artifact struct {
	Type        ArtifactType    `json:"type"`
	StoragePath string          `json:"storage_path"`
	Checksum    string          `json:"checksum"` // SHA256 hash for tamper detection
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	SizeBytes   int64           `json:"size_bytes,omitempty"`
}

// ArtifactType represents the classification of an evidence artifact
type ArtifactType string

const (
	ArtifactTypeDiffReport     ArtifactType = "diff_report"
	ArtifactTypeConflictReport ArtifactType = "conflict_report"
	ArtifactTypeTestResults    ArtifactType = "test_results"
	ArtifactTypeTestCoverage   ArtifactType = "test_coverage"
	ArtifactTypeApprovalRecord ArtifactType = "approval_record"
	ArtifactTypeUARLog         ArtifactType = "uar_log"
	ArtifactTypeDeploymentLog  ArtifactType = "deployment_log"
	ArtifactTypeRollbackLog    ArtifactType = "rollback_log"
	ArtifactTypeIcebergSnapshot ArtifactType = "iceberg_snapshot"
)

// ComplianceReport represents a regulator-facing summary
type ComplianceReport struct {
	BundleID         uuid.UUID       `json:"bundle_id"`
	GeneratedAt      time.Time       `json:"generated_at"`
	ExecutiveSummary ExecutiveSummary `json:"executive_summary"`
	ChangeInventory  []ChangeRecord  `json:"change_inventory"`
	TestSummary      TestSummary     `json:"test_summary"`
	ApprovalChain    []ApprovalDecision `json:"approval_chain"`
	DeploymentLog    DeploymentSummary `json:"deployment_log"`
	Artifacts        []Artifact      `json:"artifacts"`
}

// ExecutiveSummary provides high-level upgrade outcome
type ExecutiveSummary struct {
	Status                 string  `json:"status"`
	RiskLevel              string  `json:"risk_level"` // "LOW", "MEDIUM", "HIGH"
	BreakingChanges        int     `json:"breaking_changes"`
	AdditiveChanges        int     `json:"additive_changes"`
	TestPassRate           float64 `json:"test_pass_rate"`
	DeploymentSuccess      bool    `json:"deployment_success"`
	RollbacksRequired      int     `json:"rollbacks_required"`
}

// ChangeRecord represents a single metadata change
type ChangeRecord struct {
	Path     string      `json:"path"`
	Type     string      `json:"type"`
	Severity string      `json:"severity"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
	Impact   string      `json:"impact"`
}

// TestSummary provides aggregated test results
type TestSummary struct {
	TotalTests    int     `json:"total_tests"`
	PassedTests   int     `json:"passed_tests"`
	FailedTests   int     `json:"failed_tests"`
	SkippedTests  int     `json:"skipped_tests"`
	Coverage      float64 `json:"coverage"`
	ExecutionTime int64   `json:"execution_time_ms"`
	FailedTestDetails []FailedTest `json:"failed_test_details,omitempty"`
}

// FailedTest provides details on a failed test case
type FailedTest struct {
	TestName     string `json:"test_name"`
	ErrorMessage string `json:"error_message"`
	RelatedDiff  string `json:"related_diff,omitempty"`
}

// DeploymentSummary provides rollout details
type DeploymentSummary struct {
	StartedAt        time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	TargetTenants    []string  `json:"target_tenants"`
	SuccessfulDeploys int      `json:"successful_deploys"`
	FailedDeploys    int      `json:"failed_deploys"`
	RollbackEvents   []RollbackEvent `json:"rollback_events,omitempty"`
}

// RollbackEvent represents a deployment rollback
type RollbackEvent struct {
	TenantID  string    `json:"tenant_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
	ActorID   string    `json:"actor_id"`
}
