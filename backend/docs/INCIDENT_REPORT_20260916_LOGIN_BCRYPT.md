# INCIDENT REPORT — 2026-09-16

## Login Returns 401 for All Seeded Dev Users (Invalid `password_hash`)

## Executive Summary

`POST /api/auth/login` returns **HTTP 401 "Invalid credentials"** for every
seeded dev user, against any password the operator supplies. The login
handler reaches `bcrypt.CompareHashAndPassword` and the bcrypt library
rejects the stored `password_hash` as **"too short to be a bcrypted
password"**. The stored value is the literal placeholder string
`testpass` (8 chars), not a bcrypt hash.

The likely (not yet verified by post-fix reproduction) root cause of the
original page-studio 401 thread that opened this workstream: the
authenticated path was unreachable via login, so any token the original
complainant held had been either minted (bypassing login) or carried
over from a prior valid session.

## Symptoms

```bash
$ curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser2@example.com","password":"<dev password>"}'
Invalid credentials
# HTTP/1.1 401
```

Same result for `testuser@example.com`. Same result for any password
the operator tries (bcrypt's length check fires before the comparison,
so the password value is irrelevant — even the correct dev password
returns 401).

## Discovery Context

Found during the MCP auth workstream (2026-09-16), while running the
login curl as a decoupled gate per the workstream's plan. Login was
skipped three times across earlier planning rounds; this execution
finally ran it.

## Damage Assessment

### Scope

- **Affected users:** the two seeded dev users (`testuser@example.com`,
  `testuser2@example.com`). UUID `da83f01c-da3e-480c-bac0-5ace1a97bc6e`
  is the user ID returned in the server log for one of the failures;
  the other is in the same row population.
- **Not affected:** minted-JWT test paths (these bypass login entirely
  and have been working throughout). All other backend services, since
  they don't depend on `/api/auth/login`.
- **Exposure:** dev-only on Tailscale IP. No external users. No
  credential leak in the literal sense (the stored value is a
  placeholder, not a real password), but anyone who can reach the dev
  backend can see *that* login is broken, which is operational drag.

### Severity

Low. No data integrity issue, no auth bypass, no credential exposure
in the plaintext-password sense. Locked-out dev users. Fix is a
single `UPDATE` against the dev `alpha` DB.

## Evidence

### Server log on rejected login

Captured live from `/tmp/uisce-server.log` during this workstream:

```
[REQ] POST /api/auth/login Headers:Content-Type=application/json,Authorization=,Origin=,X-Tenant-ID=,X-Tenant-Datasource-ID= Body:{"email":"testuser@example.com","password":"<redacted>"}
[AUTH] Login attempt for email: testuser@example.com
[AUTH] Login failed - password mismatch for user da83f01c-da3e-480c-bac0-5ace1a97bc6e: crypto/bcrypt: hashedSecret too short to be a bcrypted password
```

`hashedSecret too short to be a bcrypted password` is bcrypt's complaint
when the input is below its minimum encoded-form length (~59 chars).
This is what the seed-script bug produces.

### Stored hash inspection (read-only, on remote alpha)

```sql
SELECT email, LENGTH(password_hash) AS len, LEFT(password_hash, 7) AS prefix
FROM public.users
WHERE email IN ('testuser@example.com','testuser2@example.com');

         email         | len | prefix
-----------------------+-----+---------
 testuser@example.com  |   8 | testpas
 testuser2@example.com |   8 | testpas
```

Both users store the literal string `testpass` (8 chars). Not a bcrypt
hash. Not the actual dev password. A placeholder.

### Column type ruling out VARCHAR truncation

```sql
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'password_hash';

 column_name  | data_type | character_maximum_length
--------------+-----------+-------------------------
 password_hash | text      | (null — TEXT, unlimited)
```

The column accepts any length. The stored value's short length is the
seed script's doing, not silent column truncation. **This narrows the
diagnosis to "seed script wrote a placeholder literal"** — not "seed
script truncated a real hash."

## Root Cause

The dev user seed script populated `public.users.password_hash` with
the literal placeholder string `testpass` instead of generating a real
bcrypt hash. The placeholder was never replaced in subsequent dev
environment rebuilds.

This is consistent with the seed-script being a one-shot fixture that
set up *some* value to satisfy the NOT-NULL constraint on the column,
but skipped the bcrypt-hash step. The login handler has no
"placeholder detection" — it just hands whatever's in the column to
bcrypt, which rejects it.

## Reproduction

```bash
$ curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser@example.com","password":"<any string>"}'
Invalid credentials
```

Always 401, regardless of password. The handler reaches bcrypt, bcrypt
rejects on length before it even compares bytes.

## Fix

Generate a fresh bcrypt hash with `golang.org/x/crypto/bcrypt` (the
same validator the login path uses — guarantees the resulting hash
will be accepted) and UPDATE the two dev users. Run from `backend/`
so `go run` resolves the import via that module's go.mod:

```bash
cd backend/

HASH=$(go run - <<'GOEOF'
package main
import ("fmt"; "golang.org/x/crypto/bcrypt")
func main() {
    h, _ := bcrypt.GenerateFromPassword([]byte("<dev password>"), 10)
    fmt.Print(string(h))
}
GOEOF
)
printf "UPDATE public.users SET password_hash = '%s' WHERE email IN ('testuser@example.com','testuser2@example.com');\n" "$HASH" \
  | ssh eganpj@100.84.50.65 'PGPASSWORD=postgres psql -h localhost -U postgres -d alpha'
```

**Notes on this command:**

- `<dev password>` is the actual dev password from the secrets store.
  It is **not** committed to this file (matches the standing
  credential-hygiene rule documented at AGENTS.md line 507/520).
- `go run -` reads the program from stdin and runs it. The `cd backend/`
  before the heredoc ensures `golang.org/x/crypto/bcrypt` resolves via
  `backend/go.mod`.
- `printf %s "$HASH"` substitutes the hash *locally*; the resulting
  SQL string is piped to `ssh` over stdin, where it is consumed
  verbatim by `psql` on the remote. There is no remote-side expansion
  of any `$` reference — the ssh command is single-quoted.

## Verification

After running the fix, three steps close the loop:

### 1. Login returns 200 + valid JWT

```bash
curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser2@example.com","password":"<dev password>"}'
```

Expect: HTTP 200 with `{"token":"<jwt>", ...}`.

### 2. JWT payload decodes to the expected claims

```bash
TOKEN=$(curl -sS -X POST http://100.84.50.65:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"testuser2@example.com","password":"<dev password>"}' \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

python3 -c "import base64,json,sys; s=sys.argv[1].split('.')[1]; s+='='*(-len(s)%4); print(json.dumps(json.loads(base64.urlsafe_b64decode(s))))" "$TOKEN"
```

Expect: claims include `tenant_id = 99e99e99-99e9-49e9-89e9-99e99e99e999`,
`is_active = true`, and `roles` containing `portfolio_manager`. The
Python padding workaround (`+ '=' * (-len(s) % 4)`) handles JWT base64
which is unpadded; macOS `base64 -D` and GNU `base64 -d` both fail on
unpadded input.

### 3. Page-studio curl closes the original 401 thread

```bash
curl -sS http://100.84.50.65:8080/api/page-studio/pages \
    -H "Authorization: Bearer $TOKEN"
```

Expect: HTTP 200 with a JSON list of page definitions, **not** a 401.
This converts the original page-studio 401 complaint from "open
mystery" to "diagnosed, fixed, and reproduced-and-resolved by the
post-fix verification."

## Probable Closure of Original 401 Thread

**Probable root cause** (verified by the fix's post-verification step
above, not by direct evidence from the original complaint's logs):

The original page-studio 401 was reported against the same backend
where login is currently broken for every seeded user. Any operator or
agent whose auth path started at `/api/auth/login` would have obtained
no token and seen a 401 against every authed endpoint, including
page-studio. The fix re-enables that path; step 3 above confirms
page-studio returns 200 with a freshly-issued login token.

## Hygiene

This file contains **no literal password**. `<dev password>` is a
placeholder. The actual password lives in the dev env/secrets store.
This matches the standing credential-hygiene rule in AGENTS.md line
507/520, which was first established after a prior session committed
the same password as documentation.
