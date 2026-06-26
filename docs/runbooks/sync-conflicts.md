# Runbook: High Number of Sync Conflicts

## Alert
- **Name**: SyncConflictsBacklog
- **Severity**: Warning
- **Threshold**: >50 pending conflicts for 30 minutes

## Symptoms
- Users report events not syncing correctly
- Duplicate events appearing
- Missing events after sync

## Diagnosis

### 1. Check Conflict Types
```bash
curl -s http://localhost:8081/api/v1/sync/conflicts/stats?tenant_id=<TENANT_ID> | jq .
```

### 2. Review Pending Conflicts
```bash
curl -s http://localhost:8081/api/v1/sync/conflicts?tenant_id=<TENANT_ID>&status=pending | jq .
```

### 3. Check Recent Sync Jobs
```bash
curl -s http://localhost:8081/api/v1/sync/active?user_id=<USER_ID> | jq .
```

## Resolution

### Auto-Resolve Low-Severity Conflicts
```bash
curl -X POST http://localhost:8081/api/v1/sync/conflicts/auto-resolve \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"<TENANT_ID>","severity":["info","warning"]}'
```

### Manual Review for Critical Conflicts
1. Review each critical conflict in dashboard
2. Apply appropriate resolution strategy:
   - `keep_google`: Use Google Calendar version
   - `keep_internal`: Use internal version
   - `merge`: Combine both versions
   - `skip`: Don't sync this event

### Bulk Resolution (if appropriate)
```bash
curl -X POST http://localhost:8081/api/v1/sync/conflicts/bulk-resolve \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id":"<TENANT_ID>",
    "conflict_type":"time_overlap",
    "strategy":"keep_google"
  }'
```

## Verification
1. Check conflict count reduced:
   ```bash
   curl -s http://localhost:8081/api/v1/sync/conflicts/stats?tenant_id=<TENANT_ID> | jq '.pending'
   ```
2. Verify events synced correctly:
   ```bash
   curl http://localhost:8081/api/v1/sync/google/events?tenant_id=<TENANT_ID>
   ```

## Prevention
- Improve conflict detection rules
- Increase auto-resolution threshold
- Better user education on sync behavior
- Regular conflict review meetings
