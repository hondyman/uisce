package semantic_bridge

import (
	"time"

	"github.com/google/uuid"
)

type VendorType string

const (
	VendorSnowflakeCortex VendorType = "SNOWFLAKE_CORTEX"
	VendorDatabricksGenie VendorType = "DATABRICKS_GENIE"
	VendorClaudeMCP       VendorType = "CLAUDE_MCP"
	VendorCopilotMCP      VendorType = "COPILOT_MCP"
	VendorOpenAIAssistant VendorType = "OPENAI_ASSISTANT"
)

type BridgeTarget struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	TenantID           uuid.UUID              `json:"tenantId" db:"tenant_id"`
	VendorType         VendorType             `json:"vendorType" db:"vendor_type"`
	TargetName         string                 `json:"targetName" db:"target_name"`
	IsActive           bool                   `json:"isActive" db:"is_active"`
	CredentialsVaulted map[string]interface{} `json:"credentialsVaulted,omitempty" db:"credentials_vaulted"`
	ConfigPayload      map[string]interface{} `json:"configPayload" db:"config_payload"`
	SyncFrequency      string                 `json:"syncFrequency" db:"sync_frequency"`
	LastSyncAt         *time.Time             `json:"lastSyncAt" db:"last_sync_at"`
	LastSyncStatus     string                 `json:"lastSyncStatus" db:"last_sync_status"`
	LastSyncError      string                 `json:"lastSyncError,omitempty" db:"last_sync_error"`
	CreatedAt          time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time              `json:"updatedAt" db:"updated_at"`
}

type SyncLog struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenantId" db:"tenant_id"`
	TargetID        *uuid.UUID `json:"targetId" db:"target_id"`
	VendorType      string     `json:"vendorType" db:"vendor_type"`
	Action          string     `json:"action" db:"action"`
	PayloadHash     string     `json:"payloadHash" db:"payload_hash"`
	ArtifactPayload string     `json:"artifactPayload" db:"artifact_payload"`
	Status          string     `json:"status" db:"status"`
	HTTPStatus      int        `json:"httpStatus" db:"http_status"`
	ResponseBody    string     `json:"responseBody" db:"response_body"`
	ExecutionTimeMS int        `json:"executionTimeMs" db:"execution_time_ms"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
}
