package boresolver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateSQLIdentifier_RejectsInjection(t *testing.T) {
	bad := []string{
		"orders; DROP TABLE",
		"orders--",
		"orders/*x*/",
		`"orders"`,
		"orders space",
		"orders' OR '1'='1",
		"",
		// Leading digit still rejected (per-part validation).
		"1orders",
		// Empty middle part still rejected (the dot-group's
		// `[A-Za-z_]` start requires a non-dot character after every
		// dot, so `a..b` cannot match).
		"a..b",
		// Whitespace inside a part still rejected.
		"a. b",
		// Trailing dot still rejected (a part must have at least one
		// character after every dot).
		"a.b.",
		// Leading dot still rejected (the first part must start with
		// `[A-Za-z_]`).
		".a",
	}
	for _, v := range bad {
		if err := ValidateSQLIdentifier("table", v); err == nil {
			t.Errorf("expected reject for %q", v)
		}
	}
	good := []string{
		"orders",
		"hot.orm_order",
		"effective_date",
		"id",
		// a.b.c is now valid: it was previously rejected by the
		// over-restrictive regex (the `?` quantifier capped at 2
		// parts). 3+ part qualified names are the production
		// bitemporal convention (see the regression test below).
		"a.b.c",
	}
	for _, v := range good {
		if err := ValidateSQLIdentifier("table", v); err != nil {
			t.Errorf("expected allow %q: %v", v, err)
		}
	}
}

// TestValidateSQLIdentifier_AcceptsProductionBitemporalNaming is the
// regression test that pins the production naming convention to the
// validator. The 3-part names below are copied verbatim from the
// bitemporal_range_compiler fixtures that broke when the previous
// `?`-quantifier regex capped accepted input at 2 parts
// (`starrocks.t_99e99e99.portfolio_positions_realtime` and
// `iceberg.t_99e99e99.portfolio_positions_archive` are the engine.
// tenant-shorthand. table names used by the PureCold / PureHot /
// SplitAndStitchWithDeduplication fixtures). If anyone ever re-
// tightens the regex to reject 3+ part names, this test fails with a
// message that points at the production shape — not at a 1-off
// refactor noise — so the failure is unmistakable.
//
// The four-part case is included to document that the regex has no
// upper bound on part count: every part is independently identifier-
// shaped, so a hypothetical catalog.schema.table.col is harmless.
// Don't add an arbitrary cap; that would recreate the same over-
// restriction bug one level up.
func TestValidateSQLIdentifier_AcceptsProductionBitemporalNaming(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"bare_identifier", "orders"},
		{"two_part_schema_table", "oms.orm_execution"},
		{"three_part_engine_tenant_table_hot", "starrocks.t_99e99e99.portfolio_positions_realtime"},
		{"three_part_engine_tenant_table_cold", "iceberg.t_99e99e99.portfolio_positions_archive"},
		{"four_part_catalog_schema_table_col", "catalog.oms.orm_execution.id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateSQLIdentifier("table", tc.in); err != nil {
				t.Errorf("expected allow %q (%s): %v", tc.in, tc.name, err)
			}
		})
	}
}

func TestCompileRangeQuery_RejectsInjectionIdentifiers(t *testing.T) {
	c := NewBitemporalRangeCompiler()
	base := BitemporalRangeRequest{
		TenantID:           uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"),
		EffectiveStartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveEndDate:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		WatermarkDate:      time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
		HotTableName:       "hot.ok",
		ColdTableName:      "cold.ok",
	}
	cases := []struct {
		name string
		mut  func(*BitemporalRangeRequest)
	}{
		{"hot_inject", func(r *BitemporalRangeRequest) { r.HotTableName = "orders; DROP TABLE t" }},
		{"cold_comment", func(r *BitemporalRangeRequest) { r.ColdTableName = "cold.t--" }},
		{"col_inject", func(r *BitemporalRangeRequest) { r.SelectedColumns = []string{"id;--"} }},
		{"pk_inject", func(r *BitemporalRangeRequest) { r.BusinessKeyColumns = []string{"id OR 1=1"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mut(&req)
			_, err := c.CompileRangeQuery(context.Background(), req)
			if err == nil || !strings.Contains(err.Error(), "allowlisted") && !strings.Contains(err.Error(), "invalid") {
				t.Fatalf("expected identifier rejection, got %v", err)
			}
		})
	}
}
