# Entity Schema Builder v2 - Quick Start & Visual Tour

## 🎬 Quick Demo (2 Minutes)

### Step 1: Open the Definition Tab
Navigate to: `http://localhost:5173/config`

You should see:
```
┌─────────────────────────────────────────────────────────┐
│ 🟢 Definitions | Entity Schema Builder                 │
│ (Workday-Style BOs)                                     │
│                                                         │
│ Search: [Search entities, descriptions, subtypes...]  │
│                                                         │
│ [SAVE & APPLY (0)]                                     │
└─────────────────────────────────────────────────────────┘
```

### Step 2: Scroll Through Core BOs

You'll see three core business objects as cards:

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ 🔒 CORE BO      │  │ 🔒 CORE BO      │  │ 🔒 CORE BO      │
│                 │  │                 │  │                 │
│ ClientInvestor  │  │ Portfolio       │  │ Trade           │
│ Investor        │  │ Asset portfolio │  │ Security        │
│ profile with    │  │ management      │  │ transaction     │
│ relationships   │  │                 │  │                 │
│                 │  │                 │  │                 │
│ IndividualInves │  │ DiscretionaryPo │  │ RegularTrade    │
│ InstitutionalIn │  │                 │  │ BlockTrade      │
│                 │  │                 │  │                 │
│ 5 Fields        │  │ 4 Fields        │  │ 5 Fields        │
│ 2 Subtypes      │  │ 1 Subtype       │  │ 2 Subtypes      │
│                 │  │                 │  │                 │
│ [✏️] [🔄] [🗑️] │  │ [✏️] [🔄] [🗑️] │  │ [✏️] [🔄] [🗑️] │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### Step 3: Clone a Core BO

**Action:** Click 🔄 (clone icon) on **ClientInvestor** card

**Result:** 
- New card appears: "ClientInvestor (Custom)" ✏️ CUSTOM
- All 5 core fields automatically copied
- Both subtypes copied with their fields
- Ready to customize

**Check Changes:** SAVE & APPLY button now shows **(1)** - 1 change detected

### Step 4: Edit the Cloned Entity

**Action:** Click ✏️ (edit icon) on "ClientInvestor (Custom)"

**Drawer Opens:**
```
┌─ Edit Entity: ClientInvestor (Custom) ────────────────┐
│ [SUBTYPES] [FIELDS]                                   │
│                                                        │
│ When on SUBTYPES tab:                                │
│ ┌──────────────────────────────────┐                │
│ │ + Add Subtype                    │                │
│ │                                  │                │
│ │ Name          Type    Fields      │                │
│ │ Individual    🔒 Core   2  [🗑️] │                │
│ │ Institutional 🔒 Core   2  [🗑️] │                │
│ └──────────────────────────────────┘                │
│                                                        │
│ When on FIELDS tab:                                  │
│ ┌──────────────────────────────────┐                │
│ │ + Add Field                      │                │
│ │                                  │                │
│ │ 🔒 CORE FIELDS (INHERITED):      │                │
│ │ • investor_id (text)             │                │
│ │ • legal_name (text)              │                │
│ │ • email (text)                   │                │
│ │ • phone (text)                   │                │
│ │ • aum (number)                   │                │
│ │                                  │                │
│ │ ✏️ CUSTOM FIELDS:                │                │
│ │ (empty - add your own)           │                │
│ └──────────────────────────────────┘                │
└────────────────────────────────────────────────────────┘
```

### Step 5: Add a Custom Field

**Action:** Click "+ Add Field" button

**Modal Opens:**
```
┌─ Add Field ────────────────────────────────────────┐
│                                                    │
│ Field Name:     [_______________________]          │
│                 Example: "esg_focus_areas"         │
│                                                    │
│ Field Type:     [▼ text]                          │
│                 • text                            │
│                 • number                          │
│                 • date                            │
│                 • boolean                         │
│                                                    │
│ Add to:         [▼ Entity Level]                  │
│                 • Entity Level                    │
│                 • Subtype: IndividualInvestor     │
│                 • Subtype: InstitutionalInvestor  │
│                                                    │
│             [OK] [CANCEL]                         │
└────────────────────────────────────────────────────┘
```

**Fill in:**
- Field Name: `esg_focus_areas`
- Field Type: `text`
- Add to: `Entity Level`

**Click OK**

**Result:** Drawer refreshes, new field appears in ✏️ CUSTOM FIELDS section

### Step 6: Save Changes

**Action:** Close drawer, click **SAVE & APPLY (1)**

**Network Request Sent:**
```json
POST /api/entity-schema

{
  "changed": {
    "client_investor_custom_1": {
      "name": "ClientInvestor (Custom)",
      "description": "Custom clone of ClientInvestor",
      "isCore": false,
      "clonesFrom": "client_investor",
      "coreFields": [
        { "key": "investor_id", "name": "Investor ID", "type": "text", "isCore": true },
        // ... more core fields
      ],
      "customFields": [
        { "key": "esg_focus_areas", "name": "esg_focus_areas", "type": "text", "isCore": false }
      ],
      // ... full entity structure
    }
  },
  "deleted": []
}

Response: 200 OK
{
  "success": true,
  "message": "Entity schema saved successfully"
}
```

**Result:** 
- ✅ Toast: "✅ Saved! 1 changed, 0 deleted"
- SAVE & APPLY button disabled (no pending changes)
- Data persisted to backend

### Step 7: Refresh & Verify Persistence

**Action:** Press F5 (refresh page)

**Result:** 
- Page loads
- Fetches schema from backend
- "ClientInvestor (Custom)" still there with `esg_focus_areas` field
- ✅ Data persisted!

---

## 🎯 Feature Walkthrough

### Feature 1: Search

**Action:** Type "individual" in search box

**Result:** 
- Cards filtered to show only entities containing "individual" in name/subtypes
- Other cards hidden
- Matches: ClientInvestor (has IndividualInvestor subtype) + Portfolio, Trade are hidden

**Action:** Clear search (click X)

**Result:** All entities reappear

### Feature 2: Add New Entity

**Action:** Click "➕ ADD NEW ENTITY" card

**Modal Opens:**
```
┌─ Create New Entity ────────────────────────────┐
│                                                │
│ Entity Name:      [___________________]        │
│                    e.g., "Order"               │
│                                                │
│ Description:      [___________________]        │
│                    (optional)                  │
│                    e.g., "Customer orders"    │
│                                                │
│              [CREATE] [CANCEL]                 │
└────────────────────────────────────────────────┘
```

**Fill in:**
- Entity Name: `Order`
- Description: `Customer purchase orders`

**Click CREATE**

**Result:**
- New card appears: "Order" ✏️ CUSTOM
- Description shown below name
- (No subtypes yet, 0 Fields)
- Ready to edit

**Check Changes:** SAVE & APPLY shows **(2)** - 2 changes (cloned + new)

### Feature 3: Add Subtypes

**Action:** Click ✏️ on "Order" entity

**Drawer Opens on SUBTYPES tab**

**Action:** Click "+ Add Subtype"

**Modal Opens:**
```
┌─ Add Subtype ──────────────────────────┐
│                                        │
│ Subtype Name: [_____________]          │
│               e.g., "PurchaseOrder"    │
│                                        │
│                   [OK] [CANCEL]        │
└────────────────────────────────────────┘
```

**Fill in:** `PurchaseOrder`

**Click OK**

**Drawer refreshes:** Table now shows:
```
│ Name          Type     Fields   Actions │
│ PurchaseOrder ✏️ Custom  0     [🗑️]  │
```

**Repeat:** Add another subtype `SalesOrder`

**Result:**
```
│ Name          Type     Fields   Actions │
│ PurchaseOrder ✏️ Custom  0     [🗑️]  │
│ SalesOrder    ✏️ Custom  0     [🗑️]  │
```

### Feature 4: Add Fields to Subtype

**Action:** Still in drawer, click "+ Add Field"

**Modal Opens with "Add to" dropdown:**
```
Add to: [▼ Entity Level]
  • Entity Level
  • Subtype: PurchaseOrder
  • Subtype: SalesOrder
```

**Fill in:**
- Field Name: `order_number`
- Field Type: `text`
- Add to: `Entity Level`

**Click OK**

**Result:** Field added to Entity level

**Repeat for subtype field:**
- "+ Add Field"
- Field Name: `po_number`
- Field Type: `text`
- Add to: `Subtype: PurchaseOrder`
- OK

**Result:** Drawer shows:
```
FIELDS Tab:

+ Add Field

📌 ENTITY FIELDS:
• order_number (text)

One or more subtype fields visible if added
```

### Feature 5: Delete Entity

**Action:** Go back to main list (close drawer)

**Action:** Click 🗑️ on any custom entity card

**Confirmation Dialog:**
```
┌─ Delete Entity? ────────────────────┐
│ This will delete the entity and    │
│ all its subtypes/fields.           │
│                                    │
│           [DELETE] [CANCEL]        │
└────────────────────────────────────┘
```

**Click DELETE**

**Result:**
- Card removed from list
- SAVE & APPLY updated
- Entity added to "deleted" array
- Ready to persist with SAVE & APPLY

---

## 📊 Visual Reference: Color Codes

### Entity Type Badges
```
🔒 CORE BO  → Blue badge     (Workday-delivered, read-only)
✏️ CUSTOM   → Green badge    (User-created or cloned, editable)
✏️ Clone    → Orange badge   (Cloned from core BO)
```

### Field Type Colors
```
Core Fields:   🔒 Blue tag        (Inherited from template)
Custom Fields: ✏️ Green tag       (Tenant-specific additions)
Inherited:     Gray tag          (From parent entity)
```

### Subtype Colors
```
Core Subtype:   Blue (🔒 Core)      (From template)
Custom Subtype: Cyan (✏️ Custom)    (Created by user)
```

---

## 🚀 Example: Build a Complete Custom BO

### Goal: Create "Investment Advisor" Custom BO

**Step 1: Clone Worker (Worker BO exists as core template)**
```
Click 🔄 on Worker
→ Creates "worker_custom_1"
```

**Step 2: Customize**
```
Click ✏️ on "worker_custom_1"
→ Tab: Subtypes
  → Add Subtype: "FinancialAdvisor"
  → Add Subtype: "PortfolioManager"

→ Tab: Fields
  → Add Field: "finra_certification" (text) at Entity Level
  → Add Field: "aum_under_management" (number) at Entity Level
  → Add Field: "series_7_date" (date) in FinancialAdvisor subtype
  → Add Field: "cfa_charter" (boolean) in PortfolioManager subtype
```

**Step 3: Save**
```
Click SAVE & APPLY
→ Backend stores complete structure
→ Upgrade safe: all "isCore: true" fields preserved
→ Custom fields protected from future updates
```

**Result JSON (simplified):**
```json
{
  "worker_custom_1": {
    "name": "Investment Advisor",
    "description": "Custom advisor profile",
    "isCore": false,
    "clonesFrom": "worker",
    "coreFields": [
      { "key": "worker_id", "type": "text", "isCore": true },
      { "key": "name", "type": "text", "isCore": true },
      { "key": "email", "type": "text", "isCore": true },
      // ... more core fields from Worker template
    ],
    "customFields": [
      { "key": "finra_certification", "type": "text", "isCore": false },
      { "key": "aum_under_management", "type": "number", "isCore": false }
    ],
    "subtypes": {
      "financial_advisor": {
        "name": "FinancialAdvisor",
        "isCore": false,
        "subtype_fields": [
          { "key": "series_7_date", "type": "date", "isCore": false }
        ]
      },
      "portfolio_manager": {
        "name": "PortfolioManager",
        "isCore": false,
        "subtype_fields": [
          { "key": "cfa_charter", "type": "boolean", "isCore": false }
        ]
      }
    }
  }
}
```

---

## 🎓 Best Practices

### ✅ DO:
- ✅ Clone core BOs for tenant customization
- ✅ Add custom fields at entity level for broad attributes
- ✅ Use subtypes to represent specializations
- ✅ Add fields to subtypes for type-specific attributes
- ✅ Search before creating to avoid duplicates
- ✅ Use clear, descriptive entity/subtype names

### ❌ DON'T:
- ❌ Delete core BOs (they're templates)
- ❌ Edit core fields (they're read-only by design)
- ❌ Create duplicate entities with same purpose
- ❌ Use abbreviations (use full names for clarity)
- ❌ Forget to SAVE & APPLY after changes
- ❌ Delete entities without confirming they're not used

---

## 🔍 Troubleshooting

### Q: Changes not saving?
**A:** 
1. Check SAVE & APPLY button shows > 0 changes
2. Check browser console (F12) for errors
3. Verify tenant is selected (top-right selector)
4. Check X-Tenant-ID header in Network tab

### Q: Old entity still appears after delete?
**A:**
1. Refresh page (F5)
2. If still there, check backend logs: `docker compose logs backend | grep entity-schema`

### Q: Can't add field to subtype?
**A:**
1. Make sure subtype exists (check SUBTYPES tab)
2. Make sure you selected "Subtype: [name]" in Add Field modal
3. Check the subtype appears in the dropdown

### Q: Core fields showing as editable?
**A:**
1. They should be read-only (no delete button)
2. If you can delete them, clear browser cache and refresh
3. Check that `isCore: true` is set on the field

### Q: Tenant scope error?
**A:**
1. Select tenant from top-right selector
2. Check localStorage: `localStorage.getItem('selected_tenant')`
3. If empty, select tenant again and refresh

---

## 🎉 You're Ready!

Navigate to `/config` and start building your custom Business Objects! 🚀

**Tips:**
- Start by cloning a core BO to see how it works
- Add a few custom fields
- Create a new entity from scratch
- Explore the subtypes and inheritance model
- Check the Network tab to see the delta payload
- Refresh to verify persistence
