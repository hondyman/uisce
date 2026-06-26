# Runbook: Google Calendar Sync Job Failed

## Alert
- **Name**: GoogleSyncJobFailed
- **Severity**: Warning
- **Threshold**: >0.1 failed syncs per 5 minutes

## Symptoms
- Users report events not syncing from Google Calendar
- Sync jobs showing "failed" status in dashboard
- Increased error rate in Google API calls

## Diagnosis

### 1. Check Sync Job Logs
```bash
kubectl logs -l app=calendar-service -n calendar --grep="sync" --since=1h
```

### 2. Check Google API Status
```bash
curl -s http://localhost:8081/metrics | grep google_calendar_api_errors
```

### 3. Check OAuth Token Status
```bash
curl -s http://localhost:8081/metrics | grep oauth_token_errors
```

### 4. Check Redis Connectivity
```bash
redis-cli ping
kubectl get pods -l app=redis -n calendar
```

## Resolution

### OAuth Token Issues
1. Check if tokens are expired:
   ```bash
   curl -s http://localhost:8081/api/v1/sync/google/token/info?user_id=<USER_ID>
   ```
2. If expired, notify user to re-authenticate
3. Check token refresh logic in logs

### Google API Rate Limit
1. Check rate limit metrics:
   ```bash
   curl -s http://localhost:8081/metrics | grep rate_limit
   ```
2. Reduce sync frequency temporarily
3. Wait for rate limit reset (usually 1 hour)

### Network Issues
1. Check connectivity to Google APIs:
   ```bash
   curl -I https://www.googleapis.com/calendar/v3
   ```
2. Check DNS resolution
3. Check firewall rules

### Database Issues
1. Check database connectivity:
   ```bash
   psql $DATABASE_URL -c "SELECT 1"
   ```
2. Check for locked tables
3. Check disk space

## Verification
1. Trigger manual sync:
   ```bash
   curl -X POST http://localhost:8081/api/v1/sync/google/sync \
     -H "Content-Type: application/json" \
     -d '{"user_id":"<USER_ID>","google_calendar_id":"primary"}'
   ```
2. Monitor sync status:
   ```bash
   curl http://localhost:8081/api/v1/sync/status/<SYNC_ID>
   ```
3. Verify events synced:
   ```bash
   curl http://localhost:8081/api/v1/sync/google/events?tenant_id=<TENANT_ID>
   ```

## Escalation
- If unresolved after 30 minutes: Escalate to Platform Team
- If OAuth issues persist: Contact Google Cloud support
- If database issues: Escalate to Database Team

## Prevention
- Monitor sync success rate dashboard
- Set up alerts for OAuth token expiry
- Implement exponential backoff for API calls
- Regular token rotation before expiry
