# Semantic Playground - Complete Implementation Guide

## 📋 Overview

The **Semantic Playground** is a premium developer tool for designing, debugging, and exploring the semantic layer. It brings together:

- **NL Input Panel** - Write natural language queries
- **Semantic Query Editor** - JSON-based semantic query visualization and editing
- **SQL Viewer** - See generated SQL with syntax highlighting
- **Results Table** - Paginated, searchable results with export capabilities

## 🎯 Architecture

### Component Structure

```
semantic-playground/
├── components/
│   ├── NLInputPanel.tsx          # Natural language input (left pane)
│   ├── SemanticQueryEditor.tsx   # JSON query editor (middle pane)
│   ├── SQLViewer.tsx             # Generated SQL viewer (top right)
│   ├── ResultsTable.tsx          # Results with pagination (bottom right)
│   └── index.ts
├── hooks/
│   ├── usePlanner.ts             # Calls planner LLM
│   ├── useExecutor.ts            # Calls executor LLM
│   ├── useSQLRunner.ts           # Runs SQL queries
│   ├── useSemanticBundle.ts      # Fetches bundle metadata
│   └── index.ts
├── utils/
│   ├── api.ts                    # API client for all endpoints
│   ├── jsonSchema.ts             # JSON schema + Monaco config
│   └── types.ts                  # TypeScript types and interfaces
├── PlaygroundPage.tsx            # Main orchestration component
├── index.ts                      # Export entry point
└── README.md                     # This file
```

### Data Flow

```
User Input (NL)
    ↓
usePlanner Hook
    ↓
Backend /api/semantic/plan
    ↓
SemanticQuery JSON
    ↓
SemanticQueryEditor (display & optional edit)
    ↓
User clicks "Execute"
    ↓
useExecutor Hook
    ↓
Backend /api/semantic/execute
    ↓
Generated SQL
    ↓
SQLViewer (display)
    ↓
User clicks "Run"
    ↓
useSQLRunner Hook
    ↓
Backend /api/sql/run
    ↓
Query Results
    ↓
ResultsTable (display with pagination, search, sort)
```

## 🔌 API Integration

### Exposed Hooks

#### `usePlanner()`
Converts natural language → semantic query JSON

```typescript
const {
  semanticQuery,      // SemanticQuery object
  explanation,        // Optional AI explanation
  confidence,         // LLM confidence (0-1)
  warnings,           // Array of warnings
  loading,            // Loading state
  error,              // Error message
  callPlanner,        // Trigger planner LLM
} = usePlanner();

// Usage:
await callPlanner({
  datasource: "customers",
  version: "v1",
  prompt: "Show recent US customers",
  mode: "exploratory",
});
```

#### `useExecutor()`
Converts semantic query → SQL

```typescript
const {
  generatedSQL,       // SQL string
  warnings,           // Array of warnings
  loading,            // Loading state
  error,              // Error message
  callExecutor,       // Trigger executor LLM
} = useExecutor();

// Usage:
await callExecutor({
  datasource: "customers",
  version: "v1",
  semantic_query: { /* ... */ },
});
```

#### `useSQLRunner()`
Runs SQL and fetches results

```typescript
const {
  results,            // QueryExecutionResponse
  executionTime,      // Time in milliseconds
  loading,            // Loading state
  error,              // Error message
  runSQL,             // Trigger SQL execution
} = useSQLRunner();

// Usage:
await runSQL("SELECT * FROM customers LIMIT 20");
```

#### `useSemanticBundle()`
Fetches semantic bundle metadata

```typescript
const {
  bundle,             // SemanticBundle object
  versions,           // Array of versions
  loading,            // Loading state
  error,              // Error message
  fetchBundle,        // Fetch bundle by datasource/version
  fetchVersions,      // Fetch available versions
} = useSemanticBundle();

// Usage:
await fetchBundle("customers", "v1");
await fetchVersions("customers");
```

### API Client (semanticPlaygroundApi)

All endpoints automatically include `X-Tenant-ID` header:

```typescript
// Datasources
await semanticPlaygroundApi.getDatasources()
  → GET /api/semantic/datasources

// Bundles
await semanticPlaygroundApi.getBundle(datasource, version)
  → GET /api/semantic/bundles/by-id?datasource=...&version=...

await semanticPlaygroundApi.getBundleVersions(datasource)
  → GET /api/semantic/bundles/{datasource}/versions

// Planning (NL → SemanticQuery)
await semanticPlaygroundApi.callPlanner(request)
  → POST /api/semantic/plan
  {
    "datasource": "customers",
    "version": "v1",
    "prompt": "Show recent customers",
    "mode": "exploratory"
  }

// Execution (SemanticQuery → SQL)
await semanticPlaygroundApi.callExecutor(request)
  → POST /api/semantic/execute
  {
    "datasource": "customers",
    "version": "v1",
    "semantic_query": { /* ... */ }
  }

// SQL Execution
await semanticPlaygroundApi.runSQL(request)
  → POST /api/sql/run
  {
    "sql": "SELECT * FROM customers LIMIT 20",
    "limit": 1000,
    "timeout": 30000
  }

// Lineage
await semanticPlaygroundApi.getFieldLineage(fieldId)
  → GET /api/semantic/lineage/{fieldId}

// Diff
await semanticPlaygroundApi.diffBundles(datasource, fromVersion, toVersion)
  → GET /api/semantic/bundles/diff?datasource=...&from=...&to=...

// Explain
await semanticPlaygroundApi.explainQuery(datasource, query)
  → POST /api/semantic/explain
  {
    "datasource": "customers",
    "query": { /* semantic query */ }
  }
```

## 🎨 Component Details

### NLInputPanel

**Props:**
- `datasources` - List of available datasources
- `selectedDatasource` - Currently selected datasource ID
- `selectedVersion` - Currently selected version
- `versions` - Array of available versions
- `prompt` - Current NL prompt text
- `mode` - Query mode ("exploratory" | "strict" | "CRUD")
- `loading` - Loading indicator
- `error` - Error message to display
- `onDatasourceChange` - Callback when datasource changes
- `onVersionChange` - Callback when version changes
- `onPromptChange` - Callback when prompt text changes
- `onModeChange` - Callback when mode changes
- `onGenerate` - Callback when "Generate Query" clicked
- `onClear` - Callback when "Clear" clicked
- `bundle` - Optional current bundle for metadata display

**Features:**
- Dark theme with accent colors
- Dropdown selectors for datasource, version, and mode
- Multiline textarea for NL prompt
- Mode selector with inline help text
- Status chips showing field count
- Keyboard shortcut: Ctrl+Enter to generate
- Clear button
- Error display

### SemanticQueryEditor

**Props:**
- `query` - Current SemanticQuery JSON
- `bundle` - SemanticBundle for context
- `loading` - Loading indicator
- `error` - Validation error message
- `warnings` - Array of warnings
- `onQueryChange` - Callback when query is modified
- `onExplain` - Optional callback to explain query
- `onShowLineage` - Optional callback to show lineage

**Features:**
- JSON display with syntax highlighting
- Edit mode with JSON validation
- Format JSON button
- Copy to clipboard
- Explain query (generates LLM explanation)
- Show lineage (field → physical column trail)
- Validation error display
- Field count badge
- Warning indicator

### SQLViewer

**Props:**
- `sql` - Generated SQL string
- `loading` - Loading indicator
- `error` - Error message
- `warnings` - Array of warnings
- `onExecute` - Callback to execute SQL
- `onDownloadCSV` - Optional callback to export CSV
- `executingSQL` - SQL execution loading state

**Features:**
- SQL syntax highlighting
- Automatic formatting (SELECT, FROM, WHERE, etc. on new lines)
- Copy button
- Execute button (triggers SQL runner)
- Export CSV button
- Error and warning display
- Loading state

### ResultsTable

**Props:**
- `results` - QueryExecutionResponse with rows
- `executionTime` - Time in milliseconds
- `loading` - Loading indicator
- `error` - Error message
- `sql` - Optional SQL for reference

**Features:**
- Sortable columns (click header to toggle asc/desc)
- Search/filter across all columns
- Pagination (10/20/50/100 rows per page)
- Sticky header
- Dark theme rows with alternating background
- NULL value display
- JSON/object serialization
- Row count and execution time badges
- CSV export
- Monospace font for values

## 🚀 Getting Started

### 1. Installation

The Semantic Playground is already integrated into your frontend. No additional installation needed.

### 2. Add to Routes

Update your `AppRoutes.tsx` or routing configuration:

```typescript
import { lazy } from 'react';

const SemanticPlaygroundPage = lazy(() => 
  import('./pages/semantic-playground').then(m => ({ 
    default: m.SemanticPlaygroundPage 
  }))
);

// In your routes array:
{
  path: '/semantic-playground',
  element: <SemanticPlaygroundPage />
}
```

### 3. Add Navigation Link

Add a menu item to your navigation:

```typescript
import StorageIcon from '@mui/icons-material/Storage';

// In your navigation menu:
<MenuItem 
  component={Link} 
  to="/semantic-playground"
>
  <StorageIcon sx={{ mr: 1 }} />
  Semantic Playground
</MenuItem>
```

### 4. Configure Environment

Ensure your `.env` or `vite.config.ts` sets the API URL:

```bash
# .env
VITE_API_URL=http://localhost:8080/api
```

Or in `vite.config.ts`:

```typescript
export default defineConfig({
  define: {
    'import.meta.env.VITE_API_URL': 
      JSON.stringify('http://localhost:8080/api')
  }
})
```

### 5. Verification

Check that MUI components are properly imported in your project. The playground uses:
- `@mui/material` (Box, Card, Button, Table, etc.)
- `@mui/icons-material` (various icons)

If missing:
```bash
npm install @mui/material @mui/icons-material @emotion/react @emotion/styled
```

## 🎯 Usage Examples

### Example 1: Simple Query

1. Select datasource: "customers"
2. Select version: "v1"
3. Enter prompt: "Show me the 20 most recent customers"
4. Click "Generate Query"
5. Review semantic query
6. Click "Execute"
7. See generated SQL
8. Click "Run"
9. View results

### Example 2: Advanced Query with Subtype Filtering

1. Select datasource: "customers"
2. Select version: "v1"
3. Select mode: "exploratory"
4. Enter prompt: "Show loyalty_points for retail customers in the US"
5. Click "Generate Query"
6. Semantic query should infer: `{"select": ["loyalty_points"], "filters": [{"field": "country", "op": "=", "value": "US"}, {"field": "customer_type", "op": "=", "value": "RETAIL"}]}`
7. Click "Execute"
8. Generated SQL includes: `WHERE customer_type = 'RETAIL' AND country = 'US'`
9. Run and export results

### Example 3: Manual Query Editing

1. Generate query as usual
2. Click "Edit JSON" in Semantic Query Editor
3. Modify the JSON (e.g., change limit, add filters)
4. Click "Apply"
5. Click "Execute" to regenerate SQL with changes
6. Run new SQL

## 🔍 Advanced Features

### Explain Query

Click "Explain" in the Semantic Query Editor:
- Opens dialog with LLM explanation
- Shows why fields were included
- Shows inferred fields (if exploratory mode)
- Shows physical column mappings

### Show Lineage

Click "Lineage" in the Semantic Query Editor:
- Shows field UUID
- Shows physical table/column
- Shows subtype information
- Shows field version history
- Shows field aliases

### Query Modes

**Exploratory Mode**
- LLM infers related fields
- Example: `loyalty_points` → automatically adds `customer_type = 'RETAIL'`
- Best for discovery

**Strict Mode**
- LLM only includes explicitly mentioned fields
- No inference
- Best for production/reporting

**CRUD Mode**
- For CREATE/UPDATE/DELETE operations
- Similar to strict mode but for data modification

## 🛠️ Customization

### Theme

Colors used (dark theme):
- Background: `#0d1117`
- Card background: `#1e1e1e`
- Borders: `#333`
- Text: `#ddd`
- Accent: `#2196F3` (blue), `#4CAF50` (green), `#FF9800` (orange)

To customize, modify theme in component `sx` props:

```typescript
sx={{
  backgroundColor: '#1e1e1e',  // Change this
  color: '#fff',               // And this
}}
```

### Column Customization

In `ResultsTable`, customize column display:

```typescript
// Add this to component props
columnConfig?: {
  hidden?: string[];        // Hide columns
  order?: string[];         // Reorder columns
  formatters?: Record<string, (val: any) => string>;
}
```

### Export Formats

Currently supports CSV. To add JSON export:

```typescript
const handleExportJSON = () => {
  const json = JSON.stringify(results?.rows, null, 2);
  const blob = new Blob([json], { type: 'application/json' });
  // ... similar to CSV export
};
```

## 🐛 Troubleshooting

### "Failed to load datasources"
- Check API URL in environment variables
- Verify backend is running
- Check CORS headers

### "Planner LLM not working"
- Verify `GEMINI_API_KEY` is set in backend
- Check backend logs for Gemini errors
- Verify network request to `/api/semantic/plan`

### Results showing as "NULL"
- This is normal for NULL values in database
- Check the actual SQL to verify
- Filters may filter out all rows

### Text too small/large
- Adjust `fontSize` in editor options (jsonSchema.ts)
- Modify table cell padding (ResultsTable.tsx)

## 📚 API Endpoint Details

### POST /api/semantic/plan

**Request:**
```json
{
  "datasource": "customers",
  "version": "v1",
  "prompt": "Show me recent customers",
  "mode": "exploratory"
}
```

**Response:**
```json
{
  "semantic_query": {
    "datasource": "customers",
    "version": "v1",
    "select": ["customer_id", "customer_name", "created_at"],
    "filters": [],
    "order_by": [{"field": "created_at", "direction": "desc"}],
    "limit": 20
  },
  "explanation": "Selected fields for customer overview, ordered by most recent",
  "confidence": 0.92,
  "warnings": []
}
```

### POST /api/semantic/execute

**Request:**
```json
{
  "datasource": "customers",
  "version": "v1",
  "semantic_query": { /* ... */ }
}
```

**Response:**
```json
{
  "generated_sql": "SELECT ... FROM ... WHERE ... LIMIT 20",
  "semantic_sql": { /* echoed back */ },
  "execution_plan": "...",
  "warnings": []
}
```

### POST /api/sql/run

**Request:**
```json
{
  "sql": "SELECT * FROM customers LIMIT 20",
  "limit": 1000,
  "timeout": 30000
}
```

**Response:**
```json
{
  "rows": [
    {"customer_id": 1, "name": "Alice", "email": "alice@example.com"},
    {"customer_id": 2, "name": "Bob", "email": "bob@example.com"}
  ],
  "row_count": 2,
  "columns": ["customer_id", "name", "email"],
  "execution_time_ms": 45
}
```

## 🎓 Learning Resources

- **Semantic Layer Docs**: [See backend documentation]
- **SemanticQuery JSON Schema**: [See jsonSchema.ts]
- **LLM Prompt Examples**: [See backend FULL_EXAMPLE_WALKTHROUGH.md]
- **API Reference**: [See backend API endpoints]

## 📝 Future Enhancements

- [ ] Query history/saved queries
- [ ] Collaboration features (share queries)
- [ ] Performance monitoring/explain plans
- [ ] Query optimization suggestions
- [ ] Multi-query support (unions, CTEs)
- [ ] Visual query builder (no-code mode)
- [ ] Real-time query validation
- [ ] Dark/light theme toggle
- [ ] Keyboard shortcuts reference
- [ ] Query templates/snippets

## 🤝 Contributing

Code is organized into logical sections:
- Components in `components/` with individual files
- Hooks in `hooks/` with naming convention `use*`
- Utilities in `utils/` for shared logic
- Main orchestration in `PlaygroundPage.tsx`

To add a feature:
1. Create component in `components/`
2. Add hooks if needed in `hooks/`
3. Import in `PlaygroundPage.tsx`
4. Integrate into layout

---

**Happy Querying!** 🚀
