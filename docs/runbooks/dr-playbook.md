# Disaster Recovery Playbook

> **Semlayer Platform DR Procedures** - Last updated: 2025-01

## 📋 Table of Contents

1. [Overview](#overview)
2. [Recovery Objectives](#recovery-objectives)
3. [Incident Classification](#incident-classification)
4. [Communication Protocol](#communication-protocol)
5. [Recovery Procedures](#recovery-procedures)
6. [Quarterly Drill Schedule](#quarterly-drill-schedule)
7. [Post-Incident Review](#post-incident-review)

---

## Overview

This playbook documents disaster recovery procedures for the Semlayer platform, covering:
- Cube.js semantic layer
- StarRocks analytics database
- Tenant management services
- Pre-aggregation pipelines
- API gateway and authentication

### Key Contacts

| Role | Name | Contact |
|------|------|---------|
| Incident Commander | TBD | @incident-commander |
| Platform Lead | TBD | @platform-lead |
| Database Lead | TBD | @database-lead |
| Security Lead | TBD | @security-lead |

---

## Recovery Objectives

### Service Tier Definitions

| Tier | Services | RTO | RPO |
|------|----------|-----|-----|
| **P0** | Auth, API Gateway | 15 min | 0 (no data loss) |
| **P1** | Cube.js, Query Engine | 30 min | 5 min |
| **P2** | Pre-aggregations, Reports | 2 hours | 1 hour |
| **P3** | Analytics, Monitoring | 4 hours | 4 hours |

### Key Metrics

- **RTO (Recovery Time Objective)**: Maximum acceptable downtime
- **RPO (Recovery Point Objective)**: Maximum acceptable data loss

---

## Incident Classification

### Severity Levels

| Severity | Definition | Response Time | Example |
|----------|------------|---------------|---------|
| **SEV-1** | Complete service outage | Immediate | All tenants unable to query |
| **SEV-2** | Major degradation | 15 min | Pre-aggs failing, >50% latency increase |
| **SEV-3** | Partial degradation | 1 hour | Single tenant affected |
| **SEV-4** | Minor issue | 4 hours | Non-critical feature unavailable |

### Escalation Matrix

```
SEV-1: On-call → Platform Lead → VP Engineering → CTO (within 30 min)
SEV-2: On-call → Platform Lead (within 1 hour)
SEV-3: On-call → Team Lead (within 4 hours)
SEV-4: Normal ticket workflow
```

---

## Communication Protocol

### Internal Communication

1. **Incident Channel**: Create `#incident-YYYY-MM-DD-brief-description`
2. **Status Updates**: Every 15 min for SEV-1/2, every 30 min for SEV-3
3. **Bridge Call**: Zoom link in incident channel for SEV-1

### External Communication

1. **Status Page**: Update status.semlayer.io
2. **Customer Notification**: Affected tenants via email for SEV-1/2
3. **Template Messages**:

```markdown
## Investigating
We are investigating reports of [service] issues. Updates to follow.

## Identified  
We have identified the cause of [service] issues and are implementing a fix.

## Resolved
[Service] issues have been resolved. All services operating normally.
```

---

## Recovery Procedures

### 1. StarRocks Database Failure

**Symptoms:**
- Query timeouts
- "Connection refused" errors
- High latency across all tenants

**Immediate Actions:**

```bash
# 1. Check cluster health
kubectl get pods -n starrocks
kubectl logs -n starrocks starrocks-fe-0 --tail=100

# 2. Check disk space
kubectl exec -n starrocks starrocks-be-0 -- df -h

# 3. Verify network connectivity
kubectl exec -n starrocks starrocks-fe-0 -- curl -s localhost:8030/api/health
```

**Recovery Steps:**

```bash
# Option A: Rolling restart (preferred)
kubectl rollout restart statefulset/starrocks-fe -n starrocks
kubectl rollout restart statefulset/starrocks-be -n starrocks

# Option B: Failover to replica
# Update cube.js connection string to replica
kubectl set env deployment/cube-api CUBE_DB_HOST=starrocks-replica.internal

# Option C: Restore from backup (last resort)
./scripts/restore-starrocks.sh --backup-id=<latest>
```

**Validation:**

```bash
# Verify cluster health
curl -s http://starrocks-fe:8030/api/health | jq .
# Expected: {"status":"OK"}

# Run test query
mysql -h starrocks-fe -P 9030 -u root -e "SHOW BACKENDS;"
```

---

### 2. Cube.js Service Failure

**Symptoms:**
- 502/503 errors on `/cubejs-api/*` endpoints
- Scheduled refresh jobs failing
- Pre-aggregation build failures

**Immediate Actions:**

```bash
# 1. Check pod status
kubectl get pods -n cube -l app=cube-api

# 2. Check logs for errors
kubectl logs -n cube deployment/cube-api --tail=200 | grep -i error

# 3. Check Redis connectivity
kubectl exec -n cube deployment/cube-api -- redis-cli -h redis ping
```

**Recovery Steps:**

```bash
# Option A: Restart pods
kubectl rollout restart deployment/cube-api -n cube

# Option B: Scale up if resource exhaustion
kubectl scale deployment/cube-api --replicas=5 -n cube

# Option C: Clear Redis cache
kubectl exec -n cube deployment/cube-api -- redis-cli -h redis FLUSHDB

# Option D: Rebuild pre-aggregations
curl -X POST http://cube-api:4000/cubejs-system/v1/pre-aggregations/build \
  -H "Authorization: Bearer $CUBE_API_SECRET"
```

**Validation:**

```bash
# Health check
curl -s http://cube-api:4000/readiness | jq .

# Test query
curl -s http://cube-api:4000/cubejs-api/v1/meta \
  -H "Authorization: $CUBE_TOKEN" | jq '.cubes | length'
```

---

### 3. Tenant Data Corruption

**Symptoms:**
- Incorrect query results for specific tenant
- Schema validation failures
- Pre-aggregation data mismatch

**Immediate Actions:**

```bash
# 1. Identify affected tenant
grep "tenant_id" /var/log/cube/errors.log | sort | uniq -c | sort -rn | head

# 2. Disable tenant temporarily
curl -X POST http://admin-api/api/tenants/<tenant_id>/disable \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 3. Capture current state
pg_dump -t tenant_configs -t tenant_schemas --data-only > tenant_backup.sql
```

**Recovery Steps:**

```bash
# Option A: Rebuild pre-aggregations for tenant
curl -X POST "http://cube-api:4000/cubejs-system/v1/pre-aggregations/build" \
  -H "Authorization: Bearer $CUBE_API_SECRET" \
  -d '{"securityContext": {"tenant_id": "<tenant_id>"}}'

# Option B: Restore tenant from backup
./scripts/restore-tenant.sh --tenant-id=<tenant_id> --backup-date=<date>

# Option C: Re-sync from source
./scripts/sync-tenant-data.sh --tenant-id=<tenant_id> --full-refresh
```

**Validation:**

```bash
# Verify data integrity
./scripts/validate-tenant.sh --tenant-id=<tenant_id>

# Compare with source system
./scripts/compare-tenant-data.sh --tenant-id=<tenant_id>
```

---

### 4. Complete Platform Outage

**Symptoms:**
- All services unreachable
- Multiple component failures
- Infrastructure-level issue (cloud provider, network)

**Immediate Actions:**

1. **Assess scope**: Cloud provider status page, network monitoring
2. **Activate incident bridge**: Conference call with all leads
3. **Notify stakeholders**: Status page update, customer communication

**Recovery Priority Order:**

```
1. Network/Infrastructure (if applicable)
2. PostgreSQL (auth, tenant config)
3. Redis (cache, sessions)
4. StarRocks (analytics data)
5. API Gateway
6. Cube.js services
7. Pre-aggregation workers
8. Monitoring/Alerting
```

**Full Recovery Procedure:**

```bash
# 1. Verify infrastructure
./scripts/check-infrastructure.sh

# 2. Start core databases
kubectl apply -f k8s/postgres/
kubectl apply -f k8s/redis/

# 3. Wait for DB readiness
./scripts/wait-for-postgres.sh
./scripts/wait-for-redis.sh

# 4. Start StarRocks
kubectl apply -f k8s/starrocks/
./scripts/wait-for-starrocks.sh

# 5. Start application services
kubectl apply -f k8s/api-gateway/
kubectl apply -f k8s/cube/

# 6. Verify services
./scripts/smoke-test.sh --all

# 7. Re-enable traffic
kubectl patch svc api-gateway -p '{"spec":{"selector":{"active":"true"}}}'
```

---

### 5. Security Incident (Data Breach)

**Symptoms:**
- Unauthorized access alerts
- Unusual query patterns
- Data exfiltration indicators

**Immediate Actions:**

```bash
# 1. Isolate affected components
kubectl cordon <affected-nodes>
kubectl delete pods -l tenant-id=<affected-tenant> --force

# 2. Revoke credentials
./scripts/rotate-credentials.sh --emergency

# 3. Capture forensic data
kubectl logs -n cube --all-containers --timestamps > forensics/cube-logs.txt
pg_dump audit_logs > forensics/audit-logs.sql
```

**DO NOT:**
- Delete logs before capturing
- Restart services before forensic capture
- Communicate details externally without legal approval

**Escalation:**
- Immediately notify Security Lead
- Legal team within 1 hour
- Consider external forensics engagement

---

## Quarterly Drill Schedule

### 2025 Schedule

| Quarter | Date | Scenario | Lead |
|---------|------|----------|------|
| Q1 | Jan 15 | StarRocks failover | Database Lead |
| Q1 | Feb 12 | Cube.js service recovery | Platform Lead |
| Q2 | Apr 16 | Tenant data restoration | Platform Lead |
| Q2 | May 14 | Full platform recovery | VP Engineering |
| Q3 | Jul 16 | Security incident response | Security Lead |
| Q3 | Aug 13 | Network partition | Platform Lead |
| Q4 | Oct 15 | Multi-region failover | VP Engineering |
| Q4 | Nov 12 | Tabletop exercise (full outage) | CTO |

### Drill Execution Checklist

**Before Drill:**
- [ ] Schedule approved by stakeholders
- [ ] Affected tenants notified (if production)
- [ ] Monitoring dashboards ready
- [ ] Runbook printed/accessible
- [ ] Roll-back plan documented

**During Drill:**
- [ ] Start time recorded
- [ ] All steps timed
- [ ] Issues documented in real-time
- [ ] Communication tested

**After Drill:**
- [ ] Recovery time recorded
- [ ] Gaps identified
- [ ] Runbook updates documented
- [ ] Lessons learned meeting scheduled

### Drill Report Template

```markdown
## DR Drill Report

**Date:** YYYY-MM-DD
**Scenario:** [Description]
**Duration:** X hours Y minutes
**Participants:** [List]

### Objectives
- [ ] Objective 1
- [ ] Objective 2

### Timeline
| Time | Event | Notes |
|------|-------|-------|
| HH:MM | Drill started | |
| HH:MM | [Step] | |
| HH:MM | Recovery complete | |

### Metrics
- RTO Target: X min
- RTO Actual: X min
- RPO Target: X min  
- RPO Actual: X min

### Issues Identified
1. [Issue description]
   - Impact: [High/Medium/Low]
   - Action: [Remediation]

### Recommendations
1. [Recommendation]

### Sign-off
- [ ] Platform Lead
- [ ] Security Lead
- [ ] VP Engineering
```

---

## Post-Incident Review

### Timeline (after incident resolution)

| Time | Action |
|------|--------|
| +24 hours | Preliminary timeline documented |
| +48 hours | Post-incident review meeting |
| +1 week | Written post-mortem published |
| +2 weeks | Action items assigned |
| +30 days | Action items verified complete |

### Post-Mortem Template

```markdown
## Post-Mortem: [Incident Title]

**Date:** YYYY-MM-DD
**Duration:** X hours Y minutes
**Severity:** SEV-X
**Author:** [Name]

### Summary
[2-3 sentence summary]

### Impact
- Affected tenants: X
- Failed queries: X
- Revenue impact: $X

### Timeline
[Detailed timeline with timestamps]

### Root Cause
[Technical explanation]

### Resolution
[What fixed it]

### What Went Well
- Item 1
- Item 2

### What Went Poorly
- Item 1
- Item 2

### Action Items
| ID | Action | Owner | Due Date |
|----|--------|-------|----------|
| 1 | [Action] | @owner | YYYY-MM-DD |

### Lessons Learned
[Key takeaways]
```

---

## Appendix

### Useful Commands

```bash
# Quick health check all services
./scripts/health-check.sh --all

# List recent incidents
./scripts/incidents.sh --list --days=30

# Generate incident report
./scripts/incidents.sh --report --id=<incident_id>

# Validate backups
./scripts/validate-backups.sh --all

# Test failover (dry-run)
./scripts/failover.sh --dry-run --target=starrocks-replica
```

### Emergency Contacts

| Service | Contact |
|---------|---------|
| AWS Support | [Support Case Link] |
| GCP Support | [Support Case Link] |
| PagerDuty | [Escalation Policy] |
| Slack | #incident-response |

### Related Documents

- [Runbook: Tenant Onboarding](./tenant-onboarding.md)
- [Runbook: StarRocks Setup](./starrocks-setup.md)
- [Architecture: Platform Overview](../architecture/platform-overview.md)
- [Security: Incident Response](../security/incident-response.md)
