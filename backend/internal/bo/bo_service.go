package bo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/jmoiron/sqlx"
)

type BusinessObjectService struct {
	db        *sqlx.DB
	outboxMgr *audit.TransactionalOutboxManager
}

func NewBusinessObjectService(db *sqlx.DB, outboxMgr *audit.TransactionalOutboxManager) *BusinessObjectService {
	return &BusinessObjectService{db: db, outboxMgr: outboxMgr}
}

// SaveBusinessObjectAtomic commits the semantic shell, bindings, field mappings, and audit events in a single Tx
func (s *BusinessObjectService) SaveBusinessObjectAtomic(ctx context.Context, actorID, actorRole string, req SaveBusinessObjectRequest) (uuid.UUID, error) {
	if req.TenantID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed initiating transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Commit / Upsert Semantic Business Object Shell
	var boID uuid.UUID
	if req.BusinessObject.ID != nil && *req.BusinessObject.ID != uuid.Nil {
		boID = *req.BusinessObject.ID
	} else {
		boID = uuid.New()
	}

	upsertBOSQL := `
		INSERT INTO public.business_objects (
			id, tenant_id, model_id, bo_key, bo_name, description, bo_type,
			classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
			sti_discriminator_column, active_subtype_filter, is_active, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, true, NOW())
		ON CONFLICT (tenant_id, bo_key) DO UPDATE SET
			bo_name = EXCLUDED.bo_name,
			description = EXCLUDED.description,
			classification_node_id = EXCLUDED.classification_node_id,
			business_key_node_id = EXCLUDED.business_key_node_id,
			semantic_id_node_id = EXCLUDED.semantic_id_node_id,
			grain_node_id = EXCLUDED.grain_node_id,
			sti_discriminator_column = EXCLUDED.sti_discriminator_column,
			active_subtype_filter = EXCLUDED.active_subtype_filter,
			updated_at = NOW()
		RETURNING id;
	`
	err = tx.QueryRowContext(ctx, upsertBOSQL,
		boID, req.TenantID, req.ModelID, req.BusinessObject.BOKey, req.BusinessObject.BOName,
		req.BusinessObject.Description, req.BusinessObject.BOType, req.BusinessObject.ClassificationNodeID,
		req.BusinessObject.BusinessKeyNodeID, req.BusinessObject.SemanticIDNodeID, req.BusinessObject.GrainNodeID,
		req.BusinessObject.STIDiscriminatorColumn, req.BusinessObject.ActiveSubtypeFilter,
	).Scan(&boID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed saving business object entity: %w", err)
	}

	// 2. Iterate Backend Bindings
	for _, b := range req.Bindings {
		var bindingID uuid.UUID
		upsertBindingSQL := `
			INSERT INTO public.business_object_bindings (
				tenant_id, bo_id, backend_id, backend_type, driving_node_id, is_default, temporal_override, base_sql, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
			ON CONFLICT (tenant_id, bo_id, backend_id) DO UPDATE SET
				driving_node_id = EXCLUDED.driving_node_id,
				is_default = EXCLUDED.is_default,
				temporal_override = EXCLUDED.temporal_override,
				base_sql = EXCLUDED.base_sql,
				updated_at = NOW()
			RETURNING id;
		`
		err = tx.QueryRowContext(ctx, upsertBindingSQL,
			req.TenantID, boID, b.BackendID, b.BackendType, b.DrivingNodeID, b.IsDefault, b.TemporalOverride, b.BaseSQL,
		).Scan(&bindingID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed persisting backend binding: %w", err)
		}

		// 3. Persist Fields & Field Bindings
		for _, f := range b.Fields {
			var fieldID uuid.UUID
			upsertFieldSQL := `
				INSERT INTO public.business_object_fields (
					tenant_id, bo_id, term_node_id, field_name, field_role, aggregation_type,
					binding_requirement, eligibility_source, subtype_scope, is_exposed, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, NOW())
				ON CONFLICT (tenant_id, bo_id, field_name) DO UPDATE SET
					field_role = EXCLUDED.field_role,
					aggregation_type = EXCLUDED.aggregation_type,
					binding_requirement = EXCLUDED.binding_requirement,
					updated_at = NOW()
				RETURNING id;
			`
			err = tx.QueryRowContext(ctx, upsertFieldSQL,
				req.TenantID, boID, f.TermNodeID, f.FieldName, f.FieldRole, f.AggregationType,
				f.BindingRequirement, f.EligibilitySource, f.SubtypeScope,
			).Scan(&fieldID)
			if err != nil {
				return uuid.Nil, fmt.Errorf("failed persisting logical field %s: %w", f.FieldName, err)
			}

			upsertFieldBindingSQL := `
				INSERT INTO public.field_bindings (
					tenant_id, bo_id, binding_id, field_id, source_node_id, source_type,
					transformation_type, transformation_sql, json_path, binding_status, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'RESOLVED', NOW())
				ON CONFLICT (tenant_id, binding_id, field_id) DO UPDATE SET
					source_node_id = EXCLUDED.source_node_id,
					source_type = EXCLUDED.source_type,
					transformation_type = EXCLUDED.transformation_type,
					transformation_sql = EXCLUDED.transformation_sql,
					json_path = EXCLUDED.json_path,
					binding_status = 'RESOLVED',
					updated_at = NOW();
			`
			_, err = tx.ExecContext(ctx, upsertFieldBindingSQL,
				req.TenantID, boID, bindingID, fieldID, f.SourceNodeID, f.SourceType,
				f.TransformationType, f.TransformationSQL, f.JSONPath,
			)
			if err != nil {
				return uuid.Nil, fmt.Errorf("failed persisting physical field binding: %w", err)
			}
		}

		// 4. Persist Relationships
		for _, rel := range b.Relationships {
			var relID uuid.UUID
			upsertRelSQL := `
				INSERT INTO public.business_object_relationships (
					tenant_id, from_bo_id, to_bo_id, rel_key, rel_name, cardinality, join_type, is_active
				) VALUES ($1, $2, $3, $4, $5, $6, $7, true)
				ON CONFLICT (tenant_id, from_bo_id, rel_key) DO UPDATE SET
					cardinality = EXCLUDED.cardinality,
					join_type = EXCLUDED.join_type
				RETURNING id;
			`
			err = tx.QueryRowContext(ctx, upsertRelSQL,
				req.TenantID, boID, rel.ToBOID, rel.RelKey, rel.RelName, rel.Cardinality, rel.JoinType,
			).Scan(&relID)
			if err != nil {
				return uuid.Nil, fmt.Errorf("failed persisting relationship %s: %w", rel.RelKey, err)
			}

			upsertRelBindSQL := `
				INSERT INTO public.relationship_bindings (
					tenant_id, rel_id, binding_id, join_condition_sql
				) VALUES ($1, $2, $3, $4)
				ON CONFLICT (tenant_id, rel_id, binding_id) DO UPDATE SET
					join_condition_sql = EXCLUDED.join_condition_sql;
			`
			_, err = tx.ExecContext(ctx, upsertRelBindSQL, req.TenantID, relID, bindingID, rel.JoinConditionSQL)
			if err != nil {
				return uuid.Nil, fmt.Errorf("failed persisting relationship join binding: %w", err)
			}
		}
	}

	// 5. SEC Rule 17a-4 Cryptographic Audit Outbox Staging
	eventType := "BO_DRAFT_SAVED"
	if req.Publish {
		eventType = "BO_PUBLISHED"
	}
	if s.outboxMgr != nil {
		err = s.outboxMgr.StageOutboxEventAtomic(
			ctx, tx, req.TenantID, boID, "BUSINESS_OBJECT", eventType, actorID, actorRole,
			map[string]interface{}{
				"bo_key":         req.BusinessObject.BOKey,
				"bindings_count": len(req.Bindings),
				"published":      req.Publish,
				"timestamp":      time.Now().UTC(),
			},
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed staging outbox audit seal: %w", err)
		}
	}

	return boID, tx.Commit()
}
