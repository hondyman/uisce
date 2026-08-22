package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// GraphRebaseResult represents the outcome of a 3-way graph merge on a node or edge.
type GraphRebaseResult struct {
	EntityID        uuid.UUID              `json:"entity_id"`
	EntityType      string                 `json:"entity_type"` // CATALOG_NODE, CATALOG_EDGE
	EntityName      string                 `json:"entity_name,omitempty"`
	HasConflict     bool                   `json:"has_conflict"`
	MergedPayload   map[string]interface{} `json:"merged_payload,omitempty"`
	ConflictingKeys []string               `json:"conflicting_keys,omitempty"`
	RequiresReview  bool                   `json:"requires_review"`
	Status          string                 `json:"status"` // CLEAN_MERGE, CONFLICT_DETECTED, UP_TO_DATE
	BaseV1Version   int                    `json:"base_v1_version"`
	BaseV2Version   int                    `json:"base_v2_version"`
}

// RebaseConflictRecord represents a collision recorded in the ledger.
type RebaseConflictRecord struct {
	ConflictID          uuid.UUID              `json:"conflict_id" db:"conflict_id"`
	TenantID            uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EntityType          string                 `json:"entity_type" db:"entity_type"`
	EntityID            uuid.UUID              `json:"entity_id" db:"entity_id"`
	GoldCopyNodeID      uuid.UUID              `json:"gold_copy_node_id" db:"gold_copy_node_id"`
	BaseV1Version       int                    `json:"base_v1_version" db:"base_v1_version"`
	BaseV2Version       int                    `json:"base_v2_version" db:"base_v2_version"`
	BaseV1Payload       map[string]interface{} `json:"base_v1_payload"`
	BaseV2Payload       map[string]interface{} `json:"base_v2_payload"`
	TenantCustomPayload map[string]interface{} `json:"tenant_custom_payload"`
	ConflictingKeys     []string               `json:"conflicting_keys"`
	ResolutionStatus    string                 `json:"resolution_status" db:"resolution_status"`
	ResolvedBy          *string                `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolvedAt          *time.Time             `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
}

// GoldCopyRebaseService coordinates 3-way synchronization across tenant metadata graphs.
type GoldCopyRebaseService struct {
	db *sqlx.DB
}

// NewGoldCopyRebaseService creates a new Gold-Copy rebase service.
func NewGoldCopyRebaseService(db *sqlx.DB) *GoldCopyRebaseService {
	return &GoldCopyRebaseService{db: db}
}

// Compute3WayMerge performs field-by-field mathematical delta evaluation:
// Δ_Tenant = Tenant_Custom ⊖ Base_v1
// Δ_Core   = Base_v2 ⊖ Base_v1
func (s *GoldCopyRebaseService) Compute3WayMerge(
	baseV1, baseV2, tenantCustom map[string]interface{},
) (map[string]interface{}, []string) {
	merged := make(map[string]interface{})
	var conflicts []string

	// Collect all keys across all 3 snapshots
	allKeys := make(map[string]bool)
	for k := range baseV1 {
		allKeys[k] = true
	}
	for k := range baseV2 {
		allKeys[k] = true
	}
	for k := range tenantCustom {
		allKeys[k] = true
	}

	for k := range allKeys {
		v1, inV1 := baseV1[k]
		v2, inV2 := baseV2[k]
		tc, inTC := tenantCustom[k]

		tenantModified := !reflect.DeepEqual(v1, tc) || (inV1 != inTC)
		coreModified := !reflect.DeepEqual(v1, v2) || (inV1 != inV2)

		switch {
		case !tenantModified && !coreModified:
			// Unchanged in both
			if inV1 {
				merged[k] = v1
			}
		case tenantModified && !coreModified:
			// Tenant customized; preserve local customization
			if inTC {
				merged[k] = tc
			}
		case !tenantModified && coreModified:
			// Upstream core upgraded; auto-apply platform update
			if inV2 {
				merged[k] = v2
			}
		case tenantModified && coreModified:
			// Both modified this key
			if reflect.DeepEqual(tc, v2) {
				// Converged to same value
				if inTC {
					merged[k] = tc
				}
			} else {
				// True conflict detected
				conflicts = append(conflicts, k)
				merged[k] = tc // Default to tenant safety until explicitly arbitrated
			}
		}
	}

	return merged, conflicts
}

// RebaseTenantNode evaluates an individual tenant node against master Gold-Copy updates.
func (s *GoldCopyRebaseService) RebaseTenantNode(
	ctx context.Context,
	tenantID, tenantNodeID, goldCopyNodeID uuid.UUID,
	dryRun bool,
) (*GraphRebaseResult, error) {
	if s.db == nil {
		return nil, errors.New("database connection is nil")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Fetch current Gold Copy (Base v2)
	var gcVersion int
	var gcName string
	var gcPropsRaw []byte
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(version_id, 1), node_name, COALESCE(properties, '{}'::jsonb)
		FROM catalog_node 
		WHERE id = $1 AND (tenant_id = '00000000-0000-0000-0000-000000000000' OR tenant_id = (SELECT id::text FROM public.tenants WHERE gold_copy = true LIMIT 1))
	`, goldCopyNodeID.String()).Scan(&gcVersion, &gcName, &gcPropsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed fetching gold copy node %s: %w", goldCopyNodeID, err)
	}

	// 2. Fetch Tenant Custom Node and its Base v1 snapshot
	var tenantDerivedVersion int
	var tenantPropsRaw, baseV1PropsRaw []byte
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(derived_from_version_id, 1), COALESCE(properties, '{}'::jsonb), COALESCE(base_snapshot_properties, '{}'::jsonb)
		FROM catalog_node 
		WHERE id = $1 AND tenant_id = $2
	`, tenantNodeID.String(), tenantID.String()).Scan(&tenantDerivedVersion, &tenantPropsRaw, &baseV1PropsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed fetching tenant node %s: %w", tenantNodeID, err)
	}

	// If already up-to-date
	if tenantDerivedVersion >= gcVersion {
		return &GraphRebaseResult{
			EntityID:      tenantNodeID,
			EntityType:    "CATALOG_NODE",
			EntityName:    gcName,
			HasConflict:   false,
			Status:        "UP_TO_DATE",
			BaseV1Version: tenantDerivedVersion,
			BaseV2Version: gcVersion,
		}, nil
	}

	var baseV1, baseV2, tc map[string]interface{}
	_ = json.Unmarshal(baseV1PropsRaw, &baseV1)
	_ = json.Unmarshal(gcPropsRaw, &baseV2)
	_ = json.Unmarshal(tenantPropsRaw, &tc)

	mergedProps, conflicts := s.Compute3WayMerge(baseV1, baseV2, tc)
	hasConflict := len(conflicts) > 0

	status := "CLEAN_MERGE"
	if hasConflict {
		status = "CONFLICT_DETECTED"
	}

	if !dryRun {
		mergedJSON, _ := json.Marshal(mergedProps)
		if hasConflict {
			conflictsJSON, _ := json.Marshal(conflicts)
			_, err = tx.ExecContext(ctx, `
				INSERT INTO catalog_rebase_conflict_ledger (
					tenant_id, entity_type, entity_id, gold_copy_node_id,
					base_v1_version, base_v2_version, base_v1_payload,
					base_v2_payload, tenant_custom_payload, conflicting_keys, resolution_status
				) VALUES ($1, 'CATALOG_NODE', $2, $3, $4, $5, $6, $7, $8, $9, 'PENDING_REVIEW')
			`, tenantID, tenantNodeID, goldCopyNodeID, tenantDerivedVersion, gcVersion,
				baseV1PropsRaw, gcPropsRaw, tenantPropsRaw, conflictsJSON)
			if err != nil {
				return nil, fmt.Errorf("failed recording rebase conflict: %w", err)
			}
		} else {
			// Clean merge: write merged properties, advance derived_from_version_id, snapshot Base v2
			_, err = tx.ExecContext(ctx, `
				UPDATE catalog_node
				SET properties = $1,
				    derived_from_version_id = $2,
				    base_snapshot_properties = $3,
				    last_rebased_at = NOW()
				WHERE id = $4 AND tenant_id = $5
			`, mergedJSON, gcVersion, gcPropsRaw, tenantNodeID.String(), tenantID.String())
			if err != nil {
				return nil, fmt.Errorf("failed updating rebased node: %w", err)
			}
		}
	}

	// 3. Reconcile Rejection Store Entries (Rejection Memory Guard)
	var rationaleDrift bool
	_ = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM catalog_edge_rejection_store r
			JOIN catalog_edge ce ON (ce.source_id = r.source_node_id OR ce.target_id = r.rejected_target_id)
			WHERE r.tenant_id = $1 
			  AND (r.source_node_id = $2 OR r.rejected_target_id = $2)
			  AND r.gold_copy_rationale_snapshot IS NOT NULL
			  AND r.gold_copy_rationale_snapshot != COALESCE(ce.properties->>'distinction_rationale', '')
		)
	`, tenantID, tenantNodeID).Scan(&rationaleDrift)

	if rationaleDrift && !dryRun {
		_, _ = tx.ExecContext(ctx, `
			UPDATE catalog_edge_rejection_store
			SET rebase_status = 'REVIEW_REQUIRED',
			    review_notes = 'Gold Copy distinction rationale was updated upstream. Please verify if rejection remains valid.'
			WHERE tenant_id = $1 AND (source_node_id = $2 OR rejected_target_id = $2)
		`, tenantID, tenantNodeID)
	}

	if !dryRun {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}

	return &GraphRebaseResult{
		EntityID:        tenantNodeID,
		EntityType:      "CATALOG_NODE",
		EntityName:      gcName,
		HasConflict:     hasConflict,
		MergedPayload:   mergedProps,
		ConflictingKeys: conflicts,
		RequiresReview:  rationaleDrift,
		Status:          status,
		BaseV1Version:   tenantDerivedVersion,
		BaseV2Version:   gcVersion,
	}, nil
}

// DryRunRebase previews all prospective 3-way merge changes for a tenant.
func (s *GoldCopyRebaseService) DryRunRebase(ctx context.Context, tenantID uuid.UUID) ([]GraphRebaseResult, error) {
	return s.executeBatchRebase(ctx, tenantID, true)
}

// ApplyRebase applies the 3-way merge and records audit events.
func (s *GoldCopyRebaseService) ApplyRebase(ctx context.Context, tenantID uuid.UUID) ([]GraphRebaseResult, error) {
	return s.executeBatchRebase(ctx, tenantID, false)
}

func (s *GoldCopyRebaseService) executeBatchRebase(ctx context.Context, tenantID uuid.UUID, dryRun bool) ([]GraphRebaseResult, error) {
	if s.db == nil {
		return []GraphRebaseResult{}, nil
	}

	// Query all tenant nodes that are linked to core / gold-copy nodes
	query := `
		SELECT tn.id as tenant_node_id, gc.id as gold_copy_node_id
		FROM catalog_node tn
		JOIN catalog_node gc ON (UPPER(tn.node_name) = UPPER(gc.node_name) OR tn.properties->>'core_reference_id' = gc.id::text)
		WHERE tn.tenant_id = $1
		  AND (gc.tenant_id = '00000000-0000-0000-0000-000000000000' OR gc.tenant_id = (SELECT id::text FROM public.tenants WHERE gold_copy = true LIMIT 1))
		  AND COALESCE(tn.derived_from_version_id, 1) < COALESCE(gc.version_id, 1)
	`
	var pairs []struct {
		TenantNodeID   uuid.UUID `db:"tenant_node_id"`
		GoldCopyNodeID uuid.UUID `db:"gold_copy_node_id"`
	}

	if err := s.db.SelectContext(ctx, &pairs, query, tenantID.String()); err != nil {
		logging.GetLogger().Sugar().Warnf("Batch rebase query note: %v", err)
	}

	results := make([]GraphRebaseResult, 0)
	for _, p := range pairs {
		res, err := s.RebaseTenantNode(ctx, tenantID, p.TenantNodeID, p.GoldCopyNodeID, dryRun)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("Failed rebasing node %s: %v", p.TenantNodeID, err)
			continue
		}
		if res != nil {
			results = append(results, *res)
		}
	}

	return results, nil
}

// ListConflicts retrieves active rebase conflicts for a tenant.
func (s *GoldCopyRebaseService) ListConflicts(ctx context.Context, tenantID uuid.UUID, status string) ([]RebaseConflictRecord, error) {
	if s.db == nil {
		return []RebaseConflictRecord{}, nil
	}

	if status == "" {
		status = "PENDING_REVIEW"
	}

	query := `
		SELECT conflict_id, tenant_id, entity_type, entity_id, gold_copy_node_id,
		       base_v1_version, base_v2_version, base_v1_payload, base_v2_payload,
		       tenant_custom_payload, conflicting_keys, resolution_status,
		       resolved_by, resolved_at, created_at
		FROM catalog_rebase_conflict_ledger
		WHERE tenant_id = $1 AND resolution_status = $2
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]RebaseConflictRecord, 0)
	for rows.Next() {
		var rec RebaseConflictRecord
		var b1, b2, tc, ck []byte
		err := rows.Scan(
			&rec.ConflictID, &rec.TenantID, &rec.EntityType, &rec.EntityID,
			&rec.GoldCopyNodeID, &rec.BaseV1Version, &rec.BaseV2Version,
			&b1, &b2, &tc, &ck, &rec.ResolutionStatus, &rec.ResolvedBy,
			&rec.ResolvedAt, &rec.CreatedAt,
		)
		if err != nil {
			continue
		}
		_ = json.Unmarshal(b1, &rec.BaseV1Payload)
		_ = json.Unmarshal(b2, &rec.BaseV2Payload)
		_ = json.Unmarshal(tc, &rec.TenantCustomPayload)
		_ = json.Unmarshal(ck, &rec.ConflictingKeys)
		records = append(records, rec)
	}

	return records, nil
}

// ResolveConflict resolves a conflict ledger entry with chosen strategy.
func (s *GoldCopyRebaseService) ResolveConflict(
	ctx context.Context,
	tenantID, conflictID uuid.UUID,
	resolution string, // RESOLVED_TENANT_OVERRIDE or RESOLVED_GOLD_COPY_ADOPTED
	resolvedBy string,
) error {
	if s.db == nil {
		return errors.New("db is nil")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var entityID, gcNodeID uuid.UUID
	var gcVersion int
	var b2Raw, tcRaw []byte

	err = tx.QueryRowContext(ctx, `
		SELECT entity_id, gold_copy_node_id, base_v2_version, base_v2_payload, tenant_custom_payload
		FROM catalog_rebase_conflict_ledger
		WHERE conflict_id = $1 AND tenant_id = $2
	`, conflictID, tenantID).Scan(&entityID, &gcNodeID, &gcVersion, &b2Raw, &tcRaw)
	if err != nil {
		return fmt.Errorf("conflict not found: %w", err)
	}

	targetPropsRaw := tcRaw
	if resolution == "RESOLVED_GOLD_COPY_ADOPTED" {
		targetPropsRaw = b2Raw
	}

	// Update node with resolved properties
	_, err = tx.ExecContext(ctx, `
		UPDATE catalog_node
		SET properties = $1,
		    derived_from_version_id = $2,
		    base_snapshot_properties = $3,
		    last_rebased_at = NOW()
		WHERE id = $4 AND tenant_id = $5
	`, targetPropsRaw, gcVersion, b2Raw, entityID.String(), tenantID.String())
	if err != nil {
		return fmt.Errorf("failed updating resolved node: %w", err)
	}

	// Mark conflict resolved
	_, err = tx.ExecContext(ctx, `
		UPDATE catalog_rebase_conflict_ledger
		SET resolution_status = $1,
		    resolved_by = $2,
		    resolved_at = NOW()
		WHERE conflict_id = $3 AND tenant_id = $4
	`, resolution, resolvedBy, conflictID, tenantID)
	if err != nil {
		return fmt.Errorf("failed updating conflict ledger: %w", err)
	}

	return tx.Commit()
}
