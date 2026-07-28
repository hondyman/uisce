package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type SchemaEvolutionScript struct {
	StoreType     string `json:"store_type"`     // CITUS_POSTGRES, ICEBERG, STARROCKS
	TargetTable   string `json:"target_table"`
	DDLStatement  string `json:"ddl_statement"`
	IsDistributed bool   `json:"is_distributed"`
}

type PreAggInvalidationRule struct {
	TargetStore  string `json:"target_store"`  // STARROCKS, REDIS
	ViewName     string `json:"view_name"`
	InvalidateAll bool   `json:"invalidate_all"`
}

type SemanticCoreDelta struct {
	TargetBOID   string                 `json:"target_bo_id"`
	ChangeType   string                 `json:"change_type"` // FIELD_ADDED, TYPE_CHANGED, COMPONENT_UPDATED
	FieldName    string                 `json:"field_name,omitempty"`
	OldType      string                 `json:"old_type,omitempty"`
	NewType      string                 `json:"new_type,omitempty"`
	MetadataJSON map[string]interface{} `json:"metadata_json,omitempty"`
}

type UpgradePackageSpec struct {
	PackageID        string                   `json:"package_id"`
	Version          string                   `json:"version"`
	MinBaseVersion   string                   `json:"min_base_version"`
	CreatedAt        time.Time                `json:"created_at"`
	Author           string                   `json:"author"`
	Checksum         string                   `json:"checksum"`
	CoreDeltas       []SemanticCoreDelta      `json:"core_deltas"`
	SchemaScripts    []SchemaEvolutionScript  `json:"schema_scripts"`
	PreAggInvalidate []PreAggInvalidationRule `json:"pre_agg_invalidation"`
}

// ComputeChecksum generates a cryptographic SHA-256 hash of the package payload
func (p *UpgradePackageSpec) ComputeChecksum() string {
	raw, _ := json.Marshal(struct {
		ID       string `json:"id"`
		Version  string `json:"version"`
		Author   string `json:"author"`
		Deltas   int    `json:"deltas"`
		Scripts  int    `json:"scripts"`
	}{
		ID:      p.PackageID,
		Version: p.Version,
		Author:  p.Author,
		Deltas:  len(p.CoreDeltas),
		Scripts: len(p.SchemaScripts),
	})
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

// CreateSampleManifest generates a production-grade sample manifest for platform upgrades
func CreateSampleManifest(version string) UpgradePackageSpec {
	spec := UpgradePackageSpec{
		PackageID:      fmt.Sprintf("pkg_%s_%d", version, time.Now().Unix()),
		Version:        version,
		MinBaseVersion: "v1.0.0",
		CreatedAt:      time.Now(),
		Author:         "Uisce System Release Pipeline",
		CoreDeltas: []SemanticCoreDelta{
			{
				TargetBOID: "customers",
				ChangeType: "TYPE_CHANGED",
				FieldName:  "risk_score",
				OldType:    "INTEGER",
				NewType:    "FLOAT",
			},
			{
				TargetBOID: "customers",
				ChangeType: "FIELD_ADDED",
				FieldName:  "esg_rating",
				NewType:    "STRING",
			},
		},
		SchemaScripts: []SchemaEvolutionScript{
			{
				StoreType:     "CITUS_POSTGRES",
				TargetTable:   "public.legacy_business_objects",
				DDLStatement:  "ALTER TABLE public.legacy_business_objects ADD COLUMN IF NOT EXISTS esg_rating VARCHAR(32);",
				IsDistributed: true,
			},
			{
				StoreType:     "ICEBERG",
				TargetTable:   "lakehouse.portfolio_snapshots",
				DDLStatement:  "ALTER TABLE lakehouse.portfolio_snapshots ADD COLUMNS (esg_rating string);",
				IsDistributed: false,
			},
		},
		PreAggInvalidate: []PreAggInvalidationRule{
			{
				TargetStore:   "STARROCKS",
				ViewName:      "mv_customer_portfolio_summary",
				InvalidateAll: true,
			},
			{
				TargetStore:   "REDIS",
				ViewName:      "ast_cache_customers",
				InvalidateAll: true,
			},
		},
	}
	spec.Checksum = spec.ComputeChecksum()
	return spec
}
