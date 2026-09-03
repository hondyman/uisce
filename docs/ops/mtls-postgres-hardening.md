# Postgres mTLS Hardening — Handoff Summary

> *Transcribed from the operator handoff pasted in the working session
> of Sept 2–3 2026; the original text lives outside the repo. This
> document is a snapshot, not a live extraction — future operators
> cross-checking against live `pg_hba.conf` or `/etc/ssl` must verify
> the values here haven't drifted.*

## Context

The Postgres instance at `100.84.50.65:5432` (database `alpha`, plus
`infisical`, `keycloak`, `nessie` databases) previously allowed
unauthenticated access to *any* role from Docker's bridge network
(`172.16.0.0/12 trust` in `pg_hba.conf`) and password-only access
from the Tailscale range (`100.64.0.0/10 scram-sha-256`). Both
catch-alls have been removed. Every one of the 9 login roles now
requires either a client certificate (mutual TLS) or, for one role
that architecturally can't present a cert, TLS + password against a
dedicated non-superuser account.

## PKI

- Self-signed CA: `CN=uisce-dev-postgres-ca`, 4096-bit RSA, 10-year
  validity.
- Server cert: `CN=100.84.50.65`, with SAN
  `IP:100.84.50.65, DNS:100.84.50.65` (required — modern TLS stacks,
  e.g. Go's `crypto/x509` and Java's default TLS, ignore CN for
  hostname verification and require a SAN).
- Per-role client certs: 2048-bit RSA, CN = the exact Postgres role
  name (required for `clientcert=verify-full`, which checks cert CN
  against the connecting role).
- All cert/key material generated locally; CA private key
  (`ca.key`) exists only on the machine that did this work — needed
  for issuing any future certs (e.g. rotating a service, adding a new
  role). **Not committed anywhere durable yet — see Gaps below.**
- The `postgres`-role client cert/key/CA bundle used by this app's
  own backend lives at `~/.uisce/certs/` on the dev Mac:
  `ca.crt`, `postgres-client.crt`, `postgres-client.key` (PEM),
  `postgres-client.pk8` (same key, PKCS#8 DER — for Java/JDBC
  clients like DBeaver; see below).

## Final `pg_hba.conf` state (relevant rules, in order)

```
local   all   postgres                          peer
local   all   all                               peer
host    all   all       127.0.0.1/32            scram-sha-256
host    all   all       ::1/128                 scram-sha-256
hostssl all   postgres  0.0.0.0/0  cert clientcert=verify-full
hostssl all   keycloak  0.0.0.0/0  cert clientcert=verify-full
hostssl all   temporal  0.0.0.0/0  cert clientcert=verify-full
hostssl all   nessie    0.0.0.0/0  cert clientcert=verify-full
hostssl all   semlayer_lookups_replica  0.0.0.0/0  cert clientcert=verify-full
hostssl all   usice_ops   0.0.0.0/0  cert clientcert=verify-full
hostssl all   usice_app   0.0.0.0/0  cert clientcert=verify-full
hostssl all   app_user    0.0.0.0/0  cert clientcert=verify-full
hostssl infisical infisical 0.0.0.0/0  scram-sha-256   -- TLS required, password auth (see below)
-- (the two catch-alls that used to sit here — 100.64.0.0/10 scram-sha-256 and
--  172.16.0.0/12 trust, both `all` database/`all` role — have been deleted)
local   replication   all                       peer
host    replication   all   127.0.0.1/32        scram-sha-256
host    replication   all   100.64.0.0/10       scram-sha-256
host    replication   all   172.16.0.0/12       trust
```

Replication-scoped rules were deliberately left untouched — out of
scope, separate risk domain.

## Per-role / per-service disposition

| Role | Service | Auth | Notes |
|---|---|---|---|
| `postgres` | This app's own Go backend (`cmd/server`, runs locally on the dev Mac) | mTLS | `DATABASE_URL`/`POSTGRES_DSN`/`UISCE_DATABASE_URL` now carry `sslmode=verify-full&sslcert=...&sslkey=...&sslrootcert=...`. Driver is `lib/pq`, which takes PEM file paths directly (no DER conversion needed for Go). |
| `keycloak` | `uisce-keycloak` container (OIDC provider) | mTLS | Was previously connecting via the `trust` bypass, undiscovered until the catch-alls were being removed. Uses `pgjdbc` (`KC_DB_URL` JDBC URL) — needed the client key converted to **PKCS#8 DER**, not PEM. Given its own dedicated role + cert (was sharing `postgres` role before). |
| `temporal` | `semlayer-temporal` container | mTLS | Uses `SQL_TLS_ENABLED`/`SQL_CA`/`SQL_CERT`/`SQL_CERT_KEY`/`SQL_HOST_VERIFICATION`/`SQL_HOST_NAME` env vars (Temporal's Postgres plugin). PEM key worked directly (no DER conversion needed for this driver). |
| `nessie` | `semlayer-iceberg-rest` container (Project Nessie, Iceberg REST catalog) | mTLS | Also `pgjdbc` — same PKCS#8 DER key requirement as Keycloak. JDBC URL host switched from `host.docker.internal` to `100.84.50.65` to match the server cert's SAN. |
| `app_user`, `usice_app`, `usice_ops`, `semlayer_lookups_replica` | Dormant — no live connections found | mTLS | Locked down pre-emptively; zero live risk since nothing currently uses them. Never independently verified against a real client (see Gaps). |
| `infisical` | `uisce-infisical` container (secrets manager) | **TLS + password, not mTLS** | Infisical's own `knexfile.mjs` only supports `DB_ROOT_CERT` (server CA verification) — no `cert`/`key` option exists in its Postgres SSL config at all. Client-cert auth is architecturally impossible without patching Infisical's source. Mitigated by giving it a dedicated non-superuser role (previously used the `postgres` superuser role for its own tiny `infisical` database) with TLS required and a strong random password. |

## Connecting with DBeaver

DBeaver's PostgreSQL driver is JDBC-based (`pgjdbc`), same family as
Keycloak/Nessie above — **it needs the client private key in PKCS#8
DER format, not PEM.** The `postgres-client.pk8` file already exists
for this at `~/.uisce/certs/postgres-client.pk8`; if connecting as a
different role, convert that role's PEM key the same way:

```bash
openssl pkcs8 -topk8 -inform PEM -outform DER -in <role>.key -out <role>.pk8 -nocrypt
```

Steps in DBeaver:

1. New connection → PostgreSQL.
2. **Main tab**: Host `100.84.50.65`, Port `5432`, Database `alpha`
   (or `keycloak`/`nessie`/`infisical` as needed), Username = the role
   you're connecting as (e.g. `postgres`). Leave password blank —
   cert auth ignores it.
3. **SSL tab**: enable SSL.
   - SSL mode: `verify-full` (this both requires TLS and checks the
     server cert's SAN against the host you typed — `100.84.50.65`
     matches).
   - Root certificate: `~/.uisce/certs/ca.crt`
   - Client certificate: `~/.uisce/certs/postgres-client.crt`
   - Client certificate key: `~/.uisce/certs/postgres-client.pk8`
     (the **DER** file, not the `.key` PEM)
4. Test Connection — should succeed with no password prompt.

For the `infisical` role specifically: skip the client cert fields
(it doesn't have one — see table above), set SSL mode to `require` or
`verify-ca`, and supply the role's password instead.

## Connecting from Go code

Any new Go code connecting to this Postgres instance as an
mTLS-enforced role needs the same four query parameters `lib/pq`/`pgx`
already use in this app's `DATABASE_URL`:

```
postgresql://<role>@100.84.50.65:5432/<database>?sslmode=verify-full&sslcert=<path-to-role.crt>&sslkey=<path-to-role.key>&sslrootcert=<path-to-ca.crt>
```

- No password in the DSN — cert auth doesn't check it.
- `sslmode=verify-full` is required, not just `require`/`verify-ca`
  — it's what makes the client check the server cert's hostname
  (SAN) too, not just trust any cert signed by the CA.
- `lib/pq` and `pgx`'s stdlib driver both accept the PEM `.key`
  directly via `sslkey=` — **no DER conversion needed for Go**,
  unlike Java/JDBC clients (DBeaver, Keycloak, Nessie above).
- If a new role is needed that doesn't have a cert yet, it has to
  be created first (`CREATE ROLE <name> LOGIN`, grant appropriate
  privileges) and issued a cert signed by `ca.key`/`ca.crt` with
  `CN=<exact role name>` — `clientcert=verify-full` matches the
  cert's CN against the connecting role, so a mismatched CN gets
  rejected even with a valid cert.
- The current live example, worth pointing an LLM at directly, is
  `backend/cmd/server/main.go` (reads `DATABASE_URL` via
  `os.Getenv`, opens with `sql.Open("postgres", dbURL)`) and
  `backend/.env`'s `DATABASE_URL` line.

## Gaps / follow-ups worth knowing about

1. **Cert file paths are machine-specific.** The backend's
   `DATABASE_URL` etc. point at `/Users/eganpj/.uisce/certs/...` —
   a path that only exists on this one Mac. If the backend is ever
   deployed elsewhere, this breaks. `lib/pq` doesn't support inline
   PEM content in the connection string, only file paths, so making
   this portable would need either shipping the certs to every
   runtime host or a code change.
2. **`hostssl all postgres 0.0.0.0/0 cert`** — superuser reachable
   from anywhere holding the cert. Dev-acceptable; the eventual
   posture is a dedicated least-privilege app role instead of the
   backend connecting as `postgres`, plus source-range restrictions.
   Note it, don't do it now.
3. **`ca.key` custody on one machine.** Existing certs keep working
   (10-year CA), but a disk failure means never issuing another cert
   — rotation, new roles, all locked out. Recovery procedure:
   `docs/ops/.issues/uisce-pki-ca-key-recovery.md` (Finder Trash
   check first, then original-issuing-host check). This is the one
   gap with unbounded future cost.
4. **Infisical role's password is not itself protected by anything
   beyond TLS.** It's a strong random value, but Infisical's design
   means the *database* side of this can never be fully
   passwordless/certless. Consider whether Infisical should be
   swapped for a tool with proper client-cert support if this needs
   to reach a higher bar.
5. **An account password and a live `INFISICAL_TOKEN` secret were
   pasted into plaintext chat/terminal output earlier** (in an
   unrelated debugging detour by another agent working the same
   session). Both should be treated as compromised and rotated if
   not already done.
6. **Infisical secret duplication risk**: this project's
   Infisical instance has secrets living at both the project root
   (`/`) and inside a `/core` folder, with the bootstrap script
   (`scripts/infisical-bootstrap.sh`) merging them (`/core` wins on
   collision). When updating any DB/URL-shaped secret going forward,
   check *both* locations don't disagree — a stale duplicate in the
   losing location is invisible until something reads it under
   different tooling/merge order.
7. **Not independently verified**: the `app_user`/`usice_app`/
   `usice_ops`/`semlayer_lookups_replica` roles were locked down but
   never had a real client actually exercise the new rule (nothing
   currently connects as them). If/when something starts using one
   of these roles, confirm the corresponding cert (already generated,
   CN matches role name) actually works before assuming it does.
8. **CA key custody**: `ca.key` (needed to issue any further certs)
   exists only in the working machine's scratch directory from this
   session, not committed anywhere durable. Move it somewhere it
   won't be lost (a password manager, an ops secrets vault) if this
   setup needs to be maintained long-term.

## Verification method used throughout

Every enforcement change was checked two ways before being trusted:
- **Positive**: the real service reconnects and shows up in
  `pg_stat_ssl` joined to `pg_stat_activity` with `ssl=true` and
  `client_dn` matching the expected role CN.
- **Negative**: a deliberate cert-less connection attempt as that
  role gets rejected with `FATAL: connection requires a valid
  client certificate` (for cert-enforced roles) or `no pg_hba.conf
  entry` (after the catch-alls were removed) — not silently falling
  through to some other rule.
