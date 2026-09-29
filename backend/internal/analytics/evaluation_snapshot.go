package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
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
// optionally filtering by domain and timing.
func (s *ValidationRuleService) LoadRuleSnapshot(ctx context.Context, tenantID, boName, domain, timing string) (*RuleSnapshot, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if boName == "" {
		return nil, fmt.Errorf("bo_name is required")
	}

	descriptors, err := s.ListByBO(ctx, tenantID, boName, domain)
	if err != nil {
		return nil, fmt.Errorf("list rules for BO %s: %w", boName, err)
	}

	evaluableRules := make([]EvaluableRule, 0, len(descriptors))
	for _, desc := range descriptors {
		if !desc.IsActive {
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

	snapID := uuid.New().String()
	return &RuleSnapshot{
		TenantID:   tenantID,
		BOName:     boName,
		SnapshotID: snapID,
		LoadedAt:   time.Now().UTC(),
		Rules:      evaluableRules,
		ColumnMap:  make(map[string]string),
	}, nil
}
