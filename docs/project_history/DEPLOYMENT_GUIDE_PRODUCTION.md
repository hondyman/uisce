# Deployment & Production Rollout Guide

**Project**: Advanced Validation Rules System v2.0  
**Date**: October 20, 2025  
**Target**: Production Deployment  
**Timeline**: 1-2 weeks from approval  

---

## 📋 Pre-Deployment Checklist

### Code Quality ✅
- [x] Frontend code compiles (0 TypeScript errors)
- [x] Backend code compiles (go build success)
- [x] All imports resolved
- [x] No console errors in dev
- [x] Type safety verified

### Testing ✅
- [ ] UAT passed (pending)
- [ ] All 7 test scenarios pass
- [ ] Browser compatibility verified
- [ ] Performance benchmarks met
- [ ] Load testing completed

### Documentation ✅
- [x] Integration guide created
- [x] Backend API documented
- [x] UAT guide created
- [x] User training materials prepared
- [x] Deployment guide (this doc)

### Infrastructure
- [ ] Staging environment ready
- [ ] Production environment verified
- [ ] Database backups created
- [ ] Monitoring configured
- [ ] Rollback procedure documented

### Security
- [ ] Tenant scope validated in code review
- [ ] No hardcoded secrets
- [ ] Error messages don't leak sensitive data
- [ ] API headers properly validated
- [ ] SQL injection risks reviewed

---

## 🚀 Deployment Timeline

### Week 1: Staging Deployment

**Day 1-2: Staging Setup**
```
Monday - Tuesday
├── Deploy frontend to staging
├── Deploy backend changes to staging
├── Run smoke tests
├── Setup monitoring and logging
└── Notify stakeholders
```

**Day 3: Staging Testing**
```
Wednesday
├── Run full UAT in staging
├── Check API endpoints
├── Verify entity definitions load
├── Test all 7 scenarios
└── Document any issues
```

**Day 4-5: UAT & Feedback**
```
Thursday - Friday
├── Business users access staging
├── Run through test scenarios
├── Collect feedback
├── Document issues
└── Plan fixes if needed
```

### Week 2: Production Deployment

**Day 1-2: Final Verification**
```
Monday - Tuesday
├── Code review completion
├── Security audit
├── Performance tests
├── Final UAT sign-off
└── Prepare release notes
```

**Day 3-4: Production Deployment**
```
Wednesday - Thursday
├── Maintenance window (2 hours)
├── Deploy frontend
├── Deploy backend
├── Run smoke tests
├── Verify all features
├── Monitor for errors
└── Enable in Feature Flags
```

**Day 5: Post-Deployment**
```
Friday
├── Monitor metrics
├── Check error rates
├── Collect user feedback
├── Document lessons learned
└── Plan follow-up sessions
```

---

## 📦 Deployment Steps

### Stage 1: Pre-Deployment (Day before)

**1. Create Database Backup**
```bash
# Backup production database
pg_dump \
  --host=prod-db.example.com \
  --username=postgres \
  --password \
  semlayer > backup-prod-$(date +%Y%m%d).sql

# Store in secure location
aws s3 cp backup-prod-$(date +%Y%m%d).sql s3://backups/semlayer/
```

**2. Verify Rollback Plan**
```bash
# Document current version
git tag -a "prod-v1.0" HEAD -m "Pre-deployment tag"
git push origin prod-v1.0

# Document rollback commands
echo "Rollback: git checkout prod-v1.0" >> ROLLBACK.md
```

**3. Notify Stakeholders**
```
Email: All Teams
Subject: Validation Rules System v2.0 Deployment - [DATE] at [TIME]

Scheduled Deployment:
- Date: [DATE]
- Time: [TIME] - [TIME] UTC
- Duration: ~2 hours
- Impact: Brief outage to validation rules UI

What's New:
- Advanced field selector with relationships
- Rule cloning to prevent duplicates
- Sample data generation for testing
- Enhanced impact analysis

Rollback: Available if issues found
Support: [CONTACT INFO]
```

### Stage 2: Deployment (Maintenance Window)

**1. Deploy Frontend**
```bash
# Build production bundle
cd frontend
npm run build

# Deploy to CDN/static hosting
# Option A: S3 + CloudFront
aws s3 sync dist/ s3://semlayer-prod-frontend/

# Option B: Docker
docker build -t semlayer-frontend:v2.0 .
docker push registry.example.com/semlayer-frontend:v2.0
kubectl set image deployment/semlayer-frontend frontend=registry.example.com/semlayer-frontend:v2.0

# Clear CDN cache
aws cloudfront create-invalidation --distribution-id E123EXAMPLE --paths "/*"
```

**2. Deploy Backend**
```bash
# Build backend binary
cd backend
go build -o semlayer-api ./cmd/server
# Add version info
go build -ldflags "-X main.Version=v2.0 -X main.BuildTime=$(date)" \
  -o semlayer-api ./cmd/server

# Deploy binary
# Option A: Docker
docker build -t semlayer-api:v2.0 -f Dockerfile.prod .
docker push registry.example.com/semlayer-api:v2.0
kubectl set image deployment/semlayer-api api=registry.example.com/semlayer-api:v2.0

# Option B: Direct deployment
scp semlayer-api prod-api-server:/opt/semlayer/
ssh prod-api-server "systemctl restart semlayer"

# Option C: Blue-Green
./deploy-blue-green.sh v2.0  # Custom script
```

**3. Run Health Checks**
```bash
# Check frontend loading
curl -I https://semlayer.example.com/

# Check API endpoints
curl -X GET "https://api.semlayer.example.com/api/entities?tenant_id=test&datasource_id=test" \
  -H "X-Tenant-ID: test" \
  -H "X-Tenant-Datasource-ID: test"

# Check database connection
curl -X GET "https://api.semlayer.example.com/api/health"
```

**4. Verify Critical Paths**
```bash
# Test 1: Load validation rules editor
curl -s "https://api.semlayer.example.com/api/rules?tenant_id=<LIVE_ID>&datasource_id=<LIVE_ID>" \
  -H "X-Tenant-ID: <LIVE_ID>" \
  -H "X-Tenant-Datasource-ID: <LIVE_ID>" | jq '.rules | length'

# Test 2: Get entity definitions
curl -s "https://api.semlayer.example.com/api/entities?tenant_id=<LIVE_ID>&datasource_id=<LIVE_ID>" \
  -H "X-Tenant-ID: <LIVE_ID>" \
  -H "X-Tenant-Datasource-ID: <LIVE_ID>" | jq '.count'

# Test 3: Get conflict rules
curl -s "https://api.semlayer.example.com/api/rules?tenant_id=<LIVE_ID>&datasource_id=<LIVE_ID>&entity=Employee&field=email" \
  -H "X-Tenant-ID: <LIVE_ID>" \
  -H "X-Tenant-Datasource-ID: <LIVE_ID>" | jq '.rules'

# All tests should return valid responses (no 500 errors)
```

### Stage 3: Post-Deployment (After Maintenance)

**1. Enable Feature Flags** (if using feature flags)
```bash
# Enable "AdvancedValidationRules" feature
curl -X POST "https://api.semlayer.example.com/api/features" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -d '{
    "feature": "AdvancedValidationRules",
    "enabled": true,
    "rollout_percentage": 10  # Start with 10% rollout
  }'

# Gradually increase rollout
# 10% → 25% → 50% → 100% (over 24-48 hours)
```

**2. Monitor Metrics**
```
Watch for:
✅ Error rate: Should remain < 0.5%
✅ Response time: Should be < 500ms
✅ User feedback: Monitor support channel
✅ API health: All endpoints returning 200/400 (not 500)
✅ Database performance: Query times normal

Tools:
- Datadog/New Relic - Application monitoring
- CloudWatch - AWS metrics
- Custom dashboards - Company's APM
```

**3. Enable Logging**
```
Log key events:
- Entity definition API calls (count, entities requested)
- Rule creation/updates (count, severity distribution)
- Sample data generation (count, record counts)
- Conflict detection triggers (count, conflicts found)
- User errors (count, error types)
```

**4. Notify Completion**
```
Email: All Teams
Subject: Validation Rules System v2.0 - Deployment Complete ✅

Status: SUCCESSFUL DEPLOYMENT

What's available:
✅ Advanced field selector with entity relationships
✅ Rule cloning to prevent duplicates
✅ Sample data generation for testing
✅ Enhanced impact analysis
✅ Conflict detection

Known issues: None (all tests passing)
Support: [CONTACT INFO]
Feedback: [LINK TO SURVEY]

Training: [SCHEDULE LINK]
Documentation: [LINK TO GUIDES]
```

---

## 🔄 Rollback Procedure

**If critical issues are found:**

### Immediate Rollback (Within 4 hours)
```bash
# 1. Identify the issue
# Check error logs, user reports, monitoring alerts

# 2. Decide to rollback
# Get approval from [APPROVER]

# 3. Execute rollback

# Frontend rollback
aws s3 sync s3://semlayer-prod-frontend-backup/ s3://semlayer-prod-frontend/
aws cloudfront create-invalidation --distribution-id E123EXAMPLE --paths "/*"

# Backend rollback
kubectl rollout undo deployment/semlayer-api
# OR
git checkout prod-v1.0
go build -o semlayer-api ./cmd/server
scp semlayer-api prod-api-server:/opt/semlayer/
ssh prod-api-server "systemctl restart semlayer"

# 4. Verify rollback
curl -X GET "https://api.semlayer.example.com/api/health"

# 5. Notify stakeholders
# Send status update email

# 6. Post-mortem
# Schedule incident review
```

---

## 📊 Post-Deployment Monitoring

### Dashboard Setup

**Create monitoring dashboard** showing:
```
Real-time Metrics:
├── Requests/second (should stay normal)
├── Error rate (should stay < 0.5%)
├── Response time P95 (should be < 1s)
├── New validation rules created (track adoption)
├── Entity API calls (should see traffic)
├── Sample data generations (track usage)
└── Conflict detection triggers (effectiveness)

Alert Thresholds:
├── Error rate > 2% → ALERT
├── Response time > 2s → WARNING
├── API down > 30s → CRITICAL
├── Database query slow > 5s → WARNING
└── Storage full > 80% → WARNING
```

### Monitoring Commands

**Check Logs**
```bash
# Recent errors
kubectl logs -l app=semlayer-api --since=1h | grep -i error

# API performance
kubectl logs -l app=semlayer-api | grep -i "duration\|latency" | tail -100

# Database queries
tail -100 /var/log/postgresql/postgresql.log | grep slow
```

**Check Metrics**
```bash
# CPU usage
kubectl top pod -l app=semlayer-api

# Memory usage
kubectl top node

# API latency percentiles
curl -s http://localhost:9090/metrics | grep http_request_duration_seconds
```

---

## 🎓 Training Schedule

### Week 2 Post-Deployment

**Monday 10:00 AM**: Business Analysts Webinar
```
Duration: 1 hour
Topics:
1. Creating rules from templates (10 min)
2. Understanding impact analysis (15 min)
3. Q&A (20 min)
4. Live walkthrough (15 min)

Recording: Available after training
Materials: [LINK]
```

**Tuesday 2:00 PM**: Data Stewards Workshop
```
Duration: 1.5 hours
Topics:
1. Complete workflow walkthrough (30 min)
2. Reviewing rules for conflicts (20 min)
3. Best practices & patterns (20 min)
4. Hands-on lab (20 min)

Sandbox: [LOGIN LINK]
```

**Wednesday 9:00 AM**: IT/Admin Technical Briefing
```
Duration: 1.5 hours
Topics:
1. Backend API endpoints (20 min)
2. Entity definitions (15 min)
3. Troubleshooting guide (15 min)
4. Performance tuning (15 min)
5. Q&A (15 min)

Docs: [ADMIN GUIDE LINK]
```

---

## 📈 Success Metrics (Post-Deployment)

**Track these metrics for 2 weeks post-deployment:**

### Adoption
- [ ] 30+ users have accessed validation rules
- [ ] 10+ new rules created in production
- [ ] 5+ rules cloned from existing rules
- [ ] 100+ sample data sets generated

### Quality
- [ ] 0 critical issues reported
- [ ] <5 medium issues reported
- [ ] >95% conflict detection accuracy (if tested)
- [ ] 0 duplicate rules created

### Performance
- [ ] API response time: 95th percentile < 500ms
- [ ] Error rate: < 0.1%
- [ ] Frontend load time: < 2 seconds
- [ ] Database queries: < 200ms avg

### User Satisfaction
- [ ] Training attendance: >80%
- [ ] User feedback: >4/5 stars
- [ ] Support tickets: < 5 about new features
- [ ] Recommendation score: >7/10

---

## 🔒 Security Checklist

Before deploying to production:

### Code Security
- [ ] No hardcoded credentials in code
- [ ] No secrets in git history
- [ ] SQL injection prevention reviewed
- [ ] XSS prevention verified
- [ ] CSRF protection in place

### API Security
- [ ] Tenant scope properly validated
- [ ] Headers properly checked
- [ ] Rate limiting configured
- [ ] Invalid input handling tested
- [ ] Error messages safe (no info leaks)

### Infrastructure Security
- [ ] SSL/TLS certificates valid
- [ ] Firewalls properly configured
- [ ] Database access restricted
- [ ] VPN/Bastion host in place
- [ ] Backups encrypted

### Compliance
- [ ] Data residency rules followed
- [ ] Access controls documented
- [ ] Audit logging enabled
- [ ] User permissions verified
- [ ] Security review completed

---

## 📞 Support & Escalation

### During Deployment
```
Issues contact: [ON-CALL ENGINEER]
Phone: [PHONE]
Slack: #semlayer-deployment
```

### Post-Deployment Support
```
First Response: < 15 minutes
Priority: Critical (app down) > High (feature broken) > Medium > Low

Support Channels:
- Slack: #semlayer-support
- Email: support@semlayer.example.com
- Jira: Create issue in SEMLAYER project
- Phone: [SUPPORT LINE]
```

---

## 📝 Deployment Log Template

```
DEPLOYMENT: Advanced Validation Rules v2.0
DATE: [DATE]
TIME: [START] - [END] UTC

PRE-DEPLOYMENT
├── [ ] Backup created
├── [ ] Notifications sent
├── [ ] Rollback plan verified
└── [ ] Health checks ready

DEPLOYMENT
├── [ ] Frontend deployed at [TIME]
├── [ ] Backend deployed at [TIME]
├── [ ] Health checks passed
└── [ ] Monitoring enabled

VERIFICATION
├── [ ] Test 1: [PASS/FAIL]
├── [ ] Test 2: [PASS/FAIL]
├── [ ] Test 3: [PASS/FAIL]
└── [ ] Test 4: [PASS/FAIL]

POST-DEPLOYMENT
├── [ ] Metrics normal
├── [ ] Error rate acceptable
├── [ ] Users can access
└── [ ] Team notified

ISSUES ENCOUNTERED
[None / List any issues]

RESOLUTION
[Describe how issues were resolved]

SIGN-OFF
Deployed by: [NAME]
Verified by: [NAME]
Approved by: [NAME]
```

---

## 🎉 Deployment Success Criteria

**Deployment is considered successful if:**

✅ No critical errors in logs  
✅ All endpoints responding  
✅ Error rate < 1%  
✅ Response times normal  
✅ User feedback positive  
✅ Rollback not needed  
✅ Monitoring in place  
✅ Support ready  

---

## 🚀 Next Steps

### Immediate (Week 1 Post-Deployment)
- [ ] Run monitoring checks
- [ ] Gather user feedback
- [ ] Document issues
- [ ] Schedule follow-up

### Short Term (Weeks 2-4)
- [ ] Promote feature flag to 100%
- [ ] Schedule training sessions
- [ ] Monitor adoption metrics
- [ ] Optimize performance

### Medium Term (Months 2-3)
- [ ] Collect feedback for improvements
- [ ] Plan v2.1 enhancements
- [ ] Migrate entity definitions to database
- [ ] Add advanced features (rules packages, templates marketplace)

---

*Deployment Guide - Advanced Validation Rules System v2.0*  
*Ready for Production Deployment*  
*Prepared: October 20, 2025*
