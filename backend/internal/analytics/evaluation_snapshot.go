package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/lib/pq"
)

// EvaluableRule represents a single pre-compiled validation rule ready for in-memory evaluation.
type EvaluableRule struct {
	RuleID      string       `json:"rule_id"`
	RuleKey     string       `json:"rule_key"`
	RuleName    string       `json:"rule_name"`
	BOName      string       `json:"bo_name"`
	Severity    string       `json:"severity"` // "BLOCK" | "WARN"
	Timing      string       `json:"timing,omitempty"`
	Domain      string       `json:"domain,omitempty"`
	RuleVersion string       `json:"rule_version,omitempty"`
	AST         *vm.RuleNode `json:"ast"`
	FieldRefs   []string     `json:"field_refs"`
	BindingIDs  []string     `json:"binding_ids,omitempty"`
}

// RuleSnapshot is an immutable in-memory snapshot of active validation rules for a given
// tenant and Business Object, enabling high-throughput evaluation without querying catalog nodes
// on every row.
type RuleSnapshot struct {
	TenantID   string            `json:"tenant_id"`
	BOName     string            `json:"bo_name"`
	SnapshotID string            `json:"snapshot_id"`
	LoadedAt   time.Time         `json:"loaded_at"`
	Rules      []EvaluableRule   `json:"rules"`
	ColumnMap  map[string]string `json:"column_map,omitempty"` // semantic term -> physical column
}

// LoadRuleSnapshot compiles and returns an in-memory RuleSnapshot for tenantID and boName,
// enforcing the live write-path choke point: only published, active rules are evaluated.
func (s *ValidationRuleService) LoadRuleSnapshot(ctx context.Context, tenantID, boName, domain, timing string) (*RuleSnapshot, error) {
	return s.LoadRuleSnapshotAsOf(ctx, tenantID, boName, domain, timing, time.Time{})
}

// LoadRuleSnapshotAsOf loads rules as-of a given point in time (bitemporal lookup).
// If asOf is zero, it queries the live catalog enforcing governance_status = 'published'.
func (s *ValidationRuleService) LoadRuleSnapshotAsOf(ctx context.Context, tenantID, boName, domain, timing string, asOf time.Time) (*RuleSnapshot, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if boName == "" {
		return nil, fmt.Errorf("bo_name is required")
	}

	gold, err := goldCopyTenantID(ctx, s.db)
	if err != nil {
		return nil, err
	}
	visible := visibleTenants(tenantID, gold)

	// 1. As-of query against bitemporal validation_rule_versions
	if !asOf.IsZero() {
		return s.loadSnapshotFromVersions(ctx, tenantID, gold, visible, boName, domain, timing, asOf)
	}

	// 2. Live snapshot query against catalog_node
	descriptors, err := s.ListByBO(ctx, tenantID, boName, domain)
	if err != nil {
		return nil, fmt.Errorf("list rules for BO %s: %w", boName, err)
	}

	evaluableRules := make([]EvaluableRule, 0, len(descriptors))
	for _, desc := range descriptors {
		if !desc.IsActive {
			continue
		}
		// Write-path choke point: only strictly published rules are evaluated
		if desc.GovernanceStatus != models.ValidationRuleGovernancePublished {
			continue
		}

		if len(desc.RuleAST) == 0 {
			continue
		}
		if domain != "" && desc.Domain != domain {
			continue
		}
		if timing != "" && desc.Timing != timing {
			continue
		}

		var node vm.RuleNode
		if err := json.Unmarshal(desc.RuleAST, &node); err != nil {
			return nil, fmt.Errorf("parse rule %s (%s) AST: %w", desc.Name, desc.ID, err)
		}

		refs := vm.FieldRefs(node)
		ruleKey := desc.RuleKey
		if ruleKey == "" {
			ruleKey = desc.Name
		}

		severity := desc.Severity
		if severity == "" {
			severity = models.ValidationRuleSeverityBlock
		}

		evaluableRules = append(evaluableRules, EvaluableRule{
			RuleID:      desc.ID.String(),
			RuleKey:     ruleKey,
			RuleName:    desc.Name,
			BOName:      desc.BOName,
			Severity:    severity,
			Timing:      desc.Timing,
			Domain:      desc.Domain,
			RuleVersion: "1",
			AST:         &node,
			FieldRefs:   refs,
			BindingIDs:  desc.BindingIDs,
		})
	}

	colMap, _ := s.boColumns(ctx, tenantID, gold, visible, boName)
	if colMap == nil {
		colMap = make(map[string]string)
	}

	snapID := uuid.New().String()
	return &RuleSnapshot{
		TenantID:   tenantID,
		BOName:     boName,
		SnapshotID: snapID,
		LoadedAt:   time.Now().UTC(),
		Rules:      evaluableRules,
		ColumnMap:  colMap,
	}, nil
}

func (s *ValidationRuleService) loadSnapshotFromVersions(ctx context.Context, tenantID, gold string, visible []string, boName, domain, timing string, asOf time.Time) (*RuleSnapshot, error) {
	var rows []struct {
		RuleNodeID uuid.UUID       `db:"rule_node_id"`
		Version    int             `db:"version"`
		RuleAST    json.RawMessage `db:"rule_ast"`
		Properties json.RawMessage `db:"properties"`
	}

	query := `
		SELECT v.rule_node_id, v.version, v.rule_ast, v.properties
		FROM validation_rule_versions v
		WHERE v.tenant_id = ANY($1::uuid[])
		  AND (v.properties->>'bo_name' = $2 OR $2 = '')
		  AND v.effective_from <= $3
		  AND (v.effective_to IS NULL OR v.effective_to > $3)
		  AND v.recorded_at <= $3
		  AND (v.superseded_at IS NULL OR v.superseded_at > $3)
		  AND v.governance_status = 'published'
		ORDER BY v.version DESC
	`
	err := s.db.SelectContext(ctx, &rows, query, pq.Array(visible), boName, asOf)
	if err != nil {
		return nil, fmt.Errorf("load bitemporal rule versions as-of %s: %w", asOf.Format(time.RFC3339), err)
	}

	seenRules := make(map[uuid.UUID]bool)
	evaluableRules := make([]EvaluableRule, 0, len(rows))

	for _, row := range rows {
		if seenRules[row.RuleNodeID] {
			continue // pick latest active version for each rule_node_id
		}
		seenRules[row.RuleNodeID] = true

		props, err := models.ParseValidationRuleProperties(row.Properties)
		if err != nil {
			continue
		}
		if domain != "" && props.Domain != domain {
			continue
		}
		if timing != "" && props.Timing != timing {
			continue
		}

		var node vm.RuleNode
		if err := json.Unmarshal(row.RuleAST, &node); err != nil {
			continue
		}

		refs := vm.FieldRefs(node)
		ruleKey := props.RuleKey
		if ruleKey == "" {
			ruleKey = row.RuleNodeID.String()
		}

		evaluableRules = append(evaluableRules, EvaluableRule{
			RuleID:      row.RuleNodeID.String(),
			RuleKey:     ruleKey,
			RuleName:    ruleKey,
			BOName:      props.BOName,
			Severity:    props.Severity,
			Timing:      props.Timing,
			Domain:      props.Domain,
			RuleVersion: fmt.Sprintf("%d", row.Version),
			AST:         &node,
			FieldRefs:   refs,
			BindingIDs:  props.BindingIDs,
		})
	}

	colMap, _ := s.boColumns(ctx, tenantID, gold, visible, boName)
	if colMap == nil {
		colMap = make(map[string]string)
	}

	return &RuleSnapshot{
		TenantID:   tenantID,
		BOName:     boName,
		SnapshotID: fmt.Sprintf("asof_%s_%d", boName, asOf.Unix()),
		LoadedAt:   asOf.UTC(),
		Rules:      evaluableRules,
		ColumnMap:  colMap,
	}, nil
}
