# FieldAutocomplete Keyboard Navigation Guide

## Quick Reference Card

Print this or display it as a tooltip in your application!

```
╔════════════════════════════════════════════════════════════╗
║           FIELD AUTOCOMPLETE - KEYBOARD SHORTCUTS           ║
╠════════════════════════════════════════════════════════════╣
║                                                            ║
║  🔽  ARROW DOWN         Open dropdown or move down        ║
║  🔼  ARROW UP           Move up (cycles to bottom)        ║
║  ✓   ENTER              Select highlighted field          ║
║  ✕   ESCAPE             Close dropdown                    ║
║  🖱️   MOUSE              Hover to highlight items        ║
║                                                            ║
║  💡 Start typing to search across field names and         ║
║     descriptions. Recently used fields appear first!      ║
║                                                            ║
╚════════════════════════════════════════════════════════════╝
```

## Detailed Keyboard Navigation

### Opening the Dropdown
```
User Action              Result
─────────────────────────────────────────────────────────────
Click input field    →  Dropdown opens, shows all/recent fields
Type a character     →  Dropdown opens, shows matching fields
Press Arrow Down     →  Dropdown opens, first item highlighted
```

### Navigating Within Dropdown
```
Key              Result
─────────────────────────────────────────────────────────────
Arrow Down       Move highlight to next field
                 (cycles to first field when at end)

Arrow Up         Move highlight to previous field
                 (cycles to last field when at start)

Mouse Move       Automatically highlights the item under cursor
```

### Selecting a Field
```
Method           Result
─────────────────────────────────────────────────────────────
Press Enter      Select highlighted field, close dropdown

Click Item       Select field, close dropdown

Mouse Down       Select field, close dropdown
```

### Closing Dropdown Without Selection
```
Key/Action       Result
─────────────────────────────────────────────────────────────
Press Escape     Close dropdown, keep search text

Click Outside    Close dropdown, keep search text

Tab Away         Close dropdown (standard form behavior)
```

## Common Navigation Patterns

### Pattern 1: Find Field by Name
```
1. Click input field
2. Type first few letters: "empl"
3. Matching fields appear: "employee_id", "employee_name"
4. Press Arrow Down to highlight "employee_id"
5. Press Enter to select
```

### Pattern 2: Use Recently Used Field
```
1. Click input field (don't type anything)
2. See "RECENTLY USED" section at top
3. Press Arrow Down to highlight recently used field
4. Press Enter to select
```

### Pattern 3: Browse All Fields
```
1. Click input field
2. Press Arrow Down to open dropdown
3. See all fields
4. Press Arrow Down multiple times to browse
5. When you see the right field, press Enter
```

### Pattern 4: Search by Description
```
1. Click input field
2. Type a word from field description: "salary"
3. Matching fields appear (e.g., "salary" field)
4. Press Enter to select the only match
   (or use Arrow Down if multiple matches)
```

### Pattern 5: Power User Rapid Selection
```
1. Click field
2. Type: "u" (filters to UUID fields)
3. Press Arrow Down repeatedly to find the right one
4. Press Enter
5. All in < 3 seconds!
```

## Accessibility Features

### Screen Reader Users
```
✓ Semantic HTML structure
✓ ARIA attributes for dropdown state
✓ Clear field type and relationship descriptions
✓ Error messages announced
✓ Selection feedback
```

### Keyboard-Only Users
```
✓ All functionality accessible via keyboard
✓ No mouse required
✓ Logical tab order
✓ Escape to cancel
✓ Enter to confirm
```

### Motor Control Users
```
✓ Large click targets
✓ Keyboard navigation for users with tremor
✓ Mouse hover support for precise pointing
✓ No time-dependent interactions
✓ Consistent keyboard shortcuts
```

## Type Indicators Reference

When you see these icons next to field names, they indicate the data type:

```
🔑 UUID/ID           Primary keys and references
📝 Text/Varchar      Text fields and descriptions
#️⃣ Integer           Whole numbers (counts, IDs)
📊 BigInt            Large numbers
💰 Decimal/Numeric   Money and precise decimals
✓  Boolean           True/False values
⏰ Timestamp          Date and time together
📅 Date              Just the date
🕐 Time              Just the time
{} JSON/JSONB        Structured data
[] Array             Lists of values
📦 Other             Rare or custom types
```

## Color Badges Reference

Field type badges use color coding:

```
Purple Badge    🔑 uuid          Unique identifiers
Blue Badge      📝 text/varchar  Text data
Green Badge     #️⃣ integers      Numeric counts
Amber Badge     💰 decimal       Money/precise numbers
Red Badge       ✓  boolean       True/False values
Indigo Badge    ⏰ timestamps     Date and time
Slate Badge     {} json          Structured data
Gray Badge      ⊘ nullable       Allows empty values
```

## Navigation Tips for Power Users

### Tip 1: Partial Text Search
```
Looking for employee ID?
Type "empl" → "id" search → narrows to "employee_id"
Much faster than typing the full name!
```

### Tip 2: Use Recently Used
```
Frequently selecting the same field?
It will appear in "RECENTLY USED" section
Just press Down arrow once, then Enter!
```

### Tip 3: Search by Description
```
Don't know the field name?
Search by what it means instead!
Type "salary" → finds "employee_salary" field
```

### Tip 4: Arrow Key Cycling
```
At the bottom of list? Press Down again → jumps to top
At the top of list? Press Up again → jumps to bottom
Never need to reach for the mouse!
```

### Tip 5: Quick Validation
```
In a rush? Here's the fastest flow:
1. Click field (1 click)
2. Type 2-3 letters (0.5 seconds)
3. Press Enter (0.2 seconds)
Total: ~1 second per selection!
```

## Troubleshooting

### "Nothing Happens When I Press Arrow Down"
```
Solution: Click in the input field first to focus it
The dropdown won't open without focus
```

### "Arrow Keys Are Moving Page Instead of List"
```
Solution: Make sure focus is in the input field
Some browser extensions can interfere with shortcuts
Try Alt+F4 to reset focus, then retry
```

### "I Pressed Escape But Nothing Happened"
```
Solution: Check if dropdown was actually open
You need to press Arrow Down first to open it
Or start typing to trigger the dropdown
```

### "Mouse Won't Highlight Items"
```
Solution: The dropdown should auto-open on mouseover
Try moving your mouse more slowly
If still not working, try clicking in the field first
```

### "My Recently Used Field Disappeared"
```
Solution: Recently used fields are cleared when:
- Browser session ends (browser closed)
- Browser cache is cleared
- SessionStorage is disabled
- You select a different field (top 5 only)
```

## Mobile/Touch Device Support

### On Tablets/Phones
```
✓ Touch to open dropdown
✓ Scroll to browse fields
✓ Tap to select field

Note: Keyboard shortcuts work if:
- Using keyboard with tablet
- Using Bluetooth keyboard
- Using keyboard on mobile (external)
```

## Comparison: Before vs After

### Before: Plain TextField
```
❌ Type full field name or search results scattered
❌ No recently used history
❌ Field type not visible
❌ No keyboard navigation
❌ Slow discovery process

Example time: ~5 seconds per field
```

### After: FieldAutocomplete
```
✅ Smart search across name and description
✅ Recently used fields appear first
✅ Field types with visual indicators
✅ Full keyboard navigation
✅ Fast discovery process

Example time: ~1 second per field
```

## User Journey Examples

### Scenario 1: New User
```
New to system, doesn't know field names

1. Clicks input → sees "RECENTLY USED" or "ALL FIELDS"
2. Reads field descriptions to understand what they do
3. Sees type badges to understand data
4. Clicks or presses Arrow Down + Enter
5. Successfully selects field!

Time: ~10 seconds (includes learning time)
```

### Scenario 2: Frequent User
```
Uses same fields regularly

1. Clicks input → sees recently used list
2. Presses Down arrow once
3. Presses Enter
4. Done!

Time: ~1 second (muscle memory)
```

### Scenario 3: Power User (Keyboard)
```
Experienced, uses keyboard shortcuts

1. Tab to field (standard form navigation)
2. Type "sal" (filters to salary fields)
3. Press Down if needed to disambiguate
4. Press Enter
5. Done!

Time: ~0.5 seconds
```

## Feedback & Improvements

### What Feedback to Report
```
If you find an issue with:
- Keyboard shortcuts not working
- Recently used fields not persisting
- Dropdown not appearing
- Search not finding fields
- Performance issues
- Accessibility problems

Please report to: [support email/channel]
```

---

## Quick Start (TL;DR)

| I want to... | Do this... |
|---|---|
| Open dropdown | Click field or press ↓ |
| Search fields | Start typing |
| Navigate | Use ↑↓ keys |
| Select field | Press Enter or click |
| Close | Press Esc or click outside |
| Find recently used | Click field, look at top |
| Find by type | Look for icon/color badge |
| Understand field | Read description text |

---

**Master the FieldAutocomplete keyboard shortcuts and you'll be selecting fields in under a second!** ⚡
