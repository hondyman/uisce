# 🚀 Phase 7 Quick Reference - Production Deployment

**Status:** ✅ COMPLETE AND READY FOR DEPLOYMENT  
**Date:** February 18, 2026  
**Version:** 1.0.0  

---

## 📋 Quick Navigation

### Main Documentation
- [PHASE_7_DEPLOYMENT_GUIDE.md](PHASE_7_DEPLOYMENT_GUIDE.md) - Complete deployment procedures
- [PHASE_7_DEPLOYMENT_CHECKLIST.md](PHASE_7_DEPLOYMENT_CHECKLIST.md) - Pre/during/post deployment tasks
- [PHASE_7_COMPLETE.md](PHASE_7_COMPLETE.md) - Executive summary

### Deployment Scripts
```bash
# Deploy to staging
./scripts/deploy-staging.sh 1.0.0

# Canary to production (10% → 50% → 100%)
./scripts/deploy-canary.sh 1.0.0 10

# Emergency rollback
./scripts/emergency-rollback.sh
```

### Configuration Files
- [k8s/deployment.yaml](k8s/deployment.yaml) - Kubernetes manifests
- [config/prometheus.yml](config/prometheus.yml) - Metrics collection
- [config/alert_rules.yml](config/alert_rules.yml) - 15+ alert rules
- [docker-compose.prod.yml](#) - Production docker setup

---

## 🎯 Quick Start (First Time Deploying)

### 1. Pre-Deployment (48 hours before)
```bash
# Validate all prerequisites
./scripts/validate-env.sh

# Check all tests passing
go test -tags=security -v ./tests/security/...

# Review deployment guide
cat PHASE_7_DEPLOYMENT_GUIDE.md | head -50
```

### 2. Staging Deployment
```bash
# Deploy to staging environment
./scripts/deploy-staging.sh 1.0.0

# Wait for deployment to complete
# Verify health: curl http://staging-api/health
# Monitor metrics dashboard in Grafana
```

### 3. Production Canary
```bash
# Set environment variables first!
export JWT_SECRET="$(openssl rand -hex 32)"
export DATABASE_URL="postgresql://..."

# Execute safe canary rollout
./scripts/deploy-canary.sh 1.0.0 10

# Monitor dashboard for 5 minutes
# Auto-rollback occurs if error rate > 5%
```

### 4. Monitor (24 hours)
```bash
# Watch logs in real-time
kubectl logs -f deployment/calendar-service-prod -n calendar-service

# Monitor metrics
curl http://prometheus:9090/api/v1/query?query=http_requests_total

# Check alerts in AlertManager
curl http://alertmanager:9093/api/v1/alerts
```

### 5. Rollback (if needed)
```bash
# One-command emergency rollback
./scripts/emergency-rollback.sh

# Returns to previous version in < 5 minutes
```

---

## 🔒 Security Verification Checklist

Before deploying to production, verify:

```bash
# JWT validation working (should get 401)
curl -X GET http://api/calendars

# With valid JWT (should get 200)
curl -X GET http://api/calendars \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "X-Tenant-ID: test-tenant"

# Cross-tenant access prevented (should get 403)
curl -X GET http://api/calendars \
  -H "Authorization: Bearer $DIFFERENT_TENANT_JWT" \
  -H "X-Tenant-ID: another-tenant"

# Rate limiting working (should get 429 after 10 RPS)
for i in {1..20}; do curl http://api/info; done

# Audit logging working (mutations should appear in logs)
curl -X POST http://api/calendars \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"name":"Test Calendar"}'
```

---

## 📊 Monitoring & Alerts Dashboard

### Grafana
```
URL: http://grafana.example.com:3000
Default: admin / admin
Dashboard: "Calendar Service - Production"
```

### Prometheus Queries
```
# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# Latency p95
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Rate limit hits
rate(http_requests_total{status="429"}[5m])

# Active connections
pg_stat_activity_count
```

### Alert Rules Status
```bash
# View all alerts
curl http://prometheus:9090/api/v1/rules

# Check firing alerts
curl http://alertmanager:9093/api/v1/alerts?active=true
```

---

## 🆘 Emergency Procedures

### Service Down? Immediate Steps

```bash
# 1. Verify it's really down
curl -v http://api.example.com/health

# 2. Check pod status
kubectl get pods -n calendar-service

# 3. View recent logs
kubectl logs deployment/calendar-service-prod -n calendar-service --tail=50

# 4. Check alerts fired
kubectl describe nodes  # Check disk/memory/CPU

# 5. If can't fix in 5 min → ROLLBACK
./scripts/emergency-rollback.sh
```

### High Error Rate (> 5%)

```bash
# 1. Check recent deployments
kubectl rollout history deployment/calendar-service-prod -n calendar-service

# 2. View error logs
kubectl logs deployment/calendar-service-prod -n calendar-service --tail=100 | grep ERROR

# 3. Check database status
kubectl run -it --rm debug --image=postgres:15 -- \
  psql $DATABASE_URL -c "SELECT COUNT(*) FROM pg_stat_activity;"

# 4. If errors persist → ROLLBACK
./scripts/emergency-rollback.sh
```

### Database Connection Pool Exhausted

```bash
# 1. Check current connections
$SELECT count(*) FROM pg_stat_activity;

# 2. Kill idle connections
$SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle';

# 3. If still high → Scale back pods
kubectl scale deployment/calendar-service-prod --replicas=1

# 4. Monitor recovery
kubectl logs deployment/calendar-service-prod -n calendar-service -f
```

---

## 📈 Performance Baselines

### Expected Metrics (Post-Deployment)

| Metric | Target | Alert at |
|--------|--------|----------|
| Error Rate | < 0.5% | > 5% |
| Latency p50 | < 100ms | N/A |
| Latency p95 | < 500ms | > 1000ms |
| Latency p99 | < 1000ms | > 2000ms |
| Rate Limits | < 1% | > 5% |
| 429 Responses | 0 (normal) | > 50/5min |
| JWT Failures | < 0.1% | > 1% |
| DB Connection % | 30-50% | > 80% |

---

## 🔧 Configuration Reference

### Environment Variables

```bash
# REQUIRED (Must be set before deployment)
JWT_SECRET="random-hex-string-min-32-chars"
DATABASE_URL="postgresql://user:pass@host:5432/db_prod"

# OPTIONAL (Defaults shown)
RATE_LIMIT_RPS=10              # Requests per second per tenant
RATE_LIMIT_BURST=20            # Token bucket burst size
LOG_LEVEL=info                 # debug, info, warn, error
METRICS_ENABLED=true           # Enable Prometheus metrics
METRICS_PORT=9090              # Metrics collection port
TRACING_ENABLED=true           # Enable distributed tracing
TRACING_SAMPLE_RATE=0.1        # 10% of requests traced
```

### Docker Compose Setup

```bash
# Create production environment file
cat > .env.production << EOF
JWT_SECRET=$(openssl rand -hex 32)
DATABASE_URL=postgresql://calendar:password@postgres:5432/calendar_prod
DB_MAX_CONNECTIONS=25
DB_MIN_CONNECTIONS=5
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
LOG_LEVEL=info
METRICS_ENABLED=true
METRICS_PORT=9090
EOF

# Deploy with docker-compose
docker-compose -f docker-compose.prod.yml up -d

# Verify all services
docker-compose -f docker-compose.prod.yml ps
```

---

## 📞 Contacts & Escalation

### On-Call Rotation
```bash
# Get current on-call engineer
./scripts/get-oncall.sh

# Escalation: If no response within 5 min
escalate_to_team_lead.sh
```

### Important Channels
- **Slack:** #incidents (real-time updates)
- **PagerDuty:** (if configured)
- **Email:** oncall@example.com

### Deployment Team
- **Incident Commander:** [Name]
- **Technical Lead:** [Name]
- **Database Admin:** [Name]
- **DevOps Lead:** [Name]

---

## ✅ Post-Deployment Validation

### Immediate (After deployment completes)
- [ ] All pods READY (3/3)
- [ ] All health checks passing
- [ ] Metrics flowing to Prometheus
- [ ] Error rate < 1%
- [ ] No alerts firing

### 1 Hour
- [ ] JWT validation confirmed working
- [ ] Tenant isolation verified
- [ ] Rate limiting tested
- [ ] Audit logs appearing
- [ ] Database connections stable

### 24 Hours
- [ ] Error rate < 0.5%
- [ ] Latency trends stable
- [ ] No cascading failures
- [ ] All security features active
- [ ] No customer issues reported

### 1 Week
- [ ] Performance baseline established
- [ ] Cost trends analyzed
- [ ] Incident review completed
- [ ] Documentation updated
- [ ] Team feedback collected

---

## 🎓 Learning Resources

### Understanding the Architecture
1. Read: [PHASE_7_DEPLOYMENT_GUIDE.md](PHASE_7_DEPLOYMENT_GUIDE.md) sections 1-3
2. Review: [k8s/deployment.yaml](k8s/deployment.yaml) comments
3. Study: [config/alert_rules.yml](config/alert_rules.yml) rules

### Operational Knowledge
1. Practice staging deployment
2. Review incident response procedures
3. Know the rollback procedure
4. Understand monitoring dashboards

### Security Deep Dive
1. How JWT validation works
2. How tenant isolation is enforced
3. How rate limiting protects the service
4. How audit logging records mutations

---

## 🚀 Deployment Timeline

### Recommended Schedule

**Monday-Wednesday:** Preparation
- Code review ✓
- Testing ✓
- Team training ✓

**Thursday:** Staging Validation
- Deploy to staging ✓
- Load testing ✓
- Monitoring verification ✓

**Friday 02:00 UTC:** Production Deployment
- Execute canary rollout
- 5-min monitoring
- Scale as needed
- 24-hour continuous monitoring

**Saturday:** Post-Deployment Review
- Metrics analysis
- Lessons learned
- Documentation updates

---

## 🎯 Key Success Factors

1. **Preparation** - 48-hour checklist completed
2. **Communication** - Team briefed and ready
3. **Monitoring** - Dashboards actively watched
4. **Patience** - Full 5-min canary monitoring
5. **Quick Rollback** - < 5 min if issues detected

---

## 📝 Documentation Index

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [PHASE_7_DEPLOYMENT_GUIDE.md](PHASE_7_DEPLOYMENT_GUIDE.md) | Complete procedures | 30 min |
| [PHASE_7_DEPLOYMENT_CHECKLIST.md](PHASE_7_DEPLOYMENT_CHECKLIST.md) | Task checklist | 15 min |
| [PHASE_7_COMPLETE.md](PHASE_7_COMPLETE.md) | Executive summary | 10 min |
| [SECURITY_CHECKLIST.md](docs/deployment/SECURITY_CHECKLIST.md) | Security verification | 20 min |
| [SECURITY_RUNBOOK.md](docs/operations/SECURITY_RUNBOOK.md) | Incident response | 25 min |
| This file | Quick reference | 10 min |

---

## 🚀 You're Ready!

Everything is prepared for production deployment:

✅ Code reviewed and tested  
✅ Infrastructure provisioned  
✅ Monitoring configured  
✅ Procedures documented  
✅ Team trained  
✅ Rollback procedure ready  

**Status: READY FOR PRODUCTION DEPLOYMENT** 🎉

---

**Questions?** Review the full guides linked above.  
**Emergency?** Execute: `./scripts/emergency-rollback.sh`  
**Need help?** Contact on-call engineer.

---

Generated: February 18, 2026  
Status: ✅ Production Ready
