# Marketplace System - Architecture & Design

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MARKETPLACE SYSTEM                       │
└─────────────────────────────────────────────────────────────┘

┌──────────────────┐
│  FRONTEND LAYER  │
├──────────────────┤
│ React/TypeScript │
│  - Marketplace   │
│    Component     │
│  - Tabs: Browse, │
│    My Items, Ana │
│  - Search/Filter │
│  - Add/Remove UI │
└────────┬─────────┘
         │ HTTP REST
         │
┌──────────────────────────────┐
│   BACKEND API LAYER          │
├──────────────────────────────┤
│ Go with Chi Router           │
│ - 10 Endpoints               │
│ - Tenant Isolation           │
│ - Request Validation         │
│ - Multi-tenant Safe Queries  │
└────────┬─────────────────────┘
         │ SQL Queries
         │
┌──────────────────────────────┐
│   DATABASE LAYER             │
├──────────────────────────────┤
│ PostgreSQL                   │
│ - marketplace_items          │
│ - tenant_marketplace_items   │
│ - marketplace_item_usage     │
│ - marketplace_item_feedback  │
│ - Versioning & Audit Trail  │
└──────────────────────────────┘
```

## 📦 Data Model

### Entity Relationship Diagram

```
┌─────────────────────┐
│ marketplace_items   │  (Shared Catalog)
│───────────────────  │
│ id (PK)             │
│ name                │
│ item_type           │  'rule' or 'calculation'
│ category            │
│ severity            │
│ implementation_json │
│ is_official         │
│ is_core             │
│ usage_count         │
│ rating              │
└──────────┬──────────┘
           │ (1)
           │
           ├─────────────────────────┐
           │ (many)                  │
           │                         │
    ┌──────────────────────┐   ┌────────────────────────┐
    │ tenant_marketplace   │   │ marketplace_item       │
    │ _items               │   │ _parameters            │
    ├──────────────────────┤   ├────────────────────────┤
    │ id (PK)              │   │ id (PK)                │
    │ tenant_id (FK)       │   │ marketplace_item_id    │
    │ marketplace_item_id  │───│ (FK)                   │
    │ (FK)                 │   │ param_name             │
    │ custom_name          │   │ param_type             │
    │ custom_parameters    │   │ validation_rules       │
    │ enabled_for_tenant   │   └────────────────────────┘
    │ added_at             │
    │ usage_count          │   ┌────────────────────────┐
    │ marketplace_version  │   │ marketplace_item       │
    │ local_version        │   │ _feedback              │
    │ tenant_rating        │───┤────────────────────────┤
    │                      │   │ id (PK)                │
    └──────────┬───────────┘   │ marketplace_item_id(FK)│
               │               │ tenant_id (FK)         │
               │               │ rating (1-5)           │
    ┌──────────────────────┐   │ feedback_text          │
    │ marketplace_item     │   │ created_at             │
    │ _usage               │   └────────────────────────┘
    ├──────────────────────┤
    │ id (PK)              │   ┌────────────────────────┐
    │ marketplace_item_id  │───┤ marketplace_item       │
    │ (FK)                 │   │ _versions              │
    │ tenant_id (FK)       │   ├────────────────────────┤
    │ execution_date       │   │ id (PK)                │
    │ execution_count      │   │ marketplace_item_id(FK)│
    │ success_count        │   │ version                │
    │ failure_count        │   │ implementation_json    │
    │ last_result_status   │   │ changelog              │
    │ last_error_message   │   │ created_at             │
    └──────────────────────┘   └────────────────────────┘
```

### Table Descriptions

#### `marketplace_items` (Shared Catalog)
**Purpose:** Immutable catalog of all available rules/calculations  
**Ownership:** Platform admin  
**Access:** Read-only for tenants  
**Key Fields:**
- `id`: Unique identifier
- `name`: Item name
- `item_type`: 'rule' | 'calculation'
- `category`: Business domain (ESG, AML, Compliance, etc.)
- `severity`: BLOCK | WARNING | INFO
- `implementation_json`: Logic/algorithm definition
- `is_official`: Flag for official/recommended items
- `is_core`: Flag for essential/required items
- `rating`: Aggregate rating (1-5)
- `usage_count`: How many times added across all tenants

**Indexes:** type, category, official, public status

---

#### `tenant_marketplace_items` (Tenant's Choices)
**Purpose:** Many-to-many relationship between tenants and items  
**Ownership:** Individual tenant  
**Access:** Read-write for own items, read-only for others  
**Key Fields:**
- `tenant_id`: Which organization
- `marketplace_item_id`: Which item (FK to marketplace_items)
- `custom_name`: Tenant can rename it
- `custom_parameters`: Tenant-specific configuration
- `enabled_for_tenant`: Is it active?
- `added_at`: When tenant added it
- `marketplace_version_at_time_of_add`: Snapshot of version when added
- `local_version`: Tenant's modified version (if any)
- `tenant_rating`: This tenant's rating

**Constraints:**
- UNIQUE (tenant_id, marketplace_item_id) - prevent duplicates
- Cascade delete when marketplace item deleted

**Indexes:** tenant_id, marketplace_item_id, enabled status

---

#### `marketplace_item_parameters` (Item Configuration)
**Purpose:** Define configurable parameters for each item  
**Ownership:** Platform admin (per item)  
**Access:** Read-only for tenants  
**Key Fields:**
- `marketplace_item_id`: Which item
- `param_name`: Name of parameter (e.g., "threshold")
- `param_type`: 'string' | 'number' | 'boolean' | 'enum'
- `validation_rules`: JSON schema for validation
- `display_name`: UI label
- `default_value`: Fallback value

**Example:**
```json
{
  "param_name": "threshold",
  "param_type": "number",
  "display_name": "ESG Score Threshold",
  "default_value": 0.7,
  "validation_rules": {
    "min": 0,
    "max": 1
  }
}
```

---

#### `marketplace_item_usage` (Analytics)
**Purpose:** Track usage of each item per tenant per day  
**Ownership:** System (auto-recorded)  
**Access:** Read-only for tenants (see own usage)  
**Key Fields:**
- `tenant_id`: Which tenant
- `marketplace_item_id`: Which item
- `execution_date`: YYYY-MM-DD
- `execution_count`: How many times run
- `success_count`: Successful runs
- `failure_count`: Failed runs
- `average_execution_time_ms`: Performance
- `last_result_status`: OK | ERROR | TIMEOUT

**Constraints:**
- UNIQUE (tenant_id, marketplace_item_id, execution_date) - one row per day

---

#### `marketplace_item_feedback` (Ratings & Reviews)
**Purpose:** Store tenant feedback on items  
**Ownership:** Individual tenant  
**Access:** Tenants can read own feedback, see aggregate stats  
**Key Fields:**
- `marketplace_item_id`: Which item
- `tenant_id`: Which tenant rated it
- `rating`: 1-5 stars
- `feedback_text`: Comment/review
- `created_at`: When submitted

**Constraints:**
- One row per tenant per item (natural singleton)
- Aggregate rating computed from all tenants' ratings

---

#### `marketplace_item_versions` (Version History)
**Purpose:** Track changes to items over time  
**Ownership:** Platform admin  
**Access:** Read-only audit trail  
**Key Fields:**
- `marketplace_item_id`: Which item
- `version`: Semantic version (1.0.0)
- `implementation_json`: Implementation at this version
- `changelog`: What changed
- `deprecation_reason`: If deprecated
- `created_at`: Release date

**Use Case:**
- Tenant added item at v1.0
- Item updated to v1.1, v2.0
- Tenant can see version mismatch
- Can upgrade to new version with changelog review

---

## 🔄 Data Flow

### Adding an Item to Tenant

```
User clicks "Add to Platform"
         ↓
Frontend POST /api/marketplace/items/add-to-tenant
{
  "marketplace_item_id": "uuid-123",
  "custom_name": "My ESG Check",
  "custom_parameters": { "threshold": 0.8 }
}
         ↓
Backend validates:
- Tenant exists (X-Tenant-ID header)
- Item exists (marketplace_item_id)
- Parameters valid (against marketplace_item_parameters schema)
         ↓
Backend UPSERTs into tenant_marketplace_items:
- Insert if not exists
- Update enabled status if already added
         ↓
Backend returns: { "id": "tenant-item-uuid" }
         ↓
Frontend updates UI:
- Add shows "Already Added" badge
- Item appears in "My Items" tab
         ↓
Database state:
INSERT INTO tenant_marketplace_items (
  id, tenant_id, marketplace_item_id, custom_name,
  custom_parameters, enabled_for_tenant, added_at,
  marketplace_version_at_time_of_add, local_version
) VALUES (...)
```

### Searching Marketplace

```
User types "ESG" in search box
         ↓
Frontend filters client-side OR calls:
GET /api/marketplace/items?search=ESG&category=Compliance
         ↓
Backend searches:
SELECT * FROM marketplace_items
WHERE name ILIKE '%ESG%'
  AND category = 'Compliance'
  AND is_public = TRUE
ORDER BY name
         ↓
Backend returns: [{ id, name, ... }, ...]
         ↓
Frontend displays matching items in grid
         ↓
For each item, frontend checks:
- Is it already added? (compare with tenantItems array)
- Show "Add" or "Already Added" badge
```

### Recording Usage

```
External system executes a rule
         ↓
INSERT INTO marketplace_item_usage (
  tenant_id, marketplace_item_id, execution_date,
  execution_count, success_count, failure_count, ...
) VALUES (...)
ON CONFLICT (tenant_id, marketplace_item_id, execution_date)
DO UPDATE SET
  execution_count = execution_count + 1,
  success_count = success_count + 1
         ↓
When Analytics tab loads:
SELECT SUM(execution_count) as total_uses,
       AVG(average_execution_time_ms) as avg_time
FROM marketplace_item_usage
WHERE tenant_id = ?
         ↓
Frontend displays metrics
```

---

## 🔐 Security Design

### Multi-Tenant Isolation

**Every API call requires:**
1. `X-Tenant-ID` header (which tenant is this?)
2. Authentication (who is the user?)

**Backend enforces:**
```go
// All queries filtered by tenant_id
WHERE tenant_id = extractedFromHeader
```

**Tenant can only:**
- ✅ View public marketplace items
- ✅ Add items to their own account
- ✅ View items they've added
- ✅ Remove items they've added
- ✅ Edit their own custom_parameters
- ✅ Rate items (anonymous aggregation)
- ❌ See other tenants' items
- ❌ See other tenants' usage
- ❌ Modify marketplace items
- ❌ Modify other tenants' selections

**Database constraints:**
```sql
-- Tenant can't access other tenant's items
ALTER TABLE tenant_marketplace_items
ADD CONSTRAINT check_own_items
CHECK (tenant_id = <current_tenant>);

-- Foreign key ensures item exists
ALTER TABLE tenant_marketplace_items
ADD FOREIGN KEY (marketplace_item_id)
REFERENCES marketplace_items(id);
```

### Data Sanitization

**Input validation:**
- Item ID: Must be valid UUID
- Custom name: Max 255 chars, no SQL
- Custom parameters: Validate against schema
- Rating: 1-5 integer only

**Output sanitization:**
- Return only fields user needs
- Never leak other tenant's data
- Aggregate ratings don't show individual tenant ratings

---

## 🏃 Performance Considerations

### Indexing Strategy

```sql
-- Fast lookups
INDEX idx_marketplace_items_type (item_type)
INDEX idx_marketplace_items_category (category)
INDEX idx_marketplace_items_official (is_official)

-- Tenant queries
INDEX idx_tenant_marketplace_items_tenant (tenant_id)
INDEX idx_tenant_marketplace_items_enabled (enabled_for_tenant)

-- Analytics
INDEX idx_marketplace_item_usage_date (execution_date)
INDEX idx_marketplace_item_usage_item (marketplace_item_id)
```

### Query Optimization

**Listing items (with usage):**
```sql
SELECT mi.*, 
       COUNT(tmi.id) as adoption_count,
       AVG(mif.rating) as avg_rating
FROM marketplace_items mi
LEFT JOIN tenant_marketplace_items tmi ON mi.id = tmi.marketplace_item_id
LEFT JOIN marketplace_item_feedback mif ON mi.id = mif.marketplace_item_id
WHERE mi.is_public = TRUE
GROUP BY mi.id
ORDER BY adoption_count DESC
LIMIT 50;
```

**Tenant's items (with usage):**
```sql
SELECT tmi.*, mi.*, 
       SUM(miu.execution_count) as total_uses
FROM tenant_marketplace_items tmi
JOIN marketplace_items mi ON tmi.marketplace_item_id = mi.id
LEFT JOIN marketplace_item_usage miu 
  ON tmi.marketplace_item_id = miu.marketplace_item_id
  AND tmi.tenant_id = miu.tenant_id
WHERE tmi.tenant_id = ?
GROUP BY tmi.id, mi.id;
```

### Caching Strategy

**Cache in Frontend:**
- `marketplaceItems` - refresh every 5 minutes
- `tenantItems` - refresh on add/remove
- `selectedItem` - cache until modal closes

**Cache in Backend:**
- Popular items (top 20) - 1 hour TTL
- Item parameters - 24 hours TTL
- Aggregate ratings - 1 hour TTL

---

## 🎨 Frontend Architecture

### Component Structure

```
Marketplace.tsx (Main Container)
├── Header
│   └── Title + Description
├── Tabs
│   ├── Browse Tab
│   │   ├── Filters Sidebar
│   │   │   ├── Search
│   │   │   ├── Type Filter
│   │   │   ├── Category Filter
│   │   │   ├── Severity Filter
│   │   │   └── Official/Core Checkboxes
│   │   ├── Main Content
│   │   │   ├── View Mode Toggle (Grid/List)
│   │   │   ├── Sort Dropdown
│   │   │   ├── Item Grid/List
│   │   │   │   └── ItemCard x N
│   │   │   │       ├── Icon + Name
│   │   │   │       ├── Category + Rating
│   │   │   │       ├── Add Button / Already Added Badge
│   │   │   │       └── Click to Open Modal
│   │   │   └── Pagination (Future)
│   │   └── Detail Modal (when item clicked)
│   │       ├── Close Button
│   │       ├── Large Icon + Name + Version
│   │       ├── Official/Core Badges
│   │       ├── Description
│   │       ├── Details Grid
│   │       ├── External Providers
│   │       ├── Feedback Score
│   │       └── Add/Already Added Button
│   ├── My Items Tab
│   │   ├── Item List (Cards)
│   │   │   └── ItemCard x N
│   │   │       ├── Name + Enabled Toggle
│   │   │       ├── Date Added + Usage Count
│   │   │       ├── Actions (Details, Configure, Remove)
│   │   │       └── Version Badge
│   │   └── Empty State
│   │       └── Link back to Browse
│   └── Analytics Tab
│       ├── Metric Cards
│       │   ├── Total Items Added
│       │   ├── Total Uses
│       │   └── Active Items
│       └── Placeholder for Charts
└── Styling (Marketplace.module.css)
    ├── Container Layout
    ├── Header Styles
    ├── Tab Styles
    ├── Filter Styles
    ├── Grid/List View
    ├── Card Styles
    ├── Modal Styles
    ├── Button States
    ├── Responsive Breakpoints
    └── Dark Mode Support
```

### State Management

```typescript
// Tab Navigation
const [activeTab, setActiveTab] = useState<'browse' | 'my-items' | 'analytics'>('browse');

// Data
const [marketplaceItems, setMarketplaceItems] = useState<MarketplaceItem[]>([]);
const [tenantItems, setTenantItems] = useState<TenantMarketplaceItem[]>([]);
const [selectedItem, setSelectedItem] = useState<MarketplaceItem | null>(null);

// Filters
const [searchTerm, setSearchTerm] = useState('');
const [selectedItemType, setSelectedItemType] = useState('');
const [selectedCategories, setSelectedCategories] = useState<string[]>([]);
const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
const [showOnlyOfficial, setShowOnlyOfficial] = useState(false);
const [showOnlyCore, setShowOnlyCore] = useState(false);

// UI
const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
const [sortBy, setSortBy] = useState<'relevance' | 'popular' | 'rating' | 'newest'>('relevance');
const [isLoading, setIsLoading] = useState(false);
const [error, setError] = useState<string | null>(null);
```

### API Integration

```typescript
// Load marketplace items
useEffect(() => {
  const fetchItems = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/marketplace/items', {
        headers: {
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId
        }
      });
      const data = await response.json();
      setMarketplaceItems(data.items);
    } catch (err) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };
  fetchItems();
}, [tenantId, datasourceId]);

// Add item
const handleAddItem = async (itemId: string) => {
  try {
    const response = await fetch('/api/marketplace/items/add-to-tenant', {
      method: 'POST',
      headers: {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ marketplace_item_id: itemId })
    });
    const data = await response.json();
    // Refresh tenant items
    fetchTenantItems();
  } catch (err) {
    setError(err.message);
  }
};
```

---

## 🔌 Backend API Architecture

### Request/Response Pattern

**All endpoints follow:**
```go
func HandleXxx(w http.ResponseWriter, r *http.Request) {
    // 1. Extract tenant from header
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
        return
    }
    
    // 2. Extract/parse request data
    var req RequestBody
    json.NewDecoder(r.Body).Decode(&req)
    
    // 3. Validate input
    if err := validate(req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // 4. Query database (filtered by tenant)
    rows, err := db.Query("SELECT ... WHERE tenant_id = $1", tenantID)
    
    // 5. Build response
    resp := map[string]interface{}{ "data": ... }
    
    // 6. Return JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

### Error Handling

```go
// Standardized error responses
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code"`
    Status  int    `json:"status"`
}

// Error codes
const (
    ErrTenantRequired = "TENANT_REQUIRED"
    ErrNotFound       = "NOT_FOUND"
    ErrForbidden      = "FORBIDDEN"
    ErrConflict       = "CONFLICT"
    ErrInvalid        = "INVALID_INPUT"
)
```

---

## 📈 Scalability Design

### Current Capacity
- Supports: 1,000s of items
- Supports: 10,000s of tenants
- Supports: 1M+ usage records per day

### Future Optimization
1. **Pagination:** Add LIMIT/OFFSET for large result sets
2. **Caching Layer:** Redis for popular items
3. **Search:** Elasticsearch for full-text search
4. **Analytics:** Data warehouse for historical analysis
5. **Message Queue:** Async processing for usage tracking
6. **CDN:** Cache images/icons

### Database Scaling
```
Current: Single PostgreSQL instance
Next: Read replicas for analytics queries
Later: Sharding by tenant_id if needed
```

---

## 🧪 Testing Strategy

### Unit Tests
- Filter logic (search, category, severity)
- Validation (parameters, ratings)
- Permission checks (tenant isolation)

### Integration Tests
- Add item → verify DB insert
- Remove item → verify DB delete
- Search → verify results
- Permission boundaries

### E2E Tests
- Browse catalog
- Search and filter
- Add item to platform
- View in My Items
- Remove item
- Submit rating

### Load Testing
- 1000 concurrent users browsing
- 100 concurrent adds
- Search performance with 10K items

---

## 📋 Deployment Topology

### Development
```
localhost:3000 (React Dev Server)
    ↓ HTTP
localhost:8080 (Go API)
    ↓ SQL
localhost:5432 (PostgreSQL)
```

### Production
```
CDN → Load Balancer → [API Server 1, API Server 2, API Server 3]
                           ↓
                      Connection Pool
                           ↓
            Primary PostgreSQL + Replicas
```

---

## 🎯 Success Metrics

### Performance
- Browse page load: < 500ms
- Search results: < 100ms
- Add item: < 200ms
- Overall API latency: p95 < 500ms

### Adoption
- % of tenants browsing marketplace
- % of tenants adding items
- Avg items per tenant
- Items with 5+ ratings

### Quality
- Zero cross-tenant data leaks
- Zero failed transactions
- 99.9% API availability
- < 0.1% error rate

---

**Architecture Version:** 1.0  
**Last Updated:** 2024-10-27  
**Approved For:** Production Deployment
