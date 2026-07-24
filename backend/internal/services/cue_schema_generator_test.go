package services

import (
	"context"
	"regexp"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCueSchemaGenerator_GetSchema(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	generator := NewCueSchemaGenerator(sqlxDB)
	ctx := context.Background()

	tenantID := "tenant-1"
	boID := "bo-1"

	// 1. Expect BO fetch
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, technical_name, display_name, description FROM business_objects WHERE id = $1 AND tenant_id = $2")).
		WithArgs(boID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "technical_name", "display_name", "description"}).
			AddRow(boID, "Employee", "employee", "Employee Record", "Staff members"))

	// 2. Expect Fields fetch
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, technical_name, key, type, is_required, is_core FROM bo_fields WHERE business_object_id = $1 AND subtype_id IS NULL")).
		WithArgs(boID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "technical_name", "key", "type", "is_required", "is_core"}).
			AddRow("f1", "Full Name", "full_name", "full_name", "text", true, true).
			AddRow("f2", "Age", "age", "age", "integer", false, true).
			AddRow("f3", "Salary", "salary", "salary", "currency", false, false)) // Custom field

	// 3. Expect Subtypes fetch
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, technical_name FROM bo_subtypes WHERE business_object_id = $1")).
		WithArgs(boID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "technical_name"}).
			AddRow("st1", "Manager", "manager"))

	// 4. Expect Subtype Fields fetch
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, technical_name, key, type, is_required FROM bo_fields WHERE subtype_id = $1")).
		WithArgs("st1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "technical_name", "key", "type", "is_required"}).
			AddRow("sf1", "Department", "department", "department", "text", true))

	// Execute
	val, err := generator.GetSchema(ctx, tenantID, boID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.NotNil(t, val)

	// Validate the generated CUE
	c := cuecontext.New()

	// Ensure #employee exists
	emp := val.LookupPath(cue.ParsePath("#employee"))
	assert.True(t, emp.Exists(), "Schema should contain #employee definition")

	// Validate by Unifying valid data
	validData := c.CompileString(`{
		full_name: "John Doe"
		age: 30
		custom: {
			salary: 50000
		}
	}`)

	res := emp.Unify(validData)
	assert.NoError(t, res.Validate(), "Valid data should unify with schema")

	// Validate by Unifying invalid data (wrong type + missing field)
	invalidData := c.CompileString(`{
		full_name: 123
	}`)
	resInvalid := emp.Unify(invalidData)
	assert.Error(t, resInvalid.Validate(), "Invalid data should fail validation")

	// Validate subtype
	mgr := val.LookupPath(cue.ParsePath("#employee_manager"))
	assert.True(t, mgr.Exists(), "Schema should contain #employee_manager definition")

	// Check subtype unification
	// Note: since #employee_manager is #employee & { ... }, it inherits expectations.
	// full_name is required.
	validSubtype := c.CompileString(`{
		full_name: "Jane Doe"
		department: "Engineering"
	}`)
	resSubtype := mgr.Unify(validSubtype)
	assert.NoError(t, resSubtype.Validate(), "Subtype data should unify")
}

func TestCueSchemaGenerator_GetSchemaWithLocale(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	generator := NewCueSchemaGenerator(sqlxDB)
	ctx := context.Background()

	tenantID := "tenant-1"
	boID := "bo-1"
	locale := "es"

	// 1. Expect BO fetch
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, technical_name, display_name FROM business_objects WHERE id=$1 AND tenant_id=$2")).
		WithArgs(boID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "technical_name", "display_name"}).
			AddRow(boID, "Employee", "employee", "Employee Record"))

	// 2. Expect Fields fetch
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, technical_name, key, type, is_required, is_core FROM bo_fields WHERE business_object_id=$1 AND subtype_id IS NULL")).
		WithArgs(boID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "technical_name", "key", "type", "is_required", "is_core"}).
			AddRow("f1", "Full Name", "full_name", "full_name", "text", true, true))

	// Execute
	val, err := generator.GetSchemaWithLocale(ctx, tenantID, boID, locale)
	require.NoError(t, err)
	require.NoErrorf(t, val.Err(), "Schema compilation failed")

	// Verify the schema name includes locale
	empLocal := val.LookupPath(cue.ParsePath("#employee_es"))
	assert.True(t, empLocal.Exists(), "Schema #employee_es should exist")

	// Verify @label attribute exists on field
	// Currently CUE Go API for attributes is a bit specific. We can check if `full_name` exists.
	// To check attribute value, we might need to inspect Value.Attributes(cue.ValueAttr).
	fld := empLocal.LookupPath(cue.ParsePath("full_name"))
	assert.True(t, fld.Exists())

	attrs := fld.Attributes(cue.ValueAttr)
	foundLabel := false
	for _, a := range attrs {
		if a.Name() == "label" {
			val, _ := a.String(0)
			// Mock translation adds "Traducido_" prefix
			if val == "Traducido_Full Name" {
				foundLabel = true
			}
		}
	}
	assert.True(t, foundLabel, "Field should have translated @label attribute")
}
