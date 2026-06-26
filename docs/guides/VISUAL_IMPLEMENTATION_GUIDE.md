# Visual Implementation Guide

## 🎯 The Three Pillars of Your System

```
┌────────────────────────────────────────────────────────────────────┐
│                                                                     │
│                    API ENDPOINTS CATALOG SYSTEM                    │
│                                                                     │
├─────────────────┬──────────────────────────────┬──────────────────┤
│                 │                              │                  │
│   DISCOVERY     │      RELATIONSHIPS           │   DOCUMENTATION  │
│   ─────────────   ──────────────────────         ──────────────────│
│                 │                              │                  │
│ Find endpoints  │ Link endpoints to:           │ Rich metadata    │
│ for any entity  │ - Entities                   │ - Schemas        │
│                 │ - Datasources                │ - Parameters     │
│ Filter by:      │ - Operations                 │ - Examples       │
│ - Category      │                              │ - Versions       │
│ - Search        │ Discover context-aware       │                  │
│ - Method        │ operations                   │ Generate:        │
│ - Tags          │                              │ - OpenAPI specs  │
│                 │                              │ - Documentation  │
│                 │                              │ - SDKs           │
└─────────────────┴──────────────────────────────┴──────────────────┘
```

## 📊 Data Flow Visualization

### Creating a Rule (User Perspective)

```
User Opens Entity Manager
         ↓
Selects Entity
         ↓
Clicks "⚡ Validations" Tab
         ↓
ValidationRulesContainer Loads
         ↓
[useEffect Triggered]
         ↓
validationRulesService.listRules()
         ↓
HTTP GET /api-endpoints
    + tenant_id query param
    + X-Tenant-ID header
         ↓
Backend Route Handler
         ↓
Queries PostgreSQL
    WHERE tenant_id = ?
         ↓
Returns Rules List
         ↓
Component Updates State
         ↓
UI Renders Rules Table
         ↓
User Sees Rules
```

### Creating a New Rule (Technical Flow)

```
┌─────────────────────────────────────────────────────────────────────┐
│ FRONTEND                                                             │
│                                                                     │
│ onClick "New Rule"                                                  │
│   ↓                                                                  │
│ Show Create Form                                                    │
│   ↓                                                                  │
│ User Fills Form & Clicks Save                                      │
│   ↓                                                                  │
│ handleCreateRule() Called                                           │
│   ↓                                                                  │
│ validationRulesService.createRule(ruleData)                        │
│   ↓                                                                  │
└────────────────┬──────────────────────────────────────────────────┘
                 │
┌────────────────┴──────────────────────────────────────────────────┐
│ HTTP LAYER                                                          │
│                                                                     │
│ POST /validation-rules                                             │
│ ?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>            │
│ Headers:                                                           │
│   X-Tenant-ID: <TENANT_ID>                                       │
│   X-Tenant-Datasource-ID: <DATASOURCE_ID>                       │
│ Body: { name, description, condition, entityIds, status }       │
│   ↓                                                               │
└────────────────┬──────────────────────────────────────────────────┘
                 │
┌────────────────┴──────────────────────────────────────────────────┐
│ BACKEND                                                             │
│                                                                     │
│ handleCreateValidationRule()                                       │
│   ↓                                                               │
│ Validate Input                                                    │
│   ↓                                                               │
│ Generate UUID                                                    │
│   ↓                                                               │
│ INSERT INTO validation_rules                                     │
│   (id, tenant_id, name, description, ...)                       │
│   VALUES (?, ?, ?, ?, ...)                                      │
│   ↓                                                               │
│ Return Rule with ID                                              │
│   ↓                                                               │
└────────────────┬──────────────────────────────────────────────────┘
                 │
┌────────────────┴──────────────────────────────────────────────────┐
│ FRONTEND                                                             │
│                                                                     │
│ Response: { id, name, ... }                                       │
│   ↓                                                               │
│ Update State: rules = [...rules, newRule]                        │
│   ↓                                                               │
│ Close Form                                                        │
│   ↓                                                               │
│ Show Success Toast/Alert                                         │
│   ↓                                                               │
│ Table Updates to Show New Rule                                   │
│   ↓                                                               │
│ User Sees New Rule in List                                       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## 🗂️ File Organization

```
semlayer/
│
├── 📁 backend/internal/api/
│   │
│   ├── ✅ api_endpoints_catalog.go
│   │   ├─ APIEndpoint struct
│   │   ├─ EndpointParameter struct
│   │   ├─ EndpointExample struct
│   │   ├─ RegisterAPIEndpointsCatalogRoutes()
│   │   ├─ handleListAPIEndpoints()
│   │   ├─ handleCreateAPIEndpoint()
│   │   ├─ handleGetAPIEndpoint()
│   │   ├─ handleUpdateAPIEndpoint()
│   │   ├─ handleDeleteAPIEndpoint()
│   │   ├─ handleListAPIEndpointsByCategory()
│   │   ├─ handleSearchAPIEndpoints()
│   │   ├─ handleGetOpenAPISpec()
│   │   └─ handleGetEndpointDocumentation()
│   │
│   ├── ✅ api_endpoint_mapping_routes.go
│   │   ├─ EndpointEntityMapping struct
│   │   ├─ EndpointDatasourceMapping struct
│   │   ├─ RegisterEndpointMappingRoutes()
│   │   ├─ Entity mapping handlers (4 functions)
│   │   ├─ Datasource mapping handlers (4 functions)
│   │   ├─ handleGetEntityEndpoints()
│   │   └─ handleGetDatasourceEndpoints()
│   │
│   ├── ✅ api_endpoints_seeder.go
│   │   ├─ SeedAPIEndpointsCatalog()
│   │   └─ RegisterValidationEndpointMappings()
│   │
│   └── 📁 migrations/
│       └── ✅ 001_create_api_endpoints_catalog.sql
│           ├─ CREATE TABLE api_endpoints_catalog
│           ├─ CREATE TABLE api_endpoint_entity_mappings
│           ├─ CREATE TABLE api_endpoint_datasource_mappings
│           ├─ CREATE 8 indexes
│           ├─ CREATE 2 triggers
│           └─ Grant permissions
│
├── 📁 frontend/src/
│   │
│   ├── 📁 pages/
│   │   ├── ✅ EntityDetailsPage.tsx (UPDATED)
│   │   │   ├─ Import AdvancedRuleConfiguration
│   │   │   ├─ Import ValidationRule interface
│   │   │   ├─ ValidationRulesContainer component
│   │   │   └─ validationRules state
│   │   │
│   │   ├── ✅ EntityDetailsPage.module.css (UPDATED)
│   │   │   ├─ .validationRulesContainer
│   │   │   ├─ .validationRulesHeader
│   │   │   ├─ .validationRulesTitle
│   │   │   ├─ .validationRulesDescription
│   │   │   └─ .validationRulesCard
│   │   │
│   │   └── ✅ EntityConfigPageV2.tsx (UPDATED)
│   │       └─ Added validation tab
│   │
│   └── 📁 services/
│       └── 📋 validationRulesService.ts (DOCUMENTED, READY TO CREATE)
│           ├─ ValidationRule interface
│           ├─ ValidationRuleRequest interface
│           ├─ ValidationRuleExecutionResult interface
│           ├─ ValidationRulesService class
│           ├─ listRules()
│           ├─ getRule()
│           ├─ createRule()
│           ├─ updateRule()
│           ├─ deleteRule()
│           ├─ executeRule()
│           ├─ executeBatch()
│           ├─ getAuditTrail()
│           ├─ listValidationEndpoints()
│           └─ getEntityEndpoints()
│
└── 📁 Documentation/
    │
    ├── ✅ BACKEND_API_CATALOG_INTEGRATION.md
    │   ├─ Architecture overview (section 1)
    │   ├─ Database schema details (section 2)
    │   ├─ All 15 API endpoints (section 3)
    │   ├─ Seeding system (section 4)
    │   ├─ Classification system (section 5)
    │   ├─ Integration points (section 6)
    │   ├─ Best practices (section 7)
    │   └─ Future enhancements (section 8)
    │
    ├── ✅ FRONTEND_VALIDATION_RULES_INTEGRATION.md
    │   ├─ Service layer architecture (section 1)
    │   ├─ Complete TypeScript implementation (section 2)
    │   ├─ Updated EntityDetailsPage.tsx (section 3)
    │   ├─ Updated EntityConfigPageV2.tsx (section 4)
    │   ├─ Error handling (section 5)
    │   ├─ State management options (section 6)
    │   ├─ Testing strategies (section 7)
    │   └─ Deployment checklist (section 8)
    │
    ├── ✅ API_CATALOG_DEPLOYMENT_CHECKLIST.md
    │   ├─ Pre-deployment verification (section 1)
    │   ├─ Backend implementation checks (section 2)
    │   ├─ Frontend implementation checks (section 3)
    │   ├─ Staging deployment (section 4)
    │   ├─ Production deployment (section 5)
    │   ├─ Post-deployment validation (section 6)
    │   └─ Success criteria (section 7)
    │
    ├── ✅ API_CATALOG_QUICK_REFERENCE.md
    │   ├─ Overview (section 1)
    │   ├─ Quick start (section 2)
    │   ├─ All validation endpoints (section 3)
    │   ├─ API response examples (section 4)
    │   ├─ Classification reference (section 5)
    │   ├─ Common patterns (section 6)
    │   ├─ HTTP status codes (section 7)
    │   ├─ TypeScript types (section 8)
    │   ├─ Troubleshooting (section 9)
    │   └─ Performance benchmarks (section 10)
    │
    ├── ✅ API_CATALOG_IMPLEMENTATION_SUMMARY.md
    │   ├─ Implementation status (section 1)
    │   ├─ Deliverables (section 2)
    │   ├─ Architecture overview (section 3)
    │   ├─ Data relationships (section 4)
    │   ├─ Endpoints summary (section 5)
    │   ├─ Features delivered (section 6)
    │   ├─ Pre-seeded endpoints (section 7)
    │   ├─ Security features (section 8)
    │   ├─ Integration points (section 9)
    │   └─ Next steps (section 10)
    │
    ├── ✅ FRONTEND_BACKEND_INTEGRATION_ROADMAP.md
    │   ├─ Current status (section 1)
    │   ├─ Architecture diagram (section 2)
    │   ├─ File structure (section 3)
    │   ├─ Phase breakdown (section 4)
    │   ├─ Next actions (section 5)
    │   ├─ Key decision points (section 6)
    │   ├─ Performance expectations (section 7)
    │   ├─ Testing strategy (section 8)
    │   └─ Resources (section 9)
    │
    ├── ✅ API_DELIVERY_PACKAGE_SUMMARY.md
    │   ├─ Executive summary (section 1)
    │   ├─ What you're getting (section 2)
    │   ├─ Key features (section 3)
    │   ├─ System architecture (section 4)
    │   ├─ Deployment timeline (section 5)
    │   ├─ Implementation checklist (section 6)
    │   ├─ How to use this package (section 7)
    │   ├─ What's production-ready (section 8)
    │   ├─ Learning resources (section 9)
    │   └─ Next steps (section 10)
    │
    └── ✅ VISUAL_IMPLEMENTATION_GUIDE.md (THIS FILE)
        └─ Visual diagrams and organization
```

## 🔄 Component Interaction Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                        REACT COMPONENTS                          │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ EntityConfigPageV2 (Entity Manager Page)               │  │
│  │                                                         │  │
│  │  ┌──────────────────────────────────────────────────┐  │  │
│  │  │ Tabs:                                            │  │  │
│  │  │  - ⚙️ Entity                                     │  │  │
│  │  │  - 🔗 Related Objects                            │  │  │
│  │  │  - ⚡ Validations (NEW)                          │  │  │
│  │  │                                                  │  │  │
│  │  │   ┌──────────────────────────────────────────┐  │  │  │
│  │  │   │ ValidationRulesContainer Component       │  │  │  │
│  │  │   │                                          │  │  │  │
│  │  │   │ State:                                   │  │  │  │
│  │  │   │ - validationRules[]                      │  │  │  │
│  │  │   │ - loading: boolean                       │  │  │  │
│  │  │   │ - error?: string                         │  │  │  │
│  │  │   │ - editingRuleId?: string                 │  │  │  │
│  │  │   │                                          │  │  │  │
│  │  │   │ Methods:                                 │  │  │  │
│  │  │   │ - loadRules()                            │  │  │  │
│  │  │   │ - handleCreateRule()                     │  │  │  │
│  │  │   │ - handleUpdateRule()                     │  │  │  │
│  │  │   │ - handleDeleteRule()                     │  │  │  │
│  │  │   │ - handleExecuteRule()                    │  │  │  │
│  │  │   │                                          │  │  │  │
│  │  │   │ Renders:                                 │  │  │  │
│  │  │   │ ├─ Header with title & description      │  │  │  │
│  │  │   │ ├─ Error Alert (if error)               │  │  │  │
│  │  │   │ ├─ Create Form (if creating)            │  │  │  │
│  │  │   │ ├─ Edit Form (if editing)               │  │  │  │
│  │  │   │ └─ Rules Table with actions             │  │  │  │
│  │  │   └──────────────────────────────────────────┘  │  │  │
│  │  │                                                  │  │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  └─────────────────────────────────────────────────────────┘  │
│                            ↓                                    │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ Uses: validationRulesService                           │  │
│  │       (Service Layer)                                  │  │
│  └─────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                            ↓
┌──────────────────────────────────────────────────────────────────┐
│                   API SERVICE LAYER                              │
│                (validationRulesService.ts)                       │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ValidationRulesService Class                           │   │
│  │                                                        │   │
│  │ Methods:                                               │   │
│  │ ├─ listRules(page, limit, filters)                    │   │
│  │ ├─ getRule(ruleId)                                    │   │
│  │ ├─ createRule(ruleData)                               │   │
│  │ ├─ updateRule(ruleId, ruleData)                       │   │
│  │ ├─ deleteRule(ruleId)                                 │   │
│  │ ├─ executeRule(ruleId, data)                          │   │
│  │ ├─ executeBatch(records)                              │   │
│  │ ├─ getAuditTrail(ruleId)                              │   │
│  │ ├─ listValidationEndpoints()                          │   │
│  │ └─ getEntityEndpoints(entityId)                       │   │
│  └─────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
                            ↓ (HTTP + Headers)
┌──────────────────────────────────────────────────────────────────┐
│                     BACKEND API (Go)                             │
│                    (Chi HTTP Router)                             │
│                                                                  │
│  Route Handlers:                                                 │
│  ├─ GET /api-endpoints                                          │
│  ├─ POST /api-endpoints                                         │
│  ├─ GET /api-endpoints/{id}                                     │
│  ├─ PATCH /api-endpoints/{id}                                   │
│  ├─ DELETE /api-endpoints/{id}                                  │
│  ├─ GET /api-endpoints/category/{category}                      │
│  ├─ GET /api-endpoints/search                                   │
│  ├─ GET /api-endpoints/openapi                                  │
│  ├─ GET /api-endpoints/{id}/documentation                       │
│  ├─ GET /api-endpoints/{id}/entity-mappings                     │
│  ├─ POST /api-endpoints/{id}/entity-mappings                    │
│  ├─ DELETE /api-endpoints/{id}/entity-mappings/{entity-id}      │
│  ├─ GET /api-endpoints/{id}/datasource-mappings                 │
│  ├─ POST /api-endpoints/{id}/datasource-mappings                │
│  ├─ DELETE /api-endpoints/{id}/datasource-mappings/{id}         │
│  ├─ GET /entities/{id}/api-endpoints                            │
│  └─ GET /datasources/{id}/api-endpoints                         │
│                                                                  │
│  (Plus existing validation-rules routes)                         │
└──────────────────────────────────────────────────────────────────┘
                            ↓ (SQL)
┌──────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                           │
│                                                                  │
│  Tables:                                                         │
│  ├─ api_endpoints_catalog (metadata store)                      │
│  ├─ api_endpoint_entity_mappings (relationships)                │
│  └─ api_endpoint_datasource_mappings (relationships)            │
│                                                                  │
│  Plus original tables:                                           │
│  ├─ validation_rules                                            │
│  ├─ entities                                                    │
│  └─ datasources                                                 │
└──────────────────────────────────────────────────────────────────┘
```

## 📱 UI Flow

```
EntityManager Page
  ↓
  ├─ Tabs: [Entity] [Objects] [⚡ Validations]
  │
  └─ Click "⚡ Validations"
      ↓
      ValidationRulesContainer
        ├─ Header with "Validations for [Entity]"
        ├─ Description text
        │
        ├─ Toolbar with:
        │  ├─ [Refresh] button
        │  └─ [+ New Rule] button
        │
        ├─ Error Alert (if error)
        │
        ├─ Create Form (if creating)
        │  ├─ AdvancedRuleConfiguration
        │  ├─ [Cancel] button
        │  └─ [Save] button
        │
        ├─ Edit Form (if editing)
        │  ├─ AdvancedRuleConfiguration (pre-filled)
        │  ├─ [Cancel] button
        │  └─ [Save] button
        │
        └─ Rules Table
           ├─ Columns: Name | Description | Status | Actions
           ├─ Each row has: [Edit] [Execute] [Delete]
           └─ Empty state: "No rules, create your first"
```

## 🔐 Tenant Scope Flow

```
User selects Tenant/Product/Datasource
         ↓
Cached in localStorage:
├─ selected_tenant: { id, display_name }
├─ selected_product: { id, alpha_product: { product_name } }
└─ selected_datasource: { id, source_name }
         ↓
Page loads / Component mounts
         ↓
TenantContext.getCurrentScope()
         ↓
Returns: { tenant_id, datasource_id }
         ↓
Service layer adds to all requests:
├─ Query params: ?tenant_id=X&datasource_id=Y
├─ Headers: X-Tenant-ID: X, X-Tenant-Datasource-ID: Y
└─ Backend validates and filters
         ↓
All data isolated by tenant
```

## ✅ Validation Rules Pre-Seeded

```
When system starts:
  ↓
SeedAPIEndpointsCatalog(db, tenantID)
  ↓
Creates 8 Validation Rule Endpoints:
  ├─ 1. List Validation Rules (GET)
  ├─ 2. Create Validation Rule (POST)
  ├─ 3. Get Validation Rule (GET)
  ├─ 4. Update Validation Rule (PATCH)
  ├─ 5. Delete Validation Rule (DELETE)
  ├─ 6. Execute Single Rule (POST)
  ├─ 7. Execute Batch Rules (POST)
  └─ 8. Get Audit Trail (GET)
  ↓
For each endpoint:
  ├─ Creates record in api_endpoints_catalog
  ├─ Adds metadata (schema, examples, params)
  └─ Tags with "validation"
  ↓
RegisterValidationEndpointMappings()
  ├─ Links each endpoint to entities
  ├─ Sets relationship_type: "can_read", "can_execute"
  └─ Results in discoverable operations per entity
```

---

**This visual guide complements the detailed documentation. For complete information, see the 6 comprehensive guide files.**
