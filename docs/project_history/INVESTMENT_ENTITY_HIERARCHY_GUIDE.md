# Investment Entity Hierarchy & Integration Guide

## Overview

This guide provides comprehensive documentation for adding and managing investment entities using the Addepar-compatible business entity builder. The system supports a full hierarchy of 50+ investment types with enforced parent-child relationships and multi-tenant isolation.

## Table of Contents

1. [Entity Architecture](#entity-architecture)
2. [Hierarchical Model Types](#hierarchical-model-types)
3. [Setup Instructions](#setup-instructions)
4. [API Reference](#api-reference)
5. [Usage Examples](#usage-examples)
6. [Best Practices](#best-practices)

---

## Entity Architecture

### Core Concepts

The investment entity system is built on three foundational concepts:

```
┌─────────────────────────────────────────┐
│     Hierarchical Entity Types           │
│  (household, person_node, trust, etc)   │
└────────┬────────────────────────────────┘
         │
         ├─► model_types (Discriminator)
         │   • 50+ investment types
         │   • Category grouping (security, fund, alternative, etc)
         │   • Ownership type hints
         │
         ├─► positions (Relationships)
         │   • owner_id → owned_id edges
         │   • Multi-parent support (DAG)
         │   • Ownership percentages
         │
         └─► entity_hierarchy_rules (Enforcement)
             • Allowed parent → child combinations
             • Ownership type constraints
             • Audit logging
```

### Supported Entity Categories

| Category | Types | Use Case |
|----------|-------|----------|
| **organization** | household, person_node, prospect, manager, trust | Primary containers and clients |
| **fund** | fund, hedge_fund, private_equity_fund, managed_partnership, holding_company | Fund vehicles and partnerships |
| **container** | financial_account, sleeve, vehicle | Custody and allocation containers |
| **security** | stock, bond, etf, mutual_fund, etc. | Tradeable securities |
| **derivative** | option, futures, forward, warrant | Derivative instruments |
| **alternative** | real_estate, art, private_equity, venture_capital | Alternative assets |
| **insurance** | annuity | Insurance products |
| **debt** | loan, promissory_note | Debt instruments |
| **cash** | cash | Cash holdings |
| **digital** | digital_asset | Cryptocurrencies |
| **custom** | generic_asset | Catch-all for unspecified assets |

---

## Hierarchical Model Types

### Complete Entity Type Listing

All 50+ investment types are now available. Here are the key ones organized by hierarchy level:

#### Level 1: Top-Level Containers (Roots)

These entities have no parents and act as portfolio containers:

```
• household          - Primary portfolio container
• prospect           - Prospective client record
```

#### Level 2: Organizational Entities

Children of households, managing relationships:

```
• person_node        - Individual client/beneficiary
• trust              - Legal trust entity
• holding_company    - Corporate structure
• managed_partnership - Fund partnership
```

#### Level 3: Account & Sleeve Containers

Sub-allocations and custody accounts:

```
• financial_account  - Brokerage/custodial account
• sleeve             - Portfolio sub-allocation
• vehicle            - Investment vehicle wrapper
```

#### Level 4: Asset Holdings

Tradeable and alternative assets (leaf nodes):

```
Securities:
• stock              - Common equity
• bond               - Fixed income
• etf                - Exchange-traded fund
• mutual_fund        - Open-end fund
• reit               - Real estate investment trust

Derivatives:
• option             - Option contract
• futures_contract   - Futures contract
• forward_contract   - Forward contract

Alternatives:
• real_estate        - Real property
• private_equity_fund - PE fund investment
• venture_capital    - VC investment
• hedge_fund         - Hedge fund investment

Commodities:
• digital_asset      - Cryptocurrency
• art                - Art & collectibles
• cash               - Currency

And 20+ more...
```

### Valid Hierarchy Examples

```
Example 1: Individual Investor
─────────────────────────────
household
├── person_node (Alice)
│   ├── financial_account (Brokerage)
│   │   ├── stock (AAPL)
│   │   ├── etf (SPY)
│   │   └── cash (USD)
│   └── sleeve (Conservative)
│       ├── bond (US Treasury)
│       └── money_market_fund

Example 2: Family Office
───────────────────────
household (Family)
├── person_node (Trustee)
│   └── financial_account
│       └── etf
├── trust (Irrevocable)
│   ├── sleeve (Growth)
│   │   ├── private_equity_fund
│   │   └── venture_capital
│   └── sleeve (Income)
│       └── reit
└── managed_partnership (Fund of Funds)
    ├── hedge_fund
    └── private_equity_fund

Example 3: Diversified Portfolio
────────────────────────────────
household
├── financial_account (Stocks)
│   ├── stock (Apple)
│   ├── stock (Microsoft)
│   └── closed_end_fund
├── sleeve (Bonds)
│   ├── bond (Corporate)
│   └── certificate_of_deposit
├── sleeve (Alternatives)
│   ├── real_estate
│   ├── art
│   └── collectible
└── sleeve (Digital)
    └── digital_asset (Bitcoin)
```

---

## Setup Instructions

### 1. Database Migration

Run the hierarchy schema SQL to create tables and functions:

```bash
# Apply the migration
psql -U postgres -d alpha -f portfolio-management/database/investment_entities_hierarchy.sql

# Verify tables were created
psql -U postgres -d alpha -c "\dt | grep hierarchy"
```

Expected tables:
- `entity_hierarchy_rules` - Allowed relationships
- `entity_hierarchy_audit_log` - Change tracking
- `model_types` - Entity type definitions (populated with 50+ types)

### 2. Backend Integration

Add the hierarchy service to your Go application:

```go
package main

import (
	"github.com/your-org/semlayer/portfolio-management/backend/internal/hierarchy"
	"gorm.io/gorm"
)

func initializeHierarchyService(db *gorm.DB) *hierarchy.HierarchyService {
	return hierarchy.NewHierarchyService(db)
}
```

### 3. API Route Registration

Add hierarchy endpoints to your router:

```go
router.GET("/api/hierarchy/rules", getHierarchyRules)
router.GET("/api/hierarchy/summary", getHierarchySummary)
router.GET("/api/hierarchy/:entityID", getEntityHierarchy)
router.GET("/api/hierarchy/stats", getHierarchyStats)
router.POST("/api/hierarchy/validate", validateHierarchy)
router.POST("/api/hierarchy/bulk", bulkCreatePositions)
router.POST("/api/hierarchy/import", importHierarchyRules)
```

### 4. Frontend Integration

Update React components to support hierarchy visualization:

```typescript
// Add hierarchy tree view component
import { EntityHierarchyTree } from '@/components/EntityHierarchyTree';

export function EntityBuilderPage() {
  const [hierarchy, setHierarchy] = useState<EntityHierarchyTree | null>(null);

  useEffect(() => {
    // Fetch and display entity hierarchy
    fetchHierarchy().then(setHierarchy);
  }, [selectedEntity]);

  return <EntityHierarchyTree root={hierarchy?.roots[0]} />;
}
```

---

## API Reference

### Validate Hierarchy

```http
POST /api/hierarchy/validate
Content-Type: application/json

{
  "tenant_id": "uuid",
  "parent_model_type": "household",
  "child_model_type": "person_node"
}
```

**Response (200 OK):**
```json
{
  "valid": true,
  "errors": [],
  "warnings": [],
  "parent_model_type": "household",
  "child_model_type": "person_node",
  "matching_rules": [
    {
      "id": "uuid",
      "parent_model_type": "household",
      "child_model_type": "person_node",
      "allowed": true,
      "ownership_types": ["PERCENT_BASED"],
      "description": "Household contains clients"
    }
  ],
  "recommended_parents": ["household"],
  "recommended_children": ["financial_account", "sleeve"]
}
```

### Get Hierarchy Rules

```http
GET /api/hierarchy/rules?tenant_id=uuid
```

**Response:**
```json
{
  "rules": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "parent_model_type": "household",
      "child_model_type": "person_node",
      "allowed": true,
      "ownership_types": ["PERCENT_BASED"],
      "description": "Household contains clients",
      "created_at": "2025-10-30T00:00:00Z"
    }
    // ... 100+ more rules
  ]
}
```

### Get Entity Hierarchy

```http
GET /api/hierarchy/{entityID}?depth=-1&include_stats=true
```

**Response:**
```json
{
  "root": {
    "id": "uuid",
    "tenant_id": "uuid",
    "model_type": "household",
    "display_name": "Smith Family Portfolio",
    "parent_id": null,
    "depth": 0,
    "path_ids": ["uuid"],
    "path_names": ["Smith Family Portfolio"],
    "level": 0,
    "children": [
      {
        "id": "uuid",
        "model_type": "person_node",
        "display_name": "Alice Smith",
        "path_ids": ["uuid", "uuid"],
        "children": [
          {
            "model_type": "financial_account",
            "display_name": "Brokerage Account",
            "children": [
              {"model_type": "stock", "display_name": "AAPL"},
              {"model_type": "etf", "display_name": "SPY"}
            ]
          }
        ]
      }
    ]
  },
  "depth": 4,
  "stats": {
    "total_entities": 25,
    "total_positions": 24,
    "max_depth": 4,
    "top_level_entities": 1,
    "leaf_nodes": 10,
    "allowed_rules": 98,
    "disallowed_rules": 2
  }
}
```

### Bulk Create Positions

```http
POST /api/hierarchy/bulk
Content-Type: application/json

{
  "operations": [
    {
      "operation": "CREATE",
      "owner_id": "uuid",
      "owned_id": "uuid",
      "ownership_percentage": 100,
      "ownership_type": "PERCENT_BASED",
      "incepting_date": "2025-10-30T00:00:00Z"
    },
    {
      "operation": "CREATE",
      "owner_id": "uuid",
      "owned_id": "uuid",
      "ownership_percentage": 50,
      "ownership_type": "PERCENT_BASED"
    }
  ],
  "validate": true
}
```

**Response:**
```json
{
  "successful": 2,
  "failed": 0,
  "results": [
    {
      "operation": {...},
      "success": true,
      "message": "Position created",
      "position_id": "uuid"
    }
  ],
  "errors_summary": []
}
```

### Import Hierarchy Rules

```http
POST /api/hierarchy/import
Content-Type: application/json

{
  "rules": [
    {
      "parent_model_type": "household",
      "child_model_type": "person_node",
      "ownership_types": ["PERCENT_BASED"],
      "description": "Custom rule"
    }
  ]
}
```

---

## Usage Examples

### Creating a Household with Nested Holdings

```typescript
// Step 1: Create household entity
const household = await createEntity({
  tenant_id: 'uuid',
  model_type: 'household',
  display_name: 'Smith Family',
  entity_attributes: {
    currency: 'USD',
    status: 'active'
  }
});

// Step 2: Create person entity
const person = await createEntity({
  tenant_id: 'uuid',
  model_type: 'person_node',
  display_name: 'Alice Smith'
});

// Step 3: Create relationship (household → person)
const hierarchyResult = await api.post('/api/hierarchy/validate', {
  parent_model_type: 'household',
  child_model_type: 'person_node'
});

if (hierarchyResult.valid) {
  const position1 = await createPosition({
    owner_id: household.id,
    owned_id: person.id,
    ownership_percentage: 100,
    ownership_type: 'PERCENT_BASED'
  });
}

// Step 4: Add financial account under person
const account = await createEntity({
  tenant_id: 'uuid',
  model_type: 'financial_account',
  display_name: 'Brokerage Account'
});

const position2 = await createPosition({
  owner_id: person.id,
  owned_id: account.id,
  ownership_percentage: 100
});

// Step 5: Add holdings under account
const stock = await createEntity({
  tenant_id: 'uuid',
  model_type: 'stock',
  display_name: 'Apple Inc (AAPL)'
});

const position3 = await createPosition({
  owner_id: account.id,
  owned_id: stock.id,
  ownership_percentage: 25000 // Value-based
});

// Result: Complete hierarchy is created and validated
```

### Bulk Import with Validation

```typescript
async function setupFamilyOfficeStructure(tenantId: string) {
  const entities = [
    {
      type: 'household',
      name: 'Legacy Fund',
      children: [
        {
          type: 'trust',
          name: 'Irrevocable Trust',
          children: [
            {type: 'sleeve', name: 'Growth'},
            {type: 'sleeve', name: 'Income'}
          ]
        },
        {
          type: 'person_node',
          name: 'Trustee',
          children: [
            {type: 'financial_account', name: 'Primary Account'}
          ]
        }
      ]
    }
  ];

  // Recursively create hierarchy
  async function createHierarchy(parent, parentId) {
    for (const child of parent.children || []) {
      // Validate relationship
      const validation = await api.post('/api/hierarchy/validate', {
        tenant_id: tenantId,
        parent_model_type: parent.type,
        child_model_type: child.type
      });

      if (!validation.valid) {
        console.error(`Invalid: ${parent.type} → ${child.type}`);
        continue;
      }

      // Create entity
      const childEntity = await createEntity({
        tenant_id: tenantId,
        model_type: child.type,
        display_name: child.name
      });

      // Create position
      if (parentId) {
        await createPosition({
          owner_id: parentId,
          owned_id: childEntity.id,
          ownership_percentage: 100
        });
      }

      // Recurse
      await createHierarchy(child, childEntity.id);
    }
  }

  await createHierarchy(entities[0], null);
}
```

### Query Hierarchy and Generate Report

```typescript
async function generatePortfolioReport(householdId: string) {
  // Get complete hierarchy
  const hierarchy = await api.get(`/api/hierarchy/${householdId}?depth=-1&include_stats=true`);

  // Extract summary stats
  console.log('Portfolio Summary:', {
    total_entities: hierarchy.stats.total_entities,
    max_depth: hierarchy.stats.max_depth,
    leaf_assets: hierarchy.stats.leaf_nodes
  });

  // Get rules for this tenant
  const rules = await api.get(`/api/hierarchy/rules?tenant_id=${tenantId}`);
  console.log('Allowed relationships:', rules.length);

  // Validate consistency
  const consistency = await api.post('/api/hierarchy/validate-consistency', {
    tenant_id: tenantId
  });

  if (consistency.issues.length > 0) {
    console.warn('Hierarchy issues found:', consistency.issues);
  }

  return {
    hierarchy,
    stats: hierarchy.stats,
    rules_count: rules.length,
    consistency_check: consistency
  };
}
```

---

## Best Practices

### 1. Hierarchy Design

**DO:**
- ✅ Create top-level households first
- ✅ Keep ownership percentages consistent (sum to 100%)
- ✅ Use sleeves for major portfolio allocations
- ✅ Validate before creating positions

**DON'T:**
- ❌ Create circular references (A → B → A)
- ❌ Skip hierarchy validation
- ❌ Mix ownership types within same relationship
- ❌ Create positions between unrelated entity types

### 2. Multi-Tenant Safety

```go
// Always include tenant_id in queries
func GetHierarchyRules(tenantID string) ([]HierarchyRule, error) {
    return service.GetHierarchyRules(context.Background(), tenantID)
    // ✅ Filtered by tenant_id
}

// Use ABAC policies to enforce access
func (h *HierarchyHandler) CreatePosition(c *gin.Context) {
    tenantID := c.GetString("tenant_id") // From auth middleware
    
    // Verify user has access to this tenant
    if !h.abac.Can(c, "create", "position", map[string]any{
        "tenant_id": tenantID,
        "owner_id": req.OwnerID,
        "owned_id": req.OwnedID,
    }) {
        c.JSON(403, gin.H{"error": "Forbidden"})
        return
    }
}
```

### 3. Audit Logging

Always log significant hierarchy changes:

```go
// After creating position
auditLog := &HierarchyAuditLog{
    TenantID:        tenantID,
    EntityID:        ownedID,
    PositionID:      &positionID,
    Action:          AuditActionCreate,
    ParentModelType: parent.ModelType,
    ChildModelType:  child.ModelType,
    CreatedBy:       &userID,
    Reason:          "Created via bulk import",
}
service.LogHierarchyAudit(ctx, auditLog)
```

### 4. Performance Optimization

```sql
-- For large hierarchies, use indexes
CREATE INDEX idx_positions_owner_owned ON positions(owner_id, owned_id);
CREATE INDEX idx_positions_owned_owner ON positions(owned_id, owner_id);

-- Cache hierarchy summaries for reporting
REFRESH MATERIALIZED VIEW entity_hierarchy_summary;

-- Limit recursive depth in queries
SELECT * FROM entity_hierarchy_tree
WHERE depth <= 5  -- Don't retrieve entire tree if not needed
```

### 5. Error Handling

```typescript
async function safeCreatePosition(owner, owned) {
  try {
    // Validate first
    const validation = await api.post('/api/hierarchy/validate', {
      parent_model_type: owner.model_type,
      child_model_type: owned.model_type
    });

    if (!validation.valid) {
      throw new Error(`Invalid hierarchy: ${validation.errors.join(', ')}`);
    }

    // Create with fallback
    return await createPosition({owner, owned});
  } catch (error) {
    console.error('Position creation failed:', {
      owner: owner.id,
      owned: owned.id,
      error: error.message
    });
    
    // Notify user or fallback
    throw error;
  }
}
```

---

## Advanced Features

### Hierarchy Visualization

```typescript
// Generate Mermaid diagram
const response = await api.get(
  `/api/hierarchy/${householdId}?format=mermaid`
);

// Display in markdown
<div className="mermaid">
  {response.mermaid}
</div>

// Or generate GraphViz DOT
const dotFormat = await api.get(
  `/api/hierarchy/${householdId}?format=dot`
);

// Render with D3.js or Graphviz
```

### Custom Hierarchy Rules

```typescript
// Add firm-specific hierarchy rule
await api.post('/api/hierarchy/rules', {
  tenant_id: tenantId,
  parent_model_type: 'household',
  child_model_type: 'custom_type',
  allowed: true,
  ownership_types: ['PERCENT_BASED'],
  description: 'Custom entity relationship'
});
```

### Hierarchy Change Events

```typescript
// Subscribe to hierarchy changes
hierarchyService.on('position:created', (event) => {
  console.log('New relationship:', {
    owner: event.owner_id,
    owned: event.owned_id
  });
  
  // Trigger UI update or downstream processing
});
```

---

## Migration & Troubleshooting

### Common Issues

**Issue: "Invalid hierarchy: X → Y"**
- Solution: Check entity_hierarchy_rules table for allowed relationship
- Use: `/api/hierarchy/validate` endpoint to verify before creating

**Issue: Circular reference detected**
- Solution: Review ownership chain, ensure DAG structure
- Use: `/api/hierarchy/validate-consistency` to find issues

**Issue: Performance degradation with deep hierarchies**
- Solution: Use depth limit in queries, cache intermediate results
- Use: Materialized views for frequently accessed hierarchies

---

## Support & Reference

For additional information:
- Database schema: `portfolio-management/database/investment_entities_hierarchy.sql`
- Go models: `portfolio-management/backend/internal/hierarchy/models.go`
- Service layer: `portfolio-management/backend/internal/hierarchy/service.go`
- Frontend components: `frontend/src/components/EntityHierarchyTree.tsx`

---

**Version:** 1.0.0  
**Last Updated:** October 30, 2025  
**Status:** ✅ Production Ready
