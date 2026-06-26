# Quick Start: Unified Lineage & Impact Tab

## 🎯 What Changed?

You now have a **single unified tab** that combines lineage (upstream) and impact (downstream) analysis with a direction toggle.

## ✅ Status
- ✅ Backend: Enhanced with direction metadata
- ✅ Frontend: UnifiedLineageTab component created
- ✅ Builds: Both backend and frontend compile successfully

## 🚀 How to Use

### 1. Import the Component
```tsx
import { UnifiedLineageTab } from '@/features/impact-analysis';
```

### 2. Use in Your Modal/Tab System
```tsx
<UnifiedLineageTab
  nodeType="semantic_term"    // or 'table', 'column', 'business_object', etc.
  nodeId="your-node-id"
  initialDirection="both"     // 'upstream' | 'downstream' | 'both'
/>
```

### 3. Features Available

**Direction Toggle** (top-left of graph):
- **↑ Lineage**: Shows only upstream dependencies (what feeds into this)
- **⇅ Both**: Shows complete bidirectional graph
- **↓ Impact**: Shows only downstream impact (what this affects)

**Floating Controls** (left sidebar):
- **[◉] Graph**: Main visualization (always active)
- **[📄] Explanation**: AI-generated explanation adapted to direction
- **[💬] AI Assistant**: Ask questions about relationships
- **[▣] Sidebar**: Toggle sidebar visibility

**Statistics Chips**:
- Blue chip: Number of upstream nodes
- Yellow chip: Number of downstream nodes

## 🔄 Migration Path

### Old Setup (Two Separate Tabs)
```tsx
<Tabs>
  <Tab label="Lineage">
    <DualLineageViewer ... />
  </Tab>
  <Tab label="Impact">
    <ImpactAnalysisTab ... />
  </Tab>
</Tabs>
```

### New Setup (Single Unified Tab)
```tsx
<Tabs>
  <Tab label="Lineage & Impact">
    <UnifiedLineageTab
      nodeType={asset.type}
      nodeId={asset.id}
      initialDirection="both"
    />
  </Tab>
</Tabs>
```

## 🎨 UI Preview

```
┌─────────────────────────────────────────────────┐
│ [≡] Controls                   ┌──────────────┐ │
│ [◉] Graph                      │  Direction   │ │
│ [📄] Explanation               │  [↑ Lineage] │ │
│ [💬] AI                        │  [⇅ Both]    │ │
│ [▣] Sidebar                    │  [↓ Impact]  │ │
│                                │              │ │
│                                │  🔵 5 up     │ │
│                                │  🟡 8 down   │ │
│         Graph Area             └──────────────┘ │
│         with colored                            │
│         nodes and arrows                        │
│                                                 │
│  ┌───────────────────────────────────────────┐ │
│  │ Sidebar (Explanation or AI Assistant)     │ │
│  │ Content adapts to current direction mode  │ │
│  └───────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

## 🧪 Test It

### Start Backend
```bash
cd backend
go run ./cmd/server
```

### Start Frontend
```bash
cd frontend
npm run dev
```

### Test Flow
1. Open your app (usually http://localhost:5173)
2. Navigate to catalog
3. Select any semantic term, table, or business object
4. Open the Impact Analysis tab
5. Use direction toggle to switch between:
   - Upstream (lineage)
   - Both (full graph)
   - Downstream (impact)
6. Watch the graph filter instantly (no API calls!)
7. Check statistics chips for node counts
8. Try the AI assistant with direction-aware questions

## 📊 What the Backend Does

The backend's `FindBiDirectionalGraph()` now:

1. Fetches upstream nodes (recursive CTE)
2. Fetches downstream nodes (recursive CTE)
3. Merges and deduplicates
4. Adds direction metadata to each node/edge:
   ```json
   {
     "id": "customer_table",
     "metadata": {
       "direction": "upstream",
       "is_lineage": true
     }
   }
   ```

## 🎯 Key Benefits

✅ **Single API call** instead of separate lineage/impact requests
✅ **Instant filtering** - toggle direction without loading
✅ **Complete context** - see full graph, filter as needed
✅ **Better UX** - unified controls, consistent styling
✅ **Performance** - client-side filtering, no re-fetching

## 📝 API Endpoint

**GET** `/api/lineage/node/{nodeId}/graph?node_type={type}`

Returns:
```json
{
  "nodes": [
    {
      "id": "node1",
      "type": "semantic_term",
      "label": "Customer ID",
      "properties": {
        "metadata": {
          "direction": "upstream",
          "is_lineage": true
        }
      }
    }
  ],
  "edges": [
    {
      "id": "edge1",
      "source": "node1",
      "target": "node2",
      "type": "derives_from",
      "properties": {
        "metadata": {
          "direction": "upstream"
        }
      }
    }
  ]
}
```

## 🐛 Troubleshooting

**Graph shows no nodes?**
- Check browser console for API errors
- Verify node exists in database
- Check depth setting (default: 5 levels)

**Direction toggle not working?**
- Ensure `directionMode` prop is passed to ImpactGraph
- Check metadata is present in API response
- Verify `onStatsUpdate` callback is wired up

**Sidebar not showing?**
- Check `sidebarOpen` state
- Verify sidebar toggle button works
- Inspect CSS for `.impact-analysis-sidebar.collapsed`

## 📚 Documentation

- [UNIFIED_LINEAGE_GUIDE.md](./UNIFIED_LINEAGE_GUIDE.md) - Complete guide
- [UNIFIED_LINEAGE_IMPLEMENTATION.md](./UNIFIED_LINEAGE_IMPLEMENTATION.md) - Implementation details
- [AGE_REMOVAL_COMPLETE.md](./AGE_REMOVAL_COMPLETE.md) - AGE removal context

## 💡 Pro Tips

1. **Start with "Both" mode** to see complete context
2. **Switch to "Upstream"** when tracing data sources
3. **Switch to "Downstream"** for change impact analysis
4. **Use AI Assistant** to ask "What tables feed into this?"
5. **Fullscreen mode** for complex graphs with many nodes

---

**Need help?** Check the full [UNIFIED_LINEAGE_GUIDE.md](./UNIFIED_LINEAGE_GUIDE.md) or ask in the AI Assistant sidebar!
