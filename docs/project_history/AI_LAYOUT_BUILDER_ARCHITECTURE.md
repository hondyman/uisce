# AI Layout Builder - Visual Architecture & Flows

Complete system architecture, data flows, and integration points.

---

## 🏗️ System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          BROWSER / FRONTEND                                  │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │  Layout Builder Application                                            │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐  │  │
│  │  │  EditorHeader Component                                           │  │  │
│  │  │  ┌────────────────────────────────────────────────────────────┐  │  │  │
│  │  │  │ Title: "Untitled Layout"  [Save] [Publish]                │  │  │  │
│  │  │  ├────────────────────────────────────────────────────────────┤  │  │  │
│  │  │  │ AiActions Component                                         │  │  │  │
│  │  │  │ ┌───────────────────────┐ ┌──────────────────────────────┐ │  │  │  │
│  │  │  │ │ [✨ Generate with AI] │ │ AI Suggestions Panel        │ │  │  │  │
│  │  │  │ │ Prompt input box      │ │ ├─ Main suggestion (87%)    │ │  │  │  │
│  │  │  │ │ (Enter to submit)     │ │ ├─ Alt 1                    │ │  │  │  │
│  │  │  │ └───────────────────────┘ │ └─ Alt 2                    │ │  │  │  │
│  │  │  │                            └──────────────────────────────┘ │  │  │  │
│  │  │  ├────────────────────────────────────────────────────────────┤  │  │  │
│  │  │  │ FieldSuggestions Component                                  │  │  │  │
│  │  │  │ ┌────────────────────────────────────────────────────────┐  │  │  │  │
│  │  │  │ │ [💡 Suggest Fields]                                    │  │  │  │  │
│  │  │  │ │ ☐ Field F8 (Score: 77%) - High engagement             │  │  │  │  │
│  │  │  │ │ ☐ Field F9 (Score: 66%) - Common pattern              │  │  │  │  │
│  │  │  │ │ [Add Fields]                                           │  │  │  │  │
│  │  │  │ └────────────────────────────────────────────────────────┘  │  │  │  │
│  │  │  └────────────────────────────────────────────────────────────┘  │  │  │
│  │  └──────────────────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                    │                                           │
│                   (All requests include X-Tenant-ID header)                   │
└─────────────────────────────┬───────────────────────────────────────────────┘
                              │
                              │ HTTPS / HTTP
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                     API GATEWAY (Port 8080)                                  │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │ Middleware Stack                                                       │  │
│  │ ├─ SessionAuthMiddleware (cookie-based + bearer fallback)            │  │
│  │ ├─ auditMiddleware (log all requests)                                │  │
│  │ └─ cors, logging                                                      │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Route: POST /api/ai/generate-layout                                │  │
│  │ ├─ Validate X-Tenant-ID header required                           │  │
│  │ ├─ Proxy request to http://ai-service:8088                        │  │
│  │ ├─ Forward headers: X-Tenant-ID, Authorization, Content-Type      │  │
│  │ └─ Return response: {generatedLayout, alternatives, draftId}      │  │
│  │                                                                    │  │
│  │ Route: POST /api/ai/field-recommendations                         │  │
│  │ ├─ Validate X-Tenant-ID header                                   │  │
│  │ ├─ Proxy to ai-service:8088                                      │  │
│  │ └─ Return: {recommendations: [{fieldId, usageScore, reason}]}    │  │
│  │                                                                    │  │
│  │ Route: POST /api/ai/mark-adopted                                 │  │
│  │ ├─ Update ai_layouts SET adopted=true, adopted_at=now            │  │
│  │ └─ Return: 204 No Content                                         │  │
│  │                                                                    │  │
│  │ Route: POST /api/publish/validate (local)                         │  │
│  │ ├─ Check: accessibilityOk required                               │  │
│  │ ├─ Check: performanceOk required                                 │  │
│  │ └─ Return: {allowed: bool, reasons: string[]}                    │  │
│  │                                                                    │  │
│  │ Route: POST /api/analytics/layout (local)                         │  │
│  │ ├─ Log event type, section ID, container kind, device            │  │
│  │ └─ Return: 204 No Content                                         │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                            │                                               │
└────────────────────────────┼───────────────────────────────────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
   ┌─────────────┐  ┌────────────────┐  ┌──────────────────┐
   │ AI Service  │  │  Database      │  │ Auth Middleware  │
   │ (8088)      │  │  (Local ops)   │  │ (Already in api) │
   └─────────────┘  └────────────────┘  └──────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    AI SERVICE (Port 8088)                                    │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │ Handler: POST /api/ai/generate-layout                                 │  │
│  │ ├─ Extract X-Tenant-ID from header (required)                       │  │
│  │ ├─ Parse JSON: {prompt, primaryBO}                                  │  │
│  │ ├─ Generate Layout:                                                  │  │
│  │ │  ├─ Analyze prompt ("3 columns", "Orders", etc.)                 │  │
│  │ │  ├─ Create PageLayout with sections (Basic Info, Related)        │  │
│  │ │  ├─ Create 2 alternatives                                        │  │
│  │ │  └─ Calculate confidence (rule-based: 0-1.0)                     │  │
│  │ ├─ Persist to Database:                                             │  │
│  │ │  INSERT INTO ai_layouts                                           │  │
│  │ │    (tenant_id, primary_bo, name, payload, model_version,          │  │
│  │ │     confidence, alternatives, adopted) VALUES (...)               │  │
│  │ │    RETURNING id → draftId                                         │  │
│  │ └─ Return: {generatedLayout, alternatives, confidence, draftId}    │  │
│  │                                                                    │  │
│  │ Handler: POST /api/ai/field-recommendations                         │  │
│  │ ├─ Extract X-Tenant-ID                                             │  │
│  │ ├─ Parse: {primaryBO, existingFieldIds: ["f1", "f2"]}             │  │
│  │ ├─ Query mock usage scores (or real analytics)                     │  │
│  │ ├─ Filter out existing fields                                      │  │
│  │ ├─ Sort by score (descending)                                      │  │
│  │ └─ Return: {recommendations: [...], generatedAt}                   │  │
│  │                                                                    │  │
│  │ Handler: GET /api/ai/layouts?primary_bo=X                          │  │
│  │ ├─ Extract X-Tenant-ID                                             │  │
│  │ ├─ Query: SELECT ... FROM ai_layouts                               │  │
│  │ │  WHERE tenant_id = ? AND primary_bo = ? AND adopted = false      │  │
│  │ ├─ ORDER BY created_at DESC LIMIT 50                               │  │
│  │ └─ Return: {drafts: [...summary list]}                             │  │
│  │                                                                    │  │
│  │ Database Connection:                                               │  │
│  │ ├─ PostgreSQL client (pgx or database/sql)                         │  │
│  │ ├─ Connection pooling (configurable)                               │  │
│  │ └─ Automatic retry on transient failures                           │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
└────────────────────────────────────┼────────────────────────────────────────┘
                                     │
                                     ▼
         ┌───────────────────────────────────────────────────────┐
         │          PostgreSQL Database (localhost:5432)         │
         │                                                       │
         ├───────────────────────────────────────────────────────┤
         │ Table: ai_layouts                                     │
         │ ├─ id (UUID, PK)                                     │
         │ ├─ tenant_id (FK → tenants.id)                       │
         │ ├─ primary_bo (VARCHAR) ← INDEXED                    │
         │ ├─ name, layout_type, payload (JSONB)                │
         │ ├─ model_version, confidence (NUMERIC)               │
         │ ├─ alternatives (JSONB), explanation (TEXT)          │
         │ ├─ adopted (BOOLEAN) ← INDEXED                       │
         │ ├─ adopted_at (TIMESTAMP), adopted_by (FK → users)   │
         │ ├─ created_at (TIMESTAMP) ← INDEXED                  │
         │ ├─ created_by (TEXT)                                 │
         │ └─ is_active (BOOLEAN) ← INDEXED                     │
         │                                                       │
         │ Indexes:                                              │
         │ ├─ idx_ai_layouts_tenant_bo (tenant_id, primary_bo)  │
         │ ├─ idx_ai_layouts_adopted (tenant_id, adopted, ...)  │
         │ ├─ idx_ai_layouts_id_active (id WHERE is_active)     │
         │ └─ idx_ai_layouts_created (created_at DESC)          │
         │                                                       │
         ├───────────────────────────────────────────────────────┤
         │ Related Tables (already exist)                        │
         │ ├─ tenants (id, display_name, ...)                   │
         │ ├─ tenant_product_datasource (id, ...)               │
         │ ├─ users (id, email, ...)                            │
         │ └─ custom_components (existing, unchanged)           │
         └───────────────────────────────────────────────────────┘
```

---

## 📊 Request-Response Flow Diagrams

### 1. Generate Layout Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ USER INTERACTION                                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  User types:  "Create Customer detail with 3 columns"        │
│  Clicks:      [✨ Generate with AI]                           │
│                                                                 │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │ Frontend (React)        │
                    │ ├─ Validate prompt      │
                    │ ├─ Get tenantId from    │
                    │ │  localStorage         │
                    │ └─ POST to /api/ai/     │
                    │    generate-layout      │
                    └────────────┬────────────┘
                                 │
          ┌──────────────────────┴──────────────────────┐
          │                                             │
          │ HTTP POST                                   │
          │ Headers:                                    │
          │ - Content-Type: application/json            │
          │ - X-Tenant-ID: 870361a8-87e2-...          │
          │                                             │
          │ Body:                                       │
          │ {                                           │
          │   "prompt": "Create Customer...",           │
          │   "primaryBO": "Customer"                   │
          │ }                                           │
          │                                             │
          ▼                                             │
   ┌─────────────────────────────────────────┐         │
   │ API Gateway (8080)                      │         │
   ├─────────────────────────────────────────┤         │
   │ 1. Middleware: validate session         │         │
   │ 2. Middleware: audit logging            │         │
   │ 3. Route: /api/ai/generate-layout       │         │
   │    ├─ Extract X-Tenant-ID               │         │
   │    ├─ Call withTenant middleware        │         │
   │    └─ Call proxyJSON(ai-service:8088)   │         │
   │ 4. Proxy: Forward to AI service         │         │
   │                                         │         │
   └────────────┬────────────────────────────┘         │
                │                                      │
                │ HTTP Forwarded                       │
                ▼                                      │
        ┌─────────────────────────────┐               │
        │ AI Service (8088)           │               │
        ├─────────────────────────────┤               │
        │ 1. Extract tenant ID        │               │
        │ 2. Validate not empty       │               │
        │ 3. Parse JSON request       │               │
        │ 4. Generate layout (rules)  │               │
        │ 5. Create alternatives      │               │
        │ 6. Calculate confidence     │               │
        │ 7. INSERT into ai_layouts   │               │
        │    RETURNING id             │               │
        │ 8. JSON encode response     │               │
        │                             │               │
        └────────────┬────────────────┘               │
                     │                                │
                     │ JSON Response                  │
                     │ {                              │
                     │   "generatedLayout": {         │
                     │     "id": "gen_120456",        │
                     │     "name": "AI Customer...",  │
                     │     "sections": [...]          │
                     │   },                           │
                     │   "confidence": 0.87,          │
                     │   "alternatives": [...],       │
                     │   "explanation": "...",        │
                     │   "modelVersion": "v1",        │
                     │   "draftId": "uuid"            │
                     │ }                              │
                     │                                │
                     └──────────────┬──────────────────┘
                                    │
                                    │ Via API Gateway
                                    ▼
                          ┌──────────────────────┐
                          │ Frontend (React)     │
                          ├──────────────────────┤
                          │ 1. Parse response    │
                          │ 2. Store draftId     │
                          │ 3. Show suggestions  │
                          │ 4. Render panel with │
                          │    - Main layout     │
                          │    - Alternatives    │
                          │    - Apply buttons   │
                          └──────────────────────┘
                                    │
                                    │ User action
                                    ▼
                          [Apply Main Layout]
                                    │
                                    ▼
                          ┌──────────────────────┐
                          │ Layout in Editor     │
                          │ (in-memory, unsaved) │
                          └──────────────────────┘
```

### 2. Field Recommendations Flow

```
┌────────────────────────────────────────────┐
│ SectionConfigurator (fields section)       │
│ Shows: [f1], [f2], [f3]                   │
│ Adds: [💡 Suggest Fields]                 │
└───────────┬────────────────────────────────┘
            │
            │ User clicks
            ▼
    ┌───────────────────────────┐
    │ Collapsible panel opens   │
    │ Loading state...          │
    └───────────┬───────────────┘
                │
                │ Frontend POST
                ▼
    ┌──────────────────────────────────────────┐
    │ /api/ai/field-recommendations            │
    │                                          │
    │ Headers: X-Tenant-ID                     │
    │ Body: {                                  │
    │   "primaryBO": "Customer",               │
    │   "existingFieldIds": ["f1", "f2", "f3"]│
    │ }                                        │
    └───────────┬────────────────────────────┘
                │
                ▼
    ┌──────────────────────────────────────────┐
    │ API Gateway → AI Service                 │
    │ Query usage scores                       │
    │ Filter existing fields                   │
    │ Sort by score DESC                       │
    └───────────┬────────────────────────────┘
                │
                │ Response: {
                │   "recommendations": [
                │     {"fieldId": "f8", "usageScore": 0.77, ...},
                │     {"fieldId": "f9", "usageScore": 0.66, ...},
                │     {"fieldId": "f10", "usageScore": 0.59, ...}
                │   ]
                │ }
                │
                ▼
    ┌──────────────────────────────────────────┐
    │ Frontend renders suggestions list        │
    │ ☐ Field F8 (77%) - High engagement      │
    │ ☐ Field F9 (66%) - Common pattern       │
    │ ☐ Field F10 (59%) - Related data        │
    │ [Add Fields] (disabled until selected)   │
    └───────────┬────────────────────────────┘
                │
                │ User selects f8 + f9
                │ Clicks [Add Fields]
                ▼
    ┌──────────────────────────────────────────┐
    │ Frontend updates section:                │
    │ fieldIds = ["f1", "f2", "f3", "f8", "f9"]│
    │ Closes panel                             │
    │ Rerender section with new fields         │
    └──────────────────────────────────────────┘
```

### 3. Publish & Governance Flow

```
┌────────────────────────────────────────┐
│ User reviews final layout               │
│ Clicks: [🚀 Publish]                   │
└───────────┬────────────────────────────┘
            │
            │ EditorHeader.onPublish triggered
            ▼
┌────────────────────────────────────────┐
│ Pre-publication Validation              │
│ POST /api/publish/validate              │
│ Body: {                                 │
│   "accessibilityOk": true,              │
│   "performanceOk": true                 │
│ }                                       │
└───────────┬────────────────────────────┘
            │
            ▼
┌────────────────────────────────────────┐
│ API Gateway (local handler)             │
│ Check 1: accessibilityOk = true ✓      │
│ Check 2: performanceOk = true ✓        │
│ Result: allowed = true                  │
│ Reasons: []                             │
└───────────┬────────────────────────────┘
            │
            │ If allowed:
            ▼
┌────────────────────────────────────────┐
│ Frontend: Show confirmation dialog      │
│ "Confirm & Publish Layout?"             │
│ [Cancel] [Confirm & Publish]            │
└───────────┬────────────────────────────┘
            │
            │ User clicks [Confirm]
            ▼
┌────────────────────────────────────────┐
│ 1. Save layout to DB (custom handler)   │
│    POST /api/custom-components/save     │
│                                         │
│ 2. Mark AI draft as adopted             │
│    POST /api/ai/mark-adopted            │
│    Body: {                              │
│      "draftId": "uuid",                 │
│      "userId": "current-user-id"        │
│    }                                    │
│                                         │
│ 3. AI Service updates:                  │
│    UPDATE ai_layouts SET                │
│      adopted = true,                    │
│      adopted_at = NOW(),                │
│      adopted_by = userId                │
│    WHERE id = draftId                   │
│                                         │
│ 4. ✅ Success!                          │
│    Layout is now LIVE                   │
│    Audit trail preserved                │
└────────────────────────────────────────┘
```

---

## 🔄 Data Flow: Tenant Scoping

```
USER LAYER (Multiple Tenants)
─────────────────────────────────────────

Tenant A              Tenant B              Tenant C
├─ User 1            ├─ User 3            ├─ User 5
├─ User 2            └─ User 4            └─ User 6
└─ Products          └─ Products          └─ Products


REQUEST LAYER (Headers Enforce Scope)
─────────────────────────────────────────

POST /api/ai/generate-layout
Headers:
  X-Tenant-ID: {Tenant_A_ID}
  X-Tenant-Datasource-ID: {Datasource_ID}
  Authorization: Bearer {token}

↓ Middleware validation:
  ├─ Check X-Tenant-ID present
  ├─ Check format valid (UUID)
  └─ Pass through to handler


DATABASE LAYER (Queries Filtered)
─────────────────────────────────────────

Handler receives tenant_id from header:

SELECT * FROM ai_layouts
  WHERE tenant_id = {Tenant_A_ID}  ← ENFORCED
    AND adopted = false
  ORDER BY created_at DESC;

Result: Only Tenant A's drafts returned

INSERT INTO ai_layouts (...)
  VALUES (tenant_id = {Tenant_A_ID}, ...)
    ← tenant_id auto-filled from header

Query results:
  Tenant A sees: [Draft A1, Draft A2, ...]
  Tenant B sees: [Draft B1, Draft B2, ...]
  Tenant C sees: [Draft C1, Draft C2, ...]

Response layer:
  Returns JSON with tenant_id in response
  Frontend validates matches X-Tenant-ID header
```

---

## 🎯 Component Integration Points

```
Layout Builder Application
├─ EditorHeader (NEW)
│  ├─ Imports: AiActions, publish validation
│  ├─ Props: primaryBO, tenantId, userId, onApplyLayout, onPublish
│  ├─ State: showPublishConfirm, publishErrors
│  └─ Handlers: handlePublishClick(), confirmPublish()
│
├─ Main Layout Canvas
│  └─ Sections Loop:
│     ├─ If type === 'fields':
│     │  ├─ Field List (existing)
│     │  └─ FieldSuggestions (NEW)
│     │     ├─ Props: primaryBO, tenantId, existingFieldIds, onAddFields
│     │     ├─ State: recommendations, selected
│     │     └─ Handlers: fetchRecommendations(), handleAdd()
│     │
│     └─ If type === 'related_list':
│        └─ Related List UI (existing)
│
└─ Save/Publish Bar
   ├─ [💾 Save] → onSave handler
   ├─ [🚀 Publish] → EditorHeader.onPublish
   └─ Analytics: POST /api/analytics/layout on each action
```

---

## 📈 Error Handling & Recovery

```
┌─ Tenant Validation ──────────────────────┐
│ Missing X-Tenant-ID                      │
│ → 400 Bad Request                        │
│ "missing X-Tenant-ID header"             │
│ → User must select tenant in dropdown    │
└──────────────────────────────────────────┘

┌─ AI Service Unavailable ─────────────────┐
│ Can't connect to 8088                    │
│ → 502 Bad Gateway (from proxy)           │
│ → Frontend shows error message           │
│ → Suggests checking service status       │
└──────────────────────────────────────────┘

┌─ Database Error ─────────────────────────┐
│ INSERT fails (FK constraint)             │
│ → 500 Internal Server Error              │
│ → Log error with tenant_id for debugging │
│ → Frontend: "Failed to save layout"      │
└──────────────────────────────────────────┘

┌─ Governance Check Fails ─────────────────┐
│ accessibilityOk = false                  │
│ → 412 Precondition Failed                │
│ → Returns: {                             │
│    "allowed": false,                     │
│    "reasons": [                          │
│      "Accessibility compliance failed"   │
│    ]                                     │
│  }                                       │
│ → Frontend displays blockers             │
└──────────────────────────────────────────┘
```

---

## ✅ Deployment Architecture

```
Development                    Staging                      Production
─────────────────────────────────────────────────────────────────────

localhost:8080             docker-compose              Kubernetes/Cloud
├─ API Gateway             ├─ api-service              ├─ api-service (replicas)
├─ AI Service (8088)       ├─ ai-service (8088)        ├─ ai-service (replicas)
├─ PostgreSQL              ├─ PostgreSQL (managed)     ├─ PostgreSQL (managed)
└─ Frontend (npm dev)      └─ Frontend (nginx)         └─ Frontend (CDN + nginx)

Config:
DATABASE_URL=              DATABASE_URL=               DATABASE_URL=
postgresql://postgres      postgres://prod-user        ${RDS_ENDPOINT}
:postgres@localhost:       :${DB_PASS}@
5432/alpha?...             postgres-staging-1.c9...

Networking:
localhost → localhost:8080 docker network             VPC / Service mesh
                          ai-service:8088             Load balancer
```

---

*Complete visual reference for AI Layout Builder architecture and data flows.*
