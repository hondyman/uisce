// Package mdmrules is the validation-rule catalog for the tier 1 and 2 MDM business objects.
//
// Every rule is authored against semantic terms (see Vocabulary), never physical columns. At
// execution the engine resolves each term to the column that represents it under the active binding
// (internal/analytics.ResolveSemanticFieldMap), so one rule applies consistently to every binding of a
// BO. A rule may instead be scoped to the bindings the BO has at seed time (Scope == ScopeCurrentBindings), for
// rules that only make sense on the mastering binding.
package mdmrules

import (
	"fmt"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// Scope says which bindings of the BO a rule applies to.
type Scope int

const (
	// ScopeAll applies the rule to every binding of the BO (no binding scope on the rule).
	ScopeAll Scope = iota
	// ScopeCurrentBindings scopes the rule to the bindings the BO has when the rule is seeded. A binding
	// added later does not inherit it, which is the point: these are sourcing rules for the mastering
	// binding, not for every projection of the BO.
	ScopeCurrentBindings
)

// Case is one example record and whether the rule should pass it.
type Case struct {
	Name   string
	Record map[string]any
	Pass   bool
}

// Rule is one catalog entry.
type Rule struct {
	BO          string
	Name        string
	Description string
	Severity    string // "BLOCK" | "WARN"
	Timing      string // "pre_write" | "reconcile"
	Category    string
	Scope       Scope
	AST         vm.RuleNode
	Cases       []Case
}

const (
	blockSev = "BLOCK"
	warnSev  = "WARN"

	catIntegrity = "mdm_integrity"
	catSourcing  = "mdm_sourcing"
)

// --- AST constructors -------------------------------------------------------------------------

func cond(field, op string, value any) vm.RuleNode {
	vt := ""
	switch value.(type) {
	case float64:
		vt = "number"
	case bool:
		vt = "boolean"
	case string:
		vt = "string"
	}
	return vm.RuleNode{Type: vm.NodeTypeCondition, Condition: &vm.RuleCondition{
		Field: field, FieldPath: field, Operator: op, Value: value, ValueType: vt}}
}

func group(op string, nodes ...vm.RuleNode) vm.RuleNode {
	return vm.RuleNode{Type: vm.NodeTypeGroup, Group: &vm.RuleGroup{Operator: op, Conditions: nodes}}
}

func notNull(field string) vm.RuleNode { return cond(field, "is_not_null", nil) }
func isNull(field string) vm.RuleNode  { return cond(field, "is_null", nil) }

// geField builds the numeric comparison "a >= b" between two terms.
func geField(a, b string) vm.RuleNode {
	return vm.RuleNode{Type: vm.NodeTypeExpression, Expression: &vm.Expression{
		Root: &vm.BinaryExpr{Op: ">=", Left: &vm.FieldRef{Path: a}, Right: &vm.FieldRef{Path: b}}}}
}

// --- rule shapes -------------------------------------------------------------------------------

// required: every listed term must be present. One rule per BO, so a failure names the BO once.
func required(bo, description string, scope Scope, category string, ts ...string) Rule {
	nodes := make([]vm.RuleNode, len(ts))
	full := map[string]any{}
	for i, t := range ts {
		nodes[i] = notNull(t)
		full[t] = "x"
	}
	cases := []Case{{Name: "all present", Record: full, Pass: true}}
	for _, t := range ts {
		rec := map[string]any{}
		for k, v := range full {
			rec[k] = v
		}
		rec[t] = nil
		cases = append(cases, Case{Name: t + " missing", Record: rec, Pass: false})
	}
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.required_terms", bo), Description: description,
		Severity: blockSev, Timing: "pre_write", Category: category, Scope: scope,
		AST: group("AND", nodes...), Cases: cases}
}

// inRange: a nullable numeric term, when present, must lie in [lo, hi].
func inRange(bo, term string, lo, hi float64) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s_range", bo, snake(term)),
		Description: fmt.Sprintf("%s must be between %v and %v when set.", term, lo, hi),
		Severity:    warnSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("OR", isNull(term), group("AND", cond(term, ">=", lo), cond(term, "<=", hi))),
		Cases: []Case{
			{"null allowed", map[string]any{term: nil}, true},
			{"lower bound", map[string]any{term: lo}, true},
			{"upper bound", map[string]any{term: hi}, true},
			{"below range", map[string]any{term: lo - 1}, false},
			{"above range", map[string]any{term: hi + 1}, false},
		}}
}

// positive: a nullable numeric term, when present, must be > 0.
func positive(bo, term string) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s_positive", bo, snake(term)),
		Description: term + " must be greater than zero when set.",
		Severity:    warnSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("OR", isNull(term), cond(term, ">", float64(0))),
		Cases: []Case{
			{"null allowed", map[string]any{term: nil}, true},
			{"positive", map[string]any{term: float64(1)}, true},
			{"zero", map[string]any{term: float64(0)}, false},
			{"negative", map[string]any{term: float64(-5)}, false},
		}}
}

// atLeast: a required numeric term must be >= min.
func atLeast(bo, term string, min float64) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s_at_least_%v", bo, snake(term), min),
		Description: fmt.Sprintf("%s must be at least %v.", term, min),
		Severity:    blockSev, Timing: "pre_write", Category: catIntegrity,
		AST: cond(term, ">=", min),
		Cases: []Case{
			{"at minimum", map[string]any{term: min}, true},
			{"above minimum", map[string]any{term: min + 1}, true},
			{"below minimum", map[string]any{term: min - 1}, false},
		}}
}

// ordered: hi >= lo, checked only when both are present (both may be nullable).
func ordered(bo, name, description string, hi, lo string) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s", bo, name), Description: description,
		Severity: blockSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("OR", isNull(hi), isNull(lo), geField(hi, lo)),
		Cases: []Case{
			{"ordered", map[string]any{hi: float64(90), lo: float64(70)}, true},
			{"equal", map[string]any{hi: float64(70), lo: float64(70)}, true},
			{"hi missing", map[string]any{hi: nil, lo: float64(70)}, true},
			{"lo missing", map[string]any{hi: float64(90), lo: nil}, true},
			{"inverted", map[string]any{hi: float64(50), lo: float64(70)}, false},
		}}
}

// chain3: a >= b >= c, all required numeric terms.
func chain3(bo, name, description, a, b, c string) Rule {
	ok := map[string]any{a: float64(90), b: float64(70), c: float64(30)}
	with := func(k string, v float64) map[string]any {
		m := map[string]any{}
		for kk, vv := range ok {
			m[kk] = vv
		}
		m[k] = v
		return m
	}
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s", bo, name), Description: description,
		Severity: blockSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("AND", geField(a, b), geField(b, c)),
		Cases: []Case{
			{"ordered", ok, true},
			{a + " below " + b, with(a, 60), false},
			{b + " below " + c, with(b, 20), false},
		}}
}

// requiredIf: when boolTerm is true, needed must be present.
func requiredIf(bo, name, description, boolTerm, needed string) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s", bo, name), Description: description,
		Severity: blockSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("OR", cond(boolTerm, "==", false), notNull(needed)),
		Cases: []Case{
			{"flag off, value absent", map[string]any{boolTerm: false, needed: nil}, true},
			{"flag on, value present", map[string]any{boolTerm: true, needed: "x"}, true},
			{"flag on, value absent", map[string]any{boolTerm: true, needed: nil}, false},
		}}
}

// bothOrNeither: a and b must be set together or not at all. Used where two terms describe one fact
// (who published and when) so a half-filled pair is caught without depending on a status vocabulary.
func bothOrNeither(bo, name, description, a, b string) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s", bo, name), Description: description,
		Severity: blockSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("OR", group("AND", isNull(a), isNull(b)), group("AND", notNull(a), notNull(b))),
		Cases: []Case{
			{"neither set", map[string]any{a: nil, b: nil}, true},
			{"both set", map[string]any{a: "x", b: "y"}, true},
			{a + " only", map[string]any{a: "x", b: nil}, false},
			{b + " only", map[string]any{a: nil, b: "y"}, false},
		}}
}

// Catalog returns the tier 1 and 2 rule set.
//
// Deliberately not rules: anything on Status, Priority or an *IsActive term (enumerations and UX
// controls own those); anything on IssuerId (every issuer_* table maps its own id to that term, so it
// is ambiguous where the table also has an issuer FK); and the checks the engine cannot express yet
// (see docs/mdm-rules.md, "Engine gaps").
func Catalog() []Rule {
	var r []Rule
	add := func(x ...Rule) { r = append(r, x...) }

	// ---- tier 1: required terms -----------------------------------------------------------
	add(
		required("party", "A party needs a code, legal name and type.", ScopeAll, catIntegrity, "PartyCode", "LegalName", "PartyType"),
		required("portfolio", "A portfolio needs a code, name and type.", ScopeAll, catIntegrity, "PortfolioCode", "PortfolioName", "PortfolioType"),
		required("mandate", "A mandate needs a code and a portfolio.", ScopeAll, catIntegrity, "MandateCode", "PortfolioId"),
		required("portfolio_composite", "A composite needs a code and a name.", ScopeAll, catIntegrity, "CompositeCode", "PortfolioName"),
		required("source_system", "A source system needs a code, name and type.", ScopeAll, catIntegrity, "SourceCode", "SourceName", "SystemType"),
		required("steward", "A steward needs a name and an email.", ScopeAll, catIntegrity, "StewardName", "StewardEmail"),
		required("issuer_steward", "An issuer steward needs a user and a role.", ScopeAll, catIntegrity, "UserId", "StewardRole"),
		required("hierarchy", "A hierarchy needs a type and a name.", ScopeAll, catIntegrity, "HierarchyType", "HierarchyName"),
		required("issuer", "An issuer needs a code and a name.", ScopeAll, catIntegrity, "IssuerCode", "IssuerName"),
		required("benchmark", "A benchmark needs a code, name, type and currency.", ScopeAll, catIntegrity, "BenchmarkCode", "BenchmarkName", "BenchmarkType", "Currency"),
	)
	// ---- tier 1: numeric ------------------------------------------------------------------
	add(
		atLeast("issuer_golden_record", "GoldenVersion", 1),
		bothOrNeither("issuer_golden_record", "publication_recorded_completely",
			"PublishedAt and PublishedBy must be recorded together: a half-published golden record is inconsistent.", "PublishedAt", "PublishedBy"),
		inRange("issuer_golden_record", "OverallDqScore", 0, 100),
		inRange("issuer_golden_record", "IdentityConfidence", 0, 100),
		inRange("issuer_golden_record", "HierarchyConfidence", 0, 100),
		inRange("issuer", "DqScore", 0, 100),
	)
	// ---- tier 1: MDM-binding-scoped sourcing ----------------------------------------------
	add(required("issuer", "An issuer mastered through the MDM binding must record the system it was sourced from.",
		ScopeCurrentBindings, catSourcing, "SourceSystemIdentifier"))
	r[len(r)-1].Name = "mdm.issuer.sourced_from_a_system"

	// ---- tier 2: rule/config tables -------------------------------------------------------
	add(
		chain3("issuer_match_rule", "thresholds_ordered", "AutoMatch threshold must be >= Review threshold >= NoMatch threshold.",
			"ThresholdAutoMatch", "ThresholdReview", "ThresholdNoMatch"),
		inRange("issuer_match_rule", "ThresholdAutoMatch", 0, 100),
		inRange("issuer_match_rule", "ThresholdReview", 0, 100),
		inRange("issuer_match_rule", "ThresholdNoMatch", 0, 100),
		required("issuer_match_rule", "An issuer match rule needs a code and an issuer type.", ScopeAll, catIntegrity, "RuleCode", "IssuerType"),

		ordered("match_rule", "auto_merge_at_least_review", "AutoMergeThreshold must be >= ReviewThreshold when both are set.",
			"AutoMergeThreshold", "ReviewThreshold"),
		inRange("match_rule", "Threshold", 0, 1),

		inRange("survivorship_rule", "AnomalyTolerancePercent", 0, 100),
		positive("survivorship_rule", "StalenessMaxAgeSec"),
		inRange("issuer_survivorship_rule", "MinConfidence", 0, 100),
		positive("issuer_survivorship_rule", "MaxStalenessHours"),

		required("dq_rule", "A DQ rule needs a name, entity type, check expression and severity.", ScopeAll, catIntegrity,
			"RuleName", "EntityType", "CheckExpression", "Severity"),
		required("issuer_hierarchy_rule", "An issuer hierarchy rule needs a code, expression and severity.", ScopeAll, catIntegrity,
			"RuleCode", "RuleExpression", "Severity"),
	)
	// ---- tier 2: source mappings ----------------------------------------------------------
	add(
		required("issuer_field_mapping", "A field mapping needs the vendor field and the internal table and field.", ScopeAll, catIntegrity,
			"VendorField", "InternalTable", "InternalField"),
		required("issuer_identifier_authority", "An identifier authority needs the identifier type.", ScopeAll, catIntegrity, "IdType"),
		requiredIf("issuer_identifier_authority", "checksum_algorithm_when_required",
			"When an identifier requires a checksum, the checksum algorithm must be named.", "RequiresChecksum", "ChecksumAlgorithm"),
		required("issuer_type_mapping", "A type mapping needs the vendor type and the internal issuer type.", ScopeAll, catIntegrity,
			"VendorTypeCd", "InternalIssuerType"),
		inRange("issuer_type_mapping", "Confidence", 0, 100),
	)
	// ---- tier 2: stewardship queues -------------------------------------------------------
	add(
		inRange("match_candidate", "Score", 0, 1),
		inRange("issuer_match_candidate", "OverallScore", 0, 100),
		required("change_request", "A change request needs the entity, type, proposed changes and submitter.", ScopeAll, catIntegrity,
			"EntityId", "EntityType", "ChangeType", "ProposedChanges", "SubmittedBy"),
		required("issuer_change_request", "An issuer change request needs a reference, type, changes and requester.", ScopeAll, catIntegrity,
			"RequestReference", "ChangeType", "RequestedChanges", "RequestedBy"),
		required("dq_issue", "A DQ issue needs the entity, the rule, a description and a severity.", ScopeAll, catIntegrity,
			"EntityId", "RuleId", "IssueDescription", "Severity"),
		required("issuer_exception", "An issuer exception needs a type, severity and description.", ScopeAll, catIntegrity,
			"ExceptionType", "Severity", "ExceptionDescription"),
		required("issuer_hierarchy_review", "A hierarchy review needs the parent, child and change type.", ScopeAll, catIntegrity,
			"ParentIssuerId", "ChildIssuerId", "ChangeType"),
		inRange("issuer_hierarchy_review", "Confidence", 0, 100),
	)
	return r
}

func snake(s string) string {
	var out []rune
	for i, c := range s {
		if i > 0 && c >= 'A' && c <= 'Z' && !(s[i-1] >= 'A' && s[i-1] <= 'Z') {
			out = append(out, '_')
		}
		out = append(out, c|0x20)
	}
	return string(out)
}
