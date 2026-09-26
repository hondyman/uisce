package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// RuleFailure is one rule a row failed.
type RuleFailure struct {
	RuleID   string
	RuleName string
	Severity string // models.ValidationRuleSeverityBlock | Warn
	Message  string
}

// RuleChecker evaluates catalog rules against one row. It is the seam between
// the pipeline and the rule engine so the pipeline stays testable.
type RuleChecker interface {
	Check(ctx context.Context, tenantID string, ruleIDs []string, data map[string]any) ([]RuleFailure, error)
}

// RuleSource loads a validation rule the tenant may see (its own, or a core
// rule inherited from the gold copy). analytics.ValidationRuleService
// implements it - the same catalog_node rule path BO writes enforce.
type RuleSource interface {
	GetByIDForTenant(ctx context.Context, tenantID string, id uuid.UUID) (*models.ValidationRuleDescriptor, error)
}

// CatalogRuleChecker evaluates catalog validation rules on the rule engine
// (internal/rules/vm). Rules are loaded and parsed once per tenant+rule set
// for the life of the checker; each run gets its own (ForRun), so an edited
// rule takes effect on the next run and the cache never outlives a run.
type CatalogRuleChecker struct {
	Rules RuleSource

	mu    sync.Mutex
	cache map[string][]loadedRule
}

type loadedRule struct {
	id, name, severity string
	node               vm.RuleNode
	fields             []string // top-level fields the rule reads
}

// ForRun returns a checker with an empty cache sharing the rule source.
func (c *CatalogRuleChecker) ForRun() RuleChecker { return &CatalogRuleChecker{Rules: c.Rules} }

// perRunChecker is implemented by checkers that hold per-run state.
type perRunChecker interface{ ForRun() RuleChecker }

func (c *CatalogRuleChecker) load(ctx context.Context, tenantID string, ids []string) ([]loadedRule, error) {
	key := tenantID + "|" + strings.Join(ids, ",")
	c.mu.Lock()
	defer c.mu.Unlock()
	if rs, ok := c.cache[key]; ok {
		return rs, nil
	}
	rs := make([]loadedRule, 0, len(ids))
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("rule id %q is not a uuid", raw)
		}
		desc, err := c.Rules.GetByIDForTenant(ctx, tenantID, id)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", raw, err)
		}
		if !desc.IsActive {
			return nil, fmt.Errorf("rule %q is inactive", desc.Name)
		}
		var node vm.RuleNode
		if err := json.Unmarshal(desc.RuleAST, &node); err != nil {
			return nil, fmt.Errorf("rule %q: stored rule_ast did not parse: %w", desc.Name, err)
		}
		rs = append(rs, loadedRule{id: raw, name: desc.Name, severity: strings.ToUpper(desc.Severity), node: node, fields: topLevel(vm.FieldRefs(node))})
	}
	if c.cache == nil {
		c.cache = map[string][]loadedRule{}
	}
	c.cache[key] = rs
	return rs, nil
}

func (c *CatalogRuleChecker) Check(ctx context.Context, tenantID string, ids []string, data map[string]any) ([]RuleFailure, error) {
	rs, err := c.load(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	ev := vm.NewAdvancedEvaluator()
	var out []RuleFailure
	for _, r := range rs {
		// A field the rule reads that is not in the row is never a pass or a
		// fail of the rule: the row isn't in the rule's terms (e.g. staging
		// columns, not business object fields). Reject it and say which.
		if missing := missingFields(r.fields, data); len(missing) > 0 {
			out = append(out, RuleFailure{RuleID: r.id, RuleName: r.name, Severity: models.ValidationRuleSeverityBlock,
				Message: fmt.Sprintf("reads %s, which this row does not have", strings.Join(missing, ", "))})
			continue
		}
		ok, err := ev.Evaluate(r.node, data)
		switch {
		case err != nil:
			// An unevaluable rule never reads as a pass.
			out = append(out, RuleFailure{RuleID: r.id, RuleName: r.name, Severity: models.ValidationRuleSeverityBlock,
				Message: fmt.Sprintf("could not be evaluated: %v", err)})
		case !ok:
			out = append(out, RuleFailure{RuleID: r.id, RuleName: r.name, Severity: r.severity, Message: "condition not met"})
		}
	}
	return out, nil
}

type ruleCheckProc struct {
	cfg     RuleCheckConfig
	checker RuleChecker
	tenant  string
}

func newRuleCheckProc(n Node, ch RuleChecker) (Processor, error) {
	var c RuleCheckConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, fmt.Errorf("no rules engine is configured for this environment")
	}
	if pr, ok := ch.(perRunChecker); ok {
		ch = pr.ForRun()
	}
	return &ruleCheckProc{cfg: c, checker: ch}, nil
}

func (p *ruleCheckProc) Open(_ context.Context, rc *RunContext) error {
	p.tenant = rc.TenantID
	return nil
}
func (p *ruleCheckProc) Close(context.Context, error) error { return nil }

// blocking: BLOCK rejects the row; anything else (WARN) keeps it and records
// a warning.
func blocking(sev string) bool { return strings.EqualFold(sev, models.ValidationRuleSeverityBlock) }

func (p *ruleCheckProc) Process(ctx context.Context, rows []Row) (Result, error) {
	var res Result
	for _, r := range rows {
		fails, err := p.checker.Check(ctx, p.tenant, p.cfg.RuleIDs, r.Data)
		if err != nil {
			return res, err
		}
		block := false
		for _, f := range fails {
			name := f.RuleName
			if name == "" {
				name = f.RuleID
			}
			rj := Reject{Row: r, Field: name, Reason: fmt.Sprintf("rule %q failed: %s", name, f.Message)}
			if blocking(f.Severity) {
				res.Rejected = append(res.Rejected, rj)
				block = true
				break
			}
			res.Warnings = append(res.Warnings, rj)
		}
		if !block {
			res.Out = append(res.Out, r)
		}
	}
	return res, nil
}

// topLevel drops nested paths (a.b), which resolve through related data
// rather than the row itself.
func topLevel(fields []string) []string {
	out := fields[:0:0]
	for _, f := range fields {
		if !strings.Contains(f, ".") {
			out = append(out, f)
		}
	}
	return out
}

func missingFields(fields []string, data map[string]any) []string {
	var missing []string
	for _, f := range fields {
		if _, ok := data[f]; !ok {
			missing = append(missing, f)
		}
	}
	return missing
}
