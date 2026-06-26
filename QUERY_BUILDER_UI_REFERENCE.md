# Query Builder Results Tab - UI Layout

## Visual Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  Results                                                                    │
│  ┌──────────────────────────────┬──────────────────────────────────────────┐
│  │  Chart                       │  Table                                   │
│  │  ┌──────────────────────────┐│  ┌────────────────────────────────────┐ │
│  │  │                          ││  │ Search and Control Bar              │ │
│  │  │  [Bar Chart Image]       ││  │                                     │ │
│  │  │                          ││  │  ┌──────────────────────────────┐   │ │
│  │  │                          ││  │  │🔍 Search or select column...│ [×]│ │
│  │  │                          ││  │  └──────────────────────────────┘   │ │
│  │  │                          ││  │  [Search Dropdown showing:           │ │
│  │  │                          ││  │   - country                         │ │
│  │  │                          ││  │   - amount                          │ │
│  │  │                          ││  │   - status                          │ │
│  │  │                          ││  │   - date]                           │ │
│  │  │                          ││  │                                     │ │
│  │  │                          ││  │  [Filter Limit Dropdown] [Row Info] │ │
│  │  │                          ││  │  [Result Limit ▼]                   │ │
│  │  │                          ││  │   100 rows (selected)               │ │
│  │  │                          ││  │   1,000 rows                        │ │
│  │  │                          ││  │   10,000 rows                       │ │
│  │  │                          ││  │                                     │ │
│  │  │                          ││  │  Showing 1-100 of 1000 rows         │ │
│  │  │                          ││  │                                     │ │
│  │  │                          ││  ├────────────────────────────────────┤ │
│  │  │                          ││  │                                     │ │
│  │  └──────────────────────────┘│  │  Table Headers (Clickable):        │ │
│  │                               │  │  ┌──────┬────────┬─────┬────────┐ │ │
│  │                               │  │  │ ID ↑ │ Country│Amt  │ Status │ │ │
│  │                               │  │  │ (Sort)│        │     │        │ │ │
│  │                               │  │  ├──────┼────────┼─────┼────────┤ │ │
│  │                               │  │  │ 101  │ USA    │5000 │ Active │ │ │
│  │                               │  │  ├──────┼────────┼─────┼────────┤ │ │
│  │                               │  │  │ 102  │ Canada │3200 │ Pending│ │ │
│  │                               │  │  ├──────┼────────┼─────┼────────┤ │ │
│  │                               │  │  │ 103  │ Mexico │4100 │ Active │ │ │
│  │                               │  │  ├──────┼────────┼─────┼────────┤ │ │
│  │                               │  │  │ ... rows continue ...         │ │ │
│  │                               │  │  ├──────┴────────┴─────┴────────┤ │ │
│  │                               │  │  │                              │ │ │
│  │                               │  │  Page 1 of 50  [Previous] [Next]│ │ │
│  │                               │  │                              │ │ │
│  │                               │  └─────────────────────────────────┘ │
│  │                               │                                        │
│  └───────────────────────────────┴────────────────────────────────────────┘
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Breakdown

### 1. Search/Typeahead Box
```
Position: Top left of table section
Style: Material-UI Autocomplete field
Icon: Search icon (left side)
Placeholder: "Search results or select column..."
Width: Flexible, takes ~50% of width
Options: Column names (dynamic)
Behavior:
  - Type to search
  - Suggestions appear as you type
  - Can select column suggestion
  - Pressing Enter applies search
```

### 2. Filters Button
```
Position: Top center of table section
Type: Outlined button
Icon: Tune/Sliders icon
Label: "Filters"
Behavior:
  - Clicks open ConditionBuilderDialog
  - Allows building complex WHERE clauses
  - Shows AND/OR logic toggle
  - Can add multiple conditions
```

### 3. Result Limit Dropdown
```
Position: Left side of second row
Type: Material-UI Select field
Label: "Result Limit"
Options:
  ☐ 100 rows (default)
  ☐ 1,000 rows
  ☐ 10,000 rows
Width: ~120px minimum
Behavior:
  - Select new limit
  - Pagination resets to page 1
  - Results refresh immediately
```

### 4. Row Count Info
```
Position: Right side of second row
Format: "Showing 1-100 of 1000 rows"
Color: Secondary text color (gray)
Calculation: 
  - Start: currentPage * pageSize + 1
  - End: Math.min((currentPage + 1) * pageSize, limit)
  - Total: Math.min(limit, results.length)
Updates: Real-time when filters/limit/page change
```

### 5. Sortable Column Headers
```
Position: Top row of table
Styling:
  - Background: Light gray (#f5f5f5)
  - Font: Bold, 600 weight
  - Cursor: Pointer
Hover Effect: Darker gray (#eeeeee)
Interaction:
  - Click to sort ascending
  - Click again to sort descending
  - Click different column to change sort
Sort Indicator:
  - Arrow icon appears next to active column
  - Points up (↑) for ascending
  - Points down (↓) for descending
  - Icon rotates: 0° for ↑, 180° for ↓
```

### 6. Results Table
```
Position: Center/main area below headers
Type: Material-UI Table with sticky header
Max Height: 440px (scroll if needed)
Border: 1px solid divider color
Border Radius: 4px
Rows: Fixed at 100 per page
Content: Dynamic based on results
Hover: Light gray background (#fafafa)
Empty State: "No results match your search criteria"
```

### 7. Pagination Controls
```
Position: Bottom right of table
Visibility: Only shown if multiple pages
Layout:
  ┌─────────────────────────────┐
  │ Page 1 of 50  [Prev] [Next] │
  └─────────────────────────────┘

Components:
  - Text: "Page X of Y" (left)
  - Buttons: Previous, Next (right)

Button States:
  - Previous: Disabled on page 1, enabled otherwise
  - Next: Disabled on last page, enabled otherwise
  
Button Behavior:
  - Previous: setCurrentPage(p => Math.max(0, p - 1))
  - Next: setCurrentPage(p => p + 1)
  
Calculation:
  - Total Pages: Math.ceil(limit / pageSize)
  - Disable Next: (currentPage + 1) * pageSize >= limit
```

---

## Responsive Behavior

### Desktop (≥960px)
```
[Search] [Filters]
[Limit ▼] Showing X-Y of Z rows

Chart (50%)     Table (50%)
```

### Tablet (600-959px)
```
[Search] [Filters]
[Limit ▼] Showing X-Y of Z rows

Chart (100%)
Table (100%)
```

### Mobile (<600px)
```
[Search]
[Filters]
[Limit ▼]
Showing X-Y of Z

Chart (full width)
Table (full width, horizontal scroll)
```

---

## Color Scheme

```
Element              | Color Code | MUI Palette
---------------------|------------|------------------
Header Background    | #f5f5f5    | grey.100
Header Hover         | #eeeeee    | grey.200
Row Hover            | #fafafa    | grey.50
Sort Icon (active)   | primary    | primary.main
Button (disabled)    | gray       | action.disabled
Text (primary)       | black      | text.primary
Text (secondary)     | gray       | text.secondary
Text (caption)       | lightgray  | text.secondary (caption)
Divider              | lightgray  | divider
```

---

## Interactive States

### Search Box
```
Idle:      [🔍 Search or select column...]
Focused:   [🔍 |typed text____] 
With dropdown: [🔍 typed text]
             ├─ country
             ├─ amount
             └─ status
Empty/Clear: [🔍__________]
```

### Sort Icon
```
Inactive:  No icon
Active ↑:  Arrow pointing up (0° rotation)
Active ↓:  Arrow pointing down (180° rotation)
Hover:     Icon becomes more visible
```

### Buttons
```
Previous Button:
  ✓ Enabled:  [Previous] → clickable
  ✗ Disabled: [Previous] (grayed out)

Next Button:
  ✓ Enabled:  [Next] → clickable
  ✗ Disabled: [Next] (grayed out)
```

### Dropdown (Result Limit)
```
Closed:    [Result Limit ▼]
Open:      [Result Limit ▲]
           ├─ 100 rows ✓
           ├─ 1,000 rows
           └─ 10,000 rows
Selected:  [Result Limit ▼] shows "1000 rows"
```

---

## Animation & Transitions

### Sort Icon Rotation
```css
transition: transform 0.2s ease-in-out;
/* 0° for ascending */
/* 180° for descending */
```

### Table Row Hover
```css
transition: background-color 0.15s ease-in-out;
/* #ffffff → #fafafa */
```

### Column Header Hover
```css
transition: background-color 0.15s ease-in-out;
/* #f5f5f5 → #eeeeee */
```

### Pagination Button Disable
```css
/* Opacity reduces, cursor changes to not-allowed */
opacity: 0.5;
cursor: not-allowed;
```

---

## Spacing & Dimensions

### Spacing (Stack gaps)
```
Row 1 (Search): 2 units (16px)
Row 2 (Limit):  2 units (16px)
Between Rows:   2 units (16px)
```

### Dimensions
```
Search Box:     flex: 1, minWidth: 250px
Limit Dropdown: minWidth: 120px
Sort Icon:      fontSize: 16px
Row Height:     ~40px (table cell padding)
Table Max Ht:   440px
```

### Padding
```
Paper (overall): p: 2 (16px)
Table Cell:     Size="small" (8px)
Stack:          Various (see spacing)
```

---

## Text & Typography

### Labels
```
"Result Limit"      | InputLabel (gray, caption style)
"Showing X-Y of Z"  | Typography variant="body2"
"Page X of Y"       | Typography variant="caption"
Column Headers      | fontWeight: 600, fontSize: default
```

### Placeholder Text
```
Search: "Search results or select column..."
Dropdown: Default (no placeholder needed)
```

### Status Messages
```
No Results:    "No results match your search criteria."
               (Typography variant="body2", centered, gray)

Pagination:    "Page 1 of 50"
               (Typography variant="caption", secondary text)
```

---

## Accessibility

### Keyboard Navigation
```
Tab:       Navigate through elements (search → filters → limit → prev/next)
Enter:     Activate buttons or apply search
Escape:    Close dropdown/dialog
Arrow Keys: Navigate dropdown options (native)
```

### ARIA Labels
```
<Button aria-label="Previous page">Previous</Button>
<Button aria-label="Next page">Next</Button>
<Autocomplete inputId="search-box" />
```

### Screen Reader Text
```
Sort Icon: "Sorted by ID ascending"
           (via aria-sort attribute)
```

---

## Performance Considerations

### Rendering
- Table rows: Efficient with key={index}
- Autocomplete: Lightweight (only column names)
- Sort icon: CSS rotation (GPU accelerated)

### Memory
- processedResults: O(pageSize) = 100 rows in DOM
- Full results: O(limit) = up to 10k rows in state
- No memory leaks with proper cleanup

### Network
- Search: Client-side (no API calls)
- Sort: Client-side (no API calls)
- Filter: Requires new query execution
- Pagination: Client-side slicing (no API calls)

---

## Browser DevTools Tips

### Debug Search
```javascript
// In console:
searchText  // Shows current search text
processedResults.length  // Shows filtered count
results.length  // Shows total count
```

### Debug Sort
```javascript
sortColumn  // Shows active sort column
sortDirection  // Shows 'asc' or 'desc'
```

### Debug Pagination
```javascript
currentPage  // Current page (0-indexed)
pageSize  // Rows per page (100)
limit  // Total row limit
Math.ceil(limit / pageSize)  // Total pages
```

---

## Known Limitations

1. **No Multi-Column Sort**
   - Only one column can be sorted at a time
   - Future: Ctrl+Click for secondary sort

2. **Client-Side Processing**
   - All processing happens in browser
   - Large datasets (100k+) will be slow
   - Future: Server-side processing option

3. **No Virtual Scrolling**
   - Table shows all rows in page
   - 100 rows is reasonable limit
   - Future: Virtual scrolling for performance

4. **Fixed Page Size**
   - Page size locked at 100 rows
   - Future: User-configurable page size

---

## Future UI Enhancements

```
1. Column Visibility Toggle
   □ Eye icon to hide/show columns
   □ Checkbox list of columns

2. Data Type Icons
   □ # for numbers
   □ "A" for strings
   □ 📅 for dates

3. Filter Presets
   □ Save current filters
   □ Load saved presets
   □ Share with team

4. Export Button
   □ Download as CSV
   □ Download as JSON
   □ Email results

5. Statistics Footer
   □ Sum/Avg/Min/Max in footer
   □ Per column aggregations

6. Column Resizing
   □ Drag column borders
   □ Save column widths

7. Row Selection
   □ Checkbox column
   □ Bulk select actions
```

---

## Summary

The Query Builder Results UI combines:
- **Intuitive search** with column suggestions
- **One-click sorting** with visual indicators
- **Smart filtering** with conditional logic
- **Efficient pagination** for large datasets
- **Result limits** to control data volume
- **Responsive design** for all screen sizes
- **Accessibility** for all users

All features are production-ready and optimized for typical usage patterns.
