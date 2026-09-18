package swift

import (
	"errors"
	"strings"

	"github.com/lib/pq"
)

// PostgreSQL SQLSTATE codes used by the SWIFT gateway.
const (
	// pgCodeUniqueViolation is SQLSTATE 23505.
	pgCodeUniqueViolation = "23505"
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
// Driver note: cmd/worker and cmd/server blank-import "github.com/lib/pq" with
// sql.Open("postgres",...), so runtime errors surface as *pq.Error.
// The pgconn/pgx-native paths are included as belt-and-suspenders for future
// driver migration or any path that uses pgx stdlib.OpenDB directly.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == pgCodeUniqueViolation
	}
	// Fallback: check error message for any driver that embeds SQLSTATE.
	return strings.Contains(err.Error(), pgCodeUniqueViolation)
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
	// Primary check: lib/pq with exact constraint name.
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == pgCodeUniqueViolation &&
			pqErr.Constraint == ConstraintUETRUniq // empty name → not a match
	}
	// Belt-and-suspenders: constraint name present in error string.
	// Only matches if the exact constraint name is visible (pgx native, future drivers).
	return strings.Contains(err.Error(), ConstraintUETRUniq)
}
