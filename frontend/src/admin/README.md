# SemLayer Admin UI

Production-ready admin interface for managing tenants, API keys, and usage analytics.

## Features

- **Tenant Management**: Create, list, update, suspend/unsuspend tenants
- **API Key Management**: Create and manage API keys with role-based access control
- **Usage Analytics**: View real-time usage statistics, trends, and endpoint breakdowns
- **Responsive Design**: Mobile-friendly interface with collapsible sidebar navigation
- **TypeScript**: Fully typed React components and API responses
- **Error Handling**: Comprehensive error messages and validation

## Project Structure

```
frontend/src/admin/
├── hooks/
│   └── useAdmin.ts           # React hooks for API interactions
├── layout/
│   ├── AdminLayout.tsx       # Main shell with sidebar navigation
│   └── AdminLayout.css
├── pages/
│   ├── DashboardPage.tsx     # Overview and quick stats
│   ├── DashboardPage.css
│   ├── TenantsPage.tsx       # Tenant CRUD interface
│   ├── TenantsPage.css
│   ├── APIKeysPage.tsx       # API key management
│   ├── APIKeysPage.css
│   ├── UsageAnalyticsPage.tsx # Usage statistics and charts
│   ├── UsageAnalyticsPage.css
├── types/
│   └── index.ts              # TypeScript type definitions
├── routes.tsx                # Route configuration
├── index.ts                  # Public exports
└── README.md                 # This file
```

## Installation & Setup

### 1. Environment Configuration

Create or update `.env` in your frontend root:

```env
REACT_APP_API_URL=http://localhost:8082/api
```

Adjust the URL to match your backend API endpoint.

### 2. Dependencies

Ensure your `package.json` includes:

```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.x.x"
  }
}
```

### 3. Route Integration

In your main `App.tsx` or router configuration:

```tsx
import { adminRoutes } from "./admin";
import { useRoutes } from "react-router-dom";

export function App() {
  const routes = [
    // ... other routes
    ...adminRoutes,
  ];
  
  return useRoutes(routes);
}
```

### 4. Authentication

The admin UI expects a JWT token in `localStorage` under the key `token`. The token should contain:

```json
{
  "sub": "user-id",
  "roles": ["GLOBAL_OPS"],
  "tenant_ids": ["tenant-1", "tenant-2"]
}
```

On backend authentication failure, the user will be redirected to `/login`.

## API Endpoints

All endpoints are prefixed with `/api/admin` and require `GLOBAL_OPS` role:

### Tenants

- `GET /tenants?limit=50&offset=0` - List tenants
- `POST /tenants` - Create tenant
- `GET /tenants/{id}` - Get tenant
- `PATCH /tenants/{id}` - Update tenant
- `DELETE /tenants/{id}` - Delete tenant
- `POST /tenants/{id}/suspend` - Suspend tenant
- `POST /tenants/{id}/unsuspend` - Unsuspend tenant

### API Keys

- `GET /api-keys?limit=50&offset=0` - List API keys
- `POST /api-keys` - Create API key
- `GET /api-keys/{id}/usage?limit=100` - Get API key usage

### Usage Analytics

- `GET /tenants/{id}/usage/daily?days=30` - Daily usage stats
- `GET /tenants/{id}/usage/endpoints?limit=20` - Top endpoints
- `GET /tenants/{id}/usage/recent?limit=100` - Recent requests

## Components

### AdminLayout

Main shell component providing sidebar navigation and layout structure.

**Props**: None (uses React Router)

**Exports**:
- Sidebar with collapsible navigation
- Main content area with outlet
- Header with current user info and time

```tsx
<AdminLayout>
  <DashboardPage />
</AdminLayout>
```

### DashboardPage

Admin overview showing quick statistics, recent activity, and quick actions.

**Features**:
- Total/active tenants and API keys
- Recent activity cards
- Quick action buttons
- Platform information

### TenantsPage

Tenant management interface with CRUD operations.

**Features**:
- List all tenants with pagination
- Create new tenant modal
- Tenant details (name, code, region, plan, rate limits)
- Suspend/unsuspend tenants
- Edit tenant information

**Form Fields**:
- Name (required)
- Code (required)
- Region (select)
- Plan (free/pro/enterprise)

### APIKeysPage

API key management interface.

**Features**:
- List all API keys
- Create new API key with role assignment
- Scoped access (global or tenant-specific)
- Revocation status tracking
- Usage links

**Form Fields**:
- Name (required)
- Tenant IDs (comma-separated, optional)
- Roles (multi-select: USER, TENANT_ADMIN, GLOBAL_OPS)

### UsageAnalyticsPage

Usage statistics and analytics dashboard.

**Features**:
- Tenant selection
- Daily usage trends (bar chart)
- Top endpoints breakdown
- Summary statistics (total, average, peak)
- Export capabilities (placeholder)

**Visualizations**:
- Daily trend bar chart (last 14 days)
- Endpoint distribution table
- Summary cards

## Hooks

### useTenants(limit, offset)

Fetch list of tenants with pagination.

```tsx
const { tenants, total, loading, error, refetch } = useTenants(50, 0);
```

### useTenant(tenantId)

Fetch single tenant details.

```tsx
const { tenant, loading, error, refetch } = useTenant(tenantId);
```

### useCreateTenant()

Create a new tenant.

```tsx
const { create, loading, error } = useCreateTenant();

const newTenant = await create({
  name: "Acme Corp",
  code: "acme",
  region: "us-east-1",
  plan: "pro",
});
```

### useUpdateTenant(tenantId)

Update existing tenant.

```tsx
const { update, loading, error } = useUpdateTenant(tenantId);

await update({
  name: "Updated Name",
  plan: "enterprise",
});
```

### useSuspendTenant(tenantId)

Suspend or unsuspend tenant.

```tsx
const { suspend, unsuspend, loading, error } = useSuspendTenant(tenantId);

await suspend();
await unsuspend();
```

### useAPIKeys(limit, offset)

Fetch list of API keys.

```tsx
const { keys, total, loading, error, refetch } = useAPIKeys(50, 0);
```

### useAPIKeyUsage(apiKeyId, limit)

Fetch usage history for API key.

```tsx
const { usage, loading, error, refetch } = useAPIKeyUsage(keyId, 100);
```

### useTenantDailyUsage(tenantId, days)

Fetch daily usage statistics for tenant.

```tsx
const { stats, loading, error, refetch } = useTenantDailyUsage(tenantId, 30);
```

### useTenantEndpointUsage(tenantId, limit)

Fetch top endpoints by request count.

```tsx
const { stats, loading, error, refetch } = useTenantEndpointUsage(tenantId, 20);
```

## TypeScript Types

All types are exported from `admin/types/index.ts`:

```tsx
import type {
  Tenant,
  APIKey,
  APIKeyUsage,
  DailyUsageStats,
  EndpointUsageStats,
  ListTenantsResponse,
} from "@/admin";
```

## Styling

The admin UI uses:
- CSS Grid for layouts
- CSS Flexbox for alignment
- CSS custom properties for theming (gradients, colors)
- Responsive design with mobile breakpoints

### Color Palette

- Primary: `#667eea` to `#764ba2` (gradient)
- Success: `#52c41a`
- Warning: `#faad14`
- Error: `#f5222d`
- Neutral: `#666` to `#999`

### Responsive Breakpoints

- Desktop: 1024px+
- Tablet: 768px - 1023px
- Mobile: < 768px

## Error Handling

All API calls include error handling with user-friendly messages:

```tsx
try {
  await create(data);
} catch (err) {
  // Error automatically set in state and displayed
  alert(`Error: ${error}`);
}
```

Error messages are shown in dismissible alerts or inline form validation.

## Future Enhancements

- [ ] Export analytics to CSV
- [ ] Generate custom reports
- [ ] API key rotation
- [ ] Tenant usage quotas and notifications
- [ ] Real-time activity feed
- [ ] Audit logs with filtering
- [ ] Integration with analytics provider (Recharts, Chart.js)
- [ ] Dark mode support
- [ ] Internationalization (i18n)

## Development

### Running the Admin UI

```bash
cd frontend
npm install
npm start
```

The admin panel will be available at `http://localhost:3000/admin`

### Running Tests

```bash
npm test
```

### Building for Production

```bash
npm run build
```

## Backend Integration Checklist

- [ ] All `/api/admin/*` endpoints implemented
- [ ] GLOBAL_OPS role enforcement on all endpoints
- [ ] Tenant store with CRUD operations
- [ ] API key store with creation and revocation
- [ ] Usage middleware logging requests
- [ ] Usage analytics endpoints returning proper schemas

See [Backend Documentation](../../backend/README.md) for server-side setup.

## Troubleshooting

### API calls return 401

Check that:
1. JWT token is stored in localStorage under `token` key
2. Token contains `roles: ["GLOBAL_OPS"]`
3. Backend is running on configured API_URL

### Form submission fails

Check browser console for:
1. Network error messages
2. CORS issues (ensure backend allows frontend origin)
3. Validation errors in response

### Sidebar navigation not working

Ensure React Router is properly configured as parent component and routes are registered with `adminRoutes`.

## Support

For issues or feature requests, contact the development team or open an issue in the repository.
