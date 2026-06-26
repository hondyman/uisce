# API Endpoints Catalog: Deployment Checklist

## Pre-Deployment

### Database Setup
- [ ] Apply migration: `001_create_api_endpoints_catalog.sql`
  ```bash
  psql postgres://postgres:postgres@localhost:5432/alpha < backend/internal/api/migrations/001_create_api_endpoints_catalog.sql
  ```
- [ ] Verify tables created:
  ```sql
  SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename LIKE 'api_%';
  ```
- [ ] Verify indexes:
  ```sql
  SELECT * FROM pg_indexes WHERE tablename LIKE 'api_%';
  ```

### Backend Implementation
- [ ] File: `backend/internal/api/api_endpoints_catalog.go`
  - [ ] CRUD endpoints implemented
  - [ ] Pagination support added
  - [ ] Search and filtering working
  - [ ] OpenAPI spec generation working

- [ ] File: `backend/internal/api/api_endpoint_mapping_routes.go`
  - [ ] Entity mapping endpoints working
  - [ ] Datasource mapping endpoints working
  - [ ] Reverse lookup endpoints functional

- [ ] File: `backend/internal/api/api_endpoints_seeder.go`
  - [ ] Validation rule endpoints seeded
  - [ ] Automatic mappings created
  - [ ] No duplicate seeding

### API Router Registration
- [ ] In main API initialization code:
  ```go
  api.RegisterAPIEndpointsCatalogRoutes(r, db)
  api.RegisterEndpointMappingRoutes(r, db)
  ```

- [ ] Seed on startup:
  ```go
  if err := api.SeedAPIEndpointsCatalog(db, tenantID); err != nil {
    log.Printf("Warning: Failed to seed catalog: %v", err)
  }
  ```

### API Testing
- [ ] Test GET `/api-endpoints` - List endpoints
  ```bash
  curl -X GET "http://localhost:8080/api-endpoints?tenant_id=<TENANT_ID>" \
    -H "X-Tenant-ID: <TENANT_ID>"
  ```

- [ ] Test POST `/api-endpoints` - Create endpoint
  ```bash
  curl -X POST "http://localhost:8080/api-endpoints?tenant_id=<TENANT_ID>" \
    -H "Content-Type: application/json" \
    -d '{"endpoint_name":"Test","http_method":"GET","url_path":"/test",...}'
  ```

- [ ] Test GET `/api-endpoints/category/validation` - Filter by category
- [ ] Test GET `/api-endpoints/search` - Search functionality
- [ ] Test entity mappings endpoints
- [ ] Test datasource mappings endpoints
- [ ] Test reverse lookup: GET `/entities/{id}/api-endpoints`

### Frontend Implementation
- [ ] Create file: `frontend/src/services/validationRulesService.ts`
  - [ ] API client initialization
  - [ ] All CRUD methods implemented
  - [ ] Error handling in place
  - [ ] Type definitions complete

- [ ] Update file: `frontend/src/pages/EntityDetailsPage.tsx`
  - [ ] Import ValidationRulesService
  - [ ] Add useEffect for loading rules
  - [ ] Implement create/update/delete handlers
  - [ ] Add execute rule functionality
  - [ ] Error states displayed
  - [ ] Loading indicators shown

- [ ] Update file: `frontend/src/pages/EntityConfigPageV2.tsx`
  - [ ] Add validation tab
  - [ ] Pass service props to container
  - [ ] Tab routing working

### Frontend Testing
- [ ] Test listing validation rules
  - [ ] Rules load on page open
  - [ ] Pagination works
  - [ ] Filters apply correctly
  - [ ] No console errors

- [ ] Test creating validation rule
  - [ ] Form displays correctly
  - [ ] Data submitted to backend
  - [ ] New rule appears in list
  - [ ] Success feedback shown

- [ ] Test updating validation rule
  - [ ] Edit form populates correctly
  - [ ] Changes saved to backend
  - [ ] List updates

- [ ] Test deleting validation rule
  - [ ] Confirmation dialog shown
  - [ ] Rule deleted from backend
  - [ ] List updates immediately

- [ ] Test executing validation rule
  - [ ] Execution completes
  - [ ] Results displayed
  - [ ] Errors handled gracefully

- [ ] Test error handling
  - [ ] Network errors shown to user
  - [ ] Validation errors displayed
  - [ ] Auto-retry logic (if implemented)

### Tenant Scope Testing
- [ ] Test without tenant selection
  - [ ] Requests blocked
  - [ ] User directed to select tenant
  - [ ] Clear error message

- [ ] Test with tenant scope set
  - [ ] Query parameters included: `?tenant_id=...&datasource_id=...`
  - [ ] Headers included: `X-Tenant-ID`, `X-Tenant-Datasource-ID`
  - [ ] Data isolated by tenant

- [ ] Test cross-tenant isolation
  - [ ] Cannot access other tenant's rules
  - [ ] Cannot create rules for other tenants
  - [ ] No data leakage

### Documentation
- [ ] `BACKEND_API_CATALOG_INTEGRATION.md` complete
- [ ] `FRONTEND_VALIDATION_RULES_INTEGRATION.md` complete
- [ ] API examples documented
- [ ] Error codes documented
- [ ] Schema examples provided

## Deployment

### Staging Environment
```bash
# 1. Deploy database migrations
psql $STAGING_DB_URL < backend/internal/api/migrations/001_create_api_endpoints_catalog.sql

# 2. Verify migration
psql $STAGING_DB_URL -c "SELECT COUNT(*) FROM api_endpoints_catalog;"

# 3. Deploy backend changes
git checkout staging
git pull origin staging
go build ./...
docker build -t semlayer-backend:staging .
docker push semlayer-backend:staging

# 4. Deploy frontend changes
cd frontend
npm run build
docker build -t semlayer-frontend:staging .
docker push semlayer-frontend:staging

# 5. Update services
kubectl set image deployment/semlayer-backend \
  backend=semlayer-backend:staging -n staging

kubectl set image deployment/semlayer-frontend \
  frontend=semlayer-frontend:staging -n staging

# 6. Monitor deployment
kubectl rollout status deployment/semlayer-backend -n staging
kubectl rollout status deployment/semlayer-frontend -n staging
```

### Production Deployment
```bash
# 1. Tag release
git tag -a v1.5.0 -m "API Endpoints Catalog integration"
git push origin v1.5.0

# 2. Build artifacts
docker build -t semlayer-backend:v1.5.0 .
docker push semlayer-backend:v1.5.0

docker build -t semlayer-frontend:v1.5.0 -f Dockerfile.frontend .
docker push semlayer-frontend:v1.5.0

# 3. Apply migrations (with backup first!)
pg_dump $PROD_DB_URL > backup_pre_migration_v1.5.0.sql
psql $PROD_DB_URL < backend/internal/api/migrations/001_create_api_endpoints_catalog.sql

# 4. Blue-green deployment
# Keep old version running, deploy new version alongside
# Test new version endpoints
# Switch load balancer traffic to new version
# Keep old version as rollback option for 30 minutes

# 5. Monitor
kubectl logs -f deployment/semlayer-backend -n production
kubectl logs -f deployment/semlayer-frontend -n production

# 6. Verify
curl -X GET "https://api.semlayer.com/api-endpoints?tenant_id=..." \
  -H "X-Tenant-ID: ..."
```

## Post-Deployment

### Verification
- [ ] All endpoints responding correctly
- [ ] Validation rules visible in UI
- [ ] Create/update/delete operations working
- [ ] Execute rule functionality operational
- [ ] Audit trail recording changes
- [ ] Performance metrics normal
  - [ ] P95 latency < 500ms
  - [ ] Error rate < 0.1%

### Monitoring Setup
- [ ] Logs aggregation enabled
- [ ] Metrics collection configured
  - [ ] API endpoint hit counts
  - [ ] Error rates by endpoint
  - [ ] Response times
- [ ] Alerts configured
  - [ ] High error rate (> 5%)
  - [ ] Slow responses (P95 > 2s)
  - [ ] Database connection pool exhaustion

### Rollback Plan
```bash
# If issues detected, rollback:
kubectl rollout undo deployment/semlayer-backend -n production
kubectl rollout undo deployment/semlayer-frontend -n production

# Verify rollback
kubectl rollout status deployment/semlayer-backend -n production
kubectl rollout status deployment/semlayer-frontend -n production

# If database rollback needed
psql $PROD_DB_URL < backup_pre_migration_v1.5.0.sql
```

## Validation Checklist

### Functionality
- [ ] ✅ List validation rules works
- [ ] ✅ Create validation rule works
- [ ] ✅ Edit validation rule works
- [ ] ✅ Delete validation rule works
- [ ] ✅ Execute validation rule works
- [ ] ✅ Audit trail visible
- [ ] ✅ API endpoint catalog populated
- [ ] ✅ Entity mappings created
- [ ] ✅ Reverse lookups working

### Performance
- [ ] ✅ List endpoint: < 200ms (50 rules)
- [ ] ✅ Create endpoint: < 500ms
- [ ] ✅ Search endpoint: < 300ms
- [ ] ✅ Execute endpoint: < 2s
- [ ] ✅ Batch execute: < 5s (100 records)

### Reliability
- [ ] ✅ Tenant isolation verified
- [ ] ✅ No data leakage between tenants
- [ ] ✅ Error handling working
- [ ] ✅ Graceful degradation on DB issues
- [ ] ✅ Automatic retries functional

### Security
- [ ] ✅ Authentication required on all endpoints
- [ ] ✅ Tenant scope validated
- [ ] ✅ Authorization checks in place
- [ ] ✅ SQL injection prevention (parameterized queries)
- [ ] ✅ Rate limiting configured

### Documentation
- [ ] ✅ API endpoints documented
- [ ] ✅ Error codes documented
- [ ] ✅ Examples provided
- [ ] ✅ Integration guide complete
- [ ] ✅ Deployment guide complete

## Success Criteria

### Functional Success
- Validation Rules tab appears and functions in Entity Manager
- All CRUD operations work from UI
- Backend API catalog shows validation endpoints
- Relationships created between entities and endpoints
- Reverse lookups return appropriate data

### Performance Success
- Initial page load: < 2s
- Rule listing: < 500ms
- Create/update operations: < 1s
- No UI lag or freezing

### User Experience Success
- Clear error messages for failures
- Loading indicators while fetching
- Confirmation dialogs for destructive actions
- Success notifications for operations
- Intuitive navigation and workflows

### Operations Success
- All metrics within SLA
- No critical errors in logs
- Deployment successful with zero downtime
- Rollback capability verified
- Monitoring and alerting working

## Sign-Off

- [ ] **Backend Engineer**: Verified API implementation and database
- [ ] **Frontend Engineer**: Verified UI integration and service layer
- [ ] **QA Engineer**: All tests passing, no regressions
- [ ] **DevOps Engineer**: Deployment successful, monitoring operational
- [ ] **Product Manager**: Feature complete and meets requirements

---

## Quick References

### Key Files
- Backend routes: `backend/internal/api/api_endpoints_catalog.go`
- Mapping routes: `backend/internal/api/api_endpoint_mapping_routes.go`
- Seeding logic: `backend/internal/api/api_endpoints_seeder.go`
- Database migrations: `backend/internal/api/migrations/001_create_api_endpoints_catalog.sql`
- Frontend service: `frontend/src/services/validationRulesService.ts`
- Frontend component: `frontend/src/pages/EntityDetailsPage.tsx`

### Key Endpoints
- List rules: `GET /api-endpoints?tenant_id=...`
- Create rule: `POST /api-endpoints?tenant_id=...`
- Execute rule: `POST /validation-rules/{id}/execute?tenant_id=...`
- Get entity endpoints: `GET /entities/{id}/api-endpoints?tenant_id=...`

### Key Database Tables
- `api_endpoints_catalog` - Endpoint metadata
- `api_endpoint_entity_mappings` - Endpoint-to-entity relationships
- `api_endpoint_datasource_mappings` - Endpoint-to-datasource relationships

### Troubleshooting
| Issue | Solution |
|-------|----------|
| API endpoints not returning data | Check `X-Tenant-ID` header and `tenant_id` parameter |
| Validation tab not showing | Verify EntityConfigPageV2.tsx import statements |
| Rules not saving | Check network tab in DevTools for API errors |
| Performance issues | Verify database indexes are created |
| Tenant isolation issues | Check query filters include `tenant_id` |

