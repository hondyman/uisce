# Phase 3 Extension: Wiring Handlers to Service Layer

**Purpose:** Practical guide for integrating handlers with the new tenant-aware service layer

**Status:** Ready to implement  
**Estimated Effort:** 2-3 hours  
**Risk Level:** Low (backward compatible)

---

## Overview

### Current State (End of Phase 2)

```
Handler Layer (JWT context extracted)
    ↓
Direct Repository Calls (without service layer)
    ↓
In-Memory Repository (testing) / PostgreSQL (production)
```

### Target State (Phase 3 Extension)

```
Handler Layer (JWT context extracted)
    ↓
Service Layer (Business logic, validation, tenant checks)
    ↓
Repository Layer (Data access with mandatory tenant filters)
    ↓
In-Memory Repository (testing) / PostgreSQL (production)
```

---

## CalendarHandler Integration

### Before (Current)

```go
type CalendarHandler struct {
    repo   repository.CalendarRepository  // Direct repository access
    logger *logrus.Entry
}

func NewCalendarHandler(repo repository.CalendarRepository, logger *logrus.Entry) *CalendarHandler {
    return &CalendarHandler{
        repo:   repo,
        logger: logger,
    }
}

func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    // Parse request
    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Timezone    string `json:"timezone"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Create directly via repository (no service layer!)
    calendar := &repository.Calendar{
        ID:          uuid.New().String(),
        TenantID:    tenantID,
        Name:        req.Name,
        Description: req.Description,
        Timezone:    req.Timezone,
        CreatedBy:   userID,
        CreatedAt:   time.Now(),
    }

    if err := h.repo.Create(ctx, calendar); err != nil {
        http.Error(w, "Failed to create calendar", http.StatusInternalServerError)
        return
    }

    // Response...
}
```

**Problems:**
- ❌ No tenant verification (service layer missing)
- ❌ Business logic duplicated across handlers
- ❌ No consistent error handling
- ❌ Audit logging inconsistent
- ❌ Cache layer can't be easily added

### After (Target)

```go
type CalendarHandler struct {
    service services.CalendarServiceTenantAware  // Service layer injection
    logger  *logrus.Entry
}

func NewCalendarHandler(
    service services.CalendarServiceTenantAware,
    logger *logrus.Entry,
) *CalendarHandler {
    return &CalendarHandler{
        service: service,
        logger:  logger,
    }
}

func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    // Parse request
    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Timezone    string `json:"timezone"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Delegate to service layer (includes validation, audit logging, etc.)
    calendar, err := h.service.Create(
        ctx,
        tenantID,
        userID,
        req.Name,
        req.Description,
        req.Timezone,
    )

    if err != nil {
        // Service layer handles errors consistently
        h.handleServiceError(w, err)
        return
    }

    // Response (service handles audit logging internally)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(calendar)
}
```

**Benefits:**
- ✅ Tenant verification in service layer
- ✅ Audit logging centralized
- ✅ Consistent error handling
- ✅ Cache layer easily added
- ✅ Business logic testable independently

---

## Step-by-Step Integration

### Step 1: Update Handler Constructor

**File:** `internal/api/calendar_handlers.go`

```go
// BEFORE
func NewCalendarHandler(repo repository.CalendarRepository, logger *logrus.Entry) *CalendarHandler {
    return &CalendarHandler{
        repo:   repo,
        logger: logger,
    }
}

// AFTER
func NewCalendarHandler(
    service services.CalendarServiceTenantAware,
    logger *logrus.Entry,
) *CalendarHandler {
    return &CalendarHandler{
        service: service,
        logger:  logger,
    }
}
```

### Step 2: Update CalendarHandler Struct

```go
// BEFORE
type CalendarHandler struct {
    repo   repository.CalendarRepository
    logger *logrus.Entry
}

// AFTER
type CalendarHandler struct {
    service services.CalendarServiceTenantAware
    logger  *logrus.Entry
}
```

### Step 3: Update Create() Method

**Before:**
```go
func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Timezone    string `json:"timezone"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Direct repository call (no service layer!)
    calendar := &repository.Calendar{
        ID:          uuid.New().String(),
        TenantID:    tenantID,
        Name:        req.Name,
        Description: req.Description,
        Timezone:    req.Timezone,
        CreatedBy:   userID,
        CreatedAt:   time.Now(),
    }

    if err := h.repo.Create(ctx, calendar); err != nil {
        h.logger.WithError(err).Error("Failed to create calendar")
        http.Error(w, "Failed to create calendar", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(calendar)
}
```

**After:**
```go
func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Timezone    string `json:"timezone"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // CHANGED: Delegate to service layer (not direct repository!)
    calendar, err := h.service.Create(
        ctx,
        tenantID,
        userID,
        req.Name,
        req.Description,
        req.Timezone,
    )

    if err != nil {
        h.handleServiceError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(calendar)
}
```

### Step 4: Update Get() Method

**Before:**
```go
func (h *CalendarHandler) Get(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    // Parse URL param: /calendars/{id}
    calendarID := r.PathValue("id")
    if calendarID == "" {
        http.Error(w, "Missing calendar ID", http.StatusBadRequest)
        return
    }

    // Direct repository call (no tenant verification!)
    calendar, err := h.repo.GetByID(ctx, calendarID)
    if err != nil {
        http.Error(w, "Calendar not found", http.StatusNotFound)
        return
    }

    // Check tenant (manual verification - duplicated code)
    if calendar.TenantID != tenantID {
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(calendar)
}
```

**After:**
```go
func (h *CalendarHandler) Get(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    // Parse URL param: /calendars/{id}
    calendarID := r.PathValue("id")
    if calendarID == "" {
        http.Error(w, "Missing calendar ID", http.StatusBadRequest)
        return
    }

    // CHANGED: Service layer handles tenant verification
    calendar, err := h.service.GetByID(ctx, tenantID, calendarID)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(calendar)
}
```

### Step 5: Update List() Method

**Before:**
```go
func (h *CalendarHandler) List(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    // Parse pagination: ?limit=10&offset=0
    limit := 10
    offset := 0
    if l := r.URL.Query().Get("limit"); l != "" {
        if val, err := strconv.Atoi(l); err == nil {
            limit = val
        }
    }

    // Direct repository call (no tenant verification!)
    calendars, err := h.repo.ListByTenant(ctx, limit, offset)
    if err != nil {
        http.Error(w, "Failed to list calendars", http.StatusInternalServerError)
        return
    }

    // Manual tenant filtering (should be in service!)
    filtered := []*repository.Calendar{}
    for _, cal := range calendars {
        if cal.TenantID == tenantID {
            filtered = append(filtered, cal)
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "calendars": filtered,
    })
}
```

**After:**
```go
func (h *CalendarHandler) List(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    // Parse pagination: ?limit=10&offset=0
    limit := 10
    offset := 0
    if l := r.URL.Query().Get("limit"); l != "" {
        if val, err := strconv.Atoi(l); err == nil {
            limit = val
        }
    }

    // CHANGED: Service layer only returns tenant's calendars
    calendars, err := h.service.ListByTenant(ctx, tenantID, limit, offset)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "calendars": calendars,
    })
}
```

### Step 6: Update Update() Method

**Before:**
```go
func (h *CalendarHandler) Update(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    calendarID := r.PathValue("id")
    if calendarID == "" {
        http.Error(w, "Missing calendar ID", http.StatusBadRequest)
        return
    }

    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Timezone    string `json:"timezone"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Get current calendar (no service layer!)
    calendar, err := h.repo.GetByID(ctx, calendarID)
    if err != nil {
        http.Error(w, "Calendar not found", http.StatusNotFound)
        return
    }

    // Manual tenant check
    if calendar.TenantID != tenantID {
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }

    // Manual update
    if req.Name != "" {
        calendar.Name = req.Name
    }
    if req.Description != "" {
        calendar.Description = req.Description
    }
    if req.Timezone != "" {
        calendar.Timezone = req.Timezone
    }
    calendar.UpdatedBy = userID
    calendar.UpdatedAt = time.Now()

    if err := h.repo.Update(ctx, calendar); err != nil {
        http.Error(w, "Failed to update calendar", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(calendar)
}
```

**After:**
```go
func (h *CalendarHandler) Update(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    calendarID := r.PathValue("id")
    if calendarID == "" {
        http.Error(w, "Missing calendar ID", http.StatusBadRequest)
        return
    }

    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Timezone    string `json:"timezone"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // CHANGED: Service layer handles tenant verification and update
    updates := map[string]interface{}{}
    if req.Name != "" {
        updates["name"] = req.Name
    }
    if req.Description != "" {
        updates["description"] = req.Description
    }
    if req.Timezone != "" {
        updates["timezone"] = req.Timezone
    }

    calendar, err := h.service.Update(
        ctx,
        tenantID,
        calendarID,
        userID,
        updates,
    )

    if err != nil {
        h.handleServiceError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(calendar)
}
```

### Step 7: Update Delete() Method

**Before:**
```go
func (h *CalendarHandler) Delete(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    calendarID := r.PathValue("id")
    if calendarID == "" {
        http.Error(w, "Missing calendar ID", http.StatusBadRequest)
        return
    }

    // Get calendar (no service layer!)
    calendar, err := h.repo.GetByID(ctx, calendarID)
    if err != nil {
        http.Error(w, "Calendar not found", http.StatusNotFound)
        return
    }

    // Manual tenant check
    if calendar.TenantID != tenantID {
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }

    // Direct delete
    if err := h.repo.Delete(ctx, calendarID); err != nil {
        http.Error(w, "Failed to delete calendar", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

**After:**
```go
func (h *CalendarHandler) Delete(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)

    calendarID := r.PathValue("id")
    if calendarID == "" {
        http.Error(w, "Missing calendar ID", http.StatusBadRequest)
        return
    }

    // CHANGED: Service layer handles tenant verification and soft-delete
    err := h.service.Delete(ctx, tenantID, calendarID, userID)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

### Step 8: Add Error Handler Helper

Add this method to CalendarHandler:

```go
// handleServiceError converts service layer errors to HTTP responses
func (h *CalendarHandler) handleServiceError(w http.ResponseWriter, err error) {
    if err == nil {
        return
    }

    // Map service errors to HTTP status codes
    switch {
    case errors.Is(err, sql.ErrNoRows):
        // Generic not found (doesn't leak cross-tenant info)
        http.Error(w, "Resource not found", http.StatusNotFound)

    case errors.Is(err, context.DeadlineExceeded):
        http.Error(w, "Request timeout", http.StatusGatewayTimeout)

    case errors.Is(err, context.Canceled):
        http.Error(w, "Request canceled", http.StatusBadRequest)

    case errors.Is(err, errAccessDenied):
        // Generic access denied (cross-tenant attempts return this)
        http.Error(w, "Access denied", http.StatusForbidden)

    default:
        h.logger.WithError(err).Error("Unhandled error in calendar handler")
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}
```

### Step 9: Update Router Initialization

Update the router setup to inject service instead of repository:

**Before:**
```go
// internal/server/router.go
calendarRepo := repository.NewInMemoryCalendarRepository(logger)
calendarHandler := api.NewCalendarHandler(calendarRepo, logger)

router.HandleFunc("POST /calendars", calendarHandler.Create)
router.HandleFunc("GET /calendars/{id}", calendarHandler.Get)
// ... other routes
```

**After:**
```go
// internal/server/router.go
calendarRepo := repository.NewInMemoryCalendarRepository(logger)
calendarService := services.NewCalendarServiceImpl(calendarRepo, logger)
calendarHandler := api.NewCalendarHandler(calendarService, logger)

router.HandleFunc("POST /calendars", calendarHandler.Create)
router.HandleFunc("GET /calendars/{id}", calendarHandler.Get)
// ... other routes
```

---

## Testing the Integration

### Run Tests

```bash
# 1. Run service integration tests
go test ./internal/services/... -v
# Expected: 11 tests passing

# 2. Run handler integration tests
go test ./internal/api/... -v -run "Phase3"
# Expected: 11 tests passing

# 3. Run all unit tests
go test ./... -v
# Expected: 20+ tests passing
```

### Manual Integration Test

```bash
# 1. Start service
go run ./cmd/calendar-service &

# 2. Generate JWT
TOKEN=$(./scripts/generate-jwt.sh tenant-a user-a)

# 3. Create calendar (via handler → service → repository)
curl -X POST http://localhost:8080/api/v1/calendars \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Integration Test Calendar",
    "timezone": "UTC"
  }'

# Expected: 201 Created with calendar data

# 4. List calendars
curl -X GET http://localhost:8080/api/v1/calendars \
  -H "Authorization: Bearer $TOKEN"

# Expected: 200 OK with calendar in list

# 5. Test cross-tenant access denial
TOKEN_B=$(./scripts/generate-jwt.sh tenant-b user-b)

curl -X GET http://localhost:8080/api/v1/calendars \
  -H "Authorization: Bearer $TOKEN_B"

# Expected: 200 OK but empty list (only tenant-b's calendars)
```

---

## Verification Checklist

Before committing changes:

- [ ] All 5 handler methods updated (Create, Get, List, Update, Delete)
- [ ] CalendarHandler struct updated (service field instead of repo)
- [ ] NewCalendarHandler() constructor updated
- [ ] Error handler (handleServiceError) added
- [ ] Router initialization updated to inject service
- [ ] All tests passing (22+)
- [ ] No breaking changes to API
- [ ] Compilation clean: `go build ./internal/...`
- [ ] Cross-tenant test passes
- [ ] Audit logging still working

---

## Rollout Plan

### Phase 3A: Handler Integration (2-3 hours)

1. Update CalendarHandler (5 methods)
2. Add error handler
3. Update router
4. Run tests

### Phase 3B: Apply to Other Handlers (1 day)

1. AvailabilityHandler (3 methods)
2. BlackoutHandler (3 methods)
3. TenantHandler (5 methods)

### Phase 3C: Production Verification (2-3 hours)

1. Deploy to staging
2. Run integration tests
3. Load test
4. Security audit
5. Deploy to production

---

**Total Timeline: 2-3 days**

**Risk: LOW** (backward compatible, tested pattern)

**Rollback: Simple** (revert to direct repository calls if needed)

---

**Phase 3 Extension Status: READY TO IMPLEMENT**
