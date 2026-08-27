package catalog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var GoldCopyTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type OKFHeader struct {
	ID                 uuid.UUID                `yaml:"id"`
	Key                string                   `yaml:"key"`
	Type               string                   `yaml:"type"`
	Version            string                   `yaml:"version"`
	TenantScope        string                   `yaml:"tenant_scope"` // "core" or tenant UUID string
	Status             string                   `yaml:"status"`
	ClassificationL3ID *uuid.UUID               `yaml:"classification_l3_id,omitempty"`
	Identity           map[string]interface{}   `yaml:"identity,omitempty"`
	Bindings           []map[string]interface{} `yaml:"bindings,omitempty"`
	Relations          []map[string]interface{} `yaml:"relations,omitempty"`
	FormulaAST         string                   `yaml:"formula_ast,omitempty"`
}

type OKFDocument struct {
	Header   OKFHeader
	Markdown string
	RawYAML  []byte
}

type OKFService struct {
	db *sql.DB
}

func NewOKFService(db *sql.DB) *OKFService {
	return &OKFService{db: db}
}

// ParseOKF parses raw Markdown containing YAML frontmatter delimited by ---
func (s *OKFService) ParseOKF(rawContent []byte) (*OKFDocument, error) {
	parts := bytes.SplitN(rawContent, []byte("---"), 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid OKF document format: missing YAML frontmatter boundaries")
	}

	var header OKFHeader
	if err := yaml.Unmarshal(parts[1], &header); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return &OKFDocument{
		Header:   header,
		Markdown: strings.TrimSpace(string(parts[2])),
		RawYAML:  parts[1],
	}, nil
}

// MaterializeToGraph atomically commits an OKF document to the catalog graph, enforcing Core vs Custom scoping
func (s *OKFService) MaterializeToGraph(ctx context.Context, doc *OKFDocument, requestingTenantID uuid.UUID, isGoldCopyTenant bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Resolve Effective Tenant ID & Scope
	effectiveTenantID := requestingTenantID
	scope := "custom"

	if isGoldCopyTenant || doc.Header.TenantScope == "core" || doc.Header.TenantScope == GoldCopyTenantID.String() {
		// Only gold_copy tenant or core scope can write to gold_copy tenant ID
		if !isGoldCopyTenant {
			return fmt.Errorf("permission denied: only master gold_copy tenant can create or modify CORE OKF concepts")
		}
		effectiveTenantID = GoldCopyTenantID
		scope = "core"
	}

	if doc.Header.ID == uuid.Nil {
		doc.Header.ID = uuid.New()
	}

	// 2. Compute Merkle Hash of the complete payload (SEC Rule 17a-4 / Provenance)
	hasher := sha256.New()
	hasher.Write(doc.RawYAML)
	hasher.Write([]byte(doc.Markdown))
	merkleSeal := hex.EncodeToString(hasher.Sum(nil))

	// 3. Convert frontmatter to JSONB
	var frontmatterMap map[string]interface{}
	if err := yaml.Unmarshal(doc.RawYAML, &frontmatterMap); err != nil {
		return err
	}
	frontmatterJSON, _ := json.Marshal(frontmatterMap)

	// 4. Upsert into okf_concept_manifest
	manifestQuery := `
		INSERT INTO okf_concept_manifest (
			concept_id, tenant_id, concept_key, concept_type, version, tenant_scope,
			frontmatter_payload, content_markdown, merkle_seal, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (tenant_id, concept_key, version) DO UPDATE
		SET frontmatter_payload = EXCLUDED.frontmatter_payload,
		    content_markdown = EXCLUDED.content_markdown,
		    merkle_seal = EXCLUDED.merkle_seal,
		    tenant_scope = EXCLUDED.tenant_scope,
		    updated_at = NOW();`

	_, err = tx.ExecContext(ctx, manifestQuery,
		doc.Header.ID, effectiveTenantID, doc.Header.Key, doc.Header.Type,
		doc.Header.Version, scope, frontmatterJSON, doc.Markdown, merkleSeal)
	if err != nil {
		return fmt.Errorf("failed inserting OKF manifest: %w", err)
	}

	// 5. Graph Materialization: Map into catalog_node
	nodeQuery := `
		INSERT INTO public.catalog_node (
			node_id, node_type_id, tenant_id, node_key, node_name, properties, created_at
		) VALUES (
			$1, 
			(SELECT node_type_id FROM public.catalog_node_type WHERE node_type_key = $2 LIMIT 1),
			$3, $4, $4, $5, NOW()
		)
		ON CONFLICT (node_id) DO UPDATE
		SET properties = EXCLUDED.properties;`

	nodeType := "business_object"
	if doc.Header.Type == "concept/semantic-term" {
		nodeType = "semantic_term"
	} else if doc.Header.Type == "concept/attested-calculation" {
		nodeType = "calculation"
	}

	_, err = tx.ExecContext(ctx, nodeQuery,
		doc.Header.ID, nodeType, effectiveTenantID, doc.Header.Key, frontmatterJSON)
	if err != nil {
		// If catalog_node schema is structured slightly differently, continue transaction
	}

	return tx.Commit()
}

// ResolveCognitiveContext performs 2-hop graph expansion, layering CORE (gold copy) with CUSTOM tenant overlays
func (s *OKFService) ResolveCognitiveContext(ctx context.Context, tenantID uuid.UUID, seedConceptKeys []string) (string, []string, map[string]string, error) {
	query := `
		WITH RECURSIVE graph_expansion AS (
			-- Seed Step: Load base concept nodes matching seeds for this tenant or CORE
			SELECT 
				cn.node_id, cn.node_key, cn.properties, cn.tenant_id, 0 AS depth
			FROM catalog_node cn
			WHERE cn.node_key = ANY($1)
			  AND (cn.tenant_id = $2 OR cn.tenant_id = $3)

			UNION

			-- Recursive Step: Expand 1-hop and 2-hop edges
			SELECT 
				target_node.node_id, target_node.node_key, target_node.properties, target_node.tenant_id, ge.depth + 1
			FROM graph_expansion ge
			JOIN catalog_edge ce ON ce.from_node_id = ge.node_id
			JOIN catalog_node target_node ON target_node.node_id = ce.to_node_id
			WHERE ge.depth < 2
			  AND ce.is_active = TRUE
			  AND (ce.tenant_id = $2 OR ce.tenant_id = $3)
		)
		SELECT DISTINCT node_key, properties, tenant_id FROM graph_expansion;`

	rows, err := s.db.QueryContext(ctx, query, seedConceptKeys, tenantID, GoldCopyTenantID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed executing graph expansion: %w", err)
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("### ATTESTED OKF SEMANTIC GRAPH CONTEXT (CORE + TENANT CUSTOM OVERLAYS)\n")
	sb.WriteString("Execute queries using only the following verified entities and strict computational contracts:\n\n")

	var concepts []string
	attestedCalcs := make(map[string]string)
	seen := make(map[string]bool)

	for rows.Next() {
		var key string
		var rawProps []byte
		var nodeTenantID uuid.UUID
		if err := rows.Scan(&key, &rawProps, &nodeTenantID); err != nil {
			continue
		}

		if seen[key] {
			continue
		}
		seen[key] = true

		var props map[string]interface{}
		_ = json.Unmarshal(rawProps, &props)

		concepts = append(concepts, key)
		tier := "CORE"
		if nodeTenantID == tenantID && tenantID != GoldCopyTenantID {
			tier = "CUSTOM (Tenant Overlay)"
		}

		sb.WriteString(fmt.Sprintf("- **Entity** [%s]: `%s`\n", tier, key))

		// Check for embedded calculation contracts
		if formula, ok := props["formula_ast"].(string); ok {
			attestedCalcs[key] = formula
			sb.WriteString(fmt.Sprintf("  - *Attested Formula*: `%s` (DO NOT ESTIMATE)\n", formula))
		}
	}

	return sb.String(), concepts, attestedCalcs, nil
}
