# StarRocks 3.3.22 → 4.1.x upgrade runbook

Single-node StarRocks on `100.84.50.65` (`starrocks-fe` + `starrocks-be` in Docker).
This runbook takes the cluster from the current `3.3.22-753696f` to a pinned
`4.1.1+` build with multi-tenant isolation preserved end-to-end.

---

## Constraints that shape this runbook (read these first)

- **Single FE + single BE**: every upgrade step is a full cluster outage. There is
  no rolling upgrade possible on this topology. The outage must be announced and
  covered by a real backup.
- **Container constraint**: do NOT target `4.1.0`. Per the StarRocks release notes
  (PR #71825), the `v4.1.0` container image has an unstable load order that
  causes BE processes to fail to start reliably in container environments.
  Pin to **`4.1.1` or later**, ideally by image digest after the first successful
  pull.
- **Downgrade is one-way across major versions.** The release notes explicitly
  state: *"After upgrading to v4.1, DO NOT downgrade to any v4.0 version below
  v4.0.6."* And there is no supported downgrade from 4.x to 3.x. Phase 4
  (rollback) is **backup + recompose at old digests**, not in-place downgrade.
- **Root is passwordless on the live cluster as of 2026-10-06.** Verified via
  `mysql -uroot` succeeding and `mysql -uroot -p""` being rejected. Phase 1 must
  rotate root *before* the upgrade window starts; doing it during the window
  entangles two failure domains.
- **`mysql -uroot -p""` is rejected on 3.3** but `mysql -uroot` (no `-p`) succeeds.
  This is the empty-string-password vs absent-password distinction. Any script
  that connects with `-p"$PASSWORD"` must omit `-p` when the password is empty.
- **`SHOW ALL GRANTS` and `SELECT FROM mysql.user` are NOT accessible** to regular
  users on 3.3 (the system schema is hidden). To enumerate users during the
  capture step, use `SHOW GRANTS FOR '<user>'@'%'` for each known user.
- **`RESOURCE GROUP` (not `WORKLOAD GROUP`) is the SQL construct in BOTH 3.3 and
  4.1.** 4.1's "workload filter" terminology in some pasted plans is incorrect.
  The 3.3 resource groups we create during Phase 1 survive the 4.1 upgrade
  unchanged — the only 4.1 additions are additive parameters (`cpu_weight_percent`,
  `exclusive_cpu_percent`, `mem_pool`).

---

## Phase 0 — Decisions

All five must be resolved before Phase 1.

| Decision | Choice |
|---|---|
| Tenant DB / role / user names | **`tenant_northwinds`** and **`tenant_crd_bakeoff`** (confirmed). The Go registry passes real `public.tenants.name` values; the env config maps them via slug (`tenantEnvKey`) to these targets. |
| Upgrade path | 3.3.22 → **4.1.1+** direct is supported as a minor-version upgrade. Cited from the official docs at https://docs.starrocks.io/docs/deployment/manage_deployment/upgrade/: *"From StarRocks v2.0 onwards, you can upgrade a StarRocks cluster across minor versions, for example, from v2.2.x directly to v2.5.x."* The same page recommends stepping consecutive (v2.2 → v2.3 → v2.4 → v2.5); we choose the direct 3.3 → 4.1 path here because the data volume is small and a full backup fits comfortably in the window. **Document this choice in the change-request so a future operator who hits the docs can find our reasoning.** |
| Container image pin | Pull `starrocks/fe-ubuntu:4.1.1` and `starrocks/be-ubuntu:4.1.1` first, resolve to `sha256:...` digests, and pin compose to the digest. Do NOT use floating `4.1-latest` for the upgrade window. **Verified digests 2026-10-06: FE `sha256:47982a1a16dee7aa436d4f7b2b87f37d1e2b405a520f750e10f4928505ef334d`, BE `sha256:5db07c5321eb45f33dd76ecdabe24b45fd7eef543703098de8a9ca0ec76ed0eb`. Both images are cached locally.** |
| Window | Single FE + BE means **full analytics outage during Phase 2**. Pick a window the downstream consumers can tolerate; announce it; require the warehouse bucket + Iceberg catalog to be read-only (or paused) for the duration. |
| **In-place vs greenfield** | See "Greenfield vs in-place upgrade" below — defaults to **greenfield at the current data volume**. |
| **External catalogs to recreate on greenfield** | See "External catalog recreation on greenfield" below — `lakekeeper_iceberg`, `cube_iceberg_smoke`, `pg_alpha` are not in `starrocks_init.sql`. |
| **Fate of `tenant_99e99e9999e949e989e999e99e99e999`** | See "The placeholder tenant DB" below — greenfield is the natural moment to drop it. |

**Phase 0 verification gate:** all seven rows above have a written resolution. No
work starts until this is true.

### External catalog recreation on greenfield

External catalogs are FE metadata (BDBJE under `fe_meta`), not table data.
On a **greenfield deploy** the volumes start empty, so the catalog state does
not come back from a tar restore — it has to be re-`CREATE`d from scratch.

**Verified on the live cluster 2026-10-06:** `starrocks_init.sql` declares
**one** external catalog (`iceberg_catalog` over Nessie REST), but the cluster
has **three** in production:

| Catalog | Defined in `starrocks_init.sql`? | Source |
|---|---|---|
| `iceberg_catalog` (Nessie REST) | yes | init SQL |
| `lakekeeper_iceberg` | **no** | added by hand |
| `cube_iceberg_smoke` | **no** | added by hand |
| `pg_alpha` | **no** | added by hand |

`SHOW CATALOGS` on the live cluster confirms all four (plus `default_catalog`).
The three hand-added catalogs are exactly the state the cold backup *and* the
init script both miss — `fe_meta` captures them in BDBJE, but a greenfield
volume has no BDBJE.

**Greenfield checklist for catalogs (run before declaring Phase 0 done):**

1. Run `SHOW CATALOGS` on the current 3.3 cluster and copy the list verbatim.
2. For every catalog not declared in `starrocks_init.sql`, capture the
   `CREATE EXTERNAL CATALOG` statement that produced it. The cleanest source
   is the operator who added it; the alternative is to extract from the
   live cluster's `fe_meta/image/` (manually, not part of this runbook —
   BDBJE is opaque).
3. Add the captured `CREATE` statements to `starrocks_init.sql` before the
   greenfield window — otherwise those catalogs won't exist on the fresh 4.1.1
   cluster and the audit copy path / stream-loaders / MV refreshes will all
   error with "Unknown catalog".

The in-place path does not have this concern: a cold backup of `fe_meta`
restores BDBJE including the catalog state, so the catalogs come back
intact. **Catalog recreation is a greenfield-only concern.**

### The placeholder tenant DB

The cluster carries a database called `tenant_99e99e9999e949e989e999e99e99e999`
that is clearly a placeholder (32-hex UUID format, no real-name tenant known to
match it). Verified 2026-10-06 via `SHOW DATABASES`. **Greenfield is the
natural moment to not recreate it.**

**Phase 0 checkbox (mandatory):** before any window, decide:
- (a) **Drop it** — greenfield cleanup happens for free; the RBAC template
  only creates `tenant_northwinds` and `tenant_crd_bakeoff`.
- (b) **Keep it** — it has load-bearing data or roles you didn't realize.
  In that case, give it a real name in the final convention and add a
  matching `CREATE DATABASE tenant_<realname>` to the greenfield plan.
- (c) **Audit first** — `SHOW GRANTS FOR any_user_with_grant_on_that_db` to
  see if anything is wired to it. If nothing is, drop. If something is, kill
  the wiring first, then drop.

The placeholder being live on a greenfield is a free kill. Letting it
survive the cluster rebuild is the worst of the three choices — you'd be
perpetuating the placeholder into 4.1.

### Greenfield vs in-place upgrade

This is the most consequential Phase 0 decision and the one most likely to be
made by default without thinking. Pick wrong and you spend an afternoon on a
metadata-migration that's irrelevant at your scale; pick right and the
upgrade is an hour of work with no downgrade risk.

**Heuristic threshold:** use greenfield when total data volume is below ~1 GB
**AND** you can recreate users/roles/grants from a script. Use in-place when
data is large, or non-trivial user-generated state exists that can't be
scripted.

**Why this matters at all:**
- An in-place 3.3 → 4.1 upgrade buys you metadata continuity and data locality.
  At small scale, you have almost none of either that matters — the metadata is
  one RBAC template + a handful of CREATE TABLE statements, and the data is
  reconstructable from upstream Kafka / Iceberg / batch loads.
- The 4.1 release notes are explicit: there is **no supported downgrade from
  4.x to 3.x.** Phase 4 (rollback) of an in-place upgrade is backup + restore
  + recompose at old digests, which is the same effort as a greenfield.
- An in-place upgrade has metadata-migration risk that doesn't exist on a
  greenfield cluster. The 4.x major-minor jump touches BDBJE under `fe_meta`
  in ways the docs describe but the recovery story for is "start fresh."

**Recommended default for this cluster:** **greenfield.**
- Real data volume on the live cluster: **~16 KB across `oms.compliance_evaluations` (5 rows) and empty `mdm_analytics.vendor_*` tables** (verified 2026-10-06).
- Users / roles / grants: a single SQL template (`tenant_rbac.sql.tpl`) already exists in this repo. Re-applying it on a fresh 4.1.1 cluster is exactly Phase 1.4 of this runbook with no rollback risk.
- Schema: `starrocks_init.sql` already lives in the compose mount, replayed on container first-start.
- Stream loaders: from upstream Kafka topics, re-boot from scratch is the same progress-bar after restart.
- Workload groups: created once in 4.1-native syntax from the start. **Phase 3's "recreate on 4.1" section of this runbook evaporates.**

**When to use the in-place path instead:** any of these would force the in-place upgrade:
- Data volume > ~1 GB (re-creating from upstream is no longer cheap).
- The Iceberg / lakehouse copy path has streamed-in data that isn't recreatable
  from upstream within the window.
- A workload has been running against the existing 3.3 cluster long enough
  that "we'd just rebuild from upstream" is not actually true.

**In either case, the rest of this runbook is your reference.** The greenfield
path uses Phase 2.1–2.2 as the equivalent of "stop cluster, swap image, start
cluster" — same container lifecycle. The in-place path is the version that
goes container-by-container with metadata-migration gates. Pick one and
follow the matching sub-section of Phase 2.

**Greenfield plan, condensed** (full detail lives in §2 Greenfield below):
1. Cold backup Phase 1.2 (containers stopped; tar both volumes).
2. Resolve digest-pinned images. **Do not use floating tags.** Pull
   `starrocks/fe-ubuntu:4.1.1` and `starrocks/be-ubuntu:4.1.1`, resolve to
   `sha256:...`, pin compose to the digests. (Verified 2026-10-06: FE
   `sha256:47982a1a16dee7aa436d4f7b2b87f37d1e2b405a520f750e10f4928505ef334d`,
   BE `sha256:5db07c5321eb45f33dd76ecdabe24b45fd7eef543703098de8a9ca0ec76ed0eb`.)
3. Decide the placeholder DB fate (Phase 0 checkbox above). If dropping,
   just don't recreate it; if keeping, give it a real name.
4. `docker compose down` (keep volumes!).
5. Edit `docker-compose.yml` to point at 4.1.1 digests; start the cluster.
6. FE first / BE next (same as in-place Phase 2); health gates.
7. Apply `tenant_rbac.sql.tpl` (this creates users, roles, resource groups in
   4.1-native syntax from the start — no Phase 3 rebuild).
8. **Re-create external catalogs.** `starrocks_init.sql` only declares
   `iceberg_catalog`; the live cluster has three more (`lakekeeper_iceberg`,
   `cube_iceberg_smoke`, `pg_alpha`) added by hand. Add the captured `CREATE
   EXTERNAL CATALOG` statements to `starrocks_init.sql` before the window, or
   capture them as a separate post-CREATE script that runs after
   `starrocks_init.sql`. Verify with `SHOW CATALOGS` post-restart that all
   four (plus `default_catalog`) are present.
10. Replay `starrocks_init.sql` (the schema dump from Phase 1.1) into 4.1.
11. Replay data from upstream (Kafka → stream loaders; Iceberg copy).
12. Re-point the app via `LAKEHOUSE_STARROCKS_DSN_TENANT_*` env (already in
    `tenants.env`).
13. Pin compose by digest (already done at step 2). Done.

Total time: ~1 hour, most of it verification. **No metadata-migration risk,
no no-downgrade constraint, workload groups created once.**

**In-place plan, condensed:** Phase 2 as currently written (§2.1 BE first,
§2.2 FE next, §2.3 behavior-change verification, §3 isolation verification,
§4 rollback if needed). Reach for this when the greenfield row above is
false for your cluster.

---

## Phase 1 — Pre-flight

**Goal:** produce a backup that is good enough to flip on Phase 4, capture the
current state so we can diff post-upgrade, rotate root, and apply the tenant
RBAC that the rest of the runbook assumes.

### 1.0 AuditLoader plugin compatibility gate

**Verified on the live cluster 2026-10-06: there is no third-party audit
plugin installed.** `SHOW PLUGINS` returns only `__builtin_AuditLogBuilder`
(Status: INSTALLED). The audit_loader-vs-FE-4.1 compatibility question is
moot for this cluster, so this gate reduces to a no-op verification.

If a future deployment adds `audit_loader`, run this gate for real:

```bash
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" -e '
    SHOW PLUGINS;
  '
"
# Look for: PluginName = audit_loader, Status = READY, PluginVersion.
# Cross-check the PluginVersion against the audit-loader's compatibility matrix
# for FE 4.1.x. If no published matrix exists, the gate's PASS criteria is:
#   - audit_loader Status = READY on 3.3 (live), AND
#   - the jar's source repo or release notes mention a 4.1-compatible release,
#     OR you have local evidence (test cluster on 4.1) that it loads clean.
# If the version is unknown or known-incompatible:
#   1. Disable the audit_loader plugin before Phase 2 (`UNINSTALL PLUGIN`).
#   2. Capture audit logs to a local file sink during Phase 2.
#   3. Re-enable in Phase 3 once 4.1 is up.
```

### 1.1 Capture current state

These outputs are the diff baseline for Phase 3 verification. Save them with a
timestamped filename (`starrocks-state-<ISO8601>.sql`).

```bash
# From the docker host:
TS=$(date -u +%Y%m%dT%H%M%SZ)
OUTDIR=/tmp/starrocks-preflight
mkdir -p "$OUTDIR"

ssh eganpj@100.84.50.65 "
  cd /tmp/tenant_rbac_probe
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot > ${OUTDIR}/resource-groups-${TS}.sql <<'SQL'
SHOW RESOURCE GROUPS ALL;
SQL
"

# Enumerate users one-by-one (SHOW ALL GRANTS / mysql.user are hidden on 3.3).
# Update the list as new users appear.
for USER in root stream_loader tenant_northwinds_svc tenant_crd_bakeoff_svc; do
  ssh eganpj@100.84.50.65 "
      docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot \
        > ${OUTDIR}/grants-${USER}-${TS}.sql 2>&1 <<SQL
SHOW GRANTS FOR ${USER}@'%';
SQL
    "
done

# Schema dump for every non-system database.
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -N -uroot -e \
    \"SELECT SCHEMA_NAME FROM information_schema.schemata WHERE SCHEMA_NAME NOT IN ('information_schema','mysql','sys','_statistics_');\" \
    | while read DB; do
        docker exec starrocks-fe mysqldump -h127.0.0.1 -P9030 -uroot --no-data \"\$DB\" \
          > ${OUTDIR}/schema-\${DB}-${TS}.sql 2>&1;
      done
"

# BE storage size for sanity (small data volume; should fit on local disk).
ssh eganpj@100.84.50.65 "
  docker exec starrocks-be du -sh /opt/starrocks/be/storage \
    > ${OUTDIR}/be-storage-size-${TS}.txt 2>&1
"

ls -la "$OUTDIR"
```

### 1.2 Backup

Two independent copies. Phase 4 is meaningless if either copy is bad. **Native
`BACKUP SNAPSHOT` is the primary path; the filesystem tar is the fallback.**
Two hard gates run before either:

**Gate A: size the data first.** The BE volume is provisioned at 1.6 TB but
that is provisioned capacity, not data. `du` will tell you the real size, and
that determines whether the tar fallback is hours or minutes.

**Verified on the live cluster 2026-10-06:**
- `uisce_starrocks_be_storage` volume: **21.88 MB**
- `uisce_starrocks_fe_meta` volume: **104.5 MB**
- `oms.compliance_evaluations`: 5 rows, 0.016 MB of real data
- `mdm_analytics.vendor_*`: 0 rows each
- **Total data: ~16 KB of real rows; ~125 MB on disk after metadata overhead.**

This means the backup story is radically different from a 1.6 TB cluster.
The filesystem tar is the right primary path — takes seconds, not hours.
Native `BACKUP SNAPSHOT` is overkill at this scale; reserve it if the data
grows past 100 GB.

```bash
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -D oms --table -e 'SHOW DATA;'
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -D mdm_analytics --table -e 'SHOW DATA;'
  docker system df -v | grep -E 'uisce_starrocks_'
"
```

Write the size numbers into the change-request. If the data is bigger than
expected (re-run monthly), recompute the backup-time estimate and the window.

**Gate B: the backup repository is real and writable.** An untested backup
is not a backup. Run a trivial test snapshot and restore into a scratch name
*before* you commit to the window.

**Verified on the live cluster 2026-10-06:** `SHOW REPOSITORIES` returns empty.
**No backup repository is registered.** To use the native BACKUP path you
must first create a REPOSITORY (point at MinIO under your compose) — that's
an operational decision not a runbook step. Given the 125 MB on-disk size,
**skip the BACKUP path; tar is the primary.** Document this in the change-
request. The Gate B step below is only relevant if you decide to set up a
REPO anyway.

```bash
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" <<'SQL'
SHOW REPOSITORIES;
SQL
"
# If empty: skip the BACKUP step. If you DO have a repo, run the test-restore
# loop from the previous version of this section before committing to the window.
```

After both gates pass, run the real backup:

```bash
# COLD BACKUP — required. Tar-ing `fe_meta` while the FE is running can
# produce a corrupt RocksDB artifact; the live writer holds torn writes and
# at ~100 MB you'd never notice until you needed the restore. Stop the
# cluster first, tar both volumes (seconds at this scale), start again.

ssh eganpj@100.84.50.65 "
  docker compose -f /Users/eganpj/GitHub/uisce/docker-compose.yml stop starrocks-fe starrocks-be
  # Confirm both containers are down before we touch the volumes.
  docker ps --format 'table {{.Names}}\t{{.Status}}' | grep -E 'starrocks-(fe|be)' || echo 'OK: both stopped'
"

# Tar the volumes while the cluster is stopped. The compose stacks I have
# access to use the `uisce_` prefix; adjust if yours differs.
ssh eganpj@100.84.50.65 "
  docker run --rm \
    -v uisce_starrocks_fe_meta:/src:ro \
    -v /tmp/starrocks-backup-${TS}:/dst \
    alpine:3.20 sh -c 'apk add tar && tar cf - -C /src . | tar xf - -C /dst/fe_meta'
  docker run --rm \
    -v uisce_starrocks_be_storage:/src:ro \
    -v /tmp/starrocks-backup-${TS}:/dst \
    alpine:3.20 sh -c 'apk add tar && tar cf - -C /src . | tar xf - -C /dst/be_storage'
  ls -la /tmp/starrocks-backup-${TS}/
  du -sh /tmp/starrocks-backup-${TS}/*
"

# Bring the cluster back up. Total cold-time: ~30–60 seconds.
ssh eganpj@100.84.50.65 "
  docker compose -f /Users/eganpj/GitHub/uisce/docker-compose.yml up -d
  # Wait for FE to come back up; ping SHOW FRONTENDS via the container.
  until docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -N -e 'SELECT 1' >/dev/null 2>&1; do
    sleep 5
  done
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot --table -e 'SHOW BACKENDS;'
"

# Second copy off-host (the cluster is on a single VM; lose that VM, lose
# the local backup).
rsync -a /tmp/starrocks-backup-${TS}/ backup-host:/srv/starrocks-pre-4.1/${TS}/

# The filesystem tar of fe_meta DOES capture user/grants/resource-group metadata
# because StarRocks persists them in BDBJE under fe_meta. Verify by `ls
# /tmp/starrocks-backup-${TS}/fe_meta/image/ | head` after the backup runs.
# Restore to fresh containers is the same dance in reverse: `docker volume
# create <name>` then tar with -v into the new volume, then `docker compose up`.
```

### 1.3 Rotate root

`mysql -uroot` currently works because root has no password. Phase 1 fixes that
*before* the upgrade window, so a misconfigured compose / leaked credential
during the upgrade cannot grant cluster-wide access.

```bash
# Generate the new password, store it in the secrets manager, then apply:
NEW_PW=$(openssl rand -base64 24)
# store in infisical / vault / your secrets manager under LAKEHOUSE_STARROCKS_ROOT_PASSWORD
echo "${NEW_PW}" > /tmp/root-pw.txt
chmod 600 /tmp/root-pw.txt

ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot <<SQL
ALTER USER root IDENTIFIED BY '${NEW_PW}';
SQL
"
# Verify the new password works AND the old (no-password) path is closed:
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" -e 'SELECT 1;'
"
# ^ this should succeed.
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -e 'SELECT 1;' 2>&1 | tail -3
"
# ^ this should fail with Access denied.
```

After rotation, **do not** commit the password anywhere — only the secrets
manager reference (`LAKEHOUSE_STARROCKS_ROOT_PASSWORD`) is authoritative.

### 1.4 Apply RBAC

This is the first time the wiring from the prior turn is in production. Users,
roles, grants, and resource groups all persist across the 4.1 upgrade.

```bash
# 1) Pull the 4.1 docker images first (lets us verify the pull / registry access
#    before we depend on it during Phase 2).
ssh eganpj@100.84.50.65 "
  docker pull starrocks/fe-ubuntu:4.1.1
  docker pull starrocks/be-ubuntu:4.1.1
"
# Resolve to digests NOW so Phase 2's compose change can pin by digest.
ssh eganpj@100.84.50.65 "
  FE_DIGEST=\$(docker inspect --format '{{index .RepoDigests 0}}' starrocks/fe-ubuntu:4.1.1)
  BE_DIGEST=\$(docker inspect --format '{{index .RepoDigests 0}}' starrocks/be-ubuntu:4.1.1)
  echo \"FE digest: \$FE_DIGEST\"
  echo \"BE digest: \$BE_DIGEST\"
"
# Save these digests; you'll use them in Phase 2's compose bump.

# 2) Create tenants.env with real secrets, then apply RBAC.
ssh eganpj@100.84.50.65 "cp /Users/eganpj/GitHub/uisce/backend/internal/lakehouse/{tenants.env.example,tenants.env}"
# Fill STARROCKS_ROOT_PASSWORD with /tmp/root-pw.txt, generate the tenant + loader
# passwords (openssl rand -base64 24 each), and chmod 600.
ssh eganpj@100.84.50.65 "
  chmod 600 tenants.env
  bash /Users/eganpj/GitHub/uisce/backend/internal/lakehouse/render_tenant_rbac.sh
"
# Render script refuses empty passwords (this is the guard we put in against
# recreating the empty-root situation).
```

**Phase 1 verification gate:**

| Check | Command | Pass criteria |
|---|---|---|
| Backup exists, two copies, sizes match | `ls -la /tmp/starrocks-backup-${TS}/backup-host:/srv/starrocks-pre-4.1/${TS}/` | Both directories non-empty; FE meta + BE storage both present |
| Root password rotated | `docker exec starrocks-fe mysql -uroot` | "Access denied"; `-p` with new password works |
| RBAC applied | `SHOW RESOURCE GROUPS ALL;` from the new `tenant_*_svc` user | Both `tenant_northwinds_wg` and `tenant_crd_bakeoff_wg` exist with the right classifiers |
| Per-tenant init SQL works | SSH-test in login as `tenant_northwinds_svc`, run `SET resource_group = 'tenant_northwinds_wg';` | `SHOW VARIABLES LIKE 'resource_group'` returns the name |
| 4.1 images cached locally | `docker images | grep 'starrocks/'` | Both `4.1.1` tags present |

Do not start Phase 2 until every row is green.

---

## Phase 2 — Upgrade

**Goal:** bring the cluster to `4.1.1+` (pinned by digest) with **BE first,
FE second**, and a healthy-state gate between each. The StarRocks upgrade docs
are explicit: *"By design, BEs and CNs are backward compatible with the FEs.
Therefore, you need to upgrade BEs and CNs first and then FEs."* This matters
less on single-node (both restart anyway) but we follow convention as a
precaution — a half-upgraded single-node is briefly a single-BE cluster and the
BE needs to be the version the FE expects.

### 2.0 Drain the loaders (concrete steps before any restart)

Loaders that retry against a half-upgraded BE produce noise that turns a clean
health-gate check ambiguous. Drain *before* you touch compose.

```bash
# 1) List and stop the stream-loader containers.
ssh eganpj@100.84.50.65 "
  docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}' | grep -E 'uisce-stream-loader|stream-loader' || echo 'none running'
  docker compose -f docker-compose.yml stop uisce-stream-loader || true
  docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}' | grep -E 'uisce-stream-loader|stream-loader' || echo 'OK: no stream loaders running'
"

# 2) Check in-flight work on the FE.
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" --table -e '
    SHOW LOAD FROM <db> WHERE State = \"LOADING\" LIMIT 50;
    SHOW ALL ROUTINE LOAD\\G;
  '
"
# Expect: zero rows in SHOW LOAD with State='LOADING'; for SHOW ALL ROUTINE LOAD,
# every job's Progress / LatestSourcePosition / ErrorLog should be quiescent.
# Note: 4.x changes Routine Load's offset behavior (Phase 2.3 check 3). Document
# current offset behavior in the change-request so you have a baseline to
# diff against post-upgrade.

# 3) Wait for zero in-flight work, then proceed.
```

**Phase 2.0 verification gate:** no `uisce-stream-loader` containers running,
zero `SHOW LOAD` rows in LOADING state, every Routine Load job's Progress
column unchanged for at least 60 seconds.

**Verified on the live cluster 2026-10-06:** `SHOW ALL ROUTINE LOAD` returns
zero rows. There are no Routine Load jobs on this cluster, so the
offset-discovery-behavior-change check (Phase 2.3 #3) is also vacuous. The
drain step is purely a defense-in-depth "kill the loaders" — there is
nothing in flight to lose.

### 2.1 Compose bump — BE first

```bash
# Bring down the FE first (the BE will be alone for a moment, which is fine —
# BEs can run without an FE in a development/test sense, and the FE is the component
# that needs to be restarted with the new image).
ssh eganpj@100.84.50.65 "
  docker compose -f docker-compose.yml stop starrocks-fe
"

# Bump the BE image (FE stays on 3.3 for now). Pin by digest, not by tag.
# Use the BE_DIGEST you saved in Phase 1.4 step 1.
ssh eganpj@100.84.50.65 "
  cd /Users/eganpj/GitHub/uisce
  sed -i.bak 's|image: starrocks/be-ubuntu:3.3-latest|image: starrocks/be-ubuntu@sha256:<BE_DIGEST>|' docker-compose.yml
  grep 'starrocks/be-ubuntu' docker-compose.yml
"

# Bring the BE up on 4.1, with the FE still on 3.3. The BE will start in
# 'BACKWARD_COMPATIBLE' mode relative to the FE. Watch for the success log.
ssh eganpj@100.84.50.65 "
  docker compose -f docker-compose.yml up -d starrocks-be
  docker logs -f starrocks-be 2>&1 | grep -m 1 -E 'start BE successfully|finished init|be started'
"

# Health gate (with FE still on 3.3):
#   - SHOW BACKENDS shows the BE as Alive=true.
#   - The BE is willing to talk to a 3.3 FE (this is the whole point of BE-first).
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" \
    --table -e 'SHOW BACKENDS\\G;'
"
```

**Stop here if the BE health gate fails.** Do NOT proceed to FE upgrade.

### 2.2 Compose bump — FE next

```bash
# Bump the FE image to the 4.1 digest. Bring the FE up.
ssh eganpj@100.84.50.65 "
  cd /Users/eganpj/GitHub/uisce
  sed -i 's|image: starrocks/fe-ubuntu:3.3-latest|image: starrocks/fe-ubuntu@sha256:<FE_DIGEST>|' docker-compose.yml
  grep 'starrocks/fe-ubuntu' docker-compose.yml
  docker compose -f docker-compose.yml up -d starrocks-fe
  docker logs -f starrocks-fe 2>&1 | grep -m 1 -E 'started successfully|alter cluster|is leader|finished'
"

# Health gate. The FE is "healthy" when ALL of:
#   - SHOW FRONTENDS shows the local FE as Alive=true and a role that includes LEADER
#     (or FOLLOWER if you have multiple FEs; we don't, so it's LEADER).
#   - SHOW BACKENDS shows the BE still as Alive=true (now the FE is on 4.1, BE on 4.1).
#   - A trivial query against the *system* catalog succeeds.
#   - SELECT VERSION() returns a 4.1.x string.
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" \
    --table -e \"
      SHOW FRONTENDS\\G;
      SHOW BACKENDS\\G;
      SELECT 1 AS one;
      SELECT VERSION();
    \"
"
```

**Phase 2 verification gate:**

| Check | Pass criteria |
|---|---|
| `SHOW FRONTENDS` | Local FE is `Alive=true` with `role=LEADER` |
| `SHOW BACKENDS` | Single BE is `Alive=true`, `TabletNum` matches pre-upgrade baseline |
| `SELECT VERSION()` | Returns a 4.1.x string |
| Real workloads smoke-test | `oms` and `mdm_analytics` tables visible; a `SELECT COUNT(*)` from each returns the pre-upgrade count exactly |

If the count from `oms` or `mdm_analytics` is off by even one row, **stop**. The
upgrade broke something; investigate before continuing.

### 2.3 The three behavior changes to verify concretely

These were flagged in the very first exchange. Each maps to a concrete check:

1. **VARCHAR length preservation in CTAS.** Pick a real table from `oms` or
   `mdm_analytics` that has a non-trivial VARCHAR column (>1 char). Run
   `SHOW CREATE TABLE <db>.<table>\G` and compare against the Phase 1.1
   capture. If the column length is collapsed to `VARCHAR(1)`, the
   loader-side `CREATE TABLE AS SELECT` statement needs a fix.

   ```bash
   ssh eganpj@100.84.50.65 "
     docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p... \
       oms -e 'SHOW CREATE TABLE orm_execution\\G' \
       > /tmp/oms-orm-execution-4.1.sql
   "
   diff /tmp/starrocks-preflight/schema-oms-<pre>.sql \
        /tmp/oms-orm-execution-4.1.sql
   # ^ should match modulo timestamps / replica counts
   ```

2. **Non-UTC INT64 Parquet timezone shift.** If any Parquet source feeds INT64
   timestamp columns and your loaders don't set a session `time_zone`,
   the default changed. Pick a table whose upstream is Parquet-INT64 (the
   `analytics/inventory_*` tables in `oms` are a likely candidate); compare
   `SELECT MIN(<at>), MAX(<at>) FROM <db>.<table>` against the same query
   captured pre-upgrade. If the offset shifted by your tz offset, the fix
   is `SET time_zone = 'UTC'` in the loader's session init.

3. **Routine Load offset discovery default change.** Run `SHOW ALL ROUTINE
   LOAD\G` and compare `LatestSourcePosition`, `ErrorLog`, and `Progress`
   against the pre-upgrade capture. If any Routine Load has shifted to
   "auto-offset" semantics where it was previously strict, the fix is in
   the `CREATE ROUTINE LOAD` job spec, not the cluster.

If any of these fail, the fix is in your loader/transformer code, not the
cluster. Do not patch the cluster to mask it.

**Stop here if any of the three checks fail.** Do NOT proceed to Phase 3.

---

## Phase 3 — Verify isolation on 4.1

**Goal:** confirm the tenant RBAC that Phase 1 applied still works, the Go
router's `SET resource_group = '<rg>'` init SQL still routes traffic correctly,
and (optionally) migrate to 4.1's cleaner `cpu_weight_percent`.

### 3.1 Resource groups survived the upgrade

```bash
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" \
    --table -e 'SHOW RESOURCE GROUPS ALL;'
"
# Expected: default_mv_wg, default_wg, tenant_northwinds_wg, tenant_crd_bakeoff_wg,
# with the same cpu_weight / mem_limit / concurrency_limit you set in 3.3.
# If anything is missing, see Phase 4.

# Compare against your Phase 1 capture; if SHOW RESOURCE GROUPS ALL output
# matches the pre-upgrade capture row-for-row, the upgrade preserved isolation.
diff <(grep -E '^\|' /tmp/starrocks-preflight/resource-groups-<pre>.sql) \
     <(docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p... \
        -e 'SHOW RESOURCE GROUPS ALL;' | grep -E '^\|')
```

### 3.2 Per-tenant routing still works

```bash
# Connect as tenant_northwinds_svc, run SET resource_group = 'tenant_northwinds_wg',
# and check that the query lands in the right group.
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -utenant_northwinds_svc -p\"\${NORTHWINDS_PW}\" <<'SQL'
SET resource_group = 'tenant_northwinds_wg';
SELECT 1;
SHOW VARIABLES LIKE 'resource_group';
SQL
"
# If SHOW VARIABLES returns nothing on 4.1 (the variable name may have changed to
# workload_group), the 3.3 form of the init SQL has stopped working. Verify the
# actual variable name via:
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p... -e 'SHOW VARIABLES LIKE \"%group%\";'
"
# and update the Go init SQL accordingly.

# Verify routing via the running query path, not just SHOW VARIABLES:
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p... -e \"
    SELECT resource_group, COUNT(*) FROM information_schema.task_runs
    WHERE type='QUERY' GROUP BY resource_group;
  \"
"
# ^ tenant queries should show up in their group, not default_wg.
```

### 3.3 Optional: migrate to `cpu_weight_percent`

The 3.3 `cpu_weight` value is bounded by `avg_be_cpu_cores` (12 here, so each
tenant got `cpu_weight=4` ≈ 33%). 4.1's `cpu_weight_percent` is a clean
percentage. Migrating is **optional** — leaving the 3.3 values in place still
works. To migrate:

```sql
ALTER RESOURCE GROUP tenant_northwinds_wg WITH ('cpu_weight_percent' = '33');
ALTER RESOURCE GROUP tenant_crd_bakeoff_wg WITH ('cpu_weight_percent' = '33');
-- `cpu_weight` and `cpu_weight_percent` are mutually exclusive in the same group;
-- the ALTER will replace, not add.
```

After the ALTER, re-run 3.2 to confirm routing still works.

### 3.4 Confirm the Go init SQL is left unset on 4.1

This step is already done in code: `backend/internal/lakehouse/infra/starrocks.go`
now reads a single `LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP` env var and only
appends `SET resource_group = '<rg>'` to every per-tenant connection when that
env var is set. The default — empty, no append — is what 4.1 wants.

**Verification only on this stage:**

```bash
# 1) Confirm the deploy-time env var is empty (post-restore check):
ssh eganpj@100.84.50.65 '
  grep LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP /etc/uisce/starrocks.env \
    || echo "OK: var not set"
  # OK message is the expected line; anything else means the deploy set it
  # and you have belt-and-braces init SQL pinned to one group — which is fine
  # on 3.3 (classifier still routes), but on 4.1 the SET may be rejected.
'

# 2) Confirm classification via the metrics surface. On 4.1 the per-query
# group assignment is visible in fe.audit.log's ResourceGroup column AND
# via SHOW USAGE RESOURCE GROUPS.
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p\"\$(cat /tmp/root-pw.txt)\" --table -e '
    SHOW USAGE RESOURCE GROUPS;
  '
"
# Expected after a real workload has run:
# - rows for tenant_northwinds_wg AND tenant_crd_bakeoff_wg
# - default_wg shows ONLY traffic that doesn't have a classifier match
# - NO traffic on a single shared group (which is what an always-on
# `LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP` would produce).
```

**Belt-and-braces on 3.3 is still possible.** If you want `SET resource_group`
appended for 3.3's safety net, set `LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP=
tenant_northwinds_wg` (or any group) before the upgrade. Note: this is a
SINGLE value applied uniformly to every per-tenant connection — the
classifier is what enforces real per-tenant isolation, not the SET. On 4.1
**unset** this variable; otherwise the SET may fail (the session-variable
name may have changed) and every tenant pool will die.

**Phase 3 verification gate:**

| Check | Pass criteria |
|---|---|
| `SHOW RESOURCE GROUPS ALL` post-upgrade | Identical to Phase 1 capture |
| Per-tenant routing | `SELECT 1` as `tenant_X_svc` lands in `tenant_X_wg` (per `information_schema.task_runs` or `fe.audit.log` ResourceGroup column) |
| Tenant queries served | Tenant's workload appears in `SHOW USAGE RESOURCE GROUPS` for the right group |
| `LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP` is empty in the deploy env (per 3.4) | Classifier-only routing, no init SQL failure mode on 4.1 |

---

## Phase 4 — Rollback posture (honest)

The 4.1 release notes are unambiguous: there is no supported downgrade from 4.x
back to 3.x. The 4.x-internal downgrade (4.1 → 4.0.6+) is supported but we have
no 4.0 cluster to downgrade to. **Real rollback means restore from backup and
recompose at the 3.3 digests.**

### When to trigger rollback

Any of:
- FE health gate in Phase 2 fails (FE does not come up clean).
- A 4.1 behavior-change check (Phase 2.3) breaks a workload and the fix is not
  a one-line code patch.
- A post-upgrade Phase 3 check fails in a way that can't be repaired in
  reasonable time.
- BE will not start clean after `4.1.1+` (e.g. a similar-but-undiscovered
  container startup issue).

### How to roll back

```bash
# 1) Stop both containers.
ssh eganpj@100.84.50.65 "
  docker compose -f docker-compose.yml down
"

# 2) Restore volumes from the Phase 1 backup (NOT the second copy, the first
#    one — the second is the off-host insurance).
ssh eganpj@100.84.50.65 "
  docker volume create starrocks_fe_meta
  docker volume create starrocks_be_storage
  docker run --rm \
    -v starrocks_fe_meta:/dst \
    -v /tmp/starrocks-backup-${TS}:/src:ro \
    alpine:3.20 sh -c 'apk add tar && tar cf - -C /src/fe-meta . | tar xf - -C /dst'
  docker run --rm \
    -v starrocks_be_storage:/dst \
    -v /tmp/starrocks-backup-${TS}/be:/src:ro \
    alpine:3.20 sh -c 'apk add tar && tar cf - -C /src . | tar xf - -C /dst'
"

# 3) Pin compose back to 3.3 digests.
ssh eganpj@100.84.50.65 "
  cd /Users/eganpj/GitHub/uisce
  # Revert the Phase 2.1 and 2.2 sed edits; restore from the .bak file from
  # Phase 2.1 step.
  mv docker-compose.yml.bak docker-compose.yml
  grep 'starrocks' docker-compose.yml
"
# If you did NOT make a .bak in Phase 2.1, the safer move is git revert + git
# reset on the docker-compose.yml change before re-running.

# 4) Bring the cluster back up.
ssh eganpj@100.84.50.65 "
  docker compose -f docker-compose.yml up -d
"

# 5) Verify it really is 3.3 again.
ssh eganpj@100.84.50.65 "
  docker exec starrocks-fe mysql -h127.0.0.1 -P9030 -uroot -p... -e 'SELECT VERSION();'
"
# Expected: '3.3.22-...'
```

### What the runbook does NOT promise

- **In-place downgrade from 4.x to 3.x.** Not supported by StarRocks; would
  require backup + recompose.
- **Rollback that preserves post-upgrade schema changes.** The backup is a
  pre-upgrade snapshot; rolling back means the cluster is back to *exactly* the
  pre-upgrade state, including the empty-root user that Phase 1.3 just
  hardened. After rollback, **redo Phase 1.3 + 1.4**.

If your backup is thin (Phase 1.2 was skipped or the off-host copy failed), the
runbook has no real rollback. That is not a runbook failure — it is a
prerequisite you have to meet before starting Phase 2.

---

## Phase 5 — Closeout

### 5.1 Pin the upgrade

After a 24h observation with no regressions, pin compose permanently:

```bash
# The compose is pinned by digest from Phase 2. Verify:
grep -E 'starrocks/(fe|be)-ubuntu' docker-compose.yml
# Both lines should reference sha256:... digests, NOT tags like 4.1-latest.
git diff --stat docker-compose.yml
git add docker-compose.yml
git commit -m 'pin: starrocks to 4.1.1 (sha256:fe/...) and 4.1.1 (sha256:be/...)'
git tag        # require the SHA before this commit becomes the merge candidate
```

### 5.2 Re-run the per-tenant integration probe (from the prior turn)

This is the same probe we used to verify the Go wiring on 3.3. It must still
be green on 4.1:

### 5.2a Re-run `starrocks_tenant_test.go`

The router's `initRG` opt-in design changed (was `name2rg`, now `LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP`).
The test suite covers that contract; a re-run on the new Go build catches
silent regressions introduced by the post-upgrade follow-ups (env file
format, switch order, etc.).

```bash
cd /Users/eganpj/GitHub/uisce/backend
go test -count=1 ./internal/lakehouse/infra/... \
  -run 'TestTenantRouter|TestInitResourceGroupEnv|TestScanTenantEnv|TestInitConnector'
# All green. The initRG tests are the ones that exercise the env-var contract
# end-to-end through the recording opener.
```

```bash
ssh eganpj@100.84.50.65 '
  # Apply the RBAC template (idempotent on users/roles/grants; resource groups
  # may already exist, see the re-runnability note in render_tenant_rbac.sh).
  bash /tmp/tenant_rbac_probe/probe.sh
  # Per-tenant init SQL smoke:
  bash /tmp/probe_init_sg.sh
  # Cleanup:
  docker exec -i starrocks-fe mysql -h127.0.0.1 -P9030 -uroot <<SQL
DROP RESOURCE GROUP tenant_northwinds_wg;
DROP RESOURCE GROUP tenant_crd_bakeoff_wg;
DROP USER tenant_northwinds_svc;
DROP USER tenant_crd_bakeoff_svc;
DROP USER stream_loader;
DROP ROLE tenant_northwinds_rw;
DROP ROLE tenant_crd_bakeoff_rw;
DROP ROLE loader;
DROP DATABASE tenant_northwinds;
DROP DATABASE tenant_crd_bakeoff;
SQL
'
```

### 5.3 Update the env example if anything shifted

If Phase 3.2 revealed that 4.1 renamed `resource_group` → `workload_group`, or
the Go init SQL was stripped (3.4), update `backend/internal/lakehouse/tenants.env.example`
comments to reflect the new contract.

---

## What this runbook does not cover

- **Stream-loader per-tenant wiring.** The `cmd/stream-loader/main.go` binaries
  use a single `stream_loader` user that has `INSERT` on both tenant databases.
  Wiring each loader to a per-tenant user is a separate change and a separate
  runbook if you decide it is worth doing. Today they share one user because
  each loader binary targets a specific table.
- **Production HA.** A 3-FE + 3-BE topology would let the upgrade roll instead
  of being a full outage. We are not there; the runbook assumes single-node.
- **AuditLoader compatibility matrix** is now a Phase 1 gate (section 1.0),
  not an out-of-scope note. If your plugin's compatibility matrix doesn't list
  4.1, follow the disable / file-sink / re-enable path in section 1.1.

---

## One-line summary of every phase

| Phase | Goal | Time | Risk |
|---|---|---|---|
| 0 | Resolve tenant naming, upgrade path, image pin, window | minutes | low (decisions only) |
| 1 | Backup, root rotation, RBAC apply, image cached | 30-60 min | low (cluster still on 3.3, can re-run) |
| 2 | Upgrade FE, health gate, upgrade BE, smoke test workloads | 30-90 min | **high** (outage window; rollback is expensive) |
| 3 | Verify resource groups, routing, optionally migrate, optionally strip init SQL | 30-60 min | low (cluster already upgraded; failures are diagnostic) |
| 4 | Rollback = backup restore + recompose | 60-120 min | medium (data-volume-bound) |
| 5 | Pin, re-run probe, update docs | 30 min | low |

**Total happy-path time: ~3-4 hours of cluster work + a 30-90 minute outage in Phase 2.**