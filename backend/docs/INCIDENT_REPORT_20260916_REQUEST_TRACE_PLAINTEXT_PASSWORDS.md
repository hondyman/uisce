# INCIDENT REPORT — 2026-09-16

## `request_trace_middleware` (dev-mode `r.Use` in `api.go`) Logs Plaintext Passwords to stderr

## Executive Summary

A dev-mode middleware registered globally on the chi router via `r.Use(...)`
in `backend/internal/api/api.go` logs the full request body of every
incoming request to `os.Stderr` (which the running server redirects to
`/tmp/uisce-server.log`). For `POST /api/auth/login`, this means every
login attempt's password is written to the log file in plaintext — once
per attempt.

**Status: Open.** Documented here as a standing follow-up. Currently
limited to dev-only Tailscale exposure; becomes a production incident the
moment the same middleware ships in any environment with broader
reach.

## Discovery Context

Found during post-fix verification of
`backend/docs/INCIDENT_REPORT_20260916_LOGIN_BCRYPT.md`. While reading
`/tmp/uisce-server.log` to confirm the `crypto/bcrypt: hashedSecret too
short` rejection message, the log line also contained the full request
body, including the `password` field, in plaintext. The login incident
report's evidence section already redacted the password when quoting
the same line — the instinct was correct; it just never got promoted
from "redact when quoting" to "the log itself shouldn't contain it."

## Symptom

```
[REQ] POST /api/auth/login Headers:Content-Type=application/json,Authorization=,Origin=,X-Tenant-ID=,X-Tenant-Datasource-ID= Body:{"email":"testuser2@example.com","password":"<redacted>"}
```

The `Body:{"..."}` segment after `Body:` is the raw, unredacted JSON
request body. The password sits at the same indentation as the email
and is plainly visible.

Direct count from `/tmp/uisce-server.log` (2026-09-16 reading):

| Pattern | Lines |
|---|---|
| Lines containing the literal `password` field | 11 |
| Lines containing the prior-session password (Keycloak admin credential, redacted) | 2 |
| Lines containing the new dev password (this session) | 6 |

This is one session's worth of traffic on a dev-only deployment. The
pattern accumulates indefinitely — `/tmp/uisce-server.log` is not
logrotated.

## Source Location

`backend/internal/api/api.go:706-739`:

```go
// Development middleware: log every incoming request (method, path, headers, body)
// This is intentionally verbose and should only be enabled during local debugging.
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        // Read body (if any) for logging and restore it for downstream handlers
        var bodyBytes []byte
        if req.Body != nil {
            bodyBytes, _ = io.ReadAll(req.Body)
            req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
        }
        // Collect a subset of headers for brevity
        headersToLog := []string{"Content-Type", "Authorization", "Origin", "X-Tenant-ID", "X-Tenant-Datasource-ID"}
        headerParts := []string{}
        for _, h := range headersToLog {
            headerParts = append(headerParts, fmt.Sprintf("%s=%s", h, req.Header.Get(h)))
        }
        fmt.Fprintf(os.Stderr, "[REQ] %s %s Headers:%s Body:%s\n", req.Method, req.URL.Path, strings.Join(headerParts, ","), string(bodyBytes))
        ...
        next.ServeHTTP(w, req)
    })
})
```

The comment at line 706 explicitly says "should only be enabled during
local debugging" — but the middleware is registered unconditionally
via `r.Use` at the top of `SetupRouter`, with no environment flag. It
runs in every environment the binary is built for.

## Why This Matters

This is the same finding class as the prior session-password-in-AGENTS.md
incident (AGENTS.md line 507/520, 2026-09-13): plaintext credentials
in an artifact that persists beyond the session that produced them.
Two differences make this one more dangerous:

1. **No size bound.** `/tmp/uisce-server.log` is a regular file; the
   body logger appends one line per request. Password accumulation
   grows unbounded until the file is rotated (or the disk fills).
2. **No rotation policy.** Unlike journald or syslog, the file has no
   automatic rotation. A long-running deployment produces a file
   containing every login password ever submitted against the backend.

The incident report's own evidence section redacted the password when
quoting the log line; that redaction was the instinctive recognition
that the field shouldn't be there in plaintext. The fix is to make the
middleware match the instinct.

## Severity

Low (today). Dev-only on Tailscale; no external reach; no production
credentials involved (the dev password and the prior-session Keycloak
admin password are both dev-tier).

**Becomes high the moment any of the following change:**

- The binary is deployed to a non-Tailscale-reachable environment
  with shared log access
- `/tmp/uisce-server.log` is shipped to a log aggregator that
  multiple operators can read
- The middleware is copied to another service binary that handles
  production credentials

## Reproduction

```bash
ssh eganpj@100.84.50.65
grep -c "Body:{" /tmp/uisce-server.log
grep "Body:{" /tmp/uisce-server.log | grep "api/auth/login" | head -3
```

The second command returns lines containing `password` in plaintext.
Both the prior-session Keycloak admin credential (redacted) and the current dev password
appear.

## Proposed Fix

Either of two approaches is sufficient; pick by environment.

### Approach A — redact password fields (preferred)

Modify the body logger at `api.go:722` to skip known-credential fields
in known-credential paths:

```go
redactedBody := redactBody(req.URL.Path, bodyBytes)
fmt.Fprintf(os.Stderr, "[REQ] %s %s Headers:%s Body:%s\n", req.Method, req.URL.Path, strings.Join(headerParts, ","), string(redactedBody))
```

`redactBody` rules:

- For `POST /api/auth/login`: replace the `"password":"..."` field
  with `"password":"<redacted>"`. Other fields (email) pass through.
- For other paths: if a JSON object body contains a top-level key
  named `password`, `current_password`, `new_password`,
  `old_password`, or `secret`, redact that field's value.
- Non-JSON bodies: pass through unchanged (the formatter only applies
  to known-JSON paths).

This is targeted (no behavior change outside login/auth-rotation
endpoints), low-risk, and matches the redaction practice already
followed when quoting log lines in incident reports.

### Approach B — exclude `/api/auth/login` from body logging

Skip the body read entirely for the login path:

```go
if req.URL.Path == "/api/auth/login" {
    fmt.Fprintf(os.Stderr, "[REQ] %s %s Headers:%s Body:<redacted>\n", req.Method, req.URL.Path, strings.Join(headerParts, ","))
} else {
    fmt.Fprintf(os.Stderr, "[REQ] %s %s Headers:%s Body:%s\n", req.Method, req.URL.Path, strings.Join(headerParts, ","), string(bodyBytes))
}
```

Simpler, but only handles the login endpoint. Any future
credential-bearing endpoint (e.g. password rotation, API key creation)
needs its own case.

### Recommended: Approach A.

`Approach A`'s field-name redaction generalizes — any endpoint that
later takes a `password` or `secret` field is covered without code
changes. Approach B requires per-endpoint maintenance.

### Plus: gate behind an environment flag

The middleware's own comment says it "should only be enabled during
local debugging." Wire that intent:

```go
if os.Getenv("REQUEST_TRACE_VERBOSE") == "true" {
    r.Use(verboseRequestLogger)
}
```

Default off in production-equivalent environments; opt-in for local
debugging. Verbose mode retains the unredacted behavior for operators
who need it; default mode applies Approach A redaction.

## Verification (post-fix)

```bash
# 1. Login succeeds (unchanged)
curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser@example.com","password":"<dev password>"}'

# 2. Tail the log; the body should contain '"password":"<redacted>"', not the literal
ssh eganpj@100.84.50.65 'tail -F /tmp/uisce-server.log | grep "api/auth/login"'
# After triggering a login: Body should contain "<redacted>" in place of the password value.
```

## Severity Revisited After Fix

After Approach A lands:

- Login attempts still produce a log line (for debugging value) but
  the password field is replaced with `<redacted>`.
- No credential material accumulates in the log.
- The middleware's diagnostic utility (request method, path, headers,
  body shape, business-object routing markers) is preserved.

## Hygiene Note

`backend/internal/api/api.go:706` middleware is the second
plaintext-credential-leak finding this session (first:
session-password-in-AGENTS.md, 2026-09-13). Both follow the same
shape: credential produced in-session, then persists in an artifact
(markdown documentation / log file) that outlives the session. The
broader rule: **anything that touches a credential must default to
redacting it on its way out of the process.** The incident report's
"redact when quoting" instinct is the same rule, applied at the
documentation layer; this ticket is applying it at the source.
