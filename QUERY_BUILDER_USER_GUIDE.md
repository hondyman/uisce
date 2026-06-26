# Query Builder - Feature Usage Guide

## Quick Start

Navigate to: **http://localhost:5173/reports/query-builder**

---

## 1. Execute a Query

1. Left panel: Select dimensions and measures from schema
2. Main panel: Configure view, dimensions, measures
3. Click **"Execute Query"** button
4. See results in chart and table below

---

## 2. Search Results (Typeahead)

### Basic Text Search
```
1. Click the search box at the top of the results table
2. Type: "USA", "active", "2024", etc.
3. Results filter in real-time as you type
4. Shows: "Showing 45 of 1000 rows" (filtered count)
```

### Column Typeahead
```
1. Click the search box
2. Start typing a column name: "coun", "stat", "reg"
3. See matching column suggestions appear
4. Click a suggestion (e.g., "country")
5. Search will now focus on that column (optional feature)
```

### Clear Search
```
1. Click the search box
2. Clear the text (Ctrl+A, Delete)
3. Results return to unfiltered state
```

---

## 3. Sort Columns

### Single Column Sort

**Ascending (↑):**
```
1. Click any column header
2. Arrow icon appears pointing up
3. Results sorted A→Z or 0→9
```

**Descending (↓):**
```
1. Click the same column header again
2. Arrow icon flips pointing down
3. Results sorted Z→A or 9→0
```

**Change Sort Column:**
```
1. Currently: Sorting by "country" ↑
2. Click different column: "amount"
3. Sort immediately switches to "amount" ↑
```

**Remove Sort:**
```
1. (Sorting always active - can't turn off)
2. Just click different columns to change
```

---

## 4. Build Filters (Conditional Builder)

### Open the Builder
```
1. Click blue "Filters" button in control bar
2. Dialog opens with condition builder
```

### Add Conditions
```
1. Click "Add Condition" button
2. Select field from dropdown (e.g., "status")
3. Select operator (=, !=, >, <, LIKE, IN, IS NULL, etc.)
4. Enter value in text box
5. Additional conditions automatically combine with AND
```

### Combine Conditions

**AND Logic (all conditions must match):**
```
Example:
  country = "USA" AND amount > 1000
  → Only shows US orders over $1000
```

**OR Logic (any condition can match):**
```
1. Click "Combine with" dropdown (currently "AND")
2. Select "OR"
3. Now shows: country = "USA" OR amount > 1000
  → Shows ANY US row OR ANY row over $1000
```

### Remove Conditions
```
1. Find the condition
2. Click the red X (delete) button
3. Condition removed from filter
```

### Apply Filters
```
1. Build your conditions
2. Click "Apply" or "OK" button
3. Filter automatically applied to results
4. WHERE clause shown in query section
5. Results update with filter applied
```

### Clear All Filters
```
1. Edit the filter section
2. Delete all conditions
3. Results return to showing all rows
```

---

## 5. Select Result Limit

### Change Row Limit
```
1. Find "Result Limit" dropdown above table
2. Current value: "100 rows"
3. Click dropdown arrow
4. Choose one of:
   • 100 rows (default, fast)
   • 1,000 rows (standard)
   • 10,000 rows (large result set)
5. Results immediately reload with new limit
6. Row count updates: "Showing 1-100 of [new limit]"
```

### When to Use Each Limit

**100 Rows:**
- Quick preview
- Default for performance
- Best with search/filters

**1,000 Rows:**
- Detailed analysis
- Comfortable pagination
- Balanced performance

**10,000 Rows:**
- Complete data export
- Comprehensive analysis
- Slower navigation

---

## 6. Pagination (Lazy Loading)

### Navigate Pages

**View Current Page:**
```
Look at bottom right of table:
"Page 1 of 50" (if limit=100 and results=5000)
```

**Go to Next Page:**
```
1. Results showing rows 1-100
2. Click "Next" button
3. Results show rows 101-200
4. Page counter updates to "Page 2 of 50"
```

**Go to Previous Page:**
```
1. Results showing rows 101-200
2. Click "Previous" button
3. Results show rows 1-100
4. Page counter updates to "Page 1 of 50"
```

**Disable Navigation:**
```
- "Previous" button disabled on first page
- "Next" button disabled on last page
- Shows: "Page 1 of 1" if results fit on one page
```

---

## 7. Combined Workflows

### Workflow 1: Quick Data Preview
```
1. Execute query
2. Type search term to narrow down
3. Click column to sort by most relevant
4. Review top 100 rows
```

### Workflow 2: Detailed Analysis
```
1. Execute query
2. Click "Filters" → Build complex WHERE clause
3. Apply filters
4. Change limit to 1000
5. Click column to sort by key metric
6. Page through results with Previous/Next
```

### Workflow 3: Export Preparation
```
1. Execute query
2. Build filters (only want specific records)
3. Apply filters
4. Change limit to 10000
5. Navigate all pages to verify data
6. (Future: Export button would download here)
```

### Workflow 4: Sorting & Search
```
1. Execute query
2. Click "Amount" column → sort ascending
3. Type "USA" in search → filter to USA
4. Results now show: USA orders, smallest to largest
5. Review top entries in detail
```

---

## 8. Troubleshooting

### Search Not Working
```
❌ "My search term doesn't filter results"
✅ Check: Are you typing in the search box?
✅ Try: Search for different term or clear box
✅ Try: Refresh page and re-execute query
```

### Sort Not Changing
```
❌ "Column header click doesn't change order"
✅ Check: Click appears to work, but order same?
✅ Try: Values might actually be sorted (check carefully)
✅ Try: All values might be identical
```

### Pagination Buttons Disabled
```
❌ "Can't click Next or Previous"
✅ Expected: At first/last page, buttons disabled
✅ Expected: Results fit on 1 page, pagination hidden
✅ Try: Change limit to get more pages
```

### Filters Not Applied
```
❌ "Built filter but results don't change"
✅ Check: Did you click Apply button?
✅ Check: Does filter actually match any rows?
✅ Try: Click "Filters" again to verify conditions
```

### Limit Change Does Nothing
```
❌ "Changed to 1000 rows but still showing 100"
✅ Check: Did results originally have < 100 rows?
✅ Try: Execute a new query that returns more data
```

---

## 9. Keyboard Shortcuts

| Action | Shortcut |
|--------|----------|
| Focus search | Tab (navigate to field) |
| Clear search | Ctrl+A then Delete |
| Open filters | Alt+F (if enabled) |
| Sort column | Click header |
| Next page | Ctrl+→ (if implemented) |
| Prev page | Ctrl+← (if implemented) |

---

## 10. Tips & Tricks

### Pro Tips

1. **Combine Search + Sort**
   - Search for specific records
   - Then click column to sort them
   - Great for finding top results

2. **Use Filters for Complex Logic**
   - Conditional builder handles AND/OR
   - Saves time vs. manually typing WHERE
   - Can save filter presets (future feature)

3. **Increase Limit Progressively**
   - Start with 100 rows for quick look
   - Move to 1,000 for analysis
   - Use 10,000 only when needed

4. **Cross-Tab Sorting**
   - Sort by one column
   - Search while sorted
   - Naturally shows relevant records first

### Performance Tips

1. Apply filters BEFORE increasing limit
2. Use search to narrow results first
3. Sort after filtering (smaller result set)
4. Don't need 10,000 rows? Keep it at 100-1,000

---

## 11. Feature Status

| Feature | Status | Notes |
|---------|--------|-------|
| Search/Typeahead | ✅ Live | Real-time filtering |
| Column Sorting | ✅ Live | Single-column sort |
| Conditional Filters | ✅ Live | AND/OR logic |
| Limit Selector | ✅ Live | 100/1k/10k rows |
| Pagination | ✅ Live | Page-by-page nav |
| Export | ⏳ Future | Download filtered results |
| Saved Filters | ⏳ Future | Save/load presets |
| Virtual Scroll | ⏳ Future | For 100k+ rows |
| Multi-Sort | ⏳ Future | Sort by multiple columns |

---

## Support

For issues or feature requests, check:
1. This guide above
2. Troubleshooting section (#8)
3. Report issues to team
