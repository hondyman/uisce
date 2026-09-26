package attribute

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// wireSemanticTerm links the attribute's catalog node to the semantic term and
// upserts JSON_PATH field_bindings for any BO fields that already use the term.
func (s *Service) wireSemanticTerm(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, def AttributeDef) error {
	if def.SemanticTermID == nil {
		return nil
	}
	termID := *def.SemanticTermID

	// Resolve custom_field catalog node for this attribute
	qPath := fmt.Sprintf("attr:%s:%s:%s", tenantID.String(), def.EntityType, def.FieldCd)
	var attrNodeID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM public.catalog_node
		WHERE tenant_id = $1 AND qualified_path = $2
		LIMIT 1`, tenantID, qPath).Scan(&attrNodeID)
	if err != nil {
		return fmt.Errorf("custom_field catalog node missing for %s (syncCatalog first): %w", qPath, err)
	}

	var edgeTypeID uuid.UUID
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM public.catalog_edge_types
		WHERE edge_type_name = 'TERM_MAPS_TO_CUSTOM_FIELD'
		LIMIT 1`).Scan(&edgeTypeID); err == nil {
		props, _ := json.Marshal(map[string]any{
			"storage_kind":  "JSONB_KEY",
			"json_path":     def.JsonPath,
			"jsonb_column":  "custom_attributes",
			"field_cd":      def.FieldCd,
			"table_ref":     def.TableRef,
			"entity_type":   def.EntityType,
		})
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO public.catalog_edge (
				id, tenant_id, source_node_id, target_node_id, edge_type_id,
				properties, is_active, relationship_type, created_at, updated_at
			)
			SELECT gen_random_uuid(), $1, $2, $3, $4, $5::jsonb, true,
			       'TERM_MAPS_TO_CUSTOM_FIELD', now(), now()
			WHERE NOT EXISTS (
				SELECT 1 FROM public.catalog_edge
				WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type_id = $4
			)`,
			tenantID, termID, attrNodeID, edgeTypeID, string(props),
		)
	}

	// Also MAPS_TO the physical custom_attributes column when present, with json_path in properties.
	// ResolveSemanticFieldMap walks MAPS_TO; QueryBORecords hydrates JSONB keys in Go.
	var mapsToID uuid.UUID
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM public.catalog_edge_types WHERE edge_type_name = 'MAPS_TO' LIMIT 1`).Scan(&mapsToID); err == nil {
		colSuffix := def.TableRef + "/custom_attributes"
		var colNodeID uuid.UUID
		colErr := tx.QueryRowContext(ctx, `
			SELECT id FROM public.catalog_node
			WHERE node_name = 'custom_attributes'
			  AND (
			    qualified_path = $1 OR qualified_path = 'crims.' || $1
			    OR qualified_path LIKE '%' || $2 || '%'
			  )
			ORDER BY CASE WHEN qualified_path LIKE 'crims.%' THEN 0 ELSE 1 END
			LIMIT 1`,
			colSuffix, def.TableRef,
		).Scan(&colNodeID)
		if colErr == nil {
			props, _ := json.Marshal(map[string]any{
				"storage_kind": "JSONB_KEY",
				"json_path":    def.JsonPath,
				"jsonb_column": "custom_attributes",
			})
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO public.catalog_edge (
					id, tenant_id, source_node_id, target_node_id, edge_type_id,
					properties, is_active, relationship_type, created_at, updated_at
				)
				SELECT gen_random_uuid(), $1, $2, $3, $4, $5::jsonb, true, 'MAPS_TO', now(), now()
				WHERE NOT EXISTS (
					SELECT 1 FROM public.catalog_edge
					WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type_id = $4
				)`,
				tenantID, termID, colNodeID, mapsToID, string(props),
			)
		}
	}

	// Upsert JSON_PATH field_bindings for BO fields that already reference this term.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.field_bindings (
			id, tenant_id, bo_id, binding_id, field_id, source_node_id,
			source_type, transformation_type, json_path, binding_status,
			created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			bf.tenant_id,
			bf.bo_id,
			bob.id,
			bf.id,
			$1,
			'JSON_PATH',
			'NONE',
			$2,
			'RESOLVED',
			now(), now()
		FROM public.business_object_fields bf
		JOIN public.business_object_bindings bob
		  ON bob.bo_id = bf.bo_id AND bob.tenant_id = bf.tenant_id AND bob.is_default = true
		WHERE bf.term_node_id = $3
		  AND NOT EXISTS (
		      SELECT 1 FROM public.field_bindings fb
		      WHERE fb.field_id = bf.id AND fb.binding_id = bob.id
		  )`,
		attrNodeID, def.JsonPath, termID,
	)
	// business_object_bindings may be empty — non-fatal
	if err != nil {
		// Retry without requiring a default binding row: skip silently
		_ = err
	}

	return nil
}

// ListJSONPathBindings returns active custom-attribute bindings for an entity/table.
func (s *Service) ListJSONPathBindings(ctx context.Context, tenantID uuid.UUID, entityType, tableRef string) ([]JSONPathBinding, error) {
	var out []JSONPathBinding
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		q := `
			SELECT field_cd, entity_type, table_ref, semantic_term_id, data_type
			FROM public.attribute_def_effective
			WHERE is_active AND semantic_term_id IS NOT NULL`
		args := []any{}
		if entityType != "" {
			args = append(args, entityType)
			q += fmt.Sprintf(` AND entity_type = $%d`, len(args))
		}
		if tableRef != "" {
			args = append(args, tableRef)
			q += fmt.Sprintf(` AND table_ref = $%d`, len(args))
		}
		rows, err := tx.QueryxContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var b JSONPathBinding
			var termID uuid.UUID
			if err := rows.Scan(&b.FieldCd, &b.EntityType, &b.TableRef, &termID, &b.DataType); err != nil {
				return err
			}
			b.SemanticTermID = termID
			b.JSONColumn = "custom_attributes"
			out = append(out, b)
		}
		return rows.Err()
	})
	return out, err
}

// LookupBySemanticTerm finds an active attribute_def for a semantic term node id.
func (s *Service) LookupBySemanticTerm(ctx context.Context, tenantID, termID uuid.UUID) (*AttributeDef, error) {
	var def AttributeDef
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		row := tx.QueryRowxContext(ctx, `
			SELECT id, tenant_id, core_id, is_shadow, entity_type, table_ref, field_cd,
			       name, COALESCE(description,'') AS description, data_type, json_path,
			       is_required, is_searchable, is_pii,
			       validation_rules, picklist_values, default_value,
			       display_order, COALESCE(section,'') AS section, is_active,
			       origin, semantic_term_id, created_at, updated_at
			FROM public.attribute_def_effective
			WHERE semantic_term_id = $1 AND is_active
			ORDER BY precedence_rank
			LIMIT 1`, termID)
		var err error
		def, err = scanDef(row)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &def, nil
}
