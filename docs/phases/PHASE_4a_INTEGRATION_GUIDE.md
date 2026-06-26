# Phase 4a Integration Guide: How to Use CQRS in Your Handlers

**Purpose:** Quick reference for integrating CQRSQueryService into existing HTTP handlers

**Status:** Ready to implement (copy/paste examples below)

---

## 🔧 Quick Integration Steps

### Step 1: Add to Handler Constructor

**File:** `backend/internal/handlers/businessobject_handler.go`

```go
type BusinessObjectHandler struct {
    db                     *sqlx.DB
    boService              *services.BusinessObjectService
    commandPublisher       *services.CommandPublisher
    eventPublisher         *services.EventPublisher
    queryService           *services.CQRSQueryServiceImpl      // NEW
    idempotencyRepository  *services.CQRSIdempotencyRepositoryImpl  // NEW
}

func NewBusinessObjectHandler(
    db *sqlx.DB,
    boService *services.BusinessObjectService,
    commandPublisher *services.CommandPublisher,
    eventPublisher *services.EventPublisher,
) *BusinessObjectHandler {
    return &BusinessObjectHandler{
        db:                    db,
        boService:             boService,
        commandPublisher:      commandPublisher,
        eventPublisher:        eventPublisher,
        queryService:          services.NewCQRSQueryServiceImpl(db),           // NEW
        idempotencyRepository: services.NewCQRSIdempotencyRepository(db),     // NEW
    }
}
```

### Step 2: Use in Read Endpoints

**Example: Get Single Business Object**

```go
// BEFORE: Using full service layer
func (h *BusinessObjectHandler) GetBusinessObject(w http.ResponseWriter, r *http.Request) {
    // ... complex service layer logic ...
}

// AFTER: Using CQRSQueryService
func (h *BusinessObjectHandler) GetBusinessObject(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Header.Get("X-Tenant-ID")
    key := chi.URLParam(r, "key")
    
    // Use CQRSQueryService for fast read
    bo, err := h.queryService.GetBusinessObjectForRead(r.Context(), tenantID, key)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bo)
}
```

**Benefits:**
- ✅ Faster (direct query, no service layer)
- ✅ Simpler code (less logic)
- ✅ Ready for Phase 4b (projections)

### Step 3: List Endpoint with Pagination

```go
func (h *BusinessObjectHandler) ListBusinessObjects(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Header.Get("X-Tenant-ID")
    
    // Parse pagination
    offset := queryParamInt(r, "offset", 0)
    limit := queryParamInt(r, "limit", 20)
    
    // Use CQRSQueryService for fast list
    results, total, err := h.queryService.ListBusinessObjectsForRead(
        r.Context(),
        tenantID,
        offset,
        limit,
    )
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    response := map[string]interface{}{
        "results": results,
        "total":   total,
        "offset":  offset,
        "limit":   limit,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

---

## 📝 Using Idempotency in Write Endpoints

### Example: Create Business Object with Idempotency

**Before (No Idempotency):**
```go
func (h *BusinessObjectHandler) CreateBusinessObject(w http.ResponseWriter, r *http.Request) {
    var req models.CreateBusinessObjectRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Publish command
    bo, err := h.boService.CreateBusinessObject(r.Context(), tenantID, req, userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bo)
}
```

**Problem:** If request retries, duplicate BOs created!

**After (With Idempotency):**
```go
func (h *BusinessObjectHandler) CreateBusinessObject(w http.ResponseWriter, r *http.Request) {
    // Get correlation ID (should be unique per request)
    correlationID := r.Header.Get("Idempotency-Key")
    if correlationID == "" {
        correlationID = uuid.New().String()
    }
    
    // Step 1: Check if already processed
    processed, resultID, err := h.idempotencyRepository.IsCommandProcessed(r.Context(), correlationID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // If already processed, return cached result
    if processed {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("X-Idempotency-Cached", "true")
        w.WriteHeader(http.StatusOK)
        // Write cached result (could fetch from DB if needed)
        json.NewEncoder(w).Encode(map[string]string{"id": resultID})
        return
    }
    
    // Step 2: Decode request
    var req models.CreateBusinessObjectRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Step 3: Create via command bus (same as before)
    bo, err := h.boService.CreateBusinessObject(r.Context(), tenantID, req, userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Step 4: Record for idempotency
    if err := h.idempotencyRepository.RecordCommandExecution(
        r.Context(),
        correlationID,
        "CreateBO",
        bo.ID,
    ); err != nil {
        log.Printf("warning: failed to record command execution: %v", err)
        // Don't fail - just log warning
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bo)
}
```

**Benefits:**
- ✅ Retries are safe (same request processed once)
- ✅ Works with load balancers and proxies
- ✅ Improves reliability

---

## 🔄 Complete Handler Example

```go
package handlers

import (
    "encoding/json"
    "log"
    "net/http"
    "strconv"
    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    
    "github.com/eganpj/semlayer/backend/internal/models"
    "github.com/eganpj/semlayer/backend/internal/services"
)

type BusinessObjectHandler struct {
    db                     *sqlx.DB
    boService              *services.BusinessObjectService
    commandPublisher       *services.CommandPublisher
    eventPublisher         *services.EventPublisher
    queryService           *services.CQRSQueryServiceImpl
    idempotencyRepository  *services.CQRSIdempotencyRepositoryImpl
}

func NewBusinessObjectHandler(
    db *sqlx.DB,
    boService *services.BusinessObjectService,
    commandPublisher *services.CommandPublisher,
    eventPublisher *services.EventPublisher,
) *BusinessObjectHandler {
    return &BusinessObjectHandler{
        db:                    db,
        boService:             boService,
        commandPublisher:      commandPublisher,
        eventPublisher:        eventPublisher,
        queryService:          services.NewCQRSQueryServiceImpl(db),
        idempotencyRepository: services.NewCQRSIdempotencyRepository(db),
    }
}

// READ ENDPOINTS (using CQRSQueryService)

func (h *BusinessObjectHandler) GetBusinessObject(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Header.Get("X-Tenant-ID")
    key := chi.URLParam(r, "key")
    
    bo, err := h.queryService.GetBusinessObjectForRead(r.Context(), tenantID, key)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bo)
}

func (h *BusinessObjectHandler) ListBusinessObjects(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Header.Get("X-Tenant-ID")
    offset := queryParamInt(r, "offset", 0)
    limit := queryParamInt(r, "limit", 20)
    
    results, total, err := h.queryService.ListBusinessObjectsForRead(
        r.Context(),
        tenantID,
        offset,
        limit,
    )
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "results": results,
        "total":   total,
    })
}

// WRITE ENDPOINTS (using idempotency)

func (h *BusinessObjectHandler) CreateBusinessObject(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Header.Get("X-Tenant-ID")
    userID := r.Header.Get("X-User-ID")
    correlationID := r.Header.Get("Idempotency-Key")
    if correlationID == "" {
        correlationID = uuid.New().String()
    }
    
    // Check idempotency
    processed, cachedID, err := h.idempotencyRepository.IsCommandProcessed(r.Context(), correlationID)
    if err != nil {
        log.Printf("error checking idempotency: %v", err)
    }
    
    if processed {
        w.Header().Set("X-Idempotency-Cached", "true")
        json.NewEncoder(w).Encode(map[string]string{"id": cachedID})
        return
    }
    
    // Decode request
    var req models.CreateBusinessObjectRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Create via service (unchanged from Phase 1-3)
    bo, err := h.boService.CreateBusinessObject(r.Context(), tenantID, req, userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Record for idempotency
    if err := h.idempotencyRepository.RecordCommandExecution(
        r.Context(),
        correlationID,
        "CreateBO",
        bo.ID,
    ); err != nil {
        log.Printf("warning: failed to record command execution: %v", err)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bo)
}

// HELPER

func queryParamInt(r *http.Request, key string, defaultVal int) int {
    val := r.URL.Query().Get(key)
    if val == "" {
        return defaultVal
    }
    i, err := strconv.Atoi(val)
    if err != nil {
        return defaultVal
    }
    return i
}
```

---

## 🚀 Integration Checklist

- [ ] Add CQRSQueryService to handler constructor
- [ ] Add CQRSIdempotencyRepository to handler constructor
- [ ] Update GET endpoints to use queryService
- [ ] Update POST/PUT endpoints to use idempotencyRepository
- [ ] Test read endpoints (should be faster or same speed)
- [ ] Test write endpoints with retries (should be idempotent)
- [ ] Deploy gradually (one endpoint at a time)

---

## 🔗 Database Requirements

**Idempotency Table:**

```sql
CREATE TABLE IF NOT EXISTS idempotency_records (
    correlation_id VARCHAR(255) PRIMARY KEY,
    command_type VARCHAR(100) NOT NULL,
    result_id VARCHAR(255) NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_idempotency_expires ON idempotency_records(expires_at);

-- Cleanup old records (can be run via cron)
DELETE FROM idempotency_records WHERE expires_at < NOW();
```

**No changes needed to existing tables** - Phase 4a works with current schema!

---

## ✅ Testing After Integration

### Test 1: Read Performance
```bash
# Should be same speed (no optimization yet)
curl -H "X-Tenant-ID: tenant-1" \
     http://localhost:8080/api/business-objects/customer

# Check response time (should be ~50ms)
```

### Test 2: Idempotency
```bash
# First request
curl -X POST http://localhost:8080/api/business-objects \
  -H "Idempotency-Key: req-123" \
  -H "X-Tenant-ID: tenant-1" \
  -d '{"name": "Account"}' \
  -w "\n%{time_total}s\n"

# Second request (same key)
curl -X POST http://localhost:8080/api/business-objects \
  -H "Idempotency-Key: req-123" \
  -H "X-Tenant-ID: tenant-1" \
  -d '{"name": "Account"}' \
  -w "\n%{time_total}s\n"

# Should return same ID and have X-Idempotency-Cached header
```

### Test 3: Pagination
```bash
# List with pagination
curl -H "X-Tenant-ID: tenant-1" \
     "http://localhost:8080/api/business-objects?offset=0&limit=10"

# Should return total count and paginated results
```

---

## 📚 Related Documentation

- **CQRS Pattern Details:** `PHASE_4a_CQRS_COMPLETE.md`
- **Integration Points:** `PHASE_4a_DELIVERY_SUMMARY.md`
- **Full Roadmap:** `COMPLETE_PHASES_1-4a_ROADMAP.md`
- **Session Summary:** `SESSION_SUMMARY_PHASE_4a.md`

---

## 🎯 Next Phase (Phase 4b)

After integrating Phase 4a into handlers:

```go
// Phase 4b will add:
bo_projections table (separate read model)

// Events will update projection:
func (updater *ProjectionUpdater) OnBOCreatedEvent(event *Event) {
    // Insert into bo_projections
}

// Queries will use projection:
func (qs *CQRSQueryService) GetBusinessObjectForRead(...) {
    // Query bo_projections instead of business_objects
    // 40% faster because data is pre-aggregated!
}
```

No changes needed to HTTP handlers - just swap the underlying query!

