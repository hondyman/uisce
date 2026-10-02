package stagingbind

import (
	"net/http"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// SetStagingBind is the staging bindings message set (seeded in
// db/migrations/20261029_006_message_catalog_staging_bindings.up.sql).
const SetStagingBind = 9300

func m(nbr int, params ...any) *msgcat.Error { return msgcat.New(SetStagingBind, nbr, params...) }

func msgNoBO(key string) *msgcat.Error { return m(1, key).WithStatus(http.StatusNotFound) }
func msgNotStaging(t string) *msgcat.Error {
	return m(2, t)
}
func msgNoField(bo, f string) *msgcat.Error     { return m(3, bo, f) }
func msgNoColumn(table, c string) *msgcat.Error { return m(4, table, c) }
func msgNoFields() *msgcat.Error                { return m(5) }
func msgOwnChange() *msgcat.Error               { return m(6).WithStatus(http.StatusForbidden) }
func msgChangeNotFound(id string) *msgcat.Error { return m(7, id).WithStatus(http.StatusNotFound) }
func msgDecided() *msgcat.Error                 { return m(8).WithStatus(http.StatusConflict) }
func msgNotProposer() *msgcat.Error             { return m(9).WithStatus(http.StatusForbidden) }
func msgNotAllowed() *msgcat.Error              { return m(10).WithStatus(http.StatusForbidden) }
func msgStale() *msgcat.Error                   { return m(11).WithStatus(http.StatusConflict) }
func msgNoBinding(table, bo string) *msgcat.Error {
	return m(12, table, bo).WithStatus(http.StatusNotFound)
}
func msgNoStagingDB() *msgcat.Error {
	return m(13).WithStatus(http.StatusServiceUnavailable)
}
func msgNoTenant() *msgcat.Error { return m(14) }

// MsgNoBinding lets callers (the pipeline check) report a missing binding
// in the catalog's words.
func MsgNoBinding(table, bo string) *msgcat.Error { return msgNoBinding(table, bo) }
