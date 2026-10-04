# Alpha session — 2026-10-04

**One session. Three streams, in this order. Do not reorder them.**

| # | Stream | Closes | Needs |
|---|---|---|---|
| **1** | Control-plane restore, **executed** | #383 (P0) | alpha + a scratch database |
| **2** | Snapshot regeneration + migration-vs-reality | the last #376 verification debt | alpha read access |
| **3** | C2 schema audit | the sole blocker on calc-term lineage | alpha read access |

Stream 3 is executed from `docs/alpha-audit-2026-10.md`, which is the single
source for it and is **not** restated here. Open it and follow it. Its Q5
stop-rule still applies and can dissolve its own finding; respect it.

**Why this order.** Stream 1 is the only item that can surface during an actual
disaster, when it is most expensive, and it is the only one whose result
changes what is safe to do next: until a restore has run, the shared ORM
database must not be dropped (`docs/runbooks/orm-cutover.md` §"Dropping the
shared database is a gated, two-person decision"). Stream 2 closes a debt with
a deadline. Stream 3 unblocks a work stream. Run them in that order and stop if
the session is cut short — the ordering is chosen so that being interrupted
loses the least.

**If the session is cut short after stream 1:** that is the most valuable
outcome available, and everything else is still owed. Streams 2 and 3 have
their own documents; nothing in them is invalidated by a partial session.

---

## Session header

| Field | Value |
|---|---|
| Date run | |
| Operator | |
| Environment (alpha / staging / other) | |
| `origin/main` SHA at session time | |
| Scratch database name (stream 1) | |

**Discipline, the same for all three streams.** Record raw output, not
interpretation. Paste query results verbatim; if a command errors, paste the
error. Conclusions go in a conclusions section afterwards, separately, so a
later reader can check them against the raw output still sitting above them.

Mark every answer as one of:

- `verified-live` — ran it, have the output
- `couldn't-look` — could not obtain it. This is an honest and useful answer
  and is **not** the same as "the answer is no"

Anything this session cannot establish stays in the repository as a named debt.
It does not become a footnote.

---

## Stream 1 — Control-plane restore, executed (#383, P0)

### Why this is first

`docs/runbooks/dr-playbook.md` invokes **15 scripts, none of which exists in the
repository**, and applies **5 Kubernetes manifest paths, none of which exists**,
for a platform that is deployed with Docker Compose. The playbook's service
tier table claims `P0 · RTO 15 min · RPO 0` for auth and the API gateway, and
that claim is currently substantiated by nothing executable.

The control plane is the database whose loss loses everything: the datasource
registry, the tenant bindings, the catalog. It has no restore procedure, no
`pg_dump` step, and no backup mechanism — the only dumps in the tree are
committed artefacts, the newest dated **2025-11-03**.

### What "closed" means

Not "a restore script exists". Not "the playbook documents a restore". **A
restore that has been run, once, to a scratch target, with the elapsed time
recorded against the claimed RTO.** A restore script that has never been
executed is not a restore path.

### Steps

1. **Find what actually backs up the control plane today.** If nothing does, say
   so — that is the finding, and it is worse than "the script is missing".

   ```bash
   # What backup mechanism exists for alpha, if any?
   ```

   **Raw output:**

   ```
   ```

2. **Take a backup of the control plane by whatever means is available**, and
   record the size, the timestamp, and the command that produced it.

   ```bash
   ```

   **Raw output:**

   ```
   ```

3. **Restore it into a scratch database.** Not alpha. Not a tenant.

   ```bash
   ```

   **Raw output:**

   ```
   ```

4. **Prove the restored database is the control plane**, by answering questions
   that only the control plane can answer. A database that restored without
   error but is missing the registry has restored nothing that matters.

   ```sql
   -- Something whose absence means this is not a usable control plane.
   SELECT count(*) FROM <datasource registry table>;
   SELECT count(*) FROM <tenant binding table>;
   SELECT count(*) FROM public.catalog_node;
   ```

   **Raw output:**

   ```
   ```

5. **Record the elapsed time** for backup-then-restore, and compare it to the
   claimed `RTO 15 min / RPO 0`. If it is minutes rather than hours, the
   headline claim is conservative and should say so with evidence. If it is
   hours, the tier table is wrong and #383 stays open.

   **Elapsed, backup→restore verified:**

   ```
   ```

6. **Decide the deployment topology the playbook should describe** — Compose or
   Kubernetes — and record it. This is a decision, not a lookup; whichever way
   it goes, §4 of the DR playbook is currently wrong about how this platform is
   deployed.

### Answer

`verified-live` / `couldn't-look`

### What this does not close

It does not fix the playbook. It converts "unverified" into "verified at time
T", which is what makes every other claim in the document checkable. Writing
the missing scripts is separate work, and is worth doing **after** this, using
whatever this session learned about the real topology.

---

## Stream 2 — Snapshot regeneration, and the last #376 debt

### What is already settled, so this is smaller than it looks

The verification debt from #376 is "did the ORM migration produce what it
claims". It has three halves, and **half A is already done in the repository**
(ADR-047):

- **A — migration vs snapshot: DONE.** The tenant migration
  `backend/db/tenant_migrations/orm/0001_orm_schema.up.sql` and
  `backend/db/snapshots/schema-snapshot.sql` describe the **same 34 `orm`
  objects**, with identical names, and every one is tenant-scoped in both.

So this stream is only the two halves that need a live database. Both are
required: the snapshot has already been shown once to misrepresent reality, so
a clean comparison against it is evidence about two *files*, not about alpha.

### Step 1 — does the live control plane match the snapshot?

```sql
-- The orm tables alpha actually has, against the 34 the snapshot claims.
```

**Raw output:**

```
```

**Answer:** `verified-live` / `couldn't-look`

**If this disagrees with the snapshot:** the snapshot is stale, and the
discrepancy is the finding. Record which side moved and by how much — do not
regenerate until the difference is written down, or the evidence is lost.

### Step 2 — does a migrated tenant database match the migration?

Requires a tenant that has actually been cut over
(`docs/runbooks/orm-cutover.md` Steps 0–5). If no tenant has been migrated, say
so: `couldn't-look`, and note that half C is therefore unproven.

```sql
-- On a migrated tenant database: the 34 orm objects, and their columns.
```

**Raw output:**

```
```

**Answer:** `verified-live` / `couldn't-look`

**Prefer a structural comparison over a textual one.** Loading both schemas
into scratch databases and diffing `information_schema` gave a definitive answer
on half A where a text diff had produced phantom column differences. If a text
diff is the only option, diff parsed structures, not raw text — `character
varying(20)` and `character` are the same type.

### Step 3 — has the fleet move ever run, and what did it report?

`ormmove.FleetReport` carries `Moved`, `Failed` and `Done`. This is a field, not
a hope — read it.

**Raw output:**

```
```

**Answer:** `verified-live` / `couldn't-look`

### What this closes

The #376 verification debt, in full, **only if** all three of the above are
`verified-live`. Two out of three leaves the debt open with the remaining half
named.

---

## Stream 3 — C2 schema audit

**Execute `docs/alpha-audit-2026-10.md` as written. Do not restate it here.**

That document is the single source for this stream. It carries its own session
header, its own `verified-live` / `couldn't-look` discipline, and a **Q5
stop-rule that runs first and can dissolve the entire finding** — if Q5 returns
non-zero, phases 2 and 3 are not needed and the approach is re-derived.

The ordering constraint between streams still applies: **if the session is
running out of time, drop this stream, not stream 1.** This one is worth a full
session of its own rather than a rushed half of one.

---

## Conclusions

Written after the raw output above, not instead of it. Every claim here should
be checkable against a block above it.

| Item | Stream | Answer | What it unblocks |
|---|---|---|---|
| Control-plane restore executed | 1 | | #383 can close; the shared ORM database drop gate can be satisfied |
| Live `orm` vs snapshot | 2 | | the snapshot's trustworthiness |
| Migrated tenant vs migration | 2 | | the cutover's terminal claim |
| Fleet report | 2 | | whether the move has ever run |
| C2 Q5 / Q1–Q4 / four-file sweep | 3 | | calc-term lineage, reconciler backfill |
| Cube DDL + golden corpus capture | 3 | | the cube stream |

### Debt that remains after this session

- The DR playbook's 15 missing scripts and 5 wrong manifest paths remain.
  Stream 1 proves recoverability; it does not document it.
- The deployment-topology decision from stream 1 step 6 needs an owner.
- `docs/alpha-audit-2026-10.md`'s open items, if stream 3 is dropped.
- #379: the four `go.work` modules with no build and no test in any workflow.
  Not an alpha item; it is repository work and should not wait for this session.

### Related

- #383 — the DR playbook finding this session is the proof for
- ADR-046 — the ORM cutover's verification debt, and the point-of-no-return gate
- ADR-047 — half A of stream 2, already settled
- `docs/runbooks/orm-cutover.md` — the cutover procedure and the drop gate
- `docs/ARCHITECTURAL_DECISIONS.md` ADR-042 — the `orm` schema decision, with two
  corrections on the evidence
