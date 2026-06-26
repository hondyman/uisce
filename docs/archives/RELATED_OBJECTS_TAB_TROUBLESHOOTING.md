# Related Objects Tab - Troubleshooting Guide

## Common Issues & Solutions

### Issue 1: "Error loading related objects: ApolloError..."
**Status**: ✅ **FIXED** - This is why we rewrote the component

**Solution**: The new RelatedObjectsTab component uses REST API instead of Apollo GraphQL, so this error no longer occurs.

---

### Issue 2: "Please select a tenant and datasource to view relationships"
**Status**: Expected behavior (when scope not selected)

**Cause**: User hasn't selected a tenant and datasource in the Fabric Builder shell

**Solution**:
1. Go to Fabric Builder main page
2. Use the tenant picker dropdown at top
3. Select a tenant
4. Select a product
5. Select a datasource
6. Navigate to Entity Manager
7. Click an entity
8. Related Objects tab should now work

---

### Issue 3: No relationships showing (blank card grid)
**Status**: Expected when entity has no relationships defined

**Solutions**:
1. **Verify API is running**: Check backend is started
2. **Check API endpoint**: Verify `/api/relationships/objects` exists
3. **Check database**: Ensure relationships are defined in the database for this entity
4. **Check logs**: Look at browser console (F12) for network errors

---

### Issue 4: Diagram view not displaying correctly
**Status**: Minor display issue

**Solutions**:
1. **Clear browser cache**: `Ctrl+Shift+Del` (or `Cmd+Shift+Del` on Mac)
2. **Check window size**: Diagram needs at least 600px height
3. **Resize window**: Sometimes layout breaks if window is too small
4. **Reload page**: `Ctrl+R` or `Cmd+R`

---

### Issue 5: Dark mode colors don't look right
**Status**: Verify theme is applied

**Solutions**:
1. **Check theme setting**: Ensure dark mode is selected in app settings
2. **Verify Tailwind config**: Check `tailwind.config.js` has `darkMode: 'class'`
3. **Check HTML element**: Verify `<html>` has `class="dark"` attribute
4. **Force refresh**: Clear cache and reload: `Ctrl+Shift+R`

---

### Issue 6: Edit/Delete buttons not working
**Status**: Placeholder (functionality not implemented yet)

**Note**: These buttons are UI-only. Backend implementation needed for actual functionality.

**To implement**:
1. Add click handlers in RelatedObjectsTab.tsx
2. Create mutation functions for edit/delete
3. Call backend API endpoints
4. Handle responses and error states

---

### Issue 7: Relationships load slowly
**Status**: Performance optimization opportunity

**Potential causes**:
1. **Large number of relationships**: If entity has hundreds of relationships
2. **Network latency**: Slow API response
3. **Browser performance**: Heavy page with many components

**Solutions**:
1. **Enable pagination**: Add offset/limit to API call
2. **Use virtual scrolling**: For large datasets
3. **Optimize network**: Check network tab in DevTools
4. **Profile app**: Use Chrome DevTools Performance tab

---

### Issue 8: "Invalid X-Tenant-ID" error
**Status**: Authentication/scoping issue

**Cause**: Tenant scope not passed to API or invalid value

**Solution**:
1. Verify localStorage has tenant data:
   ```javascript
   // In browser console
   console.log(localStorage.getItem('selected_tenant'));
   console.log(localStorage.getItem('selected_datasource'));
   ```
2. If empty, select tenant again in UI
3. Check EntityDetailsPage passes correct props to RelatedObjectsTab

---

### Issue 9: Component not found (import error)
**Status**: Build error

**Error**: 
```
Cannot find module '../components/relationship/RelatedObjectsTab'
```

**Solution**:
1. Verify file exists at `frontend/src/components/relationship/RelatedObjectsTab.tsx`
2. Create directory if missing: `mkdir -p frontend/src/components/relationship`
3. Rebuild: `npm run build`

---

### Issue 10: TypeScript compilation errors
**Status**: Type safety

**Solutions**:
1. **Run type check**: `npx tsc --noEmit`
2. **Fix errors**: Update types to match API response
3. **Check prop types**: Verify RelatedObjectsTab receives correct prop types
4. **Rebuild**: `npm run build`

---

## Verification Checklist

### After Implementation
- [ ] Build succeeds: `npm run build` ✓
- [ ] No TypeScript errors: `npx tsc --noEmit` ✓
- [ ] Component imports correctly
- [ ] EntityDetailsPage loads Related Objects tab
- [ ] Tab displays without errors

### User Testing
- [ ] Select entity → Related Objects tab shows
- [ ] Card view displays relationships
- [ ] Diagram view displays relationships
- [ ] Dark/light mode toggle works
- [ ] Toggle between card and diagram views
- [ ] Error messages appear when appropriate

---

## Development Debugging

### Enable Detailed Logging
The component uses `devLog` and `devError` utilities:

```typescript
// These are enabled automatically
devLog('🔗 Fetching relationships...', data);
devError('Error fetching relationships:', error);
```

**Check console output**:
1. Open browser DevTools (F12)
2. Go to Console tab
3. Look for messages with 🔗 emoji
4. Check for any 🔴 errors

### Network Debugging
1. Open DevTools (F12)
2. Go to Network tab
3. Refresh page
4. Look for `/api/relationships/objects` request
5. Check:
   - Status code (200 = success)
   - Headers (X-Tenant-ID present?)
   - Response body (valid JSON?)

### Check API Response Format

Expected successful response:
```json
{
  "relationships": [
    {
      "id": "rel-1",
      "sourceEntity": "Customer",
      "targetEntity": "Order",
      "cardinality": "One-to-Many",
      "keyFields": {
        "source": "Customer(CustomerID)",
        "target": "Order(CustomerID)"
      }
    }
  ]
}
```

---

## Performance Tips

### For Large Relationship Sets (100+)
1. **Implement pagination**: 
   ```typescript
   const limit = 50;
   const offset = (page - 1) * limit;
   // Add to API call: `&limit=${limit}&offset=${offset}`
   ```

2. **Use virtual scrolling**:
   ```typescript
   import { FixedSizeList } from 'react-window';
   ```

3. **Lazy load diagram**:
   ```typescript
   // Don't render diagram until user clicks tab
   {viewType === 'diagram' && <DiagramView />}
   ```

---

## Browser Compatibility

| Browser | Support | Notes |
|---------|---------|-------|
| Chrome 90+ | ✅ Full | SVG, CSS Grid, modern CSS |
| Firefox 88+ | ✅ Full | Full support |
| Safari 14+ | ✅ Full | Full support |
| Edge 90+ | ✅ Full | Chromium-based |
| IE 11 | ❌ Not | Uses modern JavaScript/CSS |

---

## Backend Configuration Needed

For the component to work, ensure backend has:

1. **Endpoint**: `GET /api/relationships/objects`
2. **Query params**: 
   - `tenant_id`: UUID
   - `datasource_id`: UUID
   - `entity`: string (entity name)
3. **Headers accepted**: 
   - `X-Tenant-ID`
   - `X-Tenant-Datasource-ID`
4. **Returns**: List of relationship objects with structure shown above

---

## Quick Fixes

### Component won't load
```bash
# Clear node_modules cache
rm -rf node_modules/.vite
npm run build
```

### Types not working
```bash
# Regenerate TypeScript definitions
npx tsc --noEmit
```

### Dark mode not applied
```javascript
// In browser console
document.documentElement.classList.add('dark');
// or
document.documentElement.classList.remove('dark');
```

---

## Support Resources

- **Component file**: `frontend/src/components/relationship/RelatedObjectsTab.tsx`
- **Integration**: `frontend/src/pages/EntityDetailsPage.tsx`
- **Styles**: `frontend/src/components/relationship/RelatedObjectsTab.module.css`
- **Documentation**: `RELATED_OBJECTS_TAB_IMPLEMENTATION.md`
- **This guide**: `RELATED_OBJECTS_TAB_TROUBLESHOOTING.md`

---

## Need Help?

1. Check this troubleshooting guide first
2. Search existing issues in GitHub
3. Check browser console for error messages
4. Review network requests (Network tab in DevTools)
5. Check backend logs for API errors
