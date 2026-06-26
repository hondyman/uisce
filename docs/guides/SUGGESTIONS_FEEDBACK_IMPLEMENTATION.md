# Business Term Suggestions with ML Feedback Implementation

## Overview
This document describes the implementation of AI-powered business term suggestions with user feedback integration for continuous learning.

## Database Schema

### Migration: `000027_create_suggestion_feedback.sql`

**Table: `suggestion_feedback`**
```sql
CREATE TABLE public.suggestion_feedback (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    semantic_term_id uuid NOT NULL,
    business_term_id uuid,
    business_term_name text NOT NULL,
    action text NOT NULL CHECK (action IN ('accept', 'reject')),
    confidence numeric(5, 4),
    reason text,
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    CONSTRAINT fk_semantic_term FOREIGN KEY (semantic_term_id) 
        REFERENCES public.catalog_node(id) ON DELETE CASCADE,
    CONSTRAINT fk_business_term FOREIGN KEY (business_term_id) 
        REFERENCES public.catalog_node(id) ON DELETE SET NULL
);
```

**View: `suggestion_feedback_stats`**
```sql
CREATE VIEW public.suggestion_feedback_stats AS
SELECT
    business_term_name,
    COUNT(*) FILTER (WHERE action = 'accept') as accept_count,
    COUNT(*) FILTER (WHERE action = 'reject') as reject_count,
    COUNT(*) as total_feedback,
    ROUND(COUNT(*) FILTER (WHERE action = 'accept')::numeric / 
          NULLIF(COUNT(*), 0) * 100, 2) as acceptance_rate,
    AVG(confidence) FILTER (WHERE action = 'accept') as avg_confidence_accepted,
    AVG(confidence) FILTER (WHERE action = 'reject') as avg_confidence_rejected,
    array_agg(DISTINCT reason) FILTER (WHERE reason IS NOT NULL) as rejection_reasons
FROM public.suggestion_feedback
GROUP BY business_term_name
ORDER BY total_feedback DESC;
```

## Backend Implementation

### API Endpoints

**POST `/api/business-term/suggestions`**
- Generates AI suggestions for a semantic term
- Request body: `{ "semantic_term_id": "uuid" }`
- Returns array of suggestions with confidence scores

**POST `/api/business-term/suggestion-feedback`**
- Records user feedback on suggestions
- Request body:
```json
{
  "semantic_term_id": "uuid",
  "business_term_id": "uuid",
  "business_term_name": "string",
  "action": "accept|reject",
  "confidence": 0.85,
  "reason": "optional reason"
}
```

### Service: `SemanticMappingService`

**New Method: `fetchFeedbackStats`**
- Queries `suggestion_feedback` table for historical acceptance/rejection rates
- Returns map of business term names to `FeedbackStats`
- Gracefully handles missing table (returns empty map)

**Enhanced Method: `SuggestBusinessTerms`**
- Fetches feedback statistics for all business terms
- Applies feedback-based confidence adjustment:
  - Only applied when ≥3 feedback data points exist
  - Max adjustment: ±15% (0.3 weight × (acceptance_rate - 0.5))
  - Clamps final confidence between 0.0 and 1.0
- Adds "Historical Feedback" to confidence breakdown
- Appends feedback info to suggestion reason

**Confidence Adjustment Formula:**
```
adjusted_confidence = base_confidence + ((acceptance_rate - 0.5) * 0.3)
```

Examples:
- 90% acceptance rate → +12% boost
- 50% acceptance rate → no change
- 20% acceptance rate → -9% penalty

### New Types

**`FeedbackStats`**
```go
type FeedbackStats struct {
    BusinessTermName string
    AcceptCount      int
    RejectCount      int
    TotalFeedback    int
    AcceptanceRate   float64
}
```

## Frontend Implementation

### Components

**BusinessTermMapper**
- Added state for suggestions and loading
- Added `generateSuggestions()` function
- Added `handleAcceptSuggestion()` callback
- Added `handleRejectSuggestion()` callback
- New UI section: "AI-Powered Business Term Suggestions"

### UI Features

**Generate Suggestions Button**
- Calls backend API for each unmapped semantic term
- Displays loading state with spinner
- Shows count of suggestions generated

**Suggestion Cards**
- Grid layout showing up to 9 suggestions
- Displays:
  - Semantic term name
  - Suggested business term name
  - Confidence percentage (color-coded chip)
  - Reason for suggestion
  - Accept/Reject buttons

**Accept/Reject Actions**
- Accept: Creates mapping + records feedback
- Reject: Records feedback + removes from list
- Both: Call `recordSuggestionFeedback()` API

### Color Coding
- Green (>80%): High confidence
- Blue (60-80%): Medium confidence  
- Orange (<60%): Low confidence

## Feedback Loop

### How It Works

1. **Initial Suggestion**: Backend generates suggestions using name similarity, abbreviation expansion, etc.
2. **User Action**: User accepts or rejects suggestion
3. **Feedback Recording**: Frontend calls `/suggestion-feedback` endpoint
4. **Data Storage**: Feedback stored in `suggestion_feedback` table
5. **Learning**: Next time suggestions are generated, historical acceptance rates adjust confidence scores
6. **Improvement**: Terms with high acceptance rates get boosted, rejected terms get penalized

### Benefits

- **Continuous Learning**: System improves over time based on user behavior
- **Tenant-Specific**: Feedback is scoped to tenant + datasource
- **Transparent**: Users see confidence breakdowns including feedback component
- **Fail-Safe**: If feedback table doesn't exist, suggestions still work without adjustment

## Testing Checklist

- [ ] Run migration `000027_create_suggestion_feedback.sql`
- [ ] Verify `suggestion_feedback` table created
- [ ] Verify `suggestion_feedback_stats` view created
- [ ] Test POST `/api/business-term/suggestions` endpoint
- [ ] Test POST `/api/business-term/suggestion-feedback` endpoint
- [ ] Verify suggestions UI renders
- [ ] Click "Generate Suggestions" button
- [ ] Accept a suggestion and verify:
  - Feedback recorded in database
  - Mapping created
  - Suggestion removed from list
- [ ] Reject a suggestion and verify:
  - Feedback recorded in database
  - Suggestion removed from list
- [ ] Accept same term multiple times across different semantic terms
- [ ] Generate suggestions again and verify confidence boost for accepted term
- [ ] Check `suggestion_feedback_stats` view shows correct aggregations

## Future Enhancements

1. **ML Model Integration**: Replace simple name matching with trained ML model
2. **Contextual Feedback**: Capture more detailed rejection reasons (dropdown)
3. **Confidence Thresholds**: User-configurable minimum confidence for suggestions
4. **Batch Operations**: Accept/reject multiple suggestions at once
5. **Suggestion Explanations**: Show detailed breakdown of why suggestion was made
6. **Historical Trends**: Dashboard showing acceptance rates over time
7. **Auto-Apply**: Automatically accept suggestions above X% confidence
8. **Rollback**: Undo accepted suggestions that were mistakes

## API Usage Examples

### Generate Suggestions
```bash
curl -X POST http://localhost:8080/api/business-term/suggestions \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ..." \
  -H "X-Tenant-Datasource-ID: ..." \
  -d '{"semantic_term_id": "123e4567-e89b-12d3-a456-426614174000"}'
```

### Record Feedback
```bash
curl -X POST http://localhost:8080/api/business-term/suggestion-feedback \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ..." \
  -H "X-Tenant-Datasource-ID: ..." \
  -d '{
    "semantic_term_id": "123e4567-e89b-12d3-a456-426614174000",
    "business_term_name": "CUSTOMER_NAME",
    "action": "accept",
    "confidence": 0.85
  }'
```

### Query Feedback Stats
```bash
psql -c "SELECT * FROM public.suggestion_feedback_stats WHERE total_feedback >= 3 ORDER BY acceptance_rate DESC LIMIT 10;"
```

## Database Queries

### Find Most Accepted Terms
```sql
SELECT 
    business_term_name, 
    accept_count, 
    total_feedback, 
    acceptance_rate
FROM public.suggestion_feedback_stats
WHERE total_feedback >= 5
ORDER BY acceptance_rate DESC
LIMIT 20;
```

### Find Most Rejected Terms
```sql
SELECT 
    business_term_name, 
    reject_count, 
    total_feedback, 
    acceptance_rate,
    rejection_reasons
FROM public.suggestion_feedback_stats
WHERE total_feedback >= 5
ORDER BY acceptance_rate ASC
LIMIT 20;
```

### Feedback by Tenant
```sql
SELECT 
    tenant_id,
    COUNT(*) as total_feedback,
    COUNT(*) FILTER (WHERE action = 'accept') as accepts,
    COUNT(*) FILTER (WHERE action = 'reject') as rejects,
    ROUND(COUNT(*) FILTER (WHERE action = 'accept')::numeric / COUNT(*) * 100, 1) as acceptance_pct
FROM public.suggestion_feedback
GROUP BY tenant_id
ORDER BY total_feedback DESC;
```
