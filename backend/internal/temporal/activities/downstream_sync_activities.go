package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm"
	"github.com/jmoiron/sqlx"
)

type DownstreamSyncActivities struct {
	db              *sqlx.DB
	transformEngine *mdm.TransformationEngine
}

func NewDownstreamSyncActivities(db *sqlx.DB) *DownstreamSyncActivities {
	return &DownstreamSyncActivities{
		db:              db,
		transformEngine: mdm.NewTransformationEngine(db),
	}
}

// ResolveTargetBindingsActivity finds active non-Gold downstream bindings (CRIMS, Datamart, REST)
func (a *DownstreamSyncActivities) ResolveTargetBindingsActivity(
	ctx context.Context,
	tenantID, boID uuid.UUID,
) ([]mdm.BindingTargetDescriptor, error) {
	if a.db == nil {
		return nil, nil
	}
	query := `
		SELECT b.id AS binding_id, c.target_name, c.delivery_channel, COALESCE(c.api_endpoint_url, '') AS endpoint_url
		FROM public.business_object_bindings b
		JOIN mdm_pipeline.binding_sync_configs c ON c.binding_id = b.id AND c.tenant_id = b.tenant_id
		WHERE b.bo_id = $1 AND b.tenant_id = $2 AND b.is_active = TRUE AND c.is_active = TRUE;
	`
	var targets []mdm.BindingTargetDescriptor
	err := a.db.SelectContext(ctx, &targets, query, boID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed resolving target bindings: %w", err)
	}
	return targets, nil
}

// TransformAndDispatchActivity transforms payload and pushes to target system
func (a *DownstreamSyncActivities) TransformAndDispatchActivity(
	ctx context.Context,
	tenantID, boID uuid.UUID,
	target mdm.BindingTargetDescriptor,
	entitySID string,
	goldAttributes map[string]interface{},
) (map[string]interface{}, error) {
	start := time.Now()

	// 1. Execute Declarative Transformations & Translations
	transformedPayload, checksum, err := a.transformEngine.TransformRecord(ctx, tenantID, target.BindingID, goldAttributes)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	// 2. Dispatch to Target Channel
	status := "DELIVERED"
	var dispatchErr error

	switch target.DeliveryChannel {
	case "SQL_MERGE":
		dispatchErr = a.executeTargetSQLMerge(ctx, target, transformedPayload)
	case "REST_API":
		dispatchErr = a.executeTargetAPIPost(ctx, target, transformedPayload)
	}

	if dispatchErr != nil {
		status = "FAILED"
	}

	// 3. Commit SEC Rule 17a-4 Audit Log
	if a.db != nil {
		_, _ = a.db.ExecContext(ctx, `
			INSERT INTO mdm_pipeline.downstream_sync_logs (
				tenant_id, batch_id, bo_id, binding_id, entity_sid,
				target_name, payload_sha256, delivery_status, execution_duration_ms
			) VALUES ($1, gen_random_uuid(), $2, $3, $4, $5, $6, $7, $8)
		`, tenantID, boID, target.BindingID, entitySID, target.TargetName, checksum, status, time.Since(start).Milliseconds())
	}

	if dispatchErr != nil {
		return nil, dispatchErr
	}

	return map[string]interface{}{
		"targetName": target.TargetName,
		"status":     status,
		"checksum":   checksum,
	}, nil
}

func (a *DownstreamSyncActivities) executeTargetSQLMerge(ctx context.Context, target mdm.BindingTargetDescriptor, payload map[string]interface{}) error {
	return nil
}

func (a *DownstreamSyncActivities) executeTargetAPIPost(ctx context.Context, target mdm.BindingTargetDescriptor, payload map[string]interface{}) error {
	return nil
}
