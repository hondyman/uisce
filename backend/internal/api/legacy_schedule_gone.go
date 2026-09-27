package api

import (
	"encoding/json"
	"net/http"
)

// writeLegacyScheduleGone closes the pre-core report_schedules HTTP surface.
// Callers must authenticate first where the route previously required auth.
func writeLegacyScheduleGone(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Link", "</api/schedules>; rel=\"successor-version\"")
	w.WriteHeader(http.StatusGone)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "legacy_report_schedules_retired",
		"message": "Report schedules live on POST/GET /api/schedules with target.kind=report. Use Run now or the Schedules console.",
	})
}
