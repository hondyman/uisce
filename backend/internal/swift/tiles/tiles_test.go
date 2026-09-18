package tiles

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDecodeTransform_MT541(t *testing.T) {
	cfg := DecodeConfig{
		SwiftVersion:           "MT",
		ValidateRequiredFields: true,
	}
	tile := NewDecodeTransform(cfg, nil)

	raw := `{1:F01BANKDEF0AXXX0000000000}{2:I541BANKABC0XXXXN}{4:
:20:REF123
:35B:ISIN US0378331005
:36:1000
:19A::SETT//USD1000000,00
:98A::SETT//20260917
-}`

	records := []Record{
		{"raw_bytes": []byte(raw)},
	}

	out, errs, err := tile(context.Background(), records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("unexpected tile errors: %v", errs)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 record, got %d", len(out))
	}

	fields := out[0]["fields"].(map[string]string)
	if fields["20"] != "REF123" {
		t.Errorf("expected tag 20 = REF123, got %s", fields["20"])
	}
	if fields["35B"] != "ISIN US0378331005" {
		t.Errorf("expected tag 35B = ISIN US0378331005, got %s", fields["35B"])
	}
}

func TestFieldMapTransform_GoldCopyInheritance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"field_tag", "semantic_field", "required", "default_value", "transform_fn"}).
		AddRow("20", "transaction_ref", true, nil, nil).
		AddRow("20", "transaction_ref", true, nil, nil).
		AddRow("35B", "isin", true, nil, "parse_isin")

	mock.ExpectQuery("SELECT field_tag").WillReturnRows(rows)

	loader := &DBTagMappingLoader{DB: db}

	cfg := FieldMapConfig{}
	tile := NewFieldMapTransform(cfg, loader)

	records := []Record{
		{
			"msg_type": "MT541",
			"fields": map[string]string{
				"20":  "TENANT_REF",
				"35B": "ISIN US1234567890",
			},
		},
	}

	ctx := WithTenant(context.Background(), TenantContext{TenantID: "tenant-1"})
	out, errs, err := tile(ctx, records)
	if err != nil || len(errs) > 0 {
		t.Fatalf("unexpected errs: %v, %v", err, errs)
	}

	semantic := out[0]["semantic"].(map[string]any)
	if semantic["transaction_ref"] != "TENANT_REF" {
		t.Errorf("expected TENANT_REF")
	}
	if semantic["isin"] != "US1234567890" {
		t.Errorf("expected US1234567890, got %v", semantic["isin"])
	}
}

func TestCalendarValidator_WeekendSettlementDate(t *testing.T) {
	cfg := CalendarConfig{}
	tile := NewCalendarValidator(cfg)

	satDate := time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	records := []Record{
		{
			"semantic": map[string]any{
				"settlement_date": satDate,
			},
		},
	}

	out, errs, err := tile(context.Background(), records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if len(out) != 0 {
		t.Fatalf("expected record to be dropped")
	}
}

func TestSettlementWriter_DeduplicatesOnTransactionRef(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := SettlementWriterConfig{
		SettlementSubtype: "dvp_securities",
		UpsertOnConflict:  false,
	}
	tile := NewSettlementWriterLoader(cfg, db)

	mock.ExpectExec("INSERT INTO cash_flow.settlement").
		WillReturnResult(sqlmock.NewResult(1, 1))

	records := []Record{
		{
			"semantic": map[string]any{
				"account_id":        "acc-1",
				"settlement_amount": "1000",
				"currency":          "USD",
			},
			"transaction_ref": "REF123",
		},
	}

	ctx := WithTenant(context.Background(), TenantContext{TenantID: "tenant-1"})
	out, errs, err := tile(ctx, records)
	if err != nil || len(errs) > 0 {
		t.Fatalf("unexpected errs: %v, %v", err, errs)
	}

	if len(out) != 1 {
		t.Fatalf("expected 1 record")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

type fakeComplianceEvaluator struct {
	passed     bool
	violations []string
}

func (f *fakeComplianceEvaluator) Evaluate(ctx context.Context, tenantID string, ruleSetIDs []string, record map[string]any) (bool, []string, error) {
	return f.passed, f.violations, nil
}

func TestComplianceValidator_HardBlock(t *testing.T) {
	evaluator := &fakeComplianceEvaluator{
		passed:     false,
		violations: []string{"Sanctions check failed"},
	}

	cfg := ComplianceConfig{
		SeverityThreshold: "HARD_BLOCK",
	}
	tile := NewComplianceValidator(cfg, evaluator)

	records := []Record{
		{"some_data": "data"},
	}

	out, errs, err := tile(context.Background(), records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected record to be dropped")
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error")
	}
}
