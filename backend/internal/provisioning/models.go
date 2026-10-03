package provisioning

import (
	"time"

	"github.com/google/uuid"
)

type ProvisionTenantRequest struct {
	TenantName   string `json:"tenant_name" validate:"required,min=2,max=100"`
	InstanceName string `json:"instance_name" validate:"required,min=2,max=100"`
	TenantCode   string `json:"tenant_code,omitempty"`
	RequesterID  string `json:"requester_id,omitempty"`
}

type ProvisionTenantResponse struct {
	WorkflowID    string    `json:"workflow_id"`
	TenantID      string    `json:"tenant_id"`
	InstanceID    string    `json:"instance_id"`
	DatabaseName  string    `json:"database_name"`
	LakekeeperNS  string    `json:"lakekeeper_namespace"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
}

type ProvisioningStatus struct {
	WorkflowID   string    `json:"workflow_id"`
	TenantID     string    `json:"tenant_id,omitempty"`
	InstanceID   string    `json:"instance_id,omitempty"`
	DatabaseName string    `json:"database_name,omitempty"`
	Status       string    `json:"status"`
	Error        string    `json:"error,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
}

type ProvisioningWorkflowInput struct {
	TenantID           string
	TenantName         string
	TenantCode         string
	InstanceID         string
	InstanceName       string
	GoldCopyTenantID   string
	GoldCopyInstanceID string
	GoldCopyDatabase   string
	DatabaseName       string
	LakekeeperNS      string
	RequesterID        string
}

type ProvisioningWorkflowResult struct {
	TenantID     string    `json:"tenant_id"`
	InstanceID   string    `json:"instance_id"`
	DatabaseName string    `json:"database_name"`
	LakekeeperNS string    `json:"lakekeeper_namespace"`
	Status       string    `json:"status"`
	Error        string    `json:"error,omitempty"`
	CompletedAt  time.Time `json:"completed_at"`
}

type RegisterTenantInput struct {
	TenantID   string
	TenantName string
	TenantCode string
}

type RegisterInstanceInput struct {
	TenantID     string
	InstanceID   string
	InstanceName string
}

type CreateDatabaseInput struct {
	DatabaseName string
}

type CloneSchemaInput struct {
	SourceDatabase string
	TargetDatabase string
}

type CreateNamespaceInput struct {
	Namespace string
}

type CloneProductsInput struct {
	GoldCopyTenantID   string
	GoldCopyInstanceID string
	TargetTenantID    string
	TargetInstanceID  string
}

type EmitEventInput struct {
	TenantID     string
	InstanceID   string
	DatabaseName string
	Status       string
	Error        string
	CompletedAt  time.Time
}

// ProvisioningState is what the saga may safely undo. RegisterTenant and
// RegisterInstance are upserts that return an EXISTING row's id, and
// CreateTenantDatabase treats "already exists" as success, so a failed run cannot
// assume it created what it touched. It is read once, after registration and
// before anything is created.
type ProvisioningState struct {
	// TenantOwned and InstanceOwned are true only while the row is still in
	// status 'provisioning': created by this run, or abandoned by an earlier one.
	// An active tenant or instance is never owned by a running saga.
	TenantOwned   bool
	InstanceOwned bool
	// DatabaseExisted is true if the tenant database already existed before this
	// run created it, in which case the run must not drop it.
	DatabaseExisted bool
}

// Owned reports whether the saga may compensate at all.
func (s ProvisioningState) Owned() bool { return s.TenantOwned && s.InstanceOwned }

func GenerateTenantCode(name string) string {
	code := ""
	for i, c := range name {
		if i >= 5 {
			break
		}
		if c >= 'A' && c <= 'Z' {
			code += string(c + 32)
		} else if c >= 'a' && c <= 'z' {
			code += string(c)
		} else if c >= '0' && c <= '9' {
			code += string(c)
		} else {
			code += "_"
		}
	}
	return code
}

func NewUUID() string {
	return uuid.New().String()
}
