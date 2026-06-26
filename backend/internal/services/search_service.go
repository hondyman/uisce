package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
)

// SearchService provides methods for semantic search.
type SearchService struct {
	db          *sqlx.DB
	llmProvider llm.LLMProvider
}

// NewSearchService creates a new SearchService.
func NewSearchService(db *sqlx.DB, llmProvider llm.LLMProvider) *SearchService {
	return &SearchService{
		db:          db,
		llmProvider: llmProvider,
	}
}

// checkClaimsForAsset simulates checking a user's claims against an asset.
// In a real system, this would call a claims service (like the CollaborationService)
// with the user's context and the asset's ID/domain.
func checkClaimsForAsset(viewName string) (hasAccess bool, isRestricted bool, claimMatchScore float64) {
	// Mock claims check based on view_name.
	switch viewName {
	case "orders_view": // Assume full access
		return true, false, 1.0
	case "customer_pii_data": // Assume restricted access (e.g., OLS on some columns)
		return true, true, 0.6
	case "finance_kpis": // Assume no access
		return false, false, 0.0
	default: // Default to full access for other views
		return true, false, 1.0
	}
}

// calculateRelevanceScore computes a composite score based on the blueprint's formula.
func calculateRelevanceScore(
	embeddingSimilarity,
	claimMatch,
	certificationBonus,
	recentUsage float64,
) float64 {
	// score = (0.4 * embeddingSimilarity + 0.3 * claimMatch + 0.2 * certificationBonus + 0.1 * recentUsage)
	score := (0.4*embeddingSimilarity +
		0.3*claimMatch +
		0.2*certificationBonus +
		0.1*recentUsage)
	return score
}

// HybridSearch performs a combined keyword (BM25) and vector search.
func (s *SearchService) HybridSearch(ctx context.Context, req models.SemanticSearchRequest, secCtx security.Context) ([]models.SemanticSearchResultItem, error) {
	datasourceID := secCtx.DatasourceID

	// 1. Generate embedding for the query
	queryEmbedding, err := s.llmProvider.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	queryVector := vectorToString(queryEmbedding)

	// 2. Prepare SQL query
	// We'll search both catalog_node and business_terms using a UNION
	// We use Reciprocal Rank Fusion (RRF) concept or weighted sum.
	// Here we'll use a weighted sum of normalized scores.
	// ts_rank_cd gives a rank for text match.
	// (1 - (embedding <=> query)) gives cosine similarity (0 to 1).

	sqlQuery := `
		WITH catalog_results AS (
			SELECT 
				id, 
				node_name as name, 
				description, 
				'catalog_node' as type,
				qualified_path,
				ts_rank_cd(to_tsvector('english', node_name || ' ' || COALESCE(description, '')), websearch_to_tsquery('english', $1)) as text_score,
				(1 - (embedding <=> $2::vector)) as vector_score
			FROM catalog_node
			WHERE tenant_id = $3
			AND ($4 = '' OR tenant_datasource_id = $4)
			AND (
				to_tsvector('english', node_name || ' ' || COALESCE(description, '')) @@ websearch_to_tsquery('english', $1)
				OR
				embedding <=> $2::vector < 0.5 -- Threshold for vector match
			)
		),
		term_results AS (
			SELECT 
				id, 
				term as name, 
				definition as description, 
				'business_term' as type,
				canonical_key as qualified_path,
				ts_rank_cd(to_tsvector('english', term || ' ' || definition), websearch_to_tsquery('english', $1)) as text_score,
				(1 - (embedding <=> $2::vector)) as vector_score
			FROM business_terms
			WHERE tenant_id = $3
			AND (
				to_tsvector('english', term || ' ' || definition) @@ websearch_to_tsquery('english', $1)
				OR
				embedding <=> $2::vector < 0.5
			)
		)
		SELECT * FROM (
			SELECT *, (0.3 * text_score + 0.7 * vector_score) as final_score FROM catalog_results
			UNION ALL
			SELECT *, (0.3 * text_score + 0.7 * vector_score) as final_score FROM term_results
		) combined
		ORDER BY final_score DESC
		LIMIT 20
	`

	var dbResults []struct {
		ID            uuid.UUID      `db:"id"`
		Name          string         `db:"name"`
		Description   sql.NullString `db:"description"`
		Type          string         `db:"type"`
		QualifiedPath sql.NullString `db:"qualified_path"`
		TextScore     float64        `db:"text_score"`
		VectorScore   float64        `db:"vector_score"`
		FinalScore    float64        `db:"final_score"`
	}

	err = s.db.SelectContext(ctx, &dbResults, sqlQuery, req.Query, queryVector, secCtx.TenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	// 3. Map to response
	var results []models.SemanticSearchResultItem
	for _, r := range dbResults {
		desc := ""
		if r.Description.Valid {
			desc = r.Description.String
		}
		qp := ""
		if r.QualifiedPath.Valid {
			qp = r.QualifiedPath.String
		}
		results = append(results, models.SemanticSearchResultItem{
			ID:                 r.ID,
			Name:               r.Name,
			Description:        sql.NullString{String: desc, Valid: true},
			Type:               r.Type,
			Score:              r.FinalScore,
			SemanticSimilarity: r.VectorScore,
			QualifiedPath:      qp,
		})
	}

	return results, nil
}

// SemanticSearch performs a search across different asset types.
// This now delegates to HybridSearch for a more robust implementation.
func (s *SearchService) SemanticSearch(ctx context.Context, req models.SemanticSearchRequest, secCtx security.Context) ([]models.SemanticSearchResultItem, error) {
	return s.HybridSearch(ctx, req, secCtx)
}

// GetSuggestions returns personalized suggestions for a user.
// NOTE: This is a mocked implementation.
func (s *SearchService) GetSuggestions(ctx context.Context, userID, datasourceID string) ([]models.SemanticSearchResultItem, error) {
	// In a real implementation, this would query a suggestion engine that considers
	// user history, team usage, semantic similarity, etc.

	// Mock response with more personalized reasons
	suggestions := []models.SemanticSearchResultItem{
		{
			Type:        "query",
			ID:          uuid.New(),
			Name:        "Weekly Active Users",
			Score:       0.95,
			OwnerUserID: "analytics_team",
			Reason:      "Popular with your team",
			Certified:   true,
			Popular:     true,
		},
		{
			Type:        "workbook",
			ID:          uuid.New(),
			Name:        "Customer Churn Deep Dive",
			Score:       0.88,
			OwnerUserID: "product_team",
			Reason:      "Similar to your recent activity",
		},
		{
			Type:        "query",
			ID:          uuid.New(),
			Name:        "Daily Revenue Flash",
			Score:       0.85,
			OwnerUserID: "finance_team",
			Reason:      "You recently ran this",
		},
	}
	return suggestions, nil
}

// LogFeedback records a user's interaction with a search result.
func (s *SearchService) LogFeedback(ctx context.Context, req models.SearchFeedbackRequest, userID string) error {
	feedback := models.SearchFeedback{
		ID:         uuid.New(),
		UserID:     userID,
		Query:      req.Query,
		ResultID:   req.ResultID,
		ResultType: req.ResultType,
		Action:     req.Action,
		Timestamp:  time.Now(),
	}
	query := `INSERT INTO explorer_search_feedback (id, user_id, query, result_id, result_type, action, timestamp) VALUES (:id, :user_id, :query, :result_id, :result_type, :action, :timestamp)`
	_, err := s.db.NamedExecContext(ctx, query, feedback)
	if err != nil {
		// Log error but don't fail the request for the user, as this is a background task.
		logging.GetLogger().Sugar().Errorf("ERROR: failed to log search feedback: %v", err)
	}

	// In a real system, this would also trigger an update to the usage/ranking cache.
	return nil
}

func contains(slice []string, item string) bool {
	if slice == nil {
		return false
	}
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func vectorToString(vector []float32) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range vector {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%f", v))
	}
	sb.WriteString("]")
	return sb.String()
}
