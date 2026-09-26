package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeUisce is Keycloak's token endpoint and the schedules API in one server.
type fakeUisce struct {
	final        string // status the run ends in
	polls        int32  // status polls before the run is final
	tokens       atomic.Int32
	triggers     atomic.Int32
	unavailable  atomic.Int32 // answer 503 this many times first
	unauthorized bool
	gotKey       string
	gotRegion    string
}

func (f *fakeUisce) handler(t *testing.T) http.Handler {
	var seen atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_secret") != "s3cret" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error_description":"bad client"}`))
			return
		}
		f.tokens.Add(1)
		// Already inside the refresh margin: every call fetches a new token.
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":10}`))
	})
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if f.unavailable.Load() > 0 {
				f.unavailable.Add(-1)
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			if f.unauthorized || r.Header.Get("Authorization") != "Bearer tok" {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"Service accounts can only read and trigger schedules.","error_code":"9200-22"}`))
				return
			}
			f.gotRegion = r.Header.Get("X-Tenant-Region")
			next(w, r)
		}
	}
	mux.HandleFunc("/api/schedules/", auth(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/schedules/":
			_, _ = w.Write([]byte(`{"schedules":[{"id":"s1","name":"Order volume - London close"},{"id":"s2","name":"Dup"},{"id":"s3","name":"dup"}]}`))
		case strings.HasSuffix(r.URL.Path, "/trigger"):
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			f.gotKey = in["idempotency_key"]
			f.triggers.Add(1)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"schedule_id":"s1","status":"queued"}`))
		case strings.Contains(r.URL.Path, "/triggers/"):
			if seen.Add(1) <= f.polls {
				_, _ = w.Write([]byte(`{"schedule_id":"s1","status":"running","run":{"id":"r1"}}`))
				return
			}
			body := map[string]any{"schedule_id": "s1", "status": f.final, "run": map[string]any{
				"id": "r1", "outcome": map[string]any{"summary": "Order Volume by Status: 187 rows"}, "error_code": map[bool]string{true: "1-4"}[f.final == "failed"],
				"skip_reason": map[bool]string{true: "XLON is closed"}[f.final == "skipped"]}}
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	return mux
}

func runCLI(t *testing.T, f *fakeUisce, args ...string) (int, string, string) {
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	env := map[string]string{
		"UISCE_URL": srv.URL, "UISCE_TOKEN_URL": srv.URL + "/token", "UISCE_CLIENT_ID": "tidal-northwind",
		"UISCE_CLIENT_SECRET": "s3cret", "UISCE_REGION": "us-west",
	}
	var out, errb bytes.Buffer
	code := run(context.Background(), args, func(k string) string { return env[k] }, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunWaitsAndExitsWithTheResult(t *testing.T) {
	for final, want := range map[string]int{"succeeded": exitOK, "failed": exitFailed, "skipped": exitSkipped} {
		f := &fakeUisce{final: final, polls: 2}
		code, out, errOut := runCLI(t, f, "run", "--schedule", "order volume - london close", "--key", "TIDAL-4711", "--wait")
		if code != want {
			t.Errorf("%s: exit %d, want %d (stderr %s)", final, code, want, errOut)
		}
		if !strings.Contains(out, "status="+final) || !strings.Contains(out, "run_id=r1") || !strings.Contains(out, "schedule=s1") {
			t.Errorf("%s: output %q", final, out)
		}
		if f.gotKey != "TIDAL-4711" || f.triggers.Load() != 1 || f.gotRegion != "us-west" {
			t.Errorf("%s: key=%q triggers=%d region=%q", final, f.gotKey, f.triggers.Load(), f.gotRegion)
		}
		if f.tokens.Load() < 2 {
			t.Errorf("%s: a short-lived token must be refreshed during the wait (fetched %d)", final, f.tokens.Load())
		}
	}
}

func TestRunWithoutWaitSucceedsOnceAccepted(t *testing.T) {
	code, out, _ := runCLI(t, &fakeUisce{final: "succeeded"}, "run", "--schedule", "s1", "--key", "k1")
	if code != exitOK || !strings.Contains(out, "status=queued") {
		t.Errorf("exit %d, output %q", code, out)
	}
}

func TestTimeout(t *testing.T) {
	code, _, errOut := runCLI(t, &fakeUisce{final: "succeeded", polls: 1 << 20}, "run", "--schedule", "s1", "--key", "k", "--wait", "--timeout", "1ms")
	if code != exitTimeout || !strings.Contains(errOut, "timed out") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
}

func TestRetriesAnUnavailableServer(t *testing.T) {
	f := &fakeUisce{final: "succeeded"}
	f.unavailable.Store(1)
	if code, _, errOut := runCLI(t, f, "run", "--schedule", "s1", "--key", "k"); code != exitOK {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
}

func TestErrors(t *testing.T) {
	if code, _, errOut := runCLI(t, &fakeUisce{unauthorized: true}, "run", "--schedule", "s1", "--key", "k"); code != exitUnauthorized || !strings.Contains(errOut, "9200-22") {
		t.Errorf("forbidden: exit %d, stderr %q", code, errOut)
	}
	if code, _, _ := runCLI(t, &fakeUisce{}, "run", "--schedule", "s1"); code != exitUsage {
		t.Errorf("no key: exit %d", code)
	}
	if code, _, errOut := runCLI(t, &fakeUisce{}, "run", "--schedule", "nope", "--key", "k"); code != exitAPI || !strings.Contains(errOut, `no schedule is called "nope"`) {
		t.Errorf("unknown: exit %d, stderr %q", code, errOut)
	}
	if code, _, errOut := runCLI(t, &fakeUisce{}, "run", "--schedule", "DUP", "--key", "k"); code != exitAPI || !strings.Contains(errOut, "use the id") {
		t.Errorf("ambiguous: exit %d, stderr %q", code, errOut)
	}
	var errb bytes.Buffer
	if code := run(context.Background(), []string{"run", "--schedule", "s1", "--key", "k"}, func(string) string { return "" }, &bytes.Buffer{}, &errb); code != exitUsage || !strings.Contains(errb.String(), "UISCE_URL") {
		t.Errorf("no config: exit %d, stderr %q", code, errb.String())
	}
	if code := run(context.Background(), nil, func(string) string { return "" }, &bytes.Buffer{}, &bytes.Buffer{}); code != exitUsage {
		t.Errorf("no command: exit %d", code)
	}
}

// A wrong client secret is "not authorized", not a generic failure.
func TestBadClientSecret(t *testing.T) {
	srv := httptest.NewServer((&fakeUisce{}).handler(t))
	defer srv.Close()
	env := map[string]string{"UISCE_URL": srv.URL, "UISCE_TOKEN_URL": srv.URL + "/token", "UISCE_CLIENT_ID": "c", "UISCE_CLIENT_SECRET": "wrong"}
	var errb bytes.Buffer
	if code := run(context.Background(), []string{"run", "--schedule", "s1", "--key", "k"}, func(k string) string { return env[k] }, &bytes.Buffer{}, &errb); code != exitUnauthorized {
		t.Errorf("exit %d, stderr %q", code, errb.String())
	}
}

