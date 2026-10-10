package provisioning

import (
	"time"

	"github.com/google/uuid"
)

type ProvisionTenantRequest struct {
	TenantName   string `json:"tenant_name" validate:"required,min=2,max=100"`
	InstanceName string `json:"instance_name" validate:"required,min=2,max=100"`
	TenantCode   string `json:"tenant_code,omitempty"`
	// RequesterID is ignored: the requester is always the authenticated caller.
	RequesterID string `json:"requester_id,omitempty"`
	// App names the application whose datasource the new tenant database serves (e.g. "orm").
	// When set, the tenant also gets its own role, binding, migrations and a probe (ADR-030).
	App string `json:"app,omitempty"`
	// TemplateDatasourceID, with App, builds the tenant's structure from what alpha holds for this datasource after its
	// scan, instead of cloning the gold copy's database (ADR-050). It must be the gold-copy tenant's datasource; the
	// saga checks that before it creates anything. Empty keeps the clone.
	TemplateDatasourceID string `json:"template_datasource_id,omitempty"`
	// StructureFromGoldCopy, with App, builds the structure from the datasource the gold copy marks as the template for App
	// (ADR-050), so the request names no id. The saga refuses when none, or more than one, is marked. Exclusive with
	// TemplateDatasourceID.
	StructureFromGoldCopy bool `json:"structure_from_gold_copy,omitempty"`
	// Region is the region the tenant is created in. A code ("us-east-1"), or a name or its first
	// words ("US East"), which is resolved against the configured regions. Required with Products.
	Region string `json:"region,omitempty"`
	// Products are the products to register, each with the label that names its database
	// (<label>_<product>). When set, the request takes the product path: only these products are
	// registered, the database is named from the label, and the region chooses the cluster. App and
	// the structure flags are derived and must not be set.
	Products []ProductRequest `json:"products,omitempty"`
}

type ProvisionTenantResponse struct {
	WorkflowID    string    `json:"workflow_id"`
	TenantID      string    `json:"tenant_id"`
	InstanceID    string    `json:"instance_id"`
	DatabaseName  string    `json:"database_name"`
	LakekeeperNS  string    `json:"lakekeeper_namespace"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
	// Plan is set on the product path: what the request will create.
	Plan *Plan `json:"plan,omitempty"`
}

type ProvisioningStatus struct {
	WorkflowID   string    `json:"workflow_id"`
	TenantID     string    `json:"tenant_id,omitempty"`
	InstanceID   string    `json:"instance_id,omitempty"`
	DatabaseName string    `json:"database_name,omitempty"`
	Status       string    `json:"status"`
	// Step is the saga step in flight while the run is provisioning.
	Step        string    `json:"step,omitempty"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
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

	// App names the application whose datasource the tenant database serves (e.g. "orm"). When
	// set, the saga also gives the database its own role and credential, repoints that app's
	// cloned datasource at it, applies the app's tenant migrations and probes the connection
	// before the binding goes active (ADR-030). Empty keeps the saga exactly as it was.
	App string `json:"app,omitempty"`
	// BaselineThrough, when set, is the last tenant-migration file already contained in the
	// gold-copy schema this database was cloned from; it is recorded, not run.
	BaselineThrough string `json:"baseline_through,omitempty"`
	// TemplateDatasourceID: see ProvisionTenantRequest. Requires App.
	TemplateDatasourceID string `json:"template_datasource_id,omitempty"`
	// StructureFromGoldCopy: see ProvisionTenantRequest. Requires App.
	StructureFromGoldCopy bool `json:"structure_from_gold_copy,omitempty"`
	// Region, ClusterHost and ClusterPort are set on the product path. The worker creates databases
	// only on the cluster it holds administrator credentials for, and refuses when the region's
	// cluster is another one.
	Region      string `json:"region,omitempty"`
	ClusterHost string `json:"cluster_host,omitempty"`
	ClusterPort int    `json:"cluster_port,omitempty"`
	// ProductCodes are the products to register for the tenant. Empty keeps the clone of every gold-copy product.
	ProductCodes []string `json:"product_codes,omitempty"`
	// Seed asks for the app's reference rows after the structure is applied.
	Seed bool `json:"seed,omitempty"`
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
	// Region, when set, is stored as the tenant's region, default region and only allowed region.
	// Empty keeps the column defaults, as before.
	Region string `json:"region,omitempty"`
}

// RegionDatabaseInput names a database and the cluster the region puts it on.
type RegionDatabaseInput struct {
	Region       string
	Host         string
	Port         int
	DatabaseName string
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
	// ProductCodes, when set, registers only these products. Empty clones every gold-copy product.
	ProductCodes []string `json:"product_codes,omitempty"`
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

// TenantDatabaseInput is the input of the tenant-database saga steps (ADR-030).
type TenantDatabaseInput struct {
	TenantID         string
	InstanceID       string
	App              string
	DatabaseName     string
	GoldCopyDatabase string
	// DatasourceID is set from BindTenantDatabase's result for every later step.
	DatasourceID    string
	BaselineThrough string
	// TemplateDatasourceID and StructureHash drive the structure steps (ADR-050). The hash is what the planning step
	// compiled; applying refuses if the template compiles to something else by then.
	TemplateDatasourceID string
	StructureHash        string
}

// StructurePlan is what planning a tenant's structure found. It carries a summary, never the SQL: the script is large
// and belongs in no workflow history.
type StructurePlan struct {
	// TemplateDatasourceID is the datasource the structure was compiled from; with Hash it is the record of what this run
	// deployed, kept in the workflow's history.
	TemplateDatasourceID string
	Hash       string
	Tables     int
	Statements int
	Schemas    []string
}

// TenantDatabaseBinding is what BindTenantDatabase decided; no secret is in it.
type TenantDatabaseBinding struct {
	DatasourceID string
	Role         string
	SecretPath   string
}
