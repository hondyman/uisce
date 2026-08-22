package bo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LayoutService resolves and caches taxonomy-driven field groupings and subtype facets
type LayoutService struct {
	db        *sqlx.DB
	redis     *redis.Client
	memCache  sync.Map
	cacheTTL  time.Duration
}

type memCacheEntry struct {
	data      *BOLayoutResponse
	expiresAt time.Time
}

// NewLayoutService initializes the LayoutService with DB and optional Redis client
func NewLayoutService(db *sqlx.DB, redisClient *redis.Client) *LayoutService {
	return &LayoutService{
		db:       db,
		redis:    redisClient,
		cacheTTL: 1 * time.Hour,
	}
}

// GetResolvedBOLayout resolves taxonomy-grouped fields for a given BO and subtype facet filter
func (s *LayoutService) GetResolvedBOLayout(ctx context.Context, tenantID, boID uuid.UUID, subtypeFilter string) (*BOLayoutResponse, error) {
	if subtypeFilter == "" {
		subtypeFilter = "ALL"
	}
	cacheKey := fmt.Sprintf("uisce:bo:layout:%s:%s:%s", tenantID, boID, subtypeFilter)

	// 1. Check L1 In-Memory Cache
	if val, ok := s.memCache.Load(cacheKey); ok {
		if entry, ok := val.(memCacheEntry); ok && time.Now().Before(entry.expiresAt) {
			return entry.data, nil
		}
		s.memCache.Delete(cacheKey)
	}

	// 2. Check L2 Redis Cache
	if s.redis != nil {
		if val, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var cached BOLayoutResponse
			if json.Unmarshal(val, &cached) == nil {
				s.memCache.Store(cacheKey, memCacheEntry{data: &cached, expiresAt: time.Now().Add(5 * time.Minute)})
				return &cached, nil
			}
		}
	}

	// 3. Fetch Business Object Core Metadata
	var boMeta struct {
		Key                string  `db:"key"`
		DiscriminatorField *string `db:"discriminator_field"`
		SubtypesConfigJSON []byte  `db:"subtypes_config"`
	}
	boQuery := `
		SELECT key, discriminator_field, COALESCE(subtypes_config, '{}'::jsonb) AS subtypes_config 
		FROM public.business_objects 
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
	`
	if err := s.db.GetContext(ctx, &boMeta, boQuery, boID, tenantID); err != nil {
		// Fallback lookup by key if boID didn't match UUID
		return nil, fmt.Errorf("business object not found: %w", err)
	}

	var subtypes map[string]SubtypeConfig
	if len(boMeta.SubtypesConfigJSON) > 0 {
		_ = json.Unmarshal(boMeta.SubtypesConfigJSON, &subtypes)
	}

	// 4. Resolve Taxonomy Groups via Graph Traversal CTE
	fieldQuery := `
	WITH raw_fields AS (
		SELECT 
			bof.id AS field_id,
			COALESCE(bof.field_key, bof.name, '') AS field_key,
			COALESCE(bof.display_name, bof.name, '') AS display_name,
			COALESCE(bof.field_role, bof.role, 'DIMENSION') AS field_role,
			COALESCE(bof.data_type, bof.type, 'string') AS data_type,
			COALESCE(bof.subtype_scope, 'CORE') AS subtype_scope,
			COALESCE(bof.is_required, false) AS is_required,
			COALESCE(st.properties->>'formula', bof.formula, '') AS formula,
			COALESCE(
				bof.custom_ui_group,
				tax.node_name,
				CASE 
					WHEN COALESCE(bof.subtype_scope, 'CORE') != 'CORE' THEN 'Specific ' || bof.subtype_scope || ' Attributes'
					ELSE 'General Properties'
				END
			) AS group_name,
			COALESCE(tax.node_key, LOWER(REPLACE(COALESCE(bof.subtype_scope, 'CORE'), ' ', '_'))) AS group_key,
			COALESCE((tax.properties->>'sequence')::int, bof.ui_sequence, 100) AS group_seq
		FROM public.business_object_fields bof
		LEFT JOIN public.catalog_node st 
			ON st.node_id = bof.term_node_id AND st.node_type = 'SEMANTIC_TERM'
		LEFT JOIN public.catalog_edge e_bt 
			ON e_bt.from_node_id = st.node_id AND e_bt.edge_type IN ('DEFINED_BY', 'DESCRIBES')
		LEFT JOIN public.catalog_node bt 
			ON bt.node_id = e_bt.to_node_id AND bt.node_type = 'BUSINESS_TERM'
		LEFT JOIN public.catalog_edge e_tax 
			ON e_tax.from_node_id = bt.node_id AND e_tax.edge_type = 'MEMBER_OF'
		LEFT JOIN public.catalog_node tax 
			ON tax.node_id = e_tax.to_node_id AND tax.node_type = 'TAXONOMY_NODE'
		WHERE bof.bo_id = $1 
		  AND (bof.tenant_id = $2 OR bof.tenant_id IS NULL)
		  AND COALESCE(bof.is_active, true) = TRUE
		  AND ($3 = '' OR $3 = 'ALL' OR COALESCE(bof.subtype_scope, 'CORE') = 'CORE' OR bof.subtype_scope = $3)
	)
	SELECT 
		group_key,
		group_name,
		group_seq,
		json_agg(json_build_object(
			'fieldId', field_id,
			'key', field_key,
			'displayName', display_name,
			'role', field_role,
			'dataType', data_type,
			'subtypeScope', subtype_scope,
			'isRequired', is_required,
			'isGoverned', (formula != ''),
			'formula', formula
		) ORDER BY is_required DESC, display_name ASC) AS fields_json
	FROM raw_fields
	GROUP BY group_key, group_name, group_seq
	ORDER BY group_seq ASC, group_name ASC;
	`

	rows, err := s.db.QueryContext(ctx, fieldQuery, boID, tenantID, subtypeFilter)
	if err != nil {
		return nil, fmt.Errorf("failed resolving bo field groupings: %w", err)
	}
	defer rows.Close()

	var groups []FieldGroup
	totalFields := 0

	for rows.Next() {
		var g FieldGroup
		var fieldsBlob []byte
		if err := rows.Scan(&g.GroupKey, &g.GroupName, &g.Sequence, &fieldsBlob); err != nil {
			continue
		}
		if err := json.Unmarshal(fieldsBlob, &g.Fields); err == nil {
			totalFields += len(g.Fields)
			groups = append(groups, g)
		}
	}

	discField := ""
	if boMeta.DiscriminatorField != nil {
		discField = *boMeta.DiscriminatorField
	}

	response := &BOLayoutResponse{
		BOID:               boID,
		BOKey:              boMeta.Key,
		DiscriminatorField: discField,
		Subtypes:           subtypes,
		ActiveSubtype:      subtypeFilter,
		Groups:             groups,
		TotalFields:        totalFields,
	}

	// 5. Store in L1 Memory & L2 Redis Cache
	s.memCache.Store(cacheKey, memCacheEntry{data: response, expiresAt: time.Now().Add(5 * time.Minute)})
	if s.redis != nil {
		if payload, err := json.Marshal(response); err == nil {
			_ = s.redis.Set(ctx, cacheKey, payload, s.cacheTTL).Err()
		}
	}

	return response, nil
}

// InvalidateBOLayoutCache purges the layout cache for a specific BO on schema/term mutation
func (s *LayoutService) InvalidateBOLayoutCache(ctx context.Context, tenantID, boID uuid.UUID) error {
	// Purge L1 memory cache
	s.memCache.Range(func(key, value any) bool {
		kStr := fmt.Sprintf("%v", key)
		targetPrefix := fmt.Sprintf("uisce:bo:layout:%s:%s", tenantID, boID)
		if len(kStr) >= len(targetPrefix) && kStr[:len(targetPrefix)] == targetPrefix {
			s.memCache.Delete(key)
		}
		return true
	})

	// Purge L2 Redis cache
	if s.redis == nil {
		return nil
	}
	pattern := fmt.Sprintf("uisce:bo:layout:%s:%s:*", tenantID, boID)
	keys, err := s.redis.Keys(ctx, pattern).Result()
	if err != nil || len(keys) == 0 {
		return err
	}
	return s.redis.Del(ctx, keys...).Err()
}
