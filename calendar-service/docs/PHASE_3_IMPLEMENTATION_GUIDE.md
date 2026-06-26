# Phase 3: Service & Repository Layer Integration - IMPLEMENTATION GUIDE

**Date:** February 17, 2026  
**Status:** In Progress  
**Scope:** Service layer tenant context, repository tenant filtering, integration tests  

---

## Overview

Phase 3 integrates tenant context from JWT (extracted in handlers) through the service and repository layers, ensuring data isolation at every level:

```
Handler              Service              Repository          Database
─────────────────────────────────────────────────────────────────────
Extract JWT   →   Verify Tenant   →   Filter by Tenant   →  Query with WHERE tenant_id
userID                                                 
tenantID     →   Propagate Context →  Enforce Isolation  → Guarantee Isolation
roles        →   Audit Log         →  Log Access        → Audit Trail
```

---

## Key Principles

### 1. Trust Boundary
- ✅ **Handler Layer**: Trust JWT context (cryptographically verified)
- ✅ **Service Layer**: Accept tenantID as parameter, verify in logic
- ✅ **Repository Layer**: Always filter by tenantID in queries
- ✅ **Database Layer**: Constraint enforces tenant isolation

### 2. Tenant Isolation Layers
```
Layer 1: Application Logic (Service)
  - Verify tenantID provided
  - Validate user has access
  
Layer 2: Repository Logic (Queries)
  - Always add WHERE tenant_id = $1
  - Cannot override for any reason
  
Layer 3: Database (Schema)
  - Foreign key: tenant_id (cannot be null)
  - Index on (tenant_id, id) for performance
  
Layer 4: Row-Level Security (RLS)
  - PostgreSQL policy: WHERE tenant_id = current_setting('tenant.id')
```

### 3. New Method Signatures

**Before Phase 3:**
```go
func (s *CalendarService) Create(ctx context.Context, name string, ...) (*Calendar, error)
// Problem: Tenant assumed from somewhere, no explicit context
```

**After Phase 3:**
```go
func (s *CalendarService) Create(ctx context.Context, tenantID, userID, name string, ...) (*Calendar, error)
// Better: Explicit parameters, can't forget tenant
```

---

## Implementation Checklist

### ✅ Service Layer

#### 1. Tenant-Aware CRUD Methods

```go
// ✅ Pattern: Tenant as first parameter after context
func (s *CalendarService) Create(ctx context.Context, tenantID, userID, name string) (*Calendar, error)
func (s *CalendarService) GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error)
func (s *CalendarService) ListByTenant(ctx context.Context, tenantID string) ([]Calendar, error)
func (s *CalendarService) Update(ctx context.Context, tenantID, calendarID, userID string, updates map[string]interface{}) (*Calendar, error)
func (s *CalendarService) Delete(ctx context.Context, tenantID, calendarID, userID string) error
```

#### 2. Tenant Validation

```go
// ✅ Verify resource belongs to tenant
func (s *CalendarService) verifyCalendarAccess(ctx context.Context, tenantID, calendarID string) error {
    calendar, err := s.repo.GetByID(ctx, calendarID)
    if err != nil {
        return err
    }
    if calendar.TenantID != tenantID {
        s.logger.WithFields(logrus.Fields{
            "tenant_id": tenantID,
            "calendar_id": calendarID,
            "resource_tenant": calendar.TenantID,
        }).Warn("Cross-tenant access denied")
        return ErrAccessDenied
    }
    return nil
}
```

#### 3. Audit Context

```go
// ✅ Set audit context before repository calls
s.auditCtx = setAuditContext(ctx, map[string]interface{}{
    "tenant_id": tenantID,
    "user_id": userID,
    "action": "create_calendar",
})
```

### ✅ Repository Layer

#### 1. Tenant-Scoped Queries

```go
// ✅ EVERY query must filter by tenant
func (r *calendarsRepo) ListByTenant(ctx context.Context, tenantID string) ([]Calendar, error) {
    query := `
    SELECT id, tenant_id, name, created_at
    FROM calendars
    WHERE tenant_id = $1 AND deleted_at IS NULL
    ORDER BY created_at DESC
    `
    // Tenant is explicit in WHERE clause
}

// ❌ NEVER do this:
func (r *calendarsRepo) List(ctx context.Context) ([]Calendar, error) {
    query := `SELECT * FROM calendars`
    // This could return data from ALL tenants!
}
```

#### 2. Query Builder Pattern

```go
// ✅ Helper to build tenant-scoped WHERE clause
func (r *calendarsRepo) tenantCondition(tenantID string) string {
    return fmt.Sprintf("tenant_id = '%s'", 
        strings.NewReplacer("'", "''").Replace(tenantID))
}

func (r *calendarsRepo) GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error) {
    query := fmt.Sprintf(`
    SELECT * FROM calendars
    WHERE id = $1 AND %s
    `, r.tenantCondition(tenantID))
    
    // Even with ID, verify tenant
}
```

#### 3. Batch Operations

```go
// ✅ List with tenant filter
func (r *calendarsRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Calendar, error) {
    // All rows guaranteed to have tenant_id = tenantID
}

// ✅ Batch update with tenant verification
func (r *calendarsRepo) UpdateByTenant(ctx context.Context, tenantID string, calendarID string, updates map[string]interface{}) error {
    // Update only rows where tenant_id matches
}
```

### ✅ Handler Integration

#### 1. Service Injection

```go
type CalendarHandler struct {
    logger *logrus.Entry
    service *CalendarService  // ← Injected
}

func NewCalendarHandler(logger *logrus.Entry, service *CalendarService) *CalendarHandler {
    return &CalendarHandler{
        logger: logger,
        service: service,
    }
}
```

#### 2. Context Propagation

```go
func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)
    
    var req CreateCalendarRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        return
    }
    
    // ✅ Pass explicit tenant context to service
    calendar, err := h.service.Create(
        ctx,
        tenantID,      // From JWT ✅
        userID,        // From JWT ✅
        req.Name,
    )
    if err != nil {
        return
    }
    
    json.NewEncoder(w).Encode(calendar)
}
```

### ✅ Integration Tests

#### 1. Test Pattern

```go
func TestCreateCalendarWithTenant(t *testing.T) {
    // Arrange
    tenantID := "tenant-123"
    userID := "user-456"
    
    service := NewCalendarService(repo, logger)
    
    // Act
    calendar, err := service.Create(
        context.Background(),
        tenantID,
        userID,
        "Test Calendar",
    )
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, tenantID, calendar.TenantID)
    assert.Equal(t, userID, calendar.CreatedBy)
}
```

#### 2. Cross-Tenant Prevention

```go
func TestCrossTenantAccessDenied(t *testing.T) {
    // User from Tenant A tries to access Tenant B
    calendar := createTestCalendarForTenant("tenant-b")
    
    _, err := service.GetByID(
        context.Background(),
        "tenant-a",  // Different tenant
        calendar.ID,
    )
    
    assert.Error(t, err)
    assert.Equal(t, ErrAccessDenied, err)
}
```

---

## Caching Strategy (Redis)

### Cache Key Scoping

**Before Phase 3:**
```go
// ❌ Problem: Multi-tenant collisions
cacheKey := fmt.Sprintf("calendars:%s", calendarID)
// Could get cross-tenant cache hit

cacheKey := fmt.Sprintf("calendars:user-%s", userID)
// Doesn't isolate by tenant
```

**After Phase 3:**
```go
// ✅ Solution: Tenant-first keys
cacheKey := fmt.Sprintf("calendars:tenant-%s:id-%s", tenantID, calendarID)
// Guaranteed no cross-tenant collisions

cacheKey := fmt.Sprintf("calendars:tenant-%s:user-%s", tenantID, userID)
// Full tenant scope
```

### Implementation

```go
type CacheKey struct {
    TenantID   string
    ResourceID string
    Suffix     string
}

func (k CacheKey) String() string {
    return fmt.Sprintf("calendars:tenant-%s:id-%s:%s",
        k.TenantID, k.ResourceID, k.Suffix)
}

func (s *CalendarService) getFromCache(ctx context.Context, tenantID, calendarID string) (*Calendar, error) {
    key := CacheKey{
        TenantID:   tenantID,
        ResourceID: calendarID,
    }.String()
    
    // Cache hit guaranteed to be same tenant
    return s.cache.Get(ctx, key)
}
```

---

## Error Handling

### Tenant Isolation Errors

```go
// ✅ New error types
var (
    ErrAccessDenied = errors.New("access denied: cross-tenant access")
    ErrNotFound = errors.New("resource not found or access denied")
    ErrTenantMismatch = errors.New("tenant_id mismatch")
    ErrTenantRequired = errors.New("tenant_id is required")
)

// ✅ Return same error for not-found and access-denied
// Don't leak whether resource exists in another tenant
func (s *CalendarService) GetByID(ctx context.Context, tenantID, calendarID string) (*Calendar, error) {
    if calendar.TenantID != tenantID {
        // Return NotFound, not AccessDenied
        // Client shouldn't know if calendar exists
        return nil, ErrNotFound
    }
}
```

---

## Audit Logging

### Audit Events

```go
// ✅ Every operation logs tenant context
auditEvent := map[string]interface{}{
    "timestamp": time.Now(),
    "user_id": userID,
    "tenant_id": tenantID,
    "action": "create_calendar",
    "resource_type": "calendar",
    "resource_id": calendarID,
    "status": "success",
    "details": map[string]interface{}{
        "name": calendarName,
        "region": region,
    },
}

service.logAudit(ctx, auditEvent)
```

### Audit Storage

```go
// ✅ Audit logs also partitioned by tenant
func (r *auditLogRepo) CreateAuditLog(ctx context.Context, tenantID string, event map[string]interface{}) error {
    query := `
    INSERT INTO audit_logs (tenant_id, user_id, action, details, created_at)
    VALUES ($1, $2, $3, $4, $5)
    `
    // tenant_id is indexed for per-tenant audit queries
}
```

---

## Data Flow: Complete End-to-End

```
1. Client Request
   POST /api/v1/calendars
   Authorization: Bearer <JWT>
   X-Tenant-ID: tenant-123
   {"name": "Q1 Calendar"}
   
   ↓

2. Router → JWTMiddleware
   ✅ Validates signature
   ✅ Adds to context: user_id, tenant_id
   
   ↓

3. Handler: CalendarHandler.Create(w, r)
   ✅ Extracts userID, tenantID from context
   ✅ Parses request body (only name)
   
   ✅ Calls: service.Create(ctx, tenantID, userID, name)
   
   ↓

4. Service: CalendarService.Create()
   ✅ Validates tenantID is provided
   ✅ Generates calendar ID
   ✅ Calls: repo.Create(ctx, calendar)
   ✅ Logs audit event with tenant_id, user_id
   
   ↓

5. Repository: calendarsRepo.Create()
   ✅ Builds query with WHERE tenant_id = $1
   ✅ Inserts calendar
   ✅ Returns calendar with tenant_id verified
   
   ↓

6. Database
   INSERT INTO calendars (id, tenant_id, name, created_by, created_at)
   VALUES ($1, $2, $3, $4, $5)
   
   ← DB constraint: tenant_id cannot be null
   ← DB index: (tenant_id, id) for performance
   
   ↓

7. Response
   201 Created
   {
     "id": "cal-123",
     "tenant_id": "tenant-123",
     "name": "Q1 Calendar",
     "created_by": "user-456",
     "created_at": "2026-02-17T10:30:00Z"
   }
   
   ↓

8. Audit Log
   {
     "timestamp": "2026-02-17T10:30:00Z",
     "tenant_id": "tenant-123",
     "user_id": "user-456",
     "action": "create_calendar",
     "resource_id": "cal-123",
     "status": "success"
   }
```

---

## Testing Architecture

### Unit Tests
```go
// Mock repository with tenant verification
type mockRepo struct {
    tenantID string  // Enforce single tenant per test
}

func (m *mockRepo) Create(ctx context.Context, calendar *Calendar) error {
    if calendar.TenantID != m.tenantID {
        return ErrTenantMismatch  // Fail if mismatch
    }
    return nil
}

// Service test verifies tenant context flows through
```

### Integration Tests
```go
// Real PostgreSQL in test container
// Real Redis cache
// Verify full stack with tenant isolation

func TestFullStackCalendarCreation(t *testing.T) {
    // Create calendar for tenant-A
    // Verify cannot be accessed by tenant-B
    // Verify cache is scoped to tenant-A
}

func TestCrossTenantCachePrevention(t *testing.T) {
    // Put calendar in cache for tenant-A
    // Request from tenant-B must get different result
    // Cache key must include tenant_id
}
```

---

## Configuration & Environment

### Required Environment Variables
```bash
# Tenant repository config
TENANT_REPO_TYPE=postgres  # or "hasura"
DATABASE_URL=postgresql://...

# Cache config
REDIS_URL=redis://localhost:6379
CACHE_TTL=300  # seconds
CACHE_KEY_PREFIX=calendar-service:

# Audit logging
AUDIT_LOG_LEVEL=INFO
AUDIT_LOG_STORAGE=postgres  # or "elasticsearch"
```

---

## Performance Considerations

### Indexes for Tenant Isolation
```sql
-- Tenant-scoped lookups
CREATE INDEX idx_calendars_tenant_id ON calendars(tenant_id);

-- Tenant + ID lookups (common pattern)
CREATE INDEX idx_calendars_tenant_id_id ON calendars(tenant_id, id);

-- Tenant + created_at (list queries)
CREATE INDEX idx_calendars_tenant_created ON calendars(tenant_id, created_at DESC);

-- Audit log tenant queries
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
```

### Query Optimization
```go
// ✅ Good: Single query with tenant filter
SELECT * FROM calendars WHERE tenant_id = $1 AND id = $2 LIMIT 1

// ❌ Bad: Query all then filter in application
SELECT * FROM calendars
rows.filter(func(c *Calendar) bool {
    return c.TenantID == tenantID
})
```

---

## Security Checklist

- [ ] All service methods have tenantID parameter
- [ ] All repository queries include tenant filter
- [ ] Cannot accidentally query cross-tenant data
- [ ] Audit logs include tenant_id
- [ ] Cache keys include tenant_id
- [ ] Errors don't leak cross-tenant existence
- [ ] Tests verify cross-tenant prevention
- [ ] Integration tests use real database
- [ ] Performance indexes created
- [ ] Deployment verified with real JWTs

---

## Deployment Strategy

### Pre-Deployment
1. Verify all queries include tenant filter
2. Run cross-tenant access tests
3. Migrate audit logs to new schema
4. Warm up cache with tenant-scoped keys
5. Configure PostgreSQL RLS policies

### Deployment
1. Deploy code with service layer
2. Verify audit logs flowing
3. Monitor cross-tenant access attempts (should be zero)
4. Verify cache key format

### Post-Deployment
1. Verify all operations have tenant context
2. Audit logs show user_id, tenant_id
3. No cross-tenant data access
4. Performance meets SLA with indexes
5. Cache hit rate >80% (tenant-scoped)

---

## Next Steps

### Immediate (Day 1)
1. [ ] Add tenantID parameter to all service methods
2. [ ] Update all repository queries with tenant filter
3. [ ] Wire handlers to pass tenant context to services
4. [ ] Add tenant validation in service layer

### Short-term (Week 1)
1. [ ] Create integration tests
2. [ ] Implement cross-tenant prevention tests
3. [ ] Add audit logging
4. [ ] Implement cache key scoping
5. [ ] Create deployment runbook

### Medium-term (Week 2)
1. [ ] Performance testing with tenant queries
2. [ ] Optimize indexes
3. [ ] Load testing
4. [ ] Security audit
5. [ ] Documentation

---

## Success Criteria

✅ **Data Isolation**: No data accessible across tenant boundaries  
✅ **Audit Trail**: All operations logged with user/tenant context  
✅ **Performance**: Query response <100ms with tenant filter  
✅ **Security**: Cross-tenant access tests pass 100%  
✅ **Compliance**: Audit logs meet compliance requirements  
✅ **Testing**: >90% test coverage of tenant isolation paths  
✅ **Deployment**: Process documented and tested  

---

**Status:** Ready for Implementation  
**Estimated Effort:** 3-4 days for full implementation + testing  
**Risk Level:** Medium (requires careful tenant context threading)  
**Rollback Plan:** Data stays scoped to tenant regardless, easy rollback  

