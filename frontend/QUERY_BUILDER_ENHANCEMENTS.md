# Query Builder Results Tab Enhancements

## Overview
Enhanced the Query Builder component with advanced results table features:
1. **Typeahead/Intelligent Search** - Real-time filtering across all columns
2. **Column Header Sorting** - Click column headers to sort ascending/descending
3. **Visual WHERE Clause Builder** - Conditional rule builder inspired by ValidationRuleCreator

## Features Implemented

### 1. Typeahead Search (`searchText` state)
- **Location**: Results table toolbar
- **Behavior**: 
  - Type in the search box to filter results in real-time
  - Searches across ALL columns and values
  - Shows row count: "X of Y rows"
  - Case-insensitive matching
- **Implementation**: 
  ```typescript
  const processedResults = useMemo(() => {
    if (searchText.trim()) {
      const searchLower = searchText.toLowerCase();
      filtered = filtered.filter(row =>
        Object.values(row).some(val =>
          String(val).toLowerCase().includes(searchLower)
        )
      );
    }
  }, [results, searchText]);
  ```

### 2. Column Sorting
- **Location**: Click on any column header in the table
- **Features**:
  - Sort ascending (A→Z, 0→9)
  - Sort descending (Z→A, 9→0)
  - Visual indicator (up/down arrow) shows current sort direction
  - Supports numeric, string, and date comparisons
  - Null values handled gracefully
- **Implementation**:
  ```typescript
  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };
  ```

### 3. Conditional WHERE Clause Builder
- **Location**: "Filters" button in results toolbar
- **Design Pattern**: Inspired by ValidationRuleCreator
- **Features**:
  - Add/remove conditions dynamically
  - Choose fields from available columns (with autocomplete)
  - 13 operators: equals, not_equals, contains, starts_with, ends_with, >, <, >=, <=, between, in, IS NULL, IS NOT NULL
  - Set values for each condition
  - Choose logic: match ALL conditions (AND) or ANY condition (OR)
  - Visual condition cards with delete buttons
  - Automatically generates SQL WHERE clause

#### Supported Operators
| Operator | SQL | Use Cases |
|----------|-----|-----------|
| equals | `= 'value'` | Exact matches |
| not_equals | `!= 'value'` | Exclusions |
| contains | `LIKE '%value%'` | Substring searches |
| starts_with | `LIKE 'value%'` | Prefix matches |
| ends_with | `LIKE '%value'` | Suffix matches |
| greater_than | `> value` | Numeric comparisons |
| less_than | `< value` | Numeric comparisons |
| greater_or_equal | `>= value` | Numeric ranges |
| less_or_equal | `<= value` | Numeric ranges |
| between | Custom SQL | Range queries |
| in | Custom SQL | Multiple values |
| is_null | `IS NULL` | Null checks |
| is_not_null | `IS NOT NULL` | Non-null checks |

## Component Structure

### ConditionBuilderDialog
A reusable dialog component for building WHERE clauses:
```tsx
<ConditionBuilderDialog
  open={conditionBuilderOpen}
  onClose={() => setConditionBuilderOpen(false)}
  onApply={handleApplyConditions}
  availableFields={getTableHeaders()}
  currentConditions={whereConditions}
/>
```

**Props**:
- `open`: boolean - Control dialog visibility
- `onClose`: () => void - Called when dialog closes
- `onApply`: (conditions: ConditionGroup) => void - Called when filters applied
- `availableFields`: string[] - List of column names to filter on
- `currentConditions`: ConditionGroup - Initial conditions (for editing)

## Usage Example

```tsx
// In results section:
<Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }}>
  {/* Search */}
  <TextField
    placeholder="Search results..."
    variant="outlined"
    size="small"
    value={searchText}
    onChange={(e) => setSearchText(e.target.value)}
    InputProps={{
      startAdornment: <SearchIcon sx={{ mr: 1, color: 'text.secondary' }} />,
    }}
    sx={{ flex: 1 }}
  />
  
  {/* Filter Builder */}
  <Button
    variant="outlined"
    size="small"
    startIcon={<TuneIcon />}
    onClick={() => setConditionBuilderOpen(true)}
  >
    Filters
  </Button>
  
  {/* Row Count */}
  <Typography variant="body2">
    {processedResults.length} of {results.length} rows
  </Typography>
</Stack>
```

## State Management

### New State Variables
```typescript
const [searchText, setSearchText] = useState('');           // Search input
const [sortColumn, setSortColumn] = useState<string | null>(null);  // Current sort column
const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc'); // Sort direction
const [conditionBuilderOpen, setConditionBuilderOpen] = useState(false);  // Dialog control
const [whereConditions, setWhereConditions] = useState<ConditionGroup>({ 
  conditions: [], 
  logic: 'AND' 
});  // Current filter conditions
```

### Derived State
```typescript
const processedResults = useMemo(() => {
  // Applies search + sorting to results
  // Updates whenever: results, searchText, sortColumn, or sortDirection changes
}, [results, searchText, sortColumn, sortDirection]);
```

## Integration with Query Execution

When filters are applied via the Condition Builder:
1. User clicks "Filters" button
2. Opens ConditionBuilderDialog
3. User builds conditions visually
4. Clicks "Apply Filters"
5. `handleApplyConditions()` converts to SQL WHERE clause
6. Updates the `filters` state
7. Next query execution includes the WHERE clause

```typescript
const handleApplyConditions = (conditions: ConditionGroup) => {
  setWhereConditions(conditions);
  
  // Convert conditions to SQL WHERE clause
  const whereClauses = conditions.conditions
    .filter(c => c.field && c.operator && (c.value || ...))
    .map(c => {
      // Switch statement converts each condition to SQL
    })
    .filter(Boolean);

  if (whereClauses.length > 0) {
    // Sets filters state with SQL WHERE clause
    setFilters(whereClauses.join(` ${conditions.logic} `));
  }
};
```

## UI/UX Features

### Search Box
- 🔍 Search icon in input
- Real-time filtering (no debounce needed for client-side)
- Shows matching row count
- Searches across all columns simultaneously

### Sortable Headers
- Click column header to sort
- Arrow icon indicates sort direction
- Hover effect shows it's clickable
- Smart type detection (numeric vs string)

### Filter Builder Dialog
- Modal dialog interface
- Card-based condition layout
- Autocomplete field selection
- Dropdown operator selection
- Dynamic value input
- Add/remove condition buttons
- Apply/Cancel actions

## Performance Considerations

- **useMemo**: Results filtering and sorting memoized to prevent re-renders
- **Client-side Filtering**: Search and sort happen in-browser (fast for small datasets)
- **Large Datasets**: For 10k+ rows, consider server-side pagination/filtering
- **Nullable Comparisons**: Properly handles NULL values in sorting

## Future Enhancements

1. **Server-side Filtering**: Send WHERE clause to API for large datasets
2. **Saved Filters**: Store and retrieve common filter combinations
3. **Advanced Expressions**: Support for complex expressions (e.g., `(A AND B) OR C`)
4. **Export Results**: Download filtered results as CSV/Excel
5. **Column Visibility**: Toggle columns on/off
6. **Group By**: Add grouping functionality
7. **Pagination**: Client-side pagination for large result sets
8. **Result Statistics**: Min/max/average for numeric columns

## Files Modified

- `/Users/eganpj/GitHub/semlayer/frontend/src/features/query-builder/pages/QueryBuilder.tsx`
  - Added imports for new MUI components and icons
  - Added types: `WhereCondition`, `ConditionGroup`
  - Added `ConditionBuilderDialog` component
  - Added state for search, sort, and conditions
  - Added `processedResults` memoized selector
  - Added `handleSort()` method
  - Added `handleApplyConditions()` method
  - Enhanced results section with search, sorting, and filter UI

## Testing Checklist

- [ ] Search across multiple columns
- [ ] Search with special characters
- [ ] Sort ascending/descending
- [ ] Sort by different data types (string, number, date)
- [ ] Add conditions in filter builder
- [ ] Remove conditions
- [ ] Switch AND/OR logic
- [ ] Apply filters and verify SQL WHERE clause
- [ ] Combine search + filtering
- [ ] Combine sorting + filtering
- [ ] Empty results handling
- [ ] NULL value handling
