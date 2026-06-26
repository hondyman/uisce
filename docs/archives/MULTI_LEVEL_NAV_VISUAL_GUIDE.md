# Multi-Level Navigation - Visual Guide

## Navigation Flow

### Initial State
```
┌─────────────────────────────────────────────────────────┐
│ SemLayer ▼    [Bundles ▼] [Models ▼] [Lineage ▼] ...  │
│ (showing Weave category menus - currently selected)    │
└─────────────────────────────────────────────────────────┘
```

### Click SemLayer Button
```
┌─────────────────────────┐
│ SemLayer ▼              │
├─────────────────────────┤
│ 🔵 Catalog              │ ← Category selector
│ 🟣 Weave         ✓      │    (Weave currently selected)
│ 🟠 Entity               │
└─────────────────────────┘
```

### Select Catalog
```
┌─────────────────────────────────────────────────────────────┐
│ SemLayer ▼    [APIs ▼] [Schemas ▼] [Views ▼] [Glossary ▼] │
│ (showing Catalog category menus - now selected)            │
└─────────────────────────────────────────────────────────────┘
```

### Click a Menu Button (e.g., "APIs")
```
┌──────────────────────────┐
│ APIs ▼                   │
├──────────────────────────┤
│ 🌐 API Catalog           │ ← Items in this menu
│    Browse and manage     │
│    APIs                  │
└──────────────────────────┘
```

### Click an Item
```
Navigate to that page
Menu closes
Stays in Catalog category
```

---

## Category Visual Representations

### 🔵 CATALOG (Blue #2196F3)
```
┌─────────────────────────────────────────────────────────┐
│ SemLayer ▼    [APIs ▼] [Schemas ▼] [Views ▼] ...      │
│                                                         │
│ Top Navigation in BLUE color theme                     │
│ Menus change to reflect Catalog focus                  │
│                                                         │
│ Selected menu: Blue underline & bold text              │
└─────────────────────────────────────────────────────────┘
```

**Menus:**
1. APIs → 1 item
2. Schemas → 1 item
3. Views → 1 item
4. Glossary → 2 items
5. Domains → 1 item

---

### 🟣 WEAVE (Purple #9C27B0)
```
┌─────────────────────────────────────────────────────────┐
│ SemLayer ▼    [Bundles ▼] [Models ▼] [Lineage ▼] ...  │
│                                                         │
│ Top Navigation in PURPLE color theme                   │
│ More menus than Catalog, focused on semantic creation  │
│                                                         │
│ Selected menu: Purple underline & bold text            │
└─────────────────────────────────────────────────────────┘
```

**Menus:**
1. Bundles → 1 item
2. Models → 2 items
3. Lineage → 3 items
4. Governance → 2 items
5. Access Control → 2 items
6. Calculations → 1 item

---

### 🟠 ENTITY (Orange #FF9800)
```
┌──────────────────────────────────────────────────────────┐
│ SemLayer ▼    [Entities ▼] [Processes ▼] [Tenants ▼]   │
│                                                          │
│ Top Navigation in ORANGE color theme                    │
│ Most menus, focused on business operations              │
│                                                          │
│ Selected menu: Orange underline & bold text             │
└──────────────────────────────────────────────────────────┘
```

**Menus:**
1. Entities → 3 items
2. Processes → 3 items
3. Tenants → 1 item
4. Validation → 1 item
5. UI & Forms → 2 items
6. Analytics → 4 items
7. System → 4 items

---

## Dropdown States

### Category Selector Dropdown
```
CLOSED:
SemLayer ▼

OPEN:
┌─────────────────┐
│ 🔵 Catalog      │
│ 🟣 Weave   ✓    │ ← Currently selected
│ 🟠 Entity       │
└─────────────────┘

AFTER SELECTION:
Top nav updates immediately with new category's menus
```

### Menu Items Dropdown
```
CLOSED:
[Bundles ▼]

OPEN (showing items):
┌──────────────────────────┐
│ 🎁 Bundles               │
│    Curated semantic...   │ ← Item description
│    [AI]                  │    Item badge
└──────────────────────────┘

HOVER:
Subtle purple background + left border

CLICK:
Navigate away, menu closes
```

---

## Color Coding System

### Color Meanings
- **🔵 Blue (Catalog)** - Cool, calm, informational
  - Data discovery and understanding
  - Glossaries and schemas
  - Read-heavy operations

- **🟣 Purple (Weave)** - Creative, connecting
  - Semantic bundling and creation
  - Lineage and mapping
  - Relationships and policies

- **🟠 Orange (Entity)** - Warm, energetic, operational
  - Business entities and management
  - Process creation and execution
  - System administration

### Application
1. **Category Button Text** - Uses category color when active
2. **Menu Buttons** - Underline in category color when active
3. **Menu Items** - Left border/highlight in category color
4. **Dropdown Background** - Subtle category color tint
5. **Icons** - Category color on active items

---

## Responsive Behavior

### Desktop (md and up)
```
┌─────────────────────────────────────────────────────────┐
│ SemLayer ▼  [Bundles ▼] [Models ▼] [Lineage ▼] ...    │
│ All menus visible, full spacing                         │
└─────────────────────────────────────────────────────────┘
```

### Tablet (sm)
```
┌────────────────────────────────────────┐
│ SemLayer ▼  [Bundles ▼] [Models ▼]... │
│ Menus wrap as needed, optimized spacing│
└────────────────────────────────────────┘
```

### Mobile (xs)
```
┌──────────────────────┐
│ SemLayer ▼           │
│ [Bundles ▼]          │
│ [Models ▼]           │
│ [Lineage ▼]          │
│ [Governance ▼]       │
│ Menus stack vertically│
└──────────────────────┘
```

---

## Interaction Patterns

### Changing Categories
```
Step 1: User clicks "SemLayer"
   ↓
Step 2: Sees category options (Catalog, Weave, Entity)
   ↓
Step 3: Clicks a category
   ↓
Step 4: Top nav updates instantly with new menus
   ↓
Step 5: Category selection dropdown closes
```

### Accessing Menu Items
```
Step 1: User sees category menus (e.g., Bundles, Models, etc.)
   ↓
Step 2: Clicks a menu button
   ↓
Step 3: Dropdown shows that menu's items
   ↓
Step 4: User sees item name, description, and badges
   ↓
Step 5: Clicks item to navigate
   ↓
Step 6: Route loads, menu closes
```

### Visual Feedback
```
Hover on Menu: Subtle background color (20% opacity)
Click Menu: Border-bottom appears in category color
Selected Menu: Bold text + underline + darker color
Hover on Item: Background color change
Selected Item: Left border in category color
```

---

## Menu Structure Examples

### Catalog > Glossary
```
BUTTON: [Glossary ▼]

DROPDOWN:
┌────────────────────────────────────┐
│ 📖 Business Glossary               │
│    Manage semantic and business    │
│    terms...                        │
├────────────────────────────────────┤
│ ⚙️  Catalog Setup                  │
│    Configure glossary structure    │
└────────────────────────────────────┘
```

### Weave > Lineage
```
BUTTON: [Lineage ▼]

DROPDOWN:
┌────────────────────────────────────┐
│ 🗺️  Semantic Mapper                 │
│    Map columns to semantic terms   │
├────────────────────────────────────┤
│ 📈 Claim Aware Lineage             │
│    Data lineage with claims        │
├────────────────────────────────────┤
│ 📊 Drift Reports                   │
│    Data drift analysis             │
└────────────────────────────────────┘
```

### Entity > Analytics
```
BUTTON: [Analytics ▼]

DROPDOWN:
┌────────────────────────────────────┐
│ 🚀 Pre-agg Advisor                 │
│    Optimization recommendations    │
├────────────────────────────────────┤
│ 🎲 Frontier Explorer               │
│    Portfolio analytics             │
├────────────────────────────────────┤
│ 📈 Reports                         │
│    Build reports                   │
├────────────────────────────────────┤
│ 📢 Notifications                   │
│    Notification analytics          │
└────────────────────────────────────┘
```

---

## State Diagram

```
               ┌─────────────────┐
               │  Initial State  │
               │  Weave Selected │
               └────────┬────────┘
                        │
                        ↓
              ┌──────────────────────┐
              │  Click SemLayer ▼    │
              │ Category Menu Opens  │
              └──────────┬───────────┘
                         │
        ┌────────────────┼────────────────┐
        ↓                ↓                ↓
   Catalog          Weave (✓)          Entity
      ↓                ↓                ↓
   [Close]         [Close]          [Close]
      │                │                │
      └────────────────┼────────────────┘
                       ↓
         ┌──────────────────────────┐
         │ Selected Category       │
         │ Updates Top Nav Menus   │
         │ Category Menu Closes    │
         └──────────────┬──────────┘
                        │
                        ↓
          ┌─────────────────────────┐
          │ User Sees New Category  │
          │ Menus in Top Nav        │
          └──────────────┬──────────┘
                         │
                         ↓
              ┌──────────────────────┐
              │ Click Menu Button    │
              │ Items Dropdown Opens │
              └──────────────┬───────┘
                             │
                             ↓
                  ┌──────────────────────┐
                  │ Click Menu Item      │
                  │ Navigate to Route    │
                  │ Dropdowns Close      │
                  │ Stay in Category     │
                  └──────────────────────┘
```

---

## Design System

### Spacing
- Category button: 1rem gap
- Menu buttons: 0.5rem gap
- Menu items: 1.5rem padding
- Dropdown: 2.8rem width

### Typography
- Category label: body2
- Menu label: body2 with 0.5rem icon margin
- Item label: body2, fontWeight 500
- Item description: caption, color text.secondary

### Colors
- Category color: primary
- Inactive: text.primary
- Active: category.primary
- Hover: category color at 10% opacity
- Description: text.secondary

### Borders & Shadows
- Menu bottom border: 2px (when active)
- Dropdown shadow: elevation 8
- Item left border: 3px (when selected)
- Hover: subtle elevation increase

---

## Example Workflow

### User Path: Finding Bundle Items
```
1. Open app → See Weave category (default)
   [Bundles ▼] [Models ▼] [Lineage ▼] ...

2. Click [Bundles ▼]
   ↓
   Dropdown opens showing:
   - 🎁 Bundles (with description & AI badge)

3. Click on "Bundles"
   ↓
   Navigate to /fabric/bundles
   Dropdown closes
   Stay in Weave category

4. Later want to explore Entity...
   Click SemLayer ▼
   ↓
   See: Catalog, Weave ✓, Entity

5. Click Entity
   ↓
   Top nav updates to:
   [Entities ▼] [Processes ▼] [Tenants ▼] ...
```

---

## Keyboard Navigation
- Tab: Move between buttons
- Enter: Open/close dropdowns
- Arrow Down: Move between items
- Escape: Close any dropdown
- Enter on Item: Navigate

---

This multi-level design is **intuitive**, **organized**, and **scalable**! 🚀
