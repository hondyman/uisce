# 🎨 Visual Preview - What You'll See

## Tenant List Page (`/tenants`)

```
════════════════════════════════════════════════════════════════
                         TENANT LIST PAGE
════════════════════════════════════════════════════════════════

Tenants
Manage your organization's tenants, configurations, and instance
hierarchy.

                                          [+ NEW TENANT]

┌──────────────────────────────────────────────────────────────┐
│ 🔍 Filter by name, ID, or region...                         │
│ [Status: All ▼] [Region: All ▼] [≡ Sort]                    │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ TENANT NAME / ID          │ STATUS    │ INSTANCES │ REGION    │
├──────────────────────────────────────────────────────────────┤
│ • Acme Corp North America │ ● Active  │ [3]       │ US-East   │
│   tnt-8492-xf3            │           │ Instances │ (N. Virg) │
│                           │           │           │           │
│   🔍 ✎ ✕                  │           │           │           │
├──────────────────────────────────────────────────────────────┤
│ • Acme Corp Europe        │ ● Active  │ [1]       │ EU-West   │
│   tnt-3921-qa9            │           │ Instance  │ (Ireland) │
│                           │           │           │           │
│   🔍 ✎ ✕                  │           │           │           │
├──────────────────────────────────────────────────────────────┤
│ • Acme Asia Pacific       │ ◐ Maint.  │ [5]       │ AP-SE     │
│   tnt-1102-az4            │           │ Instances │ (Singapore)│
│                           │           │           │           │
│   🔍 ✎ ✕                  │           │           │           │
├──────────────────────────────────────────────────────────────┤
│ • Beta Limited Staging    │ ○ Inactive│ [2]       │ US-West   │
│   tnt-5599-st1            │           │ Instances │ (Oregon)  │
│                           │           │           │           │
│   🔍 ✎ ✕                  │           │           │           │
├──────────────────────────────────────────────────────────────┤
│ Showing 1 to 4 of 24 results        [Previous] [Next ➤]     │
└──────────────────────────────────────────────────────────────┘

════════════════════════════════════════════════════════════════
```

### Key UI Elements:
- 🔍 = View Details icon
- ✎ = Edit icon  
- ✕ = Delete icon
- ● = Active status (green)
- ◐ = Maintenance status (yellow)
- ○ = Inactive status (gray)
- [3] = Count badge

---

## Tenant Detail Page (`/tenants/tnt-8492-xf3`)

```
════════════════════════════════════════════════════════════════
                    TENANT DETAIL PAGE
════════════════════════════════════════════════════════════════

Home / Tenants / Acme Corp North America

┌──────────────────────────────────────────────────────────────┐
│                                                               │
│  Acme Corp North America  [GOLD COPY]                        │
│                                                               │
│  Primary tenant for NA operations handling retail data,      │
│  logistics coordination, and direct-to-consumer sales        │
│  analytics. Connected to the central data lake via secure    │
│  gateway.                                                     │
│                                                               │
│  🔑 tnt-8492-xf3   📅 Jan 12, 2023    Status: Active         │
│                                                               │
│                                            [Edit] [Delete]    │
│                                                               │
└──────────────────────────────────────────────────────────────┘

┌─ Instances (3) ─┬─ Connections ─┬─ Audit Log ─┬─ Configuration ─┐
│                  │                │             │                 │
├──────────────────────────────────────────────────────────────┤
│ Associated Instances (3 Active)                              │
│ [≡ Filter] .................................. [+ Add Instance] │
├──────────────────────────────────────────────────────────────┤
│ INSTANCE NAME     │ PRODUCT    │ ENV  │ STATUS │ CONN │      │
├──────────────────────────────────────────────────────────────┤
│ • ERP Production  │ [S] SAP    │ Prod │ ● Actv │ 12 S │ ✎ ✕ │
│   inst-sap-001    │ S/4HANA    │      │        │      │      │
│                   │            │      │        │      │      │
├──────────────────────────────────────────────────────────────┤
│ • CRM Staging     │ [SF] SF    │ Stg  │ ◐ Maint│ 4 S  │ ✎ ✕ │
│   inst-sf-294     │ Salesforce │      │        │      │      │
│                   │            │      │        │      │      │
├──────────────────────────────────────────────────────────────┤
│ • Marketing Data  │ [SQL] SQL  │ Dev  │ ○ Offline│ 0 S │ ✎ ✕ │
│   inst-sql-902    │ Custom SQL │      │        │      │      │
│                   │            │      │        │      │      │
├──────────────────────────────────────────────────────────────┤
│ Rows per page: 10 ▼    1-3 of 3    [◀] [▶]                  │
└──────────────────────────────────────────────────────────────┘

════════════════════════════════════════════════════════════════
```

### Key UI Elements:
- Breadcrumb at top for navigation
- Professional header card with metadata
- Active tab highlighted with underline
- Instance table with color-coded status
- [S], [SF], [SQL] = Product avatars (colored circles)
- ● = Active (green)
- ◐ = Maintenance (yellow)
- ○ = Offline (gray)

---

## Dialog: Create/Edit Tenant

```
╔════════════════════════════════════════════════════════════╗
║                  Create New Tenant                         ║
╠════════════════════════════════════════════════════════════╣
║                                                             ║
║  Display Name *                                             ║
║  ┌──────────────────────────────────────────────────────┐  ║
║  │ Acme Corporation Europe                              │  ║
║  └──────────────────────────────────────────────────────┘  ║
║                                                             ║
║  Description                                                ║
║  ┌──────────────────────────────────────────────────────┐  ║
║  │ Primary operational hub for European operations     │  ║
║  │ with focus on regulatory compliance and data       │  ║
║  │ sovereignty requirements.                          │  ║
║  │                                                    │  ║
║  └──────────────────────────────────────────────────────┘  ║
║                                                             ║
║  ☑ Active                                                   ║
║                                                             ║
║                                   [Cancel] [Save Changes]  ║
╚════════════════════════════════════════════════════════════╝
```

---

## Dialog: Add Instance

```
╔════════════════════════════════════════════════════════════╗
║                      Add Instance                          ║
╠════════════════════════════════════════════════════════════╣
║                                                             ║
║  Instance Name *                                            ║
║  ┌──────────────────────────────────────────────────────┐  ║
║  │ ERP Production                                       │  ║
║  └──────────────────────────────────────────────────────┘  ║
║                                                             ║
║  Display Name *                                             ║
║  ┌──────────────────────────────────────────────────────┐  ║
║  │ Production SAP Instance                              │  ║
║  └──────────────────────────────────────────────────────┘  ║
║                                                             ║
║  Description                                                ║
║  ┌──────────────────────────────────────────────────────┐  ║
║  │ Main production instance for financial operations  │  ║
║  │                                                    │  ║
║  └──────────────────────────────────────────────────────┘  ║
║                                                             ║
║  URL                                                        ║
║  ┌──────────────────────────────────────────────────────┐  ║
║  │ https://sap-prod.company.com:8080                    │  ║
║  └──────────────────────────────────────────────────────┘  ║
║                                                             ║
║  ☑ Active                                                   ║
║                                                             ║
║                                      [Cancel] [Create]    ║
╚════════════════════════════════════════════════════════════╝
```

---

## Dialog: Delete Confirmation

```
╔════════════════════════════════════════════════════════════╗
║                   Delete Tenant                            ║
╠════════════════════════════════════════════════════════════╣
║                                                             ║
║  Are you sure you want to delete "Acme Corp North         ║
║  America"? This action cannot be undone and will affect   ║
║  all associated instances and data.                        ║
║                                                             ║
║                                   [Cancel] [Delete]        ║
║                                              (red button)   ║
╚════════════════════════════════════════════════════════════╝
```

---

## Responsive Mobile View (`/tenants` on phone)

```
┌─────────────────────────────────┐
│ ☰                               │  ← Menu button
│                                 │
│ Tenants                          │
│ Manage your organization's...   │
│                                 │
│ [+ NEW TENANT]                  │
│                                 │
│ ┌─────────────────────────────┐ │
│ │ 🔍 Filter by name, ID...    │ │
│ │ [Status] [Region] [Sort]    │ │
│ │ (buttons wrap/scroll)       │ │
│ └─────────────────────────────┘ │
│                                 │
│ ┌─────────────────────────────┐ │
│ │ Acme Corp North America     │ │
│ │ tnt-8492-xf3                │ │
│ │ Active  │ 3 Instances       │ │
│ │ US-East │ Jan 12, 2023      │ │
│ │                             │ │
│ │ [View] [Edit] [Delete]      │ │
│ └─────────────────────────────┘ │
│                                 │
│ ┌─────────────────────────────┐ │
│ │ Acme Corp Europe            │ │
│ │ tnt-3921-qa9                │ │
│ │ Active  │ 1 Instance        │ │
│ │ EU-West │ Mar 04, 2023      │ │
│ │                             │ │
│ │ [View] [Edit] [Delete]      │ │
│ └─────────────────────────────┘ │
│                                 │
│ ← Page 1 of 3 →                 │
│ [Prev] [Next]                   │
│                                 │
└─────────────────────────────────┘
```

---

## Color Scheme

### Status Indicators
- 🟢 **Active**: Green (#4caf50)
- 🟡 **Maintenance**: Yellow (#ffa726)
- ⚪ **Inactive**: Gray (#bdbdbd)

### Button Colors
- 🔵 **Primary Actions**: Blue (#0d7ff2) - New, Save, Create
- ⚫ **Secondary Actions**: Gray - Cancel, Close, Filter
- 🔴 **Destructive Actions**: Red - Delete

### Text Colors
- **Primary Text**: Dark gray (#111418)
- **Secondary Text**: Light gray (#60758a)
- **Links**: Blue (#0d7ff2)

### Backgrounds
- **Cards**: White (#ffffff)
- **Hover Rows**: Light gray (#f5f5f5)
- **Inputs**: Very light gray (#f0f2f5)

---

## Typography Hierarchy

```
Tenants                                    ← Heading 4 (h4)
Manage your organization's tenants...      ← Body 1 (regular)

Associated Instances (3 Active)             ← Heading 6 (h6)
inst-sap-001                                ← Caption (monospace)
ERP Production                              ← Subtitle 2 (medium)
```

---

## Interaction Feedback

### Hover Effects
- **Rows**: Background color changes to light gray
- **Links**: Text color changes to blue, underline appears
- **Buttons**: Opacity increases or background color darkens
- **Icons**: Color changes on hover

### Click Feedback
- **Buttons**: Scale slightly smaller momentarily
- **Checkboxes**: Toggle animation
- **Dialogs**: Fade in with modal overlay
- **Tabs**: Smooth underline animation

### Loading States
- Circular spinner appears
- Button becomes disabled
- Table shows loading skeleton (optional enhancement)

---

## Dark Mode Support (if enabled)

```
Background: #101922 (very dark blue)
Surfaces: #111418 (dark blue)
Text: White (#ffffff)
Secondary Text: Light gray (#9ca3af)
Cards: Slightly lighter dark blue
Borders: Dark gray (#2a3441)
```

---

## Accessibility Features

✅ **Keyboard Navigation**
- Tab through all interactive elements
- Enter to activate buttons/links
- Escape to close dialogs
- Arrow keys in dropdowns

✅ **Screen Readers**
- All buttons have descriptive labels
- Icons have title attributes
- Form inputs have labels
- Table headers are semantic

✅ **Focus Indicators**
- Blue outline on focused elements
- Clear tab order
- Focus maintained through interactions

✅ **Color Contrast**
- WCAG AA compliant
- Not relying on color alone for meaning
- Icons supplemented with text labels

---

## Loading States

```
While fetching data:
┌─────────────────────────────┐
│                             │
│         ⏳ Loading...        │  ← Spinner in center
│                             │
└─────────────────────────────┘

While submitting form:
Button becomes:  [⏳ Saving...]  (disabled)
```

---

## Error States

```
❌ Error Loading Tenants
   Failed to load tenants: Network error

[Retry]  ← Optional button

(shown as Alert component)
```

---

**All visual elements use Material UI components for consistency and accessibility!**
