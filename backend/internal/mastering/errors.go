package mastering

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// SetMastering is the mastering message set (seeded in
// db/migrations/20261030_001_message_catalog_mastering.up.sql).
const SetMastering = 9400

func m(nbr int, params ...any) *msgcat.Error { return msgcat.New(SetMastering, nbr, params...) }

func msgNoProfile(entity string) *msgcat.Error {
	return m(1, entity).WithStatus(http.StatusNotFound)
}
func msgNoBinding(table, bo string) *msgcat.Error {
	return m(2, table, bo).WithStatus(http.StatusNotFound)
}
func msgNoLoadRun(id string) *msgcat.Error { return m(3, id).WithStatus(http.StatusNotFound) }
func msgNoSourceKey(table string) *msgcat.Error {
	return m(4, table)
}
func msgNoDataPlane() *msgcat.Error {
	return m(5).WithStatus(http.StatusServiceUnavailable)
}
func msgNotAllowed() *msgcat.Error { return m(6).WithStatus(http.StatusForbidden) }
func msgNoRun(id string) *msgcat.Error {
	return m(7, id).WithStatus(http.StatusNotFound)
}
func msgBadProfile(entity, detail string) *msgcat.Error { return m(8, entity, detail) }
func msgNoGolden(id string) *msgcat.Error {
	return m(9, id).WithStatus(http.StatusNotFound)
}
func msgUnknownSource(cd string) *msgcat.Error { return m(10, cd) }
func msgNeedTable() *msgcat.Error              { return m(11) }
func msgBadResolution(s string) *msgcat.Error  { return m(12, s) }
func msgNoCandidate(id string) *msgcat.Error {
	return m(14, id).WithStatus(http.StatusNotFound)
}
func msgCandidateDecided(id, status string) *msgcat.Error {
	return m(15, id, status).WithStatus(http.StatusConflict)
}
func msgBadPolicy() *msgcat.Error                { return m(16) }
func msgUnknownAttribute(a string) *msgcat.Error { return m(17, a) }
func msgOverrideReason() *msgcat.Error           { return m(18) }
func msgOverrideValue(a string) *msgcat.Error    { return m(19, a) }
func msgOverrideCode(v, a string) *msgcat.Error  { return m(20, v, a) }
func msgNothingToClear(a string) *msgcat.Error   { return m(21, a) }
func msgOverridePending(a string) *msgcat.Error {
	return m(22, a).WithStatus(http.StatusConflict)
}
func msgNoOverride(id string) *msgcat.Error {
	return m(23, id).WithStatus(http.StatusNotFound)
}
func msgOverrideDecided(s string) *msgcat.Error {
	return m(24, s).WithStatus(http.StatusConflict)
}
func msgOwnOverride() *msgcat.Error  { return m(25).WithStatus(http.StatusForbidden) }
func msgAlreadyVoted() *msgcat.Error { return m(26).WithStatus(http.StatusConflict) }
func msgNotProposer() *msgcat.Error  { return m(27).WithStatus(http.StatusForbidden) }
func msgNotAdmin() *msgcat.Error     { return m(28).WithStatus(http.StatusForbidden) }
func msgMergePending() *msgcat.Error { return m(29).WithStatus(http.StatusConflict) }
func msgNoMergeRequest(id string) *msgcat.Error {
	return m(30, id).WithStatus(http.StatusNotFound)
}
func msgOwnMerge() *msgcat.Error             { return m(31).WithStatus(http.StatusForbidden) }
func msgMergeVoted() *msgcat.Error           { return m(32).WithStatus(http.StatusConflict) }
func msgNotMergeRequester() *msgcat.Error    { return m(33).WithStatus(http.StatusForbidden) }
func msgMergeDecided(s string) *msgcat.Error { return m(34, s).WithStatus(http.StatusConflict) }
func msgNoBOBinding(bo, binding string) *msgcat.Error {
	return m(36, bo, binding).WithStatus(http.StatusNotFound)
}
func msgMergeApprovalsUnavailable() *msgcat.Error {
	return m(35).WithStatus(http.StatusServiceUnavailable)
}

func isUniqueViolation(err error) bool {
	var pe *pq.Error
	return errors.As(err, &pe) && pe.Code == "23505"
}

func msgNoException(id string) *msgcat.Error {
	return m(13, id).WithStatus(http.StatusNotFound)
}

func errBadName(n string) error { return fmt.Errorf("%q is not a valid column name", n) }
