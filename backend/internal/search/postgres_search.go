package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SearchResult represents a single search result
type SearchResult struct {
	EntityType  string  `json:"entity_type"`
	EntityID    string  `json:"entity_id"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Description string  `json:"description,omitempty"`
	Rank        float32 `json:"rank"`
	Highlight   string  `json:"highlight,omitempty"`
}

// AutocompleteResult represents an autocomplete suggestion
type AutocompleteResult struct {
	EntityType      string  `json:"entity_type"`
	EntityID        string  `json:"entity_id"`
	Name            string  `json:"name"`
	DisplayName     string  `json:"display_name"`
	SimilarityScore float32 `json:"similarity_score"`
}

// FacetValue represents a facet option with count
type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// SearchFacets contains all facet categories
type SearchFacets struct {
	ObjectTypes []FacetValue `json:"object_types,omitempty"`
	Datasources []FacetValue `json:"datasources,omitempty"`
}

// FacetedSearchResponse contains results with facets
type FacetedSearchResponse struct {
	Results    []SearchResult `json:"results"`
	Facets     SearchFacets   `json:"facets"`
	TotalCount int64          `json:"total_count"`
}

// SearchOptions configures a search query
type SearchOptions struct {
	Query        string
	TenantID     uuid.UUID
	DatasourceID *uuid.UUID
	EntityTypes  []string // semantic_object, bundle, policy, table
	Limit        int
	Offset       int
	Filters      map[string]string
}

// PostgresSearchService implements search using PostgreSQL full-text search
type PostgresSearchService struct {
	db *sql.DB
}

// NewPostgresSearchService creates a new search service
func NewPostgresSearchService(db *sql.DB) *PostgresSearchService {
	return &PostgresSearchService{db: db}
}

// Search performs a full-text search across entities
func (s *PostgresSearchService) Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error) {
	if opts.Limit == 0 {
		opts.Limit = 50
	}
	if len(opts.EntityTypes) == 0 {
		opts.EntityTypes = []string{"semantic_object", "bundle", "policy", "table"}
	}

	query := `SELECT entity_type, entity_id, name, display_name, description, rank, highlight 
		FROM search_all($1, $2, $3, $4, $5, $6)`

	rows, err := s.db.QueryContext(ctx, query,
		opts.Query,
		opts.TenantID,
		opts.DatasourceID,
		opts.EntityTypes,
		opts.Limit,
		opts.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var description, highlight sql.NullString
		if err := rows.Scan(&r.EntityType, &r.EntityID, &r.Name, &r.DisplayName, &description, &r.Rank, &highlight); err != nil {
			return nil, err
		}
		r.Description = description.String
		r.Highlight = highlight.String
		results = append(results, r)
	}

	return results, rows.Err()
}

// Autocomplete returns prefix-based suggestions
func (s *PostgresSearchService) Autocomplete(ctx context.Context, opts SearchOptions) ([]AutocompleteResult, error) {
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	if len(opts.EntityTypes) == 0 {
		opts.EntityTypes = []string{"semantic_object", "bundle", "policy"}
	}

	query := `SELECT entity_type, entity_id, name, display_name, similarity_score 
		FROM autocomplete($1, $2, $3, $4, $5)`

	rows, err := s.db.QueryContext(ctx, query,
		opts.Query,
		opts.TenantID,
		opts.DatasourceID,
		opts.EntityTypes,
		opts.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AutocompleteResult
	for rows.Next() {
		var r AutocompleteResult
		if err := rows.Scan(&r.EntityType, &r.EntityID, &r.Name, &r.DisplayName, &r.SimilarityScore); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// SearchWithFacets performs search and returns faceted results
func (s *PostgresSearchService) SearchWithFacets(ctx context.Context, opts SearchOptions) (*FacetedSearchResponse, error) {
	filtersJSON, err := json.Marshal(opts.Filters)
	if err != nil {
		return nil, err
	}

	query := `SELECT results, facets, total_count FROM search_with_facets($1, $2, $3, $4)`

	var resultsJSON, facetsJSON []byte
	var totalCount int64

	err = s.db.QueryRowContext(ctx, query,
		opts.Query,
		opts.TenantID,
		opts.DatasourceID,
		filtersJSON,
	).Scan(&resultsJSON, &facetsJSON, &totalCount)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if resultsJSON != nil {
		if err := json.Unmarshal(resultsJSON, &results); err != nil {
			return nil, err
		}
	}

	var facets SearchFacets
	if facetsJSON != nil {
		if err := json.Unmarshal(facetsJSON, &facets); err != nil {
			return nil, err
		}
	}

	return &FacetedSearchResponse{
		Results:    results,
		Facets:     facets,
		TotalCount: totalCount,
	}, nil
}

// LogSearch records a search for analytics
func (s *PostgresSearchService) LogSearch(ctx context.Context, tenantID, userID uuid.UUID, query string, resultCount int, durationMs int) error {
	_, err := s.db.ExecContext(ctx,
		`SELECT log_search($1, $2, $3, $4, $5)`,
		tenantID, userID, query, resultCount, durationMs,
	)
	return err
}

// SearchMiddleware wraps search calls with timing and logging
func (s *PostgresSearchService) SearchWithAnalytics(ctx context.Context, opts SearchOptions, userID uuid.UUID) ([]SearchResult, error) {
	start := time.Now()

	results, err := s.Search(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Log asynchronously
	go func() {
		duration := int(time.Since(start).Milliseconds())
		_ = s.LogSearch(context.Background(), opts.TenantID, userID, opts.Query, len(results), duration)
	}()

	return results, nil
}

// Interface for dependency injection
type SearchService interface {
	Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
	Autocomplete(ctx context.Context, opts SearchOptions) ([]AutocompleteResult, error)
	SearchWithFacets(ctx context.Context, opts SearchOptions) (*FacetedSearchResponse, error)
	LogSearch(ctx context.Context, tenantID, userID uuid.UUID, query string, resultCount int, durationMs int) error
}

// Ensure PostgresSearchService implements SearchService
var _ SearchService = (*PostgresSearchService)(nil)
