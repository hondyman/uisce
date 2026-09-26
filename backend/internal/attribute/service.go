package attribute

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound      = errors.New("attribute definition not found")
	ErrInvalidInput  = errors.New("invalid attribute input")
	ErrImmutableCode = errors.New("field_cd is immutable")
)

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) withTenant(ctx context.Context, tenantID uuid.UUID, fn func(tx *sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true)`,
		tenantID.String(),
	); err != nil {
		return fmt.Errorf("set tenant guc: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID, entityType string, includeInactive bool) ([]AttributeDef, error) {
	var out []AttributeDef
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		q := `
			SELECT id, tenant_id, core_id, is_shadow, entity_type, table_ref, field_cd,
			       name, COALESCE(description, '') AS description, data_type, json_path,
			       is_required, is_searchable, is_pii,
			       validation_rules, picklist_values, default_value,
			       display_order, COALESCE(section, '') AS section, is_active,
			       origin, semantic_term_id, created_at, updated_at
			FROM public.attribute_def_effective
			WHERE entity_type = $1`
		if !includeInactive {
			q += ` AND is_active`
		}
		q += ` ORDER BY display_order, field_cd`

		rows, err := tx.QueryxContext(ctx, q, entityType)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			def, err := scanDef(rows)
			if err != nil {
				return err
			}
			out = append(out, def)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*AttributeDef, error) {
	var def AttributeDef
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		row := tx.QueryRowxContext(ctx, `
			SELECT id, tenant_id, core_id, is_shadow, entity_type, table_ref, field_cd,
			       name, COALESCE(description, '') AS description, data_type, json_path,
			       is_required, is_searchable, is_pii,
			       validation_rules, picklist_values, default_value,
			       display_order, COALESCE(section, '') AS section, is_active,
			       COALESCE(origin, 'CUSTOM') AS origin, semantic_term_id, created_at, updated_at
			FROM public.attribute_def_effective
			WHERE id = $1`, id)
		var err error
		def, err = scanDef(row)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &def, nil
}

func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, in CreateInput) (*AttributeDef, error) {
	if strings.TrimSpace(in.EntityType) == "" || strings.TrimSpace(in.TableRef) == "" || strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: entity_type, table_ref, and name are required", ErrInvalidInput)
	}
	if _, _, err := sanitizeTableRef(in.TableRef); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	fieldCd, err := normalizeFieldCd(in.Name, in.FieldCd)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	dataType := strings.ToLower(strings.TrimSpace(in.DataType))
	if dataType == "" {
		dataType = "string"
	}
	jsonPath := strings.TrimSpace(in.JsonPath)
	if jsonPath == "" {
		jsonPath = fieldCd
	}
	searchable := true
	if in.IsSearchable != nil {
		searchable = *in.IsSearchable
	}
	vr, _ := json.Marshal(in.ValidationRules)
	if in.ValidationRules == nil {
		vr = []byte(`{}`)
	}
	pl, _ := json.Marshal(in.PicklistValues)

	var created AttributeDef
	err = s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		row := tx.QueryRowxContext(ctx, `
			INSERT INTO public.attribute_def (
				tenant_id, entity_type, table_ref, field_cd, name, description, data_type,
				json_path, is_required, is_searchable, is_pii, validation_rules, picklist_values,
				default_value, display_order, section, semantic_term_id
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,
				$8,$9,$10,$11,$12::jsonb,$13::jsonb,
				$14,$15,$16,$17
			)
			RETURNING id, tenant_id, core_id, is_shadow, entity_type, table_ref, field_cd,
			          name, COALESCE(description,'') AS description, data_type, json_path,
			          is_required, is_searchable, is_pii,
			          validation_rules, picklist_values, default_value,
			          display_order, COALESCE(section,'') AS section, is_active,
			          semantic_term_id, created_at, updated_at`,
			tenantID, strings.ToUpper(in.EntityType), in.TableRef, fieldCd, in.Name, in.Description, dataType,
			jsonPath, in.IsRequired, searchable, in.IsPII, string(vr), nullJSON(pl),
			in.DefaultValue, in.DisplayOrder, in.Section, in.SemanticTermID,
		)
		def, err := scanDefReturning(row)
		if err != nil {
			return err
		}
		def.Origin = "CUSTOM"
		if err := s.syncCatalog(ctx, tx, tenantID, def); err != nil {
			return err
		}
		if def.SemanticTermID != nil {
			if err := s.wireSemanticTerm(ctx, tx, tenantID, def); err != nil {
				return err
			}
		}
		created = def
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, in UpdateInput) (*AttributeDef, error) {
	var updated AttributeDef
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		existing, err := s.getOwned(ctx, tx, tenantID, id)
		if err != nil {
			return err
		}
		if in.Name != nil {
			existing.Name = *in.Name
		}
		if in.Description != nil {
			existing.Description = *in.Description
		}
		if in.DataType != nil {
			existing.DataType = strings.ToLower(*in.DataType)
		}
		if in.IsRequired != nil {
			existing.IsRequired = *in.IsRequired
		}
		if in.IsSearchable != nil {
			existing.IsSearchable = *in.IsSearchable
		}
		if in.IsPII != nil {
			existing.IsPII = *in.IsPII
		}
		if in.ValidationRules != nil {
			existing.ValidationRules = *in.ValidationRules
		}
		if in.PicklistValues != nil {
			existing.PicklistValues = *in.PicklistValues
		}
		if in.DefaultValue != nil {
			existing.DefaultValue = in.DefaultValue
		}
		if in.DisplayOrder != nil {
			existing.DisplayOrder = *in.DisplayOrder
		}
		if in.Section != nil {
			existing.Section = *in.Section
		}
		if in.ClearSemanticTerm {
			existing.SemanticTermID = nil
		} else if in.SemanticTermID != nil {
			existing.SemanticTermID = in.SemanticTermID
		}

		vr, _ := json.Marshal(existing.ValidationRules)
		if existing.ValidationRules == nil {
			vr = []byte(`{}`)
		}
		pl, _ := json.Marshal(existing.PicklistValues)

		row := tx.QueryRowxContext(ctx, `
			UPDATE public.attribute_def SET
				name = $1,
				description = $2,
				data_type = $3,
				is_required = $4,
				is_searchable = $5,
				is_pii = $6,
				validation_rules = $7::jsonb,
				picklist_values = $8::jsonb,
				default_value = $9,
				display_order = $10,
				section = $11,
				semantic_term_id = $12,
				updated_at = now()
			WHERE id = $13 AND tenant_id = $14
			RETURNING id, tenant_id, core_id, is_shadow, entity_type, table_ref, field_cd,
			          name, COALESCE(description,'') AS description, data_type, json_path,
			          is_required, is_searchable, is_pii,
			          validation_rules, picklist_values, default_value,
			          display_order, COALESCE(section,'') AS section, is_active,
			          semantic_term_id, created_at, updated_at`,
			existing.Name, existing.Description, existing.DataType,
			existing.IsRequired, existing.IsSearchable, existing.IsPII,
			string(vr), nullJSON(pl), existing.DefaultValue,
			existing.DisplayOrder, existing.Section, existing.SemanticTermID, id, tenantID,
		)
		def, err := scanDefReturning(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		def.Origin = "CUSTOM"
		if err := s.syncCatalog(ctx, tx, tenantID, def); err != nil {
			return err
		}
		if def.SemanticTermID != nil {
			if err := s.wireSemanticTerm(ctx, tx, tenantID, def); err != nil {
				return err
			}
		}
		updated = def
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		existing, err := s.getOwned(ctx, tx, tenantID, id)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `
			UPDATE public.attribute_def
			SET is_active = false, updated_at = now()
			WHERE id = $1 AND tenant_id = $2`, id, tenantID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrNotFound
		}
		existing.IsActive = false
		return s.syncCatalog(ctx, tx, tenantID, existing)
	})
}

func (s *Service) getOwned(ctx context.Context, tx *sqlx.Tx, tenantID, id uuid.UUID) (AttributeDef, error) {
	row := tx.QueryRowxContext(ctx, `
		SELECT id, tenant_id, core_id, is_shadow, entity_type, table_ref, field_cd,
		       name, COALESCE(description,'') AS description, data_type, json_path,
		       is_required, is_searchable, is_pii,
		       validation_rules, picklist_values, default_value,
		       display_order, COALESCE(section,'') AS section, is_active,
		       semantic_term_id, created_at, updated_at
		FROM public.attribute_def
		WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	def, err := scanDefReturning(row)
	if errors.Is(err, sql.ErrNoRows) {
		return AttributeDef{}, ErrNotFound
	}
	def.Origin = "CUSTOM"
	return def, err
}

func (s *Service) syncCatalog(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, def AttributeDef) error {
	var nodeTypeID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM public.catalog_node_types
		WHERE catalog_type_name = 'custom_field' AND is_active
		ORDER BY created_at NULLS LAST
		LIMIT 1`).Scan(&nodeTypeID)
	if err != nil {
		// Catalog types not seeded yet — definition still saved.
		return nil
	}

	props := map[string]any{
		"entity_type":   def.EntityType,
		"table_ref":     def.TableRef,
		"field_cd":      def.FieldCd,
		"data_type":     def.DataType,
		"json_path":     def.JsonPath,
		"is_required":   def.IsRequired,
		"is_searchable": def.IsSearchable,
		"is_active":     def.IsActive,
		"section":       def.Section,
		"origin":        def.Origin,
	}
	propsJSON, _ := json.Marshal(props)
	qPath := fmt.Sprintf("attr:%s:%s:%s", tenantID.String(), def.EntityType, def.FieldCd)

	var nodeID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO public.catalog_node (
			id, tenant_id, node_type_id, node_name, description, properties,
			qualified_path, is_active, node_type, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5::jsonb,
			$6, $7, 'custom_field', now(), now()
		)
		ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
			node_name = EXCLUDED.node_name,
			description = EXCLUDED.description,
			properties = EXCLUDED.properties,
			is_active = EXCLUDED.is_active,
			updated_at = now()
		RETURNING id`,
		tenantID, nodeTypeID, def.Name, def.Description, string(propsJSON),
		qPath, def.IsActive,
	).Scan(&nodeID)
	if err != nil {
		return fmt.Errorf("upsert catalog node: %w", err)
	}

	var edgeTypeID uuid.UUID
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM public.catalog_edge_types
		WHERE edge_type_name = 'BO_HAS_ATTRIBUTE' LIMIT 1`).Scan(&edgeTypeID); err == nil {
		// Link from table node when present
		var tableNodeID uuid.UUID
		tableErr := tx.QueryRowContext(ctx, `
			SELECT id FROM public.catalog_node
			WHERE tenant_id = $1
			  AND (
			    qualified_path = $2
			    OR qualified_path = 'crims.' || $2
			    OR qualified_path LIKE '%.' || $2
			  )
			  AND (node_type ILIKE 'table' OR node_type_id IN (
			      SELECT id FROM public.catalog_node_types WHERE catalog_type_name = 'table'
			  ))
			ORDER BY CASE WHEN qualified_path LIKE 'crims.%' THEN 0 ELSE 1 END
			LIMIT 1`,
			tenantID, def.TableRef,
		).Scan(&tableNodeID)
		if tableErr == nil {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO public.catalog_edge (
					id, tenant_id, source_node_id, target_node_id, edge_type_id,
					properties, is_active, relationship_type, created_at, updated_at
				)
				SELECT gen_random_uuid(), $1, $2, $3, $4,
				       $5::jsonb, true, 'BO_HAS_ATTRIBUTE', now(), now()
				WHERE NOT EXISTS (
					SELECT 1 FROM public.catalog_edge
					WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type_id = $4
				)`,
				tenantID, tableNodeID, nodeID, edgeTypeID,
				fmt.Sprintf(`{"is_core":%v}`, def.Origin == "CORE"),
			)
		}
	}

	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM public.catalog_edge_types
		WHERE edge_type_name = 'ATTRIBUTE_STORED_IN' LIMIT 1`).Scan(&edgeTypeID); err == nil {
		colSuffix := def.TableRef + "/custom_attributes"
		var colNodeID uuid.UUID
		colErr := tx.QueryRowContext(ctx, `
			SELECT id FROM public.catalog_node
			WHERE tenant_id = $1
			  AND (
			    qualified_path = $2
			    OR qualified_path = 'crims.' || $2
			    OR qualified_path LIKE '%/' || 'custom_attributes'
			       AND (qualified_path LIKE '%' || $3 || '%')
			  )
			  AND node_name = 'custom_attributes'
			ORDER BY CASE WHEN qualified_path LIKE 'crims.%' THEN 0 ELSE 1 END
			LIMIT 1`,
			tenantID, colSuffix, def.TableRef,
		).Scan(&colNodeID)
		if colErr == nil {
			storeProps, _ := json.Marshal(map[string]any{
				"storage_kind": "JSONB_KEY",
				"json_path":    def.JsonPath,
			})
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO public.catalog_edge (
					id, tenant_id, source_node_id, target_node_id, edge_type_id,
					properties, is_active, relationship_type, created_at, updated_at
				)
				SELECT gen_random_uuid(), $1, $2, $3, $4,
				       $5::jsonb, true, 'ATTRIBUTE_STORED_IN', now(), now()
				WHERE NOT EXISTS (
					SELECT 1 FROM public.catalog_edge
					WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type_id = $4
				)`,
				tenantID, nodeID, colNodeID, edgeTypeID, string(storeProps),
			)
		}
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDef(row interface {
	Scan(dest ...any) error
}) (AttributeDef, error) {
	var d AttributeDef
	var vr, pl []byte
	var origin sql.NullString
	err := row.Scan(
		&d.ID, &d.TenantID, &d.CoreID, &d.IsShadow, &d.EntityType, &d.TableRef, &d.FieldCd,
		&d.Name, &d.Description, &d.DataType, &d.JsonPath,
		&d.IsRequired, &d.IsSearchable, &d.IsPII,
		&vr, &pl, &d.DefaultValue,
		&d.DisplayOrder, &d.Section, &d.IsActive,
		&origin, &d.SemanticTermID, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return d, err
	}
	_ = json.Unmarshal(vr, &d.ValidationRules)
	if d.ValidationRules == nil {
		d.ValidationRules = map[string]any{}
	}
	if len(pl) > 0 && string(pl) != "null" {
		_ = json.Unmarshal(pl, &d.PicklistValues)
	}
	if origin.Valid {
		d.Origin = origin.String
	}
	return d, nil
}

func scanDefReturning(row interface {
	Scan(dest ...any) error
}) (AttributeDef, error) {
	var d AttributeDef
	var vr, pl []byte
	err := row.Scan(
		&d.ID, &d.TenantID, &d.CoreID, &d.IsShadow, &d.EntityType, &d.TableRef, &d.FieldCd,
		&d.Name, &d.Description, &d.DataType, &d.JsonPath,
		&d.IsRequired, &d.IsSearchable, &d.IsPII,
		&vr, &pl, &d.DefaultValue,
		&d.DisplayOrder, &d.Section, &d.IsActive,
		&d.SemanticTermID, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return d, err
	}
	_ = json.Unmarshal(vr, &d.ValidationRules)
	if d.ValidationRules == nil {
		d.ValidationRules = map[string]any{}
	}
	if len(pl) > 0 && string(pl) != "null" {
		_ = json.Unmarshal(pl, &d.PicklistValues)
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = d.CreatedAt
	}
	return d, nil
}

func nullJSON(b []byte) any {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return string(b)
}
