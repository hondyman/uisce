package contracts

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestTypeCompatible(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{"same type", "INTEGER", "INTEGER", true},
		{"int to bigint", "INTEGER", "BIGINT", true},
		{"int to varchar", "INTEGER", "VARCHAR", false},
		{"varchar to text", "VARCHAR", "TEXT", true},
		{"char to varchar", "CHAR", "VARCHAR", true},
		{"date to timestamp", "DATE", "TIMESTAMP", true},
		{"real to double", "REAL", "DOUBLE PRECISION", true},
		{"text to varchar", "TEXT", "VARCHAR", false},
		{"different unrelated", "FLOAT", "JSON", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := typeCompatible(tt.old, tt.new); got != tt.expected {
				t.Errorf("typeCompatible(%q, %q) = %v, want %v", tt.old, tt.new, got, tt.expected)
			}
		})
	}
}

func TestSeverityToExitCode(t *testing.T) {
	if got := SeverityToExitCode(SeverityCritical); got != 1 {
		t.Errorf("SeverityToExitCode(CRITICAL) = %d, want 1", got)
	}
	if got := SeverityToExitCode(SeveritySafe); got != 0 {
		t.Errorf("SeverityToExitCode(SAFE) = %d, want 0", got)
	}
}

func TestContractValidationRequest_Validate_NilRequest(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	_, err := g.Validate(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}

func TestContractValidationRequest_Validate_EmptyTenant(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	_, err := g.Validate(context.Background(), &ContractValidationRequest{})
	if err == nil {
		t.Error("expected error for empty tenant_id")
	}
}

func TestContractValidationRequest_Validate_EmptyDiffs(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	resp, err := g.Validate(context.Background(), &ContractValidationRequest{
		TenantID:      "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.AllSafe {
		t.Error("expected AllSafe=true for empty diffs")
	}
	if resp.HasCritical {
		t.Error("expected HasCritical=false for empty diffs")
	}
}

func TestContractValidationRequest_Validate_AddingNullableColumn(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	resp, err := g.Validate(context.Background(), &ContractValidationRequest{
		TenantID:      "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{
				TableName:    "users",
				DatasourceID: "ds-1",
				Columns: []ColumnDiff{
					{
						ColumnName: "new_col",
						ChangeKind: ColumnAdded,
						NewNull:    boolPtr(true),
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.AllSafe {
		t.Error("expected AllSafe=true for nullable column with no default")
	}
}

func TestContractValidationRequest_Validate_AddingNonNullableWithoutDefault(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	resp, err := g.Validate(context.Background(), &ContractValidationRequest{
		TenantID:      "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{
				TableName:    "orders",
				DatasourceID: "ds-1",
				Columns: []ColumnDiff{
					{
						ColumnName: "required_field",
						ChangeKind: ColumnAdded,
						NewNull:    boolPtr(false),
						NewDefault: "",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AllSafe {
		t.Error("expected AllSafe=false for NOT NULL column without default")
	}
	if !resp.HasCritical {
		t.Error("expected HasCritical=true")
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	result := resp.Results[0]
	if result.SafeToApply {
		t.Error("expected SafeToApply=false")
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation")
	}
	if result.Violations[0].Type != ViolationRequiredColumnAdded {
		t.Errorf("expected violation type REQUIRED_COLUMN_ADDED, got %s", result.Violations[0].Type)
	}
}

func TestContractValidationRequest_Validate_TypeChange_Incompatible(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	resp, err := g.Validate(context.Background(), &ContractValidationRequest{
		TenantID:      "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{
				TableName:    "prices",
				DatasourceID: "ds-1",
				Columns: []ColumnDiff{
					{
						ColumnName: "price",
						ChangeKind: ColumnTypeChanged,
						OldType:    "NUMERIC",
						NewType:   "VARCHAR",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AllSafe {
		t.Error("expected critical for numeric->varchar change")
	}
}

func TestContractValidationRequest_Validate_TypeChange_Compatible(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	resp, err := g.Validate(context.Background(), &ContractValidationRequest{
		TenantID:      "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{
				TableName:    "prices",
				DatasourceID: "ds-1",
				Columns: []ColumnDiff{
					{
						ColumnName: "price",
						ChangeKind: ColumnTypeChanged,
						OldType:    "INTEGER",
						NewType:    "BIGINT",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.AllSafe {
		t.Error("expected SAFE for integer->bigint widening")
	}
}

func TestContractValidationRequest_Validate_ColumnRename(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	resp, err := g.Validate(context.Background(), &ContractValidationRequest{
		TenantID:      "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{
				TableName:    "customers",
				DatasourceID: "ds-1",
				Columns: []ColumnDiff{
					{
						ColumnName: "old_name",
						ChangeKind: ColumnRenamed,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AllSafe {
		t.Error("expected critical for column rename")
	}
	if !resp.HasCritical {
		t.Error("expected HasCritical=true for column rename")
	}
}

func TestContractValidationRequest_Validate_NonNullToNull(t *testing.T) {
	g := &Gatekeeper{db: nil, mcService: nil}
	resp, err := g.Validate(context.Background(), &ContractValidationRequest{
		TenantID:      "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{
				TableName:    "accounts",
				DatasourceID: "ds-1",
				Columns: []ColumnDiff{
					{
						ColumnName: "balance",
						ChangeKind: ColumnNullabilityChanged,
						OldNull:    boolPtr(false),
						NewNull:    boolPtr(true),
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AllSafe {
		t.Error("expected critical for NON-NULL to NULL change")
	}
}

func TestContractValidationRequest_Validate_ComputeContractID(t *testing.T) {
	g := &Gatekeeper{db: nil}
	req1 := &ContractValidationRequest{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{TableName: "orders"},
		},
	}
	req2 := &ContractValidationRequest{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{TableName: "orders"},
		},
	}
	req3 := &ContractValidationRequest{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		ProposedDiffs: []TableDiff{
			{TableName: "customers"},
		},
	}

	id1 := g.computeContractID(req1)
	id2 := g.computeContractID(req2)
	id3 := g.computeContractID(req3)

	if id1 != id2 {
		t.Error("same request should produce same contract ID")
	}
	if id1 == id3 {
		t.Error("different tables should produce different contract ID")
	}
}

func TestContractValidationResponse_JSON(t *testing.T) {
	resp := &ContractValidationResponse{
		RequestID:   "req-1",
		TenantID:    "tenant-1",
		AllSafe:     false,
		HasCritical: true,
		Results: []ValidationResult{
			{
				TableName:    "users",
				Severity:    SeverityCritical,
				SafeToApply: false,
				Violations: []Violation{
					{
						Type:        ViolationRequiredFieldDropped,
						Severity:    SeverityCritical,
						Column:      "id",
						Description: "Required field dropped",
					},
				},
			},
		},
		ViolationsCount: 1,
		SafeCount:      0,
		EvaluatedAt:    time.Now(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var unmarshaled ContractValidationResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if unmarshaled.AllSafe != resp.AllSafe {
		t.Errorf("AllSafe = %v, want %v", unmarshaled.AllSafe, resp.AllSafe)
	}
	if unmarshaled.HasCritical != resp.HasCritical {
		t.Errorf("HasCritical = %v, want %v", unmarshaled.HasCritical, resp.HasCritical)
	}
	if len(unmarshaled.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(unmarshaled.Results))
	}
	if unmarshaled.Results[0].Violations[0].Type != ViolationRequiredFieldDropped {
		t.Errorf("expected violation type REQUIRED_FIELD_DROPPED, got %s", unmarshaled.Results[0].Violations[0].Type)
	}
}

func TestViolationTypes(t *testing.T) {
	if ViolationRequiredFieldDropped != "REQUIRED_FIELD_DROPPED" {
		t.Errorf("unexpected value: %s", ViolationRequiredFieldDropped)
	}
	if ViolationBusinessKeyAltered != "BUSINESS_KEY_ALTERED" {
		t.Errorf("unexpected value: %s", ViolationBusinessKeyAltered)
	}
	if ViolationTypeIncompatible != "TYPE_INCOMPATIBLE" {
		t.Errorf("unexpected value: %s", ViolationTypeIncompatible)
	}
}

func TestContractStatuses(t *testing.T) {
	if ContractStatusPending != "PENDING_REVIEW" {
		t.Errorf("unexpected value: %s", ContractStatusPending)
	}
	if ContractStatusApproved != "APPROVED" {
		t.Errorf("unexpected value: %s", ContractStatusApproved)
	}
	if ContractStatusBlocked != "BLOCKED" {
		t.Errorf("unexpected value: %s", ContractStatusBlocked)
	}
	if ContractStatusTicketed != "TICKET_OPENED" {
		t.Errorf("unexpected value: %s", ContractStatusTicketed)
	}
}

func boolPtr(b bool) *bool { return &b }
