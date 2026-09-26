package schedule

import (
	"net/http"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// SetSchedule is the scheduler's message set (seeded in
// db/migrations/20261029_002_message_catalog_schedule.up.sql).
const SetSchedule = 9200

func m(nbr int, params ...any) *msgcat.Error { return msgcat.New(SetSchedule, nbr, params...) }

func msgNotFound(id string) *msgcat.Error        { return m(1, id).WithStatus(http.StatusNotFound) }
func msgBadCron(cron string) *msgcat.Error       { return m(2, cron) }
func msgTooFrequent() *msgcat.Error              { return m(3) }
func msgUnknownTimeZone(tz string) *msgcat.Error { return m(4, tz) }
func msgCalendarNotFound(cd string) *msgcat.Error {
	return m(5, cd).WithStatus(http.StatusNotFound)
}
func msgRuleNeedsCalendar() *msgcat.Error      { return m(6) }
func msgBadBusinessDay() *msgcat.Error         { return m(7) }
func msgUnknownKind(kind string) *msgcat.Error { return m(8, kind) }
func msgTargetNotFound(kind, ref string) *msgcat.Error {
	return m(9, kind, ref).WithStatus(http.StatusNotFound)
}
func msgEndBeforeStart() *msgcat.Error { return m(10) }
func msgCalendarGap(cd, date string) *msgcat.Error {
	return m(11, cd, date).WithStatus(http.StatusUnprocessableEntity)
}
func msgEngineUnavailable() *msgcat.Error {
	return m(12).WithStatus(http.StatusServiceUnavailable)
}
func msgNoOutput(runID string) *msgcat.Error { return m(13, runID).WithStatus(http.StatusNotFound) }
func msgNameRequired() *msgcat.Error         { return m(14) }
func msgRunNotFound(id string) *msgcat.Error { return m(15, id).WithStatus(http.StatusNotFound) }
func msgBadRule(rule string) *msgcat.Error   { return m(16, rule) }
func msgBadDate(v string) *msgcat.Error      { return m(17, v) }

// MsgTargetNotFound lets runners report a missing target in the scheduler's
// vocabulary ("Report 42 was not found").
func MsgTargetNotFound(kind, ref string) *msgcat.Error { return msgTargetNotFound(kind, ref) }
