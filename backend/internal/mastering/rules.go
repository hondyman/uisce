package mastering

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// RuleLister lists a business object's validation rules as the tenant sees
// them (its own plus the gold copy's core rules).
// analytics.ValidationRuleService implements it.
type RuleLister interface {
	ListByBO(ctx context.Context, tenantID, boName, domain string) ([]models.ValidationRuleDescriptor, error)
}

type rule struct {
	id, name, severity string
	node               vm.RuleNode
	fields             []string
}

// RuleSet is the BO's active validation rules, parsed once per run. The same
// rules run at every stage, in BO field names (the terms they were written
// in): on each canonical record, only the rules whose fields the source
// supplies (a source that doesn't send a field can't fail a rule on it);
// on each golden record, all of them.
type RuleSet struct{ rules []rule }

func loadRules(ctx context.Context, l RuleLister, tenantID, boKey string) (*RuleSet, error) {
	descs, err := l.ListByBO(ctx, tenantID, boKey, "")
	if err != nil {
		return nil, fmt.Errorf("loading %s rules: %w", boKey, err)
	}
	rs := &RuleSet{}
	for _, d := range descs {
		if !d.IsActive {
			continue
		}
		var node vm.RuleNode
		if err := json.Unmarshal(d.RuleAST, &node); err != nil {
			return nil, fmt.Errorf("rule %q: stored rule_ast did not parse: %w", d.Name, err)
		}
		var fields []string
		for _, f := range vm.FieldRefs(node) {
			if !strings.Contains(f, ".") {
				fields = append(fields, f)
			}
		}
		rs.rules = append(rs.rules, rule{id: d.ID.String(), name: d.Name, severity: strings.ToUpper(d.Severity), node: node, fields: fields})
	}
	return rs, nil
}

// Canonical checks a record (in BO field names): only rules whose every
// field is present.
func (rs *RuleSet) Canonical(data map[string]any) []Issue {
	return rs.check(data, false)
}

// Golden checks a golden record: every rule; a field the golden record has
// no value for reads as null (so "name is present" fails, as it should).
func (rs *RuleSet) Golden(data map[string]any) []Issue {
	return rs.check(data, true)
}

func (rs *RuleSet) check(data map[string]any, all bool) []Issue {
	if rs == nil {
		return nil
	}
	ev := vm.NewAdvancedEvaluator()
	var out []Issue
	for _, r := range rs.rules {
		row := data
		if all {
			row = make(map[string]any, len(data)+len(r.fields))
			for k, v := range data {
				row[k] = v
			}
			for _, f := range r.fields {
				if _, ok := row[f]; !ok {
					row[f] = nil
				}
			}
		} else if !hasAll(data, r.fields) {
			continue
		}
		sev := SevWarning
		if r.severity == models.ValidationRuleSeverityBlock {
			sev = SevError
		}
		ok, err := ev.Evaluate(r.node, row)
		switch {
		case err != nil:
			// An unevaluable rule never reads as a pass.
			out = append(out, Issue{Code: "RULE", Severity: SevError, RuleID: r.id, Message: fmt.Sprintf("%s could not be evaluated: %v", r.name, err)})
		case !ok:
			out = append(out, Issue{Code: "RULE", Severity: sev, RuleID: r.id, Message: r.name + ": condition not met"})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Severity < out[j].Severity })
	return out
}

func hasAll(data map[string]any, fields []string) bool {
	for _, f := range fields {
		if v, ok := data[f]; !ok || v == nil {
			return false
		}
	}
	return true
}

// inFields renames golden attributes to BO field names for rule checks.
// References are presented under their code attribute, which is what the
// rule author sees on the record.
func inFields(attrs map[string]any, attrField map[string]string) map[string]any {
	out := make(map[string]any, len(attrs))
	for a, v := range attrs {
		if f, ok := attrField[a]; ok {
			out[f] = v
		}
	}
	return out
}
