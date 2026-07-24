package analytics

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestResolveEffectiveFields_DeltaMerge(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewBOGraphService(sqlx.NewDb(db, "postgres"))

	tenantID := uuid.New()
	parentBOID := uuid.New()
	childBOID := uuid.New()
	termID := uuid.New()
	goldFieldID := uuid.New()
	localFieldID := uuid.New()

	// 1. Expect tenant BO lookup: child has a parent.
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, parent_bo_id FROM public.business_objects WHERE id = $1 AND tenant_id = $2;`,
	)).WithArgs(childBOID, tenantID).WillReturnRows(
		sqlmock.NewRows([]string{"id", "parent_bo_id"}).AddRow(childBOID, parentBOID),
	)

	// 2. Expect Gold Copy fields for the parent BO.
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT
			id,
			business_object_id,
			COALESCE(field_name, name) as field_name,
			COALESCE(semantic_term_id, '00000000-0000-0000-0000-000000000000')::uuid as semantic_term_id,
			COALESCE(field_role, type, 'DIMENSION') as field_role,
			COALESCE(binding_requirement, CASE WHEN is_required THEN 'REQUIRED' ELSE 'OPTIONAL' END) as binding_requirement,
			COALESCE(binding_status, 'RESOLVED') as binding_status,
			parent_field_id,
			eligibility_source,
			is_inherited_override
		FROM public.bo_fields
		WHERE business_object_id = $1 AND tenant_id = $2;`,
	)).WithArgs(parentBOID, GoldCopyTenantID).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "business_object_id", "field_name", "semantic_term_id",
			"field_role", "binding_requirement", "binding_status",
			"parent_field_id", "eligibility_source", "is_inherited_override",
		}).AddRow(
			goldFieldID, parentBOID, "order_total", termID,
			"MEASURE", "REQUIRED", "RESOLVED",
			nil, "DIRECT", false,
		),
	)

	// 3. Expect local tenant fields for the child BO (override aggregation).
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT
			id,
			business_object_id,
			COALESCE(field_name, name) as field_name,
			COALESCE(semantic_term_id, '00000000-0000-0000-0000-000000000000')::uuid as semantic_term_id,
			COALESCE(field_role, type, 'DIMENSION') as field_role,
			COALESCE(binding_requirement, CASE WHEN is_required THEN 'REQUIRED' ELSE 'OPTIONAL' END) as binding_requirement,
			COALESCE(binding_status, 'RESOLVED') as binding_status,
			parent_field_id,
			eligibility_source,
			is_inherited_override
		FROM public.bo_fields
		WHERE business_object_id = $1 AND tenant_id = $2;`,
	)).WithArgs(childBOID, tenantID).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "business_object_id", "field_name", "semantic_term_id",
			"field_role", "binding_requirement", "binding_status",
			"parent_field_id", "eligibility_source", "is_inherited_override",
		}).AddRow(
			localFieldID, childBOID, "order_total", termID,
			"MEASURE", "REQUIRED", "RESOLVED",
			nil, "DIRECT", false,
		),
	)

	fields, err := svc.ResolveEffectiveFields(ctx, childBOID, tenantID)
	if err != nil {
		t.Fatalf("failed to calculate combined delta configuration hierarchy: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected exactly 1 collapsed field node, got: %d", len(fields))
	}

	if fields[0].EligibilitySource != "OVERRIDE" {
		t.Errorf("expected field merge classification tag to resolve as OVERRIDE, got: %s", fields[0].EligibilitySource)
	}

	if fields[0].FieldRole != "MEASURE" {
		t.Errorf("expected overridden field role to be preserved, got: %s", fields[0].FieldRole)
	}

	if fields[0].ParentFieldID == nil || *fields[0].ParentFieldID != goldFieldID {
		t.Errorf("expected parent_field_id to point to gold copy field %s, got: %v", goldFieldID, fields[0].ParentFieldID)
	}
}

func TestResolveEffectiveFields_DirectOnly(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewBOGraphService(sqlx.NewDb(db, "postgres"))

	tenantID := uuid.New()
	boID := uuid.New()
	termID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, parent_bo_id FROM public.business_objects WHERE id = $1 AND tenant_id = $2;`,
	)).WithArgs(boID, tenantID).WillReturnRows(
		sqlmock.NewRows([]string{"id", "parent_bo_id"}).AddRow(boID, nil),
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT
			id,
			business_object_id,
			COALESCE(field_name, name) as field_name,
			COALESCE(semantic_term_id, '00000000-0000-0000-0000-000000000000')::uuid as semantic_term_id,
			COALESCE(field_role, type, 'DIMENSION') as field_role,
			COALESCE(binding_requirement, CASE WHEN is_required THEN 'REQUIRED' ELSE 'OPTIONAL' END) as binding_requirement,
			COALESCE(binding_status, 'RESOLVED') as binding_status,
			parent_field_id,
			eligibility_source,
			is_inherited_override
		FROM public.bo_fields
		WHERE business_object_id = $1 AND tenant_id = $2;`,
	)).WithArgs(boID, tenantID).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "business_object_id", "field_name", "semantic_term_id",
			"field_role", "binding_requirement", "binding_status",
			"parent_field_id", "eligibility_source", "is_inherited_override",
		}).AddRow(
			uuid.New(), boID, "customer_name", termID,
			"DIMENSION", "REQUIRED", "RESOLVED",
			nil, "DIRECT", false,
		),
	)

	fields, err := svc.ResolveEffectiveFields(ctx, boID, tenantID)
	if err != nil {
		t.Fatalf("ResolveEffectiveFields failed: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}

	if fields[0].EligibilitySource != "DIRECT" {
		t.Errorf("expected DIRECT eligibility source, got %s", fields[0].EligibilitySource)
	}
}
