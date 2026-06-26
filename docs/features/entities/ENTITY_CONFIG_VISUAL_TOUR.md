# Entity Config v2.2: Visual Tour

**This is exactly what you asked for!**

---

## 🎯 The UI Layout You Get

### Full Page View

```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                  ENTITY CONFIG BUILDER (v2.2)                ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃  [Search entities and subtypes...]        [SAVE & APPLY (0)]  ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃                                                                ┃
┃  LEFT PANE (300px) │                RIGHT PANEL              ┃
┃  ─────────────────────────────────────────────────────────   ┃
┃                    │                                          ┃
┃  📋 Hierarchy      │  Select an entity or subtype             ┃
┃  ┌──────────────┐  │  to view fields                          ┃
┃  │ ◢ Entity 1   │  │                                          ┃
┃  │   ◢ Sub 1.1 │◄─┼──→ (Click here)                          ┃
┃  │   ◢ Sub 1.2 │  │                                          ┃
┃  │   ◢ Sub 1.3 │  │                                          ┃
┃  │ ◢ Entity 2   │  │                                          ┃
┃  │   ◢ Sub 2.1 │  │                                          ┃
┃  │ ◢ Entity 3   │  │                                          ┃
┃  │   ◢ Sub 3.1 │  │                                          ┃
┃  │   ◢ Sub 3.2 │  │                                          ┃
┃  │   ◢ Sub 3.3 │  │                                          ┃
┃  └──────────────┘  │                                          ┃
┃                    │                                          ┃
┃  [Search...]       │                                          ┃
┃                    │                                          ┃
┗━━━━━━━━━━━━━━━━━━━━┷━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

---

## 📍 After Clicking a Subtype

### Example: Click "Individual Investor"

```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                  ENTITY CONFIG BUILDER (v2.2)                ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃  [Search entities and subtypes...]        [SAVE & APPLY (0)]  ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃                                                                ┃
┃  LEFT PANE (300px) │                RIGHT PANEL              ┃
┃  ─────────────────────────────────────────────────────────   ┃
┃                    │                                          ┃
┃  📋 Hierarchy      │  Client Investor > Individual Investor  ┃
┃  ┌──────────────┐  │  ═════════════════════════════════════ ┃
┃  │ ◢ Entity 1   │  │                                          ┃
┃  │   ◢ Sub 1.1 │  │  🔒 Inherited Fields (2)                ┃
┃  │ ★ Sub 1.2   │◄─┼────────────────────────────────────────┃
┃  │   ◢ Sub 1.3 │  │  ┌──────────────────────────────────┐   ┃
┃  │ ◢ Entity 2   │  │  │ Business  Technical  Type  Sem   │   ┃
┃  │ ◢ Entity 3   │  │  │ Name      Name       Type  Term  │   ┃
┃  │              │  │  ├──────────────────────────────────┤   ┃
┃  └──────────────┘  │  │ Investor  investor_  text  ID    │   ┃
┃                    │  │ ID        id                      │   ┃
┃                    │  │                                    │   ┃
┃  [Search...]       │  │ Legal     legal_    text  Name   │   ┃
┃                    │  │ Name      entity_name             │   ┃
┃                    │  │                                    │   ┃
┃                    │  │ SSN       ssn       text  SSN    │   ┃
┃                    │  │ (inherited from subtype)          │   ┃
┃                    │  └──────────────────────────────────┘   ┃
┃                    │                                          ┃
┃                    │  ✏️ Assigned Fields (2) [+Add Field]   ┃
┃                    │                                          ┃
┃                    │  ┌──────────────────────────────────┐   ┃
┃                    │  │ Business  Technical  Type  Sem ↑↓X │   ┃
┃                    │  │ Name      Name       Type  Term    │   ┃
┃                    │  ├──────────────────────────────────┤   ┃
┃                    │  │ Tax ID    tax_id    text  Tax ↑↓X   ┃
┃                    │  │                                    │   ┃
┃                    │  │ Birth     birth_    date  Birth ↑X   ┃
┃                    │  │ Date      date             (last)    │   ┃
┃                    │  └──────────────────────────────────┘   ┃
┃                    │                                          ┃
┗━━━━━━━━━━━━━━━━━━━━┷━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

---

## 🎨 Field Table Details

### Inherited Fields (🔒 Locked, Read-Only, Blue)

```
Shows fields that come from the parent entity.
These are LOCKED and cannot be edited, deleted, or reordered.

Example for "Individual Investor" subtype:
  - Investor ID (from Client Investor)
  - Legal Name (from Client Investor)
  - SSN (from Individual Investor parent)

Visual: Blue badge, lock icon, no action buttons
```

### Assigned Fields (✏️ Editable, Green)

```
Shows fields that were added specifically to this entity/subtype.
These can be EDITED, DELETED, or REORDERED.

Example for "Individual Investor" subtype:
  - Tax ID       [↑] [↓] [🗑]
  - Birth Date   [↑] [🗑]
  - Status       [↑]

Buttons:
  ↑  = Move field UP in display order
  ↓  = Move field DOWN in display order
  🗑  = Delete field

Visual: Green badge, action buttons, [+Add Field] at top
```

---

## 🖱️ Interaction Flow

### Scenario 1: Click Entity

```
BEFORE:
  Left Tree             Right Panel
  ┌──────────────┐      (empty)
  │ Client Invest
  │   Sub 1.1
  │   Sub 1.2
  └──────────────┘

YOU CLICK: "Client Investor" (the entity itself)

AFTER:
  Left Tree             Right Panel
  ┌──────────────┐      ┌─────────────────┐
  │★Client Invest     │ Client Investor │
  │   Sub 1.1       │                   │
  │   Sub 1.2       │ 🔒 Inherited: 2  │
  └──────────────┘      │ - Investor ID   │
                        │ - Legal Name    │
                        │                 │
                        │ ✏️ Assigned: 0  │
                        │ [+Add Field]    │
                        └─────────────────┘
```

### Scenario 2: Click Subtype

```
BEFORE:
  Left Tree             Right Panel
  ┌──────────────┐      Shows entity fields
  │ Client Invest
  │ ★ Sub 1.2  ◄──────
  │
  └──────────────┘

YOU CLICK: "Individual Investor" (subtype)

AFTER:
  Left Tree             Right Panel
  ┌──────────────┐      ┌─────────────────────────────┐
  │ Client Invest  │      │ Client Investor >          │
  │ ★ Sub 1.2  ◄──┼────→ │   Individual Investor      │
  │                │      │                            │
  └──────────────┘       │ 🔒 Inherited: 3           │
                        │ - Investor ID (parent)     │
                        │ - Legal Name (parent)      │
                        │ - SSN (from subtype)       │
                        │                            │
                        │ ✏️ Assigned: 2             │
                        │ - Tax ID        [↑↓X]      │
                        │ - Birth Date    [↑X]       │
                        │                            │
                        │ [+Add Field]                │
                        │ [SAVE & APPLY]              │
                        └─────────────────────────────┘
```

### Scenario 3: Add a Field

```
STEP 1: Click [+Add Field]
        → Modal opens

┌──────────────────────────────────┐
│ Add Field - Select Semantic Term │
├──────────────────────────────────┤
│ [Search semantic terms...     ]  │
│                                  │
│ Tax ID              [Add]        │
│   Technical: tax_id              │
│   Type: text                     │
│                                  │
│ Status              [Add]        │
│   Technical: status              │
│   Type: enum                     │
│                                  │
│ Birth Date          [Add]        │
│   Technical: birth_date          │
│   Type: date                     │
└──────────────────────────────────┘

STEP 2: Click [Add] next to "Tax ID"
        → Field added to table
        → Modal closes
        → Table refreshes

STEP 3: Assigned Fields now shows:
        Tax ID        [↑↓X]
        Birth Date    [↑X]
        Status        [↑X]  ← NEW

STEP 4: Click [SAVE & APPLY]
        → Backend updated
        → "✅ Saved!"
```

---

## 🎯 Color Coding Reference

```
LEFT SIDEBAR (Tree):
  🔵 Blue Badge   = Core entity (seeded data)
  🟢 Green Badge  = Custom entity (user-created)

RIGHT PANEL (Fields):
  🔒 Blue Section    = Inherited fields (read-only, protected)
  ✏️ Green Section   = Assigned fields (editable, full control)

Field Actions:
  ↑   = Move up (enabled if not first)
  ↓   = Move down (enabled if not last)
  🗑   = Delete (delete icon, red color)
  ✏️   = Edit (disabled for now, ready for v2.3)
```

---

## 📊 What You Can Do

```
✅ TREE NAVIGATION:
   - Search entities by name
   - Click entity → shows entity fields
   - Click subtype → shows subtype fields
   - Expand/collapse subtypes

✅ FIELD MANAGEMENT:
   - Add field from semantic catalog (modal search)
   - Delete field (with confirmation)
   - Reorder fields (up/down buttons)
   - View inherited vs assigned (color-coded)

✅ DATA PERSISTENCE:
   - Changes tracked in real-time
   - Save button shows count of changes
   - Backend stores all changes
   - Reload page → data persists

✅ QUALITY ASSURANCE:
   - Inherited fields locked (can't break parent)
   - Semantic terms required (consistent names)
   - Type-safe operations
   - Audit trail (who added field, when)
```

---

## 🚀 Try It Now

```bash
# 1. Start backend
docker compose up -d backend

# 2. Start frontend
cd frontend && npm run dev

# 3. Go to browser
http://localhost:5173/entity-config

# 4. You should see:
   - Left sidebar with entity tree
   - Right panel with help text
   - Click any entity or subtype
   - See its fields appear on right
```

---

## ✨ Summary

| What | Where | How |
|------|-------|-----|
| **See entities** | Left sidebar (tree) | All entities listed, collapsible |
| **See subtypes** | Left sidebar (under entity) | Listed when entity expanded |
| **View fields** | Right panel | Click entity/subtype on left |
| **See inherited** | Right panel (blue section) | Read-only fields from parent |
| **See assigned** | Right panel (green section) | Editable fields on this entity |
| **Add field** | [+Add Field] button | Select from semantic catalog |
| **Delete field** | 🗑 icon on field row | Click, confirm, done |
| **Reorder fields** | ↑↓ buttons on field row | Click to move up/down |
| **Save changes** | [SAVE & APPLY] button | Click to persist to backend |

---

**This is exactly what you asked for!** 🎉

