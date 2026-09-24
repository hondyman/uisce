# INCIDENT REPORT — 2026-09-16

## Login Returns 401 for All Seeded Dev Users (Invalid `password_hash`)

## Executive Summary

`POST /api/auth/login` returned **HTTP 401 "Invalid credentials"** for every
seeded dev user, against any password the operator supplied. The login
handler reached `bcrypt.CompareHashAndPassword` and bcrypt rejected the
stored `password_hash` as **"too short to be a bcrypted password"**. The
stored value was the literal placeholder string `testpass` (8 chars), not
a bcrypt hash.

**Status: Resolved.** Verified 2026-09-16. Login now returns HTTP 200
with a valid JWT carrying the expected claims; `/api/page-studio/pages`
with the login-issued token returns HTTP 200 with the tenant's page
definitions. The original page-studio 401 thread that opened this
workstream is closed (loop-closer executed; see Verification section).

## Symptoms

```bash
$ curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser2@example.com","password":"<any string>"}'
Invalid credentials
# HTTP/1.1 401
```

Same result for `testuser@example.com`. Same result for any password the
operator tried, because bcrypt's length check fires *before* any
comparison — the password value is irrelevant. Even the correct dev
password returned 401.

## Discovery Context

Found during the MCP auth workstream (2026-09-16), while running the
login curl as a decoupled gate per the workstream's plan. Login was
skipped three times across earlier planning rounds; this execution
finally ran it.

## Damage Assessment

### Scope

- **Affected users:** the two seeded dev users (`testuser@example.com`,
  `testuser2@example.com`). UUID `811e9f41-622f-4ef0-90ad-1098e1407d85`
  for `testuser2@example.com`; `da83f01c-da3e-480c-bac0-5ace1a97bc6e` for
  `testuser@example.com`.
- **Not affected:** minted-JWT test paths (these bypass login entirely
  and have been working throughout). All other backend services, since
  they don't depend on `/api/auth/login`.
- **Exposure:** dev-only on Tailscale IP. No external users. No
  credential leak in the literal sense (the stored value was a
  placeholder, not a real password), but anyone reaching the dev backend
  could see that login was broken.

### Severity

Low. No data integrity issue, no auth bypass, no credential exposure
in the plaintext-password sense. Locked-out dev users. Fix is a single
file: `backend/db/migrations/20260916_fix_dev_user_password_hashes.up.sql`.

## Evidence

### Server log on rejected login (pre-fix)

Captured live from `/tmp/uisce-server.log` during this workstream:

```
[REQ] POST /api/auth/login Headers:Content-Type=application/json,Authorization=,Origin=,X-Tenant-ID=,X-Tenant-Datasource-ID= Body:{"email":"testuser@example.com","password":"<redacted>"}
[AUTH] Login attempt for email: testuser@example.com
[AUTH] Login failed - password mismatch for user da83f01c-da3e-480c-bac0-5ace1a97bc6e: crypto/bcrypt: hashedSecret too short to be a bcrypted password
```

`hashedSecret too short to be a bcrypted password` is bcrypt's complaint
when the input is below its minimum encoded-form length (~59 chars).
This is what the seed-script bug produces.

### Stored hash inspection (pre-fix)

```sql
SELECT email, LENGTH(password_hash) AS len, LEFT(password_hash, 7) AS prefix
FROM public.app_user
WHERE email IN ('testuser@example.com','testuser2@example.com');

         email         | len | prefix
-----------------------+-----+---------
 testuser@example.com  |   8 | testpas
 testuser2@example.com |   8 | testpas
```

Both users stored the literal string `testpass` (8 chars). Not a bcrypt
hash. Not the actual dev password. A placeholder.

### `public.users` is a VIEW on `public.app_user`

`public.users` is a view, not a base table. Defined at
`backend/fix_auth_schema.sql:36-52` as a single-table auto-updatable
view over `public.app_user`:

```sql
CREATE OR REPLACE VIEW public.users AS
SELECT id, email, name, role, organization, permissions, is_core_admin,
       is_active, password_hash, tenant_id, username, display_name,
       created_at, salt
FROM public.app_user;
```

The login handler (`backend/internal/api/auth_handlers.go:110`) queries
`public.users`, which reads from the base table `public.app_user`. **All
fixes and the new migration must target `app_user`** — using the view
works today (single-table auto-updatable) but couples the doc to a
Postgres behavior that can silently change in future schema revisions.

### Column type ruling out VARCHAR truncation

```sql
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'app_user' AND column_name = 'password_hash';

 column_name  | data_type | character_maximum_length
--------------+-----------+-------------------------
 password_hash | text      | (null — TEXT, unlimited)
```

`app_user.password_hash` is `TEXT` (unlimited). The 8-char stored value's
short length is the seed script's doing, not silent column truncation.
**Narrows the diagnosis to "seed script wrote a placeholder literal"** —
not "seed script truncated a real hash."

## Root Cause

The dev user seed for `testuser@example.com` and `testuser2@example.com`
populated `public.app_user.password_hash` with the literal placeholder
string `testpass` instead of generating a real bcrypt hash.

**Original rows' provenance is unknown.** No committed migration or
fixture file seeds these users. Grep across `*.sql`, `*.sh`, `*.go`
returns zero matches for either UUID (`da83f01c-...`, `811e9f41-...`) or
email. The rows exist via an unrecorded action — chain of custody is
broken at the origin. Unrecorded mutations to a dev DB are themselves a
finding: they hide the chain of custody and are how the next mystery
starts.

The 20260825 alpha-wealth seed
(`backend/db/migrations/20260825_seed_alpha_wealth_users.up.sql`)
populates a different user set (e.g. `sarah.c@alpha-wealth.com`) into
`public.app_user` with real bcrypt hashes — these were never broken.

## Reproduction

```bash
$ curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser@example.com","password":"<any string>"}'
Invalid credentials
```

Always 401, regardless of password. The handler reaches bcrypt, bcrypt
rejects on length before any comparison.

## Fix

Generate a fresh bcrypt hash with `golang.org/x/crypto/bcrypt` (the
same validator the login path uses — guarantees the resulting hash is
accepted) and apply the upsert migration
`backend/db/migrations/20260916_fix_dev_user_password_hashes.up.sql`.

The migration is an **upsert** (not just an UPDATE) — it inserts the
rows on a fresh rebuild and updates the column on the current DB.
Either way, after the migration runs, both rows carry a real bcrypt
hash. The column set mirrors the live rows (id, email, username,
display_name, name, is_active, password_hash, tenant_id, language,
status, attributes, permissions, is_core_admin, created_at, updated_at).
**`tenant_id` is the load-bearing column** — without it, login would
work but every tenant-scoped endpoint would return 401, recreating the
original complaint with a different shape.

The migration contains no `BEGIN`/`COMMIT` because the migration runner
(`backend/internal/migrations/runner.go:224-226`) rejects
transaction-control statements (it wraps each file in its own tx).

### Application

```bash
ssh eganpj@100.84.50.65 'PGPASSWORD=postgres psql -h localhost -U postgres -d alpha' \
  < backend/db/migrations/20260916_fix_dev_user_password_hashes.up.sql
# INSERT 0 2
```

### Post-fix hash inspection

```sql
SELECT email, LENGTH(password_hash) AS len, LEFT(password_hash, 7) AS prefix, tenant_id, is_active
FROM public.app_user
WHERE email IN ('testuser@example.com','testuser2@example.com');

         email         | len | prefix  |              tenant_id               | is_active
-----------------------+-----+---------+--------------------------------------+-----------
 testuser@example.com  |  60 | $2a$10$ | 99e99e99-99e9-49e9-89e9-99e99e99e999 | t
 testuser2@example.com |  60 | $2a$10$ | 99e99e99-99e9-49e9-89e9-99e99e99e999 | t
```

Both rows: 60-char hash, `$2a$10$` prefix (valid bcrypt cost-10), correct
`tenant_id`, `is_active = t`.

### One-shot equivalent

If the migration cannot be applied immediately (no runner available,
operator wants direct psql), the same effect via:

```sql
UPDATE public.app_user
SET password_hash = '<bcrypt hash>',
    updated_at    = NOW()
WHERE email IN ('testuser@example.com','testuser2@example.com');
```

**Both the doc and the migration target `app_user`.** Do not use
`public.users` (the view) — auto-updatable today, but couples to
Postgres behavior that may change.

### Trap avoided: `echo` trailing newline

First attempt at the fix used `echo "$PASS" > /tmp/dev_user_password.txt`,
which writes a trailing newline. `bcrypt.GenerateFromPassword` then
hashed 17 bytes (16 chars + `\n`) instead of 16, and login's
`CompareHashAndPassword` rejected the 16-char password from the curl
body. Fix: `printf '%s' "$PASS" > ...` or strip the newline before
hashing. **Verification step (login → 200) catches this — that's why
Step 7 reads the response body, not just the status code.**

## Verification

After running the fix, three steps close the loop. Executed 2026-09-16
with `<dev password>` = a 16-char random string from
`openssl rand -base64 12` (not committed; only the bcrypt hash is in the
migration).

### 1. Login returns 200 with valid JWT

```bash
curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser2@example.com","password":"<dev password>"}'
```

Observed (response body):

```json
{
  "user": {
    "id": "811e9f41-622f-4ef0-90ad-1098e1407d85",
    "email": "testuser2@example.com",
    "is_active": true,
    "tenant_id": "99e99e99-99e9-49e9-89e9-99e99e99e999"
  },
  "access_token": "<jwt>",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

HTTP 200. `is_active = true`. JWT decodes to claims including
`tenant_id = 99e99e99-99e9-49e9-89e9-99e99e99e999` and
`user_id = 811e9f41-622f-4ef0-90ad-1098e1407d85`.

### 2. JWT payload decodes to expected claims

```bash
TOKEN=$(curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser2@example.com","password":"<dev password>"}' \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

python3 -c "import base64,json,sys; s=sys.argv[1].split('.')[1]; s+='='*(-len(s)%4); print(json.dumps(json.loads(base64.urlsafe_b64decode(s)), indent=2))" "$TOKEN"
```

Expected: `tenant_id = 99e99e99-99e9-49e9-89e9-99e99e99e999`,
`user_id = 811e9f41-622f-4ef0-90ad-1098e1407d85`, `email = testuser2@example.com`.

Note: `is_active` is not a top-level JWT claim in this issuer's setup —
it lives in the `user` object of the login response. The
`tenant_id` assertion is the load-bearing one (catches the NULL-tenant
failure mode the upsert migration's full-column INSERT is designed to
prevent).

### 3. Page-studio curl closes the original 401 thread

```bash
curl -sS http://100.84.50.65:8080/api/page-studio/pages \
    -H "Authorization: Bearer $TOKEN"
```

Observed: HTTP 200 with a JSON array of two page definitions
(`Order Detail`, `Order List`) — the tenant's actual pages. **Not 401.**
This converts the original page-studio 401 complaint from "open mystery"
to "diagnosed, fixed, reproduced-and-resolved by the post-fix
verification."

## Recurrence Prevention

The fix is an **upsert migration** at
`backend/db/migrations/20260916_fix_dev_user_password_hashes.up.sql`.
On fresh rebuild, the migration inserts valid rows (where they were
previously unrecorded); on the current DB, it updates the column. Either
way, after the migration runs, both rows carry a real bcrypt hash.

The migration runner (`backend/internal/migrations/runner.go`) discovers
the file via the `db/migrations/` glob and registers its SHA-256 in
`oms.migration_log`. Re-running on next startup is a no-op (hash match,
line 217). The upsert pattern makes re-application harmless even if the
log entry is missing.

## Password Selection

The placeholder `testpass` was a stand-in; no one chose a real dev
password at seed time. The fix picks a fresh **16-char random preimage**
generated by `openssl rand -base64 12` — captured to a temp file before
hashing so the literal never sits in shell history without intent.

The literal never appears in:
- The migration (only the bcrypt hash)
- This incident report (placeholder `<dev password>`)
- AGENTS.md

The bcrypt hash that *does* appear in the migration is safe to commit
because the preimage is high-entropy (16 chars ≈ 96 bits). A memorable
password would be crackable offline from the repo; a random 16-char one
is not.

### Infisical entry (deferred follow-up)

The 16-char preimage was *not* pushed to `shared/DEV_USER_PASSWORD`
during this workstream. The live Infisical instance at
`http://100.84.50.65:8085` returned **404 Not Found** for the project
ID documented in `frontend/.env.local` and `uisce_backend/.env`
(`9af25976-bafc-4895-a057-2effec13c620`):

```
Request: GET http://100.84.50.65:8085/api/v4/secrets?environment=dev&...&projectId=9af25976-bafc-4895-a057-2effec13c620&...
Response Code: 404 Not Found
Message: Project with ID '9af25976-bafc-4895-a057-2effec13c620' not found during bot lookup.
```

Per the workstream's "don't retry or improvise around it" rule, the
Infisical push, the `infisical-secrets-template.json` entry, and the
AGENTS.md line 543 rewording are all **deferred together**. They
happen, or they don't — not the hybrid "AGENTS.md updated to claim the
entry will exist after a follow-up."

Follow-up owed (not done this session):
1. Locate the correct project ID in the live Infisical. Candidates:
   workspace ID `81988783-b8ef-4e09-bf71-e7849fe8350d` from
   `.env.infisical.bak`, or project slug `uisce-tst-7`.
2. `infisical secrets set DEV_USER_PASSWORD='<16-char random>' --projectId=<correct ID> --env=dev --path=/shared`
3. Verify with `infisical secrets get DEV_USER_PASSWORD --projectId=<correct ID> --env=dev`.
4. *Then* update `AGENTS.md` line 543 to reference the now-existing entry.
5. *Then* add `shared/DEV_USER_PASSWORD` to `infisical-secrets-template.json`.

Until the follow-up runs, AGENTS.md line 543 keeps its generic pointer
("password in the dev env/secrets store"). The 16-char preimage is **not**
in any documented location: the Infisical push was deferred (Path B), and
the temp file (`/tmp/dev_user_password.txt`) was deleted at Step 8 of the
workstream. The preimage currently exists in exactly one place — the
`/tmp/uisce-server.log` file, where the dev-mode request-trace
middleware logged it on every verification curl.

**Correction (added after the original commit `276c815b5`):** the prior
version of this section claimed the password was in shell history. That
claim was incorrect. Verified this turn: `~/.zsh_history` (1512 lines,
66 KB) and `~/.bash_history` (382 lines, 21 KB) both return zero matches
for the literal. All verification curls used `$(cat
/tmp/dev_user_password.txt)` substitution, not the literal, so the
literal never expanded into shell history. The only surviving copy at
the time this section was written was the log file.

**Recovery procedure (initial):** before any log truncation (the
`/tmp/uisce-server.log` redirect-truncate in the request-trace
middleware workstream), extract the literal from the log with
`ssh eganpj@100.84.50.65 'grep -o "<literal>" /tmp/uisce-server.log
| head -1' > /tmp/recovered.txt` and present it to the operator for
recording in their preferred credential store. The log truncation
then proceeds; the literal lives only with the operator and (after
rotation) in the Infisical entry. See
`backend/docs/INCIDENT_REPORT_20260916_REQUEST_TRACE_PLAINTEXT_PASSWORDS.md`
for the request-trace ticket that documents the middleware fix.

**Recovery failure (recorded 2026-09-16, commit `82321608c`):** the
initial recovery was performed but the operator handoff did not
happen — the recovery file was deleted at cleanup without the
operator recording the literal externally. The committed docs at
that point asserted the preimage "lives only with the operator,"
which was false. The session caught this in its closing review and
chose to regenerate rather than leave the docs asserting a handoff
that didn't occur.

**Multi-step regeneration sequence.** Across the workstream's run, the
preimage was lost three times before this one landed durably:

1. **Original preimage** — the value initially hashed into migration
   `ab4e17b32`. Never recorded in a credential store; the only
   surviving copy was in `/tmp/uisce-server.log` from the request-trace
   middleware, which the redaction workstream's truncate (commit
   `58b2ae9f9`) then destroyed. The first recovery failure.

2. **First regen preimage** (commit `82321608c`): a fresh
   `openssl rand -base64 16` value, hashed, applied, login verified
   200. Temp file deleted at cleanup **without** the operator handoff.
   The second recovery failure.

3. **Second regen preimage (out-of-band, uncommitted):** an attempt to
   present the password to the operator ran a regeneration and
   applied the SQL directly to the live DB via `psql < /tmp/mig2.sql`,
   **without first writing the migration file**. The migration kept
   the previous hash; the DB carried the new one. Discovered via the
   DB-vs-migration hash cross-check (`893f4ef5a`'s commit message
   documents the divergence). The preimage was never handed off and
   never committed. The third recovery failure.

4. **Third regen** (commit `893f4ef5a`): a fresh `openssl rand -base64
   16` value, the migration **written with the correct hash before
   applying**, applied, cross-checked DB hash == migration hash.
   Atomic. Login verified 200. Temp file held for operator recording.
   **This preimage is the one the operator recorded externally.**

The superseded preimages exist nowhere today — only their bcrypt
hashes are in repo history, which is inert without the preimages. All
three were captured only to temp files (no operator-side credential
store entry, no log, no shell history).

**Regenerated preimage (current state):** the dev user passwords are
now hashed from a 24-char random value (`openssl rand -base64 16`).
The migration `20260916_regenerate_dev_user_password.up.sql` carries
a single bcrypt hash
(`$2a$10$n1btyilO6DUZ83bh/HP5n.ay6wYS1nqyv.bK5pgQPJAUX5A/.izNu`)
that matches the live `public.app_user.password_hash`. Migration
committed in `893f4ef5a` is the source of truth; migration committed
in `82321608c` was an intermediate state with a different hash and
is operationally superseded (but kept in the repo for history).

The operator handoff for this preimage was performed correctly (this
time): the literal was extracted to a temp file and held until the
operator recorded it externally. The temp file is deleted after
recording. Verified end-to-end: login returns 200, JWT decodes with
`tenant_id = 99e99e99-99e9-49e9-89e9-99e99e99e999`,
`/api/page-studio/pages` returns 200.

The previous preimage (`ab4e17b32`'s) is **burnt** — it lives
nowhere. Migration `ab4e17b32` now produces login 401 because the
preimage it was generated against is unrecoverable. Migration
`893f4ef5a` supersedes it operationally; both files exist in the
repo. The 401 if anyone runs `ab4e17b32` against a fresh DB is
itself an alarm bell that the password chain was rotated.

Neither location (operator's credential store, then Infisical) is
appropriate as a *committed* artifact. The Infisical follow-up above
is load-bearing: until it lands, dev-user auth is one lost
operator-side credential store away from another regeneration cycle.

## Probable Closure of Original 401 Thread

**Probable root cause** of the original page-studio 401 thread: any
agent or human whose auth path started at `/api/auth/login` would have
obtained no token and seen a 401 against every authed endpoint,
including page-studio. Step 3 of the Verification section above
converts this from inference to **verified closure** by demonstrating
that with a valid bcrypt hash, login → JWT → page-studio returns 200
end-to-end.

## Hygiene

This file contains **no literal password**. `<dev password>` is a
placeholder throughout. The current preimage (24 chars from
`openssl rand -base64 16`) lives in the operator's preferred credential
store (recorded 2026-09-16, after three earlier failed handoffs
caught this session's recurring unrecorded-cleanup pattern; the
password was regenerated a third time under commit `893f4ef5a`).
The bcrypt hash is in migration
`20260916_regenerate_dev_user_password.up.sql`; the preimage is not.
This matches the standing credential-hygiene rule in AGENTS.md line
507/520, established after a prior session committed the same
password as documentation.
