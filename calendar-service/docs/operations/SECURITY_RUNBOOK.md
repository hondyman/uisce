# Security Incident Response Runbook

**Last Updated:** February 17, 2026  
**Version:** 1.0  
**Severity Levels:** CRITICAL (P0) | HIGH (P1) | MEDIUM (P2) | LOW (P3)  

---

## Incident Severity Matrix

| Severity | Description | Response Time | Escalation |
|----------|-------------|----------------|------------|
| **P0 - CRITICAL** | Data breach, auth bypass, >10% of requests failing | Immediate (5 min) | Security + Eng Lead + Compliance |
| **P1 - HIGH** | JWT compromise, tenant isolation bypass, >1% errors | 15 minutes | Security + Eng Lead |
| **P2 - MEDIUM** | Sustained rate limiting, unusual patterns, <1% errors | 1 hour | Security + Team |
| **P3 - LOW** | Single user blocked, minor config issue | 4 hours | Team |

---

## P0: CRITICAL - Suspected JWT Compromise

### Detection
```
Symptoms:
- Unexpected API calls from unknown IPs/user agents
- Audit logs show actions by unexpected user_id (e.g., admin user taking tenant-scoped actions)
- JWT validation errors spike to >50/min
- Multiple tenants reporting unauthorized access
- Tokens still valid despite rotation
```

### Immediate Actions (First 5 minutes)

**1. Alert team and escalate to Compliance**
```bash
# Send emergency notification
echo "CRITICAL: JWT Compromise Suspected - Activating response procedures" | \
  slack -c #security -f danger

# Create incident tracking
jira create -p SECURITY -t "JWT Compromise Response" -s CRITICAL
```

**2. Begin token revocation**
```bash
# Rotate JWT_SECRET immediately (new secret for issuance)
# WARNING: All existing tokens become invalid immediately

# This is phase1 - block all existing tokens
export NEW_JWT_SECRET="$(openssl rand -hex 32)"

# Update Vault immediately  
vault kv put secret/calendar-service \
  JWT_SECRET="$NEW_JWT_SECRET" \
  ROTATED_AT="$(date -Iseconds)" \
  PREVIOUS_JWT_SECRET="$OLD_SECRET"

# Trigger deployment restart
kubectl set env deployment/calendar-service \
  JWT_SECRET="$NEW_JWT_SECRET" \
  -n calendar-production

# Monitor for successful pod restart
watch kubectl get pods -n calendar-production
```

**3. Analyze compromised tokens**
```bash
# Extract compromised token from logs
kubectl logs deployment/calendar-service \
  -n calendar-production | grep "JWT validated successfully" | \
  head -1 | jq '.token' > /tmp/investigate-token

# Decode token to identify:
# - user_id of compromised user
# - tenant_ids they accessed
# - timestamp of compromise
jwt-cli --token "$(cat /tmp/investigate-token)" decode | \
  jq '.payload | {user_id, tenant_id, tenant_ids, iat, exp}'

# Record findings
COMPROMISED_USER=$(jwt-cli --token "..." | jq -r '.payload.user_id')
COMPROMISED_TENANTS=$(jwt-cli --token "..." | jq -r '.payload.tenant_ids[]')
COMPROMISE_TIME=$(jwt-cli --token "..." | jq -r '.payload.iat')
```

**4. Notify affected tenants**
```bash
# Query audit logs for impacted entities
psql -h $DB_HOST -U $DB_USER -d calendar_service << EOF
SELECT DISTINCT tenant_id, COUNT(*) as action_count
FROM audits
WHERE changed_by = '$COMPROMISED_USER'
  AND timestamp > to_timestamp($COMPROMISE_TIME)
ORDER BY action_count DESC;
EOF

# Send notifications to affected tenant admins
# Include: "Please reset all user passwords immediately"
```

### Investigation Phase (Within 1 hour)

**5. Determine compromise vector**
```bash
# Was JWT leaked in logs?
grep -r "$JWT_SECRET" /var/log/calendar-service/ 2>/dev/null

# Was JWT leaked in error responses?
kubectl logs deployment/calendar-service \
  -n calendar-production | grep "Authorization: Bearer" | head -20

# Was JWT leaked in URLs?
kubectl logs deployment/calendar-service \
  -n calendar-production | grep "\"path\":" | \
  grep -E "token|jwt|auth" | head -20

# Check for log aggregation leakage  
curl -s https://logs.example.com/query \
  -d 'query=JWT_SECRET' | jq '.
```

**6. Temporary mitigation**
```bash
# If original JWT_SECRET appears in logs, don't rotate again yet
# First, audit all logs for leakage

# Block suspicious IPs at firewall (if identified)
iptables -A INPUT -s 192.0.2.1 -j DROP  # Example - DO NOT use real IPs blindly

# Enable extra logging for next 24 hours
kubectl set env deployment/calendar-service \
  LOG_LEVEL="debug" \
  -n calendar-production
```

### Recovery Phase (Within 4 hours)

**7. Post-compromise assessment**
```bash
#Query what was accessed with compromised token
psql -h $DB_HOST -U $DB_USER -d calendar_service << EOF
-- Find all tenants affected
SELECT DISTINCT tenant_id, COUNT(*) actions_performed
FROM audits
WHERE changed_by = '$COMPROMISED_USER'
  AND timestamp > to_timestamp($COMPROMISE_TIME)
GROUP BY tenant_id;

-- Find entities modified
SELECT entity_type, entity_id, action, changed_at
FROM audits
WHERE changed_by = '$COMPROMISED_USER'
  AND timestamp > to_timestamp($COMPROMISE_TIME)
ORDER BY changed_at DESC;
EOF
```

**8. Rollback if necessary**
```bash
# If malicious changes detected, restore from backup
# Example: Calendar was deleted
kubectl exec -it calendar-service-0 -c db-restore -- \
  restore-calendar.sh --calendar-id="$DELETED_CALENDAR_ID" \
    --from-backup-time="$(date -d '1 hour ago' -Iseconds)"

# Re-apply audit trail
psql -h $DB_HOST -U $DB_USER -d calendar_service \
  -f /backup/audit-trail-1-hour-ago.sql
```

**9. Change compromised user's password + force re-auth**
```bash
# In Auth Service
auth-cli revoke-all-tokens --user-id="$COMPROMISED_USER"

# Send email to user
echo "Your account was compromised. Your password has been reset. \
  Please set a new password at https://auth.example.com/reset" | \
  mail -s "Security Alert: Account Compromised" "$COMPROMISED_USER@example.com"
```

### Documentation

**10. Post-Incident**
```bash
# Document timeline
echo "P0-2026-02-17-JWT-COMPROMISE-RUNBOOK
Time: UTC timestamps below
- 14:22:31 - Alert triggered: JWT validation errors spike
- 14:23:15 - Team escalated to Compliance
- 14:25:00 - JWT_SECRET rotated, pods restarted
- 14:27:30 - Compromised tokens identified
- 14:35:00 - Affected tenants notified
- 14:45:00 - Mitigation complete
- 15:30:00 - Root cause analysis completed
" > /var/log/security-incidents/P0-2026-02-17.log

# Schedule post-mortem
Meeting: Calendar-Service Security Post-Mortem
Time: Next business day at 10:00 AM
Attendees: Security, Eng Lead, Compliance, Tenant Success

# Record in knowledge base
wiki add incident/jwt-compromise-p0-response.md
```

---

## P1: HIGH - Tenant Isolation Bypass Attempt

### Detection
```
Symptoms:
- 403 errors spike for cross-tenant access attempts
- Logs show mismatched X-Hasura-Tenant-Id vs JWT tenant_id
- Unusual query patterns targeting other tenant IDs
- Single user attempting access to >5 different tenant calendars
```

### Response Procedures

**Immediate (< 5 minutes)**
```bash
# Verify isolation is working
kubectl logs deployment/calendar-service -n calendar-production | \
  grep "Tenant access denied" | tail -20 | jq .

# Identify attacking user
ATTACKER_IP=$(kubectl logs deployment/calendar-service \
  -n calendar-production | \
  grep "Tenant access denied" | jq -r '.client_ip' | head -1)

ATTACKER_USER=$(kubectl logs deployment/calendar-service \
  -n calendar-production | \
  grep "Tenant access denied" | jq -r '.user_id' | head -1)

echo "Attacker detected: User=$ATTACKER_USER IP=$ATTACKER_IP"

# Block at firewall (temporarily)
iptables -A INPUT -s "$ATTACKER_IP" -j REJECT

# Alert tenant of potential breach
echo "We detected an attempted unauthorized access to your account. \
  If this wasn't you, please change your password immediately." | \
  mail -s "Security Alert" "$(get-tenant-admin-email $ATTACKER_USER)"
```

**Investigation (< 1 hour)**
```bash
# Check if code was changed recently
git log --oneline --since="1 day ago" -- internal/middleware/jwt_auth.go
git diff HEAD~5..HEAD -- internal/middleware/

# Verify Hasura RLS policies are still in place
curl -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET" \
  https://hasura.example.com/v1/metadata | jq '.metadata.sources[].tables[] | select(.table.name=="calendars").permissions'

# Test isolation with admin role
curl -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET" \
  -d '{"user_id":"test-a","tenant_id":"tenant-b"}' \
  https://hasura.example.com/api/rest/calendars
# Should be BLOCKED by RLS policy
```

**Fix (if code issue found)**
```bash
# If bypass found in code
git revert HEAD~X  # Revert the breaking change

# Rebuild and test
go test -tags=integration ./tests/integration -run TestTenantIsolation -v

# Redeploy
kubectl apply -f k8s/production/calendar-service.yaml

# Verify fix
bash e2e-tests.sh --env production --focus=tenant-isolation
```

---

## P2: MEDIUM - Sustained Rate Limiting / Abuse Pattern

### Detection
```
Symptoms:
- >100 429 responses in 1 hour
- Single tenant hitting limits consistently
- Legitimate users unable to access service
- Traffic pattern suggests bot or brute-force attempt
```

### Response

**Analyze pattern**
```bash
# Get rate limit metrics
kubectl logs deployment/calendar-service \
  -n calendar-production | grep "rate_limit_exceeded" | \
  jq '{tenant_id, user_id, path, timestamp}' | tail -50 | \
  jq -s 'group_by(.tenant_id) | map({tenant: .[0].tenant_id, count: length})'

# Identify the abusive tenant
ABUSIVE_TENANT=$(kubectl logs deployment/calendar-service \
  -n calendar-production | grep "rate_limit_exceeded" | \
  jq -r '.tenant_id' | sort | uniq -c | sort -rn | head -1 | awk '{print $2}')

echo "Abusive tenant: $ABUSIVE_TENANT"
```

**Actions**
```bash
# Increase limit for known good tenants if needed
kubectl set env deployment/calendar-service \
  RATE_LIMIT_TENANT_EXCEPTIONS="$ABUSIVE_TENANT=1,other-tenant=5" \
  -n calendar-production

# OR: Block temporarily pending investigation  
kubectl patch deployment calendar-service \
  -p '{"spec": {"template": {"spec": {"nodeSelector": {"abusive-tenant-tenant-to-skip":"true"}}}}}' \
  -n calendar-production

# Notify tenant
echo "We've detected unusual traffic patterns on your account that triggered \
  rate limits. Contact support@example.com to investigate." | \
  mail -s "Rate Limit Alert" "admin@$ABUSIVE_TENANT.example.com"
```

---

## P3: LOW - Single User Issue / Configuration Problem

### Detection & Response
```bash
# User reports "too many requests" error

# Check if they're hitting the rate limit
kubectl logs deployment/calendar-service \
  -n calendar-production | grep "$USER_ID" | grep "rate_limit"

# Verify they're configured correctly
psql -h $DB_HOST -U calendar_user -d calendar_service \
  -c "SELECT * FROM tenants WHERE id='$TENANT_ID';"

# If limit is too strict, consider:
# 1. Scale up deployment
# 2. Increase rate limit
# 3. Confirm it's not actual abuse
```

---

## Monitoring & Alert Rules

### Prometheus Alerts

```yaml
groups:
  - name: calendar-service-security
    interval: 30s
    rules:
      # P0: Excessive JWT failures  
      - alert: JWTValidationErrors
        expr: rate(jwt_validation_errors_total[5m]) > 50
        for: 2m
        annotations:
          severity: critical
          description: "{{ $value }} JWT errors/sec in prod"
          
      # P1: Potential tenant isolation attack
      - alert: TenantAccessDeniedSpike
        expr: rate(tenant_access_denied_total[5m]) > 10
        for: 5m
        annotations:
          severity: high
          description: "{{ $value }} cross-tenant denials/sec"
          
      # P2: Rate limiting abuse
      - alert: RateLimitAbuse
        expr: rate(rate_limit_exceeded_total[5m]) > 100
        for: 10m
        annotations:
          severity: medium
          description: "{{ $value }} 429s/sec from {{ $labels.tenant_id }}"
```

### Log Patterns to Watch

```bash
# Set up continuous monitoring
kubectl logs -f deployment/calendar-service -n calendar-production | \
  grepstream -E "(error|failed|denied|exceeded|invalid)" | \
  alert-on-pattern --threshold=50 --window=5m
```

---

## Escalation Path
```
User reports issue
    ↓
On-Call Engineer investigates (< 15 min)
    ↓ (if P2+)
Notify #security channel with details
    ↓ (if P1+)
Page Security Engineer via PagerDuty
    ↓ (if P0)
Trigger full incident response
    ↓ (if security team says so)
Notify Compliance Officer
    ↓ (if data breach suspected)
Legal team + Customer notifications
```

---

## Key Contact Information

**Security Team**: security@example.com | Slack: #security  
**On-Call Engineer**: PagerDuty calendar  
**Compliance Officer**: compliance@example.com  
**Platform Lead**: platform-lead@example.com  

**Emergency Hotline**: +1-XXX-XXX-XXXX  
**Incident Channel**: #security-incidents Slack  

---

**Last Updated**: February 17, 2026  
**Next Review**: Quarterly (May 17, 2026)
