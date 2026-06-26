# Unified Lineage & Impact Analysis

## Overview

The Unified Lineage Tab merges lineage (upstream dependencies) and impact (downstream dependencies) into a single, interactive view. This provides complete visibility into data flow in both directions.

## Features

### 🔀 Bidirectional Graph Traversal
- **Upstream (Lineage)**: Shows all sources that feed into the selected node
- **Downstream (Impact)**: Shows all targets affected by changes to the node
- **Both**: Combined view showing the complete dependency graph

### 🎨 Visual Indicators
- **Color-coded nodes** by type (Business Objects, Semantic Terms, DB Columns, etc.)
- **Direction metadata** embedded in nodes and edges
- **Dynamic filtering** by direction without re-fetching data
- **Statistics chips** showing upstream/downstream counts

### 🤖 AI-Powered Analysis
- **Smart Explanation**: Context-aware descriptions based on direction
  - Lineage mode: "These sources feed into..."
  - Impact mode: "Changes here will affect..."
  - Both mode: "This node connects..."
- **Interactive Q&A**: Ask questions about relationships, filtered by direction

### 📊 Enhanced Controls
- **Direction Toggle**: Three-button control (Upstream/Both/Downstream)
- **Floating Toolbar**: Graph, Explanation, AI Assistant, Sidebar toggle
- **MiniMap**: Optional overview for large graphs
- **Fullscreen Mode**: Dedicated view for complex lineages

## Component Usage

### UnifiedLineageTab

```tsx
import { UnifiedLineageTab } from '@/features/impact-analysis';

<UnifiedLineageTab
  nodeType="semantic_term"
  nodeId="customer_id"
  initialDirection="both" // 'upstream' | 'downstream' | 'both'
/>
```

### ImpactGraph (Updated)

```tsx
import { ImpactGraph } from '@/features/impact-analysis';

<ImpactGraph
  nodeType="table"
  nodeId="customers"
  directionMode="upstream"
  onStatsUpdate={(stats) => console.log(stats)}
  highlightedNodeIds={['node1', 'node2']}
/>
```

## Backend API

### Enhanced Bidirectional Endpoint

**GET** `/api/lineage/node/{id}/graph`

Returns graph with direction metadata:

```json
{
  "nodes": [
    {
      "id": "customer_table",
      "type": "table",
      "metadata": {
        "direction": "upstream",
        "is_lineage": true
      }
    },
    {
      "id": "report_view",
      "type": "view",
      "metadata": {
        "direction": "downstream",
        "is_impact": true
      }
    }
  ],
  "edges": [
    {
      "from_id": "customer_table",
      "to_id": "customer_view",
      "metadata": {
        "direction": "upstream"
      }
    }
  ]
}
```

### Direction Metadata

Each node/edge includes:
- `direction`: `"upstream"` | `"downstream"` | `"both"` | `"unknown"`
- `is_lineage`: `true` for upstream nodes
- `is_impact`: `true` for downstream nodes

Nodes appearing in both upstream and downstream paths are marked with `direction: "both"` and have both flags set.

## Architecture

### Backend (Go)
- **File**: `backend/internal/lineage/sql_repo.go`
- **Method**: `FindBiDirectionalGraph(ctx, rootID, depth)`
- **Implementation**:
  1. Calls `FindUpstreamGraph()` - recursive CTE for sources
  2. Calls `FindDownstreamGraph()` - recursive CTE for targets
  3. Merges results with direction metadata
  4. Deduplicates nodes appearing in both directions

### Frontend (React)
- **File**: `frontend/src/features/impact-analysis/components/UnifiedLineageTab.tsx`
- **Key Features**:
  - Direction toggle (ToggleButtonGroup)
  - Statistics chips (upstream/downstream counts)
  - Sidebar modes (explanation/assistant)
  - Floating controls

- **Graph Component**: `ImpactGraph.tsx`
  - Stores all nodes/edges in state
  - Filters by direction client-side
  - Recalculates layout on filter change
  - Updates statistics via callback

## Database Schema

Uses relational tables instead of graph database:

```sql
-- Catalog nodes (entities)
CREATE TABLE catalog_node (
  id UUID PRIMARY KEY,
  type TEXT,
  name TEXT,
  qualified_path TEXT
);

-- Catalog edges (relationships)
CREATE TABLE catalog_edge (
  id UUID PRIMARY KEY,
  from_id UUID REFERENCES catalog_node(id),
  to_id UUID REFERENCES catalog_node(id),
  type TEXT
);

-- Semantic lineage (denormalized for performance)
CREATE TABLE semantic.lineage_nodes (...);
CREATE TABLE semantic.lineage_edges (...);
```

### Recursive CTE Example

```sql
WITH RECURSIVE upstream AS (
  SELECT from_id AS id, 1 AS depth
  FROM semantic.lineage_edges
  WHERE to_id = $1

  UNION ALL

  SELECT e2.from_id AS id, d.depth + 1
  FROM semantic.lineage_edges e2
  JOIN upstream d ON e2.to_id = d.id
  WHERE d.depth < $2
)
SELECT * FROM upstream;
```

## Migration from Separate Tabs

### Before (Two Separate Tabs)
- **Lineage Tab**: DualLineageViewer (technical + semantic)
- **Impact Tab**: ImpactAnalysisTab (downstream only)
- Limited integration, duplicated controls

### After (Unified Tab)
- **UnifiedLineageTab**: Combines all features
- **Direction Toggle**: User controls view mode
- **Shared Sidebar**: Explanation and AI assistant adapt to mode
- **Statistics**: Real-time counts for both directions

## Testing

### Backend Test
```bash
cd backend
go test ./internal/lineage -v -run TestFindBiDirectionalGraph
```

### Frontend Test
```bash
cd frontend
npm test -- ImpactGraph
```

### Integration Test
1. Select a semantic term or table
2. Open Impact Analysis tab
3. Toggle direction modes (upstream/downstream/both)
4. Verify counts match graph nodes
5. Check explanation changes based on direction

## Performance

- **Client-side filtering**: No API calls when changing direction
- **Single fetch**: Backend returns complete graph once
- **Deduplication**: Nodes in both paths merged with "both" direction
- **Lazy layout**: ReactFlow recalculates positions only on filter change

## Future Enhancements

- [ ] Separate depth controls (upstream vs downstream)
- [ ] Path highlighting (trace specific route)
- [ ] Export as PNG/SVG
- [ ] Comparison mode (compare two nodes)
- [ ] Time-travel (historical lineage)
- [ ] Column-level lineage integration

## Related Files

### Backend
- `backend/internal/lineage/sql_repo.go` - Bidirectional graph with metadata
- `backend/internal/api/lineage_handler.go` - HTTP handlers
- `backend/internal/analytics/impact_service.go` - Business logic

### Frontend
- `frontend/src/features/impact-analysis/components/UnifiedLineageTab.tsx`
- `frontend/src/features/impact-analysis/components/ImpactGraph.tsx`
- `frontend/src/features/impact-analysis/components/ImpactExplanation.tsx`
- `frontend/src/features/impact-analysis/components/ImpactQA.tsx`

### Documentation
- `AGE_REMOVAL_COMPLETE.md` - AGE to relational migration
- `REMOVE_AGE_INSTRUCTIONS.md` - Quick start guide
