# API Endpoint Catalog Page - Implementation Guide

## Overview

A new frontend page has been created to browse, search, and view detailed information about all API endpoints in your catalog. This page integrates with your backend API endpoints system.

## 📍 Location & Route

- **File**: `/frontend/src/pages/catalog/APIEndpointCatalogPage.tsx`
- **Route**: `/core/api-endpoint-catalog`
- **Access**: Available from the main navigation after selecting a tenant and datasource

## ✨ Features

### 1. **Endpoint Table View**
- Displays all API endpoints in a responsive Material-UI table
- Shows: Method, Path, Description, Category, Version, Status
- Color-coded HTTP method badges (GET=blue, POST=green, PUT=orange, DELETE=red)
- Active/Inactive and Deprecated status indicators

### 2. **Search & Filter**
- **Full-text search**: Search by endpoint name, path, or description
- **Category filter**: Dropdown to filter by endpoint category
- **Real-time filtering**: Results update as you type
- Results summary shows matching count

### 3. **Endpoint Details Modal**
Click "Details" button on any endpoint to see:
- **Basic Information**: Path, Method, Version, Category, Description
- **Status**: Active/Inactive, Deprecated, Auth Required flags
- **Request & Response Schemas**: Full JSON schema definitions
- **Metadata**: Rate limit, Subcategory, Created/Updated dates

### 4. **Tenant Scope Integration**
- Requires tenant and datasource selection via Connections
- Shows warning if no scope selected
- Automatically fetches data for selected scope
- Includes tenant ID in API headers for security

### 5. **Loading & Error States**
- Loading indicator while fetching endpoints
- Helpful error messages if fetch fails
- Empty state if no endpoints in catalog
- No results state if search returns nothing

## 🔗 API Integration

### Backend Endpoint Used
```
GET /api/api-endpoints?tenant_id={tenantId}&datasource_id={datasourceId}
```

### Required Headers
```
X-Tenant-ID: {tenantId}
X-Tenant-Datasource-ID: {datasourceId}
```

### Expected Response Format
```json
[
  {
    "id": "uuid",
    "endpoint_name": "List Entities",
    "endpoint_path": "/api/entities",
    "http_method": "GET",
    "description": "Retrieve all entities",
    "category": "Entities",
    "subcategory": "List",
    "request_schema": { /* JSON schema */ },
    "response_schema": { /* JSON schema */ },
    "request_examples": [ /* array of examples */ ],
    "response_examples": [ /* array of examples */ ],
    "is_active": true,
    "version": "v1.0",
    "deprecated": false,
    "auth_required": true,
    "rate_limit": 100,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
]
```

## 🎨 UI Components Used

- **Material-UI**: Table, Paper, Card, Dialog, Button, Chip, TextField, Select, Alert
- **Icons**: SearchIcon, RefreshIcon, InfoIcon, ErrorIcon
- **Context**: TenantContext (for tenant/datasource scope)
- **Styling**: MUI sx prop for all styling (no CSS files needed)

## 📱 Responsive Design

- Mobile-friendly table with horizontal scroll
- Filter toolbar wraps on smaller screens
- Modal dialog is full width on desktop, adjusted on mobile
- All text truncation and sizing adapts to viewport

## 🔄 State Management

Uses React hooks:
- `useState` for: endpoints, loading, error, searchTerm, selectedCategory, selectedEndpoint, categories, openDetailsDialog
- `useEffect` for: fetching endpoints when tenant/datasource changes

## 🚀 Usage

### View API Endpoints
1. Navigate to menu → Core → **API Endpoint Catalog** (or go to `/core/api-endpoint-catalog`)
2. Select a tenant and datasource from Connections if not already selected
3. Table loads with all endpoints

### Search Endpoints
1. Type in the search box to filter by name, path, or description
2. Results update in real-time

### Filter by Category
1. Use the "Category" dropdown to show only endpoints in a specific category
2. Select "All Categories" to clear filter

### View Endpoint Details
1. Click the **Details** button on any row
2. Modal opens showing full endpoint information
3. Close button or click outside to dismiss modal

### Refresh Data
1. Click the **Refresh** button in the toolbar
2. Latest data fetches from backend

## 🔌 Adding to Navigation

The page is already added to App.tsx routes. To add it to your navigation menu, update your MainNavigation component:

```tsx
// In MainNavigation.tsx
<MenuItem href="/core/api-endpoint-catalog">API Catalog</MenuItem>
```

## 📊 Data Types

```typescript
interface APIEndpoint {
  id: string;
  endpoint_name: string;
  endpoint_path: string;
  http_method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  description: string;
  category: string;
  subcategory: string;
  request_schema: Record<string, unknown>;
  response_schema: Record<string, unknown>;
  request_examples: Record<string, unknown>[];
  response_examples: Record<string, unknown>[];
  is_active: boolean;
  version: string;
  deprecated: boolean;
  auth_required: boolean;
  rate_limit: number;
  created_at: string;
  updated_at: string;
}
```

## 🛠️ Customization

### Change the Route
Edit `/App.tsx`:
```tsx
<Route path="/your-custom-path" element={<APIEndpointCatalogPage />} />
```

### Modify Table Columns
In `APIEndpointCatalogPage.tsx`, update the `TableHead` to add/remove `TableCell` components with new columns.

### Adjust Colors
- HTTP method colors: Edit the `getMethodColor()` function
- Status colors: Modify the color props on `Chip` components

### Change Modal Size
In the Dialog component, adjust the `maxWidth` prop:
```tsx
<Dialog open={openDetailsDialog} onClose={handleCloseDetails} maxWidth="lg" fullWidth>
```

## ⚠️ Known Limitations

1. No pagination support (shows all endpoints at once)
2. No sorting by column (only search/filter available)
3. No create/edit/delete actions (read-only catalog view)
4. Details modal uses simple JSON display (no code highlighting)

## 🐛 Troubleshooting

### "Please select a tenant" warning appears
- **Cause**: No tenant/datasource selected
- **Solution**: Go to Connections and select a tenant and datasource

### No endpoints display
- **Cause**: API returns empty list or error
- **Solution**: Check backend logs and ensure `/api/api-endpoints` endpoint is working

### Search not working
- **Cause**: Endpoints data structure mismatch
- **Solution**: Check JSON response format matches the expected interface

### Modal doesn't open
- **Cause**: Click handler not firing or endpoint data missing
- **Solution**: Check browser console for errors

## 📈 Future Enhancements

1. **Pagination**: Add infinite scroll or paginated table
2. **Sorting**: Click column headers to sort
3. **CRUD Operations**: Add buttons to create/edit/delete endpoints
4. **Code Highlighting**: Use Prism.js for syntax highlighting in schemas
5. **Export**: Export filtered endpoints to CSV/JSON
6. **Related Objects**: Show entities/datasources using each endpoint
7. **API Testing**: Built-in endpoint testing widget
8. **Versioning**: Show endpoint version history
9. **Performance Metrics**: Display usage statistics and error rates
10. **Documentation**: Link to full API documentation

## 🔗 Related Files

- Backend catalog implementation: `backend/internal/api/api_endpoints_catalog.go`
- Backend migration: `backend/internal/api/migrations/001_create_api_endpoints_catalog.sql`
- Backend integration guide: `BACKEND_API_CATALOG_INTEGRATION.md`
- Event syndication system: All `EVENT_SYNDICATION_*.md` files

## 📚 Documentation

See the following documents for more context:
- `FRONTEND_BACKEND_INTEGRATION_ROADMAP.md` - Overall integration progress
- `BACKEND_API_CATALOG_INTEGRATION.md` - Backend API documentation
- `EVENT_SYNDICATION_DELIVERY_SUMMARY.md` - Event system overview

## ✅ Testing Checklist

- [ ] Page loads without errors
- [ ] Tenant scope validation works
- [ ] Endpoints load from backend
- [ ] Search filters correctly
- [ ] Category filter works
- [ ] Details modal opens and displays correctly
- [ ] Refresh button fetches latest data
- [ ] Error states display appropriate messages
- [ ] Empty states display when no data
- [ ] Mobile responsive layout works
- [ ] All keyboard navigation works (tab, enter, escape)
