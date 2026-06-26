# Validation Rules UI: Lazy Loading & Faceted Search Implementation Guide

## Overview

This guide provides complete implementation details for adding lazy loading and faceted search (sidebar filters) to the Validation Rules feature.

**Current Date**: October 20, 2025  
**Phase**: 6.4 (Post-Deployment Monitoring)

## What's Implemented

### 1. Frontend Components ✅
- **Location**: `/frontend/src/components/ValidationRules/ValidationRulesWithFacets.tsx`
- **Styles**: `/frontend/src/components/ValidationRules/ValidationRulesWithFacets.css`
- **Features**:
  - Left sidebar with collapsible facets (Entity, Sub-Entity, Rule Type, Severity)
  - Lazy loading with 20-item pagination
  - Search bar integrated with facet filters
  - Real-time facet count updates as filters change
  - "Load More" button showing remaining items
  - Responsive design (mobile-friendly)

### 2. Backend API Handler ✅
- **Location**: `/backend/internal/handlers/validation_rules_list.go`
- **Endpoint**: `GET /api/validation-rules`
- **Features**:
  - Pagination (page, limit parameters)
  - Multi-filter support (entities, sub_entities, rule_types, severities)
  - Full-text search on rule name and description
  - Facet count calculation
  - Tenant-scoped queries (required tenant_id, datasource_id)

### 3. Updated UI Mockups ✅
- **Location**: `/VALIDATION_RULES_UI_MOCKUPS.md`
- **Sections Added**:
  - Visual mockup of new layout with sidebar facets
  - Lazy loading behavior documentation
  - API endpoint specification
  - Database query optimization guide
  - Performance considerations

## Key Features

### Lazy Loading
- **Default Page Size**: 20 rules per load
- **Maximum Page Size**: 100 rules (configurable)
- **Trigger**: Click "Load More" button or scroll to bottom
- **Performance**: Loads incrementally to handle 1,600+ rules efficiently

### Faceted Search
- **Left Sidebar Location**: Fixed width (240px), scrollable
- **Facet Types**:
  1. **Entities**: Filter by target entity (Order, Customer, Product, etc.)
  2. **Sub-Entity Types**: Filter by related entities (OrderItems, LineItems, etc.)
  3. **Rule Types**: Business Logic, Field Format, Cardinality, Referential, Uniqueness
  4. **Severity**: Error, Warning, Info

- **Multiple Selection**: Users can select multiple facet values
- **Live Updates**: Facet counts update as other filters change
- **Clear All**: Single button to reset all filters

### Search Integration
- **Scope**: Searches rule name and description fields
- **Type**: Full-text search with PostgreSQL `to_tsvector`
- **Debounce**: 300ms to reduce API calls
- **Combination**: Works with facet filters (AND logic across categories, OR within)

## Query Parameters

### API Request Example
```
GET /api/validation-rules?
  page=1&
  limit=20&
  entities=Order,Customer&
  sub_entities=Order.Items&
  rule_types=business_logic,field_format&
  severities=error&
  search=discount&
  tenant_id=<TENANT_ID>&
  datasource_id=<DATASOURCE_ID>
```

### Response Structure
```json
{
  "rules": [
    {
      "id": "rule-123",
      "rule_name": "Order Total Positive",
      "rule_type": "business_logic",
      "target_entity": "Order",
      "sub_entity_type": "Order.Items",
      "severity": "error",
      "description": "Order total must be positive",
      "condition": {...},
      "is_active": true,
      "created_at": "2025-10-19T10:30:00Z"
    }
  ],
  "total": 245,
  "page": 1,
  "limit": 20,
  "has_more": true,
  "entity_facets": [
    {"value": "Order", "label": "Order", "count": 45},
    {"value": "Customer", "label": "Customer", "count": 38}
  ],
  "sub_entity_facets": [...],
  "rule_type_facets": [...],
  "severity_facets": [...]
}
```

## Database Indexes

Add these indexes to improve query performance:

```sql
-- Individual facet queries
CREATE INDEX idx_rules_entity ON validation_rules(target_entity);
CREATE INDEX idx_rules_sub_entity ON validation_rules(sub_entity_type);
CREATE INDEX idx_rules_type ON validation_rules(rule_type);
CREATE INDEX idx_rules_severity ON validation_rules(severity);
CREATE INDEX idx_rules_active ON validation_rules(is_active);

-- Composite index for common combinations
CREATE INDEX idx_rules_entity_type_active 
  ON validation_rules(target_entity, rule_type, is_active);

-- Full-text search index
CREATE INDEX idx_rules_name_search 
  ON validation_rules USING GIN(to_tsvector('english', rule_name || ' ' || description));

-- Tenant scope optimization
CREATE INDEX idx_rules_tenant_scope 
  ON validation_rules(tenant_id, datasource_id);
```

## Integration Steps

### Step 1: Replace Existing Component
```typescript
// In your page that uses validation rules:
import ValidationRulesWithFacets from '@/components/ValidationRules/ValidationRulesWithFacets';

// Replace old component with:
<ValidationRulesWithFacets 
  tenantId={selectedTenant.id}
  datasourceId={selectedDatasource.id}
/>
```

### Step 2: Update Backend Route
```go
// In your router setup:
router.HandleFunc("/api/validation-rules", ListValidationRulesHandler(db)).Methods("GET")
```

### Step 3: Add Database Indexes
```bash
# Connect to database and run:
psql postgres://user:pass@host/alpha < indexes.sql
```

### Step 4: Test Filters
1. Create multiple rules with different entities, types, and severities
2. Select facet options and verify counts update
3. Test search with partial names/descriptions
4. Verify lazy loading loads additional batches

## Performance Metrics

### Load Time Targets
- **Initial load (20 rules)**: < 200ms
- **Facet counts**: < 100ms
- **Search with debounce**: < 300ms
- **Load more click**: < 150ms

### Expected Query Performance
With proper indexing:
- Facet query: ~0.5ms (1,600 rules)
- Paginated rules query: ~2-3ms
- Full-text search: ~5-10ms
- Combined (all filters + search): ~15-20ms

### API Response Size
- Single rule: ~200-300 bytes
- 20 rules: ~4-6 KB
- Facets metadata: ~2-3 KB
- Total response: ~6-10 KB per page

## Browser Compatibility

- **Chrome/Edge**: ✅ Full support
- **Firefox**: ✅ Full support
- **Safari**: ✅ Full support
- **Mobile**: ✅ Responsive design with horizontal scroll for facets

## Troubleshooting

### Issue: Facet counts are incorrect
**Solution**: Ensure indexes are created and verify filter logic in `getFacets()` function

### Issue: Search returns no results
**Solution**: Check PostgreSQL full-text search is enabled and index exists

### Issue: Lazy loading not working
**Solution**: Verify `has_more` flag in response; check pagination offset calculation

### Issue: Slow initial load
**Solution**: Add database indexes; reduce default page size; enable query caching

## Future Enhancements

1. **Facet Persistence**: Save filter state to localStorage
2. **Facet Presets**: Save/load common filter combinations
3. **Advanced Search**: Query builder for complex conditions
4. **Export**: Download filtered rules as CSV/JSON
5. **Bulk Actions**: Select multiple rules for bulk edit/delete
6. **Sort Options**: Sort by name, date, severity, etc.
7. **Analytics**: Track which facets are used most

## Monitoring in Phase 6.4

This feature rollout is being monitored during Phase 6.4 (Oct 19-26, 2025):

- **Day 1 Baseline**: Query latency 23ms (target: 22-25ms) ✅
- **Daily Tests**: Validation rules accessibility verified
- **Load Testing**: Concurrent request handling (240+ req/sec)
- **Multi-Entity**: 1,609 rules with entities verified

The faceted search implementation should maintain these performance targets.

## Contact & Support

For questions or issues with implementation, refer to:
- Backend logic: `/backend/internal/handlers/validation_rules_list.go`
- Frontend component: `/frontend/src/components/ValidationRules/`
- UI specs: `/VALIDATION_RULES_UI_MOCKUPS.md`

---

**Status**: Ready for implementation  
**Last Updated**: October 20, 2025  
**Project Phase**: 6.4 - Post-Deployment Monitoring
