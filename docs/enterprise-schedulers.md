# Driving Uisce schedules from an enterprise scheduler

Tidal, Control-M, AutoSys, Stonebranch and similar tools can start Uisce
schedules. Every run still goes through the one scheduler: business
calendars, run history, audit trail and translated error messages.

## 1. Make the schedule externally triggered

In **Build › Automation › Schedules**, edit the schedule and choose
**How it runs › Triggered by an external scheduler**. It then has no timetable
of its own. Its business calendar can still refuse a closed day ("skip"); it
never defers a run, because the caller is waiting for an answer.

Only externally triggered schedules accept triggers (a timetable schedule
answers 9200-18), so a job can't be fired twice by two schedulers.

## 2. Give the scheduler a service account (once per tenant)

A platform administrator runs:

```bash
infrastructure/keycloak/create-scheduler-client.sh \
  --tenant-id <tenant uuid> --client-id tidal-<tenant> \
  --secret-out ~/.uisce/tidal-<tenant>.secret      # or --infisical-path /schedulers
```

It creates a confidential Keycloak client that can only use the
client-credentials grant, fixes the tenant in its tokens, and grants the
realm roles `uisce_service_account` (read and trigger only - it can never
create, change, pause or delete schedules) and `schedule_trigger`. The secret
is written to a 0600 file or Infisical; it is never printed. Use
`--dry-run` first.

## 3. Run it from the scheduler

### Command job (any scheduler with agents)

Put the `uisce-job` binary (`go build ./cmd/uisce-job`) on the agent and
set, in the job's environment:

| Variable | Value |
|---|---|
| `UISCE_URL` | e.g. `https://uisce.example.com` |
| `UISCE_TOKEN_URL` | `https://<keycloak>/realms/uisce/protocol/openid-connect/token` |
| `UISCE_CLIENT_ID` | `tidal-<tenant>` |
| `UISCE_CLIENT_SECRET_FILE` | path to the secret file, readable only by the agent user |
| `UISCE_REGION` | the tenant's region, e.g. `us-west` |

```bash
uisce-job run --schedule "Order volume - London close" \
  --key "$TIDAL_JOB_RUN_ID" --system tidal --ref "$TIDAL_JOB_NAME" --wait
```

`--key` makes the trigger idempotent: use the scheduler's own run id, so a
retried or restarted job returns the first run instead of starting a second.
`--schedule` takes the schedule id or its exact name.

| Exit code | Meaning | Typical scheduler action |
|---|---|---|
| 0 | succeeded (or, without `--wait`, accepted) | complete |
| 1 | the run failed | fail the job; the catalog message is on stderr |
| 2 | skipped (calendar closed) | treat as complete-no-op, or branch |
| 3 | timed out waiting (`--timeout`, default 12h) | alert; re-running with the same key only waits again |
| 4 | not authorized | fix the client or its roles |
| 5 | bad usage or configuration | fix the job definition |
| 6 | other API error (unknown schedule, paused, ...) | fail the job |

stdout is one line: `schedule=... key=... status=... run_id=... summary="..."`.

### REST (schedulers with a web-service job type)

```
POST /api/schedules/{id}/trigger
Authorization: Bearer <client-credentials token>
X-Tenant-Region: us-west
{"idempotency_key": "<run id>", "system": "tidal", "ref": "<job name>"}

-> 202 {"schedule_id": "...", "idempotency_key": "...", "status": "queued"}
   (a repeat with the same key returns where that trigger stands)

GET /api/schedules/{id}/triggers/{key}?wait=60s
-> 200 {"status": "queued|running|succeeded|failed|skipped", "run": {...}}
```

`wait` long-polls up to 2 minutes per call; poll again for longer. Errors are
catalog messages (`error`, `error_code`, `user_action`), translated per
`Accept-Language`.

## What the run history shows

Externally triggered runs carry the calling system and job reference (e.g.
"tidal · EOD_ORDERS"), and the idempotency key, next to scheduled and manual
runs. The trigger is audited as the service account.
