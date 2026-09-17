# INCIDENT REPORT — 2026-09-16

## Rotate the Current Keycloak Admin Credential

## Executive Summary

The current Keycloak admin credential (value redacted here while live)
was logged in plaintext in `/tmp/uisce-server.log` before this session's
re-redaction middleware and log truncation landed. The middleware
(`backend/internal/api/api.go` `redactBody`/`redactAuthHeader`, commit
`58b2ae9f9`) stops further accumulation; rotation eliminates the
already-accumulated value.

**Status: Open.** Documented here as a standing follow-up. Currently
limited to dev-only Tailscale exposure; becomes a routine rotation
item once the credential is rotated.

## Discovery Context

Discovered during the post-fix cleanup of
`backend/docs/INCIDENT_REPORT_20260916_REQUEST_TRACE_PLAINTEXT_PASSWORDS.md`.
The pre-fix log contained 2 lines with the live Keycloak admin
credential in plaintext (alongside 6 lines with the dev-user password
and 3 unrelated password-field lines). The dev-user password was
rotated in this workstream; the Keycloak admin credential was not,
because its rotation procedure is outside this session's scope.

## Damage Assessment

- **Severity:** Low (today). Dev-only Tailscale; the log file was
  readable only by the local operator; the credential is single-operator
  (Keycloak admin, not a service credential).
- **Becomes routine:** the moment any other operator or log-aggregation
  pipeline gets read access to `/tmp/uisce-server.log` from a window
  where the credential was logged.

## First Step — Verify Which Candidate Is Live

Two candidates exist for the live Keycloak admin credential:

1. The historical prior-session value (Argon2id-hashed by Keycloak itself,
   documented in the 2026-09-13 incident postmortem at AGENTS.md line 520).
2. The bootstrap value in `docker-compose.remote.yml`:
   `KC_BOOTSTRAP_ADMIN_PASSWORD: password` (verified this session; see
   below).

Both can coexist — bootstrap sets the initial admin password at first
container start; subsequent rotation lives in Keycloak's own DB; the
compose file's value goes stale and is only consulted on a fresh
bootstrap. But the rotation ticket must not guess which is currently
live.

**Verify-then-rotate procedure:**

1. Try authenticating to Keycloak admin UI at `http://100.84.50.65:8443`
   with candidate 1 (the prior-session value, available in AGENTS.md
   line 520 historical context). If accepted, candidate 1 is live.
2. If candidate 1 fails, try candidate 2 (`password`). If accepted,
   candidate 2 is live — meaning the post-2026-09-13 rotation that
   was supposed to change it didn't take, or compose-bootstrap ran
   against a fresh DB at some point.
3. Record whichever candidate authenticated. That is the value to
   rotate.

## Procedure (out of scope for this session)

Rotation requires Keycloak admin access, which lives outside the repo:

1. Keycloak admin UI at `http://100.84.50.65:8443`, or
2. `docker exec uisce-keycloak /opt/keycloak/bin/kcadm.sh` with the
   verified-live admin credentials.

After rotation, update:
- `backend/.env` and any other `.env` files carrying the value
  (gitignored)
- Any operator-side credential stores
- The running Keycloak container's bootstrap admin (Keycloak stores
  these in its DB, not in compose env vars at runtime)

## Compose ≠ Source of Truth

**Compose files do not contain the live credential.** Verified this
session:

- `docker-compose.remote.yml` IS tracked in git (`git ls-files
  --error-unmatch` exit 0; `git check-ignore` exit 1 = NOT ignored).
- The file contains `KC_BOOTSTRAP_ADMIN_PASSWORD: password` — a literal
  `password`, NOT the live prior-session value.
- No tracked compose file contains the live credential. The single
  tracked file containing the historical prior-session value is
  `AGENTS.md` (postmortem context, line 507/520).
- `docs/archives/docker-compose.override.yml` (an archived file)
  carries `KEYCLOOK_ADMIN_PASSWORD` (note typo: KEYCLOOK, not
  KEYCLOAK) but the grep returned no literal match — different format.

Implication: the rotation procedure is primarily a `.env` update +
Keycloak admin UI update, not a compose change. Compose's
`KC_BOOTSTRAP_ADMIN_PASSWORD: password` stays — it's a separate,
weaker literal that the AGENTS.md pre-push hook at line 481 already
defends against as `admin:<historical prior-session value>` (different value, different
defense line).

## Hygiene Items Surfaced During This Investigation

### Tracked compose file carrying a bootstrap admin password literal

`docker-compose.remote.yml` contains `KC_BOOTSTRAP_ADMIN_PASSWORD: password`
as a committed literal. The AGENTS.md pre-push hook (line 481) defends
against `<historical prior-session value>` reintroduction, but `password` sits committed in the
very file class the hook watches. This is a separate, smaller hygiene
issue: out of scope this PR; mentioned for awareness. The hook regex
could be extended to catch trivial literals like `password` on
production-relevant env vars, but that's a follow-up design call.

### AGENTS.md Posture

`AGENTS.md` line 481 (pre-push hook tripwire), 507 (section heading),
520 (postmortem body) reference the historical prior-session value.
These mentions are part of the 2026-09-13 incident postmortem — they
document the incident that put the value in committed history; they
are **not new references to a live credential**. The historical-dead
posture holds: the value becomes historical after rotation, the
mentions stay.

After rotation: optionally update line 481's tripwire regex to track
the *new* value, so the pre-push hook catches re-introduction of the
new value too.

## Hygiene Note

This file contains **no literal credential value** — by the same
rule the login fix and the request-trace middleware established. The
post-rotation value lands in `.env` files (gitignored) and operator-side
credential stores, not in committed documentation. If a future ticket
needs to reference the *old* value as burned historical evidence,
that's the moment the literal becomes legitimate to record.
