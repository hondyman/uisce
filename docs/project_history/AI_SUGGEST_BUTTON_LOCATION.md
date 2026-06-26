# AI Suggest Button Location

## Button Now Visible on Every Unmapped Row!

The "AI Suggest" button now appears **directly on the row header** for every unmapped semantic term - you don't need to expand the row first!

### Visual Layout

```
┌─────────────────────────────────────────────────────────┐
│ Semantic Term: customer_id                              │
│ Business Term: [No business term selected]              │
│ [Unmapped]                    [AI Suggest] [▼]         │ ← BUTTON HERE!
└─────────────────────────────────────────────────────────┘

After clicking "AI Suggest":
┌─────────────────────────────────────────────────────────┐
│ Semantic Term: customer_id                              │
│ Business Term: [No business term selected]              │
│ [Unmapped]                   [Refresh AI] [▲]          │ ← Changes to "Refresh AI"
├─────────────────────────────────────────────────────────┤
│ ✨ AI Suggestions                     [Refresh]        │ ← Auto-expanded!
│                                                         │
│ ┌───────────────┐ ┌───────────────┐ ┌──────────────┐ │
│ │ Customer Id   │ │ Customer ID   │ │ Client Id    │ │
│ │ 87%          │ │ 75%          │ │ 62%         │ │
│ │ [Accept]     │ │ [Accept]     │ │ [Accept]    │ │
│ │ [Reject]     │ │ [Reject]     │ │ [Reject]    │ │
│ └───────────────┘ └───────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Button States

| State | Row Status | Button Text | Action |
|-------|-----------|-------------|---------|
| **Default** | Unmapped, no suggestions | "AI Suggest" | Loads suggestions & auto-expands row |
| **After generation** | Unmapped, has suggestions | "Refresh AI" | Reloads suggestions |
| **Selected term** | Has business term | "Save Mapping" | Saves the mapping |
| **Mapped** | Edge exists | No button | Already mapped ✓ |

### How It Works

1. **For unmapped terms without selection**: Shows "AI Suggest" button
2. **Click the button**: 
   - Loads suggestions for that specific term
   - Automatically expands the row
   - Shows up to 3 suggestions inline
   - Button changes to "Refresh AI"
3. **Inside expanded row**: Full suggestions UI with Accept/Reject buttons
4. **After selecting a term**: Button changes to "Save Mapping"

### Key Features

✅ **Always visible** - No need to expand rows to find the button  
✅ **Per-row operation** - Each term has its own button  
✅ **Auto-expand** - Row opens automatically when suggestions load  
✅ **Visual feedback** - Button shows loading spinner  
✅ **Smart text** - "AI Suggest" → "Refresh AI" after first load  
✅ **Only on unmapped** - Button disappears once term is mapped  

### Example Workflow

```
1. Open Business Term Mapper
   └─ See list of semantic terms
   
2. Find unmapped term (shows [Unmapped] chip)
   └─ See "AI Suggest" button on the right
   
3. Click "AI Suggest"
   └─ Button shows loading spinner
   └─ Row auto-expands
   └─ 3 suggestions appear
   
4. Click "Accept" on a suggestion
   └─ Business term selected
   └─ Suggestions cleared
   └─ Button changes to "Save Mapping"
   
5. Click "Save Mapping"
   └─ Edge created in database
   └─ Chip changes to [Mapped]
   └─ Button disappears
```

### Button Visibility Rules

```typescript
// Show "AI Suggest" button when:
!mapping.edge_exists &&           // Not already mapped
!mapping.selected_business_term   // No term selected yet

// Show "Save Mapping" button when:
mapping.selected_business_term && // Term is selected
!mapping.edge_exists              // But not saved yet

// Show no button when:
mapping.edge_exists               // Already mapped ✓
```

### Styling

- **Border**: Outlined style (not filled)
- **Color**: Primary blue
- **Icon**: ✨ (AutoAwesome icon)
- **Size**: Small
- **Loading**: Shows circular progress when generating

### Benefits

1. **Discoverability**: Button is always visible, no hunting required
2. **Efficiency**: One-click to generate and view suggestions
3. **Context**: Each row manages its own suggestions independently
4. **Feedback**: Clear visual states (AI Suggest → Refresh AI → Save Mapping)
5. **Workflow**: Natural progression from suggest → select → save

Now you should see the "AI Suggest" button prominently displayed on every unmapped semantic term row! 🎉
