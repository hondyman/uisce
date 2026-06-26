# Security Subsystem - Implementation Checklist

Use this checklist to track integration and deployment progress.

## Phase 1: Code Integration ✅ COMPLETE

- [x] Create backend API handlers
- [x] Create service layer
- [x] Create repository layer
- [x] Create DSL parser
- [x] Create impact analyzer
- [x] Create data models
- [x] Create Temporal workflow
- [x] Create Temporal activities
- [x] Create frontend pages
- [x] Create frontend components
- [x] Create API client
- [x] Write unit tests
- [x] Write documentation
- [x] Create OpenAPI spec

## Phase 2: Backend Integration 🔧 TODO

### Database Setup
- [ ] Run database migration to create `access_rule` table
- [ ] Create indexes (tenant, BO, group, status)
- [ ] Verify JSONB support for column_masks
- [ ] Seed default rules from `seed_access_rules.sql`

### Code Integration
- [ ] Fix import cycle (move Principal to separate package)
- [ ] Wire SecurityRulesHandler into main router
- [ ] Add Principal extraction middleware
- [ ] Connect to existing LDAP/AD integration
- [ ] Initialize cache (Redis or in-memory)
- [ ] Add metrics instrumentation

### Configuration
- [ ] Add security config section to `config.yaml`
  ```yaml
  security:
    cache:
      ttl: 15m
      type: redis  # or "memory"
      redis_url: redis://localhost:6379
    ldap:
      server: ldap://ldap.example.com
      base_dn: dc=example,dc=com
  ```

## Phase 3: Frontend Integration 🎨 TODO

- [ ] Verify routes are wired under `/security/access-rules`
- [ ] Test navigation from security menu
- [ ] Fix any TypeScript errors
- [ ] Add to main navigation menu (if not already there)
- [ ] Style consistency check with rest of app
- [ ] Responsive design verification

## Phase 4: Temporal Workflow Setup ⚙️ TODO (Optional)

- [ ] Deploy Temporal server (if not already running)
- [ ] Create `security-workflows` task queue
- [ ] Register workflow and activities with worker
- [ ] Start Temporal worker process
- [ ] Test workflow execution
- [ ] Set up workflow monitoring

## Phase 5: Testing 🧪 TODO

### Unit Tests
- [ ] Run Go unit tests: `go test ./internal/services -v`
- [ ] Run DSL parser tests: `go test ./internal/dsl -v`
- [ ] Achieve >80% coverage on service layer

### Integration Tests
- [ ] Create rule via API → verify in DB
- [ ] Update rule → verify version increment
- [ ] Validate DSL with valid expression → success
- [ ] Validate DSL with invalid expression → error
- [ ] Get impact for rule → verify artifact list
- [ ] Delete rule → verify removal

### End-to-End Tests
- [ ] User with single group queries BO → correct filtering
- [ ] User with multiple groups → correct composition
- [ ] User not in any rule groups → forbidden
- [ ] Row predicate correctly filters results
- [ ] Column masks correctly hide fields
- [ ] Column masks correctly obfuscate values
- [ ] Scope filtering works (API/BI/AI)
- [ ] Cache hit on second query

### Performance Tests
- [ ] Load test: 1000 concurrent queries with security
- [ ] Measure cache hit ratio (target >90%)
- [ ] Measure rule resolution latency (target <10ms cached)
- [ ] Measure SQL rewrite overhead (target <5ms)
- [ ] Profile memory usage with large rule sets

## Phase 6: Security Review 🔒 TODO

- [ ] Code review by security team
- [ ] SQL injection prevention audit (DSL parser)
- [ ] Authorization bypass testing
- [ ] Cache poisoning prevention review
- [ ] LDAP injection prevention check
- [ ] Audit logging completeness verification
- [ ] Penetration testing

## Phase 7: Documentation & Training 📚 TODO

### Documentation
- [ ] Update main README with security section
- [ ] Document group naming conventions
- [ ] Create runbook for common scenarios
- [ ] Document troubleshooting steps
- [ ] Add API examples to developer docs

### Training
- [ ] Train security admins on rule creation
- [ ] Train developers on enforcement model
- [ ] Train ops on monitoring and metrics
- [ ] Create demo video walkthrough
- [ ] Hold Q&A session

## Phase 8: Staging Deployment 🚀 TODO

- [ ] Deploy to staging environment
- [ ] Run smoke tests
- [ ] Verify metrics collection
- [ ] Verify logging
- [ ] Test workflow promotion
- [ ] Performance testing in staging
- [ ] Load testing in staging

## Phase 9: Production Deployment 🎯 TODO

### Pre-Deployment
- [ ] Final security review sign-off
- [ ] Final code review sign-off
- [ ] Backup current production DB
- [ ] Create rollback plan
- [ ] Schedule maintenance window
- [ ] Notify stakeholders

### Deployment
- [ ] Deploy database schema
- [ ] Deploy backend services
- [ ] Deploy frontend assets
- [ ] Deploy Temporal workers
- [ ] Run smoke tests
- [ ] Monitor for errors

### Post-Deployment
- [ ] Verify metrics are flowing
- [ ] Check error rates
- [ ] Monitor cache hit ratio
- [ ] Verify audit logs
- [ ] Gradual rollout to users
- [ ] Monitor performance

## Phase 10: Monitoring & Operations 📊 TODO

### Metrics Setup
- [ ] Set up dashboards for:
  - `security_rule_resolution_ms`
  - `security_sql_rewrite_ms`
  - `security_cache_hit_ratio`
  - `security_filtered_rows`
  - `security_forbidden_requests`

### Alerting
- [ ] Alert: Cache hit ratio < 90%
- [ ] Alert: Rule resolution latency > 100ms (p99)
- [ ] Alert: High rate of forbidden requests
- [ ] Alert: Workflow failures
- [ ] Alert: Database connection errors

### Maintenance
- [ ] Schedule: Weekly cache statistics review
- [ ] Schedule: Monthly rule audit
- [ ] Schedule: Quarterly performance review
- [ ] Schedule: Quarterly security audit

## Success Metrics 🎉

Track these to measure success:

### Performance
- [ ] Cache hit ratio > 90%
- [ ] Rule resolution < 10ms (p95, cached)
- [ ] SQL rewrite overhead < 5ms (p95)
- [ ] No user-visible latency impact

### Security
- [ ] Zero unauthorized data access incidents
- [ ] 100% of BO queries go through enforcement
- [ ] All rule changes audited
- [ ] Zero SQL injection vulnerabilities

### Usability
- [ ] Security admins can create rules without developer help
- [ ] Average rule creation time < 5 minutes
- [ ] < 5% rule validation errors
- [ ] Positive feedback from security team

### Reliability
- [ ] 99.9% uptime for security service
- [ ] < 0.1% failed workflow promotions
- [ ] Zero data exposure incidents
- [ ] Successful disaster recovery drill

## Quick Start Commands

### Backend
```bash
# Run tests
cd backend
go test ./internal/services -v
go test ./internal/dsl -v

# Build
go build -o server ./cmd/server

# Run
./server
```

### Frontend
```bash
# Install deps
cd frontend
pnpm install

# Lint
pnpm lint

# Build
pnpm build

# Dev
pnpm dev
```

### Database
```bash
# Run migration
psql -U postgres -d alpha -f backend/migrations/001_create_access_rule_table.sql

# Seed data
psql -U postgres -d alpha -f backend/migrations/misc/seed_access_rules.sql
```

### Temporal
```bash
# Start worker
go run ./cmd/temporal-worker

# Test workflow
go run ./cmd/test-promotion-workflow
```

## Contacts

- **Security Lead:** [Name] - [email]
- **Backend Lead:** [Name] - [email]
- **Frontend Lead:** [Name] - [email]
- **DevOps Lead:** [Name] - [email]
- **Product Owner:** [Name] - [email]

## Resources

- [SECURITY_SUBSYSTEM_README.md](backend/SECURITY_SUBSYSTEM_README.md) - Complete implementation guide
- [SECURITY_QUICK_REFERENCE.md](backend/SECURITY_QUICK_REFERENCE.md) - Quick reference card
- [SECURITY_ARCHITECTURE_DIAGRAM.md](SECURITY_ARCHITECTURE_DIAGRAM.md) - Architecture diagrams
- [INTEGRATION_GUIDE.go](backend/INTEGRATION_GUIDE.go) - Code integration examples
- [openapi-security-rules.yaml](backend/api/openapi-security-rules.yaml) - API specification

## Notes

Use this section to track decisions, issues, or deviations from plan:

```
[Date] [Initials] - [Note]

Example:
2026-01-22 JD - Decided to use Redis cache instead of in-memory for multi-instance support
2026-01-23 SK - Import cycle fixed by moving Principal to internal/auth package
```
