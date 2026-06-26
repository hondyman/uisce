# Field Selection Wizard - User Guide

## Overview
The new Field Selection Wizard provides a first-class, wizard-like experience for mapping semantic fields to your Business Objects. No more scrolling through cramped tables!

## Quick Start

### Step 1: Open Business Object Editor
1. Navigate to the Business Objects page
2. Click "Create New" or edit an existing object
3. Fill in the basic information (name, display name, description)

### Step 2: Select Driver Table
1. Look for the "🗂️ Driver Table (Source)" section
2. Click the search field and type a table name
3. Select the table from the dropdown
4. The table name will appear as a chip below

### Step 3: Map Fields
1. Once a driver table is selected, the "+ Add Fields" button becomes active
2. Click the button to open the Field Selection Wizard
3. The wizard shows all available fields from your driver table
4. Fields are color-coded by data type:
   - **Blue**: Numeric fields (INT, FLOAT, DECIMAL, etc.)
   - **Green**: String fields (VARCHAR, TEXT, STRING, etc.)
   - **Orange**: Boolean/Unknown fields

### Step 4: Select Fields
You have multiple ways to find and select fields:

#### Method A: Browsing
- Start with the grid view to see field cards
- Scroll through and click cards to select them
- Or click the checkboxes for multi-select

#### Method B: Searching
- Type in the search box to filter fields by name or path
- Results update instantly as you type
- Clear the search to see all fields again

#### Method C: Type Filtering
- Click type filter chips: "All Types", "Numeric", "String", "Boolean"
- Only matching fields will be displayed
- Combine with search for precise filtering

#### Method D: List View
- Click the "List" tab to see a more compact view
- Fields are grouped by table/schema for easier navigation
- Still supports searching and filtering

### Step 5: Review Selections
- Selected fields appear at the bottom in a success-colored card
- Shows the count of selected fields
- Click the X on any chip to remove it immediately

### Step 6: Confirm Selection
- Click the "Add (N)" button to add all selected fields
- Fields are added to your Business Object's field mapping
- The wizard closes automatically

### Step 7: Review and Save
- Back in the main editor, review your mapped fields
- Each field shows:
  - Field name
  - Full qualified path
  - Data type badge
  - Remove button (if you want to change it)
- Continue editing other properties as needed
- Click "✓ Create" or "✓ Update" to save

## Field Selection Details

### What Fields Can Be Selected?
✅ **Can Select:**
- Any field from the selected driver table
- Fields that have semantic term mappings
- Fields not already mapped to this object

❌ **Cannot Select:**
- Fields with no semantic term mapping (filtered automatically)
- Fields already mapped to this object (hidden from wizard)
- Fields from other tables (filtered by driver table)

### View Modes

#### Grid View
```
┌─────────────────┐  ┌─────────────────┐
│ customer_id     │  │ customer_name   │
│ path/to/field   │  │ path/to/field   │
│ INT      ☐      │  │ VARCHAR   ☑      │
└─────────────────┘  └─────────────────┘
```
- Card-style layout
- Great for visual scanning
- Shows all field details at once
- Best for finding fields by visual inspection

#### List View
```
Schema/Table Group
  ☐ customer_id (INT)
    └ path/to/field
  ☑ customer_name (VARCHAR)
    └ path/to/field
```
- Compact hierarchical layout
- Grouped by schema/table
- Best when you know what you're looking for
- Easier to scan when table has many fields

### Filtering Options

#### Type Filter Chips
- **All Types**: Show all available fields (default)
- **Numeric**: Fields with numeric data types
- **String**: Fields with text/string data types  
- **Boolean**: Boolean/bit fields
- Filters are applied instantly
- Can be combined with search

#### Search Box
- Search by field name
- Search by qualified path
- Search by data type
- Results update as you type
- Case-insensitive matching

### Selection Summary
Shows:
- Count of selected fields (e.g., "3 fields selected")
- Chips for each selected field
- Click X to deselect individual fields
- Only fields you can deselect appear here

## Common Workflows

### Workflow 1: Map All Numeric Fields
1. Open wizard
2. Click "Numeric" filter
3. Click "Select All" checkbox in table header
4. Click "Add (N)"

### Workflow 2: Find a Specific Field
1. Open wizard
2. Type field name in search box
3. Results filter as you type
4. Click the field to select
5. Click "Add (1)"

### Workflow 3: Map by Data Type
1. Open wizard
2. Click "String" filter
3. Browse and select relevant fields
4. Click type filter "Numeric"
5. Select numeric fields
6. Click "Add (N)"

### Workflow 4: Remove and Remap
1. In main editor, click "✕ Remove" next to a field
2. Field is removed immediately
3. Click "+ Add Fields" again
4. Field is now available for selection
5. Reselect if needed

## Tips & Tricks

### 💡 Pro Tips
1. **Use View Preference**: Use grid for quick visual scanning, list for knowing exactly what you need
2. **Filter by Type First**: Narrow down with type filter, then search within the results
3. **Multi-Select Efficiency**: Use the header checkbox to select/deselect all visible fields at once
4. **Review Before Confirming**: Check the selection summary to verify you've selected the right fields
5. **Incremental Mapping**: You don't need to map all fields at once - come back anytime with "+ Add Fields"

### ⚠️ Common Issues

**Q: I can't find a field I need**
A: Check if it has a semantic term mapping. Fields without semantic terms can't be used. Contact your data engineer to add the semantic mapping.

**Q: A field is greyed out/hidden**
A: It's probably already mapped to this object. Look in the "Mapped Fields" section to see it.

**Q: The wizard won't open**
A: You need to select a driver table first. Go back to the main editor and pick a table.

**Q: I selected wrong fields**
A: Click the "Add (N)" button and you'll go back. Then click "✕ Remove" on the fields you don't want.

**Q: No fields are showing**
A: This usually means no semantic term mappings exist for your driver table. Check with your admin.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Enter` | Confirm selection and close wizard |
| `Escape` | Cancel and close wizard |
| `Ctrl/Cmd + A` | Select all visible fields |
| `Ctrl/Cmd + Click` | Multi-select mode |
| `/` | Focus search box |

## Accessibility

The wizard is fully accessible:
- ✓ Keyboard navigable (Tab, Enter, Escape)
- ✓ Screen reader friendly
- ✓ High contrast mode supported
- ✓ Touch-friendly checkboxes and buttons
- ✓ Mobile responsive layout

## FAQ

**Q: Can I map multiple fields at once?**
A: Yes! Select multiple fields using checkboxes, then click "Add (N)" to add them all at once.

**Q: How many fields can I map?**
A: As many as needed. The wizard can handle thousands of fields smoothly thanks to client-side filtering.

**Q: Are fields required?**
A: No. You can create a Business Object without any mapped fields, though it's usually recommended to map at least a few key fields.

**Q: Can I change field mappings later?**
A: Yes! Edit the object anytime and add/remove fields using the same wizard.

**Q: Do I need admin permissions?**
A: No. If you can create/edit Business Objects, you can map fields. Semantic term mappings are created by data engineers.

**Q: What if I want to map a field that's not in the wizard?**
A: It might not have a semantic term mapping yet. Ask your data engineer to create one in the Semantic Catalog.

## Next Steps

After mapping fields:
1. Review field mappings in the "Mapped Fields" section
2. Continue editing other Business Object properties
3. Add subtypes if needed
4. Add validation rules if needed
5. Click "✓ Create" or "✓ Update" to save

Enjoy the improved field selection experience! 🎯

---

**Need help?** Check the Semantic Catalog documentation or contact your data engineering team.
