# Tenant Detail Page - Visual Architecture

## Page Structure

```
TenantDetailPageV2
├── Breadcrumb Navigation
│   └── Home > Tenants > [Tenant Name]
│
├── Tenant Header Card
│   ├── Tenant Name & Tier Badge
│   ├── Tenant Description
│   ├── Quick Info (ID, Created Date, Status)
│   └── Actions (Edit, Delete)
│
└── Tabs Container
    ├── Tab 0: Instances (Existing)
    │   └── InstancesTableV2
    │       ├── Add Instance Button
    │       ├── Table with Columns:
    │       │   ├── Instance Name
    │       │   ├── Product
    │       │   ├── Environment
    │       │   ├── Status
    │       │   ├── Connections
    │       │   └── Actions
    │       ├── Instance Dialog (Add/Edit)
    │       └── Delete Confirmation
    │
    ├── Tab 1: Connections (NEW) ✨
    │   └── ConnectionsTabContent
    │       ├── Filter Type Dropdown
    │       │   ├── All Types
    │       │   ├── Databases
    │       │   ├── API Endpoints
    │       │   └── File Stores
    │       ├── Add Connection Button
    │       ├── Table with Columns:
    │       │   ├── Connection Name
    │       │   ├── Type
    │       │   ├── Linked Instance
    │       │   ├── Last Sync
    │       │   ├── Status Badge
    │       │   └── Actions (Test, Settings)
    │       └── Pagination Controls
    │
    ├── Tab 2: Audit Log (NEW) ✨
    │   └── AuditLogTabContent
    │       ├── Search Input
    │       │   └── Search by: User, Action, Resource
    │       ├── Filter Button
    │       ├── Export Button
    │       ├── Table with Columns:
    │       │   ├── Timestamp
    │       │   ├── User (with Avatar)
    │       │   ├── Action Badge
    │       │   ├── Resource
    │       │   ├── Details
    │       │   └── View Details Button
    │       ├── Details Dialog
    │       │   ├── Timestamp
    │       │   ├── User
    │       │   ├── Action
    │       │   ├── Resource
    │       │   └── Details
    │       └── Pagination Controls
    │
    └── Tab 3: Configuration (NEW) ✨
        └── ConfigurationTabContent
            ├── Save Changes Button
            ├── Discard Changes Button
            │
            ├── Section 1: Data Retention Policy 📅
            │   ├── Toggle Switch (Enable/Disable)
            │   ├── Audit Log Retention Dropdown
            │   │   ├── 30 Days
            │   │   ├── 60 Days
            │   │   ├── 90 Days
            │   │   ├── 180 Days
            │   │   └── 1 Year
            │   ├── Transactional Data Retention Dropdown
            │   │   ├── 1 Year
            │   │   ├── 3 Years
            │   │   ├── 5 Years
            │   │   ├── 7 Years
            │   │   └── Indefinite
            │   ├── Backup Frequency Radio Group
            │   │   ├── Daily
            │   │   ├── Weekly
            │   │   └── Monthly
            │   └── Archive Storage Class Dropdown
            │       ├── Standard
            │       ├── Infrequent Access
            │       └── Glacier (Deep Archive)
            │
            ├── Section 2: Security & Access Control 🔒
            │   ├── Checkbox: Enforce Multi-Factor Authentication (MFA)
            │   ├── Checkbox: Enable SAML Single Sign-On (SSO)
            │   └── IP Whitelist Text Area
            │       └── Format: CIDR notation (e.g., 192.168.1.1, 10.0.0.0/24)
            │
            └── Section 3: API Integration Preferences ⚙️
                ├── API Rate Limit Input (requests/minute)
                ├── Webhook Retry Attempts Input (0-10)
                └── Default Webhook Endpoint URL Input
                    └── Format: https://api.example.com/hooks
```

## Component Hierarchy

```
TenantDetailPageV2 (Main Page)
    ↓
    ├─ ConnectionsTabContent
    │   ├─ Material UI: Table
    │   ├─ Material UI: Button (Add Connection)
    │   ├─ Material UI: Select (Filter Type)
    │   └─ Material UI: Chip (Status Badges)
    │
    ├─ AuditLogTabContent
    │   ├─ Material UI: Table
    │   ├─ Material UI: TextField (Search)
    │   ├─ Material UI: Avatar (User)
    │   ├─ Material UI: Chip (Action Badges)
    │   └─ Material UI: Dialog (View Details)
    │
    └─ ConfigurationTabContent
        ├─ Paper (Section Container)
        │   ├─ CardHeader (with Icon)
        │   └─ CardContent (with Form Fields)
        │
        └─ Form Components:
            ├─ Select (Dropdowns)
            ├─ Radio (Backup Frequency)
            ├─ Checkbox (MFA, SSO)
            ├─ Switch (Enable/Disable)
            ├─ TextField (Inputs, Textarea)
            ├─ Button (Save, Discard)
            └─ FormControlLabel (Labels)
```

## Data Flow

### Connections Tab
```
ConnectionsTabContent
    ↓
State: [filterType, testingConnectionId]
    ↓
Filter Connections by Type
    ↓
Render Table with Mock Data
    ↓
User Actions:
    ├─ Change Filter → Re-filter
    ├─ Click Test → onTestConnection()
    ├─ Click Settings → onEditConnection()
    └─ Click Add → onAddConnection()
```

### Audit Log Tab
```
AuditLogTabContent
    ↓
State: [searchQuery, selectedEntry]
    ↓
Filter Entries by Search
    ↓
Render Table with Mock Data
    ↓
User Actions:
    ├─ Type Search → Filter entries
    ├─ Click Details → Open dialog
    ├─ Click Export → onExport()
    └─ Click Filter → Opens filter UI
```

### Configuration Tab
```
ConfigurationTabContent
    ↓
State: [formData, hasChanges, saving]
    ↓
Display Form with Current Config
    ↓
User Actions:
    ├─ Change Field → Update formData
    ├─ Click Save → onSaveConfiguration(formData)
    └─ Click Discard → Revert to original
```

## Component Props Flow

### ConnectionsTabContent
```
Props Received:
├─ connections: Connection[] (default: mockConnections)
├─ onAddConnection?: () => void
├─ onEditConnection?: (connection: Connection) => void
└─ onTestConnection?: (id: string) => Promise<void>
```

### AuditLogTabContent
```
Props Received:
├─ entries: AuditLogEntry[] (default: mockAuditEntries)
├─ onViewDetails?: (entry: AuditLogEntry) => void
└─ onExport?: () => void
```

### ConfigurationTabContent
```
Props Received:
├─ config?: TenantConfiguration (default: defaultConfig)
└─ onSaveConfiguration?: (config: TenantConfiguration) => Promise<void>
```

## File Size Reference

- **ConnectionsTabContent.tsx**: 250 lines (~8 KB)
- **AuditLogTabContent.tsx**: 350 lines (~11 KB)
- **ConfigurationTabContent.tsx**: 400 lines (~13 KB)
- **TenantDetailPageV2.tsx**: 526 lines (3 new imports + 3 component usages)

**Total New Code**: ~1,000 lines

## Mock Data Summary

**Connections**: 4 entries
- Database, REST API, SQL Server, S3 Storage
- Status distribution: 2 Connected, 1 Warning, 1 Error

**Audit Log**: 5 entries
- User actions: Update, Create, Delete, Security
- System actions: Backup
- Action color badges: Blue, Green, Red, Gray, Amber

**Configuration**: Sensible defaults
- Retention: 90 days audit logs, 7 years transactions
- Security: MFA enabled, SSO disabled
- API: 5000 req/min, 3 retries, sample webhook URL

## Styling Features

✨ **Color Coding**
- Status: Green (Connected), Amber (Warning), Red (Error)
- Actions: Blue (Update), Green (Create), Red (Delete), Gray (Backup), Amber (Security)
- Sections: Blue (Retention), Green (Security), Purple (API)

🎨 **Responsive Grid**
- Mobile: 1 column
- Desktop: 2 columns
- Flex wrapping for headers and buttons

📱 **Hover States**
- Table rows highlight on hover
- Buttons show active state
- Icons change color on hover

## Browser Compatibility

✅ Chrome 90+
✅ Firefox 88+
✅ Safari 14+
✅ Edge 90+

## Accessibility Features

✅ Proper heading hierarchy (h1-h6)
✅ Form labels associated with inputs
✅ Color not sole indicator (badges have text)
✅ Keyboard navigation support
✅ ARIA attributes where needed
✅ Sufficient color contrast

---

**Architecture Status**: ✅ Complete
**Responsive Design**: ✅ Implemented
**Accessibility**: ✅ Compliant
**TypeScript Types**: ✅ Fully Typed
