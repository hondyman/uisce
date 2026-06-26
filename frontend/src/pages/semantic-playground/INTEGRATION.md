# Semantic Playground Integration Guide

## Quick Start - 5 Steps

### Step 1: Route Integration

Add this to your `AppRoutes.tsx` (or routing configuration):

```typescript
import { lazy, Suspense } from 'react';
import Loading from '@/components/Loading'; // Your loading component

const SemanticPlaygroundPage = lazy(() =>
  import('@/pages/semantic-playground').then(m => ({
    default: m.SemanticPlaygroundPage
  }))
);

// In your routes array:
export const appRoutes = [
  // ... other routes ...
  {
    path: '/semantic-playground',
    element: (
      <Suspense fallback={<Loading />}>
        <SemanticPlaygroundPage />
      </Suspense>
    ),
    title: 'Semantic Playground'
  }
];
```

### Step 2: Navigation Link

Add to your main navigation (AppBar, Sidebar, Menu, etc.):

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

// Or as a button:
<Button 
  component={Link} 
  to="/semantic-playground"
  startIcon={<StorageIcon />}
>
  Semantic Playground
</Button>
```

### Step 3: API Environment

Ensure API base URL is configured:

**In `vite.config.ts`:**
```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  define: {
    'import.meta.env.VITE_API_URL': 
      JSON.stringify(process.env.VITE_API_URL || 'http://localhost:8080/api')
  }
});
```

**In `.env` (development):**
```bash
VITE_API_URL=http://localhost:8080/api
```

**In `.env.production`:**
```bash
VITE_API_URL=https://api.yourdomain.com/api
```

### Step 4: Verify Dependencies

Ensure these packages are installed:

```bash
npm list @mui/material @mui/icons-material react axios

# If missing, install:
npm install @mui/material @mui/icons-material @emotion/react @emotion/styled axios
```

### Step 5: Test

1. Start your app: `npm run dev`
2. Navigate to `/semantic-playground`
3. You should see:
   - Left pane: Datasource/version selectors + NL prompt
   - Middle pane: Empty (waiting for query)
   - Right pane: Empty (waiting for SQL/results)
4. Select a datasource and enter a prompt
5. Click "Generate Query" - should call backend planner LLM
6. Review generated semantic query
7. Click "Execute" - should call executor LLM to generate SQL
8. Click "Run" - should execute and show results

## Backend Verification Checklist

Before testing, verify these endpoints exist and are working:

### Required Endpoints

- [ ] `GET /api/semantic/datasources` - List available datasources
- [ ] `GET /api/semantic/bundles/by-id?datasource=X&version=Y` - Get bundle metadata
- [ ] `GET /api/semantic/bundles/{datasource}/versions` - List bundle versions
- [ ] `POST /api/semantic/plan` - Planner LLM (NL → SemanticQuery)
- [ ] `POST /api/semantic/execute` - Executor LLM (SemanticQuery → SQL)
- [ ] `POST /api/sql/run` - SQL execution

### Optional Endpoints

- [ ] `GET /api/semantic/lineage/{fieldId}` - Get field lineage
- [ ] `GET /api/semantic/bundles/diff` - Compare bundle versions
- [ ] `POST /api/semantic/explain` - Explain query via LLM

**To test endpoints:**

```bash
# List datasources
curl http://localhost:8080/api/semantic/datasources

# Get bundle
curl "http://localhost:8080/api/semantic/bundles/by-id?datasource=customers&version=v1"

# Call planner
curl -X POST http://localhost:8080/api/semantic/plan \
  -H "Content-Type: application/json" \
  -d '{
    "datasource": "customers",
    "version": "v1",
    "prompt": "Show me customers",
    "mode": "exploratory"
  }'
```

## Troubleshooting

### Issue: Module not found errors

**Solution:** Verify import paths match your project structure.

If your page structure is different, update imports in `PlaygroundPage.tsx`:

```typescript
// If components are in different location:
import { NLInputPanel } from '@/components/semantic-playground/panels';

// Or with different path:
import { NLInputPanel } from '../panels/NL';
```

### Issue: "Cannot find module @mui/material"

**Solution:** Install MUI packages:

```bash
npm install @mui/material @mui/icons-material @emotion/react @emotion/styled
```

### Issue: API calls return 404

**Verify:**
1. Backend is running: `curl http://localhost:8080/api/health`
2. API URL is correct in environment
3. Endpoints are implemented in backend
4. CORS headers are set (if needed)

**Add debug logging:**

```typescript
// In api.ts, add at top of each method:
console.log('API Call:', method, url, payload);
```

### Issue: Results showing empty

**Check:**
1. SQL is valid - copy from SQLViewer and test in database UI
2. Query returned no rows (NULL is valid result, empty set is valid)
3. Filters are too restrictive
4. Datasource is empty

### Issue: "X is not a function" errors

**Likely causes:**
1. Import from wrong path
2. Hook used outside of React component
3. Component not wrapped in Suspense boundary

**Fix:**
1. Check import statements match file structure
2. Hooks must be in functional components
3. Lazy-loaded component must be in Suspense

### Issue: Styling looks broken

**Check:**
1. MUI theme is provided at app root:
   ```typescript
   <ThemeProvider theme={theme}>
     <App />
   </ThemeProvider>
   ```
2. Dark mode is correctly configured
3. Colors from semantic-ui theme are available

## File Structure Reference

```
frontend/src/pages/semantic-playground/
├── README.md                     # Feature documentation
├── INTEGRATION.md                # This file
├── types.ts                      # Type definitions
├── index.ts                      # Module exports
├── PlaygroundPage.tsx            # Main component
├── components/
│   ├── NLInputPanel.tsx         # Left pane
│   ├── SemanticQueryEditor.tsx  # Middle pane
│   ├── SQLViewer.tsx            # Top right
│   ├── ResultsTable.tsx         # Bottom right
│   └── index.ts
├── hooks/
│   ├── usePlanner.ts            # Planner LLM
│   ├── useExecutor.ts           # Executor LLM
│   ├── useSQLRunner.ts          # SQL execution
│   ├── useSemanticBundle.ts     # Bundle loading
│   └── index.ts
└── utils/
    ├── api.ts                   # API client
    ├── jsonSchema.ts            # JSON schema config
    └── types.ts                 # Type exports
```

## Performance Optimization

### Enable Code Splitting

The playground uses lazy loading to improve initial load time:

```typescript
const SemanticPlaygroundPage = lazy(() =>
  import('@/pages/semantic-playground')
);
```

This ensures the playground code (~50KB) is only loaded when accessed.

### API Optimization

To cache datasources/versions:

```typescript
// In semantic-playground/hooks/useSemanticBundle.ts, add caching:
const cache = useRef<Record<string, SemanticBundle>>({});

const fetchBundle = useCallback(async (ds: string, v: string) => {
  const key = `${ds}:${v}`;
  if (cache.current[key]) {
    setBundle(cache.current[key]);
    return;
  }
  // ... fetch and cache
  cache.current[key] = result;
}, []);
```

### Results Table Performance

For large result sets (>10K rows):

```typescript
// In ResultsTable.tsx, virtualize rows:
import { FixedSizeList as List } from 'react-window';

// Replace Table.Body with virtual list
<List
  height={500}
  itemCount={results?.rows.length || 0}
  itemSize={35}
  width="100%"
>
  {({ index, style }) => (
    <TableRow style={style}>
      {/* render row */}
    </TableRow>
  )}
</List>
```

## Security Considerations

### CORS

Ensure backend CORS headers include your frontend URL:

```go
// In backend CORS middleware:
router.Use(cors.New(cors.Config{
  AllowOrigins: []string{"http://localhost:3000", "https://yourdomain.com"},
  AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
}))
```

### Authentication

Tenant ID is automatically pulled from localStorage. Ensure backend validates:

```typescript
// In api.ts, tenant ID is added to all requests:
const tenantId = localStorage.getItem('tenant_id');
headers['X-Tenant-ID'] = tenantId;
```

Backend should validate this header:

```go
// In backend middleware:
tenantID := c.GetHeader("X-Tenant-ID")
if tenantID == "" {
  c.JSON(401, "Missing X-Tenant-ID header")
  return
}
```

### Query Validation

Backend should validate:
1. Datasource exists for user's tenant
2. Version is valid
3. Fields in SemanticQuery exist in bundle
4. Generated SQL is safe to execute

## Monitoring & Analytics

### Add Event Tracking

In `PlaygroundPage.tsx`, add analytics:

```typescript
const handleGenerateQuery = async () => {
  analytics.track('semantic_playground.generate_query', {
    datasource: selectedDatasource,
    mode: mode,
  });
  // ... rest of logic
};

const handleRunSQL = async () => {
  analytics.track('semantic_playground.run_sql', {
    datasource: selectedDatasource,
    execution_time: executionTime,
    row_count: results?.row_count,
  });
  // ... rest of logic
};
```

### Add Error Tracking

```typescript
const handleGenerateQuery = async () => {
  try {
    // ...
  } catch (err) {
    errorTracking.captureException(err, {
      context: 'semantic_playground.planner',
      datasource: selectedDatasource,
    });
    showSnackbar('Query generation failed', 'error');
  }
};
```

## Deployment Checklist

- [ ] API URL configured for production environment
- [ ] CORS headers configured on backend
- [ ] Authentication (X-Tenant-ID) configured
- [ ] Backend endpoints verified
- [ ] Error handling tested
- [ ] Results table tested with large datasets
- [ ] CSV export tested
- [ ] Mobile responsiveness verified
- [ ] Dark theme applied correctly
- [ ] Analytics tracking wired
- [ ] Error tracking configured

## Next Steps

1. **Immediate (Today):**
   - [ ] Add route to AppRoutes.tsx
   - [ ] Add navigation link
   - [ ] Test with backend in development

2. **Short-term (This Week):**
   - [ ] Test all three API endpoints
   - [ ] Test with different datasources
   - [ ] Test with large result sets
   - [ ] Fix any endpoint contract mismatches

3. **Medium-term (This Sprint):**
   - [ ] Add lineage visualizer
   - [ ] Add query history/saved queries
   - [ ] Add performance monitoring
   - [ ] Add keyboard shortcuts

4. **Long-term (Future):**
   - [ ] Query templates
   - [ ] Collaboration features
   - [ ] Advanced visualizations
   - [ ] Query optimization suggestions

---

**Questions?** See [README.md](./README.md) for detailed documentation.

