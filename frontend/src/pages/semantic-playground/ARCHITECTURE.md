# Semantic Playground - Architecture Overview

## 🏗️ System Architecture

### High-Level Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     SEMANTIC PLAYGROUND UI                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    PlaygroundPage.tsx                     │   │
│  │  Main orchestrator component managing entire workflow    │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                    │
│         ┌────────────────────┼────────────────────┐              │
│         │                    │                    │              │
│   ┌─────▼──────┐      ┌─────▼──────┐      ┌─────▼──────┐        │
│   │   Left:    │      │  Middle:   │      │   Right:   │        │
│   │  NL Input  │      │  Semantic  │      │  SQL View  │        │
│   │   Panel    │      │   Query    │      │  & Results │        │
│   │            │      │   Editor   │      │   Table    │        │
│   └────────────┘      └────────────┘      └────────────┘        │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
         │                    │                    │
         │                    │                    │
    ┌────▼──────────┐  ┌─────▼───────┐  ┌────────▼────────┐
    │  usePlanner   │  │ useExecutor  │  │  useSQLRunner   │
    │   Hook        │  │   Hook       │  │    Hook         │
    └────┬──────────┘  └─────┬───────┘  └────────┬────────┘
         │                    │                    │
         │                    │                    │
    ┌────▼──────────────────────────────────────────────┐
    │    semanticPlaygroundApi - Centralized Client     │
    │  Handles all HTTP requests + tenant ID injection  │
    └────┬──────────────────────────────────────────────┘
         │
         │
    ┌────▼──────────────────────────────────┐
    │  Backend REST API (8 endpoints)        │
    │                                        │
    │  /api/semantic/datasources       [GET]│
    │  /api/semantic/bundles/by-id     [GET]│
    │  /api/semantic/bundles/{ds}/ver  [GET]│
    │  /api/semantic/plan              [POST]
    │  /api/semantic/execute           [POST]
    │  /api/sql/run                    [POST]
    │  /api/semantic/lineage/{id}      [GET]│
    │  /api/semantic/explain           [POST]
    │                                        │
    └─────────────────────────────────────────┘
         │
         │
    ┌────▼──────────────────────────────────┐
    │  LLM Processing                        │
    │                                        │
    │  ┌──────────────────────────────────┐ │
    │  │  Planner LLM (Gemini)            │ │
    │  │  Input:  Natural Language        │ │
    │  │  Output: SemanticQuery JSON      │ │
    │  └──────────────────────────────────┘ │
    │                                        │
    │  ┌──────────────────────────────────┐ │
    │  │  Executor LLM (Gemini)           │ │
    │  │  Input:  SemanticQuery JSON      │ │
    │  │  Output: SQL String              │ │
    │  └──────────────────────────────────┘ │
    │                                        │
    └─────────────────────────────────────────┘
         │
         │
    ┌────▼──────────────────────────────────┐
    │  Database                              │
    │  Execute final SQL + return rows       │
    │  Result: Record<string, any>[] format  │
    └────────────────────────────────────────┘
```

## 📦 Component Dependencies

### Dependency Tree

```
PlaygroundPage.tsx (Main Container)
├── Imports: usePlanner, useExecutor, useSQLRunner, useSemanticBundle
├── Imports: NLInputPanel, SemanticQueryEditor, SQLViewer, ResultsTable
├── Imports: MUI components (Box, Grid, AppBar, Card, Button, etc.)
└── Imports: API client (semanticPlaygroundApi)
    │
    ├── NLInputPanel.tsx
    │   ├── Imports: MUI (FormControl, Select, TextField, Button)
    │   ├── Imports: Material-UI Icons
    │   └── Uses props: datasources, selectedDatasource, versions, etc.
    │
    ├── SemanticQueryEditor.tsx
    │   ├── Imports: ReactJson (or simple pre/textarea)
    │   ├── Imports: MUI components
    │   ├── Imports: jsPonify for formatting
    │   └── Props: query, bundle, warnings, onQueryChange, onExplain
    │
    ├── SQLViewer.tsx
    │   ├── Imports: MUI components
    │   ├── Imports: sql-format library
    │   └── Props: sql, loading, error, onExecute, onDownloadCSV
    │
    └── ResultsTable.tsx
        ├── Imports: MUI Table, TablePagination
        ├── Imports: MUI components
        └── Props: results, executionTime, loading, error

Hooks Layer
├── usePlanner.ts
│   ├── Uses: useState, useCallback
│   ├── Uses: semanticPlaygroundApi.callPlanner
│   └── Returns: { semanticQuery, explanation, confidence, warnings, loading, error, callPlanner }
│
├── useExecutor.ts
│   ├── Uses: useState, useCallback
│   ├── Uses: semanticPlaygroundApi.callExecutor
│   └── Returns: { generatedSQL, warnings, loading, error, callExecutor }
│
├── useSQLRunner.ts
│   ├── Uses: useState, useCallback
│   ├── Uses: semanticPlaygroundApi.runSQL
│   └── Returns: { results, executionTime, loading, error, runSQL }
│
└── useSemanticBundle.ts
    ├── Uses: useState, useCallback, useRef (for caching)
    ├── Uses: semanticPlaygroundApi.getBundle, getBundleVersions
    └── Returns: { bundle, versions, loading, error, fetchBundle, fetchVersions }

API Client (utils/api.ts)
├── Base: axios instance with VITE_API_URL
├── Tenant ID: Auto-injected from localStorage
├── Methods:
│   ├── getDatasources()
│   ├── getBundle(datasource, version?)
│   ├── getBundleVersions(datasource)
│   ├── callPlanner(request)
│   ├── callExecutor(request)
│   ├── runSQL(request)
│   ├── getFieldLineage(fieldId)
│   ├── diffBundles(datasource, from, to)
│   └── explainQuery(datasource, query)
└── Error Handling: ApiError interface

Type System (types.ts)
├── SemanticBundle
├── SemanticField
├── SemanticQuery
├── FilterCondition
├── PhysicalMapping
├── Datasource
├── BundleVersion
├── PlannerRequest/Response
├── ExecutorRequest/Response
├── QueryExecutionRequest/Response
├── LineageNode
└── PlaygroundState

Utils (utils/jsonSchema.ts)
├── JSON Schema Draft-07 for SemanticQuery
├── Monaco Editor Options
└── SQL Editor Options
```

## 🔄 State Management Flow

### PlaygroundPage State

```typescript
// UI Selection State
const [datasources, setDatasources] = useState<Datasource[]>([]);
const [selectedDatasource, setSelectedDatasource] = useState<string | null>(null);
const [selectedVersion, setSelectedVersion] = useState<string | null>(null);
const [nlPrompt, setNlPrompt] = useState('');
const [mode, setMode] = useState<'exploratory' | 'strict' | 'CRUD'>('exploratory');

// Query Pipeline State (from hooks)
const { semanticQuery, callPlanner, ...plannerState } = usePlanner();
const { generatedSQL, callExecutor, ...executorState } = useExecutor();
const { results, executionTime, runSQL, ...runnerState } = useSQLRunner();

// Bundle Metadata
const { bundle, versions, fetchBundle, fetchVersions } = useSemanticBundle();

// Notification State
const [snackbar, setSnackbar] = useState({
  open: false,
  message: '',
  severity: 'success' as 'success' | 'error' | 'warning' | 'info'
});

// Effects
useEffect(() => {
  // Load datasources on mount
  semanticPlaygroundApi.getDatasources().then(setDatasources);
}, []);

useEffect(() => {
  // Load bundle when datasource changes
  if (selectedDatasource) {
    fetchBundle(selectedDatasource, selectedVersion);
    fetchVersions(selectedDatasource).then(versions => {
      if (versions.length > 0) {
        setSelectedVersion(versions[0]);  // Default to first version
      }
    });
  }
}, [selectedDatasource]);

// Event Handlers
const handleGenerateQuery = async () => {
  try {
    await callPlanner({
      datasource: selectedDatasource!,
      version: selectedVersion!,
      prompt: nlPrompt,
      mode
    });
    showSnackbar('Query generated! Check the middle pane.', 'success');
  } catch (err) {
    showSnackbar('Failed to generate query', 'error');
  }
};

const handleExecuteQuery = async () => {
  try {
    await callExecutor({
      datasource: selectedDatasource!,
      version: selectedVersion!,
      semantic_query: semanticQuery!
    });
    showSnackbar('SQL generated! Click Run to execute.', 'success');
  } catch (err) {
    showSnackbar('Failed to execute query', 'error');
  }
};

const handleRunSQL = async () => {
  try {
    await runSQL(generatedSQL);
    showSnackbar(`Results loaded: ${results?.row_count} rows`, 'success');
  } catch (err) {
    showSnackbar('Failed to run SQL', 'error');
  }
};
```

## 🌊 Component Lifecycle

### User Interaction Sequence

```
1. Page Loads
   ├── PlaygroundPage rendered
   ├── useEffect([], []) fires
   ├── fetchDatasources()
   ├── Display datasource dropdown
   └── "Waiting for datasource selection..."

2. User Selects Datasource
   ├── setSelectedDatasource(datasource)
   ├── useEffect([selectedDatasource])
   ├── fetchBundle(datasource)
   ├── fetchVersions(datasource)
   ├── setSelectedVersion(versions[0])
   ├── NLInputPanel becomes enabled
   └── "Ready for input..."

3. User Enters Prompt
   ├── setNlPrompt(text)
   ├── onModeChange(mode)
   └── Keyboard shortcut Ctrl+Enter available

4. User Clicks "Generate Query"
   ├── handleGenerateQuery()
   ├── Loading spinner appears
   ├── callPlanner({...})
   ├── API POST /api/semantic/plan
   ├── Gemini processes NL prompt
   ├── Response: SemanticQuery JSON
   ├── setSemanticQuery()
   ├── SemanticQueryEditor displays query
   ├── Show snackbar success
   └── "Query ready! Review or edit..."

5. User Reviews/Edits Semantic Query (Optional)
   ├── SemanticQueryEditor shows JSON
   ├── User can click "Edit JSON"
   ├── Manual editing mode enabled
   ├── User modifies JSON
   ├── Click "Apply"
   ├── JSON validated
   ├── setSemanticQuery(updated)
   └── "Query updated"

6. User Clicks "Execute"
   ├── handleExecuteQuery()
   ├── Loading spinner appears
   ├── callExecutor({datasource, version, query})
   ├── API POST /api/semantic/execute
   ├── Gemini processes SemanticQuery
   ├── Response: generated SQL
   ├── setGeneratedSQL()
   ├── SQLViewer displays SQL
   ├── Show snackbar success
   └── "SQL ready! Click Run to execute..."

7. User Clicks "Run" (or "Execute" button in SQLViewer)
   ├── handleRunSQL()
   ├── Loading spinner appears
   ├── runSQL(generatedSQL)
   ├── API POST /api/sql/run
   ├── Backend executes SQL against database
   ├── Response: rows + row_count + execution_time_ms
   ├── setResults()
   ├── ResultsTable displays data
   ├── Show snackbar success with row count
   └── "Results loaded! Sort, filter, or export..."

8. User Interacts with Results
   ├── Click column header → Sort (asc/desc)
   ├── Type in search box → Filter rows
   ├── Change rows per page → Pagination updates
   ├── Click "Export CSV" → Download file
   ├── Copy rows to clipboard → Done
   └── "Happy querying!"

9. User Generates New Query
   ├── Go back to step 3 or 4
   ├── Previous results cleared
   ├── New query pipeline starts
   └── Cycle repeats
```

## 📡 API Contract

### Request/Response Contracts

**Endpoint: GET /api/semantic/datasources**

Request:
```
GET /api/semantic/datasources
Headers: X-Tenant-ID: {tenant_id}
```

Response (200 OK):
```json
[
  {
    "id": "customers",
    "name": "Customers",
    "description": "Customer data",
    "table_count": 5,
    "field_count": 42
  },
  {
    "id": "orders",
    "name": "Orders",
    "description": "Order data",
    "table_count": 3,
    "field_count": 28
  }
]
```

**Endpoint: GET /api/semantic/bundles/by-id**

Request:
```
GET /api/semantic/bundles/by-id?datasource=customers&version=v1
Headers: X-Tenant-ID: {tenant_id}
```

Response (200 OK):
```json
{
  "datasource": "customers",
  "version": "v1",
  "description": "Customer semantic layer",
  "fields": [
    {
      "name": "customer_id",
      "type": "string",
      "description": "Unique customer identifier",
      "is_measure": false,
      "is_dimension": true,
      "physical_mapping": {
        "table": "raw_customers",
        "column": "cust_id"
      }
    },
    {
      "name": "loyalty_points",
      "type": "number",
      "description": "Loyalty program points",
      "is_measure": true,
      "is_dimension": false,
      "subtypes": []
    }
  ],
  "relationships": [],
  "physical_tables": []
}
```

**Endpoint: POST /api/semantic/plan**

Request:
```
POST /api/semantic/plan
Content-Type: application/json
X-Tenant-ID: {tenant_id}

{
  "datasource": "customers",
  "version": "v1",
  "prompt": "Show loyalty points for retail customers",
  "mode": "exploratory"
}
```

Response (200 OK):
```json
{
  "semantic_query": {
    "datasource": "customers",
    "version": "v1",
    "select": ["customer_id", "loyalty_points"],
    "filters": [
      {
        "field": "customer_type",
        "op": "=",
        "value": "RETAIL"
      }
    ],
    "order_by": [{"field": "loyalty_points", "direction": "desc"}],
    "limit": 100
  },
  "explanation": "Selected loyalty points for retail customers, ordered by highest points first",
  "confidence": 0.87,
  "warnings": []
}
```

**Endpoint: POST /api/semantic/execute**

Request:
```
POST /api/semantic/execute
Content-Type: application/json
X-Tenant-ID: {tenant_id}

{
  "datasource": "customers",
  "version": "v1",
  "semantic_query": {
    "select": ["customer_id", "loyalty_points"],
    "filters": [{"field": "customer_type", "op": "=", "value": "RETAIL"}],
    "limit": 100
  }
}
```

Response (200 OK):
```json
{
  "generated_sql": "SELECT c.cust_id AS customer_id, c.loyalty_points FROM raw_customers c WHERE c.customer_type = 'RETAIL' ORDER BY c.loyalty_points DESC LIMIT 100",
  "semantic_sql": { /* echoed request */ },
  "execution_plan": "...",
  "warnings": []
}
```

**Endpoint: POST /api/sql/run**

Request:
```
POST /api/sql/run
Content-Type: application/json
X-Tenant-ID: {tenant_id}

{
  "sql": "SELECT ... FROM ... LIMIT 100",
  "limit": 1000,
  "timeout": 30000
}
```

Response (200 OK):
```json
{
  "rows": [
    {"customer_id": "C001", "loyalty_points": 5000},
    {"customer_id": "C002", "loyalty_points": 4800}
  ],
  "row_count": 2,
  "columns": ["customer_id", "loyalty_points"],
  "execution_time_ms": 125
}
```

## 🔐 Security Model

### Tenant Isolation

```
Request Flow:
┌─────────────────────────────────────────┐
│  Frontend: User selects datasource      │
│  (e.g., "customers")                    │
└──────────────┬──────────────────────────┘
               │
               ├─> Check localStorage for tenant_id
               │
               ├─> Add to request header:
               │   X-Tenant-ID: user-tenant-123
               │
└──────────────┬──────────────────────────┐
               │                          │
          ┌────▼────┐              ┌─────▼─────┐
          │ Backend  │              │ Database  │
          │ Validates│◄─────────────┤ Auth      │
          │ Tenant ID│              │ Layer     │
          └────┬─────┘              └───────────┘
               │
         ┌─────▼──────────────────┐
         │ Verify:                │
         │ 1. User owns tenant    │
         │ 2. Datasource belongs  │
         │    to tenant           │
         │ 3. Return only user's  │
         │    data                │
         └────────────────────────┘
```

### Authentication Flow

```
1. User logs in → JWT token created
2. Token stored in localStorage AND secure cookie
3. Every API request includes tenant_id from localStorage
4. Backend middleware validates:
   - X-Tenant-ID header present
   - User has access to tenant
   - User has access to datasource
5. Only return data for that tenant
```

### CORS & Headers

```
Frontend Request:
┌────────────────────────────────────┐
│ POST /api/semantic/plan            │
│ Headers:                           │
│  - Content-Type: application/json  │
│  - X-Tenant-ID: user-tenant-123   │
│  - Authorization: Bearer {token}   │
└────────────────────────────────────┘
         │
         ▼
Backend Response:
┌────────────────────────────────────┐
│ Status: 200 OK                     │
│ Headers:                           │
│  - Access-Control-Allow-Origin:    │
│    https://yourdomain.com          │
│  - Access-Control-Allow-Methods:   │
│    GET, POST, OPTIONS              │
│  - Content-Type: application/json  │
│  - X-Content-Type-Options: nosniff │
└────────────────────────────────────┘
```

## 🚀 Performance Considerations

### Caching Strategy

```
Level 1: Browser LocalStorage
├── Datasource list (refresh on app start)
├── Bundle versions (refresh per datasource)
└── Tenant ID (persistent)

Level 2: Component State (React)
├── semanticQuery (per plan)
├── generatedSQL (per execute)
├── results (per run)
└── Current selections

Level 3: API Response Caching (Optional)
├── Cache-Control headers
├── ETag validation
└── 304 Not Modified responses

Level 4: Backend Caching
├── Redis for bundle metadata
├── Query plan cache (LLM results)
└── Execution plan cache
```

### Query Optimization

```
Planner LLM:
- Inference: 2-5 seconds
- Cached per (datasource, version, bundle)
- Configurable cache TTL

Executor LLM:
- Inference: 2-5 seconds
- Deterministic (same query → same SQL)
- Cacheable by query hash

SQL Execution:
- Variable based on complexity
- Timeout: 30 seconds
- Result pagination: 1000 rows default
```

### Memory Management

```
Results Table:
- Virtualization for large datasets (>10K rows)
- Pagination reduces in-memory footprint
- Rows per page: 10-100 (configurable)
- Total memory: ~1MB per 10K rows

Component Unmounting:
- Clear pending API requests
- Cancel Gemini API calls
- Release hook state on component unmount
```

## 📊 Data Structures

### Complete Type Hierarchy

```
SemanticBundle
├── datasource: string
├── version: string
├── description: string
├── fields: SemanticField[]
│   ├── name: string
│   ├── type: string (enum)
│   ├── description: string
│   ├── is_measure: boolean
│   ├── is_dimension: boolean
│   ├── physical_mapping: PhysicalMapping
│   │   ├── table: string
│   │   └── column: string
│   ├── subtypes: string[]
│   └── aliases: string[]
├── relationships: Relationship[]
│   ├── name: string
│   ├── from_field: string
│   ├── to_table: string
│   └── to_field: string
└── physical_tables: PhysicalTable[]
    ├── name: string
    ├── description: string
    └── columns: string[]

SemanticQuery
├── datasource: string
├── version: string
├── select: string[] (field names)
├── filters: FilterCondition[]
│   ├── field: string
│   ├── op: FilterOperator (enum)
│   └── value: any
├── order_by: OrderByClause[]
│   ├── field: string
│   └── direction: "asc" | "desc"
├── group_by: string[]
└── limit: number

QueryExecutionResponse
├── rows: Record<string, any>[]
├── row_count: number
├── columns: string[]
└── execution_time_ms: number
```

---

This architecture enables a clean separation of concerns while maintaining type safety and performance. Each layer can be developed and tested independently.
