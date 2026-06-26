# Security Scanning Configuration for Calendar Service

## 1. Trivy Configuration (Container Image Scanning)

```yaml
# .trivyignore (trivy-ignore.txt)
# Format: <CVE-ID>,<expiry-date>,<reason>

# Known false positives or acceptable risks
CVE-2023-XXXXX,2026-12-31,accepted risk - requires specific config
```

## 2. OWASP Dependency Check

```bash
#!/bin/bash
# scripts/security-scan.sh

# Scan Go dependencies
go list -json all | docker run \
  -v /dev/stdin:/dev/stdin \
  dependencycheck/dependencycheck:latest \
  dependency-check.sh \
  --scan /dev/stdin \
  --format HTML

# Scan container image
trivy image --severity HIGH,CRITICAL \
  calendar-service:latest

# Scan Kubernetes manifests
kubesec scan k8s/monitoring.yaml
```

## 3. Go Security (gosec)

Common security checks:
- SQL Injection
- Command Injection  
- Insecure Cryptography
- Weak Random Number Generation
- Hardcoded Credentials

Configuration:
```json
{
  "global": {
    "audit": true,
    "tests": [
      "G101", // Look for passwords in code
      "G102", // SQL Injection
      "G103", // Audit unsafe block
      "G202", // SQL strings
      "G301", // Bad file permissions
      "G302", // Bad file permissions
      "G303", // Creating temp file
    ]
  }
}
```

## 4. Secret Scanning (TruffleHog)

Scans for:
- AWS Keys
- GitHub Tokens
- Database Credentials
- API Keys
- Private Keys

## 5. SAST (Static Application Security Testing)

### SonarQube Analysis

Key metrics:
- Code Smells
- Vulnerabilities
- Security Hotspots
- Code Coverage
- Duplications

Thresholds:
```
Vulnerabilities: 0
Code Coverage: > 80%
Duplications: < 5%
```

## 6. Container Security Scanning

### Checklist:

✅ Non-root user (UID 1000)
✅ Read-only filesystem
✅ No privilege escalation
✅ Dropped capabilities (ALL)
✅ Resource limits (CPU, Memory)
✅ Health checks (liveness, readiness)
✅ Network policies configured
✅ Pod security policy enforced
✅ RBAC configured
✅ Secrets encrypted at rest
✅ Service account restrictions
✅ Image signed and verified

## 7. Dependency Audit

### Go Dependencies:

```bash
# Audit for vulnerabilities
go list -json -m all | nancy sleuth

# Check for outdated packages
go outdated
```

### JavaScript/React Dependencies:

```bash
# npm audit
npm audit --production

# Check for vulnerabilities
yarn audit
```

## 8. Configuration Security

### Secrets Management:

✅ JWT_SECRET: Rotated quarterly
✅ Database credentials: In sealed secrets
✅ API keys: In external vault (HashiCorp Vault)
✅ SSH keys: Only in secure storage
✅ No hardcoded credentials

### Environment Variables:

✅ Validated on startup
✅ Not logged in debug output
✅ Encrypted in transit
✅ Rotated regularly

## 9. API Security

### HTTP Security Headers:

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

### CORS Configuration:

```
Access-Control-Allow-Origin: https://trusted-domain.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 86400
Access-Control-Allow-Credentials: true
```

### JWT Validation:

✅ Token signature verified
✅ Expiration checked
✅ Issuer validated
✅ Audience validated
✅ Claims verified

## 10. Database Security

### PostgreSQL Hardening:

```sql
-- User permissions (least privilege)
REVOKE ALL ON DATABASE calendar_service FROM PUBLIC;
GRANT CONNECT ON DATABASE calendar_service TO calendar_user;
GRANT USAGE ON SCHEMA public TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO calendar_user;

-- Row-level security
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON users 
  USING (tenant_id = current_setting('app.tenant_id'));

-- Audit logging
CREATE EXTENSION IF NOT EXISTS pgaudit;
```

## 11. Network Security

### Kubernetes Network Policies:

```yaml
# Deny all ingress by default
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-ingress
spec:
  podSelector: {}
  policyTypes:
  - Ingress

# Allow from nginx-ingress
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-ingress
spec:
  podSelector:
    matchLabels:
      app: calendar-service
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: nginx-ingress
```

## 12. Logging & Monitoring Security

### Sensitive Data Masking:

✅ Passwords never logged
✅ API keys never logged
✅ JWT tokens never logged (except first/last 4 chars)
✅ PII data masked in logs
✅ Database queries logged (without credentials)
✅ All access attempts logged (success and failure)

### Log Retention:

```
Production: 90 days
Staging: 30 days
Development: 7 days
```

## 13. Compliance checks

### GDPR:

- ✅ Data encryption at rest and in transit
- ✅ Audit logs for data access
- ✅ Right to be forgotten (data deletion)
- ✅ Data export functionality
- ✅ Privacy policy compliance

### SOC 2:

- ✅ Access controls (RBAC)
- ✅ Audit trails
- ✅ Change management
- ✅ Incident response procedures
- ✅ Backup and recovery

## 14. Deployment Security Checklist

Before deploying to production:

- [ ] All dependencies scanned and approved
- [ ] No active CVEs with severity > MEDIUM
- [ ] Secrets not committed to git
- [ ] Code reviewed by security team
- [ ] Security tests pass (100%)
- [ ] Penetration testing completed
- [ ] SAST scan clean
- [ ] SCA scan clean
- [ ] Container image scanned
- [ ] Kubernetes manifests validated
- [ ] Network policies configured
- [ ] RBAC configured
- [ ] SSL/TLS certificates valid
- [ ] Monitoring/alerting configured
- [ ] Backup procedures tested
- [ ] Incident response plan reviewed

## 15. Security Testing

### Regular Testing Schedule:

- **Weekly**: Dependency scan (automated)
- **Monthly**: Penetration testing
- **Quarterly**: Security audit
- **Annually**: Third-party security assessment

### Test Scenarios:

```bash
# SQL Injection
curl -X GET "http://localhost:8080/api/v1/calendars?id=1' OR '1'='1"

# XSS Prevention
curl -X POST "http://localhost:8080/api/v1/calendars" \
  -d '{"name": "<script>alert(1)</script>"}'

# CSRF Protection
# Verify CSRF tokens on state-changing operations

# Authentication Bypass
curl -X GET "http://localhost:8080/api/v1/calendars" \
  -H "Authorization: Bearer invalid-token"

# Authorization Bypass
# Verify cross-tenant access is denied
```

## 16. Vulnerability Reporting

Report security problems to: security@example.com

Do NOT:
- Create public GitHub issues
- Post to social media
- Share with unauthorized parties

Do:
- Report privately
- Include reproduction steps
- Allow 90-day disclosure window
- Work with security team

---

**Last Updated**: February 18, 2026  
**Status**: Active  
**Review Frequency**: Monthly
