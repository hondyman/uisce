# Quick Start: Add Observability Console to Your Admin UI

## Copy-Paste Integration (5 minutes)

### Step 1: Add Routes

In your main `App.tsx` or `AdminApp.tsx`:

```tsx
import { Routes, Route } from "react-router-dom";
import { ObservabilityRoutes } from "./routes/ObservabilityRoutes";
import { useAuth } from "./context/AuthContext"; // Your auth

export function AdminApp() {
  const { user } = useAuth();
  
  return (
    <Routes>
      {/* existing routes */}
      <Route 
        path="/observability/*" 
        element={<ObservabilityRoutes currentUser={user} />} 
      />
    </Routes>
  );
}
```

### Step 2: Add Sidebar Nav

In your admin `Sidebar.tsx` or `Drawer.tsx`:

```tsx
import { DrawerContent } from "@mui/material";
import { ObservabilityNav } from "./components/navigation/ObservabilityNav";

export function AdminSidebar({ open }) {
  return (
    <Drawer open={open}>
      <DrawerContent>
        {/* existing nav items */}
        <Divider sx={{ my: 2 }} />
        <ObservabilityNav />
      </DrawerContent>
    </Drawer>
  );
}
```

### Step 3: Ensure Auth Hook Returns UserContext

Your `useAuth()` must return object with `role` field:

```tsx
interface UserContext {
  role: "admin" | "sre" | "tenant_admin" | "viewer";
  tenantId?: string;
}

const user = {
  id: "user-123",
  email: "sre@company.com",
  role: "sre",  // REQUIRED
  tenantId: "tenant-abc"
};
```

### Step 4: Backend Endpoints (Pseudo-Code)

In your backend, implement these 6 endpoints:

```go
// Global metrics
GET /api/metrics/global
→ { commitSuccessRate: "99.8%", s3Failures5m: 2, idempotencyHits5m: 412, ... }

// Region heatmap
GET /api/metrics/region-heatmap
→ [{ region: "us-east", bucket: "now", value: 150 }, ...]

// Tenant metrics
GET /api/metrics/tenant/:tenantId
→ { successRate: "99.8%", s3Failures: 3, idempotencyHits: 127, avgLatencyMs: 234 }

// Tenant plans
GET /api/plans?tenant=:tenantId&limit=10
→ [{ id: "p1", table: "customer_ltv", region: "us-east", status: "success", latency: 420, ... }]

// Plan timeline
GET /api/plans/timeline?limit=50
→ [{ planId: "p1", table: "customer_ltv", region: "us-east", status: "success", latency: 420, ... }]

// Snapshot lineage
GET /api/iceberg/lineage?table=customer_ltv
→ [{ snapshotId: 1, timestamp: "2024-01-01T00:00:00Z", fileCount: 150, dataBytes: 2500000 }, ...]
```

### Step 5: Test

Navigate to `http://localhost:3000/observability`

---

## URL Reference

| Feature | URL |
|---------|-----|
| Global Health | `/observability/` |
| Search Plans | `/observability/plan/plan-123` |
| Compare Plans | `/observability/compare?left=p1&right=p2` |
| Region Health | `/observability/regions` |
| Tenant Dashboard | `/observability/tenant/tenant-abc` |
| Plan Timeline | `/observability/timeline` |

---

## RBAC Test Cases

### As Admin
- ✅ See all pages
- ✅ Access region heatmap
- ✅ View all tenants

### As SRE
- ✅ See observability console, compare, regions, timeline
- ❌ Cannot access `/observability/tenant/...` (404)

### As Tenant Admin
- ✅ See own tenant dashboard at `/observability/tenant/{own-tenant-id}`
- ❌ Cannot see other tenant dashboards (403)

---

## Troubleshooting

### "Access Denied" page
- ✅ Verify `user.role` is set correctly in auth context
- ✅ Check role matches RequireRole expectations in ObservabilityRoutes

### "No data available" in tables
- ✅ Backend endpoints not implemented → mock data shows instead
- ✅ Implement endpoints from Step 4 checklist

### Sidebar nav not visible
- ✅ Ensure `<ObservabilityNav />` is placed inside Drawer/Sidebar
- ✅ Check CSS z-index if nav appears behind content

### Storage errors (Iceberg lineage)
- ✅ Ensure Iceberg metadata API is running
- ✅ Verify table name matches actual Iceberg table

---

## Customization Tips

### Change heatmap colors
Edit `RegionHeatmap.tsx`:
```tsx
const getColor = (value: number) => {
  if (value > 500) return "#your-red";
  if (value > 200) return "#your-orange";
  // ...
};
```

### Add custom filters to plans table
Edit `RecentPlansTable.tsx`:
```tsx
const filteredPlans = mockPlans.filter(p => p.table === selectedTable);
```

### Add more tabs to ObservabilityConsole
Edit `ObservabilityConsole.tsx`:
```tsx
<Tab label="Custom Tab" />
{activeTab === X && <YourComponent planId={planId} />}
```

---

## Performance Checklist

- [ ] Enable caching with SWR/React Query
- [ ] Implement server-side pagination for RecentPlansTable
- [ ] Add request debouncing on search box
- [ ] Lazy load pages via React Router Route code splitting (already done)
- [ ] Use backend aggregation for metrics (Prometheus queries)

---

✨ **Done!** Your Observability Console is now integrated.
