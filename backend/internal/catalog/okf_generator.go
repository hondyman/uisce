package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// OKFConceptFrontmatter represents the standard Open Knowledge Format YAML frontmatter
type OKFConceptFrontmatter struct {
	ID          string            `yaml:"id"`
	Type        string            `yaml:"type"` // e.g. concept/business-object, concept/computation, concept/term
	Version     string            `yaml:"version"`
	Status      string            `yaml:"status"` // draft, production, obsolete
	TenantScope string            `yaml:"tenant_scope"` // core, custom
	Verified    OKFVerified       `yaml:"verified"`
	Tags        []string          `yaml:"tags,omitempty"`
	Fields      []OKFField        `yaml:"fields,omitempty"`
	Parameters  []OKFParameter    `yaml:"parameters,omitempty"`
	Formula     string            `yaml:"formula,omitempty"`
	Related     []OKFRelation     `yaml:"related,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

type OKFVerified struct {
	By        string `yaml:"by"`
	Timestamp string `yaml:"timestamp"`
	Tier      string `yaml:"tier"` // unverified, machine-confirmed, human-reviewed
}

type OKFField struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required,omitempty"`
	Aggregation string `yaml:"aggregation,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type OKFParameter struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Default     string `yaml:"default,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type OKFRelation struct {
	Target   string `yaml:"target"`
	Relation string `yaml:"relation"` // HAS_MANY, ATTRIBUTE_OF, CLASSIFIED_AS
}

// OKFConceptDocument represents a full OKF markdown concept bundle
type OKFConceptDocument struct {
	Frontmatter OKFConceptFrontmatter `json:"frontmatter"`
	Markdown    string                `json:"markdown"`
	RawContent  string                `json:"raw_content"`
}

// OKFGenerator dynamically extracts and serializes catalog graphs into OKF bundles
type OKFGenerator struct{}

func NewOKFGenerator() *OKFGenerator {
	return &OKFGenerator{}
}

// GenerateFromBusinessObject generates an OKF concept from a Business Object definition
func (g *OKFGenerator) GenerateFromBusinessObject(
	ctx context.Context,
	db *sql.DB,
	tenantID uuid.UUID,
	boKey string,
) (*OKFConceptDocument, error) {
	var boID, name, displayName, technicalName, status string
	var isCore bool

	query := `
		SELECT id::text, name, display_name, technical_name, status, is_core
		FROM business_objects
		WHERE tenant_id = $1 AND key = $2
		LIMIT 1
	`
	err := db.QueryRowContext(ctx, query, tenantID, boKey).Scan(
		&boID, &name, &displayName, &technicalName, &status, &isCore,
	)
	if err != nil {
		return nil, fmt.Errorf("business object %s not found: %w", boKey, err)
	}

	// Fetch fields / columns from catalog_node
	fieldsQuery := `
		SELECT node_name, data_type, is_nullable
		FROM catalog_node
		WHERE tenant_id = $1 AND parent_path = $2
		ORDER BY node_name ASC
	`
	rows, err := db.QueryContext(ctx, fieldsQuery, tenantID, technicalName)
	var okfFields []OKFField
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var fName, dType string
			var isNullable bool
			if err := rows.Scan(&fName, &dType, &isNullable); err == nil {
				okfFields = append(okfFields, OKFField{
					Name:     fName,
					Type:     dType,
					Required: !isNullable,
				})
			}
		}
	}

	scope := "custom"
	if isCore {
		scope = "core"
	}

	fm := OKFConceptFrontmatter{
		ID:          boKey,
		Type:        "concept/business-object",
		Version:     "1.0.0",
		Status:      strings.ToLower(status),
		TenantScope: scope,
		Verified: OKFVerified{
			By:        "uisce-catalog-generator",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tier:      "machine-confirmed",
		},
		Tags:   []string{"semantic", "business-object", strings.ToLower(name)},
		Fields: okfFields,
		Metadata: map[string]string{
			"technical_name": technicalName,
			"bo_id":          boID,
		},
	}

	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OKF frontmatter: %w", err)
	}

	body := fmt.Sprintf("# %s\n\nCanonical semantic definition for **%s** (`%s`).\nGoverned under Uuisce Multi-Tenant GSIFI catalog isolation.", displayName, name, technicalName)
	raw := fmt.Sprintf("---\n%s---\n\n%s\n", string(yamlBytes), body)

	return &OKFConceptDocument{
		Frontmatter: fm,
		Markdown:    body,
		RawContent:  raw,
	}, nil
}
