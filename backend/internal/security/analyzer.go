package security

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ImpactAnalyzer computes downstream impact of access rules via graph traversal.
type ImpactAnalyzer struct {
	db       *sqlx.DB
	ruleRepo *AccessRuleRepository
}

// NewImpactAnalyzer creates a new impact analyzer.
func NewImpactAnalyzer(db *sqlx.DB, ruleRepo *AccessRuleRepository) *ImpactAnalyzer {
	return &ImpactAnalyzer{
		db:       db,
		ruleRepo: ruleRepo,
	}
}

// ComputeImpact traverses the graph to find all artifacts affected by a rule.
// Uses Apache AGE to traverse: BO → Terms → APIs / BI / AI artifacts.
func (i *ImpactAnalyzer) ComputeImpact(ctx context.Context, ruleID string) (*AccessRuleImpact, error) {
	// Fetch the rule
	rule, err := i.ruleRepo.Get(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rule: %w", err)
	}

	impact := &AccessRuleImpact{
		RuleID:           ruleID,
		BusinessObjectID: rule.BusinessObjectID,
	}

	// Traverse graph to find semantic terms linked to the BO
	terms, err := i.getSemanticTerms(ctx, rule.BusinessObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get semantic terms: %w", err)
	}
	impact.SemanticTerms = terms

	// Traverse to APIs if scope includes APIs
	if rule.AppliesToApis != nil && *rule.AppliesToApis {
		apis, err := i.getApisForTerms(ctx, terms)
		if err != nil {
			return nil, fmt.Errorf("failed to get APIs: %w", err)
		}
		impact.Apis = apis
	}

	// Traverse to BI artifacts if scope includes BI
	if rule.AppliesToBi != nil && *rule.AppliesToBi {
		bi, err := i.getBiArtifactsForTerms(ctx, terms)
		if err != nil {
			return nil, fmt.Errorf("failed to get BI artifacts: %w", err)
		}
		impact.BiArtifacts = bi
	}

	// Traverse to AI artifacts if scope includes AI
	if rule.AppliesToAi != nil && *rule.AppliesToAi {
		ai, err := i.getAiArtifactsForTerms(ctx, terms)
		if err != nil {
			return nil, fmt.Errorf("failed to get AI artifacts: %w", err)
		}
		impact.AiArtifacts = ai
	}

	return impact, nil
}

// getSemanticTerms retrieves all semantic terms linked to a business object.
func (i *ImpactAnalyzer) getSemanticTerms(ctx context.Context, businessObjectID string) ([]string, error) {
	query := `
		SELECT DISTINCT st.term_name
		FROM semantic_term st
		JOIN business_object_term bot ON st.id = bot.term_id
		WHERE bot.business_object_id = $1
	`

	rows, err := i.db.QueryContext(ctx, query, businessObjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terms []string
	for rows.Next() {
		var term string
		if err := rows.Scan(&term); err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}

	return terms, rows.Err()
}

// getApisForTerms finds all API endpoints that use the given terms.
// This would use AGE graph traversal in production.
func (i *ImpactAnalyzer) getApisForTerms(ctx context.Context, terms []string) ([]string, error) {
	if len(terms) == 0 {
		return []string{}, nil
	}

	// Placeholder: in production, use AGE cypher query:
	// MATCH (t:SemanticTerm)-[:USED_BY]->(a:ApiEndpoint)
	// WHERE t.name IN $terms
	// RETURN DISTINCT a.path

	query := `
		SELECT DISTINCT api_endpoint
		FROM api_term_usage
		WHERE term_name = ANY($1)
	`

	rows, err := i.db.QueryContext(ctx, query, terms)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apis []string
	for rows.Next() {
		var api string
		if err := rows.Scan(&api); err != nil {
			return nil, err
		}
		apis = append(apis, api)
	}

	return apis, rows.Err()
}

// getBiArtifactsForTerms finds all BI artifacts (reports, dashboards) using the terms.
func (i *ImpactAnalyzer) getBiArtifactsForTerms(ctx context.Context, terms []string) ([]string, error) {
	if len(terms) == 0 {
		return []string{}, nil
	}

	// Placeholder: AGE cypher in production:
	// MATCH (t:SemanticTerm)-[:USED_BY]->(b:BiArtifact)
	// WHERE t.name IN $terms
	// RETURN DISTINCT b.name

	query := `
		SELECT DISTINCT artifact_name
		FROM bi_term_usage
		WHERE term_name = ANY($1)
	`

	rows, err := i.db.QueryContext(ctx, query, terms)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []string
	for rows.Next() {
		var artifact string
		if err := rows.Scan(&artifact); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, rows.Err()
}

// getAiArtifactsForTerms finds all AI artifacts (models, agents) using the terms.
func (i *ImpactAnalyzer) getAiArtifactsForTerms(ctx context.Context, terms []string) ([]string, error) {
	if len(terms) == 0 {
		return []string{}, nil
	}

	// Placeholder: AGE cypher in production:
	// MATCH (t:SemanticTerm)-[:USED_BY]->(ai:AiArtifact)
	// WHERE t.name IN $terms
	// RETURN DISTINCT ai.name

	query := `
		SELECT DISTINCT artifact_name
		FROM ai_term_usage
		WHERE term_name = ANY($1)
	`

	rows, err := i.db.QueryContext(ctx, query, terms)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []string
	for rows.Next() {
		var artifact string
		if err := rows.Scan(&artifact); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, rows.Err()
}
