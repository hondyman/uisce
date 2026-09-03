# fix(pki): recover or regenerate Uisce ca.key — PKI issuance frozen on this host

## Summary

The Uisce PKI signing private key (`ca.key`) is not found on this
host via the captured search scope (`find $HOME -name 'ca.key'`,
Library-excluded; no Trash exclusion). `~/.Trash` was denied by macOS
TCC (`ls ~/.Trash/` → "Operation not permitted"). `mdfind` (Spotlight,
also TCC-gated) returned only test fixtures from
`google.golang.org/grpc@*/testdata/spiffe_end2end/ca.key` and
`confluent-kafka-go`'s `rootCA.key` — neither is the Uisce PKI key.

Files at `~/.uisce/certs/` on this host: `ca.crt`,
`postgres-client.crt`, `postgres-client.key`, `postgres-client.pk8` —
one role's client material (postgres) plus the CA cert. The CA
private key is missing.

## Implications for this host

- No future cert can be issued from here (rotation, re-issuance, new
  roles all impossible) — needs `ca.key`, which is not found in the
  captured scope.
- Issuance is frozen; verification works for postgres (the only role
  with cert material at `~/.uisce/certs/`). Other roles' certs live
  wherever the issuing host placed them (containers/mounts per the
  mTLS handoff) — verify-from-here is untested.
- Any new role would require regenerating the PKI entirely (new CA,
  new server cert, new per-role certs, new `pg_hba.conf`) — IF the
  key is unlocatable.

## Two-step recovery (in order)

### Step 1 — Finder Trash check (your action only)

macOS TCC denies this sandboxed terminal. Required: open Finder →
Empty Trash (the GUI checkbox under Finder > Settings > Advanced) and
look for `ca.key`. Or run `find $HOME -name 'ca.key' -not -path
'*/Library/*'` from a terminal with Full Disk Access.

- **If present**: drag out, store durably (encrypted backup,
  password manager, or Infisical), update the verification record's
  PKI section to mark the gap closed.
- **If absent**: proceed to Step 2.

### Step 2 — locate on the original issuing host

The mTLS handoff said `ca.key` "exists only on the machine that did
this work." Locate it on the original issuing host and copy it to
durable storage (encrypted backup, password manager, or Infisical).

If the issuing host is no longer reachable / the key is genuinely
lost: the PKI is frozen at its current 9-role surface. New roles
require full PKI regeneration (new CA, new server cert, new
per-role certs, new `pg_hba.conf`).

## Acceptance

- `ca.key` is located (Step 1 or Step 2) AND stored in durable
  storage accessible to this host, OR
- A documented decision to regenerate the PKI is recorded, with a
  plan and timeline for the new PKI.

## Reference

- Verification record: `docs/ops/finops-predictive-verification.md`
  § "ca.key (Uisce PKI signing private key) — escalation"
- mTLS handoff: `backend/.env` (the DSN params; PKI provenance
  documented in the conversation that introduced the mTLS hardening)
- First surfaced: `b8103da33` (runner.go authoritative-bookkeeping
  doc references the PKI gap); recorded in `c0a614ae8`
  (verification record commit)
