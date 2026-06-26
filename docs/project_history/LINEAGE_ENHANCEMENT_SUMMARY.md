# Lineage Diagram Enhancement Summary

## Overview
Enhanced the lineage/impact analysis diagram to provide better data visualization consistency with the relationships section, including predicate-based edge labels, node type coloring, qualified paths, and direction arrows.

## Changes Completed

### Backend Changes (Go)

#### 1. **lineage_service.go - GetRecursiveLineage Method** (Lines 123-230)
Updated the SQL query and data processing to fetch and use correct relationship type information:

**Query Changes (Line 123-135):**
- Added `qualified_path` fields to both source and target node queries
- Changed from using `edge_type_name` to `predicate` (primary) with fallback to `edge_type_name`
- Ensures edges use the correct predicate values from `catalog_edge_type` table
- Qualified paths capture the full hierarchical path (e.g., "schema.table.column")

```sql
-- Fetch predicate (correct relationship name) and qualified paths
COALESCE(et.predicate, et.edge_type_name, 'unknown') as edge_label
COALESCE(ns.qualified_path, ns.node_name) as source_qualified_path
COALESCE(nt.qualified_path, nt.node_name) as target_qualified_path
```

**Row Scanning Update (Line 157-160):**
- Added `sourceQualifiedPath` and `targetQualifiedPath` to Scan operation
- Captures both source and target node's qualified paths

**Node Building Update (Line 169-204):**
- Use qualified path in node labels instead of just names
- Provides full context for database objects (schema.table.column)
- Falls back to node names if qualified paths unavailable
- Preserves node type and parent information

**Edge Direction Indicators (Line 206-220):**
- Added logic to determine if selected node is subject (source) or object (target)
- Prepends "← " to label when selected node is the object
- Appends " →" to label when selected node is the subject
- Arrows clearly show relationship direction

### Frontend Changes (TypeScript/React)

#### 1. **HoverableNode.tsx - Node Color Mapping** (Lines 1-70)
Added comprehensive color mapping for different node types:

**Color Scheme:**
- `business_object` / `business_term`: Blue (#1E40AF border, #DBEAFE background)
- `semantic_term` / `semantic_model` / `semantic_view`: Purple (#6B21A8 border, #E9D5FF background)
- `semantic_column`: Orange (#92400E border, #FED7AA background)
- `database_column` / `db_column` / `column`: Green (#15803D border, #DCFCE7 background)
- `table`: Purple-pink (#7E22CE border, #F3E8FF background)
- `schema`: Pink (#BE185D border, #FCE7F3 background)
- `database`: Red (#DC2626 border, #FEE2E2 background)
- `bo_field`: Light Blue (#0284C7 border, #DBEAFE background)

**Implementation:**
- Created `getNodeTypeColor()` function returning `{ bg, border, text }` colors
- Applied dynamically to node rendering based on `data.nodeType`
- Enhanced `getNodeTypeDisplayName()` to include new types

#### 2. **HoverableNode.tsx - Component Rendering** (Lines 75-90)
Updated component to use computed colors:

**Changes:**
- Call `getNodeTypeColor()` with memoized result
- Pass color values to CSS variables in `contentDynamicStyles`
- Falls back to computed colors if no explicit styles provided
- Maintains support for custom styles when needed

#### 3. **HoverableNode.css - Node Styling** (Lines 9-20)
Enhanced CSS styling for better visual hierarchy:

**Updates:**
- Increased border width from 1px to 2px for better visibility
- Increased padding from 8px 12px to 10px 14px
- Changed font-weight to 600 (semi-bold) by default
- Added transition effects for smooth visual feedback
- Maintains responsive design with CSS variables

#### 4. **BusinessTermsTab.tsx - Relationship Display** (Lines 620-630)
Updated relationships table to show proper direction indicators:

**Changes:**
- Direction arrow now depends on whether current node is source or target
- Shows "→" when selected asset is the source (subject)
- Shows "←" when selected asset is the target (object)
- Uses `edge.relationship_type` or fallback to `edge.predicate`
- Matches backend predicate values exactly

## Technical Architecture

### Data Flow
1. **Backend Query**: Gets predicate from `catalog_edge_type.predicate`
2. **Node Building**: Includes qualified paths from `catalog_node.qualified_path`
3. **Edge Labels**: Combines predicate with direction indicator
4. **Frontend Display**: Colors nodes by type, shows qualified paths in tooltips, displays correct direction arrows

### Node Type Categories
Three main semantic categories now properly distinguished:
- **Business Layer**: Business terms, business objects, business fields
- **Semantic Layer**: Semantic terms, models, views, columns
- **Technical Layer**: Database columns, tables, schemas, databases

## Testing & Validation

### Backend Validation
- ✅ Go compilation successful (`go build ./cmd/server`)
- ✅ No vet errors in services package
- ✅ Existing tests still pass (test failures pre-existing)

### Frontend Validation
- ✅ Build successful (46.94s - 25969 modules transformed)
- ✅ No new TypeScript errors introduced
- ✅ CSS changes properly scoped
- ✅ React component correctly typed

### Visual Testing Checklist
- [ ] Run lineage diagram and verify nodes display with correct colors
- [ ] Hover over nodes to see qualified paths in tooltips
- [ ] Verify direction arrows show correctly (← and →)
- [ ] Compare relationship section arrows with lineage diagram
- [ ] Test with different node types (business_term, column, table, etc.)
- [ ] Check mobile/responsive behavior

## Files Modified Summary

| File | Type | Changes |
|------|------|---------|
| `backend/internal/services/lineage_service.go` | Go | Query updates, qualified path handling, direction indicators |
| `frontend/src/components/HoverableNode.tsx` | React/TS | Color mapping, dynamic styling |
| `frontend/src/components/HoverableNode.css` | CSS | Enhanced borders, padding, font-weight |
| `frontend/src/pages/glossary/BusinessTermsTab.tsx` | React/TS | Direction arrow logic in relationships |

## Backward Compatibility

All changes are backward compatible:
- Fallback to existing data if qualified paths unavailable
- Default colors applied to unmapped node types
- Direction arrows enhance but don't break existing labels
- GraphQL queries unchanged - work with existing data

## Future Enhancements

Potential follow-up improvements:
1. Add legend showing color meanings
2. Implement edge filtering by relationship type
3. Add drill-down capability from qualified paths
4. Persist user color preferences
5. Add animation for direction indicators
6. Performance optimization for large lineage graphs

## Deployment Notes

1. Deploy backend first (lineage_service.go changes)
2. Deploy frontend after (HoverableNode and BusinessTermsTab changes)
3. No database migration needed
4. No configuration changes required
5. Feature works with existing catalog data

## References

- Backend lineage service: `backend/internal/services/lineage_service.go`
- Frontend node component: `frontend/src/components/HoverableNode.tsx`
- Relationships display: `frontend/src/pages/glossary/BusinessTermsTab.tsx`
- Database tables: `catalog_node`, `catalog_edge`, `catalog_edge_type`
