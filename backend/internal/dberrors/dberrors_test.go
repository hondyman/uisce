package dberrors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func pqErr(code, constraint string) error {
	return &pq.Error{Code: pq.ErrorCode(code), Constraint: constraint}
}

func TestIsUniqueViolation(t *testing.T) {
	if IsUniqueViolation(nil) {
		t.Error("nil should not be a unique violation")
	}
	if !IsUniqueViolation(pqErr(CodeUniqueViolation, "some_constraint")) {
		t.Error("23505 should be a unique violation")
	}
	if IsUniqueViolation(pqErr(CodeForeignKeyViolation, "some_fk")) {
		t.Error("23503 should not be a unique violation")
	}
	// Wrapped error still classifies correctly via errors.As.
	wrapped := fmt.Errorf("insert failed: %w", pqErr(CodeUniqueViolation, "uq_thing"))
	if !IsUniqueViolation(wrapped) {
		t.Error("wrapped 23505 should still be detected")
	}
}

func TestIsForeignKeyNotNullCheckViolations(t *testing.T) {
	if !IsForeignKeyViolation(pqErr(CodeForeignKeyViolation, "fk_x")) {
		t.Error("23503 should be a foreign key violation")
	}
	if !IsNotNullViolation(pqErr(CodeNotNullViolation, "")) {
		t.Error("23502 should be a not-null violation")
	}
	if !IsCheckViolation(pqErr(CodeCheckViolation, "chk_x")) {
		t.Error("23514 should be a check violation")
	}
}

func TestConstraintNameAndSpecificMatch(t *testing.T) {
	err := pqErr(CodeUniqueViolation, "uq_bob_tenant_bo_backend")
	if got := ConstraintName(err); got != "uq_bob_tenant_bo_backend" {
		t.Errorf("ConstraintName = %q, want uq_bob_tenant_bo_backend", got)
	}
	if !IsUniqueViolationOnConstraint(err, "uq_bob_tenant_bo_backend") {
		t.Error("should match its own constraint name")
	}
	if IsUniqueViolationOnConstraint(err, "some_other_constraint") {
		t.Error("should not match a different constraint name")
	}
	if IsUniqueViolationOnConstraint(err, "") {
		t.Error("empty constraint name should never match")
	}
}

func TestHasCodeStringFallback(t *testing.T) {
	// A non-*pq.Error that still embeds the SQLSTATE in its message (e.g.
	// a different driver, or an error string built from a *pq.Error
	// upstream) should still be classified via the string fallback.
	err := errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)")
	if !IsUniqueViolation(err) {
		t.Error("should detect 23505 via string fallback")
	}
}
