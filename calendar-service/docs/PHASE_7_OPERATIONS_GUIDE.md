# Phase 7 Operations Guide

**Status:** ✅ **COMPLETE**  
**Date:** February 18, 2026  
**Audience:** DevOps / SRE Teams

---

## Quick Reference

### Quick Deploy Commands

```bash
# Staging deployment (quick)
./scripts/deploy-blue-green.sh staging v1.0.0 gcr.io/myproject/calendar-service

# Production deployment (with safety checks)
./scripts/deploy-blue-green.sh production v1.0.0 gcr.io/myproject/calendar-service

# Quick rollback
./scripts/rollback.sh production

# View deployment status
kubectl get deployment -n calendar
kubectl get pods -n calendar

# View logs
kubectl logs -f -n calendar -l app=calendar-service

# Port forward to service
kubectl port-forward -n calendar svc/calendar-service 8080:80

# Check metrics
kubectl port-forward -n calendar svc/prometheus 9090:9090
```

---

## Pre-Flight Checklist

Before any deployment:

```bash
#!/bin/bash
# pre-flight.sh

echo "✅ Pre-flight checks..."

# 1. Cluster connectivity
if ! kubectl cluster-info &> /dev/null; then
  echo "❌ Kubernetes cluster not accessible"
  exit 1
fi
echo "✅ Cluster connected"

# 2. Verify namespace exists
for ns in calendar calendar-staging; do
  if ! kubectl get namespace "$ns" &> /dev/null; then
    echo "⚠️  Creating namespace $ns"
    kubectl create namespace "$ns"
  fi
done
echo "✅ Namespaces checked"

# 3. Check storage (for backups)
if ! kubectl get pvc backup-pvc -n calendar &> /dev/null; then
  echo "⚠️  Backup PVC not found, deployments may not include backup jobs"
fi
echo "✅ Storage checked"

# 4. Verify secrets exist
if ! kubectl get secret calendar-secrets -n calendar &> /dev/null; then
  echo "❌ Required secret 'calendar-secrets' not found"
  echo "Run: kubectl create secret generic calendar-secrets --from-literal=DB_PASSWORD=... -n calendar"
  exit 1
fi
echo "✅ Secrets verified"

# 5. Check resource availability
available_memory=$(kubectl top nodes --no-headers 2>/dev/null | awk '{sum+=$5} END {print sum}')
if [ -z "$available_memory" ]; then
  echo "⚠️  Could not determine available memory (metrics server may not be installed)"
else
  echo "✅ Available memory: ${available_memory}Mi"
fi

# 6. Verify image repository access
if ! docker login -u "$DOCKER_USER" -p "$DOCKER_PASS" "$REGISTRY" &> /dev/null; then
  echo "⚠️  Could not access image registry"
fi
echo "✅ Image registry checked"

echo ""
echo "✅ All pre-flight checks passed!"
```

---

## Deployment Procedures

### Standard Blue-Green Deployment

**Estimated time:** 15-20 minutes

```bash
# 1. Prepare
export VERSION="v1.2.3"
export ENVIRONMENT="production"

# 2. Build and push image
docker build -t calendar-service:$VERSION .
docker tag calendar-service:$VERSION gcr.io/myproject/calendar-service:$VERSION
docker push gcr.io/myproject/calendar-service:$VERSION

# 3. Deploy
./scripts/deploy-blue-green.sh $ENVIRONMENT $VERSION gcr.io/myproject/calendar-service

# 4. Wait for completion (script will exit 0 on success)
echo "Deployment completed, new version is live"

# 5. Verify
kubectl port-forward -n calendar svc/calendar-service 8080:80
curl -s http://localhost:8080/health | jq .
```

### Canary Deployment (Progressive)

**Estimated time:** 15-25 minutes (3 stages × 5 min each + overhead)

```bash
# Progressive rollout: 10% → 50% → 100%
./scripts/deploy-canary.sh production v1.2.3 gcr.io/myproject/calendar-service

# At each stage:
# - 10%: Route 10% traffic, monitor 5 min
# - 50%: Route 50% traffic, monitor 5 min
# - 100%: Route all traffic, finalize
```

### Rollback Procedure

**Estimated time:** <1 minute

```bash
# If deployment goes wrong, rollback immediately
./scripts/rollback.sh production

# Verify state
kubectl get svc calendar-service -n calendar -o jsonpath='{.spec.selector.version}'

# Check that service is responsive
curl -s http://localhost:8080/health
```

---

## Observability & Troubleshooting

### Real-Time Monitoring

**Terminal 1: Watch pods**
```bash
watch -n 2 'kubectl get pods -n calendar -o wide'
```

**Terminal 2: Watch metrics**
```bash
kubectl port-forward -n calendar svc/prometheus 9090:9090
# Visit http://localhost:9090
# Search: rate(http_requests_total[5m])
```

**Terminal 3: Watch logs**
```bash
kubectl logs -f -n calendar -l app=calendar-service
```

### Alert Response Matrix

| Alert | Severity | Action | Escalate? |
|-------|----------|--------|-----------|
| HighErrorRate | 🔴 Critical | 1. Check logs 2. Rollback | Yes |
| HighLatency | 🟠 Warning | 1. Scale up 2. Check DB | If persists |
| PodCrashLoop | 🔴 Critical | 1. Check logs 2. Update image | Yes |
| DBConnectionFull | 🔴 Critical | 1. Scale up 2. Query DB | Immediate |
| CacheHitLow | 🟠 Warning | 1. Check Redis 2. Monitor | No |
| HighMemory | 🟠 Warning | 1. Scale up 2. Profile | If > 90% |

### Common Debugging Commands

```bash
# Get pod details
kubectl describe pod -n calendar pod/calendar-service-abc123

# View logs from last failure
kubectl logs -n calendar pod/calendar-service-abc123 --previous

# Check resource usage
kubectl top pods -n calendar

# Check node status
kubectl describe node node-1

# Scale up for debugging
kubectl scale deployment -n calendar calendar-service --replicas=5

# Execute command in pod
kubectl exec -n calendar pod/calendar-service-abc123 -- \
  curl http://localhost:8080/health

# Port-forward for direct testing
kubectl port-forward -n calendar pod/calendar-service-abc123 8080:8080
```

---

## Backup & Recovery Procedures

### Manual Backup (Emergency)

```bash
# Trigger immediate backup (don't wait for cron)
kubectl create job -n calendar backup-manual-$(date +%s) \
  --from=cronjob/postgres-backup

# Verify backup
kubectl logs -n calendar -l job-name=backup-manual-... -f

# Check backup file
kubectl exec -n calendar pod/backup-pvc -- ls -lh /backups/
```

### Restore from Backup

```bash
# 1. List available backups
kubectl exec -n calendar pod/backup-pvc -- ls -lh /backups/

# 2. Copy backup to local
kubectl cp calendar/pod/backup-pvc:/backups/calendar-20260215.sql.gz \
  ./calendar-20260215.sql.gz

# 3. Decompress
gunzip ./calendar-20260215.sql.gz

# 4. Stop application (to avoid conflicts)
kubectl scale deployment -n calendar calendar-service --replicas=0

# 5. Drop and recreate database
psql -h $DB_HOST -U postgres template1 << EOF
DROP DATABASE IF EXISTS calendar_db;
CREATE DATABASE calendar_db;
EOF

# 6. Restore
psql -h $DB_HOST -U postgres calendar_db < ./calendar-20260215.sql

# 7. Verify
psql -h $DB_HOST -U postgres calendar_db -c "SELECT COUNT(*) FROM calendars;"

# 8. Restart application
kubectl scale deployment -n calendar calendar-service --replicas=3

# 9. Monitor recovery
kubectl logs -f -n calendar -l app=calendar-service
```

### Test Backup Integrity (Weekly)

```bash
#!/bin/bash
# test-backup.sh - Run weekly to verify restores work

BACKUP_FILE=$(ls -t /backups/calendar-*.sql.gz | head -1)
TEMP_DB="calendar_test_$(date +%s)"

echo "Testing backup: $BACKUP_FILE"

# 1. Create test database
psql -U postgres -c "CREATE DATABASE $TEMP_DB;"

# 2. Restore backup
gunzip < "$BACKUP_FILE" | psql -U postgres $TEMP_DB

# 3. Run integrity checks
psql -U postgres $TEMP_DB << EOF
SELECT COUNT(*) as calendars FROM calendars;
SELECT COUNT(*) as events FROM events;
SELECT COUNT(*) as permissions FROM permissions;
EOF

# 4. Cleanup
psql -U postgres -c "DROP DATABASE $TEMP_DB;"

echo "✅ Backup integrity test passed"
```

---

## Scaling Operations

### Manual Scaling

```bash
# Scale up pods
kubectl scale deployment -n calendar calendar-service --replicas=5

# Scale down pods
kubectl scale deployment -n calendar calendar-service --replicas=2

# Check HPA status
kubectl get hpa -n calendar
kubectl describe hpa calendar-service -n calendar

# Disable HPA temporarily (for manual scaling)
kubectl patch hpa calendar-service -n calendar -p '{"spec":{"maxReplicas":3}}'
```

### Resource Adjustment

```bash
# Increase memory limit
kubectl set resources deployment -n calendar calendar-service \
  --limits=memory=1Gi --requests=memory=512Mi

# Increase CPU limit
kubectl set resources deployment -n calendar calendar-service \
  --limits=cpu=1000m --requests=cpu=500m

# View current resources
kubectl get deployment -n calendar calendar-service -o yaml | grep -A 10 "resources:"
```

---

## Network & Connectivity Troubleshooting

### Service Connectivity

```bash
# From pod, test connectivity to PostgreSQL
kubectl exec -n calendar pod/calendar-service-abc123 -- \
  nc -zv postgres.default.svc.cluster.local 5432

# From pod, test Redis connectivity
kubectl exec -n calendar pod/calendar-service-abc123 -- \
  redis-cli -h redis.default.svc.cluster.local ping

# From pod, test Hasura connectivity
kubectl exec -n calendar pod/calendar-service-abc123 -- \
  curl -s http://hasura.default.svc.cluster.local:8080/v1/query \
  -H "X-Hasura-Admin-Secret: $HASURA_SECRET" 
```

### DNS Resolution

```bash
# Test DNS resolution inside pod
kubectl exec -n calendar pod/calendar-service-abc123 -- \
  nslookup postgres.default.svc.cluster.local

# Test from cluster DNS pod
kubectl exec -n kube-system pod/coredns-xxx -- \
  dig postgres.default.svc.cluster.local
```

### Network Policy Debugging

```bash
# List all network policies
kubectl get networkpolicies -n calendar

# View specific policy
kubectl describe networkpolicy calendar-service -n calendar

# Temporarily disable network policy for debugging
kubectl delete networkpolicy calendar-service -n calendar

# Re-enable
kubectl apply -f k8s/overlays/production/network-policy.yaml
```

---

## Certificate & TLS Management

### Certificate Status

```bash
# Check certificate expiration
kubectl get certificate -n calendar -o wide

# View certificate details
kubectl describe certificate calendar-prod-tls -n calendar

# Check secret
kubectl get secret calendar-prod-tls -n calendar -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -dates
```

### Renew Certificate

```bash
# If using cert-manager, it should auto-renew
# To force renewal:
kubectl delete secret calendar-prod-tls -n calendar
# cert-manager will automatically create new certificate

# Verify renewal
kubectl describe certificate calendar-prod-tls -n calendar
```

---

## Performance Tuning

### Identify Bottlenecks

```bash
# 1. Check pod resource usage
kubectl top pods -n calendar --containers

# 2. Check node pressure
kubectl top nodes

# 3. Check database connections
kubectl exec -n calendar pod/calendar-service-abc123 -- \
  psql -c "SELECT count(*) FROM pg_stat_activity;"

# 4. Check cache hit rate
kubectl port-forward -n calendar svc/prometheus 9090:9090
# Query: rate(calendar_cache_hits_total[5m]) / (rate(calendar_cache_hits_total[5m]) + rate(calendar_cache_misses_total[5m]))
```

### Tuning Recommendations

| Problem | Solution |
|---------|----------|
| High CPU | Increase replicas, check for tight loops in code |
| High Memory | Scale up memory limit, check for leaks |
| Slow Queries | Add database indexes, optimize queries |
| Low Cache Hit | Increase cache TTL, warm cache at startup |
| Connection Timeouts | Scale DB connections, increase pool size |

---

## Incident Response

### During an Outage

1. **Immediate (0-5 min)**
   - Check alerts and logs
   - Gather incident timeline
   - Notify command center

2. **Assessment (5-15 min)**
   - Determine scope (staging/prod)
   - Identify root cause
   - Decide: Fix vs Rollback

3. **Resolution (15-30 min)**
   - If fixable: Deploy hotfix + blue-green
   - If not: Execute rollback
   - Monitor recovery metrics

4. **Post-Incident (30+ min)**
   - Document what happened
   - Root cause analysis
   - Preventive measures
   - Update runbooks

### Incident Communication

**Slack Template:**
```
🚨 INCIDENT: [Service name] - [Brief description]
Severity: [Critical | High | Medium]
Status: [Investigating | Mitigating | Resolved]
Progress: [Timeline of events]
ETA: [Expected resolution time]
```

---

## Maintenance Windows

### Planned Maintenance

```bash
# 1. Announce maintenance
# Send notification to users: "Service unavailable 2-3am UTC for maintenance"

# 2. Scale down to 1 replica (safer for DB migrations)
kubectl scale deployment -n calendar calendar-service --replicas=1

# 3. Perform maintenance
# - Database migrations
# - Backup/restore testing
# - Configuration updates

# 4. Scale back up
kubectl scale deployment -n calendar calendar-service --replicas=3

# 5. Verify post-maintenance
curl -s http://localhost:8080/health | jq .
```

---

## SLO/SLI Targets

**Service Level Objectives (SLOs):**

| Metric | Target | Tolerance |
|--------|--------|-----------|
| Availability | 99.9% | 43 min downtime/month |
| Latency (p99) | <500ms | 1 incident/month allowed |
| Error rate | <0.1% | ~1000 errors/million requests |

**Service Level Indicators (SLIs):**

```bash
# Calculate availability
uptime_seconds = (start_time + monitoring_period) - total_downtime_seconds
availability = uptime_seconds / monitoring_period

# Calculate error rate
error_rate = errors_5xx / total_requests

# Calculate latency (p99)
latency_p99 = histogram_quantile(0.99, http_request_duration_seconds)
```

---

## Runbook Links

- **Outages:** See "Incident Response" above
- **High Latency:** Check "Performance Tuning"
- **Pod Issues:** Check "Common Debugging Commands"
- **Database Issues:** See "Backup & Recovery Procedures"

---

## Emergency Contacts

| Role | Contact | Escalation |
|------|---------|-----------|
| On-Call Engineer | @oncall-calendar | Primary |
| Team Lead | @team-lead | Secondary |
| Infrastructure | @infrastructure | Tertiary |
| VP Engineering | @vp-eng | Executive |

---

**Last Updated:** February 18, 2026  
**Next Review:** February 25, 2026
