# Semantic Playground - Quick Reference

## 🎯 Most Common Tasks

### Task 1: Import & Use in a Page

```typescript
import { SemanticPlaygroundPage } from '@/pages/semantic-playground';

// In your route:
<Route path="/playground" element={<SemanticPlaygroundPage />} />
```

### Task 2: Call Planner Hook Directly (Advanced)

If you want to use the planner outside the playground:

```typescript
import { usePlanner } from '@/pages/semantic-playground/hooks';

function MyComponent() {
  const { callPlanner, semanticQuery, loading } = usePlanner();
  
  const handleGenerate = async () => {
    const query = await callPlanner({
      datasource: 'customers',
      version: 'v1',
      prompt: 'Recent US customers',
      mode: 'exploratory'
    });
  };
  
  return (
    <div>
      <button onClick={handleGenerate}>Generate</button>
      {loading && <p>Loading...</p>}
      {semanticQuery && <pre>{JSON.stringify(semanticQuery)}</pre>}
    </div>
  );
}
```

### Task 3: Call Executor Hook Directly (Advanced)

```typescript
import { useExecutor } from '@/pages/semantic-playground/hooks';

function MyComponent() {
  const { callExecutor, generatedSQL, loading } = useExecutor();
  
  const handleExecute = async () => {
    const result = await callExecutor({
      datasource: 'customers',
      version: 'v1',
      semantic_query: { /* ... */ }
    });
  };
  
  return (
    <div>
      <button onClick={handleExecute}>Execute</button>
      {loading && <p>Loading...</p>}
      {generatedSQL && <pre>{generatedSQL}</pre>}
    </div>
  );
}
```

### Task 4: Run SQL Directly (Advanced)

```typescript
import { useSQLRunner } from '@/pages/semantic-playground/hooks';

function MyComponent() {
  const { runSQL, results, loading } = useSQLRunner();
  
  const handleRun = async () => {
    await runSQL('SELECT * FROM customers LIMIT 20');
  };
  
  return (
    <div>
      <button onClick={handleRun}>Run</button>
      {loading && <p>Executing...</p>}
      {results && (
        <div>
          <p>Rows: {results.row_count}</p>
          <p>Time: {results.execution_time_ms}ms</p>
        </div>
      )}
    </div>
  );
}
```

### Task 5: Use API Client Directly

```typescript
import { semanticPlaygroundApi } from '@/pages/semantic-playground/utils/api';

// Get datasources
const datasources = await semanticPlaygroundApi.getDatasources();

// Get bundle
const bundle = await semanticPlaygroundApi.getBundle('customers', 'v1');

// Get versions
const versions = await semanticPlaygroundApi.getBundleVersions('customers');

// Call planner
const planResult = await semanticPlaygroundApi.callPlanner({
  datasource: 'customers',
  version: 'v1',
  prompt: 'Show me...',
  mode: 'exploratory'
});

// Call executor
const execResult = await semanticPlaygroundApi.callExecutor({
  datasource: 'customers',
  version: 'v1',
  semantic_query: { /* ... */ }
});

// Run SQL
const sqlResult = await semanticPlaygroundApi.runSQL({
  sql: 'SELECT * FROM customers',
  limit: 1000,
  timeout: 30000
});
```

### Task 6: Customize Components

**Change colors:**

```typescript
// In components, modify sx prop:
sx={{
  backgroundColor: '#1e1e1e',  // Your color
  color: '#fff',
}}
```

**Change button text:**

```typescript
// In NLInputPanel, find and update:
<Button>Generate Query</Button> → <Button>Search</Button>
```

**Add custom styling:**

```typescript
// Create theme or CSS module
import styles from './custom.module.css';

// In component:
sx={{
  ...styles.customClass,
  backgroundColor: '#1e1e1e'
}}
```

### Task 7: Add Custom Analytics

```typescript
// In PlaygroundPage.tsx, add after each action:
const handleGenerateQuery = async () => {
  // Analytics
  if (window.gtag) {
    gtag('event', 'semantic_query_generated', {
      datasource: selectedDatasource,
      mode: mode
    });
  }
  
  // Original logic...
  await callPlanner(/*...*/);
};
```

### Task 8: Test Endpoints

**Test datasources endpoint:**

```bash
curl http://localhost:8080/api/semantic/datasources \
  -H "X-Tenant-ID: test-tenant"
```

**Test planner endpoint:**

```bash
curl -X POST http://localhost:8080/api/semantic/plan \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -d '{
    "datasource": "customers",
    "version": "v1",
    "prompt": "Show me customers",
    "mode": "exploratory"
  }'
```

**Test executor endpoint:**

```bash
curl -X POST http://localhost:8080/api/semantic/execute \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -d '{
    "datasource": "customers",
    "version": "v1",
    "semantic_query": {
      "select": ["customer_id", "name"],
      "limit": 20
    }
  }'
```

**Test SQL endpoint:**

```bash
curl -X POST http://localhost:8080/api/sql/run \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -d '{
    "sql": "SELECT * FROM customers LIMIT 20",
    "limit": 1000,
    "timeout": 30000
  }'
```

## 📚 Type Reference

### SemanticBundle

```typescript
interface SemanticBundle {
  datasource: string;           // e.g., "customers"
  version: string;              // e.g., "v1"
  description: string;          // Human-readable description
  fields: SemanticField[];       // Available fields
  relationships: {
    name: string;
    from_field: string;
    to_table: string;
    to_field: string;
  }[];
  physical_tables: {
    name: string;
    description: string;
    columns: string[];
  }[];
}
```

### SemanticField

```typescript
interface SemanticField {
  name: string;                 // e.g., "customer_id"
  type: string;                 // "string" | "number" | "date" | etc
  description: string;
  physical_mapping: {
    table: string;
    column: string;
  };
  is_measure: boolean;          // True if aggregate
  is_dimension: boolean;        // True if grouping
  subtypes?: string[];          // e.g., ["EMAIL", "PHONE"]
}
```

### SemanticQuery

```typescript
interface SemanticQuery {
  datasource: string;
  version: string;
  select: string[];             // Field names to select
  filters?: FilterCondition[];   // WHERE conditions
  order_by?: {
    field: string;
    direction: "asc" | "desc";
  }[];
  group_by?: string[];           // GROUP BY fields
  limit?: number;                // LIMIT
}
```

### FilterCondition

```typescript
interface FilterCondition {
  field: string;                // Field name
  op: "=" | "<" | ">" | "<=" | ">=" | "IN" | "LIKE";
  value: any;                   // Filter value
}
```

### API Responses

**Planner Response:**
```typescript
{
  semantic_query: SemanticQuery;
  explanation: string;
  confidence: number;  // 0-1
  warnings: string[];
}
```

**Executor Response:**
```typescript
{
  generated_sql: string;
  semantic_sql: SemanticQuery;
  execution_plan?: string;
  warnings: string[];
}
```

**SQL Execution Response:**
```typescript
{
  rows: Record<string, any>[];
  row_count: number;
  columns: string[];
  execution_time_ms: number;
}
```

## 🔧 Common Customizations

### Change API Base URL

In `utils/api.ts`, update:

```typescript
const baseURL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';
```

### Change Theme Colors

In any component `sx` prop:

```typescript
// Dark background
backgroundColor: '#0d1117'  // Github dark

// Light background
backgroundColor: '#f6f8fa'  // Github light

// Accent blue
color: '#2196F3'  // Material-UI blue

// Success green
color: '#4CAF50'  // Material-UI green
```

### Change Panel Sizes

In `PlaygroundPage.tsx`, modify grid props:

```typescript
// Current: 4 columns each
<Grid item xs={12} md={4}>  {/* Left pane */}
<Grid item xs={12} md={4}>  {/* Middle pane */}
<Grid item xs={12} md={4}>  {/* Right pane */}

// To make middle pane larger (3-5-4):
<Grid item xs={12} md={3}>  {/* Left pane */}
<Grid item xs={12} md={5}>  {/* Middle pane - bigger */}
<Grid item xs={12} md={4}>  {/* Right pane */}
```

### Change Results per Page

In `ResultsTable.tsx`, modify:

```typescript
// Current options: [10, 20, 50, 100]
const rowsPerPageOptions = [10, 20, 50, 100];

// Change to:
const rowsPerPageOptions = [25, 50, 100, 250];
```

### Change SQL Formatting

In `SQLViewer.tsx`, modify formatSQL function:

```typescript
const formatSQL = (sql: string) => {
  return sql
    .replace(/^SELECT/gi, '\nSELECT')
    .replace(/^FROM/gim, '\nFROM')
    .replace(/^WHERE/gim, '\nWHERE')
    .replace(/^ORDER BY/gim, '\nORDER BY')
    // Add more patterns
    .trim();
};
```

## 🐛 Quick Debug Tips

### Enable Console Logging

In `api.ts`, add at top of each method:

```typescript
export const getDatasources = async () => {
  const url = `${baseURL}/semantic/datasources`;
  console.log('[API] GET', url);
  
  try {
    const response = await axios.get<Datasource[]>(url, { headers });
    console.log('[API] Response:', response.data);
    return response.data;
  } catch (error) {
    console.error('[API] Error:', error);
    throw error;
  }
};
```

### Check Browser Network Tab

1. Open DevTools (F12)
2. Go to Network tab
3. Perform an action
4. Check API calls:
   - Status code should be 200
   - Response should match expected format
   - Headers should include X-Tenant-ID

### Check Local Storage

Open DevTools Console:

```javascript
// Check tenant ID
localStorage.getItem('tenant_id')

// Check all data
Object.entries(localStorage).forEach(([k, v]) => console.log(k, v))
```

### Test Components in Isolation

Create a test file:

```typescript
// test-playground.tsx
import { usePlanner } from './hooks/usePlanner';

export function TestComponent() {
  const { semanticQuery, loading, error, callPlanner } = usePlanner();
  
  return (
    <div>
      <button onClick={() => callPlanner({
        datasource: 'test',
        version: 'v1',
        prompt: 'test',
        mode: 'exploratory'
      })}>
        Test Planner
      </button>
      {loading && <p>Loading...</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}
      {semanticQuery && <pre>{JSON.stringify(semanticQuery, null, 2)}</pre>}
    </div>
  );
}
```

## 📊 Performance Metrics

Expected performance:

| Operation | Expected Time | Notes |
|-----------|---------|-------|
| Load datasources | <500ms | Cached after first load |
| Get bundle | <500ms | Cached per datasource |
| Planner LLM | 2-5s | Depends on Gemini API |
| Executor LLM | 2-5s | Depends on Gemini API |
| SQL Execution | <5s | Depends on query complexity |
| Load results | <1s | Depends on row count |

If slower:
1. Check network latency (Network tab)
2. Check backend load (CPU, memory)
3. Check database query performance (EXPLAIN PLAN)
4. Consider caching frequently accessed bundles
5. Consider pagination for large result sets

## 🚀 Launch Checklist

Before going to production:

- [ ] Test with production database
- [ ] Test with production Gemini API key
- [ ] Test with production tenant IDs
- [ ] Dark theme verified
- [ ] Mobile responsiveness tested
- [ ] Error messages are user-friendly
- [ ] Long queries don't timeout
- [ ] Large result sets load without freezing
- [ ] CSV export works
- [ ] Analytics tracking configured
- [ ] Error tracking configured
- [ ] Security review completed
  - [ ] CORS headers correct
  - [ ] Authentication required
  - [ ] Tenant isolation verified
  - [ ] SQL injection protection in backend
  - [ ] Rate limiting configured

---

For more details, see [README.md](./README.md) or [INTEGRATION.md](./INTEGRATION.md).
