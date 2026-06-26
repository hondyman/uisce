# Lineage Node Color Scheme Reference

## Color Palette for Different Node Types

### Business Layer (Blue Shades)
```
┌──────────────────────────────────┐
│ Business Object / Business Term   │
├──────────────────────────────────┤
│ Border:     #1E40AF (Dark Blue)   │
│ Background: #DBEAFE (Light Blue)  │
│ Text:       #001F3F (Navy)        │
└──────────────────────────────────┘

┌──────────────────────────────────┐
│ Business Object Field             │
├──────────────────────────────────┤
│ Border:     #0284C7 (Bright Blue) │
│ Background: #DBEAFE (Light Blue)  │
│ Text:       #001F3F (Navy)        │
└──────────────────────────────────┘
```

### Semantic Layer (Purple Shades)
```
┌──────────────────────────────────────┐
│ Semantic Term / Model / View          │
├──────────────────────────────────────┤
│ Border:     #6B21A8 (Dark Purple)    │
│ Background: #E9D5FF (Light Purple)   │
│ Text:       #2D0052 (Deep Purple)    │
└──────────────────────────────────────┘

┌──────────────────────────────────┐
│ Semantic Column                  │
├──────────────────────────────────┤
│ Border:     #92400E (Orange)      │
│ Background: #FED7AA (Light Orange)│
│ Text:       #3F2305 (Dark Orange) │
└──────────────────────────────────┘
```

### Technical Layer (Green & Pink Shades)
```
┌──────────────────────────────────────┐
│ Database Column / Column              │
├──────────────────────────────────────┤
│ Border:     #15803D (Dark Green)      │
│ Background: #DCFCE7 (Light Green)    │
│ Text:       #052E16 (Forest Green)   │
└──────────────────────────────────────┘

┌──────────────────────────────────────┐
│ Table                                │
├──────────────────────────────────────┤
│ Border:     #7E22CE (Dark Purple)    │
│ Background: #F3E8FF (Light Purple)  │
│ Text:       #3F0F5C (Deep Purple)   │
└──────────────────────────────────────┘

┌──────────────────────────────────────┐
│ Schema                               │
├──────────────────────────────────────┤
│ Border:     #BE185D (Dark Pink)      │
│ Background: #FCE7F3 (Light Pink)    │
│ Text:       #500724 (Deep Pink)     │
└──────────────────────────────────────┘

┌──────────────────────────────────┐
│ Database                         │
├──────────────────────────────────┤
│ Border:     #DC2626 (Dark Red)    │
│ Background: #FEE2E2 (Light Red)   │
│ Text:       #4C0519 (Deep Red)    │
└──────────────────────────────────┘
```

## Visual Hierarchy in Lineage Diagram

### Example: Database Column Lineage
```
┌─────────────────────────────────────────┐
│ SEMANTIC LAYER - Purple Background      │
├─────────────────────────────────────────┤
│  ┌──────────────────────────────────┐   │
│  │ Semantic Term                    │   │
│  │ (Purple - #E9D5FF background)    │   │
│  └──────────────────────────────────┘   │
│           ↓ (depends_on)                │
│  ┌──────────────────────────────────┐   │
│  │ Semantic Column                  │   │
│  │ (Orange - #FED7AA background)    │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
           ↓ (maps_to)
┌─────────────────────────────────────────┐
│ TECHNICAL LAYER - Green Background      │
├─────────────────────────────────────────┤
│  ┌──────────────────────────────────┐   │
│  │ Table.Column                     │   │
│  │ (Green - #DCFCE7 background)     │   │
│  └──────────────────────────────────┘   │
│           ↑ (part_of)                   │
│  ┌──────────────────────────────────┐   │
│  │ Table                            │   │
│  │ (Purple - #F3E8FF background)    │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

## Direction Indicators

### Arrow Meanings
```
Source Node → Target Node    (Subject → Object)
    ↓                              ↓
  Shows "→"                   Shows "←"
    ↓                              ↓
"relationship →"           "← relationship"
```

### Example: Business Terms Relationships Table
```
Relationship          Path
━━━━━━━━━━━━━━━━━━   ━━━━━━━━━━━━━
→ depends_on          semantic.term.2
← is_dependency_of    business.object.3
→ maps_to             data.table.customer.id
```

## Implementation Details

### CSS Variables Used
- `--node-border-color`: Border color (computed from node type)
- `--node-background`: Background color (computed from node type)
- `--node-color`: Text color (computed from node type)
- `--node-font-weight`: Font weight (semi-bold by default)

### Dynamic Styling Example
```typescript
// Get colors for node type
const nodeColors = getNodeTypeColor('business_term');
// Result: { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' }

// Apply to styles
const styles = {
  '--node-background': nodeColors.bg,      // #DBEAFE
  '--node-border-color': nodeColors.border, // #1E40AF
  '--node-color': nodeColors.text           // #001F3F
};
```

## Accessibility Considerations

- High contrast ratios maintained across all color combinations
- Colors support colorblind users (shapes and labels also differentiate)
- No information conveyed by color alone (type names shown in tooltips)
- Text color chosen for readability against backgrounds

## Color Consistency

### Related to Catalog Node Types
- `business_object` → Blue (primary business concept)
- `semantic_term` → Purple (semantic layer abstraction)
- `database_column` → Green (technical implementation)
- `table` → Purple-pink (container for columns)
- `schema` → Pink (database organizational unit)

### Edge Types (Use Predicate Labels)
All edge types now use predicate values (e.g., "depends_on", "maps_to", "is_part_of") instead of AGE graph labels, ensuring consistency with the relationships section.
