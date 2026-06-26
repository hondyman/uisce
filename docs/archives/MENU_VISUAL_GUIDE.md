# Menu Restructure - Visual Guide

## Navbar Layout

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ SemLayer    [Catalog ▼]  [Weave ▼]  [Entity ▼]     [Scope Info]     🔆 🔔 ⚙️   │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Mega Menu for Each Category

### CATALOG MEGA MENU (Blue Theme)
```
┌──────────────────────────────────────────────────────────────────────────┐
│ 📊 Catalog                                                               │
│ API and semantic discovery, data organization                            │
├──────────────────────────────────────────────────────────────────────────┤
│ 🔍 Discovery & Exploration                                              │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 🌐 API Catalog          │ 📋 Schema Explorer   │                      │
│ │ Browse and manage APIs  │ Explore database     │                      │
│ │                         │ schemas and ERDs     │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 📊 Views Catalog        │                      │                      │
│ │ Generated and resolved  │                      │                      │
│ │ views                   │                      │                      │
│ └─────────────────────────┴──────────────────────┘                      │
│ 📚 Glossary & Metadata                                                  │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 📖 Business Glossary    │ ⚙️ Catalog Setup     │                      │
│ │ Manage semantic and     │ Configure node and   │                      │
│ │ business terms...       │ edge types...        │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 🏢 Data Domains         │                      │                      │
│ │ Manage the enterprise   │                      │                      │
│ │ domain hierarchy        │                      │                      │
│ └─────────────────────────┴──────────────────────┘                      │
└──────────────────────────────────────────────────────────────────────────┘
```

### WEAVE MEGA MENU (Purple Theme)
```
┌──────────────────────────────────────────────────────────────────────────┐
│ 🔗 Weave                                                                 │
│ Semantic fabrics, bundles, policies, and lineage                         │
├──────────────────────────────────────────────────────────────────────────┤
│ 📦 Bundles & Models                                                      │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 🎁 Bundles      [AI]    │ 🔨 Model Generator   │                      │
│ │ Curated semantic data   │ Generate models from │                      │
│ │ bundles and packages    │ database tables      │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 🛠️ Model Builder        │ 📐 Calculations Lib  │                      │
│ │ Build and manage custom │ Financial and        │                      │
│ │ models                  │ analytical templates │                      │
│ └─────────────────────────┴──────────────────────┘                      │
│ 🧬 Semantic & Lineage                                                   │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 🗺️ Semantic Mapper      │ 📊 Drift Reports     │                      │
│ │ Map database columns    │ Data drift analysis  │                      │
│ │ to semantic terms       │                      │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 📈 Claim Aware Lineage  │                      │                      │
│ │ Data lineage with claims│                      │                      │
│ └─────────────────────────┴──────────────────────┘                      │
│ 🔐 Governance & Policies                                                │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 📋 Policy Management    │ 👥 Role Management   │                      │
│ │ [Updated]              │ [New]                │                      │
│ │ Manage governance       │ Assign bundles,      │                      │
│ │ policies                │ claims, and scopes   │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 🔒 Access Intelligence  │ 🐛 Access Debugger   │                      │
│ │ Access control          │ Debug access        │                      │
│ │ intelligence            │ control issues      │                      │
│ └─────────────────────────┴──────────────────────┘                      │
├──────────────────────────────────────────────────────────────────────────┤
│ [View bundles workspace] [Policy management] [Role management]           │
└──────────────────────────────────────────────────────────────────────────┘
```

### ENTITY MEGA MENU (Orange Theme)
```
┌──────────────────────────────────────────────────────────────────────────┐
│ 🏢 Entity                                                                │
│ Entity management, business processes, and administration                │
├──────────────────────────────────────────────────────────────────────────┤
│ 👤 Entity Management                                                     │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 🔧 Entity Manager       │ 🤖 Related Objects   │                      │
│ │ Manage entity registry  │ [AI]                 │                      │
│ │ and customizations      │ AI-powered discovery │                      │
│ ├─────────────────────────┤                      │                      │
│ │ ⚙️ Entity Config         │                      │                      │
│ │ Configure entity        │                      │                      │
│ │ properties and behavior │                      │                      │
│ └─────────────────────────┴──────────────────────┘                      │
│ 🔄 Business Processes                                                   │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 🎯 BP Builder           │ 🛠️ BP Model Builder  │                      │
│ │ Business Process visual │ Build and manage BP  │                      │
│ │ builder                 │ models               │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 🔀 Process Flows [AI]   │                      │                      │
│ │ Design and visualize    │                      │                      │
│ │ semantic process flows  │                      │                      │
│ └─────────────────────────┴──────────────────────┘                      │
│ 🏛️ Administration                                                       │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 🏪 Tenants              │ ✅ Validation Rules  │                      │
│ │ Manage tenants and      │ Define and manage    │                      │
│ │ organizations           │ validation rules     │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 🎨 Dynamic UI Generator │ 🔍 Query Builder     │                      │
│ │ Create dynamic forms    │ Build and execute    │                      │
│ │ and UI configurations   │ queries              │                      │
│ └─────────────────────────┴──────────────────────┘                      │
│ 📊 Analytics & Monitoring                                               │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ 🚀 Pre-agg Advisor      │ 🎲 Frontier Explorer │                      │
│ │ Optimization            │ Portfolio analytics  │                      │
│ │ recommendations         │ and simulations      │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 📈 Report Builder       │ 📢 Notification Dash │                      │
│ │ Build and customize     │ Notification         │                      │
│ │ reports                 │ analytics            │                      │
│ └─────────────────────────┴──────────────────────┘                      │
│ 🔄 System & Upgrades                                                    │
│ ┌─────────────────────────┬──────────────────────┐                      │
│ │ ⬆️ Upgrade Center        │ 📊 Upgrade Compare   │                      │
│ │ Plan, review, and       │ Compare current vs   │                      │
│ │ apply upgrades          │ target versions      │                      │
│ ├─────────────────────────┤                      │                      │
│ │ 🔔 Notification Rules   │ 📢 Campaign Manager  │                      │
│ │ Configure notifications │ Manage notification  │                      │
│ │                         │ campaigns            │                      │
│ └─────────────────────────┴──────────────────────┘                      │
└──────────────────────────────────────────────────────────────────────────┘
```

## Color Palette

### Catalog (Blue)
- **Primary:** #2196F3
- **Light:** #E3F2FD
- **Dark:** #1976D2
- **Background:** rgba(33, 150, 243, 0.08)

### Weave (Purple)
- **Primary:** #9C27B0
- **Light:** #F3E5F5
- **Dark:** #7B1FA2
- **Background:** rgba(156, 39, 176, 0.08)

### Entity (Orange)
- **Primary:** #FF9800
- **Light:** #FFF3E0
- **Dark:** #F57C00
- **Background:** rgba(255, 152, 0, 0.08)

## Item Card States

### Default State
```
┌─────────────────────┐
│ 📖 Label            │
│ Description text    │
│ that explains what  │
│ this page does      │
└─────────────────────┘
```

### Hover State
```
┌─────────────────────┐  ← Lifts up
│ 📖 Label            │  ← Enhanced shadow
│ Description text    │
│ that explains what  │
│ this page does      │
└─────────────────────┘
```

### Active/Selected State
```
┌─────────────────────┐
│ 📖 Label      [Tag] │  ← Highlighted in category color
│ Description text    │  ← White/contrast text
│ that explains what  │
│ this page does      │
└─────────────────────┘
```

## Responsive Behavior

### Desktop (md and up)
- All mega menus display with full width
- Items organized in 2 columns per group
- Full descriptions visible

### Tablet (sm)
- Mega menus adapt to available space
- Items organized in 2 columns
- Descriptions preserved

### Mobile (xs)
- Mega menus stack vertically
- Items organized in 1 column
- Compact layout for smaller screens

---

## Key Features Implemented

✨ **Color-Coded Categories** - Each menu has distinct visual identity
✨ **Subcategorization** - Related items grouped logically
✨ **Mega Menu Design** - Clean, scannable layout
✨ **Interactive Feedback** - Clear hover and active states
✨ **Responsive Layout** - Works on all screen sizes
✨ **Badge System** - [AI], [Updated], [New] badges for context
✨ **Category Description** - Tagline shows purpose of category
✨ **Alternating Rows** - Visual separation between groups
