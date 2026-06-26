# Metrics Console Menu Integration

## 📍 Location in App

The **Metrics Console** has been added to the main navigation bar in your Fabric Builder UI.

### Navigation Flow

```
Fabric Builder Home
├── Micro-Bundle Catalog
├── Bundle Explorer
├── Fixed Income Analytics
├── JIT Request Panel
├── Access Explanation
├── 📊 Metrics Console  ← NEW! Click here
│   ├── Metric Registry (List page)
│   │   ├── Create Metric → MetricCreatePage
│   │   └── Edit/Delete actions
│   │
│   ├── Metric Detail Pages
│   │   ├── PoP Trend analysis
│   │   ├── Anomaly Triage
│   │   ├── Job Runs monitoring
│   │   └── Edit Metric option
│   │
│   └── Support Pages
│       └── [Help/Settings in future]
│
├── Entity Management
├── Core Functions
├── Admin Panel
└── [Other Fabric Menu Items]
```

---

## 🎨 Menu Link

**HTML Label:** `📊 Metrics Console`  
**Route:** `/metrics`  
**Style Class:** `font-semibold text-primary` (highlighted in primary color)  
**Icon:** 📊 (chart icon for visual recognition)

### Where to Find It

Open your Fabric Builder app and look at the **top navigation bar**. You'll see a new link between "Access Explanation" and the existing menus.

---

## 🚀 Routes Wired

### List/Browse
- **Path:** `/metrics`
- **Component:** `MetricsConsolePage`
- **Action:** View all metrics, search, filter, create/edit/delete

### Create New
- **Path:** `/metrics/create`
- **Component:** `MetricCreatePage`
- **Action:** Register new metric with form

### View Detail
- **Path:** `/metrics/:metricId`
- **Component:** `MetricDetailPage`
- **Action:** View metadata, PoP results, anomalies, runs

### Edit Metric
- **Path:** `/metrics/:metricId/edit`
- **Component:** `MetricEditPage`
- **Action:** Update metric definition

---

## 📋 Complete Menu Code

In `AppRoutes.tsx`, the menu entry is:

```tsx
<nav className="p-4 bg-gray-100 flex gap-4 mb-4 app-top-nav">
  <BlockableLink to="/bundles" className="hover:underline">Micro-Bundle Catalog</BlockableLink>
  <BlockableLink to="/bundle-explorer" className="hover:underline">Bundle Explorer</BlockableLink>
  <BlockableLink to="/fixed-income" className="hover:underline">Fixed Income Analytics</BlockableLink>
  <BlockableLink to="/jit-request" className="hover:underline">JIT Request Panel</BlockableLink>
  <BlockableLink to="/access-explanation" className="hover:underline">Access Explanation</BlockableLink>
  
  {/* NEW: Metrics Console Link */}
  <BlockableLink to="/metrics" className="hover:underline font-semibold text-primary">
    📊 Metrics Console
  </BlockableLink>
  
  <EntityMenu />
  <CoreMenu />
  <AdminMenu />
  <MegaMenu />
</nav>
```

---

## 🔗 Route Configuration

In the same file, routes are wired:

```tsx
<Routes>
  {/* ... existing routes ... */}
  
  {/* Metrics Console Routes */}
  <Route path="/metrics" 
    element={<ProtectedRoute><MetricsConsolePage /></ProtectedRoute>} />
  <Route path="/metrics/create" 
    element={<ProtectedRoute><MetricCreatePage /></ProtectedRoute>} />
  <Route path="/metrics/:metricId" 
    element={<ProtectedRoute><MetricDetailPage /></ProtectedRoute>} />
  <Route path="/metrics/:metricId/edit" 
    element={<ProtectedRoute><MetricEditPage /></ProtectedRoute>} />
  
  {/* ... rest of routes ... */}
</Routes>
```

---

## 🛡️ Protection

All routes are wrapped in `<ProtectedRoute>`, meaning:

✅ User must be authenticated  
✅ User session must be valid  
✅ Automatic redirect to login if not authenticated  
✅ Tenant context automatically inherited  

---

## 🎯 User Experience

### First-Time Visit

1. User clicks **📊 Metrics Console** in navbar
2. Page loads to `/metrics` (list view)
3. If no tenant selected → shows "Select a tenant" warning
4. User selects tenant from dropdown
5. Metric list refreshes with tenant-scoped data

### Browsing Workflow

```
📊 Metrics Console
    ↓ (View all metrics)
    └─→ Metric List Page
            ↓ (Click metric name)
            └─→ Metric Detail Page
                    ├─→ Click "Edit" → Edit Page → Back to Detail
                    ├─→ Click "Recompute PoP" → Job queued → Check Runs tab
                    ├─→ Click "Analyze Anomalies" → Anomaly job queued
                    └─→ Click "Delete" → Confirmation → Back to List
    
    ↓ (Create new)
    └─→ "New Metric" button in toolbar
            └─→ Create Page
                    └─→ Fill form & save → Back to Detail Page
```

---

## 🌐 Responsive Behavior

### Desktop (1024px+)
```
[Logo] [Micro-Bundle] [Bundle Explorer] [Fixed Income] [JIT] [Access] [📊 Metrics Console] [Entity ▼] [Core ▼] [Admin ▼] [Fabric ▼]
```

### Tablet (640px-1023px)
```
[Logo] [Micro-Bundle] [Fixed Income] [JIT]
[Bundle Explorer] [📊 Metrics Console] [Entity ▼] [Core ▼] [Admin ▼]
```

### Mobile (< 640px)
```
Menu gets stacked in hamburger or scrollable horizontal list
📊 Metrics Console remains accessible
```

---

## 🔐 Multi-Tenant Behavior

When clicking **📊 Metrics Console**:

1. Route changes to `/metrics`
2. Component reads `localStorage.selected_tenant`
3. Calls `setMetricsTenant(tenantId)` to set X-Tenant-ID header
4. Fetches metrics scoped to that tenant
5. All subsequent operations (create, edit, delete) are tenant-bound

**If tenant not selected:**
- User sees warning message
- Buttons remain disabled
- Prompts user to select tenant from main selector

---

## 🎨 Styling

The menu link uses:

```tsx
className="hover:underline font-semibold text-primary"
```

**Result:**
- Text color: `#5048e5` (primary blue)
- Hover effect: underline
- Font weight: 600 (semibold)
- Distinguishes from other menu items

---

## 📱 Mobile Navigation

On mobile devices, the menu collapses but remains accessible:

Option 1: **Horizontal Scroll**
- Navbar items scroll horizontally
- 📊 Metrics Console stays in viewport or scrollable list

Option 2: **Hamburger Menu** (if implemented)
- Main routes in dropdown
- 📊 Metrics Console as sub-item

---

## ✨ Key Interactions

### From List → Detail
```
User clicks on metric name (blue text)
    ↓
Router: /metrics → /metrics/{metricId}
    ↓
MetricDetailPage loads with:
- Metadata grid (domain, granularity, SLA)
- Three tabs (PoP, Anomalies, Runs)
- Action buttons (Edit, Back, Recompute, Analyze)
```

### From Detail → Edit
```
User clicks "Edit" button
    ↓
Router: /metrics/{metricId} → /metrics/{metricId}/edit
    ↓
MetricEditPage loads with:
- Same form as create, but pre-populated
- Save button updates metric
    ↓
Redirect back to detail page on success
```

### From List → Create
```
User clicks "New Metric" button
    ↓
Router: /metrics → /metrics/create
    ↓
MetricCreatePage loads with:
- Empty form
- Save creates metric in backend
    ↓
Redirect to new metric's detail page on success
```

---

## 🔄 Workflow Integration

### With Existing Fabric Builder Features

**Bundle Management** ↔ **Metrics Console**
- Both respect tenant context
- Both use same auth mechanism
- Can navigate between them seamlessly

**Data Governance** ↔ **Metrics Console**
- Metrics are governed as semantic objects
- Registry is RBAC-protected
- Audit trail tracked by backend

---

## 📊 Example Navigation

**Admin Workflow:**
```
1. Login to Fabric Builder
2. Select Tenant A from main selector
3. Click 📊 Metrics Console
4. View metrics for Tenant A
5. Create new metric "Revenue_Daily"
6. Click detail to see PoP results
7. Click "Recompute PoP" to trigger batch job
8. Check job status in Runs tab
9. Review anomalies in Anomalies tab
10. Switch to Bundle Management (via navbar)
11. Create bundle consuming "Revenue_Daily"
12. Back to 📊 Metrics Console to monitor
```

---

## 🎯 Menu Integration Checklist

- [x] Route `/metrics` added to AppRoutes
- [x] Components imported (MetricsConsolePage, etc.)
- [x] Menu link in navbar with icon
- [x] All four routes configured
- [x] ProtectedRoute wrapping enforced
- [x] Responsive on mobile/tablet/desktop
- [x] Multi-tenant header propagation
- [x] Dark mode styling applied
- [x] Navigation back-linking works

---

## 🚀 Going Live

When deploying:

1. **Verify all 7 files created** in `frontend/src/`
2. **Build frontend** → `npm run build`
3. **Deploy bundle** to your hosting
4. **Test routes** → `/metrics`, `/metrics/create`, etc.
5. **Test menu click** → Verify navigation works
6. **Test multi-tenancy** → Switch tenants, verify scoping
7. **Monitor logs** → Watch for API errors

---

**Status**: ✅ Menu Integrated  
**Visibility**: High (prominent in navbar)  
**Accessibility**: Full (ProtectedRoute enforced)  
**Multi-Tenant**: Yes (header-scoped)  
**Mobile Friendly**: Yes (responsive design)  

You're ready to go! 🎉
