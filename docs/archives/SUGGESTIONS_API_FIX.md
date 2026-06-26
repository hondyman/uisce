# Business Term Suggestions API Fix

## Problem
The frontend was calling the wrong endpoint with incorrect payload format, resulting in a 400 error:
```
POST http://localhost:8080/api/business-term/suggestions
Error: "At least one semantic term is required"
```

The old code was trying to send:
```json
{ "semantic_term_id": "single-uuid" }
```

But the `/business-term/suggestions` endpoint expects:
```json
{
  "semantic_terms": ["array", "of", "names"],
  "database_columns": ["array", "of", "column", "names"],
  "limit": 10
}
```

## Solution
Updated the frontend to use the **correct endpoint** that leverages our enhanced `SemanticMappingService` with feedback integration:

### Correct Endpoint
```
GET /api/semantic-terms/{id}/suggest-business-terms
```

This endpoint:
- ✅ Takes a single semantic term ID as URL parameter
- ✅ Uses `SemanticMappingSvc.SuggestBusinessTerms()` (our enhanced service with feedback)
- ✅ Returns array of suggestions with confidence scores
- ✅ Automatically applies historical feedback adjustments
- ✅ Includes tenant scoping via headers

### Frontend Changes
**File:** `/frontend/src/components/semantic-mapper/BusinessTermMapper.tsx`

**Before:**
```typescript
// WRONG - used POST with wrong payload
const response = await fetch(`http://localhost:8080/api/business-term/suggestions`, {
  method: 'POST',
  body: JSON.stringify({ semantic_term_id: semanticTermId })
});
```

**After:**
```typescript
// CORRECT - uses GET with semantic term ID in URL
const response = await fetch(
  `http://localhost:8080/api/semantic-terms/${term.node_id}/suggest-business-terms`, 
  {
    method: 'GET',
    credentials: 'include'
  }
);
```

### Key Improvements
1. **Parallel Requests**: Now makes concurrent requests for all unmapped terms using `Promise.all()`
2. **Better Error Handling**: Logs specific errors per term, continues if one fails
3. **Better UX**: Shows progress messages and helpful info when no suggestions found
4. **Correct Service**: Uses the service that has feedback integration built-in

### Request Flow
```
1. User clicks "Generate Suggestions" button
2. Frontend finds all unmapped semantic terms
3. For each term, makes parallel GET request:
   GET /api/semantic-terms/{term_id}/suggest-business-terms
4. Backend calls SemanticMappingService.SuggestBusinessTerms()
   - Fetches semantic term details
   - Fetches all business terms
   - Fetches historical feedback stats
   - Calculates confidence with feedback adjustment
   - Returns sorted suggestions
5. Frontend aggregates all suggestions
6. Displays in suggestion cards grid
```

### Expected Response Format
```json
[
  {
    "business_term_id": "uuid",
    "term_name": "CUSTOMER_NAME",
    "confidence": 0.87,
    "reason": "Strong name similarity match. Users frequently accept this term (85.0% acceptance).",
    "source": "INTERNAL_SIMILARITY",
    "confidence_breakdown": [
      {
        "label": "Name Similarity",
        "score": 0.85,
        "weight": 0.7
      },
      {
        "label": "Historical Feedback",
        "score": 0.85,
        "weight": 0.3,
        "details": "17 accepts, 3 rejects (85.0% acceptance)"
      }
    ]
  }
]
```

## Testing
1. Start backend: `docker compose up -d`
2. Start frontend: `cd frontend && npm run dev`
3. Navigate to Business Term Mapper
4. Click "Generate Suggestions"
5. Should see suggestions appear (if business terms exist)
6. Accept/reject suggestions to build feedback history
7. Generate suggestions again to see confidence adjustments

## Endpoints Summary

### ✅ USE THIS (Correct)
```
GET /api/semantic-terms/{id}/suggest-business-terms
```
- Uses enhanced SemanticMappingService
- Includes feedback adjustments
- Properly tenant-scoped
- Returns confidence breakdown

### ❌ DON'T USE (Wrong for this use case)
```
POST /api/business-term/suggestions
```
- Expects semantic_terms + database_columns arrays
- Uses simple name matching algorithm
- No feedback integration
- Different use case (bulk column mapping)

## Related Files
- `/backend/internal/api/api.go` - Endpoint definition (line ~920)
- `/backend/internal/services/semantic_mapping_service.go` - Enhanced SuggestBusinessTerms method
- `/frontend/src/components/semantic-mapper/BusinessTermMapper.tsx` - Fixed generateSuggestions function
- `/backend/migrations/000027_create_suggestion_feedback.sql` - Feedback table schema
