# Component Architecture & Visual Guide

## 🏗️ Component Hierarchy

```
App (AppRoutes)
├── /tenants
│   └── TenantListPage
│       ├── Header Section
│       │   ├── Title "Tenants"
│       │   └── "New Tenant" Button
│       ├── Filter & Search Card
│       │   ├── Search TextField
│       │   ├── Status Filter Button
│       │   ├── Region Filter Button
│       │   └── Sort Button
│       ├── Tenants Table Card
│       │   ├── TableHead
│       │   │   └── TableRow (headers)
│       │   │       ├── Name/ID
│       │   │       ├── Status
│       │   │       ├── Instances
│       │   │       ├── Region
│       │   │       ├── Created
│       │   │       └── Actions
│       │   ├── TableBody
│       │   │   └── TableRow (for each tenant)
│       │   │       └── [same columns]
│       │   └── TablePagination
│       ├── TenantDialog (Modal)
│       │   ├── Display Name Input
│       │   ├── Description Input
│       │   └── Save/Cancel Buttons
│       └── Delete Confirmation Dialog
│           ├── Confirmation Message
│           └── Delete/Cancel Buttons
│
└── /tenants/:tenantId
    └── TenantDetailPageV2
        ├── Breadcrumb Navigation
        │   ├── Home Link
        │   ├── Tenants Link
        │   └── Current Tenant
        ├── Tenant Header Card
        │   ├── Tenant Title
        │   ├── Tier Badge
        │   ├── Description
        │   ├── Metadata Display
        │   │   ├── Tenant ID
        │   │   ├── Created Date
        │   │   └── Status
        │   └── Actions
        │       ├── Edit Button
        │       └── Delete Button
        ├── Tab Navigation
        │   ├── Instances Tab (active)
        │   ├── Connections Tab
        │   ├── Audit Log Tab
        │   └── Configuration Tab
        ├── TabPanel 0: Instances
        │   └── InstancesTableV2
        │       ├── Header Section
        │       │   ├── Title & Count
        │       │   ├── Filter Button
        │       │   └── "Add Instance" Button
        │       ├── Table
        │       │   ├── TableHead
        │       │   │   └── Instance Name, Product, Env, Status, Connections, Actions
        │       │   ├── TableBody
        │       │   │   └── Instance Rows
        │       │   └── Footer
        │       │       ├── Row Count
        │       │       └── Pagination Buttons
        │       └── Delete Confirmation Dialog
        ├── TabPanel 1: Connections (placeholder)
        ├── TabPanel 2: Audit Log (placeholder)
        ├── TabPanel 3: Configuration (placeholder)
        ├── Instance Dialog (Modal)
        │   ├── Instance Name Input
        │   ├── Display Name Input
        │   ├── Description Input
        │   ├── URL Input
        │   ├── Active Toggle
        │   └── Create/Update Button
        └── Delete Tenant Confirmation Dialog
            ├── Confirmation Message
            └── Delete/Cancel Buttons
```

---

## 📊 Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Interface Layer                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  TenantListPage              TenantDetailPageV2                 │
│      │                            │                              │
│      └─ Search/Filter/Paginate    └─ Edit/Tab Navigation        │
│      │                                 │                         │
│      └─ Click "New Tenant"             ├─ InstancesTableV2     │
│      └─ Click "View Details"           │   └─ Add/Edit/Delete  │
│      └─ Click "Edit"                   │                        │
│      └─ Click "Delete"                 └─ Future Tabs           │
│                                                                  │
└───────────────────────────────────────┬───────────────────────┘
                                        │
                                        ▼
                ┌───────────────────────────────────────────┐
                │      Apollo Client State Management       │
                ├───────────────────────────────────────────┤
                │                                            │
                │  useQuery(GET_TENANTS)                    │
                │  useQuery(GET_SCOPED_TENANT)              │
                │  useMutation(CREATE_TENANT)               │
                │  useMutation(UPDATE_TENANT)               │
                │  useMutation(DELETE_TENANT)               │
                │  useMutation(CREATE_TENANT_INSTANCE)      │
                │  useMutation(UPDATE_TENANT_INSTANCE)      │
                │  useMutation(DELETE_TENANT_INSTANCE)      │
                │                                            │
                └───────────────────────────────────────────┘
                                        │
                                        ▼
                ┌───────────────────────────────────────────┐
                │        GraphQL API Layer                  │
                ├───────────────────────────────────────────┤
                │                                            │
                │  Queries:  GET_TENANTS,                   │
                │            GET_SCOPED_TENANT              │
                │                                            │
                │  Mutations: CREATE/UPDATE/DELETE_TENANT   │
                │             CREATE/UPDATE/DELETE_         │
                │             TENANT_INSTANCE               │
                │                                            │
                └───────────────────────────────────────────┘
                                        │
                                        ▼
                ┌───────────────────────────────────────────┐
                │      Backend API Server                   │
                ├───────────────────────────────────────────┤
                │                                            │
                │  Tenants Endpoints                        │
                │  Instances Endpoints                      │
                │  (Respects X-Tenant-ID headers)          │
                │                                            │
                └───────────────────────────────────────────┘
                                        │
                                        ▼
                ┌───────────────────────────────────────────┐
                │      Database                             │
                └───────────────────────────────────────────┘
```

---

## 🎨 UI Layout Reference

### **Tenant List Page Layout**

```
┌────────────────────────────────────────────────────────────┐
│  Tenants                                                    │
│  Manage your organization's tenants, configurations...     │
├────────────────────────────────────────────────────────────┤
│                                                              │
│  [New Tenant Button]                                       │
│                                                              │
├────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🔍 Filter by name, ID, or region...                 │  │
│  │ [Status: All ▼] [Region: All ▼] [Sort]              │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
├────────────────────────────────────────────────────────────┤
│  Tenant Name / ID    │ Status  │ Instances │ Region │ ...  │
├────────────────────────────────────────────────────────────┤
│  Acme Corp NA        │ Active  │ 3         │ US-E   │ 👁 ✎ ✕ │
│  Acme Corp Europe    │ Active  │ 1         │ EU-W   │       │
│  Acme Asia Pacific   │ Maint.  │ 5         │ AP-SE  │       │
│  Beta Limited        │ Inactive│ 2         │ US-W   │       │
├────────────────────────────────────────────────────────────┤
│  Showing 1 to 4 of 24  │ [Previous] [Next]                │
└────────────────────────────────────────────────────────────┘
```

### **Tenant Detail Page Layout**

```
┌────────────────────────────────────────────────────────────┐
│  Home / Tenants / Acme Corp North America                   │
├────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Acme Corp North America [GOLD COPY]                 │  │
│  │                                                      │  │
│  │ Primary tenant for NA operations...                 │  │
│  │                                                      │  │
│  │ ID: tnt-8492-xf3   Created: Jan 12, 2023  Active   │  │
│  │                                                      │  │
│  │                                   [Edit] [Delete]   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
├─ Instances Tab (3) ─ Connections ─ Audit Log ─ Config ──┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Associated Instances (3 Active)                      │  │
│  │ [Filter] ............................ [+ Add Instance]│  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ Instance Name │ Product │ Env │ Status │ Conn │     │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ ERP Prod      │ SAP S/4 │ Prod│ Active │ 12 S │ ✎ ✕ │  │
│  │ CRM Staging   │ SF      │ Stg │ Maint. │ 4 S  │     │  │
│  │ Marketing Data│ SQL     │ Dev │ Offline│ 0 S  │     │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ Rows per page: 10  [Previous] [Next]                │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└────────────────────────────────────────────────────────────┘
```

---

## 🔄 User Interaction Flows

### **Creating a New Tenant**
```
1. User sees TenantListPage
2. Click "New Tenant" button
3. TenantDialog opens (empty form)
4. Fill: Display Name, Description, etc.
5. Click Save
6. CREATE_TENANT mutation fires
7. GET_TENANTS query refetches
8. Dialog closes
9. New tenant appears in table
```

### **Viewing Tenant Details**
```
1. User on TenantListPage
2. Click tenant name (blue link)
3. Navigate to /tenants/{tenantId}
4. TenantDetailPageV2 loads
5. GET_SCOPED_TENANT query executes
6. Tenant data + instances display
7. Instances Tab is active by default
```

### **Adding an Instance**
```
1. User on Tenant Detail page (Instances tab)
2. Click "Add Instance" button
3. Instance dialog opens (empty form)
4. Fill: Name, Display Name, URL, Environment
5. Click Create
6. CREATE_TENANT_INSTANCE mutation fires
7. GET_SCOPED_TENANT query refetches
8. Dialog closes
9. New instance appears in table
```

### **Editing an Instance**
```
1. User on Tenant Detail page (Instances tab)
2. Hover over instance row
3. Click Edit icon
4. Instance dialog opens (populated form)
5. Modify fields
6. Click Update
7. UPDATE_TENANT_INSTANCE mutation fires
8. GET_SCOPED_TENANT query refetches
9. Dialog closes
10. Table updates with new values
```

### **Deleting with Confirmation**
```
1. User on list or detail page
2. Click Delete icon
3. Confirmation dialog appears
4. User clicks "Delete" button
5. DELETE mutation fires
6. Query refetches automatically
7. Item removed from view
8. Success feedback (implicit)
```

---

## 💾 State Management Pattern

### **TenantListPage State**
```tsx
const [searchQuery, setSearchQuery] = useState('');           // Search input
const [statusFilter, setStatusFilter] = useState('all');       // Filter state
const [page, setPage] = useState(0);                           // Pagination
const [rowsPerPage, setRowsPerPage] = useState(10);            // Pagination
const [tenantDialog, setTenantDialog] = useState({...});       // Create/Edit dialog
const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false); // Delete confirmation
const { loading, error, data, refetch } = useQuery(GET_TENANTS); // Apollo state
```

### **TenantDetailPageV2 State**
```tsx
const [activeTab, setActiveTab] = useState(0);                 // Tab navigation
const [editMode, setEditMode] = useState(false);               // Edit tenant toggle
const [tenantEditForm, setTenantEditForm] = useState({...});   // Tenant edit form
const [instanceDialogOpen, setInstanceDialogOpen] = useState(false); // Dialog
const [editingInstance, setEditingInstance] = useState(null);  // Which instance?
const [instanceForm, setInstanceForm] = useState({...});       // Instance form
const { loading, error, data, refetch } = useQuery(...);       // Apollo state
```

---

## 🎯 Key Design Decisions

| Aspect | Decision | Reason |
|--------|----------|--------|
| State Management | React Hooks + Apollo | Simplicity, integrates with existing setup |
| Table Library | Material UI Table | Native MUI support, no extra deps |
| Pagination | Built-in MUI | Clean, accessible, no external libs |
| Dialogs | MUI Dialog | Consistent with design system |
| Styling | sx prop + MUI theme | Single source of truth for styles |
| Data Fetching | Apollo GraphQL | Reuses existing infrastructure |
| Form Handling | Local state | Simple, no need for complex form libs |
| Responsive | Mobile-first flex | Works on all screen sizes |

---

## 🔗 Component Props & Interfaces

### **InstancesTableV2 Props**
```tsx
interface InstancesTableProps {
  instances: TenantInstance[];
  onAddInstance?: () => void;
  onEditInstance?: (instance: TenantInstance) => void;
  onDeleteInstance?: (instanceId: string) => void;
}
```

### **Tenant Type**
```tsx
interface Tenant {
  id: string;
  name?: string;
  display_name?: string;
  is_active: boolean;
  // ... other fields
  tenant_instances?: TenantInstance[];
}
```

### **TenantInstance Type**
```tsx
interface TenantInstance {
  id: string;
  instance_name?: string;
  display_name?: string;
  is_active?: boolean;
  // ... other fields
}
```

---

This architecture ensures:
- ✅ Clear separation of concerns
- ✅ Reusable components
- ✅ Predictable data flow
- ✅ Easy to test and maintain
- ✅ Scalable for future enhancements
