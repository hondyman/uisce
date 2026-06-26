# Marketplace System - API Reference & Integration Patterns

## 📖 API Reference

### Base URL
```
http://localhost:8080/api/marketplace
```

### Authentication
All endpoints require:
```http
X-Tenant-ID: <uuid>
X-Tenant-Datasource-ID: <uuid>  (optional, for scoping)
```

### Content-Type
```http
Content-Type: application/json
```

---

## 🔍 GET /api/marketplace/items

Browse all marketplace items with optional filtering and sorting.

### Request

**Method:** `GET`

**Query Parameters:**
| Parameter | Type | Example | Required |
|-----------|------|---------|----------|
| `search` | string | `"ESG"` | No |
| `item_type` | string | `"rule"` or `"calculation"` | No |
| `category` | string[] | `"ESG & Sustainability"`, `"Compliance & Regulatory"` | No |
| `severity` | string[] | `"BLOCK"`, `"WARNING"`, `"INFO"` | No |
| `only_official` | boolean | `true` | No |
| `only_core` | boolean | `true` | No |
| `sort_by` | string | `"relevance"`, `"popular"`, `"rating"`, `"newest"` | No |
| `page` | number | `1` | No |
| `limit` | number | `20` | No |

**Example:**
```bash
GET /api/marketplace/items?search=ESG&category=ESG%20%26%20Sustainability&sort_by=rating \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

### Response

**Status:** `200 OK`

**Body:**
```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "ESG Compliance",
      "description": "Validates portfolio compliance with ESG standards",
      "item_type": "rule",
      "version": "1.0.0",
      "category": "ESG & Sustainability",
      "subcategories": ["environmental", "social", "governance"],
      "severity": "BLOCK",
      "icon_emoji": "🌱",
      "color_hex": "#059669",
      "summary": "Check ESG score against thresholds",
      "long_description": "This rule validates that all securities...",
      "scope": "PORTFOLIO",
      "rule_type": "CONDITION",
      "frequency": "ON_TRADE",
      "evaluation_order": 1,
      "is_public": true,
      "is_official": true,
      "is_core": false,
      "status": "active",
      "external_api_providers": ["MSCI", "Refinitiv"],
      "requires_credentials": true,
      "usage_count": 342,
      "rating": 4.8,
      "downloads_count": 285,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-10-27T14:30:00Z",
      "published_at": "2024-01-15T10:00:00Z",
      "is_already_added": false
    },
    // ... more items ...
  ],
  "total_count": 125,
  "page": 1,
  "limit": 20,
  "facets": {
    "categories": [
      {
        "name": "ESG & Sustainability",
        "count": 15
      },
      {
        "name": "Compliance & Regulatory",
        "count": 28
      },
      // ...
    ],
    "severities": [
      {
        "name": "BLOCK",
        "count": 45
      },
      {
        "name": "WARNING",
        "count": 62
      },
      {
        "name": "INFO",
        "count": 18
      }
    ],
    "item_types": [
      {
        "name": "rule",
        "count": 95
      },
      {
        "name": "calculation",
        "count": 30
      }
    ]
  }
}
```

---

## 📦 GET /api/marketplace/items/{id}

Get detailed information about a specific marketplace item.

### Request

**Method:** `GET`

**Path Parameters:**
| Parameter | Type | Example |
|-----------|------|---------|
| `id` | uuid | `550e8400-e29b-41d4-a716-446655440000` |

**Example:**
```bash
GET /api/marketplace/items/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

### Response

**Status:** `200 OK`

**Body:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "ESG Compliance",
  "description": "Validates portfolio compliance with ESG standards",
  "item_type": "rule",
  "version": "1.0.0",
  "category": "ESG & Sustainability",
  "subcategories": ["environmental", "social", "governance"],
  "severity": "BLOCK",
  "icon_emoji": "🌱",
  "color_hex": "#059669",
  "summary": "Check ESG score against thresholds",
  "long_description": "This rule validates that all securities in the portfolio...",
  "implementation_json": {
    "type": "THRESHOLD_CHECK",
    "field": "esg_score",
    "operator": ">=",
    "threshold_param": "min_score"
  },
  "scope": "PORTFOLIO",
  "rule_type": "CONDITION",
  "frequency": "ON_TRADE",
  "evaluation_order": 1,
  "creator_id": "creator-uuid",
  "is_public": true,
  "is_official": true,
  "is_core": false,
  "status": "active",
  "external_api_providers": ["MSCI", "Refinitiv"],
  "requires_credentials": true,
  "usage_count": 342,
  "rating": 4.8,
  "downloads_count": 285,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-10-27T14:30:00Z",
  "published_at": "2024-01-15T10:00:00Z",
  "changelog": "v1.0.0: Initial release\nv1.0.1: Fixed edge case handling\nv1.0.2: Performance improvements"
}
```

**Error Responses:**

```json
// 404 Not Found
{
  "error": "Item not found",
  "code": "NOT_FOUND",
  "status": 404
}
```

---

## ⚙️ GET /api/marketplace/items/{id}/parameters

Get parameter definitions for an item.

### Request

**Method:** `GET`

**Path Parameters:**
| Parameter | Type | Example |
|-----------|------|---------|
| `id` | uuid | `550e8400-e29b-41d4-a716-446655440000` |

**Example:**
```bash
GET /api/marketplace/items/550e8400-e29b-41d4-a716-446655440000/parameters \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

### Response

**Status:** `200 OK`

**Body:**
```json
{
  "parameters": [
    {
      "id": "param-uuid-1",
      "marketplace_item_id": "550e8400-e29b-41d4-a716-446655440000",
      "param_name": "min_score",
      "param_type": "number",
      "display_name": "Minimum ESG Score",
      "description": "Minimum ESG score (0-100)",
      "default_value": 70,
      "is_required": true,
      "validation_rules": {
        "type": "number",
        "minimum": 0,
        "maximum": 100
      },
      "display_order": 1
    },
    {
      "id": "param-uuid-2",
      "param_name": "provider",
      "param_type": "enum",
      "display_name": "ESG Provider",
      "default_value": "MSCI",
      "is_required": true,
      "validation_rules": {
        "enum": ["MSCI", "Refinitiv", "Bloomberg"]
      },
      "display_order": 2
    },
    {
      "id": "param-uuid-3",
      "param_name": "apply_to_bonds",
      "param_type": "boolean",
      "display_name": "Apply to Bond Holdings",
      "default_value": false,
      "is_required": false,
      "display_order": 3
    }
  ]
}
```

---

## ➕ POST /api/marketplace/items/add-to-tenant

Add a marketplace item to a tenant's platform.

### Request

**Method:** `POST`

**Headers:**
```http
X-Tenant-ID: <uuid>
Content-Type: application/json
```

**Body:**
```json
{
  "marketplace_item_id": "550e8400-e29b-41d4-a716-446655440000",
  "custom_name": "My ESG Validator",
  "custom_parameters": {
    "min_score": 75,
    "provider": "MSCI",
    "apply_to_bonds": true
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/marketplace/items/add-to-tenant \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "Content-Type: application/json" \
  -d '{
    "marketplace_item_id": "550e8400-e29b-41d4-a716-446655440000",
    "custom_name": "My ESG Validator"
  }'
```

### Response

**Status:** `201 Created`

**Body:**
```json
{
  "id": "tenant-item-uuid",
  "tenant_id": "00000000-0000-0000-0000-000000000000",
  "marketplace_item_id": "550e8400-e29b-41d4-a716-446655440000",
  "custom_name": "My ESG Validator",
  "custom_parameters": {
    "min_score": 75,
    "provider": "MSCI",
    "apply_to_bonds": true
  },
  "enabled_for_tenant": true,
  "added_at": "2024-10-27T15:30:00Z",
  "marketplace_version_at_time_of_add": "1.0.2",
  "local_version": "1.0.2"
}
```

**Error Responses:**

```json
// 400 Bad Request - Missing required field
{
  "error": "marketplace_item_id is required",
  "code": "INVALID_INPUT",
  "status": 400
}

// 400 Bad Request - Parameters don't match schema
{
  "error": "custom_parameters do not match item schema",
  "code": "INVALID_INPUT",
  "status": 400
}

// 404 Not Found
{
  "error": "Marketplace item not found",
  "code": "NOT_FOUND",
  "status": 404
}

// 409 Conflict - Already added
{
  "error": "Item already added to tenant",
  "code": "CONFLICT",
  "status": 409
}
```

---

## 📋 GET /api/marketplace/tenant-items

List all items a tenant has added.

### Request

**Method:** `GET`

**Headers:**
```http
X-Tenant-ID: <uuid>
```

**Query Parameters:**
| Parameter | Type | Example |
|-----------|------|---------|
| `enabled_only` | boolean | `true` |
| `sort_by` | string | `"added_date"`, `"usage_count"` |

**Example:**
```bash
GET /api/marketplace/tenant-items?enabled_only=true&sort_by=usage_count \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

### Response

**Status:** `200 OK`

**Body:**
```json
{
  "items": [
    {
      "id": "tenant-item-uuid",
      "tenant_id": "00000000-0000-0000-0000-000000000000",
      "marketplace_item_id": "550e8400-e29b-41d4-a716-446655440000",
      "custom_name": "My ESG Validator",
      "custom_parameters": {
        "min_score": 75,
        "provider": "MSCI"
      },
      "enabled_for_tenant": true,
      "added_at": "2024-10-27T15:30:00Z",
      "last_used_at": "2024-10-27T16:45:00Z",
      "usage_count": 42,
      "marketplace_version_at_time_of_add": "1.0.2",
      "local_version": "1.0.2",
      "has_local_modifications": false,
      "marketplace_item": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "ESG Compliance",
        "category": "ESG & Sustainability",
        "icon_emoji": "🌱",
        "rating": 4.8
      }
    },
    // ... more items ...
  ],
  "total_count": 8
}
```

---

## 🔍 GET /api/marketplace/tenant-items/{id}

Get a specific item a tenant has added.

### Request

**Method:** `GET`

**Path Parameters:**
| Parameter | Type | Example |
|-----------|------|---------|
| `id` | uuid | `tenant-item-uuid` |

**Example:**
```bash
GET /api/marketplace/tenant-items/tenant-item-uuid \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

### Response

**Status:** `200 OK`

**Body:**
```json
{
  "id": "tenant-item-uuid",
  "tenant_id": "00000000-0000-0000-0000-000000000000",
  "marketplace_item_id": "550e8400-e29b-41d4-a716-446655440000",
  "custom_name": "My ESG Validator",
  "custom_parameters": {
    "min_score": 75,
    "provider": "MSCI"
  },
  "enabled_for_tenant": true,
  "added_at": "2024-10-27T15:30:00Z",
  "last_used_at": "2024-10-27T16:45:00Z",
  "usage_count": 42,
  "marketplace_version_at_time_of_add": "1.0.2",
  "local_version": "1.0.2",
  "has_local_modifications": false,
  "tenant_rating": 5,
  "tenant_feedback": "Works great!"
}
```

---

## 📝 PUT /api/marketplace/tenant-items/{id}

Update a tenant's item configuration.

### Request

**Method:** `PUT`

**Headers:**
```http
X-Tenant-ID: <uuid>
Content-Type: application/json
```

**Body:**
```json
{
  "custom_name": "Updated ESG Check",
  "custom_parameters": {
    "min_score": 80,
    "provider": "Refinitiv"
  },
  "enabled_for_tenant": false
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/marketplace/tenant-items/tenant-item-uuid \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "Content-Type: application/json" \
  -d '{
    "custom_name": "Updated ESG Check",
    "enabled_for_tenant": false
  }'
```

### Response

**Status:** `200 OK`

**Body:**
```json
{
  "id": "tenant-item-uuid",
  "custom_name": "Updated ESG Check",
  "enabled_for_tenant": false,
  "updated_at": "2024-10-27T17:00:00Z"
}
```

---

## 🗑️ DELETE /api/marketplace/tenant-items/{id}

Remove an item from a tenant's platform.

### Request

**Method:** `DELETE`

**Headers:**
```http
X-Tenant-ID: <uuid>
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/marketplace/tenant-items/tenant-item-uuid \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

### Response

**Status:** `204 No Content`

No response body.

---

## ⭐ POST /api/marketplace/items/{id}/feedback

Submit a rating and/or feedback for an item.

### Request

**Method:** `POST`

**Path Parameters:**
| Parameter | Type | Example |
|-----------|------|---------|
| `id` | uuid | `550e8400-e29b-41d4-a716-446655440000` |

**Headers:**
```http
X-Tenant-ID: <uuid>
Content-Type: application/json
```

**Body:**
```json
{
  "rating": 5,
  "feedback": "This rule helped us catch compliance issues early!"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/marketplace/items/550e8400-e29b-41d4-a716-446655440000/feedback \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "Content-Type: application/json" \
  -d '{
    "rating": 5,
    "feedback": "Excellent!"
  }'
```

### Response

**Status:** `201 Created`

**Body:**
```json
{
  "id": "feedback-uuid",
  "marketplace_item_id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "00000000-0000-0000-0000-000000000000",
  "rating": 5,
  "feedback": "This rule helped us catch compliance issues early!",
  "created_at": "2024-10-27T17:15:00Z",
  "updated_at": "2024-10-27T17:15:00Z"
}
```

---

## 📊 GET /api/marketplace/items/{id}/feedback

Get feedback and ratings for an item.

### Request

**Method:** `GET`

**Path Parameters:**
| Parameter | Type | Example |
|-----------|------|---------|
| `id` | uuid | `550e8400-e29b-41d4-a716-446655440000` |

**Example:**
```bash
GET /api/marketplace/items/550e8400-e29b-41d4-a716-446655440000/feedback \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

### Response

**Status:** `200 OK`

**Body:**
```json
{
  "aggregate": {
    "total_ratings": 42,
    "average_rating": 4.7,
    "rating_distribution": {
      "5": 32,
      "4": 8,
      "3": 2,
      "2": 0,
      "1": 0
    }
  },
  "feedback_items": [
    {
      "id": "feedback-uuid-1",
      "rating": 5,
      "feedback": "Great rule!",
      "created_at": "2024-10-27T16:00:00Z"
    },
    // ... more feedback ...
  ],
  "has_current_tenant_feedback": true,
  "current_tenant_feedback": {
    "rating": 5,
    "feedback": "This rule helped us catch compliance issues early!",
    "created_at": "2024-10-27T17:15:00Z"
  }
}
```

---

## 💻 Integration Patterns

### Pattern 1: Browse & Add Items

```typescript
// React component
function MarketplaceIntegration() {
  const [items, setItems] = useState([]);
  const tenantId = useContext(TenantContext).id;

  useEffect(() => {
    // Fetch marketplace items
    fetch(`/api/marketplace/items`, {
      headers: { 'X-Tenant-ID': tenantId }
    })
    .then(r => r.json())
    .then(data => setItems(data.items));
  }, [tenantId]);

  const handleAdd = (itemId: string) => {
    fetch(`/api/marketplace/items/add-to-tenant`, {
      method: 'POST',
      headers: {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        marketplace_item_id: itemId,
        custom_name: 'My Rule'
      })
    })
    .then(r => r.json())
    .then(result => console.log('Added!', result));
  };

  return (
    <div>
      {items.map(item => (
        <div key={item.id}>
          {item.name}
          <button onClick={() => handleAdd(item.id)}>Add</button>
        </div>
      ))}
    </div>
  );
}
```

### Pattern 2: Load & Configure

```typescript
async function loadAndConfigure(itemId: string, tenantId: string) {
  // Get item details
  const itemRes = await fetch(`/api/marketplace/items/${itemId}`, {
    headers: { 'X-Tenant-ID': tenantId }
  });
  const item = await itemRes.json();

  // Get parameters
  const paramsRes = await fetch(
    `/api/marketplace/items/${itemId}/parameters`,
    { headers: { 'X-Tenant-ID': tenantId } }
  );
  const { parameters } = await paramsRes.json();

  // Show configuration UI with parameters
  return { item, parameters };
}
```

### Pattern 3: Track Usage

```typescript
// In your rule/calculation execution engine
async function executeRule(tenantItemId: string, tenantId: string) {
  try {
    const result = await executeRuleLogic();
    
    // Track successful execution
    await recordUsage(tenantId, tenantItemId, 'success');
    return result;
  } catch (error) {
    // Track failed execution
    await recordUsage(tenantId, tenantItemId, 'failure');
    throw error;
  }
}

async function recordUsage(
  tenantId: string,
  tenantItemId: string,
  status: 'success' | 'failure'
) {
  // This would call a usage tracking endpoint
  // (to be implemented in next phase)
  const today = new Date().toISOString().split('T')[0];
  
  // Upsert into marketplace_item_usage
  await db.query(`
    INSERT INTO marketplace_item_usage (
      tenant_id, marketplace_item_id, execution_date,
      execution_count, ${status}_count, last_result_status
    ) VALUES ($1, $2, $3, 1, 1, $4)
    ON CONFLICT (tenant_id, marketplace_item_id, execution_date)
    DO UPDATE SET
      execution_count = execution_count + 1,
      ${status}_count = ${status}_count + 1,
      last_result_status = $4
  `, [tenantId, tenantItemId, today, status]);
}
```

### Pattern 4: Rate Items

```typescript
async function rateMarketplaceItem(
  itemId: string,
  rating: number,
  feedback: string,
  tenantId: string
) {
  const response = await fetch(
    `/api/marketplace/items/${itemId}/feedback`,
    {
      method: 'POST',
      headers: {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ rating, feedback })
    }
  );
  
  return await response.json();
}
```

### Pattern 5: Batch Operations

```typescript
// Add multiple items at once
async function addMultipleItems(
  itemIds: string[],
  tenantId: string
) {
  return Promise.all(
    itemIds.map(itemId =>
      fetch(`/api/marketplace/items/add-to-tenant`, {
        method: 'POST',
        headers: {
          'X-Tenant-ID': tenantId,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ marketplace_item_id: itemId })
      })
    )
  );
}
```

---

## 🧪 Test Scenarios

### Scenario 1: Complete Workflow

```bash
# 1. List marketplace items
curl -H "X-Tenant-ID: $TENANT_ID" \
  http://localhost:8080/api/marketplace/items

# 2. Get item details
curl -H "X-Tenant-ID: $TENANT_ID" \
  http://localhost:8080/api/marketplace/items/$ITEM_ID

# 3. Get item parameters
curl -H "X-Tenant-ID: $TENANT_ID" \
  http://localhost:8080/api/marketplace/items/$ITEM_ID/parameters

# 4. Add to tenant
curl -X POST -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"marketplace_item_id": "'$ITEM_ID'"}' \
  http://localhost:8080/api/marketplace/items/add-to-tenant

# 5. Get tenant's items
curl -H "X-Tenant-ID: $TENANT_ID" \
  http://localhost:8080/api/marketplace/tenant-items

# 6. Rate item
curl -X POST -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"rating": 5, "feedback": "Great!"}' \
  http://localhost:8080/api/marketplace/items/$ITEM_ID/feedback

# 7. Remove item
curl -X DELETE -H "X-Tenant-ID: $TENANT_ID" \
  http://localhost:8080/api/marketplace/tenant-items/$TENANT_ITEM_ID
```

---

**API Version:** 1.0  
**Last Updated:** 2024-10-27  
**Status:** ✅ Ready for Integration
