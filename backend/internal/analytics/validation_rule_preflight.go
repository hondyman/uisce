package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// existingRuleIndex captures the target tenant's own (custom) validation rules
// keyed by rule_key — the baseline for diff computation and prune decisions.
// Core-inherited rules are deliberately excluded: they are never mutated by import.
type existingRuleIndex struct {
	byKey map[string]existingRule
}

type existingRule struct {
	RuleKey      string
	Name         string
	BOName       string
	Domain       string
	Severity     string
	Timing       string
	Category     string
	Governance   string
	Description  string
	CanonicalAST string // vm.Compact output; "" if missing/unparseable
}

// Preflight validates an import request against the target tenant catalog and
// produces the planned diff without any database mutation. It is the sole
// authority for both dry-run reports and the gate before the transactional
// import in Task 5.
func (p *ValidationRulePorter) Preflight(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool) (*models.RuleImportReport, error) {
	bundle := req.Bundle
	report := &models.RuleImportReport{
		DryRun:     req.DryRun,
		TotalRules: len(bundle.Rules),
		Created:    []string{},
		Updated:    []string{},
		Skipped:    []string{},
		Errors:     []models.RuleImportErrorDetail{},
	}

	// --- Bundle-level validation (no DB) ---
	switch req.OverwritePolicy {
	case models.ImportOverwriteFail, models.ImportOverwriteSkip, models.ImportOverwriteOverwrite:
	default:
		return nil, fmt.Errorf("invalid overwrite_policy %q (want %q, %q, %q)",
			req.OverwritePolicy, models.ImportOverwriteFail, models.ImportOverwriteSkip, models.ImportOverwriteOverwrite)
	}
	if err := bundle.VerifyChecksum(); err != nil {
		return nil, fmt.Errorf("bundle checksum verification failed: %w", err)
	}
	if req.TargetTenantID == "" {
		return nil, fmt.Errorf("target_tenant_id is required")
	}

	// Deduplicate rule_keys inside the bundle.
	seen := map[string]int{}
	for i := range bundle.Rules {
		k := bundle.Rules[i].RuleKey
		if k == "" {
			report.Errors = append(report.Errors, models.RuleImportErrorDetail{
				Code: models.ImportErrInvalidField, Reason: "rule_key is empty",
			})
			continue
		}
		if prev, dup := seen[k]; dup {
			report.Errors = append(report.Errors, models.RuleImportErrorDetail{
				Code: models.ImportErrDuplicateKey, RuleKey: k,
				Name: bundle.Rules[i].Name,
				Reason: fmt.Sprintf("rule_key %q appears at bundle positions %d and %d", k, prev, i),
			})
			continue
		}
		seen[k] = i
	}

	// --- Dependency graph: topo-sort with cycle detection (pure) ---
	ordered, depErrs := topoSortRules(bundle.Rules)
	report.Errors = append(report.Errors, depErrs...)

	// --- Enum validation (pure) ---
	for i := range bundle.Rules {
		spec := &bundle.Rules[i]
		for _, e := range validateSpecEnums(spec) {
			report.Errors = append(report.Errors, e)
		}
		// Governance: compliance rules cannot enter as published without bypass.
		if spec.Domain == models.ValidationRuleDomainCompliance &&
			spec.GovernanceStatus == models.ValidationRuleGovernancePublished && !bypassGovernance {
			spec.GovernanceStatus = models.ValidationRuleGovernanceSubmittedForReview
			// Recorded implicitly: diff summary will show governance_status as
			// part of the plan. The apply step applies the same coercion, so
			// preflight diff and executed result stay consistent.
		}
	}
	if hasBlockingErrors(report) {
		return report, nil
	}

	// --- DB-backed pre-flight ---
	gold, err := goldCopyTenantID(ctx, p.db)
	if err != nil {
		return nil, err
	}
	visible := visibleTenants(req.TargetTenantID, gold)

	existing, err := p.loadExistingRuleIndex(ctx, req.TargetTenantID)
	if err != nil {
		return nil, fmt.Errorf("load existing rules: %w", err)
	}
	coreIndex, err := p.loadCoreRuleIndex(ctx, gold)
	if err != nil {
		return nil, fmt.Errorf("load core rules: %w", err)
	}

	// Resolve term sets lazily per unique BO.
	termSets := map[string]*boTermSet{}
	for i := range ordered {
		spec := &ordered[i]
		if spec.Deleted {
			// Tombstone: target must be an existing custom rule.
			if _, ok := existing.byKey[spec.RuleKey]; !ok {
				report.Errors = append(report.Errors, models.RuleImportErrorDetail{
					Code: models.ImportErrDeleteMissing, RuleKey: spec.RuleKey, Name: spec.Name,
					Reason: fmt.Sprintf("tombstone for rule_key %q matches no existing custom rule in tenant %s", spec.RuleKey, req.TargetTenantID),
				})
			} else {
				report.DiffSummary = append(report.DiffSummary, models.RuleDiffSummary{RuleKey: spec.RuleKey, Action: "DELETE"})
			}
			continue
		}

		// BO existence.
		ts, err := p.termSetFor(ctx, req.TargetTenantID, gold, visible, spec.BOName, termSets)
		if err != nil {
			return nil, err
		}
		if !ts.boExists {
			report.Errors = append(report.Errors, models.RuleImportErrorDetail{
				Code: models.ImportErrMissingBO, RuleKey: spec.RuleKey, Name: spec.Name,
				Reason: fmt.Sprintf("business object %q does not exist for tenant %s (or inherited core)", spec.BOName, req.TargetTenantID),
			})
			continue
		}

		// Semantic term resolution (fail loud).
		for _, reason := range unresolvedTerms(spec, ts) {
			report.Errors = append(report.Errors, models.RuleImportErrorDetail{
				Code: models.ImportErrUnresolvedTerm, RuleKey: spec.RuleKey, Name: spec.Name, Reason: reason,
			})
		}

		// Core shadow: custom rule may not reuse a core rule_key or name on the same BO.
		if spec.BOName != "" && coreIndex.shadows(spec.RuleKey, spec.Name, spec.BOName) {
			report.Errors = append(report.Errors, models.RuleImportErrorDetail{
				Code: models.ImportErrCoreShadow, RuleKey: spec.RuleKey, Name: spec.Name,
				Reason: fmt.Sprintf("custom rule shadows core rule (rule_key or name %q on BO %q); core rules are read-only in client tenants", spec.RuleKey, spec.BOName),
			})
		}

		// Diff against existing custom rule + overwrite policy.
		action, changes := diffAgainstExisting(spec, existing.byKey[spec.RuleKey], req.PreserveStatus)
		switch action {
		case "CREATE":
			report.DiffSummary = append(report.DiffSummary, models.RuleDiffSummary{RuleKey: spec.RuleKey, Action: action})
		case "NO_OP":
			report.DiffSummary = append(report.DiffSummary, models.RuleDiffSummary{RuleKey: spec.RuleKey, Action: action})
		case "UPDATE":
			if req.OverwritePolicy == models.ImportOverwriteSkip {
				report.DiffSummary = append(report.DiffSummary, models.RuleDiffSummary{RuleKey: spec.RuleKey, Action: "NO_OP"})
			} else if req.OverwritePolicy == models.ImportOverwriteFail {
				report.Errors = append(report.Errors, models.RuleImportErrorDetail{
					Code: models.ImportErrConflict, RuleKey: spec.RuleKey, Name: spec.Name,
					Reason: fmt.Sprintf("rule_key %q already exists with different content (changes: %s); policy is fail_on_conflict", spec.RuleKey, strings.Join(changes, ", ")),
				})
			} else {
				report.DiffSummary = append(report.DiffSummary, models.RuleDiffSummary{RuleKey: spec.RuleKey, Action: action, Changes: changes})
			}
		}
	}
	if hasBlockingErrors(report) {
		return report, nil
	}

	// --- Prune plan (opt-in) ---
	if req.Prune {
		inBundle := map[string]bool{}
		for i := range bundle.Rules {
			inBundle[bundle.Rules[i].RuleKey] = true
		}
		for k := range existing.byKey {
			if !inBundle[k] {
				report.DiffSummary = append(report.DiffSummary, models.RuleDiffSummary{RuleKey: k, Action: "DELETE"})
			}
		}
	}
	sortDiffSummary(report.DiffSummary)
	return report, nil
}

// hasBlockingErrors distinguishes "import cannot proceed" from
// "individual rules rejected, rest may proceed".
func hasBlockingErrors(r *models.RuleImportReport) bool {
	for _, e := range r.Errors {
		switch e.Code {
		case models.ImportErrDuplicateKey, models.ImportErrDepCycle, models.ImportErrInvalidField,
			models.ImportErrMissingBO, models.ImportErrUnresolvedTerm, models.ImportErrCoreShadow,
			models.ImportErrDeleteMissing, models.ImportErrConflict, models.ImportErrGovernance:
			// Per-rule errors: the rule is rejected; the import plan continues
			// for remaining rules. A Task-5 policy decision: default is
			// all-or-nothing transaction, so any error blocks execution.
			return true
		}
	}
	return false
}

// --- Pure helpers (unit-testable without DB) ---

// topoSortRules returns rules ordered so DependsOn precedents come first.
// Bundle-level errors (missing deps, cycles) are returned separately.
func topoSortRules(specs []models.PortableRuleSpec) ([]models.PortableRuleSpec, []models.RuleImportErrorDetail) {
	byKey := map[string]bool{}
	for i := range specs {
		byKey[specs[i].RuleKey] = true
	}
	var errs []models.RuleImportErrorDetail
	deps := map[string][]string{}
	for i := range specs {
		for _, d := range specs[i].DependsOn {
			if !byKey[d] {
				errs = append(errs, models.RuleImportErrorDetail{
					Code: models.ImportErrMissingDep, RuleKey: specs[i].RuleKey, Name: specs[i].Name,
					Reason: fmt.Sprintf("depends_on references %q which is not present in the bundle", d),
				})
			}
		}
		deps[specs[i].RuleKey] = specs[i].DependsOn
	}

	// Kahn's algorithm.
	inDegree := map[string]int{}
	dependents := map[string][]string{}
	for k, ds := range deps {
		if _, ok := inDegree[k]; !ok {
			inDegree[k] = 0
		}
		for _, d := range ds {
			if byKey[d] {
				inDegree[k]++
				dependents[d] = append(dependents[d], k)
			}
		}
	}
	var queue []string
	for k, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, k)
		}
	}
	sort.Strings(queue) // deterministic order
	pos := map[string]int{}
	for i := range specs {
		pos[specs[i].RuleKey] = i
	}
	var ordered []models.PortableRuleSpec
	for len(queue) > 0 {
		k := queue[0]
		queue = queue[1:]
		ordered = append(ordered, specs[pos[k]])
		for _, dep := range dependents[k] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}
	if len(ordered) != len(specs) {
		var cyclic []string
		for k, deg := range inDegree {
			if deg > 0 {
				cyclic = append(cyclic, k)
			}
		}
		sort.Strings(cyclic)
		errs = append(errs, models.RuleImportErrorDetail{
			Code: models.ImportErrDepCycle,
			Reason: fmt.Sprintf("dependency cycle detected among rule_keys: %s", strings.Join(cyclic, ", ")),
		})
		return specs, errs // original order; caller aborts on blocking errors
	}
	return ordered, errs
}

// validateSpecEnums checks severity/timing/domain/governance values.
func validateSpecEnums(spec *models.PortableRuleSpec) []models.RuleImportErrorDetail {
	var errs []models.RuleImportErrorDetail
	bad := func(field, val string) {
		errs = append(errs, models.RuleImportErrorDetail{
			Code: models.ImportErrInvalidField, RuleKey: spec.RuleKey, Name: spec.Name,
			Reason: fmt.Sprintf("invalid %s %q", field, val),
		})
	}
	switch spec.Severity {
	case models.ValidationRuleSeverityBlock, models.ValidationRuleSeverityWarn:
	default:
		bad("severity", spec.Severity)
	}
	switch spec.Timing {
	case models.ValidationRuleTimingPreWrite, models.ValidationRuleTimingReconcile:
	default:
		bad("timing", spec.Timing)
	}
	switch spec.Domain {
	case models.ValidationRuleDomainDefault, models.ValidationRuleDomainCompliance,
		models.ValidationRuleDomainMDM, models.ValidationRuleDomainSurvivorship:
	default:
		bad("domain", spec.Domain)
	}
	return errs
}

// unresolvedTerms returns one error string per semantic term in the rule's AST
// that cannot be resolved against the target BO's term set. Only top-level
// terms (no dot) are checked against bo fields; dotted paths resolve via
// collection keys (top segment) or context fields.
func unresolvedTerms(spec *models.PortableRuleSpec, ts *boTermSet) []string {
	if len(spec.RuleAST) == 0 {
		return []string{fmt.Sprintf("rule %q has no rule_ast", spec.RuleKey)}
	}
	var node vm.RuleNode
	if err := json.Unmarshal(spec.RuleAST, &node); err != nil {
		return []string{fmt.Sprintf("rule %q: unparseable rule_ast: %v", spec.RuleKey, err)}
	}
	refs := vm.FieldRefs(node)
	var out []string
	for _, ref := range refs {
		if ts.resolves(ref) {
			continue
		}
		out = append(out, fmt.Sprintf(
			"rule %q references term %q which does not exist on BO %q in tenant scope (checked: bo fields, collection keys, known context fields)",
			spec.RuleKey, ref, spec.BOName))
	}
	return out
}

// diffAgainstExisting classifies the planned action for one rule.
func diffAgainstExisting(spec *models.PortableRuleSpec, existing existingRule, preserveStatus bool) (string, []string) {
	if existing.RuleKey == "" {
		return "CREATE", nil
	}
	var changes []string
	if spec.Name != existing.Name {
		changes = append(changes, "name")
	}
	if spec.BOName != existing.BOName {
		changes = append(changes, "bo_name")
	}
	if spec.Severity != existing.Severity {
		changes = append(changes, "severity")
	}
	if spec.Timing != existing.Timing {
		changes = append(changes, "timing")
	}
	if spec.Domain != existing.Domain {
		changes = append(changes, "domain")
	}
	if spec.Category != existing.Category {
		changes = append(changes, "category")
	}
	if canonicalForDiff(spec.RuleAST) != existing.CanonicalAST {
		changes = append(changes, "rule_ast")
	}
	wantStatus := spec.GovernanceStatus
	if preserveStatus {
		wantStatus = existing.Governance
	}
	if wantStatus != existing.Governance {
		changes = append(changes, "governance_status")
	}
	if len(changes) == 0 {
		return "NO_OP", nil
	}
	return "UPDATE", changes
}

func canonicalForDiff(ast json.RawMessage) string {
	if len(ast) == 0 {
		return ""
	}
	var node vm.RuleNode
	if err := json.Unmarshal(ast, &node); err != nil {
		return string(ast) // unparseable; raw comparison forces an UPDATE
	}
	c, err := vm.Compact(node)
	if err != nil {
		return string(ast)
	}
	return string(c)
}

// --- DB loaders ---

// boTermSet is the resolved vocabulary for one BO in the target tenant scope.
type boTermSet struct {
	boExists bool
	fields   map[string]struct{} // business_object_fields (tenant + gold inherited)
}

func (ts *boTermSet) resolves(term string) bool {
	top := term
	if i := strings.Index(term, "."); i >= 0 {
		top = term[:i]
	}
	if _, ok := ts.fields[top]; ok {
		return true
	}
	if _, ok := ts.fields[term]; ok {
		return true
	}
	for _, c := range models.CollectionKeysForBO(topOrEmpty(top)) {
		if c == term || c == top {
			return true
		}
	}
	return models.IsKnownContextField(term)
}

func topOrEmpty(s string) string { return s }

// termSetFor loads (and memoizes) the term set for a BO.
func (p *ValidationRulePorter) termSetFor(ctx context.Context, tenantID, gold string, visible []string, boName string, cache map[string]*boTermSet) (*boTermSet, error) {
	if ts, ok := cache[boName]; ok {
		return ts, nil
	}
	ts := &boTermSet{fields: map[string]struct{}{}}
	if boName == "" {
		cache[boName] = ts
		return ts, nil
	}
	var n int
	if err := sqlx.GetContext(ctx, p.db, &n, `
		SELECT COUNT(*) FROM business_objects
		WHERE (bo_key = $1 OR bo_name = $1) AND tenant_id = ANY($2::uuid[])
	`, boName, pq.Array(visible)); err != nil {
		return nil, fmt.Errorf("check BO %q existence: %w", boName, err)
	}
	ts.boExists = n > 0

	var fieldNames []string
	if err := sqlx.SelectContext(ctx, p.db, &fieldNames, `
		SELECT DISTINCT bf.field_name
		FROM business_object_fields bf
		JOIN business_objects bo ON bo.id = bf.bo_id
		WHERE (bo.bo_key = $1 OR bo.bo_name = $1)
		  AND bo.tenant_id = ANY($2::uuid[])
		ORDER BY bf.field_name
	`, boName, pq.Array(visible)); err != nil {
		return nil, fmt.Errorf("load semantic terms for BO %q: %w", boName, err)
	}
	for _, f := range fieldNames {
		ts.fields[f] = struct{}{}
	}
	cache[boName] = ts
	return ts, nil
}

// loadExistingRuleIndex loads the target tenant's own validation rules.
func (p *ValidationRulePorter) loadExistingRuleIndex(ctx context.Context, tenantID string) (*existingRuleIndex, error) {
	return p.loadRuleIndexForTenant(ctx, tenantID)
}

// loadCoreRuleIndex loads gold-copy rules for shadow detection.
func (p *ValidationRulePorter) loadCoreRuleIndex(ctx context.Context, gold string) (*existingRuleIndex, error) {
	if gold == "" {
		return &existingRuleIndex{byKey: map[string]existingRule{}}, nil
	}
	return p.loadRuleIndexForTenant(ctx, gold)
}

func (p *ValidationRulePorter) loadRuleIndexForTenant(ctx context.Context, tenantID string) (*existingRuleIndex, error) {
	idx := &existingRuleIndex{byKey: map[string]existingRule{}}
	if tenantID == "" {
		return idx, nil
	}
	var rows []struct {
		NodeName   string          `db:"node_name"`
		Properties json.RawMessage `db:"properties"`
		Config     json.RawMessage `db:"config"`
	}
	if err := p.db.SelectContext(ctx, &rows, `
		SELECT n.node_name, n.properties, n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.tenant_id = $1::uuid
		  AND n.is_active
	`, tenantID); err != nil {
		return nil, err
	}
	for _, row := range rows {
		var props models.ValidationRuleProperties
		_ = json.Unmarshal(row.Properties, &props)
		key := props.RuleKey
		if key == "" {
			key = row.NodeName
		}
		var cfg models.ValidationRuleConfig
		_ = json.Unmarshal(row.Config, &cfg)
		idx.byKey[key] = existingRule{
			RuleKey:      key,
			Name:         row.NodeName,
			BOName:       props.BOName,
			Domain:       props.Domain,
			Severity:     props.Severity,
			Timing:       props.Timing,
			Category:     props.Category,
			Governance:   props.GovernanceStatus,
			CanonicalAST: canonicalForDiff(cfg.RuleAST),
		}
	}
	return idx, nil
}

func (idx *existingRuleIndex) shadows(ruleKey, name, boName string) bool {
	if er, ok := idx.byKey[ruleKey]; ok && er.BOName == boName {
		return true
	}
	for _, er := range idx.byKey {
		if er.BOName == boName && er.Name == name {
			return true
		}
	}
	return false
}

func sortDiffSummary(s []models.RuleDiffSummary) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].Action != s[j].Action {
			return s[i].Action < s[j].Action
		}
		return s[i].RuleKey < s[j].RuleKey
	})
}
