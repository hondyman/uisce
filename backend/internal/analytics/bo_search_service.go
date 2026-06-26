package analytics

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// BOSearchService handles BO search and discovery
type BOSearchService struct {
	db *sqlx.DB
}

// NewBOSearchService creates a new search service
func NewBOSearchService(db *sqlx.DB) *BOSearchService {
	return &BOSearchService{db: db}
}

// SearchType defines the type of search
type SearchType string

const (
	SearchTypeTerm      SearchType = "term"
	SearchTypeRelatedBO SearchType = "related_bo"
	SearchTypeTable     SearchType = "table"
	SearchTypeCalc      SearchType = "calc"
	SearchTypeAll       SearchType = "all"
)

// SearchResult represents a single search result
type SearchResult struct {
	Type               string  `json:"type"`
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	MatchType          string  `json:"match_type"`
	MatchedTerm        string  `json:"matched_term,omitempty"`
	MatchedTable       string  `json:"matched_table,omitempty"`
	MatchedCalc        string  `json:"matched_calculation,omitempty"`
	Relationship       string  `json:"relationship,omitempty"`
	TermCount          int     `json:"term_count,omitempty"`
	Score              float64 `json:"score"`
	NameSimilarity     float64 `json:"-"`
	DescSimilarity     float64 `json:"-"`
	PhysicalSimilarity float64 `json:"-"`
	GraphProximity     float64 `json:"-"`
	CalcDependency     float64 `json:"-"`
	DomainRelevance    float64 `json:"-"`
}

// SearchResponse contains search results
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// SearchWeights defines scoring weights for each dimension
type SearchWeights struct {
	Name     float64
	Desc     float64
	Physical float64
	Graph    float64
	Calc     float64
	Domain   float64
}

// Search performs multi-dimensional BO search
func (s *BOSearchService) Search(
	query string,
	searchType SearchType,
	limit int,
	offset int,
	domain string,
) (*SearchResponse, error) {
	if query == "" {
		return &SearchResponse{Results: []SearchResult{}, Total: 0}, nil
	}

	var results []SearchResult

	switch searchType {
	case SearchTypeTerm:
		results = s.searchByTerm(query, domain)
	case SearchTypeRelatedBO:
		results = s.searchByRelatedBO(query, domain)
	case SearchTypeTable:
		results = s.searchByTable(query, domain)
	case SearchTypeCalc:
		results = s.searchByCalculation(query, domain)
	case SearchTypeAll:
		results = s.searchAll(query, domain)
	}

	// Rank results
	s.rankResults(results, searchType)

	// Paginate
	total := len(results)
	if offset >= total {
		results = []SearchResult{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		results = results[offset:end]
	}

	return &SearchResponse{
		Results: results,
		Total:   total,
	}, nil
}

// searchByTerm finds BOs by term name/description/physical mapping
func (s *BOSearchService) searchByTerm(query string, domain string) []SearchResult {
	// Use PostgreSQL similarity for fuzzy matching
	querySQL := `
		SELECT DISTINCT
			bo.id,
			bo.node_name as bo_name,
			term.node_name as term_name,
			COALESCE(term.properties->>'description', '') as term_desc,
			COALESCE(term.properties->'physical_mapping'->>'table', '') as table_name,
			COALESCE(term.properties->'physical_mapping'->>'column', '') as column_name,
			COALESCE(bo.properties->>'domain', '') as bo_domain,
			similarity(term.node_name, $1) as name_sim,
			similarity(COALESCE(term.properties->>'description', ''), $1) as desc_sim,
			similarity(COALESCE(term.properties->'physical_mapping'->>'column', ''), $1) as phys_sim
		FROM catalog_node bo
		JOIN catalog_edge e ON e.source_node_id = bo.id AND e.edge_type = 'HAS_ATTRIBUTE'
		JOIN catalog_node term ON term.id = e.target_node_id
		WHERE 
			bo.node_type = 'business_object'
			AND term.node_type = 'semantic_term'
			AND (
				term.node_name ILIKE '%' || $1 || '%'
				OR term.properties->>'description' ILIKE '%' || $1 || '%'
				OR term.properties->'physical_mapping'->>'column' ILIKE '%' || $1 || '%'
				OR term.properties->'physical_mapping'->>'table' ILIKE '%' || $1 || '%'
			)
		ORDER BY name_sim DESC
		LIMIT 100
	`

	rows, err := s.db.Query(querySQL, query)
	if err != nil {
		return []SearchResult{}
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var nameSim, descSim, physSim sql.NullFloat64
		var boDomain sql.NullString

		err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.MatchedTerm,
			&r.MatchedTable,
			&r.MatchedTable,
			&r.MatchedTable,
			&boDomain,
			&nameSim,
			&descSim,
			&physSim,
		)

		if err != nil {
			continue
		}

		r.Type = "bo"
		r.MatchType = "term"
		r.NameSimilarity = nameSim.Float64
		r.DescSimilarity = descSim.Float64
		r.PhysicalSimilarity = physSim.Float64
		r.GraphProximity = 1.0 // Direct match
		r.DomainRelevance = s.calculateDomainRelevance(boDomain.String, domain)

		results = append(results, r)
	}

	return results
}

// searchByRelatedBO finds BOs by relationship graph
func (s *BOSearchService) searchByRelatedBO(query string, domain string) []SearchResult {
	// Find BOs related to the query BO
	querySQL := `
		WITH target_bo AS (
			SELECT id, node_name
			FROM catalog_node
			WHERE node_type = 'business_object'
			AND node_name ILIKE '%' || $1 || '%'
			LIMIT 1
		)
		SELECT DISTINCT
			related_bo.id,
			related_bo.node_name as bo_name,
			target_bo.node_name as related_name,
			COALESCE(related_bo.properties->>'domain', '') as bo_domain,
			1 as hop_count
		FROM target_bo
		JOIN catalog_edge e1 ON e1.source_node_id = target_bo.id
		JOIN catalog_edge e2 ON e2.target_node_id = e1.target_node_id
		JOIN catalog_node related_bo ON related_bo.id = e2.source_node_id
		WHERE 
			related_bo.node_type = 'business_object'
			AND related_bo.id != target_bo.id
		LIMIT 50
	`

	rows, err := s.db.Query(querySQL, query)
	if err != nil {
		return []SearchResult{}
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var relatedName, boDomain sql.NullString
		var hopCount int

		err := rows.Scan(&r.ID, &r.Name, &relatedName, &boDomain, &hopCount)
		if err != nil {
			continue
		}

		r.Type = "bo"
		r.MatchType = "related_bo"
		r.Relationship = fmt.Sprintf("%s → %s", relatedName.String, r.Name)
		r.GraphProximity = s.calculateGraphProximity(hopCount)
		r.NameSimilarity = 0.5
		r.DomainRelevance = s.calculateDomainRelevance(boDomain.String, domain)

		results = append(results, r)
	}

	return results
}

// searchByTable finds BOs by driving table
func (s *BOSearchService) searchByTable(query string, domain string) []SearchResult {
	querySQL := `
		SELECT DISTINCT
			bo.id,
			bo.node_name as bo_name,
			term.properties->'physical_mapping'->>'table' as table_name,
			COUNT(term.id) as term_count,
			COALESCE(bo.properties->>'domain', '') as bo_domain,
			similarity(term.properties->'physical_mapping'->>'table', $1) as table_sim
		FROM catalog_node bo
		JOIN catalog_edge e ON e.source_node_id = bo.id AND e.edge_type = 'HAS_ATTRIBUTE'
		JOIN catalog_node term ON term.id = e.target_node_id
		WHERE 
			bo.node_type = 'business_object'
			AND term.node_type = 'semantic_term'
			AND term.properties->'physical_mapping'->>'table' ILIKE '%' || $1 || '%'
		GROUP BY bo.id, bo.node_name, table_name, bo_domain
		ORDER BY table_sim DESC
		LIMIT 50
	`

	rows, err := s.db.Query(querySQL, query)
	if err != nil {
		return []SearchResult{}
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var tableSim sql.NullFloat64
		var boDomain sql.NullString

		err := rows.Scan(&r.ID, &r.Name, &r.MatchedTable, &r.TermCount, &boDomain, &tableSim)
		if err != nil {
			continue
		}

		r.Type = "bo"
		r.MatchType = "driving_table"
		r.PhysicalSimilarity = tableSim.Float64
		r.GraphProximity = 0.8
		r.DomainRelevance = s.calculateDomainRelevance(boDomain.String, domain)

		results = append(results, r)
	}

	return results
}

// searchByCalculation finds BOs by calculation
func (s *BOSearchService) searchByCalculation(query string, domain string) []SearchResult {
	querySQL := `
		SELECT DISTINCT
			bo.id,
			bo.node_name as bo_name,
			calc.node_name as calc_name,
			COALESCE(bo.properties->>'domain', '') as bo_domain,
			similarity(calc.node_name, $1) as calc_sim
		FROM catalog_node bo
		JOIN catalog_edge e ON e.source_node_id = bo.id AND e.edge_type = 'BO_HAS_CALC'
		JOIN catalog_node calc ON calc.id = e.target_node_id
		WHERE 
			bo.node_type = 'business_object'
			AND calc.node_name ILIKE '%' || $1 || '%'
		ORDER BY calc_sim DESC
		LIMIT 50
	`

	rows, err := s.db.Query(querySQL, query)
	if err != nil {
		return []SearchResult{}
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var calcSim sql.NullFloat64
		var boDomain sql.NullString

		err := rows.Scan(&r.ID, &r.Name, &r.MatchedCalc, &boDomain, &calcSim)
		if err != nil {
			continue
		}

		r.Type = "bo"
		r.MatchType = "calculation"
		r.CalcDependency = calcSim.Float64
		r.NameSimilarity = calcSim.Float64
		r.DomainRelevance = s.calculateDomainRelevance(boDomain.String, domain)

		results = append(results, r)
	}

	return results
}

// searchAll combines all search types
func (s *BOSearchService) searchAll(query string, domain string) []SearchResult {
	results := []SearchResult{}
	results = append(results, s.searchByTerm(query, domain)...)
	results = append(results, s.searchByRelatedBO(query, domain)...)
	results = append(results, s.searchByTable(query, domain)...)
	results = append(results, s.searchByCalculation(query, domain)...)

	// Deduplicate by BO ID
	seen := make(map[string]bool)
	unique := []SearchResult{}
	for _, r := range results {
		if !seen[r.ID] {
			seen[r.ID] = true
			unique = append(unique, r)
		}
	}

	return unique
}

// rankResults applies composite scoring and sorts
func (s *BOSearchService) rankResults(results []SearchResult, searchType SearchType) {
	weights := s.getWeights(searchType)

	for i := range results {
		r := &results[i]
		r.Score =
			r.NameSimilarity*weights.Name +
				r.DescSimilarity*weights.Desc +
				r.PhysicalSimilarity*weights.Physical +
				r.GraphProximity*weights.Graph +
				r.CalcDependency*weights.Calc +
				r.DomainRelevance*weights.Domain

		// Clamp to 0-1
		if r.Score > 1.0 {
			r.Score = 1.0
		}
		if r.Score < 0.0 {
			r.Score = 0.0
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		// Primary: score
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// Secondary: match type priority
		iPriority := s.getMatchTypePriority(results[i].MatchType)
		jPriority := s.getMatchTypePriority(results[j].MatchType)
		if iPriority != jPriority {
			return iPriority < jPriority
		}
		// Tertiary: alphabetical
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})
}

// getWeights returns weights based on search type
func (s *BOSearchService) getWeights(searchType SearchType) SearchWeights {
	switch searchType {
	case SearchTypeTerm:
		return SearchWeights{
			Name:     0.40,
			Desc:     0.20,
			Physical: 0.20,
			Graph:    0.10,
			Calc:     0.00,
			Domain:   0.10,
		}
	case SearchTypeRelatedBO:
		return SearchWeights{
			Name:     0.20,
			Desc:     0.10,
			Physical: 0.10,
			Graph:    0.50,
			Calc:     0.00,
			Domain:   0.10,
		}
	case SearchTypeTable:
		return SearchWeights{
			Name:     0.10,
			Desc:     0.10,
			Physical: 0.50,
			Graph:    0.20,
			Calc:     0.00,
			Domain:   0.10,
		}
	case SearchTypeCalc:
		return SearchWeights{
			Name:     0.30,
			Desc:     0.10,
			Physical: 0.00,
			Graph:    0.10,
			Calc:     0.40,
			Domain:   0.10,
		}
	default: // SearchTypeAll
		return SearchWeights{
			Name:     0.25,
			Desc:     0.15,
			Physical: 0.20,
			Graph:    0.15,
			Calc:     0.15,
			Domain:   0.10,
		}
	}
}

// calculateGraphProximity converts hop count to proximity score
func (s *BOSearchService) calculateGraphProximity(hopCount int) float64 {
	switch hopCount {
	case 0:
		return 1.0
	case 1:
		return 0.8
	case 2:
		return 0.6
	default:
		return 0.3
	}
}

// calculateDomainRelevance checks if domain matches
func (s *BOSearchService) calculateDomainRelevance(boDomain, filterDomain string) float64 {
	if filterDomain == "" {
		return 1.0 // No filter
	}
	if strings.EqualFold(boDomain, filterDomain) {
		return 1.0
	}
	return 0.0
}

// getMatchTypePriority returns priority for tie-breaking
func (s *BOSearchService) getMatchTypePriority(matchType string) int {
	switch matchType {
	case "term":
		return 1
	case "calculation":
		return 2
	case "driving_table":
		return 3
	case "related_bo":
		return 4
	default:
		return 5
	}
}
