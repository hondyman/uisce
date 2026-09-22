// Package dberrors centralizes PostgreSQL SQLSTATE error classification.
// Before this package, "is this a unique-constraint violation" was
// reimplemented ad hoc in at least seven files (internal/swift/pgerrors.go,
// internal/swift/tiles/session_log.go, internal/handlers/admin_tenant_access_handler.go,
// internal/handlers/page_studio_handler.go, internal/services/access_policy_repository.go,
// internal/reports/{repository,folder_repository,schedule_repository}.go), each doing
// its own `var pqErr *pq.Error; errors.As(err, &pqErr)` dance. One place to get this
// right (and to extend to other SQLSTATE classes) beats seven copies quietly drifting.
package dberrors

import (
	"errors"
	"strings"

	"github.com/lib/pq"
)

// PostgreSQL SQLSTATE codes this package classifies. Add more here as
// callers need them, rather than reintroducing an inline check.
const (
	CodeUniqueViolation     = "23505"
	CodeForeignKeyViolation = "23503"
	CodeNotNullViolation    = "23502"
	CodeCheckViolation      = "23514"
)

// AsPgError extracts the *pq.Error from err, if it is (or wraps) one.
// Driver note: cmd/worker and cmd/server blank-import "github.com/lib/pq"
// with sql.Open("postgres",...), so runtime errors surface as *pq.Error.
// A pgx-native path would need its own *pgconn.PgError extraction; none of
// this codebase's current callers use one, so it's not added speculatively.
func AsPgError(err error) (*pq.Error, bool) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr, true
	}
	return nil, false
}

// hasCode reports whether err is a *pq.Error with the given SQLSTATE, with
// a string-match fallback (the code appears literally in err.Error()) for
// any driver/wrapper that doesn't preserve the typed error but does
// include the SQLSTATE in its message.
func hasCode(err error, code string) bool {
	if err == nil {
		return false
	}
	if pqErr, ok := AsPgError(err); ok {
		return string(pqErr.Code) == code
	}
	return strings.Contains(err.Error(), code)
}

// IsUniqueViolation reports whether err is a Postgres 23505 (unique
// constraint violation), on any constraint. Use ConstraintName to check
// which one, or IsUniqueViolationOnConstraint for a specific one.
func IsUniqueViolation(err error) bool { return hasCode(err, CodeUniqueViolation) }

// IsForeignKeyViolation reports whether err is a Postgres 23503 (foreign
// key constraint violation) - e.g. a referenced row was deleted or never
// existed.
func IsForeignKeyViolation(err error) bool { return hasCode(err, CodeForeignKeyViolation) }

// IsNotNullViolation reports whether err is a Postgres 23502 (NOT NULL
// constraint violation).
func IsNotNullViolation(err error) bool { return hasCode(err, CodeNotNullViolation) }

// IsCheckViolation reports whether err is a Postgres 23514 (CHECK
// constraint violation).
func IsCheckViolation(err error) bool { return hasCode(err, CodeCheckViolation) }

// ConstraintName returns the name of the constraint err violated (pq
// populates this on *pq.Error for constraint-violation SQLSTATEs), or ""
// if err isn't a *pq.Error or names no constraint.
func ConstraintName(err error) string {
	if pqErr, ok := AsPgError(err); ok {
		return pqErr.Constraint
	}
	return ""
}

// IsUniqueViolationOnConstraint reports whether err is a 23505 specifically
// on the named constraint - e.g. distinguishing a legitimate
// retry-of-a-duplicate from an unrelated unique violation on the same
// table. An empty constraintName never matches (use IsUniqueViolation for
// "any constraint").
func IsUniqueViolationOnConstraint(err error, constraintName string) bool {
	if constraintName == "" {
		return false
	}
	return IsUniqueViolation(err) && ConstraintName(err) == constraintName
}
