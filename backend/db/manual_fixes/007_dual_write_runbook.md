# 007_dual_write_runbook.md
# Dual-write window + alpha→crims cutover

**Status:** READY FOR EXECUTION (pending team roster fill)
**Owner:** [TEAM_FILL]
**On-call:** [TEAM_FILL]
**Window start:** 2026-09-29 10:00 local
**Duration:** 48h
**Blocker:** DEBEZIUM_RETARGET (executes step 6.1–6.3)

---

## 0. Team roster — fill before signing off

| Role | Name | Phone |
|---|---|---|
| Window owner | [TEAM_FILL] | [TEAM_FILL] |
| On-call engineer | [TEAM_FILL] | [TEAM_FILL] |
| App owner | [TEAM_FILL] | [TEAM_FILL] |
| DBA | [TEAM_FILL] | [TEAM_FILL] |
| StarRocks owner | [TEAM_FILL] | [TEAM_FILL] |
| Escalation (CTO) | [TEAM_FILL] | [TEAM_FILL] |

Once filled, delete this section. Everything below is executable as written.

---

## 1. Locked decisions

| Parameter | Value |
|---|---|
| Window duration | 48h |
| Window start | 2026-09-29 10:00 local (Tuesday) |
| Write mode during window | dual (alpha + crims) |
| Read mode during window | crims primary |
| Change freeze | app + schema + migrations (Mon 09:00 → Fri 18:00) |
| Rollback SLA | < 15 min |
| Rollback option | A — truncate crims rows written during window |
| Snapshot mode on Debezium flip | never |
| Slot (crims) | crims_oms_slot |
| Publication (crims) | orm_cdc_publication (5 tables, pre-staged) |
| Topic prefix | orm_oms (unchanged) |
| Drop timing | 24h soak after crims_only (Thu → Fri) |
| Parity oracle | check_drift.sh (proven detector) |
| Sequence | Revised: Debezium flip AFTER dual-write (rollback-clean) |

---

## 2. Why this sequence (revision from original)

Original sequence flipped Debezium BEFORE dual-write. That leaves a
downstream gap: alpha connector off, crims not yet writing, writes during
the gap never reach Redpanda/Kafka.

Revised order keeps alpha Debezium live through the whole window, flips
only after parity is verified, then stops alpha writes. Rollback is clean
at every step.

---

## 3. Pre-window checklist (T-24h — Monday 2026-09-28)

- [ ] `copy_plan.sh` completed all domains; `check_drift.sh` returns exit 0
- [ ] `check_drift.sh` proven as detector (baseline 0 → inject → delta ≠ 0 → cleanup → 0)
- [ ] Open blockers triaged; only `DEBEZIUM_RETARGET` + this runbook remain
- [ ] `006_debezium_retarget.md` verified: snapshot.mode=never, decimal=double
- [ ] crims publication has 5 tables:
  ```sql
  SELECT count(*) FROM pg_publication_tables WHERE pubname='orm_cdc_publication';
  -- expect: 5
  ```
- [ ] `wal_level=logical` on both DBs
- [ ] `crims_oms_slot` does not exist yet:
  ```sql
  SELECT slot_name FROM pg_replication_slots WHERE slot_name='crims_oms_slot';
  -- expect: 0 rows
  ```
- [ ] App dual-write code path deployed and tested in staging
- [ ] Downstream consumers identified: [TEAM_FILL service list]
- [ ] Change freeze in effect
- [ ] On-call briefed; dry-run rollback executed in staging (< 15 min)
- [ ] Baseline captured: alpha p99 write latency, throughput, error rate

---

## 4. Window start (Tuesday 2026-09-29)

### 09:45 — Team sync (15 min)

- Confirm all present, comms channel open
- Confirm rollback authority identified
- Read §5 aloud once

### 10:00 — Step 4.1 App READ-ONLY

```bash
kubectl set env deployment/uisce-app WRITE_MODE=read_only
```

Verify no alpha writes:
```sql
SELECT count(*) FROM pg_stat_activity
WHERE datname='alpha' AND state='active' AND query ILIKE 'INSERT%';
-- expect: 0 or near 0
```

### 10:05 — Step 4.2 Parity check #1

```bash
/Users/eganpj/GitHub/uisce/backend/db/manual_fixes/check_drift.sh
# expect: exit 0
```

**Non-zero → ABORT. Revert `WRITE_MODE=normal`. Investigate. Reschedule.**

### 10:10 — Step 4.3 Dual-write ON

```bash
kubectl set env deployment/uisce-app WRITE_MODE=dual_write
```

Verify both DBs seeing writes within 60s:
```sql
SELECT datname, count(*) FROM pg_stat_activity
WHERE datname IN ('alpha','crims') AND state='active' AND query ILIKE 'INSERT%'
GROUP BY datname;
-- expect: both non-zero
```

### 10:15 — Step 4.4 Confirm alpha Debezium untouched

```bash
curl -s http://localhost:8083/connectors/orm-oms-connector/status | jq '.tasks[].state'
# expect: RUNNING (unchanged)
```

### 10:30 — Window armed

Hand to on-call. Monitoring §6 runs every 15 min.

---

## 5. Rollback procedure

Execute if any §6.6 trigger fires.

### 5.1 Immediate

```bash
kubectl set env deployment/uisce-app WRITE_MODE=alpha_only
```

Verify app health within 2 min.

### 5.2 Verify alpha stream

```bash
curl -s http://localhost:8083/connectors/orm-oms-connector/status | jq '.tasks[].state'
# expect: RUNNING — alpha connector never stopped in this sequence
```

### 5.3 Crims cleanup (Option A)

```sql
-- Count rows written during window
SELECT count(*) FROM crims.mdm.<table>
WHERE created_at > '[window_start]';

-- Truncate those rows; crims returns to pre-window state
-- Then retry dual-write after fix
```

### 5.4 Postmortem within 24h

---

## 6. Continuous monitoring (T+10min → T+48h)

Run every 15 min via cron.

### 6.1 Parity drift

```bash
check_drift.sh
# alert if exit != 0
```

### 6.2 Write latency (per DB)

```sql
SELECT datname,
       percentile_cont(0.99) WITHIN GROUP (ORDER BY total_exec_time) AS p99_ms
FROM pg_stat_statements s JOIN pg_database d ON d.oid = s.dbid
WHERE query ILIKE 'INSERT%' AND datname IN ('alpha','crims') AND calls > 100
GROUP BY datname;
-- alert if crims p99 > 2× alpha p99 for 15 consecutive min
```

### 6.3 Error rate

App metric `db.write.errors.rate`. Alert if > 0.1%.

### 6.4 Debezium task health

```bash
curl -s http://localhost:8083/connectors/orm-oms-connector/status | jq '.tasks[].state'
# alert if any task not RUNNING for > 5 min
```

### 6.5 Downstream lag

StarRocks load lag. Alert if > 5 min behind.

### 6.6 Rollback triggers (any one → §5)

| Trigger | Threshold | Sustained |
|---|---|---|
| crims p99 write latency | > 2× alpha baseline | 15 min |
| App write error rate | > 0.1% | 5 min |
| Parity drift | > 0 rows | 2 consecutive checks |
| Debezium task failure | not RUNNING | 5 min, unrecoverable |
| Downstream lag | > 30 min | 10 min |

---

## 7. Verification points

### 7.1 T+24h (Wednesday 2026-09-30 10:00)

- [ ] Drift = 0 throughout
- [ ] No rollback triggers fired
- [ ] Downstream lag within SLA
- [ ] Baseline metrics stable

If clean: continue. If not: §5.

### 7.2 T+48h (Thursday 2026-10-01 10:00)

- [ ] Drift = 0
- [ ] All metrics stable last 24h
- [ ] App logs confirm dual-write active on all write paths
- [ ] Sign-off from Window owner

Proceed to §8.

---

## 8. Cutover (Thursday 2026-10-01)

### 10:00 — Final verification (§7.2 checklist)

### 10:15 — Step 8.1 Stop alpha connector

```bash
curl -s -X DELETE http://localhost:8083/connectors/orm-oms-connector
```

### 10:16 — Step 8.2 Register crims connector

Per `006_debezium_retarget.md`:

```bash
curl -s -X POST http://localhost:8083/connectors \
  -H 'Content-Type: application/json' \
  -d @/Users/eganpj/GitHub/uisce/backend/db/manual_fixes/006_connector.json
```

Verify:
```bash
curl -s http://localhost:8083/connectors/orm-oms-connector/status | jq '.connector.state, .tasks[].state'
# expect: RUNNING, RUNNING
```

### 10:20 — Step 8.3 Smoke test

```sql
INSERT INTO orm.execution (id, ...) VALUES (gen_random_uuid(), ...);
```

Confirm StarRocks row in < 30s.

**Not visible → rollback: re-register alpha connector with slot `orm_oms_slot` (still exists on alpha).**

### 10:30 — Step 8.4 Stop alpha writes

```bash
kubectl set env deployment/uisce-app WRITE_MODE=crims_only
```

### 10:45 — Step 8.5 Final parity

```bash
check_drift.sh
# expect: exit 0
```

### 11:00 — Soak begins

**Do not drop alpha yet.** Soak 24h. Alpha stays read-only.

---

## 9. Drop (Friday 2026-10-02 10:00)

Only if:
- 24h of `crims_only` with zero errors
- `check_drift.sh` returns 0
- No incidents overnight

```bash
# Final safety dump
pg_dump -h 100.84.50.65 -U postgres -d alpha -n mdm --schema-only \
  > /Users/eganpj/GitHub/uisce/backend/db/manual_fixes/alpha_mdm_final_$(date +%Y%m%d).dump.sql

# Confirm no app queries on alpha
psql "$ALPHA_URL" -Atc "SELECT count(*) FROM pg_stat_activity
                       WHERE datname='alpha' AND application_name != 'psql';"
# expect: 0

# Drop
psql "$ALPHA_URL" -c "DROP SCHEMA mdm CASCADE;"
psql "$ALPHA_URL" -c "DROP SCHEMA edm CASCADE;"
# DO NOT drop alpha.public — fabric + catalog + OKF + tenants
```

### Announcement

Stakeholders notified. Freeze lifted Friday 18:00.

---

## 10. Cleanup (Tuesday 2026-10-06)

```bash
# FDW scaffolds (never used, but drop if any)
psql "$CRIMS_URL" -c "DROP SERVER IF EXISTS alpha_srv CASCADE;"
psql "$CRIMS_URL" -c "DROP SCHEMA IF EXISTS alpha_staging CASCADE;"

# Archive migration plan
pg_dump -h 100.84.50.65 -U postgres -d crims -n migration \
  > /Users/eganpj/GitHub/uisce/backend/db/manual_fixes/migration_archive_$(date +%Y%m%d).sql

psql "$CRIMS_URL" -c "DROP SCHEMA migration CASCADE;"

# Relax monitoring thresholds
# Archive blockers
# File postmortem
```

---

## 11. Sign-off

| Role | Name | Date |
|---|---|---|
| Migration owner | | |
| Window owner | | |
| On-call engineer | | |
| App owner | | |
| DBA | | |
| Downstream owner | | |
