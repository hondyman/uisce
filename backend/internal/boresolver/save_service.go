package boresolver

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BOSaveService struct {
	db *sqlx.DB
}

func NewBOSaveService(db *sqlx.DB) *BOSaveService {
	return &BOSaveService{db: db}
}

// SaveBusinessObjectAtomic commits the entire BO, all bindings, and fields in one transaction
func (s *BOSaveService) SaveBusinessObjectAtomic(
	ctx context.Context,
	req AtomicSaveBORequest,
) (uuid.UUID, error) {
	if req.TenantID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		if req.BusinessObject.BOID != nil && *req.BusinessObject.BOID != uuid.Nil {
			return *req.BusinessObject.BOID, nil
		}
		return uuid.New(), nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()

	// 1. Upsert Business Object Header
	boID := uuid.New()
	if req.BusinessObject.BOID != nil && *req.BusinessObject.BOID != uuid.Nil {
		boID = *req.BusinessObject.BOID
	}

	publishStatus := "DRAFT"
	if req.Publish {
		publishStatus = "PUBLISHED"
	}

	boUpsertSQL := `
		INSERT INTO public.business_objects (
			id, tenant_id, model_id, bo_key, bo_name, description,
			bo_type, classification_node_id, business_key_node_id,
			semantic_id_node_id, grain_node_id, status, is_active, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true, NOW())
		ON CONFLICT (tenant_id, bo_key) DO UPDATE SET
			bo_name = EXCLUDED.bo_name,
			description = EXCLUDED.description,
			bo_type = EXCLUDED.bo_type,
			classification_node_id = EXCLUDED.classification_node_id,
			business_key_node_id = EXCLUDED.business_key_node_id,
			semantic_id_node_id = EXCLUDED.semantic_id_node_id,
			grain_node_id = EXCLUDED.grain_node_id,
			status = EXCLUDED.status,
			updated_at = NOW();
	`
	_, err = tx.ExecContext(ctx, boUpsertSQL,
		boID, req.TenantID, req.ModelID, req.BusinessObject.BOKey, req.BusinessObject.BOName,
		req.BusinessObject.Description, req.BusinessObject.BOType, req.BusinessObject.ClassificationNodeID,
		req.BusinessObject.BusinessKeyNodeID, req.BusinessObject.SemanticIDNodeID,
		req.BusinessObject.GrainNodeID, publishStatus)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed saving business object: %w", err)
	}

	// 2. Iterate and Persist Multi-Backend Bindings
	for _, b := range req.Bindings {
		bindingID := uuid.New()
		if b.BindingID != nil && *b.BindingID != uuid.Nil {
			bindingID = *b.BindingID
		}

		bindUpsertSQL := `
			INSERT INTO public.business_object_bindings (
				id, tenant_id, bo_id, backend_id, driving_node_id,
				is_default, temporal_override, is_active, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW())
			ON CONFLICT (tenant_id, bo_id, backend_id) DO UPDATE SET
				driving_node_id = EXCLUDED.driving_node_id,
				is_default = EXCLUDED.is_default,
				temporal_override = EXCLUDED.temporal_override,
				updated_at = NOW();
		`
		_, err = tx.ExecContext(ctx, bindUpsertSQL,
			bindingID, req.TenantID, boID, b.BackendID, b.DrivingNodeID, b.IsDefault, b.TemporalOverride)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed saving binding: %w", err)
		}

		// 3. Persist Field Definitions and Bindings
		for _, f := range b.Fields {
			fieldID := uuid.New()

			// Upsert business_object_fields
			fieldUpsertSQL := `
				INSERT INTO public.business_object_fields (
					field_id, tenant_id, bo_id, term_node_id, field_name,
					field_role, binding_requirement, binding_status, is_active
				) VALUES ($1, $2, $3, $4, $5, $6, $7, 'RESOLVED', true)
				ON CONFLICT (tenant_id, bo_id, term_node_id) DO UPDATE SET
					field_name = EXCLUDED.field_name,
					field_role = EXCLUDED.field_role,
					binding_requirement = EXCLUDED.binding_requirement;
			`
			_, err = tx.ExecContext(ctx, fieldUpsertSQL,
				fieldID, req.TenantID, boID, f.TermNodeID, f.FieldName, f.FieldRole, f.BindingRequirement)
			if err != nil {
				return uuid.Nil, fmt.Errorf("failed saving business object field: %w", err)
			}

			// Upsert field_bindings
			if f.SourceNodeID != nil && *f.SourceNodeID != uuid.Nil {
				fbUpsertSQL := `
					INSERT INTO public.field_bindings (
						id, tenant_id, bo_id, binding_id, field_id,
						source_node_id, source_type, transformation_type, transformation_sql, is_active
					) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, true)
					ON CONFLICT (tenant_id, bo_id, binding_id, field_id) DO UPDATE SET
						source_node_id = EXCLUDED.source_node_id,
						source_type = EXCLUDED.source_type,
						transformation_type = EXCLUDED.transformation_type,
						transformation_sql = EXCLUDED.transformation_sql;
				`
				_, err = tx.ExecContext(ctx, fbUpsertSQL,
					req.TenantID, boID, bindingID, fieldID, *f.SourceNodeID,
					f.SourceType, f.TransformationType, f.TransformationSQL)
				if err != nil {
					return uuid.Nil, fmt.Errorf("failed saving field binding: %w", err)
				}
			}
		}
	}

	return boID, tx.Commit()
}
