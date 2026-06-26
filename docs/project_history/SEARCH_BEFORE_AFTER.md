# Search Implementation Comparison

## Before & After

### NodeTypeSetupPage - BEFORE ❌
```tsx
// Multiple states
const [filteredNodeTypes, setFilteredNodeTypes] = useState<NodeType[] | null>(null);
const [searchQuery, setSearchQuery] = useState<string>('');

// Complex injector component
const SearchResultsInjector: React.FC<{...}> = ({ tenantId, q, onResults, onError }) => {
  const { data, error } = useSearchNodeTypes(tenantId, q);
  React.useEffect(() => {
    if (error) { onError(); return; }
    if (data) { onResults(data); }
  }, [data, error, onResults, onError]);
  return null;
};

// In render:
<ProfessionalSearchInput
  onSearch={(q) => {
    setSearchQuery(q);
    if (!q.trim()) {
      setFilteredNodeTypes(null);  // Manual state update
    }
  }}
/>

{/* Separate injector component */}
{searchQuery.trim() !== '' && (
  <SearchResultsInjector
    tenantId={tenantId}
    q={searchQuery}
    onResults={(results) => setFilteredNodeTypes(results)}
    onError={() => {
      const ql = searchQuery.toLowerCase();
      setFilteredNodeTypes(
        (nodeTypes || []).filter((nt) => 
          nt.catalog_type_name.toLowerCase().includes(ql) || 
          (nt.description || '').toLowerCase().includes(ql)
        )
      );
    }}
  />
)}

{/* Complex display logic */}
{!isLoading && !error && (filteredNodeTypes ?? nodeTypes) && (
  <NodeTypeTable nodeTypes={filteredNodeTypes ?? nodeTypes ?? []} />
)}
```

### NodeTypeSetupPage - AFTER ✅
```tsx
// Single state
const [searchQuery, setSearchQuery] = useState<string>('');

// Automatic search via React Query
const { data: searchResults, isLoading: isSearching } = useSearchNodeTypes(tenantId, searchQuery);
const displayedNodeTypes = searchQuery.trim() ? (searchResults || []) : (nodeTypes || []);

// In render:
<ProfessionalSearchInput
  data={displayedNodeTypes.map(...)}
  onSearch={(q) => setSearchQuery(q)}  // Simple state update
/>

{/* Simple display logic */}
{!isLoading && !error && displayedNodeTypes && (
  <NodeTypeTable nodeTypes={displayedNodeTypes} />
)}
```

---

### EdgeTypeSetupPage - BEFORE ❌
```tsx
// State for filtering
const [filteredEdgeTypes, setFilteredEdgeTypes] = useState<EdgeType[] | null>(null);

// Client-side only search
<ProfessionalSearchInput
  onSearch={(q) => {
    if (!q.trim()) {
      setFilteredEdgeTypes(null);
      return;
    }
    const ql = q.toLowerCase();
    // Manual filtering - doesn't scale!
    setFilteredEdgeTypes(
      (edgeTypes || []).filter((et) => 
        (et.predicate || '').toLowerCase().includes(ql) || 
        (et.description || '').toLowerCase().includes(ql)
      )
    );
  }}
/>

{/* Display filtered or all */}
{!isLoading && !error && (filteredEdgeTypes ?? edgeTypes) && (
  <EdgeTypeTable edgeTypes={(filteredEdgeTypes ?? edgeTypes) || []} />
)}
```

### EdgeTypeSetupPage - AFTER ✅
```tsx
// Single state
const [searchQuery, setSearchQuery] = useState<string>('');

// Server-side search via React Query (NEW!)
const { data: searchResults, isLoading: isSearching } = useSearchEdgeTypes(tenantId, searchQuery);
const displayedEdgeTypes = searchQuery.trim() ? (searchResults || []) : (edgeTypes || []);

// In render:
<ProfessionalSearchInput
  data={displayedEdgeTypes.map(...)}
  onSearch={(q) => setSearchQuery(q)}  // Simple state update
/>

{/* Simple display logic */}
{!isLoading && !error && displayedEdgeTypes && (
  <EdgeTypeTable edgeTypes={displayedEdgeTypes} />
)}
```

---

## Key Improvements

| Aspect | Before | After |
|--------|--------|-------|
| **State Management** | 2 states + complex callbacks | 1 state + derived value |
| **Search Type** | Mixed (server + client fallback) | Pure server-side |
| **Code Lines** | ~40 lines per page | ~10 lines per page |
| **Consistency** | Different between pages | Identical pattern |
| **Performance** | Client filters large lists | Server filters efficiently |
| **Maintainability** | Complex logic to understand | Easy to read and modify |
| **Loading State** | Only `isLoading` | `isLoading \|\| isSearching` |
| **Edge Types API** | No search hook | New `useSearchEdgeTypes` hook |

---

## React Query Benefits

The new implementation leverages React Query's built-in features:

✅ **Automatic Caching**: Search results cached by query  
✅ **Request Deduplication**: Multiple components searching same term only makes 1 request  
✅ **Automatic Refetching**: Fresh data when needed  
✅ **Loading/Error States**: Built-in state management  
✅ **Conditional Fetching**: `enabled` flag prevents unnecessary requests  

---

## Pattern to Follow

For any future search implementations, use this pattern:

```tsx
// 1. State for search query
const [searchQuery, setSearchQuery] = useState<string>('');

// 2. Server-side search hook
const { data: searchResults, isLoading: isSearching } = useSearchItems(tenantId, searchQuery);

// 3. Derived display data
const displayedItems = searchQuery.trim() ? (searchResults || []) : (allItems || []);

// 4. Simple search input
<ProfessionalSearchInput
  data={displayedItems.map(...)}
  onSearch={setSearchQuery}
/>

// 5. Loading state includes search
{(isLoading || isSearching) && <LoadingSpinner />}

// 6. Display the derived data
<ItemTable items={displayedItems} />
```

This pattern is:
- **Consistent** across all pages
- **Simple** to understand and maintain
- **Performant** with server-side filtering
- **Type-safe** with TypeScript
