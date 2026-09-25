package datapipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/rules"
)

// RuleFailure is one rule a row failed.
type RuleFailure struct {
	RuleID   string
	RuleName string
	Severity string
	Message  string
}

// RuleChecker evaluates catalog rules against one row. It is the seam between
// the pipeline and the rules engine so the pipeline stays testable and the
// engine can change (AST/VM today, WASM later) without touching the spec.
type RuleChecker interface {
	Check(ctx context.Context, tenantID string, ruleIDs []string, data map[string]any) ([]RuleFailure, error)
}

// RuleLoader resolves rule ids to compiled-ready rules for a tenant. It is
// injected at wiring time because rules live in several repositories
// (tenant validation rules, core rules, compliance rules).
type RuleLoader func(ctx context.Context, tenantID string, ruleIDs []string) ([]*rules.RuleWithMetadata, error)

// EngineRuleChecker adapts *rules.RuleEngine.
type EngineRuleChecker struct {
	Engine *rules.RuleEngine
	Load   RuleLoader
	cache  map[string][]*rules.RuleWithMetadata
}

func (c *EngineRuleChecker) Check(ctx context.Context, tenantID string, ids []string, data map[string]any) ([]RuleFailure, error) {
	key := tenantID + "|" + strings.Join(ids, ",")
	if c.cache == nil {
		c.cache = map[string][]*rules.RuleWithMetadata{}
	}
	rs, ok := c.cache[key]
	if !ok {
		var err error
		if rs, err = c.Load(ctx, tenantID, ids); err != nil {
			return nil, fmt.Errorf("loading rules: %w", err)
		}
		if len(rs) != len(ids) {
			return nil, fmt.Errorf("expected %d rules, found %d (check the rule ids and tenant)", len(ids), len(rs))
		}
		c.cache[key] = rs
	}
	res := c.Engine.EvaluateBatch(ctx, tenantID, rs, data)
	var out []RuleFailure
	for _, r := range res.Results {
		if r == nil || r.Passed {
			continue
		}
		msg := strings.Join(r.FailureReasons, "; ")
		if msg == "" && len(r.Violations) > 0 {
			msg = r.Violations[0].Message
		}
		out = append(out, RuleFailure{RuleID: r.RuleID, RuleName: r.RuleName, Severity: string(r.Severity), Message: msg})
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
	return &ruleCheckProc{cfg: c, checker: ch}, nil
}

func (p *ruleCheckProc) Open(_ context.Context, rc *RunContext) error {
	p.tenant = rc.TenantID
	return nil
}
func (p *ruleCheckProc) Close(context.Context, error) error { return nil }

func blocking(sev string) bool {
	switch rules.Severity(sev) {
	case rules.SeverityError, rules.SeverityHardBlock, rules.SeverityQuarantine:
		return true
	}
	return false
}

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
