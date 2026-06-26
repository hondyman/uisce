# Entity Schema Builder v2.1 - Quick Reference Card

## 🎯 Core Concepts

### Business Name vs Technical Name
```
USER SEES (Business Name)          SYSTEM USES (Technical Name)
Legal Name               ←→         legal_name
Date of Birth            ←→         date_of_birth
Client Investor          ←→         client_investor
Individual Investor      ←→         individual_investor
```

**Rule:** Always lowercase, underscores between words, no special chars

---

## 📝 Creating an Entity

### Step 1: Add New Entity
```
1. Click [+ Add New Entity] card
2. Modal opens: "Add New Entity"
3. Enter: Business Name = "Client Portfolio"
   → Auto-generates: Technical Name = "client_portfolio"
4. Enter: Description (optional)
5. Click OK
```

### Step 2: Edit the Entity
```
1. Find card: "Client Portfolio" (✏️ CUSTOM badge)
2. Click [✏️ Edit]
3. See: 
   - 📋 Entity tab (fields + rename)
   - 📦 Subtypes tab (manage subtypes)
```

### Step 3: Add Entity Fields
```
1. In 📋 Entity tab, click [+ Add Field to Entity]
2. Enter:
   - Business Name: "Portfolio Value"
   - Type: number
   - Semantic Term: (optional) "Total Assets"
3. Technical Name: "portfolio_value" (auto)
4. Click OK
```

---

## 🔄 Cloning a Core BO

### Clone Process
```
CORE BO: "Client Investor" (🔒 blue)
         ↓ Click [🔄 Clone]
         ↓
CUSTOM: "Client Investor (Custom)" (✏️ green)
        ├── 🔒 CORE FIELDS (inherited, read-only)
        ├── ✏️ CUSTOM FIELDS (empty initially)
        └── 🔗 Cloned from: Client Investor
```

### Clone Then Rename
```
1. Click [🔄 Clone] on "Client Investor"
2. New entity created: "Client Investor (Custom)"
3. Click [✏️ Edit]
4. Go to 📋 Entity tab
5. In "Rename Entity" section, change name to:
   "Wealth Management Client"
   → Technical: "wealth_management_client"
6. Technical name updates automatically
```

### Clone + Add Custom Field
```
1. Clone core BO ✓
2. Edit clone ✓
3. Go to 📋 Entity tab
4. Click [+ Add Field to Entity]
5. Enter:
   - Business Name: "ESG Score"
   - Type: number
   - Semantic Term: "Environmental Score" (from catalog)
6. Field added to ✏️ CUSTOM FIELDS section
7. Save & Apply
```

---

## 📦 Managing Subtypes

### Add Subtype to Entity
```
1. Click [✏️ Edit] on entity
2. Go to 📦 Subtypes tab
3. Click [+ Add Subtype]
4. Enter: "Elite Investor"
   → Auto-generates: "elite_investor"
5. New subtype created
```

### Add Field to Subtype
```
1. In 📦 Subtypes tab
2. Find your subtype card
3. Click [+ Add Field]
4. Enter:
   - Business Name: "Min Asset Threshold"
   - Type: number
   - Semantic Term: "Investment Threshold"
5. Field added to that subtype
```

### Delete Subtype
```
1. In 📦 Subtypes tab
2. Click [🗑️] on subtype card
3. Confirm: "Delete subtype and all its fields?"
4. Subtype + fields deleted
```

---

## 🔐 Field Status Indicators

### Core Fields (Inherited)
```
🔒 CORE (blue tag)
├── Inherited from parent BO
├── Read-only in clone
├── No delete button
└── Can be deleted only at parent level
```

### Custom Fields (User-Added)
```
🔓 CUSTOM (green tag)
├── Created by user
├── Can edit or delete
├── Shows in custom only
└── Click [🗑️] to remove
```

### Semantic Term Link
```
📌 LINKED (info column)
├── Field points to semantic term
├── Example: "legal_name" → "Customer Name"
├── Shows in gray text: "Semantic: Customer Name"
└── Bridges to data catalog
```

---

## 💾 Saving Your Work

### What Triggers SAVE Button?
```
✏️ If you:
  - Add entity
  - Add subtype
  - Add field
  - Rename entity
  - Clone entity
  - Delete entity
  
⚠️ SAVE & APPLY button appears with count:
  "SAVE & APPLY (5 changes)"
```

### How to Save
```
1. Make changes (add entity, add field, etc)
2. Click [SAVE & APPLY] button (top right or floating)
3. See: "✅ Saved! 5 changed, 0 deleted"
4. Changes stored to backend
5. Reload page: changes persist ✓
```

---

## 🔍 Search Features

### Search By
```
1. Entity business name: "Client Investor"
2. Entity technical name: "client_investor"
3. Entity description: contains text
4. Subtype names: "Individual", "Elite"
5. Any part of above
```

### Search Example
```
Search bar: "elite"
Results:
  ✓ "Elite Investor" (subtype)
  ✓ "Wealth Management Client" (has Elite subtype)
```

---

## 🎨 Visual Reference

### Entity Card
```
┌───────────────────────────────┐
│ 🔒 CORE BO                    │
│                               │
│ Client Investor               │
│ Technical: client_investor    │
│                               │
│ Core BO: Investor profile...  │
│                               │
│ Subtypes:                     │
│ [Individual] [Institutional]  │
│                               │
│ [✏️ Edit][🔄 Clone]            │
└───────────────────────────────┘
```

### Entity Editor - Entity Tab
```
📋 ENTITY TAB
├── Rename Entity (CUSTOM ONLY)
│   ├── Business Name: [input]
│   └── Technical Name: [display, auto-generated]
│
├── Clone Parent (IF CLONED)
│   └── Parent BO: Client Investor (Upgradeable)
│
└── Entity Fields
    ├── [+ Add Field]
    └── Table:
        ├── Business Name | Technical | Type | Semantic | Status | [🗑️]
        ├── Legal Name    | legal_n... | text | Cust...  | 🔒    |
        └── ESG Score    | esg_score  | text | ESG...   | 🔓    | [🗑️]
```

### Entity Editor - Subtypes Tab
```
📦 SUBTYPES TAB (2)
├── [+ Add Subtype]
│
├── Card: Individual Investor (technical: individual_investor)
│   ├── 🔒 CORE
│   ├── Fields: [+ Add]
│   └── Table:
│       ├── SSN | ssn | text | - | 🔒 Core
│       └── DOB | dob | date | - | 🔒 Core
│
└── Card: Elite Investor (technical: elite_investor)
    ├── ✏️ CUSTOM [🗑️]
    ├── Fields: [+ Add]
    └── Table:
        └── Min Assets | min_assets | number | Threshold | 🔓 Custom | [🗑️]
```

---

## ⚡ Quick Tips

### Tip 1: Business Names First
Always enter business name → technical auto-generates
```
DON'T: Enter technical "legal_name" then business
DO:    Enter business "Legal Name" → auto: "legal_name"
```

### Tip 2: Clone for Customization
Don't modify core BOs. Clone them instead.
```
CORE (🔒) = read-only template
CLONE (✏️) = your customized version
```

### Tip 3: Semantic Terms Are Optional
Link to catalog terms only if you have them
```
✓ Link to terms: "Account Balance" → "Customer Account Balance"
✓ Leave blank: No semantic term in catalog yet
```

### Tip 4: Subtype Inheritance
When you clone entity, subtypes come with it
```
PARENT Entity
├── 🔒 Subtype A
└── 🔒 Subtype B
         ↓ Clone
CLONE Entity
├── 🔒 Subtype A (inherited)
├── 🔒 Subtype B (inherited)
└── ✏️ Subtype C (new - your add)
```

### Tip 5: Delete Only Custom
Core fields/subtypes can't be deleted at clone level
```
🔓 CUSTOM = has delete button [🗑️]
🔒 CORE   = no delete button
           = delete at parent BO only
```

### Tip 6: Save Counts
Count in button shows both additions AND modifications
```
"SAVE & APPLY (7)"
= 5 entities changed + 2 entities deleted
= 7 total operations
```

---

## 🔗 Clone Parent Tracking

### When You Clone
```
New entity stores:
  clonesFromKey: "client_investor" (parent's tech name)
  cloneParentName: "Client Investor" (parent's business name)
  
UI Shows:
  🔗 Cloned from: Client Investor
```

### Upgrade Path
```
If core BO updated:
  1. Core gets new fields
  2. Clone still has old structure
  3. User can manually add inherited fields
  4. Or wait for merge tool (future feature)
```

---

## 📋 Entity Structure Example

```
ENTITY: Wealth Management Client
Business: Wealth Management Client
Technical: wealth_management_client
Description: Custom clone for wealth management division
Status: ✏️ CUSTOM
Clone Parent: 🔗 Client Investor (original core BO)

ENTITY FIELDS:
├── 🔒 investor_id (inherited, core)
├── 🔒 legal_name (inherited, core)
├── 🔒 email (inherited, core)
├── 🔒 phone (inherited, core)
├── 🔒 aum (inherited, core)
└── ✏️ esg_score (custom, linked to "ESG Classification" semantic term)

SUBTYPES:
├── Individual Investor (inherited from parent)
│   ├── 🔒 ssn (core)
│   └── 🔒 date_of_birth (core)
│
├── Institutional Investor (inherited from parent)
│   ├── 🔒 ein (core)
│   └── 🔒 registration_status (core)
│
└── Elite Investor (NEW - custom subtype)
    ├── min_assets (number, links to "Investment Threshold")
    ├── relationship_manager (text)
    └── priority_support (boolean)
```

---

## ✅ Common Workflows

### Workflow 1: Clone & Customize
```
1. Find core BO [🔒 CORE BO]
2. Click [🔄 Clone]
3. Rename if needed [Edit → Entity tab]
4. Add custom fields [Edit → Entity tab → + Add]
5. Add custom subtypes [Edit → Subtypes tab → + Add]
6. Save [SAVE & APPLY]
✅ Done: Custom version ready
```

### Workflow 2: Add Subtype to Existing Entity
```
1. Edit entity [✏️ Edit]
2. Go to Subtypes tab
3. Click [+ Add Subtype]
4. Enter name & tech name (auto)
5. Add fields to subtype [+ Add Field]
6. Save [SAVE & APPLY]
✅ Done: Subtype with fields added
```

### Workflow 3: Link Field to Semantic Term
```
1. Edit entity [✏️ Edit]
2. Add field [Entity or Subtype → + Add]
3. Select semantic term from dropdown
4. Confirm field appears with semantic link
5. Save [SAVE & APPLY]
✅ Done: Field linked to catalog
```

---

## 🚀 Live Environment

**URL:** http://localhost:5173/config

**Steps:**
```
1. Open browser → http://localhost:5173/config
2. Select tenant (top right picker)
3. See entity cards grid
4. Start: [+ Add New Entity] or [🔄 Clone] existing BO
5. Save changes with [SAVE & APPLY]
6. Reload page to verify persistence
```

---

## 📞 Troubleshooting

### "SAVE & APPLY button disabled"
→ No changes made yet. Add entity, field, or subtype.

### "Can't delete core field"
→ Core fields (🔒) inherited from parent. Delete in parent BO or in clone-specific areas.

### "Technical name not lowercase"
→ System auto-converts. Only enter business name.

### "Semantic term dropdown empty"
→ No terms in your catalog yet. Link to catalog_node table for semantic_term type.

### "Clone renamed but technical name didn't update"
→ Check if you edited the "Technical Name (Auto-generated)" field. Leave it disabled - it updates from business name.

---

**Version:** 2.1  
**Updated:** October 17, 2025  
**Status:** Ready to Use ✅
