package schedule

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

func catalogNbr(t *testing.T, err error) int {
	t.Helper()
	var me *msgcat.Error
	if !errors.As(err, &me) || me.Set != SetSchedule {
		t.Fatalf("want a scheduler catalog error, got %v", err)
	}
	return me.Nbr
}

// An external schedule has no timetable: cron is ignored, and its calendar
// can only refuse a closed day - never defer the run the caller waits on.
func TestExternalTimingValidation(t *testing.T) {
	ok := []Timing{
		{Mode: ModeExternal, TimeZone: "Europe/London"},
		{Mode: ModeExternal, TimeZone: "Europe/London", Cron: "not a cron"},
		{Mode: ModeExternal, TimeZone: "Europe/London", Calendar: "XLON", CalendarRule: RuleSkip},
	}
	for _, tm := range ok {
		if err := tm.Validate(); err != nil {
			t.Errorf("%+v: %v", tm, err)
		}
	}
	bad := map[int]Timing{
		6:  {Mode: ModeExternal, CalendarRule: RuleSkip},
		21: {Mode: ModeExternal, Calendar: "XLON", CalendarRule: RuleNextBusinessDay},
		16: {Mode: "sometimes"},
		4:  {Mode: ModeExternal, TimeZone: "Mars/Olympus"},
	}
	for want, tm := range bad {
		if got := catalogNbr(t, tm.Validate()); got != want {
			t.Errorf("%+v: 9200-%d, want 9200-%d", tm, got, want)
		}
	}
	// A timetable schedule still needs its cron.
	if got := catalogNbr(t, Timing{Cron: "nope"}.Validate()); got != 2 {
		t.Errorf("timetable without a cron: 9200-%d", got)
	}
}

func TestExternalScheduleHasNoFirings(t *testing.T) {
	f, err := Timing{Mode: ModeExternal, Cron: "0 18 * * 1-5"}.Fires(5, time.Now())
	if err != nil || len(f) != 0 {
		t.Errorf("fires=%v err=%v", f, err)
	}
	if (Timing{Mode: ModeExternal, Cron: "0 18 * * *"}).cronArg() != "" || (Timing{}).mode() != ModeTimetable {
		t.Error("an external schedule stores no cron; the default mode is timetable")
	}
}

// The workflow id is the idempotency guard: the same key always maps to the
// same workflow, different keys and schedules never collide.
func TestTriggerWorkflowID(t *testing.T) {
	a := TriggerWorkflowID("s1", "TIDAL-4711")
	if a != TriggerWorkflowID("s1", "TIDAL-4711") {
		t.Error("same key, same workflow")
	}
	if a == TriggerWorkflowID("s1", "TIDAL-4712") || a == TriggerWorkflowID("s2", "TIDAL-4711") {
		t.Error("different key or schedule must differ")
	}
	if strings.Contains(a, "TIDAL") || len(a) > 100 {
		t.Errorf("the key is hashed into a bounded id: %s", a)
	}
}

func TestFireInputTrigger(t *testing.T) {
	cases := map[string]FireInput{
		"schedule": {},
		"manual":   {Manual: true},
		"external": {External: &ExternalTrigger{IdempotencyKey: "k"}},
	}
	for want, in := range cases {
		if got := in.trigger(); got != want {
			t.Errorf("%+v: %s, want %s", in, got, want)
		}
	}
}

func TestTriggerNeedsAKey(t *testing.T) {
	s := &Service{}
	for _, key := range []string{"", "   ", strings.Repeat("k", maxKey+1)} {
		_, err := s.Trigger(context.Background(), Actor{TenantID: "t"}, "s1", ExternalTrigger{IdempotencyKey: key})
		if got := catalogNbr(t, err); got != 20 {
			t.Errorf("key %q: 9200-%d", key, got)
		}
	}
}

// Service accounts read and trigger; every change is refused before it
// reaches the service.
func TestServiceAccountsCannotChangeSchedules(t *testing.T) {
	machine := func(canTrigger bool) *Handler {
		return &Handler{Catalog: nil, ActorFrom: func(*http.Request) (Actor, error) {
			return Actor{UserID: "service-account-tidal", TenantID: "t", Machine: true, CanTrigger: canTrigger}, nil
		}}
	}
	r := chi.NewRouter()
	machine(true).RegisterRoutes(r)
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/schedules/"},
		{http.MethodPut, "/schedules/s1"},
		{http.MethodDelete, "/schedules/s1"},
		{http.MethodPost, "/schedules/s1/pause"},
		{http.MethodPost, "/schedules/s1/resume"},
		{http.MethodPost, "/schedules/s1/run"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(c.method, c.path, strings.NewReader(`{}`)))
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "9200-22") {
			t.Errorf("%s %s: %d %s", c.method, c.path, w.Code, w.Body.String())
		}
	}

	// Without schedule_trigger a service account can't trigger either.
	r = chi.NewRouter()
	machine(false).RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/schedules/s1/trigger", strings.NewReader(`{"idempotency_key":"k"}`)))
	if w.Code != http.StatusForbidden {
		t.Errorf("trigger without schedule_trigger: %d %s", w.Code, w.Body.String())
	}
}
