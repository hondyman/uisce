package mdm

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StrategyType defines the survivorship resolution strategy
type StrategyType string

const (
	StrategySourcePriority     StrategyType = "SOURCE_PRIORITY"
	StrategyMostRecent         StrategyType = "MOST_RECENT"
	StrategyConservativeMin    StrategyType = "CONSERVATIVE_MIN"
	StrategyConservativeMax    StrategyType = "CONSERVATIVE_MAX"
	StrategyMostFrequent       StrategyType = "MOST_FREQUENT"
	StrategyWeightedConfidence  StrategyType = "WEIGHTED_CONFIDENCE"
)

// FieldPlan represents the compiled Intermediate Representation (IR) for a single attribute's survivorship
type FieldPlan struct {
	Attribute       string         `json:"attribute"`
	SourceColumn    string         `json:"source_column"`
	TargetColumn    string         `json:"target_column"`
	Strategy        StrategyType   `json:"strategy"`
	PriorityOrder   []string       `json:"priority_order,omitempty"`
	MaxStaleSeconds *int           `json:"max_stale_seconds,omitempty"`
	SelectionRuleID *string        `json:"selection_rule_id,omitempty"`
	SelectionMode   string         `json:"selection_mode,omitempty"`
	ResolutionTier  ResolutionTier `json:"resolution_tier"`
	RuleID          uuid.UUID      `json:"rule_id"`
	TiebreakerChain []string       `json:"tiebreaker_chain"`
}

// SurvivorshipBatchPlan represents the complete IR for an entity batch execution
type SurvivorshipBatchPlan struct {
	TenantID       uuid.UUID   `json:"tenant_id"`
	EntityType     string      `json:"entity_type"`
	StagingTable   string      `json:"staging_table"`
	TargetTable    string      `json:"target_table"`
	EntityKeyField string      `json:"entity_key_field"`
	SourceIDField  string      `json:"source_id_field"`
	AsOfField      string      `json:"as_of_field"`
	PKField        string      `json:"pk_field"`
	BatchCutoff    time.Time   `json:"batch_cutoff"`
	Fields         []FieldPlan `json:"fields"`
}

// PlanConfig provides configuration options for constructing a batch plan
type PlanConfig struct {
	TenantID       uuid.UUID
	EntityType     string
	StagingTable   string
	TargetTable    string
	EntityKeyField string
	SourceIDField  string
	AsOfField      string
	PKField        string
	BatchCutoff    time.Time
}

// BuildBatchPlan compiles resolved survivorship rules into an immutable Intermediate Representation (IR)
func BuildBatchPlan(cfg PlanConfig, resolvedRules map[string]ResolvedFieldRule) (*SurvivorshipBatchPlan, error) {
	if cfg.TenantID == uuid.Nil {
		return nil, errors.New("tenant_id is required")
	}
	cfg.EntityType = strings.ToUpper(strings.TrimSpace(cfg.EntityType))
	if cfg.EntityType == "" {
		return nil, errors.New("entity_type is required")
	}
	if cfg.StagingTable == "" {
		cfg.StagingTable = fmt.Sprintf("staging.%s_data", strings.ToLower(cfg.EntityType))
	}
	if cfg.TargetTable == "" {
		cfg.TargetTable = fmt.Sprintf("mdm.%s_master", strings.ToLower(cfg.EntityType))
	}
	if cfg.EntityKeyField == "" {
		cfg.EntityKeyField = fmt.Sprintf("%s_cd", strings.ToLower(cfg.EntityType))
	}
	if cfg.SourceIDField == "" {
		cfg.SourceIDField = "source_cd"
	}
	if cfg.AsOfField == "" {
		cfg.AsOfField = "as_of"
	}
	if cfg.PKField == "" {
		cfg.PKField = "id"
	}
	if cfg.BatchCutoff.IsZero() {
		cfg.BatchCutoff = time.Now().UTC()
	}

	plan := &SurvivorshipBatchPlan{
		TenantID:       cfg.TenantID,
		EntityType:     cfg.EntityType,
		StagingTable:   cfg.StagingTable,
		TargetTable:    cfg.TargetTable,
		EntityKeyField: cfg.EntityKeyField,
		SourceIDField:  cfg.SourceIDField,
		AsOfField:      cfg.AsOfField,
		PKField:        cfg.PKField,
		BatchCutoff:    cfg.BatchCutoff,
		Fields:         make([]FieldPlan, 0, len(resolvedRules)),
	}

	// Sort attribute names alphabetically to ensure strictly deterministic IR generation
	attrNames := make([]string, 0, len(resolvedRules))
	for k := range resolvedRules {
		attrNames = append(attrNames, k)
	}
	sort.Strings(attrNames)

	for _, attr := range attrNames {
		rule := resolvedRules[attr]
		strat := StrategyType(strings.ToUpper(strings.TrimSpace(rule.Strategy)))
		if strat == "" {
			strat = StrategySourcePriority
		}

		fieldPlan := FieldPlan{
			Attribute:       rule.AttributeName,
			SourceColumn:    rule.AttributeName,
			TargetColumn:    rule.AttributeName,
			Strategy:        strat,
			PriorityOrder:   rule.PriorityOrder,
			SelectionRuleID: rule.SelectionRuleID,
			SelectionMode:   rule.SelectionMode,
			ResolutionTier:  rule.ResolutionTier,
			RuleID:          rule.ID,
		}

		if rule.SelectionRuleID != nil && *rule.SelectionRuleID != "" {
			return nil, fmt.Errorf("attribute %q specifies selection_rule_id %q: dynamic selection tier is deferred; cutover is blocked for this rule", attr, *rule.SelectionRuleID)
		}

		if rule.MaxStaleSeconds > 0 {
			secs := rule.MaxStaleSeconds
			fieldPlan.MaxStaleSeconds = &secs
		}

		// Compile deterministic tiebreaker chain per Spec §6
		switch strat {
		case StrategySourcePriority:
			fieldPlan.TiebreakerChain = []string{
				fmt.Sprintf("array_position(ARRAY[%s], %s) NULLS LAST", formatPriorityArray(rule.PriorityOrder), cfg.SourceIDField),
				fmt.Sprintf("%s DESC", cfg.AsOfField),
				fmt.Sprintf("%s ASC", cfg.PKField),
			}

		case StrategyMostRecent:
			fieldPlan.TiebreakerChain = []string{
				fmt.Sprintf("%s DESC", cfg.AsOfField),
				fmt.Sprintf("%s ASC", cfg.PKField),
			}

		case StrategyConservativeMin:
			fieldPlan.TiebreakerChain = []string{
				fmt.Sprintf("%s ASC", fieldPlan.SourceColumn),
				fmt.Sprintf("%s DESC", cfg.AsOfField),
				fmt.Sprintf("%s ASC", cfg.PKField),
			}

		case StrategyConservativeMax:
			fieldPlan.TiebreakerChain = []string{
				fmt.Sprintf("%s DESC", fieldPlan.SourceColumn),
				fmt.Sprintf("%s DESC", cfg.AsOfField),
				fmt.Sprintf("%s ASC", cfg.PKField),
			}

		case StrategyMostFrequent:
			fieldPlan.TiebreakerChain = []string{
				"COUNT(*) DESC",
				fmt.Sprintf("%s::text ASC", fieldPlan.SourceColumn),
				fmt.Sprintf("%s DESC", cfg.AsOfField),
				fmt.Sprintf("%s ASC", cfg.PKField),
			}

		default:
			// Fallback tiebreaker chain
			fieldPlan.TiebreakerChain = []string{
				fmt.Sprintf("%s DESC", cfg.AsOfField),
				fmt.Sprintf("%s ASC", cfg.PKField),
			}
		}

		plan.Fields = append(plan.Fields, fieldPlan)
	}

	return plan, nil
}

func formatPriorityArray(priorities []string) string {
	if len(priorities) == 0 {
		return "''"
	}
	quoted := make([]string, len(priorities))
	for i, p := range priorities {
		quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(p, "'", "''"))
	}
	return strings.Join(quoted, ", ")
}
