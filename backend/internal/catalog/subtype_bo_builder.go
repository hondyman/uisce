package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type SubtypeBOBuilder struct {
	loader SubtypeRegistryLoader
}

func NewSubtypeBOBuilder(loader SubtypeRegistryLoader) *SubtypeBOBuilder {
	return &SubtypeBOBuilder{loader: loader}
}

func parentKey(rootObject string) string {
	switch rootObject {
	case "account", "position", "security", "trade_order":
		return "oms." + rootObject
	case "alternative_investment":
		return "altinv.alternative_investment"
	case "settlement":
		return "cash_flow.settlement"
	case "customer", "vendor", "personnel", "sales_ledger":
		return "master." + rootObject
	default:
		return rootObject
	}
}

func parentTableName(rootObject string) string {
	switch rootObject {
	case "account", "position", "security", "trade_order":
		return "oms." + rootObject
	case "alternative_investment":
		return "altinv.alternative_investment"
	case "settlement":
		return "cash_flow.settlement"
	case "customer", "vendor", "personnel", "sales_ledger":
		return "master." + rootObject
	default:
		return rootObject
	}
}

func (b *SubtypeBOBuilder) BuildForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) error {
	rows, err := b.loader.LoadAllForTenant(ctx, db, tenantID)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	boTypeID := "06bb774c-8666-4ab1-84eb-4f4d439ac84c"

	seenParents := make(map[string]string)
	for _, row := range rows {
		pKey := parentKey(row.RootObject)
		if seenParents[pKey] != "" {
			continue
		}
		seenParents[pKey] = row.RootObject

		clsNodeID := uuid.New()
		bkNodeID := uuid.New()
		semNodeID := uuid.New()
		grainNodeID := uuid.New()

		_, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, true)
			ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET updated_at = NOW()
		`, clsNodeID, tenantID, boTypeID, pKey,
			"business_object/"+pKey+"/classification",
			fmt.Sprintf(`{"role":"classification","bo_key":"%s"}`, pKey))
		if err != nil {
			return fmt.Errorf("failed upserting classification node for %s: %w", pKey, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, true)
			ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET updated_at = NOW()
		`, bkNodeID, tenantID, boTypeID, pKey+" (business key)",
			"business_object/"+pKey+"/business_key",
			fmt.Sprintf(`{"role":"business_key","bo_key":"%s"}`, pKey))
		if err != nil {
			return fmt.Errorf("failed upserting business_key node for %s: %w", pKey, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, true)
			ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET updated_at = NOW()
		`, semNodeID, tenantID, boTypeID, pKey+" (semantic id)",
			"business_object/"+pKey+"/semantic_id",
			fmt.Sprintf(`{"role":"semantic_id","bo_key":"%s"}`, pKey))
		if err != nil {
			return fmt.Errorf("failed upserting semantic_id node for %s: %w", pKey, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, true)
			ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET updated_at = NOW()
		`, grainNodeID, tenantID, boTypeID, pKey+" (grain)",
			"business_object/"+pKey+"/grain",
			fmt.Sprintf(`{"role":"grain","bo_key":"%s"}`, pKey))
		if err != nil {
			return fmt.Errorf("failed upserting grain node for %s: %w", pKey, err)
		}

		_ = strings.Title(row.RootObject) // display_name future use

		_, err = tx.ExecContext(ctx, `
			INSERT INTO business_objects
				(id, tenant_id, model_id, bo_key, bo_name, bo_type,
				 classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
				 sti_discriminator_column, active_subtype_filter,
				 is_active, is_core, description,
				 created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 'ENTITY',
			        $6, $7, $8, $9,
			        'subtype_code', $4,
			        true, true, $10,
			        NOW(), NOW())
			ON CONFLICT (tenant_id, bo_key) DO UPDATE
				SET bo_name               = EXCLUDED.bo_name,
				    classification_node_id = EXCLUDED.classification_node_id,
				    business_key_node_id  = EXCLUDED.business_key_node_id,
				    semantic_id_node_id   = EXCLUDED.semantic_id_node_id,
				    grain_node_id         = EXCLUDED.grain_node_id,
				    sti_discriminator_column  = EXCLUDED.sti_discriminator_column,
				    active_subtype_filter     = EXCLUDED.active_subtype_filter,
				    is_active  = EXCLUDED.is_active,
				    is_core    = EXCLUDED.is_core,
				    updated_at = NOW()
		`, uuid.New(), tenantID, clsNodeID, pKey, row.RootObject,
			clsNodeID, bkNodeID, semNodeID, grainNodeID,
			"STI root object for "+row.RootObject+" — subtypes managed via oms.subtype_registry")
		if err != nil {
			return fmt.Errorf("failed upserting parent BO %s: %w", pKey, err)
		}
	}

	// -----------------------------------------------------------------------
	// [REGRESSION FIX] Populate CoreFields for each parent BO.
	// Parent BOs were previously left with no business_object_fields entries.
	// The fix: for each parent, compute the union of all field names across
	// its child subtypes (from field_allowlist) and insert them as parent
	// fields with inherits_defaults=true and subtype_scope=NULL.
	// -----------------------------------------------------------------------
	parentFieldSets := make(map[string]map[string]bool)
	for _, row := range rows {
		pKey := parentKey(row.RootObject)
		if parentFieldSets[pKey] == nil {
			parentFieldSets[pKey] = make(map[string]bool)
		}
		for _, f := range row.FieldAllowlist {
			parentFieldSets[pKey][f] = true
		}
	}

	for pKey, fields := range parentFieldSets {
		var parentBOID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT id::text FROM business_objects
			WHERE tenant_id = $1 AND bo_key = $2
			LIMIT 1
		`, tenantID, pKey).Scan(&parentBOID)

		if parentBOID == nil {
			continue
		}

		for fieldName := range fields {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO business_object_fields
					(id, tenant_id, bo_id, term_node_id, field_name, field_role,
					 aggregation_type, binding_requirement, eligibility_source,
					 subtype_scope, is_exposed, inherits_defaults,
					 created_at, updated_at)
				VALUES ($1, $2, $3, NULL, $4, 'DIMENSION',
				        'NONE', 'REQUIRED', 'DIRECT',
				        NULL, true, true,
				        NOW(), NOW())
				ON CONFLICT (tenant_id, bo_id, field_name) DO UPDATE
					SET inherits_defaults = true,
					    subtype_scope    = NULL,
					    field_role       = EXCLUDED.field_role,
					    updated_at       = NOW()
			`, uuid.New(), tenantID, *parentBOID, fieldName)
			if err != nil {
				return fmt.Errorf("failed upserting parent BO field %s.%s: %w", pKey, fieldName, err)
			}
		}
	}

	for _, row := range rows {
		pKey := parentKey(row.RootObject)
		childKey := pKey + "/" + row.SubtypeCode

		var parentBOID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT id::text FROM business_objects
			WHERE tenant_id = $1 AND bo_key = $2
			LIMIT 1
		`, tenantID, pKey).Scan(&parentBOID)

		var parentModelID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT model_id::text FROM business_objects
			WHERE tenant_id = $1 AND bo_key = $2
			LIMIT 1
		`, tenantID, pKey).Scan(&parentModelID)

		var clsNodeID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT id::text FROM catalog_node
			WHERE tenant_id = $1 AND qualified_path = $2
			LIMIT 1
		`, tenantID, "business_object/"+pKey+"/classification").Scan(&clsNodeID)

		var bkNodeID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT id::text FROM catalog_node
			WHERE tenant_id = $1 AND qualified_path = $2
			LIMIT 1
		`, tenantID, "business_object/"+pKey+"/business_key").Scan(&bkNodeID)

		var semNodeID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT id::text FROM catalog_node
			WHERE tenant_id = $1 AND qualified_path = $2
			LIMIT 1
		`, tenantID, "business_object/"+pKey+"/semantic_id").Scan(&semNodeID)

		var grainNodeID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT id::text FROM catalog_node
			WHERE tenant_id = $1 AND qualified_path = $2
			LIMIT 1
		`, tenantID, "business_object/"+pKey+"/grain").Scan(&grainNodeID)

		_, err := tx.ExecContext(ctx, `
			INSERT INTO business_objects
				(id, tenant_id, model_id, bo_key, bo_name, bo_type,
				 classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
				 sti_discriminator_column, active_subtype_filter,
				 is_active, is_core, description,
				 created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 'ENTITY',
			        $6, $7, $8, $9,
			        'subtype_code', $10,
			        true, false, $11,
			        NOW(), NOW())
			ON CONFLICT (tenant_id, bo_key) DO UPDATE
				SET bo_name               = EXCLUDED.bo_name,
				    classification_node_id = EXCLUDED.classification_node_id,
				    business_key_node_id  = EXCLUDED.business_key_node_id,
				    semantic_id_node_id   = EXCLUDED.semantic_id_node_id,
				    grain_node_id         = EXCLUDED.grain_node_id,
				    is_core               = EXCLUDED.is_core,
				    updated_at            = NOW()
		`, uuid.New(), tenantID, clsNodeID, childKey, row.SubtypeCode,
			clsNodeID, bkNodeID, semNodeID, grainNodeID,
			pKey,
			"STI subtype "+row.SubtypeCode+" of "+row.RootObject)
		if err != nil {
			return fmt.Errorf("failed upserting child BO %s: %w", childKey, err)
		}

		var childBOID *string
		_ = tx.QueryRowContext(ctx, `
			SELECT id::text FROM business_objects
			WHERE tenant_id = $1 AND bo_key = $2
			LIMIT 1
		`, tenantID, childKey).Scan(&childBOID)

		if childBOID != nil && len(row.FieldAllowlist) > 0 {
			for _, fieldName := range row.FieldAllowlist {
				var fieldTermID *string
				_ = tx.QueryRowContext(ctx, `
					SELECT id::text FROM catalog_node
					WHERE tenant_id = $1 AND node_type = 'semantic_term'
					  AND (node_name = $2 OR properties->>'term_key' = $2)
					LIMIT 1
				`, tenantID, fieldName).Scan(&fieldTermID)

				if fieldTermID == nil {
					newTermID := uuid.New()
					_, err := tx.ExecContext(ctx, `
						INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
						VALUES ($1, $2, $3, $4, $5, $6, true)
						ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET updated_at = NOW()
					`, newTermID, tenantID, boTypeID, fieldName,
						"semantic_term/"+fieldName,
						fmt.Sprintf(`{"term_key":"%s","source":"oms.subtype_registry","bo_key":"%s"}`, fieldName, childKey))
					if err == nil {
						fieldTermIDStr := newTermID.String()
						fieldTermID = &fieldTermIDStr
					}
				}

				_, err := tx.ExecContext(ctx, `
					INSERT INTO business_object_fields
						(id, tenant_id, bo_id, term_node_id, field_name, field_role,
						 aggregation_type, binding_requirement, eligibility_source,
						 subtype_scope, is_exposed, inherits_defaults,
						 created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, 'DIMENSION',
					        'NONE', 'REQUIRED', 'DIRECT',
					        'ALL', true, true,
					        NOW(), NOW())
					ON CONFLICT (tenant_id, bo_id, field_name) DO UPDATE
						SET term_node_id   = EXCLUDED.term_node_id,
						    field_role     = EXCLUDED.field_role,
						    updated_at    = NOW()
				`, uuid.New(), tenantID, *childBOID, fieldTermID, fieldName)
				if err != nil {
					return fmt.Errorf("failed upserting business_object_field %s.%s: %w", childKey, fieldName, err)
				}
			}
		}
	}

	return tx.Commit()
}

func coalesceStr(a, b *string) *string {
	if a != nil && *a != "" {
		return a
	}
	return b
}
