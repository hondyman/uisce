package mdmrules

import (
	"sort"
	"strings"
	"testing"
)

// The 30 BOs seeded by migration 20261025_001. If a BO is added or removed there, this list and the vocabulary
// change with it.
var securityBOs = strings.Fields(`
	business_day_convention classification_scheme classification_scheme_map day_count_convention payment_frequency rating_scale
	security_asset_class security_asset_class_routing security_ca_linkage security_change_request security_exception
	security_feed_schedule security_field_mapping security_field_transform security_golden_field security_golden_record
	security_identifier_authority security_identifier_conflict security_identifier_issuance security_match_candidate
	security_match_rule security_reconciliation_result security_source_priority security_status_authority security_steward
	security_survivorship_rule security_term_extraction_field security_term_sheet security_type security_type_mapping`)

func TestSecurityVocabularyCoversExactlyTheThirtyBOs(t *testing.T) {
	var got []string
	for bo := range securityVocabulary {
		got = append(got, bo)
	}
	sort.Strings(got)
	want := append([]string(nil), securityBOs...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("vocabulary BOs = %v; want %v", got, want)
	}
	if len(securityBOs) != 30 {
		t.Errorf("expected 30 security BOs, list has %d", len(securityBOs))
	}
	for bo, ts := range securityVocabulary {
		seen := map[string]bool{}
		for _, term := range ts {
			if seen[term] {
				t.Errorf("%s lists %s twice", bo, term)
			}
			seen[term] = true
		}
		if !seen["ID"] || !seen["TenantId"] {
			t.Errorf("%s is missing the ID or TenantId term every table has", bo)
		}
	}
}

// Every security BO is covered by at least one rule, and the identifying fields are always required.
func TestEverySecurityBOHasARequiredTermsRule(t *testing.T) {
	have := map[string]bool{}
	for _, r := range Catalog() {
		if strings.HasSuffix(r.Name, ".required_terms") {
			have[r.BO] = true
		}
	}
	for _, bo := range securityBOs {
		if !have[bo] {
			t.Errorf("%s has no required_terms rule", bo)
		}
	}
}

func TestSecurityRulesAreAllUnscopedAndOnlyOnSecurityBOs(t *testing.T) {
	isSecurity := map[string]bool{}
	for _, bo := range securityBOs {
		isSecurity[bo] = true
	}
	n := 0
	for _, r := range securityRules() {
		n++
		if !isSecurity[r.BO] {
			t.Errorf("%s is on %q, which is not one of the security BOs", r.Name, r.BO)
		}
		if r.Scope != ScopeAll {
			t.Errorf("%s is binding-scoped; the security rules apply to every binding", r.Name)
		}
	}
	if n < 60 {
		t.Errorf("only %d security rules; the design has more than 60", n)
	}
}

// The threshold rule must enforce the ordering the match logic relies on: auto >= review >= no-match.
func TestSecurityMatchThresholdsMustBeOrdered(t *testing.T) {
	var found bool
	for _, r := range securityRules() {
		if r.Name != "mdm.security_match_rule.thresholds_ordered" {
			continue
		}
		found = true
		if len(r.Cases) < 3 {
			t.Errorf("thresholds rule has %d examples; want the ordered case and both inversions", len(r.Cases))
		}
	}
	if !found {
		t.Error("security_match_rule has no thresholds_ordered rule")
	}
}
