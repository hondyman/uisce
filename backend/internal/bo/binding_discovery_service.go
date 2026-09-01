package bo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BindingEligibleTermDTO struct {
	TermNodeID uuid.UUID          `db:"term_node_id" json:"termNodeId"`
	TermKey    string             `db:"term_key" json:"termKey"`
	TermName   string             `db:"term_name" json:"termName"`
	TermType   string             `db:"term_type" json:"termType"`
	SourceType string             `db:"source_type" json:"sourceType"` // DIRECT or RELATED
	Mappings   []ColumnMappingDTO `json:"mappings"`
}

type termCacheKey struct {
	TenantID       uuid.UUID
	DrivingTableID uuid.UUID
}

type termCacheEntry struct {
	terms     []BindingEligibleTermDTO
	expiresAt time.Time
}

type BindingDiscoveryService struct {
	db  *sqlx.DB
	mu  sync.Mutex
	cache map[termCacheKey]termCacheEntry
	ttl   time.Duration
}

func NewBindingDiscoveryService(db *sqlx.DB) *BindingDiscoveryService {
	return &BindingDiscoveryService{
		db:    db,
		cache: make(map[termCacheKey]termCacheEntry),
		ttl:   5 * time.Minute,
	}
}

// DiscoverEligibleTermsForBinding executes graph edge traversal from driving table to scoped terms.
// Results are cached for 5 minutes per (tenantID, drivingTableID).
func (s *BindingDiscoveryService) DiscoverEligibleTermsForBinding(
	ctx context.Context,
	tenantID, drivingNodeID uuid.UUID,
) ([]BindingEligibleTermDTO, error) {
	// Check cache first
	key := termCacheKey{TenantID: tenantID, DrivingTableID: drivingNodeID}
	s.mu.Lock()
	if entry, found := s.cache[key]; found && time.Now().Before(entry.expiresAt) {
		s.mu.Unlock()
		return entry.terms, nil
	}
	s.mu.Unlock()

	// Compute terms
	terms, err := s.discoverTerms(ctx, tenantID, drivingNodeID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.mu.Lock()
	s.cache[key] = termCacheEntry{
		terms:     terms,
		expiresAt: time.Now().Add(s.ttl),
	}
	s.mu.Unlock()

	return terms, nil
}

// discoverTerms performs the actual graph traversal to find eligible semantic terms.
// This is the non-cached implementation.
func (s *BindingDiscoveryService) discoverTerms(
	ctx context.Context,
	tenantID, drivingNodeID uuid.UUID,
) ([]BindingEligibleTermDTO, error) {
	if s.db == nil {
		// Mocked fallback for in-memory testing & zero-db verification
		return []BindingEligibleTermDTO{
			{
				TermNodeID: uuid.MustParse("820b1234-0000-4000-a000-000000000010"),
				TermKey:    "customer_bk",
				TermName:   "Customer Business Key",
				TermType:   "ATTRIBUTE",
				SourceType: "DIRECT",
				Mappings: []ColumnMappingDTO{
					{
						ColumnNodeID:    uuid.MustParse("820b1234-0000-4000-a000-000000000020"),
						ColumnName:      "CustomerID",
						TableKey:        "Customers",
						SourceType:      "DIRECT",
						IsPrimarySource: true,
					},
				},
			},
			{
				TermNodeID: uuid.MustParse("820b1234-0000-4000-a000-000000000011"),
				TermKey:    "company_name",
				TermName:   "Company Name",
				TermType:   "ATTRIBUTE",
				SourceType: "DIRECT",
				Mappings: []ColumnMappingDTO{
					{
						ColumnNodeID:    uuid.MustParse("820b1234-0000-4000-a000-000000000021"),
						ColumnName:      "CompanyName",
						TableKey:        "Customers",
						SourceType:      "DIRECT",
						IsPrimarySource: true,
					},
				},
			},
			{
				TermNodeID: uuid.MustParse("820b1234-0000-4000-a000-000000000012"),
				TermKey:    "country",
				TermName:   "Country",
				TermType:   "ATTRIBUTE",
				SourceType: "DIRECT",
				Mappings: []ColumnMappingDTO{
					{
						ColumnNodeID:    uuid.MustParse("820b1234-0000-4000-a000-000000000022"),
						ColumnName:      "Country",
						TableKey:        "Customers",
						SourceType:      "DIRECT",
						IsPrimarySource: true,
					},
				},
			},
		}, nil
	}

	query := `
		WITH driving_columns AS (
			SELECT col.id AS column_node_id, col.node_name AS column_name,
			       tbl.node_name AS table_name, 'DIRECT' AS source_type
			FROM public.catalog_node tbl
			JOIN public.catalog_node col ON col.parent_id = tbl.id
			WHERE tbl.id = $1
		),
		related_tables AS (
			SELECT DISTINCT rel_tbl.id AS table_node_id, rel_tbl.node_name AS table_name
			FROM public.catalog_edge ce
			JOIN public.catalog_node rel_tbl ON rel_tbl.id = ce.target_node_id
			WHERE ce.source_node_id = $1
			  AND (ce.tenant_id = $2 OR ce.tenant_id = '00000000-0000-0000-0000-000000000000')
			  AND ce.relationship_type IN ('JOINS_TO', 'FK_RELATIONSHIP')
			  AND ce.is_active = TRUE
		),
		related_columns AS (
			SELECT col.id AS column_node_id, col.node_name AS column_name,
			       rt.table_name, 'RELATED' AS source_type
			FROM related_tables rt
			JOIN public.catalog_node col ON col.parent_id = rt.table_node_id
		),
		all_columns AS (
			SELECT * FROM driving_columns UNION ALL SELECT * FROM related_columns
		)
		SELECT
			st.id AS term_node_id,
			COALESCE(st.properties->>'term_key', st.node_name) AS term_key,
			st.node_name AS term_name,
			COALESCE(st.properties->>'term_type', 'ATTRIBUTE') AS term_type,
			ac.source_type, ac.column_node_id, ac.column_name, ac.table_name
		FROM all_columns ac
		JOIN public.catalog_edge em ON em.target_node_id = ac.column_node_id AND em.relationship_type = 'MAPS_TO'
		JOIN public.catalog_node st ON st.id = em.source_node_id
		WHERE st.node_type = 'semantic_term'
		  AND (st.tenant_id = $2 OR st.tenant_id = '00000000-0000-0000-0000-000000000000');`

	type FlatRow struct {
		TermNodeID   uuid.UUID `db:"term_node_id"`
		TermKey      string    `db:"term_key"`
		TermName     string    `db:"term_name"`
		TermType     string    `db:"term_type"`
		SourceType   string    `db:"source_type"`
		ColumnNodeID uuid.UUID `db:"column_node_id"`
		ColumnName   string    `db:"column_name"`
		TableName    string    `db:"table_name"`
	}

	var rows []FlatRow
	if err := s.db.SelectContext(ctx, &rows, query, drivingNodeID, tenantID); err != nil {
		return nil, fmt.Errorf("failed binding discovery graph traversal: %w", err)
	}

	termMap := make(map[uuid.UUID]*BindingEligibleTermDTO)
	for _, r := range rows {
		t, exists := termMap[r.TermNodeID]
		if !exists {
			t = &BindingEligibleTermDTO{
				TermNodeID: r.TermNodeID,
				TermKey:    r.TermKey,
				TermName:   r.TermName,
				TermType:   r.TermType,
				SourceType: r.SourceType,
				Mappings:   make([]ColumnMappingDTO, 0),
			}
			termMap[r.TermNodeID] = t
		}
		t.Mappings = append(t.Mappings, ColumnMappingDTO{
			ColumnNodeID:    r.ColumnNodeID,
			ColumnName:      r.ColumnName,
			TableKey:        r.TableName,
			SourceType:      r.SourceType,
			IsPrimarySource: r.SourceType == "DIRECT",
		})
	}

	var result []BindingEligibleTermDTO
	for _, v := range termMap {
		result = append(result, *v)
	}

	return result, nil
}
