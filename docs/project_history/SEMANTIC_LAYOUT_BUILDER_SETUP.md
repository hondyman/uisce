# Adding Semantic Layout Builder to Your Application

## Quick Integration Patch

### Step 1: Add the lazy import

Add this line after line 58 in `/frontend/src/App.tsx`:

```typescript
const DynamicUIGeneratorPage = lazyWithRetry(() => import('./pages/DynamicUIGeneratorPage'));
const SemanticLayoutBuilder = lazyWithRetry(() => import('./pages/SemanticLayoutBuilder'));  // <-- ADD THIS LINE
```

### Step 2: Add the route

Find the section with `<Route path="/dynamic-ui" element={<DynamicUIGeneratorPage />} />` (around line 205)

Add this route nearby:

```typescript
<Route path="/dynamic-ui" element={<DynamicUIGeneratorPage />} />
<Route path="/semantic-layout-builder" element={<SemanticLayoutBuilder />} />  {/* <-- ADD THIS LINE */}
```

### Step 3: Test it

1. Start your frontend: `npm start` (or `npm run dev`)
2. Navigate to: `http://localhost:3000/semantic-layout-builder`
3. You should see the visual layout builder!

### Step 4: Add a navigation link (optional)

If you want to add it to your MainNavigation, find the appropriate section and add:

```typescript
{
  label: 'Semantic Layout Builder',
  path: '/semantic-layout-builder',
  icon: <Sparkles className="w-4 h-4" />  // or any icon you prefer
}
```

---

## That's it! 🎉

You now have a fully functional visual dashboard builder integrated with your semantic layer.

### What you can do now:

1. **Test the basic functionality**:
   - Drag components onto the canvas
   - Select semantic views (currently using mock data)
   - Choose dimensions and measures
   - Export JSON configuration

2. **Next enhancement** (when ready):
   - Replace mock `semanticViews` state with real API call
   - See `SEMANTIC_LAYOUT_BUILDER_INTEGRATION_GUIDE.md` for details

3. **Build a renderer** (when ready):
   - Create components that consume the exported JSON
   - Execute semantic queries based on configuration
   - Render charts/tables with real data

---

## Files Modified

✅ `/frontend/src/pages/SemanticLayoutBuilder.tsx` - Created
✅ `/frontend/src/App.tsx` - Need to modify (2 lines)

## Files for Reference

📖 `/SEMANTIC_LAYOUT_BUILDER_QUICK_START.md` - Quick overview & demo script
📖 `/SEMANTIC_LAYOUT_BUILDER_INTEGRATION_GUIDE.md` - Detailed integration steps

---

## Visual Preview

Once you navigate to `/semantic-layout-builder`, you'll see:

```
┌─────────────────────────────────────────────────────────┐
│ 🌟 Semantic Layout Builder                              │
│ Build dashboards from your semantic layer with no code  │
│                              [Show JSON] [Export] [Save] │
└─────────────────────────────────────────────────────────┘

[Component Library]    [Canvas Grid]           [Configuration]
  📊 KPI Card             Empty canvas            ⚙ Select a
  📋 Data Table          (drag here)              component to
  📈 Line Chart                                   configure
  📊 Bar Chart
  🥧 Pie Chart
  📈 Area Chart

  Semantic Views:
  • Portfolio Positions
  • Trade History
```

Happy building! 🚀
