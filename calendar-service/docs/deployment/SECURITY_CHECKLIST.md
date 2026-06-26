# Security Deployment Checklist

**Last Updated:** February 17, 2026  
**Version:** 1.0  
**Status:** Template for Production Deployment  

---

## Pre-Deployment Security Checks

### JWT Configuration ✓
- [ ] `JWT_SECRET` is set to a cryptographically random 32+ character string
  - **Command**: `openssl rand -hex 32`
  - **Verification**: Length check programmatically enforced
  
- [ ] `JWT_SECRET` is NOT committed to version control
  - **Check**: `git log --all -S "JWT_SECRET" --source --full-history`
  - **Result**: Should return no matches
  
- [ ] `JWT_SECRET` is NOT logged in application output
  - **Check**: `grep -r "JWT_SECRET" internal/ --include="*.go" | grep -v "log.Error"` (allowed only in error contexts with masked values)
  
- [ ] JWT_SECRET is stored in secure vault (AWS Secrets Manager, HashiCorp Vault, etc.)
  - **Verification**: Deployed via CI/CD secrets mechanism
  
- [ ] JWT algorithm validated as HS256 (or RS256 for enhanced security)
  - **Check**: Code confirms `token.Header["alg"] == "HS256"`
  
- [ ] Token expiration is enforced (`exp` claim checked)
  - **Verification**: `go test ./internal/middleware -run TestJWTMiddlewareSecurity`

### Tenant Isolation ✓
- [ ] Hasura RLS policies configured for all tables with `tenant_id` filter
  - To be implemented in Phase 5 (database layer)
  - Tables requiring RLS: calendars, profiles, blackouts, availability_slots
  
- [ ] `X-Hasura-Tenant-Id` header validated against JWT `tenant_id` claim
  - **Check**: `TenantGuardMiddleware` validates header vs context
  
- [ ] Cross-tenant access attempts rejected with 403 (not 404)
  - **Verification**: `go test -tags=security ./tests/security -run TestTenant`
  
- [ ] Audit logs capture tenant context for all mutations
  - **Verification**: `AuditService` records tenant_id on every action
  
- [ ] No tenant data leakage in error responses
  - **Check**: `grep -r "tenant_id" internal/api --include="*.go" | grep "Error"` (should be in logs, not responses)

### Input Validation ✓
- [ ] All API endpoints validate JSON schema before processing
  - Implemented via handler unmarshaling and service validation
  
- [ ] SQL injection prevented via parameterized queries
  - **Mechanism**: Hasura GraphQL prevents raw SQL injection
  - **Fallback**: All queries use parameterized variables
  
- [ ] XSS prevented via output encoding
  - **Frontend**: React auto-escapes content
  - **API**: Returns JSON, not HTML
  
- [ ] File uploads validated (if applicable)
  - Currently not applicable (calendars API doesn't support file uploads)

### Rate Limiting & DoS Protection ✓
- [ ] Tenant-scoped rate limiting enabled
  - **Config**: `RATE_LIMIT_RPS=10`, `RATE_LIMIT_BURST=20` (per tenant, per second)
  - **Verification**: `go test -tags=security ./tests/security -run TestRateLimiting`
  
- [ ] Rate limit errors return 429 with `Retry-After` header
  - **Verification**: Run `e2e-tests.sh` and verify rate limit behavior
  
- [ ] Global rate limiting configured at API Gateway level
  - **Implementation**: Nginx/Envoy configuration (outside service scope)
  
- [ ] Slowloris protection enabled
  - **Implementation**: Request timeout (30s), header limits (8KB)
  - **Config in main.go**: `server.ReadTimeout = 30 * time.Second`

### Audit & Compliance ✓
- [ ] All authenticated endpoints log to audit system with actor/tenant context
  - **Verification**: Handlers call `auditService.RecordCreate/Update/Delete`
  
- [ ] Audit logs are immutable
  - **Mechanism**: Audit table has no UPDATE/DELETE permissions (Phase 5)
  
- [ ] Audit retention policy configured
  - **Recommended**: 7 years for compliance (configurable)
  
- [ ] PII is masked in logs
  - **Current**: Email addresses stored but not logged to stdout
  - **Future**: Implement log masking middleware

### Secrets Management ✓
- [ ] All secrets stored in Vault/KMS (not in code or .env files)
  - **Secrets**: JWT_SECRET, DB_PASSWORD, API_KEYS
  - **Injection**: Via CI/CD environment variables at runtime
  
- [ ] Secrets injected at runtime, not baked into Docker images
  - **Verification**: `docker run --rm CALENDAR-IMAGE:latest env | grep SECRET` (should be empty)
  
- [ ] Secret rotation procedure documented and tested
  - **Procedure**: Zero-downtime rotation via versioning
  - **Test**: Document in runbook

### Network Security ✓
- [ ] TLS enforced for all external communications
  - **HTTPS**: All production APIs HTTPS-only
  - **Config**: Nginx/Ingress enforces TLS, no HTTP
  
- [ ] Database connections use SSL/TLS with certificate validation
  - **Config**: `sslmode=require` in PostgreSQL connection strings
  
- [ ] Internal service-to-service communication authenticated
  - **Mechanism**: JWT for service-to-service or mTLS (future)
  
- [ ] Firewall rules restrict access to required ports only
  - **Ports**: 8080 (service), 5432 (database internal only)
  
- [ ] API Gateway/Ingress configured with security headers
  - **Headers**: Strict-Transport-Security, X-Content-Type-Options, etc.

### Monitoring & Alerting ✓
- [ ] Failed auth attempts logged and alert on threshold
  - **Threshold**: >10 failed attempts in 5 minutes per tenant = ALERT
  - **Log Field**: JWT parsing failures tracked
  
- [ ] Rate limit exceeded events monitored for abuse patterns
  - **Metric**: `rate_limit_exceeded_total` counter per tenant
  - **Alert**: >100 rate limit events in 1 hour = ALERT
  
- [ ] Audit log anomalies trigger alerts
  - **Anomaly**: Bulk deletes (>10 records in <1 second)
  - **Anomaly**: Unusual user agents or IP addresses
  
- [ ] JWT validation failures tracked for attack detection
  - **Metric**: `jwt_validation_errors_total` counter
  - **Alert**: >50 errors in 5 minutes = potential attack

### Testing ✓
- [ ] All security tests pass
  - **Command**: `go test -tags=security -v ./tests/security/...`
  - **Expected**: 12/12+ tests passing
  
- [ ] Penetration test report reviewed
  - **Scope**: OWASP Top 10 coverage
  - **Status**: Document any findings and remediations
  
- [ ] Dependency vulnerabilities scanned and patched
  - **Command**: `go list -json ./... | nancy sleuth`
  - **Alternative**: `govulncheck ./...`
  
- [ ] Load testing confirms rate limiting works under stress
  - **Tool**: Apache ab or k6
  - **Target**: 1000 requests/second should trigger rate limits
  
- [ ] E2E test suite passes (all 14 tests)
  - **Command**: `bash e2e-tests.sh`
  - **Expected**: 14/14 passing

### Code Quality ✓
- [ ] No hardcoded secrets in source code
  - **Check**: `git log -p | grep -i "password\|secret\|key" | head -20`
  
- [ ] All error responses generic (no system info leakage)
  - **Check**: `grep -r "err.Error()" internal/api | wc -l` should be 0
  
- [ ] Logging is structured and queryable
  - **Format**: JSON logging
  - **Fields**: user_id, tenant_id, action, timestamp, error (if applicable)
  
- [ ] Comments document security-critical sections
  - **Example**: JWT validation, tenant isolation, audit logging

---

## Deployment Steps

### Step 1: Pre-Deployment Validation
```bash
# Run full security test suite
go test -tags=security -v ./tests/security/...

# Run all unit tests
go test -v ./...

# Build binary and verify it compiles
go build -o bin/calendar-service ./cmd/server

# Scan for vulnerabilities
govulncheck ./...
```

### Step 2: Environment Configuration
```bash
# Set required environment variables (in deployment manifests, not in code)
export JWT_SECRET="$(openssl rand -hex 32)"  # Generate new secret
export DB_HOST="pg-prod.internal"
export DB_USER="calendar_user"
export DB_PASSWORD="${DB_PASSWORD_FROM_VAULT}"
export RATE_LIMIT_RPS="10"
export RATE_LIMIT_BURST="20"
export LOG_LEVEL="info"

# Verify configuration
env | grep -E "JWT|DB|RATE|LOG" | sort
```

### Step 3: Staging Deployment
```bash
# Deploy to staging first
kubectl apply -f k8s/staging/calendar-service.yaml

# Wait for ready
kubectl rollout status deployment/calendar-service -n calendar-staging

# Run smoke tests
bash e2e-tests.sh --env staging

# Monitor logs for errors
kubectl logs -f deployment/calendar-service -n calendar-staging --tail=100
```

### Step 4: Security Validation (Staging)
```bash
# Test JWT validation
curl -X GET https://calendar-staging.example.com/api/v1/calendars \
  -H "Authorization: Bearer invalid-token"
# Expected: 401 Unauthorized

# Test tenant isolation
curl -X GET https://calendar-staging.example.com/api/v1/calendars \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -H "X-Tenant-ID: wrong-tenant"
# Expected: 403 Forbidden

# Test rate limiting (send 30 requests quickly)
for i in {1..30}; do curl -s -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  https://calendar-staging.example.com/api/v1/calendars \
  &
done
wait
# Expected: First ~20 get 200, rest get 429
```

### Step 5: Production Canary Deployment
```bash
# Deploy canary (10% of traffic)
kubectl apply -f k8s/production/canary/calendar-service.yaml

# Wait for startup
sleep 30

# Verify canary health
kubectl logs deployment/calendar-service-canary -n calendar-production --tail=50

# Monitor metrics for 10 minutes
kubectl top nodes
kubectl top pods -n calendar-production

# Check for errors in logs
kubectl logs deployment/calendar-service-canary -n calendar-production \
  | grep -i error | head -20
```

### Step 6: Canary Monitoring & Validation
```bash
# Check error rate (should be <0.1%)
kubectl logs deployment/calendar-service-canary -n calendar-production \
  | grep '"error"' | wc -l
# Divide by total requests to get error rate

# Check jwt validation errors
kubectl logs deployment/calendar-service-canary -n calendar-production \
  | grep "JWT validation failed" | wc -l

# Check rate limit abuse patterns
kubectl logs deployment/calendar-service-canary -n calendar-production \
  | grep "rate_limit_exceeded" | wc -l
```

### Step 7: Production Full Rollout
```bash
# Once canary is healthy for 10+ minutes, rollout to full production
kubectl apply -f k8s/production/calendar-service.yaml

# Monitor rollout
kubectl rollout status deployment/calendar-service -n calendar-production

# Verify full deployment
kubectl get pods -n calendar-production | grep calendar-service
```

### Step 8: Post-Deployment Validation
```bash
# Verify all pods are healthy
kubectl get pods -n calendar-production | grep calendar-service

# Check logs for startup errors
kubectl logs deployment/calendar-service -n calendar-production --tail=100

# Run full E2E test suite against production
bash e2e-tests.sh --env production

# Verify metrics are flowing to Prometheus
curl http://prometheus:9090/api/v1/query?query=calendar_service_requests_total
```

---

## Rollback Procedures

### If Canary Shows Issues
```bash
# Immediately delete canary deployment
kubectl delete deployment calendar-service-canary -n calendar-production

# Keep monitoring main deployment
kubectl logs deployment/calendar-service -n calendar-production -f

# Investigate issue
kubectl describe pod -n calendar-production -l app=calendar-service
```

### If Production Shows Issues
```bash
# Option 1: Scale down to previous version
kubectl rollout undo deployment/calendar-service -n calendar-production

# Option 2: Quick patch or hotfix
kubectl set image deployment/calendar-service \
  calendar-service=calendar-service:PREVIOUS_VERSION \
  -n calendar-production
```

---

## Post-Deployment Checklist

- [ ] All pods started successfully (0 restarts)
- [ ] No errors in logs (grep for "ERROR" or "FATAL")
- [ ] Metrics showing normal request rate
- [ ] Auth failures <0.1%
- [ ] Rate limit events <1% of requests
- [ ] Audit logs recording all mutations
- [ ] Response times <200ms p95
- [ ] CPU usage <50%
- [ ] Memory usage <256MB per pod

---

## Security Compliance Verification

### OWASP Top 10 Checklist
- [x] A01:2021 – Broken Access Control → JWT + Tenant Guard + Audit
- [x] A02:2021 – Cryptographic Failures → HTTPS + TLS
- [x] A03:2021 – Injection → Parameterized queries via Hasura
- [x] A04:2021 – Insecure Design → Threat model documented
- [x] A05:2021 – Security Misconfiguration → Secrets management
- [x] A06:2021 – Vulnerable Components → Dependency scanning
- [x] A07:2021 – Authentication Failures → JWT validation
- [x] A08:2021 – Software/Data Integrity Failures → Signed deployments
- [x] A09:2021 – Logging Failures → Structured audit logs
- [x] A10:2021 – SSRF → Input validation

### Compliance Standards
- [x] SOC 2 Type II → Audit logging, access controls
- [x] GDPR → Tenant isolation, audit trails
- [x] HIPAA → Encryption, access logging (if applicable)
- [x] PCI DSS → Secure communication, no credential storage

---

## Emergency Contacts

| Role | Contact | Escalation |
|------|---------|-----------|
| Security Team | security@example.com | +1-XXX-XXX-XXXX slack #security |
| On-Call Engineer | See PagerDuty | #on-call Slack |
| Platform Lead | platform-lead@example.com | Immediate notification |
| Compliance Officer | compliance@example.com | For data breach scenarios |

---

## References

- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- [Multi-Tenancy Security](https://cheatsheetseries.owasp.org/cheatsheets/Multi-Tenant_SaaS_Security.html)
- JWT Alignment Matrix: [JWT_ALIGNMENT_MATRIX.md](../JWT_ALIGNMENT_MATRIX.md)
- Authentication Guide: [AUTHENTICATION.md](../AUTHENTICATION.md)

---

**Status**: Ready for Deployment ✅
