package survivorship

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ResolveFieldRules loads semantic survivorship rules for entityType and maps
// them onto physical field_cd values via attribute_def.semantic_term_id.
// Fields without a term binding are omitted — callers that require terms must
// treat missing entries as a block/warn decision.
func (s *Service) ResolveFieldRules(ctx context.Context, tenantID uuid.UUID, entityType string) (map[string]mdm.FieldRule, []FieldRuleResolved, error) {
	type row struct {
		SemanticTermID  uuid.UUID      `db:"semantic_term_id"`
		Strategy        string         `db:"strategy"`
		PriorityOrder   pq.StringArray `db:"priority_order"`
		MaxStaleSeconds int            `db:"max_stale_seconds"`
		FieldCd         string         `db:"field_cd"`
	}

	var rows []row
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		// Prefer attribute_def bindings; also accept catalog_node.properties.field_cd
		// for typed master columns seeded as semantic terms without an attribute_def row.
		return tx.SelectContext(ctx, &rows, `
			SELECT DISTINCT ON (field_cd)
			       semantic_term_id, strategy, priority_order, max_stale_seconds, field_cd
			FROM (
				SELECT r.semantic_term_id, r.strategy, r.priority_order, r.max_stale_seconds,
				       a.field_cd, 1 AS pref
				FROM public.semantic_survivorship_rules r
				JOIN public.attribute_def_effective a
				  ON a.semantic_term_id = r.semantic_term_id
				 AND a.entity_type = r.entity_type
				WHERE r.entity_type = $1
				  AND r.is_active
				  AND a.is_active
				  AND a.semantic_term_id IS NOT NULL
				UNION ALL
				SELECT r.semantic_term_id, r.strategy, r.priority_order, r.max_stale_seconds,
				       cn.properties->>'field_cd' AS field_cd, 2 AS pref
				FROM public.semantic_survivorship_rules r
				JOIN public.catalog_node cn ON cn.id = r.semantic_term_id
				WHERE r.entity_type = $1
				  AND r.is_active
				  AND cn.is_active
				  AND COALESCE(cn.properties->>'field_cd', '') <> ''
			) x
			ORDER BY field_cd, pref`, entityType)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resolve survivorship rules: %w", err)
	}

	resolved := make([]FieldRuleResolved, 0, len(rows))
	engineRules := make(map[string]mdm.FieldRule, len(rows))
	for _, r := range rows {
		if r.FieldCd == "" {
			continue
		}
		prio := []string(r.PriorityOrder)
		resolved = append(resolved, FieldRuleResolved{
			FieldCd:        r.FieldCd,
			SemanticTermID: r.SemanticTermID,
			Strategy:       r.Strategy,
			PriorityOrder:  prio,
			MaxStaleSecs:   r.MaxStaleSeconds,
		})
		engineRules[r.FieldCd] = mdm.FieldRule{
			Strategy:        r.Strategy,
			PriorityOrder:   prio,
			MaxStaleSeconds: r.MaxStaleSeconds,
		}
	}
	return engineRules, resolved, nil
}

// ListRulesByTerm returns active rules keyed by semantic_term_id (no field join).
// Useful for mastering typed BO columns that share a term but are not in attribute_def.
func (s *Service) ListRulesByTerm(ctx context.Context, tenantID uuid.UUID, entityType string) (map[uuid.UUID]mdm.FieldRule, error) {
	rules, err := s.ListRules(ctx, tenantID, entityType, false)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]mdm.FieldRule, len(rules))
	for _, r := range rules {
		out[r.SemanticTermID] = mdm.FieldRule{
			Strategy:        r.Strategy,
			PriorityOrder:   []string(r.PriorityOrder),
			MaxStaleSeconds: r.MaxStaleSeconds,
		}
	}
	return out, nil
}
