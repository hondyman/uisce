# Edit Object Modal - Quick Start Guide

## What Was Fixed

The original implementation navigated to a separate detail page. Now you have an **inline modal** that shows up right away with:

✅ Proper form fields (name, display_name, description)  
✅ Driver table selection from catalog_node  
✅ Status and configuration options  
✅ Full validation before save  
✅ Real-time feedback via notifications  

## Using the Edit Modal

### Creating a New Business Object

1. Click **"➕ Create"** or **"+ New Business Object"** button
2. Modal opens with blank form
3. Fill in:
   - **Object Name** - Technical key (e.g., `customer`, `portfolio`)
   - **Display Name** - UI label (e.g., `Customer Profile`)
   - **Description** - What it represents
   - **Driver Table** - Select from catalog (optional but recommended)
   - **Status** - Draft, Active, or Deprecated
4. Click **"✓ Create"**
5. Object added to list, notification shows success

### Editing an Existing Business Object

1. Click **"Edit"** button on any object card
2. Modal opens with current data pre-filled
3. Update any field
4. Click **"✓ Update"**
5. Changes saved, list updates immediately

### Driver Table Selection

The modal includes an **autocomplete dropdown** that:

- Loads all available tables from catalog_node
- Filters as you type
- Shows qualified_path for clarity
- Pre-selects if editing (shows green checkmark)
- Can clear selection with X button

**Why Driver Table Matters:**

The driver table is the **primary source** for your business object:
- Auto-detects available fields
- Enables validation against actual schema
- Tracks data lineage in catalog
- Integrates with governance rules

## Form Fields Explained

| Field | Required | Purpose | Example |
|-------|----------|---------|---------|
| **Object Name** | Yes | Technical identifier in code | `customer`, `ips_proposal` |
| **Display Name** | Yes | Human-readable UI label | `Customer Profile`, `IPS Proposal` |
| **Description** | No | What this BO represents | `Retail customer with household linkage` |
| **Driver Table** | No | Source table in data catalog | `public.customers` |
| **Status** | Yes | Lifecycle state | `active` for production use |
| **Enable** | Yes | Turn BO on/off | Checked by default |

## Error Handling

The modal validates:

1. **Object Name** - Cannot be empty
2. **Display Name** - Cannot be empty
3. **Driver Table** - Optional, but recommended

If validation fails:
- Save button stays disabled
- Error message appears in notification
- Focus returns to invalid field

## Tips & Tricks

### Bulk Operations

Want to create multiple BOs?
1. Create first BO, close modal
2. Create opens in clean state
3. Repeat (no need to refresh)

### Reuse Driver Table

If multiple BOs share a source table:
1. Create first BO, select table
2. Create second BO, table still in autocomplete
3. Filter by table name for quick selection

### Changing Status

Edit modal? Update status field:
- **Draft** → still developing
- **Active** → ready for production
- **Deprecated** → legacy, plan migration

## What Happens Behind the Scenes

When you click save:

```typescript
// Modal calls handleSaveBusinessObject()
// Which does:
1. Validate form data
2. POST/PATCH to /api/business-objects
3. Include tenant headers
4. Get back saved object
5. Update local state
6. Show success notification
7. Close modal
8. Refresh list
```

All in under 1 second if network is fast!

## Next: Instance Management

Once you've created your Business Object, you can:

1. **Define Fields** - Core and custom field definitions
2. **Create Subtypes** - Variants like Taxable/IRA/Trust
3. **Add Instances** - Actual customer records
4. **Link Relationships** - Connect to other objects or hard tables

This will be in the **Instance Manager** tab (coming soon).

## Troubleshooting

### Modal won't open
- Ensure you've selected a tenant/datasource in header
- Check browser console for errors

### Driver table dropdown empty
- Catalog tables may not be loaded
- Try refreshing (F5) and reopening
- Ensure your datasource has tables configured

### Save button disabled
- Check if Name or Display Name are empty
- Both are required fields

### Notification doesn't show
- May have appeared briefly
- Check browser console for errors
- Try again, ensure network is connected

### Changes didn't save
- Check API response in Network tab (F12)
- Ensure tenant headers are correct
- Verify backend is running on port 8080

## API Integration (for developers)

The modal calls:

```bash
# Create
POST /api/business-objects
{
  "bo_key": "customer",
  "name": "Customer",
  "display_name": "Customer Profile",
  "driver_table_id": "node-abc123",
  "driver_table_name": "public.customers",
  "status": "active"
}

# Update
PATCH /api/business-objects/{id}
{
  "display_name": "Updated name",
  "status": "draft"
}
```

Both require headers:
```
X-Tenant-ID: {tenantId}
X-Tenant-Datasource-ID: {datasourceId}
```

See [agents.md](./agents.md) for tenant scope requirements.
