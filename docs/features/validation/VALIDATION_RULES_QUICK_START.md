# 🚀 Quick Start - Validation Rules Professional UX

## 30-Second Overview

Your Validation Rules page now has:
- ✅ Professional form for creating/editing rules
- ✅ Real-time form validation with error feedback
- ✅ Full backend API integration
- ✅ Tenant-scoped operations
- ✅ Loading states & success notifications
- ✅ Search and filter capabilities
- ✅ Responsive, mobile-friendly design

---

## 🎯 To Get Started

### 1. Start Your Services
```bash
# Terminal 1: Backend
cd /Users/eganpj/GitHub/semlayer
PORT=29080 go run ./backend/cmd/server

# Terminal 2: Frontend
cd frontend
npm run dev
```

### 2. Open the Application
```
http://localhost:5173/core/validation-rules
```

### 3. Select a Tenant
- Use the tenant picker in Fabric Builder
- Choose: Tenant → Product → Datasource
- Rules will auto-load for that tenant

### 4. Try Creating a Rule
1. Click "New Rule" button
2. Fill in the form
3. Watch real-time validation
4. Click "Create Rule"
5. See success notification
6. Rule appears in table

---

## 📝 Form Usage

### Create New Rule
```
1. Click "New Rule"
   → Dialog opens

2. Rule Name* = "Email Validation"
   → Required field

3. Rule Type* = "Field Format"
   → Dropdown with 5 types

4. Target Entity* = "Customer"
   → Required field

5. Type-Specific Fields appear:
   → Field Name: "email"
   → Regex Pattern: "^[^@]+@[^@]+\\.[^@]+$"

6. Severity* = "error"
   → Dropdown

7. Active = ☑ checked
   → Toggle checkbox

8. Click "Create Rule"
   → Loading spinner
   → Success toast
   → Dialog closes
   → Rule in table
```

### Edit Existing Rule
```
1. Click ✏️ edit icon on any rule
   → Form opens with all data

2. Make changes
   → Real-time validation

3. Click "Update Rule"
   → Success notification
   → Table updates
```

### Delete Rule
```
1. Click 🗑 delete icon
   → Confirmation dialog

2. Confirm
   → Rule deleted
   → Success notification
```

---

## 🔍 Search & Filter

### Find Rules
```
Search Box:
- Type "email"
- Table filters by name/description/entity
- Results update in real-time

Filter by Type:
- Click "Rule Type" dropdown
- Select "Field Format"
- Table shows only that type

Filter by Severity:
- Click "Severity" dropdown
- Select "error"
- Table shows only errors

Combine:
- Search "email" + Filter "Field Format"
- Only matching rules shown
```

---

## ⚠️ Common Issues

### Rules Not Loading
**Problem**: Empty table even after selecting tenant
**Solution**:
1. Check tenant is selected
2. Check browser console (F12)
3. Verify backend running: `curl http://localhost:29080/api/health`
4. Check network tab in DevTools

### Form Validation Errors
**Problem**: Can't submit form
**Solution**:
1. Fill all required fields (marked with *)
2. Check red error messages under fields
3. Type-specific fields must be filled based on rule type
4. Fix JSON if using business logic type

### Tenant Not Showing
**Problem**: "No Tenant Selected" warning
**Solution**:
1. Use tenant picker in Fabric Builder UI
2. Select Tenant → Product → Datasource
3. Wait for rules to load
4. Warning should disappear

### API Errors
**Problem**: Error toast when creating rule
**Solution**:
1. Check backend logs for details
2. Verify tenant/datasource selected
3. Try refreshing page
4. Check network connectivity

---

## 🛠️ Form Fields by Type

### Field Format Type
```
Used for: Regex pattern validation
Fields:
  - Field Name* (e.g., "email")
  - Regex Pattern* (e.g., "^[^@]+@[^@]+\\.[^@]+$")
```

### Cardinality Type
```
Used for: Threshold validation
Fields:
  - Field Name* (e.g., "stock")
  - Operator* (>, <, >=, <=, ==, !=)
  - Threshold Value* (e.g., 10)
```

### Uniqueness Type
```
Used for: Unique constraint
Fields:
  - Field Name* (e.g., "email")
```

### Referential Integrity Type
```
Used for: Foreign key validation
Fields:
  - Source Entity* (e.g., "Order")
  - Source Field* (e.g., "customer_id")
  - Target Entity* (e.g., "Customer")
  - Target Field* (e.g., "id")
```

### Business Logic Type
```
Used for: Custom logic
Fields:
  - JSON Condition* (raw JSON object)
  Example:
  {
    "field": "total",
    "operator": ">",
    "value": 0
  }
```

---

## 💾 API Operations

### Behind the Scenes
When you:
- **Create** → POST to `/api/validation-rules`
- **Read** → GET from `/api/validation-rules`
- **Update** → PATCH to `/api/validation-rules/{id}`
- **Delete** → DELETE from `/api/validation-rules/{id}`

All requests automatically include:
- `?tenant_id=X&datasource_id=Y` query params
- `X-Tenant-ID: X` header
- `X-Tenant-Datasource-ID: Y` header

---

## 🎨 Two-Tab Interface

### Rule Builder Tab (Default)
- User-friendly form layout
- Dropdowns and text inputs
- Type-specific fields
- Easy for non-technical users

### JSON Editor Tab
- Shows complete JSON
- Read-only view
- Advanced users can see structure
- Easy to copy/export

---

## 📱 Responsive Design

Works on:
- ✅ Desktop (1920px wide)
- ✅ Tablet (768px wide)
- ✅ Phone (375px wide)

Table scrolls horizontally on small screens

---

## ✨ Pro Tips

### Tip 1: Copy Rule JSON
- Click 📋 copy icon in table
- Rule JSON copied to clipboard
- Paste into file for backup
- Icon changes to ✓ for confirmation

### Tip 2: Search Everything
- Search finds rule name, description, entity
- Try: "Order", "email", "validation"
- Combines with filters

### Tip 3: Use Type Descriptions
- In dropdown, each type has description
- Helps choose right type
- Severity also has color coding

### Tip 4: Active Toggle
- Uncheck "Active" to disable rule
- Without deleting it
- Re-enable later

### Tip 5: Validation Feedback
- Errors appear automatically
- Red text under field
- Clears as you fix it
- No modal popups

---

## 🔐 Tenant Scoping

### Why It Matters
- Your rules only visible to your tenant
- Other tenants can't see your rules
- Rules can't be mixed between tenants
- Data stays isolated and secure

### How It Works
1. You select tenant in picker
2. Page shows that tenant's rules
3. New rules saved to that tenant
4. Switch tenant = different rules
5. Each tenant has isolated data

---

## 🆘 Need Help?

### Check These First
1. Is backend running? `curl http://localhost:29080/api/health`
2. Is frontend running? Can you see the page?
3. Is tenant selected? Check for warning alert
4. Are required fields filled? Look for red errors

### Debug Steps
1. Open DevTools: F12
2. Go to Network tab
3. Perform action (create/edit/delete)
4. Look at API request/response
5. Check for error messages
6. Look at browser console for errors

### Documentation
- Full guide: `VALIDATION_RULES_ENHANCED_UX.md`
- Testing guide: `VALIDATION_RULES_TESTING_GUIDE.md`
- UI mockups: `VALIDATION_RULES_UI_MOCKUPS.md`

---

## 🎯 Success Indicators

You'll know it's working when:

✅ Page loads without errors
✅ Tenant selector shows rules
✅ "New Rule" button is clickable
✅ Form opens with clean layout
✅ Type-specific fields appear
✅ Real-time validation works
✅ Create button submits
✅ Success toast appears
✅ Rule shows in table
✅ Edit button works
✅ Delete button works
✅ Search filters work
✅ No console errors
✅ Responsive on mobile

---

## 📊 Quick Reference

| Action | How |
|--------|-----|
| Create Rule | Click "New Rule" + fill form |
| Edit Rule | Click ✏️ edit + make changes |
| Delete Rule | Click 🗑 delete + confirm |
| Search | Type in search box |
| Filter Type | Click "Rule Type" dropdown |
| Filter Severity | Click "Severity" dropdown |
| Copy JSON | Click 📋 icon |
| View JSON | Click "JSON Editor" tab |
| Switch Tenant | Use tenant picker |
| Reload Rules | Refresh page |

---

## 🚀 You're Ready!

Everything is set up and working. Go ahead and:
1. Create a validation rule
2. Edit it
3. Delete it
4. Search for rules
5. Filter by type
6. Enjoy the professional UX!

---

## 📞 Questions?

See the comprehensive documentation files:
- `VALIDATION_RULES_ENHANCED_UX.md` - Full feature guide
- `VALIDATION_RULES_TESTING_GUIDE.md` - Testing steps
- `VALIDATION_RULES_UI_MOCKUPS.md` - UI layouts
- `VALIDATION_RULES_IMPLEMENTATION_CHECKLIST.md` - What's included

---

**Status**: 🟢 **Ready to Use**

Enjoy your professional Validation Rules interface! 🎉
