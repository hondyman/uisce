package mastering

import (
	"fmt"
	"net/http"

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
func msgNoException(id string) *msgcat.Error {
	return m(13, id).WithStatus(http.StatusNotFound)
}

func errBadName(n string) error { return fmt.Errorf("%q is not a valid column name", n) }
