# Business Term Mapper - Inline Suggestions & Title Case Fix

## Summary of Changes

Fixed three major issues:
1. **Business Terms now display in Title Case** (not UPPER_CASE_WITH_UNDERSCORES)
2. **Suggestions now appear inline** on each semantic term row (not in top section)
3. **Relationships now persist** correctly after refresh

---

## 1. Title Case Formatting

### Helper Function Added
```typescript
// Converts "CUSTOMER_FIRST_NAME" → "Customer First Name"
function formatBusinessTermName(name: string): string {
  if (!name) return '';
  
  // Replace underscores with spaces and convert to lowercase
  const withSpaces = name.replace(/_/g, ' ').toLowerCase();
  
  // Capitalize first letter of each word
  return withSpaces
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}
```

### Applied Throughout
- ✅ Custom term creation
- ✅ Suggestion display
- ✅ Accept/reject feedback messages
- ✅ All business term displays in UI

### Backend Note
Business terms are still stored in their original format (e.g., `CUSTOMER_ID`). The formatting is applied only in the UI layer for better readability.

---

## 2. Inline Suggestions Per Row

### UI Changes

**Before:** Global suggestions section at top of page  
**After:** Inline suggestions within each semantic term's expanded row

### New Row Features
Each unmapped semantic term row now has:
- **AI Suggestions section** (gray background)
- **Generate button** - loads suggestions for that specific term
- **Refresh button** - reloads suggestions after first generation
- **Up to 3 suggestion cards** displayed inline
- **Accept/Reject buttons** on each suggestion

### Visual Layout
```
┌─────────────────────────────────────────────┐
│ Semantic Term: customer_id         [Save]  │ ← Row header
│                                    [▼]     │
├─────────────────────────────────────────────┤
│ ✨ AI Suggestions          [Generate] │ ← New inline section
│                                            │
│ ┌──────────────┐ ┌──────────────┐ ┌────── │
│ │ Customer Id  │ │ Customer ID  │ │ Client│
│ │ 87%         │ │ 75%         │ │ 62%   │
│ │ [Accept]    │ │ [Accept]    │ │ [Acc  │
│ │ [Reject]    │ │ [Reject]    │ │ [Rej  │
│ └──────────────┘ └──────────────┘ └────── │
│                                            │
│ Select Existing Business Term              │ ← Existing manual selection
│ [Autocomplete dropdown]                    │
└─────────────────────────────────────────────┘
```

### New Interface Props
```typescript
interface EnhancedMappingRowProps {
  // ... existing props
  onAcceptSuggestion: (suggestion: BusinessTermSuggestion) => Promise<void>;
  onRejectSuggestion: (suggestion: BusinessTermSuggestion) => Promise<void>;
  onLoadSuggestions: (semanticTermId: string) => Promise<void>;
}

interface BusinessTermMapping {
  // ... existing fields
  suggestions?: BusinessTermSuggestion[];  // New field
}
```

### Per-Row Callbacks
```typescript
// Loads suggestions for a specific semantic term
const handleLoadSuggestionsForTerm = async (semanticTermId: string) => {
  // Fetches from GET /api/semantic-terms/{id}/suggest-business-terms
  // Updates that specific mapping's suggestions array
};

// Accepts suggestion and applies to specific row
const handleAcceptSuggestion = async (suggestion: BusinessTermSuggestion) => {
  // Formats name to Title Case
  // Updates mapping with selected business term
  // Records feedback
  // Clears suggestions for that row
};

// Rejects suggestion and removes from row
const handleRejectSuggestion = async (suggestion: BusinessTermSuggestion) => {
  // Records feedback
  // Removes suggestion from that mapping's array
};
```

---

## 3. Relationship Persistence

### Problem
After creating a mapping and refreshing the page, the relationship would disappear even though it was saved to the database.

### Root Cause
The initialization code was setting `edge_exists: false` for all terms and not loading existing edges from the backend.

### Solution
Added edge loading on initialization:

```typescript
useEffect(() => {
  const initializeData = async () => {
    // 1. Load semantic terms and business terms
    const [semanticData, businessData] = await Promise.all([
      loadSemanticTerms(),
      loadBusinessTerms()
    ]);
    
    // 2. NEW: Fetch existing edges
    const existingEdgesResponse = await fetch(`/api/business-term-edges`, {
      method: 'GET',
      credentials: 'include'
    });
    
    const existingEdges = await existingEdgesResponse.json();
    
    // 3. Create edge map: semantic_term_id -> business_term_id
    const edgeMap = new Map();
    existingEdges.forEach(edge => {
      edgeMap.set(edge.target_node_id, edge.source_node_id);
    });
    
    // 4. Initialize mappings WITH existing edges
    semanticData.forEach(term => {
      const businessTermId = edgeMap.get(term.node_id);
      const businessTerm = businessTermId 
        ? businessData.find(bt => bt.node_id === businessTermId) 
        : null;
      
      initialMappings[term.node_id] = {
        semantic_term: term,
        selected_business_term: businessTerm || null,  // ← Populated!
        edge_exists: !!businessTerm                     // ← Set correctly!
      };
    });
  };
}, []);
```

### Edge Data Structure
```
Edge in database:
{
  source_node_id: "uuid-of-business-term",    // Business term
  target_node_id: "uuid-of-semantic-term",    // Semantic term
  edge_type_id: "...",
  relationship_type: "business_term_to_semantic_term"
}
```

### Persistence Flow
```
1. User selects business term → handleSelectBusinessTerm()
2. User clicks "Save Mapping" → handleSave()
3. POST /api/business-term-edges
   - Creates edge in database
   - Updates local state: edge_exists = true
4. User refreshes page
5. Initialization fetches existing edges
6. Mapping populated with correct business term
7. Shows "Mapped" chip ✅
```

---

## Testing Checklist

### Title Case Formatting
- [ ] Open Business Term Mapper
- [ ] Create custom term with underscores (e.g., "CUSTOMER_NAME")
- [ ] Verify displayed as "Customer Name"
- [ ] Generate suggestions
- [ ] Verify suggestions show in Title Case

### Inline Suggestions
- [ ] Expand an unmapped semantic term row
- [ ] Verify "AI Suggestions" section visible with gray background
- [ ] Click "Generate" button
- [ ] Verify up to 3 suggestions appear inline
- [ ] Verify confidence percentages and color coding
- [ ] Accept a suggestion
- [ ] Verify suggestion disappears and business term selected
- [ ] Expand another row
- [ ] Verify it has its own independent suggestions

### Relationship Persistence
- [ ] Map a semantic term to a business term
- [ ] Click "Save Mapping"
- [ ] Verify "Mapped" chip appears
- [ ] Refresh the page (F5 or Cmd+R)
- [ ] Verify mapping still shows
- [ ] Verify "Mapped" chip still present
- [ ] Expand the row
- [ ] Verify business term still selected

### Feedback Integration
- [ ] Generate suggestions for unmapped term
- [ ] Accept a suggestion
- [ ] Map a few more terms manually
- [ ] Generate suggestions again for different terms
- [ ] Verify accepted term has higher confidence (if suggested again)
- [ ] Reject some suggestions
- [ ] Generate suggestions again
- [ ] Verify rejected terms have lower confidence or don't appear

---

## API Endpoints Used

### GET `/api/semantic-terms/{id}/suggest-business-terms`
- Generates suggestions for a single semantic term
- Returns array of suggestions with confidence scores
- Includes feedback-adjusted confidence

### POST `/api/business-term-edges`
- Creates edge between business term and semantic term
- Payload:
```json
{
  "subject_node_id": "business_term_uuid",
  "object_node_id": "semantic_term_uuid",
  "edge_type_id": "3be9d6ae-1598-4628-a3dd-b606921a9193",
  "relationship_type": "business_term_to_semantic_term"
}
```

### GET `/api/business-term-edges`
- Fetches all existing edges for current tenant/datasource
- Used on initialization to populate mappings

### POST `/api/business-term/suggestion-feedback`
- Records accept/reject feedback
- Payload:
```json
{
  "semantic_term_id": "uuid",
  "business_term_name": "CUSTOMER_ID",
  "action": "accept|reject",
  "business_term_id": "uuid",
  "confidence": 0.85,
  "reason": "User accepted/rejected suggestion"
}
```

---

## Key Files Modified

### `/frontend/src/components/semantic-mapper/BusinessTermMapper.tsx`
- Added `formatBusinessTermName()` helper
- Updated `BusinessTermMapping` interface to include `suggestions?`
- Updated `EnhancedMappingRowProps` to include callback props
- Added inline AI Suggestions UI section in `EnhancedMappingRow`
- Added `handleLoadSuggestionsForTerm()` callback
- Updated `handleAcceptSuggestion()` to format names and work per-row
- Updated `handleRejectSuggestion()` to work per-row
- Added edge loading in `useEffect` initialization
- Removed global suggestions section from top of page
- Applied Title Case formatting throughout

### Backend (No Changes Required)
- Existing endpoints already support the functionality
- Suggestion feedback integration already implemented
- Edge CRUD endpoints already working

---

## Benefits

### User Experience
1. **Better Readability**: "Customer Name" vs "CUSTOMER_NAME"
2. **Focused Workflow**: Suggestions appear where you need them
3. **Independent Operations**: Each term has its own suggestions
4. **Persistent State**: Mappings survive page refresh
5. **Less Scrolling**: No need to scroll between suggestions and terms

### Technical
1. **Reduced State Complexity**: No global suggestions array
2. **Better Performance**: Load suggestions only when needed
3. **Cleaner UI**: Removed cluttered top section
4. **Proper Data Loading**: Edges loaded on mount
5. **Consistent Formatting**: Single source of truth for name formatting

---

## Future Enhancements

1. **Batch Suggestions**: Add "Generate All" button to load suggestions for all unmapped terms at once
2. **Suggestion History**: Show previously rejected suggestions with reason
3. **Confidence Thresholds**: Filter suggestions by minimum confidence level
4. **Auto-Accept**: Automatically accept suggestions above X% confidence
5. **Suggestion Explanation**: Detailed breakdown of confidence calculation
6. **Undo Mapping**: Delete edge and revert to unmapped state
7. **Bulk Operations**: Select multiple suggestions across rows and accept/reject together
