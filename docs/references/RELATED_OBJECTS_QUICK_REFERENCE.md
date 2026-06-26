# Quick Reference: Related Objects Integration

## Access the Feature

### For Users
**Entity Manager (Schema Builder) → "🔗 Relationships" Tab**
- URL: `/entity-config` (or wherever EntityConfigPageV2 is routed)
- Select tenant/datasource via tenant picker first
- Click Relationships tab
- Select entity from dropdown
- View/manage relationships

### For Developers
**Main File**: `frontend/src/pages/EntityConfigPageV2.tsx`

## Key Code Locations

### Tab Implementation
```typescript
// Lines ~534-670
<Tabs
  activeKey={mainViewTab}
  onChange={(key) => setMainViewTab(key as 'schema' | 'relationships')}
  items={[
    { key: 'schema', label: '📋 Schema Configuration', children: /* grid view */ },
    { key: 'relationships', label: '🔗 Relationships', children: /* RelatedObjectsPanel */ }
  ]}
/>
```

### Entity Selector
```typescript
// Lines ~678-686
<Select
  placeholder="Choose an entity..."
  value={selectedEntityForRelationships}
  onChange={setSelectedEntityForRelationships}
  options={Object.entries(entities).map(([key, entity]) => ({
    label: entity.businessName || entity.name,
    value: key,
  }))}
/>
```

### RelatedObjectsPanel Integration
```typescript
// Lines ~693-700
{selectedEntityForRelationships && entities[selectedEntityForRelationships] && tenant && datasource && (
  <Card style={{ marginTop: '16px' }}>
    <RelatedObjectsPanel
      tenantId={tenant.id}
      datasourceId={datasource.id || datasource.alpha_datasource_id}
      entity={entities[selectedEntityForRelationships].businessName || entities[selectedEntityForRelationships].name}
    />
  </Card>
)}
```

## API Changes

### Before
```typescript
export function fetchEntitySchema(): Promise<Entities>
// Called without tenant scope → 400 errors
```

### After
```typescript
export function fetchEntitySchema(tenantId?: string, datasourceId?: string): Promise<Entities>
// Now includes headers:
// X-Tenant-ID: {tenantId}
// X-Tenant-Datasource-ID: {datasourceId}
```

### All Callers Updated
- ✅ RelatedObjectsPage.tsx - Line 33
- ✅ EntityConfigPage.tsx - Line 65
- ✅ EntityConfigPageV2.tsx - Line 199
- ✅ EntityConfigPageV3.tsx - Line 155

## State Management

### New State in EntityConfigPageV2
```typescript
const [mainViewTab, setMainViewTab] = useState<'schema' | 'relationships'>('schema');
const [selectedEntityForRelationships, setSelectedEntityForRelationships] = useState<string>('');
```

### Tab Switching Logic
```typescript
onChange={(key) => setMainViewTab(key as 'schema' | 'relationships')}
```

## Component Props

### RelatedObjectsPanel
```typescript
type Props = {
  tenantId: string;           // tenant.id
  datasourceId: string;       // datasource.id || datasource.alpha_datasource_id
  entity: string;             // entity.businessName || entity.name
};
```

## Common Issues & Fixes

| Issue | Cause | Fix |
|-------|-------|-----|
| 400 Bad Request | Missing tenant headers | Ensure tenant/datasource selected via picker |
| Relationships tab not showing | Could be rendering issue | Check browser console for errors |
| Entity dropdown empty | No entities loaded | Verify schema loaded successfully |
| RelatedObjectsPanel shows "Error loading" | GraphQL query failed | Check Apollo Client cache |
| Suggestions not appearing | No catalog edges defined | Create relationships or run semantic analysis |

## Testing

### Unit Test
```typescript
// Test that fetchEntitySchema passes headers
it('should include tenant headers', async () => {
  const result = await fetchEntitySchema('tenant-123', 'ds-456');
  // Verify headers were added
});
```

### Integration Test
1. Select tenant/datasource
2. Open Entity Manager
3. Click Relationships tab
4. Select entity
5. Verify RelatedObjectsPanel renders
6. Check Network tab for proper headers

### Manual Test
1. Open DevTools Network tab
2. Click Relationships tab
3. Select entity
4. Look for GraphQL queries with proper headers
5. Verify responses contain relationships

## Future Enhancements

### Easy Wins
- [ ] Persist selected entity in URL parameters
- [ ] Add entity search within relationships tab
- [ ] Show relationship count badge on tab

### Medium Complexity
- [ ] Relationship export to CSV
- [ ] Relationship import UI
- [ ] Batch operations on relationships

### High Complexity
- [ ] Graph visualization
- [ ] Relationship impact analysis
- [ ] ML-based suggestions

## Debugging Tips

### Check Tenant Scope
```javascript
// In browser console
console.log(localStorage.getItem('selected_tenant'));
console.log(localStorage.getItem('selected_datasource'));
```

### Inspect API Calls
1. Open DevTools → Network tab
2. Filter for "entity-schema"
3. Check Request Headers for `X-Tenant-ID`
4. Check Response for entity data

### Monitor State
```typescript
// Add to EntityConfigPageV2 component
useEffect(() => {
  console.log('Main tab:', mainViewTab);
  console.log('Selected entity:', selectedEntityForRelationships);
}, [mainViewTab, selectedEntityForRelationships]);
```

### Clear Cache
```javascript
localStorage.clear();
location.reload();
```

## Related Documentation

- **Full Guide**: `RELATED_OBJECTS_INTEGRATION_GUIDE.md`
- **Implementation Summary**: `RELATED_OBJECTS_INTEGRATION_COMPLETE.md`
- **API Docs**: `agents.md` (Tenant scope requirements)
- **Component**: `frontend/src/components/catalog/RelatedObjectsPanel.tsx`

---

**Last Updated**: October 24, 2025
