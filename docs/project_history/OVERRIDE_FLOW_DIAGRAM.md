# Override Flow - State Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SEMANTIC MAPPER OVERRIDE FLOW                 │
└─────────────────────────────────────────────────────────────────────┘

┌──────────────────┐
│  Initial State   │
│ ┌──────────────┐ │
│ │ Suggested    │ │  User clicks Edit Icon (✏️)
│ │ Term or      │ ├────────────────────────────┐
│ │ Mapped Term  │ │                            │
│ └──────────────┘ │                            ▼
└──────────────────┘                ┌───────────────────────┐
                                    │  Override Mode Active │
                                    │ ┌───────────────────┐ │
                                    │ │ Search Box Open   │ │
                                    │ │ Row Selected ✓    │ │
                                    │ └───────────────────┘ │
                                    └───────────────────────┘
                                             │
                                             │ User types or selects
                                             │
                    ┌────────────────────────┴────────────────────────┐
                    │                                                 │
                    ▼                                                 ▼
        ┌─────────────────────┐                         ┌─────────────────────┐
        │ Option A: Select    │                         │ Option B: Type New  │
        │ Existing Term       │                         │ Term Name           │
        │ ┌─────────────────┐ │                         │ ┌─────────────────┐ │
        │ │ Click dropdown  │ │                         │ │ Type text       │ │
        │ │ suggestion      │ │                         │ │ (2+ chars)      │ │
        │ └─────────────────┘ │                         │ └─────────────────┘ │
        └─────────────────────┘                         └─────────────────────┘
                    │                                                 │
                    │ Immediately                                     │
                    │ applied                                         ▼
                    │                                    ┌────────────────────────┐
                    │                                    │   Check if Term        │
                    │                                    │   Exists in System     │
                    │                                    └────────────────────────┘
                    │                                             │
                    │                                ┌────────────┴────────────┐
                    │                                │                         │
                    │                                ▼                         ▼
                    │                    ┌──────────────────┐    ┌──────────────────┐
                    │                    │ Term Exists      │    │ Term Doesn't     │
                    │                    │                  │    │ Exist            │
                    │                    │ Button:          │    │                  │
                    │                    │ ✓ Apply          │    │ Button:          │
                    │                    │ Existing Term    │    │ ➕ Create &      │
                    │                    └──────────────────┘    │ Apply New Term   │
                    │                             │              └──────────────────┘
                    │                             │                       │
                    │                             │ Click                 │ Click
                    │                             │                       │
                    │                             ▼                       ▼
                    │                    ┌──────────────────┐    ┌──────────────────┐
                    │                    │ Apply Existing   │    │ 1. Create term   │
                    │                    │ Term ID          │    │ 2. Assign to     │
                    │                    └──────────────────┘    │    mapping       │
                    │                             │              └──────────────────┘
                    │                             │                       │
                    └─────────────────────────────┴───────────────────────┘
                                                  │
                                                  ▼
                                    ┌───────────────────────────┐
                                    │  Term Applied State       │
                                    │ ┌───────────────────────┐ │
                                    │ │ Semantic Term Set     │ │
                                    │ │ Semantic Term ID Set  │ │
                                    │ │ Row Selected ✓        │ │
                                    │ │ Override Active       │ │
                                    │ │                       │ │
                                    │ │ 🟢 "Ready to Create   │ │
                                    │ │     Edge" chip shown  │ │
                                    │ └───────────────────────┘ │
                                    └───────────────────────────┘
                                                  │
                                                  │ User clicks
                                                  │ "Create Edges (N)"
                                                  │
                                                  ▼
                                    ┌───────────────────────────┐
                                    │  Confirmation Dialog      │
                                    │  "Create N edges?"        │
                                    └───────────────────────────┘
                                                  │
                                                  │ Confirm
                                                  ▼
                                    ┌───────────────────────────┐
                                    │  Backend API Call         │
                                    │  POST /semantic-mappings  │
                                    │       /edges              │
                                    └───────────────────────────┘
                                                  │
                                                  │ Success
                                                  ▼
                                    ┌───────────────────────────┐
                                    │  Edge Created State       │
                                    │ ┌───────────────────────┐ │
                                    │ │ Edge persisted in DB  │ │
                                    │ │ edge_exists = true    │ │
                                    │ │ Row deselected        │ │
                                    │ │ 🟢 "Mapped" chip      │ │
                                    │ │     shown             │ │
                                    │ └───────────────────────┘ │
                                    └───────────────────────────┘
                                                  │
                                                  ▼
                                    ┌───────────────────────────┐
                                    │  Page Reloads Mappings    │
                                    │  Shows Updated State      │
                                    └───────────────────────────┘


═══════════════════════════════════════════════════════════════════════
                            KEY STATES
═══════════════════════════════════════════════════════════════════════

State                      | Visual Indicator                | User Action
─────────────────────────────────────────────────────────────────────────
Normal                     | Blue term box                   | None needed
Override Mode              | Search box, orange edit icon    | Type or select term
Unsaved Changes            | Yellow warning box              | Click button to apply
Ready to Create            | Green pulse chip ✓              | Click "Create Edges"
Edge Exists                | Green "Mapped" chip             | Complete!

═══════════════════════════════════════════════════════════════════════
                        AUTOMATIC BEHAVIORS
═══════════════════════════════════════════════════════════════════════

✅ Enabling override → Automatically selects row (checkbox checked)
✅ Selecting existing term → Immediately applied to mapping
✅ Creating new term → Term created first, then applied to mapping
✅ Any term application → Shows "Ready to Create Edge" chip
✅ Creating edges → Automatically reloads page to show new state
```
