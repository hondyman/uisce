package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pgvector/pgvector-go"
)

type DynamicContextBuilder struct {
	DB               *sql.DB
	EmbeddingService *EmbeddingService
}

func NewDynamicContextBuilder(db *sql.DB, embedder *EmbeddingService) *DynamicContextBuilder {
	if embedder == nil {
		embedder = NewEmbeddingService()
	}
	return &DynamicContextBuilder{
		DB:               db,
		EmbeddingService: embedder,
	}
}

func (b *DynamicContextBuilder) BuildAugmentedSystemPrompt(
	ctx context.Context,
	tenantID string,
	accountID string,
	catalog []AIExplorerSemanticField,
	currentQuery AIExplorerQueryDefinition,
	lastUserPrompt string,
) string {
	var sb strings.Builder

	// 1. Catalog Definitions
	sb.WriteString("AVAILABLE SEMANTIC CATALOG:\n")
	for _, f := range catalog {
		sb.WriteString(fmt.Sprintf("- Field: `%s` | Label: \"%s\" | Category: %s | Type: %s | Default Agg: %s\n",
			f.ID, f.Label, f.Category, f.Type, f.Aggregation))
	}

	// 2. Tenant Aliases & Abbreviations (e.g. AUM -> total_valuation)
	aliases := b.fetchTenantAliases(ctx, tenantID)
	if len(aliases) > 0 {
		sb.WriteString("\nAPPROVED DOMAIN ALIASES & JARGON:\n")
		for _, a := range aliases {
			sb.WriteString(fmt.Sprintf("- When user mentions \"%s\", map to field `%s`\n", a.Term, a.TargetField))
		}
	}

	// 3. Vector Semantic Few-Shot Golden Query Retrieval
	var goldenExamples string
	if lastUserPrompt != "" {
		goldenExamples = b.FetchSemanticGoldenQueries(ctx, tenantID, lastUserPrompt, 3)
	} else {
		goldenExamples = b.fetchTopGoldenQueriesFallback(ctx, tenantID, 3)
	}

	if goldenExamples != "" {
		sb.WriteString("\nVERIFIED FEW-SHOT GOLDEN QUERIES (FOLLOW THIS STRUCTURE):\n")
		sb.WriteString(goldenExamples)
	}

	// 4. Session & Current Query State
	currQueryJSON, _ := json.MarshalIndent(currentQuery, "", "  ")
	sb.WriteString(fmt.Sprintf(`
SESSION METADATA:
- Tenant ID: %s
- Active Bound Account ID: %s

ACTIVE QUERY STATE (THIS IS THE CURRENT VIEW TO MUTATE):
%s

CONVERSATIONAL BI MUTATION RULES:
You are modifying the ACTIVE QUERY STATE based on the user's latest prompt.

1. DRILL-DOWN (e.g., "What about just EMEA?", "Drill into Equities", "Break down EMEA by product"):
   - Action: Add the clicked/mentioned category as a new FILTER using the current dimension.
   - Action: Replace the current dimension with the next logical hierarchical dimension (e.g., Region -> Country -> City, or Product Category -> Product Name).
   - Set mutation_intent to "drill_down".

2. DRILL-ACROSS / SWAP DIMENSION (e.g., "Show this by Product instead", "Group by Account Type"):
   - Action: Keep all existing filters and measures.
   - Action: Replace the "dimensions" array with the newly requested dimension.
   - Set mutation_intent to "drill_across".

3. CROSS-FILTER / SLICE (e.g., "Only show active status", "Filter for 2026", "Only include Institutional accounts"):
   - Action: Keep existing dimensions and measures.
   - Action: Append the new requirement to the "filters" array. DO NOT remove previous filters unless asked.
   - Set mutation_intent to "add_filter".

4. ADD MEASURE (e.g., "Also show me the trade count", "Include average return"):
   - Action: Keep everything else the same, append the new measure to the "measures" array.
   - Set mutation_intent to "add_measure".

5. RESET (e.g., "Start over", "Clear filters", "New analysis"):
   - Action: Empty the dimensions and filters array, build from scratch.
   - Set mutation_intent to "new_query".

Always output structured JSON by invoking the generate_semantic_query tool.`,
		tenantID, accountID, string(currQueryJSON)))

	return sb.String()
}

type AliasPair struct {
	Term        string
	TargetField string
}

func (b *DynamicContextBuilder) fetchTenantAliases(ctx context.Context, tenantID string) []AliasPair {
	if b.DB == nil {
		return nil
	}
	rows, err := b.DB.QueryContext(ctx, `
		SELECT alias_term, target_field_id 
		FROM semantic_field_aliases 
		WHERE tenant_id = $1 OR is_global = TRUE
	`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var list []AliasPair
	for rows.Next() {
		var p AliasPair
		if err := rows.Scan(&p.Term, &p.TargetField); err == nil {
			list = append(list, p)
		}
	}
	return list
}

// FetchSemanticGoldenQueries performs a vector similarity search via pgvector Cosine Distance (<=>)
func (b *DynamicContextBuilder) FetchSemanticGoldenQueries(ctx context.Context, tenantID string, prompt string, limit int) string {
	if b.DB == nil {
		return ""
	}

	// 1. Convert user's current prompt into an embedding vector
	if b.EmbeddingService != nil {
		embedding, err := b.EmbeddingService.GenerateEmbedding(ctx, prompt)
		if err == nil && len(embedding) > 0 {
			query := `
				SELECT prompt_pattern, verified_query, (prompt_embedding <=> $1) AS cosine_distance
				FROM ai_golden_queries 
				WHERE (tenant_id = $2 OR is_global = TRUE)
				  AND prompt_embedding IS NOT NULL
				ORDER BY cosine_distance ASC, score DESC
				LIMIT $3
			`
			rows, err := b.DB.QueryContext(ctx, query, pgvector.NewVector(embedding), tenantID, limit)
			if err == nil {
				defer rows.Close()
				var buf bytes.Buffer
				hasResults := false
				for rows.Next() {
					var promptPattern string
					var queryJSON []byte
					var distance float64
					if err := rows.Scan(&promptPattern, &queryJSON, &distance); err == nil {
						hasResults = true
						buf.WriteString(fmt.Sprintf("User: \"%s\"\nTool Call Payload: %s\n\n", promptPattern, string(queryJSON)))
					}
				}
				if hasResults {
					return buf.String()
				}
			}
		}
	}

	// Fallback to frequency-based top queries
	return b.fetchTopGoldenQueriesFallback(ctx, tenantID, limit)
}

func (b *DynamicContextBuilder) fetchTopGoldenQueriesFallback(ctx context.Context, tenantID string, limit int) string {
	if b.DB == nil {
		return ""
	}
	rows, err := b.DB.QueryContext(ctx, `
		SELECT prompt_pattern, verified_query 
		FROM ai_golden_queries 
		WHERE tenant_id = $1 OR is_global = TRUE 
		ORDER BY score DESC 
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return ""
	}
	defer rows.Close()

	var buf bytes.Buffer
	for rows.Next() {
		var prompt string
		var queryJSON []byte
		if err := rows.Scan(&prompt, &queryJSON); err == nil {
			buf.WriteString(fmt.Sprintf("User: \"%s\"\nTool Call Payload: %s\n\n", prompt, string(queryJSON)))
		}
	}
	return buf.String()
}
