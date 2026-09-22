package swift

import (
	"strings"

	"github.com/hondyman/uisce/backend/internal/dberrors"
)

// Constraint names on SWIFT tables.
const (
	// ConstraintUETRUniq is the unique index on vend.swift_session_log(tenant_id, uetr).
	// A 23505 on this constraint = SWIFTNet retransmission of an already-processed message.
	ConstraintUETRUniq = "swift_session_log_uetr_tenant_uniq"
)

// IsUniqueViolation returns true if err is a PostgreSQL 23505 unique-constraint
// violation on any constraint. Use IsUETRDuplicate for the UETR-specific check.
//
// Kept as a thin wrapper (rather than switching every SWIFT call site to
// dberrors directly) so this package's public API doesn't churn - the
// classification logic itself now lives in one place, internal/dberrors.
func IsUniqueViolation(err error) bool {
	return dberrors.IsUniqueViolation(err)
}

// IsUETRDuplicate returns true if err is a 23505 specifically on the UETR
// unique index — i.e., a SWIFTNet retransmission that should be discarded.
//
// Requires an exact constraint name match. An empty constraint name is NOT
// treated as a UETR duplicate — it returns false so the caller treats it as a
// non-fatal audit-gap error and passes the record downstream. Rationale:
// discarding inbound SWIFT messages on an ambiguous constraint name is worse
// than double-processing one; the settlement_writer's ON CONFLICT is the
// second dedup line in either case.
func IsUETRDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if dberrors.IsUniqueViolation(err) {
		// Primary check: exact constraint name, when the driver reports one.
		// An empty constraint name intentionally falls through to the
		// string fallback below rather than matching here.
		if name := dberrors.ConstraintName(err); name != "" {
			return name == ConstraintUETRUniq
		}
	}
	// Belt-and-suspenders: constraint name present in error string.
	// Only matches if the exact constraint name is visible (pgx native, future drivers).
	return strings.Contains(err.Error(), ConstraintUETRUniq)
}
