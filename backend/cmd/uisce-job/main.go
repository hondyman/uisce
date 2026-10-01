// Command uisce-job lets an enterprise scheduler (Tidal, Control-M, AutoSys,
// Stonebranch, ...) run Uisce schedules from an agent: it triggers an
// externally triggered schedule and, with --wait, waits for the run and exits
// with a code the scheduler can branch on.
//
//	uisce-job run --schedule "Order volume - London close" --key "$TIDAL_JOB_RUN_ID" --wait
//	uisce-job status --schedule <id|name> --key <key> [--wait]
//
// Exit codes: 0 succeeded, 1 failed, 2 skipped (e.g. calendar closed),
// 3 timed out waiting, 4 not authorized, 5 bad usage or configuration,
// 6 other API error.
//
// It authenticates as a Keycloak service account (client credentials):
//
//	UISCE_URL            e.g. https://uisce.example.com
//	UISCE_TOKEN_URL      e.g. https://keycloak.example.com/realms/uisce/protocol/openid-connect/token
//	UISCE_CLIENT_ID      the tenant's scheduler client
//	UISCE_CLIENT_SECRET  or UISCE_CLIENT_SECRET_FILE (preferred: a file only the agent can read)
//	UISCE_REGION         the tenant's region (X-Tenant-Region)
//
// The trigger is idempotent on --key: give it the scheduler's own run id so a
// retried job never starts a second run.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Exit codes.
const (
	exitOK = iota
	exitFailed
	exitSkipped
	exitTimeout
	exitUnauthorized
	exitUsage
	exitAPI
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type config struct {
	baseURL, tokenURL, clientID, clientSecret, region, datasource string
	http                                                          *http.Client
}

func run(ctx context.Context, args []string, env func(string) string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "run" && args[0] != "status") {
		fmt.Fprintln(stderr, "usage: uisce-job run|status --schedule <id|name> --key <idempotency key> [--wait] [--timeout 12h]")
		return exitUsage
	}
	cmd := args[0]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	schedule := fs.String("schedule", "", "schedule id or exact name")
	key := fs.String("key", env("UISCE_IDEMPOTENCY_KEY"), "idempotency key (use the scheduler's run id)")
	system := fs.String("system", orDefault(env("UISCE_EXTERNAL_SYSTEM"), "enterprise-scheduler"), "calling scheduler, e.g. tidal")
	ref := fs.String("ref", env("UISCE_EXTERNAL_REF"), "the calling job's reference, for the run history")
	wait := fs.Bool("wait", false, "wait for the run to finish and exit with its result")
	timeout := fs.Duration("timeout", 12*time.Hour, "how long --wait waits")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	cfg, err := configFrom(env)
	if err != nil {
		fmt.Fprintln(stderr, "uisce-job:", err)
		return exitUsage
	}
	if *schedule == "" || strings.TrimSpace(*key) == "" {
		fmt.Fprintln(stderr, "uisce-job: --schedule and --key are required (use the scheduler's run id as the key)")
		return exitUsage
	}
	c := &client{cfg: cfg}
	id, err := c.resolveSchedule(ctx, *schedule)
	if err != nil {
		return c.fail(stderr, err)
	}
	var st *triggerStatus
	if cmd == "run" {
		st, err = c.trigger(ctx, id, *key, *system, *ref)
	} else {
		st, err = c.status(ctx, id, *key, 0)
	}
	if err != nil {
		return c.fail(stderr, err)
	}
	if *wait {
		deadline := time.Now().Add(*timeout)
		for !st.done() {
			left := time.Until(deadline)
			if left <= 0 {
				report(stdout, id, *key, st)
				fmt.Fprintf(stderr, "uisce-job: timed out after %s waiting for the run\n", *timeout)
				return exitTimeout
			}
			poll := 60 * time.Second
			if left < poll {
				poll = left
			}
			if st, err = c.status(ctx, id, *key, poll); err != nil {
				return c.fail(stderr, err)
			}
		}
	}
	report(stdout, id, *key, st)
	return exitFor(st, *wait)
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func configFrom(env func(string) string) (config, error) {
	c := config{
		baseURL: strings.TrimRight(env("UISCE_URL"), "/"), tokenURL: env("UISCE_TOKEN_URL"),
		clientID: env("UISCE_CLIENT_ID"), clientSecret: env("UISCE_CLIENT_SECRET"),
		region: env("UISCE_REGION"), datasource: env("UISCE_DATASOURCE_ID"),
		http: &http.Client{Timeout: 3 * time.Minute},
	}
	if f := env("UISCE_CLIENT_SECRET_FILE"); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return c, fmt.Errorf("reading UISCE_CLIENT_SECRET_FILE: %w", err)
		}
		c.clientSecret = strings.TrimSpace(string(b))
	}
	var missing []string
	for name, v := range map[string]string{"UISCE_URL": c.baseURL, "UISCE_TOKEN_URL": c.tokenURL,
		"UISCE_CLIENT_ID": c.clientID, "UISCE_CLIENT_SECRET(_FILE)": c.clientSecret} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("set %s", strings.Join(sorted(missing), ", "))
	}
	return c, nil
}

func sorted(s []string) []string {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	return s
}

type triggerStatus struct {
	ScheduleID string `json:"schedule_id"`
	Status     string `json:"status"`
	Run        *struct {
		ID          string          `json:"id"`
		SkipReason  string          `json:"skip_reason"`
		ErrorCode   string          `json:"error_code"`
		ErrorParams []string        `json:"error_params"`
		Outcome     json.RawMessage `json:"outcome"`
	} `json:"run"`
}

func (s *triggerStatus) done() bool {
	return s.Status == "succeeded" || s.Status == "failed" || s.Status == "skipped"
}

func report(w io.Writer, id, key string, st *triggerStatus) {
	fmt.Fprintf(w, "schedule=%s key=%s status=%s", id, key, st.Status)
	if st.Run != nil {
		fmt.Fprintf(w, " run_id=%s", st.Run.ID)
		var o struct {
			Summary string `json:"summary"`
		}
		if json.Unmarshal(st.Run.Outcome, &o) == nil && o.Summary != "" {
			fmt.Fprintf(w, " summary=%q", o.Summary)
		}
		if st.Run.SkipReason != "" {
			fmt.Fprintf(w, " reason=%q", st.Run.SkipReason)
		}
		if st.Run.ErrorCode != "" {
			fmt.Fprintf(w, " error_code=%s", st.Run.ErrorCode)
		}
	}
	fmt.Fprintln(w)
}

// exitFor: without --wait a queued or running trigger is a success (it was
// accepted); with --wait only a succeeded run is.
func exitFor(st *triggerStatus, waited bool) int {
	switch st.Status {
	case "succeeded":
		return exitOK
	case "failed":
		return exitFailed
	case "skipped":
		return exitSkipped
	}
	if waited {
		return exitTimeout
	}
	return exitOK
}

// --- API client -------------------------------------------------------------

type apiError struct {
	status  int
	message string
	code    string
}

func (e *apiError) Error() string {
	if e.code != "" {
		return fmt.Sprintf("%s (%s, HTTP %d)", e.message, e.code, e.status)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.message, e.status)
}

type client struct {
	cfg     config
	mu      sync.Mutex
	token   string
	expires time.Time
}

func (c *client) fail(w io.Writer, err error) int {
	fmt.Fprintln(w, "uisce-job:", err)
	var ae *apiError
	if errors.As(err, &ae) && (ae.status == http.StatusUnauthorized || ae.status == http.StatusForbidden) {
		return exitUnauthorized
	}
	return exitAPI
}

// accessToken is a client-credentials token, fetched again shortly before
// it expires (service account tokens are short-lived; a --wait can be long).
func (c *client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.expires) > 30*time.Second {
		return c.token, nil
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {c.cfg.clientID}, "client_secret": {c.cfg.clientSecret}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.cfg.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("getting a token: %w", err)
	}
	defer res.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error_description"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)
	if res.StatusCode != http.StatusOK || body.AccessToken == "" {
		return "", &apiError{status: http.StatusUnauthorized, message: "the token endpoint refused the client credentials: " + body.Error}
	}
	c.token, c.expires = body.AccessToken, time.Now().Add(time.Duration(body.ExpiresIn)*time.Second)
	return c.token, nil
}

// do calls the API, retrying network errors and 502/503/504 a few times:
// every call is a read or an idempotent trigger, so a retry is safe.
func (c *client) do(ctx context.Context, method, path string, in, out any) error {
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt*attempt) * time.Second):
			}
		}
		err := c.once(ctx, method, path, in, out)
		var ae *apiError
		if err == nil || (errors.As(err, &ae) && ae.status != http.StatusBadGateway &&
			ae.status != http.StatusServiceUnavailable && ae.status != http.StatusGatewayTimeout) {
			return err
		}
		last = err
	}
	return last
}

func (c *client) once(ctx context.Context, method, path string, in, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.region != "" {
		req.Header.Set("X-Tenant-Region", c.cfg.region)
	}
	if c.cfg.datasource != "" {
		req.Header.Set("X-Tenant-Datasource-ID", c.cfg.datasource)
	}
	res, err := c.cfg.http.Do(req)
	if err != nil {
		return &apiError{status: http.StatusServiceUnavailable, message: err.Error()}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		var e struct {
			Error     string `json:"error"`
			ErrorCode string `json:"error_code"`
		}
		_ = json.Unmarshal(raw, &e)
		if e.Error == "" {
			e.Error = strings.TrimSpace(string(raw))
		}
		return &apiError{status: res.StatusCode, message: e.Error, code: e.ErrorCode}
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// resolveSchedule accepts an id or an exact (case-insensitive) name.
func (c *client) resolveSchedule(ctx context.Context, s string) (string, error) {
	var list struct {
		Schedules []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"schedules"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/schedules/", nil, &list); err != nil {
		return "", err
	}
	var hits []string
	for _, x := range list.Schedules {
		if x.ID == s {
			return x.ID, nil
		}
		if strings.EqualFold(strings.TrimSpace(x.Name), strings.TrimSpace(s)) {
			hits = append(hits, x.ID)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return "", &apiError{status: http.StatusNotFound, message: fmt.Sprintf("no schedule is called %q", s)}
	}
	return "", &apiError{status: http.StatusConflict, message: fmt.Sprintf("%d schedules are called %q; use the id", len(hits), s)}
}

func (c *client) trigger(ctx context.Context, id, key, system, ref string) (*triggerStatus, error) {
	var st triggerStatus
	err := c.do(ctx, http.MethodPost, "/api/schedules/"+url.PathEscape(id)+"/trigger",
		map[string]string{"idempotency_key": key, "system": system, "ref": ref}, &st)
	return &st, err
}

func (c *client) status(ctx context.Context, id, key string, wait time.Duration) (*triggerStatus, error) {
	var st triggerStatus
	path := "/api/schedules/" + url.PathEscape(id) + "/triggers/" + url.PathEscape(key)
	if wait > 0 {
		path += "?wait=" + wait.Round(time.Second).String()
	}
	err := c.do(ctx, http.MethodGet, path, nil, &st)
	return &st, err
}
