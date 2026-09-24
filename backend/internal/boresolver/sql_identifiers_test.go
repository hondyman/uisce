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
		"a.b.c", // too many dots
	}
	for _, v := range bad {
		if err := ValidateSQLIdentifier("table", v); err == nil {
			t.Errorf("expected reject for %q", v)
		}
	}
	good := []string{"orders", "hot.orm_order", "effective_date", "id"}
	for _, v := range good {
		if err := ValidateSQLIdentifier("table", v); err != nil {
			t.Errorf("expected allow %q: %v", v, err)
		}
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
