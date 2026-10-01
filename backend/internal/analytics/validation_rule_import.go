package analytics

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/models"
)

const importIdempotencyTTL = 24 * time.Hour

// ImportRules executes a validated bundle import as a single atomic
// transaction. It is gated by Preflight: any blocking error returns the
// report without touching the database.
//
// Concurrency: pg_advisory_xact_lock on the target tenant serializes
// concurrent imports for the same tenant (and against prune recomputation).
// The upsert itself is safe via ON CONFLICT (tenant_id, qualified_path).
//
// Idempotency: successful (non-dry-run) imports are cached for 24h keyed on
// (tenant, idempotency_key). The key defaults to sha256(checksum + tenant);
// a caller-supplied key (e.g. X-Idempotency-Key header) overrides it. A
// retried import within the TTL returns the original report without
// re-executing.
func (p *ValidationRulePorter) ImportRules(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool, idempotencyKeyHeader string) (*models.RuleImportReport, error) {
	if req.TargetTenantID == "" {
		return nil, fmt.Errorf("target_tenant_id is required")
	}

	// 1. Preflight gate. The report from a failed preflight IS the response.
	report, err := p.Preflight(ctx, req, bypassGovernance)
	if err != nil {
		return nil, err
	}
	if len(report.Errors) > 0 {
		return report, nil // all-or-nothing: never execute a plan with errors
	}
	if req.DryRun {
		return report, nil
	}

	// 2. Idempotency: return cached report if this exact import already ran.
	idemKey := idempotencyKeyHeader
	if idemKey == "" {
		sum := sha256.Sum256([]byte(req.Bundle.Checksum + "|" + req.TargetTenantID))
		idemKey = hex.EncodeToString(sum[:])
	}
	if cached, found, err := p.loadCachedReport(ctx, req.TargetTenantID, idemKey); err != nil {
		return nil, fmt.Errorf("load idempotency cache: %w", err)
	} else if found {
		return cached, nil
	}

	// 3. Execute.
	if err := p.applyImport(ctx, req, report, bypassGovernance); err != nil {
		return nil, err
	}

	// 4. Cache the report (best-effort: failure to cache must not fail an
	// already-committed import).
	_ = p.cacheReport(ctx, req.TargetTenantID, idemKey, req.Bundle.Checksum, report)
	return report, nil
}

func (p *ValidationRulePorter) applyImport(ctx context.Context, req models.RuleImportRequest, report *models.RuleImportReport, bypassGovernance bool) error {
	gold, err := goldCopyTenantID(ctx, p.db)
	if err != nil {
		return err
	}
	if req.TargetTenantID == gold {
		return fmt.Errorf("import into the gold-copy tenant is not permitted via this path; use RULES_IMPORT_CORE admin tooling")
	}

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Serialize imports + prune for this tenant. hashtext is stable across
	// connections; the lock is released at COMMIT/ROLLBACK automatically.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, req.TargetTenantID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}

	// Resolve reusable IDs once.
	var nodeTypeID, edgeTypeID string
	if err := tx.GetContext(ctx, &nodeTypeID, `SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule' LIMIT 1`); err != nil {
		return fmt.Errorf("validation_rule node type not found: %w", err)
	}
	if err := tx.GetContext(ctx, &edgeTypeID, `SELECT id FROM catalog_edge_type WHERE edge_type_name = 'GOVERNED_BY_RULE' LIMIT 1`); err != nil {
		return fmt.Errorf("GOVERNED_BY_RULE edge type not found: %w", err)
	}

	// Recompute the actionable plan inside the lock (races between preflight
	// and apply cannot corrupt state because apply re-derives CREATE/UPDATE
	// from the upsert's actual outcome rather than trusting the stale diff).
	boNodeIDs := map[string]string{} // boName -> classification_node_id

	applyRule := func(spec *models.PortableRuleSpec) error {
		if spec.Deleted {
			qualifiedPath := qualifiedPathFor(req.TargetTenantID, spec.RuleKey)
			res, err := tx.ExecContext(ctx, `
				UPDATE catalog_node SET is_active = false, updated_at = NOW()
				WHERE tenant_id = $1::uuid AND qualified_path = $2
				  AND node_type_id = $3::uuid
			`, req.TargetTenantID, qualifiedPath, nodeTypeID)
			if err != nil {
				return fmt.Errorf("tombstone %q: %w", spec.RuleKey, err)
			}
			if n, _ := res.RowsAffected(); n > 0 {
				report.Pruned = append(report.Pruned, spec.RuleKey)
			}
			return nil
		}

		boNodeID, ok := boNodeIDs[spec.BOName]
		if !ok {
			if err := tx.GetContext(ctx, &boNodeID, `
				SELECT classification_node_id FROM business_objects
				WHERE (bo_key = $1 OR bo_name = $1) AND tenant_id = ANY($2::uuid[])
				ORDER BY (tenant_id = $3::uuid) DESC
				LIMIT 1
			`, spec.BOName, pq.Array(visibleTenants(req.TargetTenantID, gold)), req.TargetTenantID); err != nil {
				return fmt.Errorf("BO %q not found for rule %q: %w", spec.BOName, spec.RuleKey, err)
			}
			boNodeIDs[spec.BOName] = boNodeID
		}

		effectiveStatus := spec.GovernanceStatus
		if !bypassGovernance && spec.Domain == models.ValidationRuleDomainCompliance &&
			spec.GovernanceStatus == models.ValidationRuleGovernancePublished {
			effectiveStatus = models.ValidationRuleGovernanceSubmittedForReview
		}
		if req.PreserveStatus {
			// Preserve the existing status on update; drafts stay drafts on create.
			var existingStatus string
			err := tx.GetContext(ctx, &existingStatus, `
				SELECT COALESCE(properties->>'governance_status', '') FROM catalog_node
				WHERE tenant_id = $1::uuid AND qualified_path = $2
			`, req.TargetTenantID, qualifiedPathFor(req.TargetTenantID, spec.RuleKey))
			if err == nil && existingStatus != "" {
				effectiveStatus = existingStatus
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("check existing governance status for %q: %w", spec.RuleKey, err)
			}
		}

		props := models.ValidationRuleProperties{
			BOName:           spec.BOName,
			TenantID:         req.TargetTenantID,
			RuleKey:          spec.RuleKey,
			Severity:         spec.Severity,
			Timing:           spec.Timing,
			Category:         spec.Category,
			GovernanceStatus: effectiveStatus,
			Domain:           spec.Domain,
		}
		propsJSON, err := json.Marshal(props)
		if err != nil {
			return fmt.Errorf("marshal properties for %q: %w", spec.RuleKey, err)
		}
		cfgJSON, err := json.Marshal(models.ValidationRuleConfig{RuleAST: spec.RuleAST})
		if err != nil {
			return fmt.Errorf("marshal config for %q: %w", spec.RuleKey, err)
		}

		var outcome struct {
			ID          string `db:"id"`
			WasInserted bool   `db:"was_inserted"`
		}
		if err := tx.GetContext(ctx, &outcome, `
			INSERT INTO catalog_node (
				id, node_name, description, node_type_id, tenant_id,
				qualified_path, properties, config, is_active, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4::uuid, $5::uuid, $6, $7, $8, true, NOW(), NOW())
			ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
				node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties,
				config = EXCLUDED.config,
				is_active = true,
				updated_at = NOW()
			RETURNING id, (xmax = 0) AS was_inserted
		`, uuid.NewString(), spec.Name, spec.Description, nodeTypeID,
			req.TargetTenantID, qualifiedPathFor(req.TargetTenantID, spec.RuleKey),
			propsJSON, cfgJSON); err != nil {
			return fmt.Errorf("upsert rule node %q: %w", spec.RuleKey, err)
		}

		if outcome.ID == "" {
			return fmt.Errorf("upsert rule node %q returned empty id", spec.RuleKey)
		}

		// GOVERNED_BY_RULE edge: subject = BO classification node, object = rule node.
		var edgeExists bool
		if err := tx.GetContext(ctx, &edgeExists, `
			SELECT EXISTS(
				SELECT 1 FROM catalog_edge
				WHERE source_node_id = $1::uuid AND target_node_id = $2::uuid AND edge_type_id = $3::uuid
			)
		`, boNodeID, outcome.ID, edgeTypeID); err != nil {
			return fmt.Errorf("check edge for %q: %w", spec.RuleKey, err)
		}
		if !edgeExists {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, tenant_id, created_at, updated_at)
				VALUES (gen_random_uuid(), $1::uuid, $2::uuid, $3::uuid, $4::uuid, NOW(), NOW())
			`, boNodeID, outcome.ID, edgeTypeID, req.TargetTenantID); err != nil {
				return fmt.Errorf("insert GOVERNED_BY_RULE edge for %q: %w", spec.RuleKey, err)
			}
		}

		if outcome.WasInserted {
			report.Created = append(report.Created, spec.RuleKey)
		} else {
			report.Updated = append(report.Updated, spec.RuleKey)
		}
		return nil
	}

	// Apply rules in preflight's topo-sorted order.
	bundle := req.Bundle
	for i := range bundle.Rules {
		spec := &bundle.Rules[i]
		if err := applyRule(spec); err != nil {
			return fmt.Errorf("apply rule %q: %w", spec.RuleKey, err)
		}
	}

	// Prune (opt-in): inactivate tenant custom rules absent from the bundle.
	if req.Prune {
		inBundle := map[string]bool{}
		for i := range bundle.Rules {
			inBundle[bundle.Rules[i].RuleKey] = true
		}
		var staleKeys []string
		if err := tx.SelectContext(ctx, &staleKeys, `
			SELECT COALESCE(NULLIF(properties->>'rule_key', ''), node_name)
			FROM catalog_node n
			JOIN catalog_node_type nt ON n.node_type_id = nt.id
			WHERE nt.catalog_type_name = 'validation_rule'
			  AND n.tenant_id = $1::uuid
			  AND n.is_active
		`, req.TargetTenantID); err != nil {
			return fmt.Errorf("enumerate rules for prune: %w", err)
		}
		for _, key := range staleKeys {
			if inBundle[key] {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				UPDATE catalog_node SET is_active = false, updated_at = NOW()
				WHERE tenant_id = $1::uuid AND qualified_path = $2 AND node_type_id = $3::uuid
			`, req.TargetTenantID, qualifiedPathFor(req.TargetTenantID, key), nodeTypeID); err != nil {
				return fmt.Errorf("prune %q: %w", key, err)
			}
			report.Pruned = append(report.Pruned, key)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	return nil
}

func qualifiedPathFor(tenantID, ruleKey string) string {
	return fmt.Sprintf("validation_rule/%s/%s", tenantID, ruleKey)
}

// --- Idempotency cache ---

func (p *ValidationRulePorter) loadCachedReport(ctx context.Context, tenantID, idemKey string) (*models.RuleImportReport, bool, error) {
	var raw []byte
	err := p.db.GetContext(ctx, &raw, `
		SELECT report FROM validation_rule_import_log
		WHERE tenant_id = $1::uuid AND idempotency_key = $2
		  AND created_at > NOW() - INTERVAL '24 hours'
	`, tenantID, idemKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var report models.RuleImportReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, false, fmt.Errorf("corrupt cached report: %w", err)
	}
	return &report, true, nil
}

func (p *ValidationRulePorter) cacheReport(ctx context.Context, tenantID, idemKey, checksum string, report *models.RuleImportReport) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, `
		INSERT INTO validation_rule_import_log (tenant_id, idempotency_key, bundle_checksum, report)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (tenant_id, idempotency_key) DO UPDATE SET
			report = EXCLUDED.report, created_at = NOW()
	`, tenantID, idemKey, checksum, raw)
	return err
}
