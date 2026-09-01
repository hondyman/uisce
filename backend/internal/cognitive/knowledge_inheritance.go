package cognitive

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var GoldCopyTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type OKFConcept struct {
	ConceptID          uuid.UUID              `json:"concept_id"`
	TenantID           uuid.UUID              `json:"tenant_id"`
	ConceptKey         string                 `json:"concept_key"`
	ConceptType        string                 `json:"concept_type"`
	Version            string                 `json:"version"`
	IsCore             bool                   `json:"is_core"`
	FrontmatterPayload map[string]interface{} `json:"frontmatter_payload"`
	ContentMarkdown    string                 `json:"content_markdown"`
}

type KnowledgeInheritanceEngine struct {
	db *sql.DB
}

func NewKnowledgeInheritanceEngine(db *sql.DB) *KnowledgeInheritanceEngine {
	return &KnowledgeInheritanceEngine{db: db}
}

// ResolveEffectiveKnowledgeFrame builds the unified context for an LLM prompt
// Layering Core (Gold Copy) with high-precedence Custom Tenant Overlays
func (e *KnowledgeInheritanceEngine) ResolveEffectiveKnowledgeFrame(
	ctx context.Context,
	tenantID uuid.UUID,
	requestedKeys []string,
) (string, []OKFConcept, error) {
	query := `
		WITH ranked_concepts AS (
			SELECT 
				concept_id, tenant_id, concept_key, concept_type, version,
				frontmatter_payload, content_markdown,
				ROW_NUMBER() OVER (
					PARTITION BY concept_key 
					ORDER BY CASE WHEN tenant_id = $1 THEN 1 ELSE 2 END, precedence_score ASC
				) AS ranking
			FROM okf_concept_manifest
			WHERE is_active = TRUE
			  AND (tenant_id = $1 OR tenant_id = $2)
			  AND (cardinality($3::text[]) = 0 OR concept_key = ANY($3))
		)
		SELECT 
			concept_id, tenant_id, concept_key, concept_type, version,
			frontmatter_payload, content_markdown
		FROM ranked_concepts
		WHERE ranking = 1;`

	keysArray := pq.Array(requestedKeys)
	rows, err := e.db.QueryContext(ctx, query, tenantID, GoldCopyTenantID, keysArray)
	if err != nil {
		return "", nil, fmt.Errorf("failed resolving knowledge inheritance: %w", err)
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("### Uisce Verified Knowledge Frame (Core + Custom Layered)\n")
	sb.WriteString("Execute reasoning strictly within the following effective contracts:\n\n")

	var resolvedList []OKFConcept

	for rows.Next() {
		var c OKFConcept
		var rawFrontmatter []byte

		if err := rows.Scan(
			&c.ConceptID, &c.TenantID, &c.ConceptKey, &c.ConceptType, &c.Version,
			&rawFrontmatter, &c.ContentMarkdown,
		); err != nil {
			return "", nil, err
		}

		_ = json.Unmarshal(rawFrontmatter, &c.FrontmatterPayload)
		c.IsCore = (c.TenantID == GoldCopyTenantID)
		resolvedList = append(resolvedList, c)

		provenance := "Custom Tenant Override"
		if c.IsCore {
			provenance = "Gold Copy Standard Template"
		}

		sb.WriteString(fmt.Sprintf("#### Concept: `%s` [%s]\n", c.ConceptKey, provenance))
		sb.WriteString(fmt.Sprintf("- **Type**: `%s` (v%s)\n", c.ConceptType, c.Version))

		if formula, ok := c.FrontmatterPayload["formula_ast"].(string); ok && formula != "" {
			sb.WriteString(fmt.Sprintf("- **Attested Formula**: `%s` (Deterministic Pushdown)\n", formula))
		}

		if tags, ok := c.FrontmatterPayload["tags"].([]interface{}); ok && len(tags) > 0 {
			var tagStrs []string
			for _, t := range tags {
				tagStrs = append(tagStrs, fmt.Sprintf("%v", t))
			}
			sb.WriteString(fmt.Sprintf("- **Tags**: %s\n", strings.Join(tagStrs, ", ")))
		}

		sb.WriteString(fmt.Sprintf("- **Directives & Invariants**:\n%s\n\n", c.ContentMarkdown))
	}

	return sb.String(), resolvedList, nil
}
