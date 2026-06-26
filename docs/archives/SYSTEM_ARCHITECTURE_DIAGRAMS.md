# 🏗️ SYSTEM ARCHITECTURE DIAGRAM

## Complete System Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DEVELOPMENT ENVIRONMENT                          │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │                   BROWSER                                       │   │
│  │              http://localhost:5173                             │   │
│  │                                                                │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │          FRONTEND (React + Vite)                       │ │   │
│  │  │                                                         │ │   │
│  │  │  1. User Action                                       │ │   │
│  │  │     ↓                                                 │ │   │
│  │  │  2. fetch('/api/entity-schema')                      │ │   │
│  │  │     ↓                                                 │ │   │
│  │  │  3. setupTenantFetch intercepts                      │ │   │
│  │  │     (window.fetch patched)                           │ │   │
│  │  │     ↓                                                 │ │   │
│  │  │  4. Reads VITE_API_BASE_URL = http://localhost:8080 │ │   │
│  │  │     ↓                                                 │ │   │
│  │  │  5. Rebases URL:                                     │ │   │
│  │  │     /api/entity-schema →                             │ │   │
│  │  │     http://localhost:8080/api/entity-schema         │ │   │
│  │  │     ↓                                                 │ │   │
│  │  │  6. Adds headers:                                    │ │   │
│  │  │     X-Tenant-ID: 910638ba-...                       │ │   │
│  │  │     X-Tenant-Datasource-ID: 982aef38-...           │ │   │
│  │  │     ↓                                                 │ │   │
│  │  │  7. originalFetch() sends request                    │ │   │
│  │  └─────────────────────────────────────────────────────┘ │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                             │                                          │
│                             │                                          │
│                             ↓                                          │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │                 NETWORK REQUEST                               │   │
│  │                                                                │   │
│  │  GET http://localhost:8080/api/entity-schema?                │   │
│  │    tenant_id=910638ba-... &datasource_id=982aef38-...        │   │
│  │                                                                │   │
│  │  Headers:                                                     │   │
│  │  X-Tenant-ID: 910638ba-...                                   │   │
│  │  X-Tenant-Datasource-ID: 982aef38-...                        │   │
│  │                                                                │   │
│  └────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                             │
                             │
        ┌────────────────────┴────────────────────┐
        │                                          │
        ↓                                          ↓
┌──────────────────────┐               ┌──────────────────────┐
│   BACKEND DOCKER     │               │   BACKEND DOCKER     │
│   (docker-compose)   │               │   (docker-compose)   │
│                      │               │                      │
│  Port 8080 (API)     │               │  Port 8888 (Hasura)  │
│  Backend Service     │               │  GraphQL Service     │
│  ✅ Receives POST/GET│               │  ✅ Receives GraphQL │
│  ✅ Processes        │               │  ✅ Executes query   │
│  ✅ Returns JSON 200 │               │  ✅ Returns results  │
│                      │               │                      │
└──────────────────────┘               └──────────────────────┘
        │                                          │
        ↓                                          ↓
   Response                                   Response
   (JSON data)                             (GraphQL results)
        │                                          │
        └────────────────────┬────────────────────┘
                             │
                             ↓
                   ┌──────────────────┐
                   │   BROWSER        │
                   │ http://localhost │
                   │    :5173         │
                   │                  │
                   │ ✅ Updates UI    │
                   │ ✅ Renders data  │
                   └──────────────────┘
```

---

## Port Allocation (`.env.ports`)

```
┌─────────────────────────────────────────────────────────────┐
│                    .env.ports                               │
│            (SINGLE SOURCE OF TRUTH)                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  BACKEND SERVICES (8000-8099)                              │
│  ├─ PORT_BACKEND_API=8080                                 │
│  ├─ PORT_FABRIC_BUILDER=8081                              │
│  └─ PORT_LEGACY_GATEWAY=8001                              │
│                                                              │
│  GRAPHQL & DATA (8200-8299)                                │
│  └─ PORT_HASURA_GRAPHQL=8888                              │
│                                                              │
│  MESSAGE QUEUE (5600-5700)                                 │
│  ├─ PORT_RABBITMQ_AMQP=5672                               │
│  └─ PORT_RABBITMQ_MANAGEMENT=15672                        │
│                                                              │
│  WORKFLOW (7200-7300)                                      │
│  ├─ PORT_TEMPORAL_SERVER=7233                             │
│  └─ PORT_TEMPORAL_UI=8088                                 │
│                                                              │
│  FRONTEND (5000-5200)                                      │
│  └─ PORT_VITE_DEV_SERVER=5173                             │
│                                                              │
│  DATABASE (5400-5500)                                      │
│  └─ PORT_POSTGRES_HOST=5432                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
         │                                    │
         ├────────────────┬───────────────────┤
         │                │                   │
         ↓                ↓                   ↓
    docker-compose    frontend/.env      scripts/
    (uses ${PORT_*})  (hardcoded 8080,   validate-ports.sh
                       8888)             (checks uniqueness)
```

---

## setupTenantFetch.ts URL Resolution

```
┌──────────────────────────────────────────────────────────────────┐
│                  appendScopeToUrl()                              │
│                   (In setupTenantFetch.ts)                       │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Input:  url = "/api/entity-schema"                             │
│          tenantId = "910638ba-..."                              │
│          datasourceId = "982aef38-..."                          │
│                                                                   │
│  Step 1: Read environment                                       │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ configuredBase = VITE_API_BASE_URL                    │    │
│  │              = "http://localhost:8080"               │    │
│  │                                                        │    │
│  │ if (!configuredBase)                                  │    │
│  │   configuredBase = VITE_BACKEND_TARGET               │    │
│  │                                                        │    │
│  │ if (!configuredBase)                                  │    │
│  │   configuredBase = "http://localhost:8080"           │    │
│  │   (PERMANENT FALLBACK)                               │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Step 2: Resolve URL                                            │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ final = new URL(url, configuredBase)                  │    │
│  │      = new URL("/api/entity-schema",                 │    │
│  │               "http://localhost:8080")              │    │
│  │      = "http://localhost:8080/api/entity-schema"     │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Step 3: Rebase if needed                                       │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ if (final.origin === frontendOrigin)  // 5173         │    │
│  │   // Rebase from frontend to backend origin            │    │
│  │   base = new URL(configuredBase)  // 8080             │    │
│  │   final = new URL(pathname+search, base.origin)       │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Step 4: Add scope parameters                                   │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ final.searchParams.set('tenant_id', tenantId)         │    │
│  │ final.searchParams.set('datasource_id', datasourceId) │    │
│  │                                                        │    │
│  │ Result:                                               │    │
│  │ http://localhost:8080/api/entity-schema?             │    │
│  │   tenant_id=910638ba-...&datasource_id=982aef38-...  │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Output: "http://localhost:8080/api/entity-schema?..."         │
│          with tenant headers added                              │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
         │
         ↓
    originalFetch(finalUrl, finalInit)
         │
         ↓
    Browser sends HTTP request to backend:8080 ✅
```

---

## Data Flow: REST API Call

```
┌──────────────────┐
│  User clicks     │
│  "Load Entities" │
└────────┬─────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  React Component                             │
│  fetch('/api/entity-schema')                 │
└────────┬─────────────────────────────────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  setupTenantFetch.ts                         │
│  (window.fetch patched)                      │
│                                              │
│  1. Intercepts fetch call                   │
│  2. Reads VITE_API_BASE_URL=http://...:8080│
│  3. Rebases URL to http://localhost:8080/..│
│  4. Adds X-Tenant-ID header                 │
│  5. Adds X-Tenant-Datasource-ID header      │
│  6. Calls originalFetch with tenant info    │
└────────┬─────────────────────────────────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  Network Request                             │
│  Host: localhost:8080                        │
│  Path: /api/entity-schema                    │
│  Method: GET                                 │
│  Headers:                                    │
│    X-Tenant-ID: 910638ba-...                │
│    X-Tenant-Datasource-ID: 982aef38-...     │
│  Query: tenant_id=...&datasource_id=...     │
└────────┬─────────────────────────────────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  Backend API (Docker, port 8080)             │
│  Go Chi server                               │
│                                              │
│  1. Receives request                         │
│  2. Validates tenant headers                 │
│  3. Queries database                         │
│  4. Returns JSON response                    │
│                                              │
│  Response Status: 200 OK                     │
│  Response Body: {...json data...}            │
└────────┬─────────────────────────────────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  Frontend (React)                            │
│  Receives response                           │
│  Updates state                               │
│  Re-renders UI with data ✅                  │
│                                              │
│  Status Code: 200 OK ✅                      │
│  Content-Type: application/json ✅           │
│  Data Displayed ✅                           │
└──────────────────────────────────────────────┘
```

---

## Data Flow: GraphQL Query

```
┌──────────────────┐
│  User component  │
│  useQuery(...)   │
└────────┬─────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  Apollo Client                               │
│  (apolloClient.tsx)                          │
│                                              │
│  1. Reads VITE_GRAPHQL_ENDPOINT             │
│  2. =" http://localhost:8888/v1/graphql"   │
│  3. Reads VITE_GRAPHQL_ADMIN_SECRET         │
│  4. = "newadminsecretkey"                   │
│  5. Creates HTTP POST request                │
└────────┬─────────────────────────────────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  Network Request                             │
│  Host: localhost:8888                        │
│  Path: /v1/graphql                           │
│  Method: POST                                │
│  Headers:                                    │
│    x-hasura-admin-secret: newadminsecretkey │
│    Content-Type: application/json            │
│  Body: {"query":"...", "variables":{...}}   │
└────────┬─────────────────────────────────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  Hasura GraphQL (Docker, port 8888)          │
│  GraphQL Engine v2.46.0                      │
│                                              │
│  1. Receives POST request                    │
│  2. Validates admin secret                   │
│  3. Parses GraphQL query                     │
│  4. Executes against database                │
│  5. Returns GraphQL response                 │
│                                              │
│  Response Status: 200 OK                     │
│  Response Body: {"data":{...}}               │
└────────┬─────────────────────────────────────┘
         │
         ↓
┌──────────────────────────────────────────────┐
│  Apollo Client (Frontend)                    │
│  Updates cache                               │
│  Notifies subscribers                        │
│  Re-renders component with new data ✅       │
│                                              │
│  Status Code: 200 OK ✅                      │
│  GraphQL Data Loaded ✅                      │
└──────────────────────────────────────────────┘
```

---

## Startup Sequence

```
Terminal 1: Start Backend Services
─────────────────────────────────────────────
$ docker compose --env-file .env.ports up -d

1. Docker Compose loads .env.ports
   ✓ PORT_BACKEND_API=8080
   ✓ PORT_HASURA_GRAPHQL=8888
   ✓ PORT_RABBITMQ_AMQP=5672
   ... (all ports loaded)

2. Starts containers:
   ✓ Backend        → 0.0.0.0:8080
   ✓ Hasura         → 0.0.0.0:8888
   ✓ RabbitMQ       → 0.0.0.0:5672, 15672
   ✓ Temporal       → 0.0.0.0:7233
   ✓ Temporal UI    → 0.0.0.0:8088

3. Services become available:
   ✓ http://localhost:8080/health → Backend ready
   ✓ http://localhost:8888/healthz → Hasura ready
   ✓ http://localhost:5672 → RabbitMQ ready


Terminal 2: Start Frontend
─────────────────────────────────────────────
$ cd frontend && npm run dev

1. Vite loads frontend/.env
   ✓ VITE_API_BASE_URL=http://localhost:8080
   ✓ VITE_GRAPHQL_ENDPOINT=http://localhost:8888/v1/graphql
   ✓ VITE_GRAPHQL_ADMIN_SECRET=newadminsecretkey

2. Webpack builds React app
   ✓ Substitutes environment variables
   ✓ Inlines Apollo client endpoint
   ✓ Sets up setupTenantFetch.ts

3. Vite dev server starts:
   ✓ http://localhost:5173 ready


Browser: Open Application
─────────────────────────────────────────────
$ open http://localhost:5173

1. Browser loads React app from Vite
2. App mounts and TenantProvider initializes
3. User selects tenant and datasource
4. localStorage is populated with scope
5. User clicks to load entities
6. fetch('/api/entity-schema') is called
7. setupTenantFetch intercepts and rebases URL
8. Request goes to http://localhost:8080 ✅
9. Backend returns JSON response
10. React renders data in UI ✅

EVERYTHING WORKS! 🎉
```

---

## Why This Architecture Is Permanent

```
┌─────────────────────────────────────────────────────┐
│           RESILIENCE THROUGH DESIGN                  │
├─────────────────────────────────────────────────────┤
│                                                      │
│  1. SINGLE SOURCE OF TRUTH                          │
│     .env.ports ← All ports defined here only        │
│     ↓                                               │
│     Everything reads from this one file             │
│                                                      │
│  2. AUTOMATIC VARIABLE SUBSTITUTION                 │
│     docker-compose: ${PORT_BACKEND_API} → 8080     │
│     frontend/.env: hardcoded 8080                   │
│     Both always in sync!                            │
│                                                      │
│  3. FALLBACK LOGIC                                  │
│     If VITE_API_BASE_URL missing                    │
│      → Try VITE_BACKEND_TARGET                      │
│      → Fall back to hardcoded 8080                  │
│     Always have a port!                             │
│                                                      │
│  4. URL REBASINGTING                                │
│     If URL somehow at frontend origin (5173)        │
│      → Automatically rebase to backend (8080)       │
│     Impossible to hit wrong server!                 │
│                                                      │
│  5. VALIDATION SCRIPT                               │
│     bash scripts/validate-ports.sh                  │
│     Checks for duplicates and configuration         │
│     Prevents port conflicts!                        │
│                                                      │
└─────────────────────────────────────────────────────┘
```

This architecture ensures your system is:
- **Permanent**: Never need manual changes
- **Automatic**: Variable substitution handles everything
- **Resilient**: Multiple fallbacks for safety
- **Validated**: Script checks for errors
- **Documented**: Clear purpose for each component
