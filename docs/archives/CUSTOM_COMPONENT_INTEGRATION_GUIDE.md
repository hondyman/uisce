# Custom Component Manager Integration Guide

Complete integration of Workday-style custom components into semlayer, with tenant-scoped features, cross-filtering, and multiple component types.

## 📁 Files Created

### Frontend Components

1. **`frontend/src/components/CustomComponentManager/CustomComponentManager.tsx`**
   - Main UI component with tabbed interface
   - Configurator for 6 component types
   - Event system and cross-filtering UI
   - Code generation and examples

2. **`frontend/src/components/CustomComponentManager/CustomComponentManager.module.css`**
   - Responsive styling for all component types
   - Theme colors and animations
   - Mobile-friendly layout

### Hooks & Services

3. **`frontend/src/hooks/useCustomComponents.ts`**
   - Custom React hook for state management
   - Auto-sync with backend API
   - Tenant-scoped data fetching

4. **`frontend/src/services/customComponentService.ts`**
   - REST API client for custom components
   - Full CRUD operations
   - Tenant/datasource headers and query params
   - Import/export functionality
   - API endpoint testing

### Pages

5. **`frontend/src/pages/CustomComponentPage.tsx`**
   - Wrapper page component for routing

## 🔧 Integration Steps

### Step 1: Add Route to AppRoutes.tsx

```tsx
import CustomComponentPage from "./pages/CustomComponentPage";

// Inside ProtectedApp() routes section:
<Route path="/custom-components" element={<ProtectedRoute><CustomComponentPage /></ProtectedRoute>} />

// Add to navigation:
<BlockableLink to="/custom-components" className="hover:underline">Custom Components</BlockableLink>
```

### Step 2: Backend API Endpoints Required

Create these endpoints in your backend:

```go
// GET /api/custom-components
// List all components for tenant + datasource
// Query params: tenant_id, datasource_id
// Headers: X-Tenant-ID, X-Tenant-Datasource-ID
func ListCustomComponents(w http.ResponseWriter, r *http.Request)

// POST /api/custom-components
// Create new component
func CreateCustomComponent(w http.ResponseWriter, r *http.Request)

// GET /api/custom-components/:id
// Get single component
func GetCustomComponent(w http.ResponseWriter, r *http.Request)

// PUT /api/custom-components/:id
// Update component
func UpdateCustomComponent(w http.ResponseWriter, r *http.Request)

// DELETE /api/custom-components/:id
// Delete component
func DeleteCustomComponent(w http.ResponseWriter, r *http.Request)

// POST /api/custom-components/test-api
// Test API endpoint connection
func TestComponentAPI(w http.ResponseWriter, r *http.Request)

// GET /api/custom-components/export
// Export all components as JSON
func ExportComponents(w http.ResponseWriter, r *http.Request)

// POST /api/custom-components/import
// Import components from JSON file
func ImportComponents(w http.ResponseWriter, r *http.Request)
```

### Step 3: Database Schema

Create table to store custom components:

```sql
CREATE TABLE custom_components (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  datasource_id UUID NOT NULL REFERENCES datasources(id),
  name VARCHAR(255) NOT NULL,
  type VARCHAR(50) NOT NULL,  -- web_component, iframe, api_integration, custom_widget, chart, custom_code
  config JSONB NOT NULL DEFAULT '{}',
  events JSONB NOT NULL DEFAULT '[]',
  filters JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id),
  is_active BOOLEAN DEFAULT true,
  description TEXT,
  UNIQUE (tenant_id, datasource_id, name)
);

CREATE INDEX idx_custom_components_tenant_ds 
  ON custom_components(tenant_id, datasource_id);
```

## 🎯 Component Types Supported

### 1. Web Component
- External ES Module URLs
- Custom tag names
- Full component lifecycle access
- Pass Workday context as props

### 2. iFrame Embed
- External URLs
- Configurable width/height
- PostMessage communication
- Cross-domain messaging

### 3. API Integration
- REST endpoint configuration
- Auto-refresh intervals (0 = manual)
- Bearer token auth with Workday API token
- Query parameter filtering

### 4. Custom Widget
- D3.js, Chart.js compatible
- Data source binding
- Custom visualization code
- CSS isolation

### 5. Interactive Chart
- Chart library support
- Click handlers with filtering
- Cross-component communication
- Real-time data updates

### 6. Custom Code
- Direct HTML/CSS/JavaScript
- Access to full Workday API
- Event emission and listening
- Secure sandboxing

## 📡 Workday Component API

Available to all custom components via `window.WorkdayAPI`:

```javascript
// Get business object data
const boData = window.WorkdayAPI.getBusinessObjectData();

// Emit events to other components
window.WorkdayAPI.emitEvent('filter', { field: 'region', value: 'West' });
window.WorkdayAPI.emitEvent('refresh', {});
window.WorkdayAPI.emitEvent('navigate', { url: '/orders', context: {} });

// Listen for events
window.WorkdayAPI.onFilter((filterData) => {
  console.log('Filter applied:', filterData);
  updateComponent(filterData);
});

window.WorkdayAPI.onRefresh(() => {
  loadData();
});

// Get auth token for API calls
const token = window.WorkdayAPI.getAuthToken();

// Navigate to other pages
window.WorkdayAPI.navigate('/detail', { id: record.id });

// Show notifications
window.WorkdayAPI.showNotification('Data loaded successfully');

// Query business objects directly
const results = await window.WorkdayAPI.queryBusinessObject('Customer', {
  filter: { region: 'West' }
});
```

## ✨ Cross-Filtering Example

### Scenario: Sales Dashboard

1. **Chart Component** (Sales by Region)
   ```javascript
   chart.on('click', (event) => {
     const region = event.data.region;
     window.WorkdayAPI.emitEvent('filter', {
       field: 'region',
       value: region,
       timestamp: Date.now()
     });
   });
   ```

2. **Orders List Component** listens:
   ```javascript
   window.WorkdayAPI.onFilter((filter) => {
     if (filter.field === 'region') {
       loadOrders({ region: filter.value });
     }
   });
   ```

3. Result: User clicks "West" bar → Orders list updates automatically!

## 🔐 Tenant Scope Requirements

All components automatically enforce tenant scope:

```typescript
// Frontend automatically adds:
GET /api/custom-components?tenant_id=xxx&datasource_id=yyy
Headers: X-Tenant-ID: xxx, X-Tenant-Datasource-ID: yyy

// Backend must validate:
1. Extract tenant_id and datasource_id from query params
2. Verify request user has access to tenant/datasource
3. Return only components for that scope
4. Reject if user lacks permission
```

See `agents.md` for full tenant scope architecture.

## 📦 Component Configuration Structure

```typescript
interface CustomComponent {
  id: string;
  name: string;
  type: 'web_component' | 'iframe' | 'api_integration' | 'custom_widget' | 'chart' | 'custom_code';
  config: {
    url?: string;              // Web component or iframe URL
    apiEndpoint?: string;       // API integration endpoint
    htmlTemplate?: string;      // Custom code HTML
    jsCode?: string;            // Custom code JavaScript
    cssCode?: string;           // Custom code CSS
    dataSource?: string;        // Chart/widget data source
    refreshInterval?: number;   // Auto-refresh in seconds
    width?: string;             // iFrame width
    height?: string;            // iFrame height
    tagName?: string;           // Web component tag name
  };
  events: Array<{
    id: string;
    eventName: string;
    action: 'refresh' | 'filter' | 'navigate' | 'custom';
    targetComponentId?: string;
    customScript?: string;
  }>;
  filters: Array<{
    id: string;
    field: string;
    operator: string;
    listenToComponent?: string;
  }>;
  tenantId?: string;
  datasourceId?: string;
  createdAt?: string;
  updatedAt?: string;
}
```

## 🛡️ Security Considerations

1. **Content Security Policy**
   - Strict CSP headers for custom code
   - Sandbox iframe contexts
   - Validate external URLs

2. **Code Execution**
   - Use Web Workers for untrusted code
   - Restrict DOM access
   - Validate all user inputs

3. **API Access**
   - All API calls use bearer tokens
   - Tenant scope validation
   - Rate limiting per component

4. **Data Protection**
   - Encrypt sensitive config
   - Audit component creation/deletion
   - Log all component executions

## 🧪 Testing

### Test Custom Code Component
```javascript
const template = `
  <div id="demo">
    <h3>Sales: <span id="total">0</span></h3>
    <button id="fetchBtn">Fetch Data</button>
  </div>
`;

const jsCode = `
  document.getElementById('fetchBtn').addEventListener('click', () => {
    window.WorkdayAPI.emitEvent('filter', {
      field: 'status',
      value: 'completed'
    });
  });
`;
```

### Test Cross-Filtering
1. Create Chart component with sales data
2. Create List component with orders
3. Configure List to listen for 'region' field
4. Click chart bar → List should filter automatically

## 📚 Related Documentation

- **agents.md** - Tenant scope architecture and requirements
- **API Layer README** - Backend API structure and patterns
- **Bundle Editor** - Similar component for bundle configuration

## 🚀 Next Steps

1. Implement backend API endpoints
2. Create database migrations
3. Add component validation
4. Build component marketplace/templates
5. Add component versioning
6. Create component testing framework
7. Build component marketplace sharing
