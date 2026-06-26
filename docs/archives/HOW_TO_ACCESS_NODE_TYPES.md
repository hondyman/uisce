# 🎯 How to Access Node Types Management

## ✅ Everything is Set Up and Running!

Both the frontend and backend servers are running, and the Node Types feature is fully integrated.

## 📍 How to Navigate to Node Types

### Option 1: Using the Core Menu (Recommended)

1. **Open your browser** to `http://localhost:5173`
2. **Look at the top navigation bar** - you'll see several links
3. **Find the "Core" button** with a dropdown arrow (▾)
4. **Click "Core"** to open the dropdown menu
5. **Click "Node Types"** in the menu

The Core menu contains these items:
- Domains
- Semantic Mapper  
- **Node Types** ← This is what you want!
- Tenant Management

### Option 2: Direct URL

Simply navigate to: **`http://localhost:5173/core/node-types`**

## 🔍 Troubleshooting

### "I don't see the Core menu"

The Core menu should be visible in the top navigation bar right after these links:
```
Micro-Bundle Catalog | Bundle Explorer | Fixed Income Analytics | JIT Request Panel | Access Explanation | Core ▾ | Fabric ▾
```

If you don't see it:
1. Refresh the page (Cmd+R or Ctrl+R)
2. Clear your browser cache
3. Check the browser console for errors (F12)

### "I see 'Tenant Required' warning"

This is expected! The Node Types feature requires tenant scope for security:

1. Look for a **tenant picker** or **tenant selector** in your UI
2. Select a tenant (e.g., "default" or any available tenant)
3. Navigate back to `/core/node-types`

Alternatively, you can manually set the tenant in browser console:
```javascript
localStorage.setItem('selected_tenant', JSON.stringify({ 
  id: 'default', 
  display_name: 'Default Tenant' 
}));
// Then reload the page
location.reload();
```

### "The page is blank when I navigate to /core/node-types"

1. **Check the browser console** (F12 → Console tab) for errors
2. **Verify both servers are running:**
   ```bash
   # Check frontend
   lsof -ti:5173
   
   # Check backend
   lsof -ti:8080
   ```
3. **Restart the frontend dev server:**
   ```bash
   cd frontend
   npm run dev
   ```

## 🎨 What You Should See

When you successfully navigate to the Node Types page, you'll see:

- **Page Title**: "Node Type Management"
- **Description**: "Configure node types and their properties for your business glossary"
- **"+ Create Node Type"** button in the top right
- **A table** listing existing node types with columns:
  - Type Name
  - Description
  - Properties (count)
  - Status (Active/Inactive)
  - Parent
  - Actions (Edit/Delete buttons)

## 🚀 Quick Start Guide

Once on the Node Types page:

1. **Click "+ Create Node Type"** to create a new node type
2. Fill in:
   - Type Name (e.g., `business_term`, `semantic_column`)
   - Description
   - Active status
   - Parent type (optional)
3. **Add Properties** to define custom fields:
   - Click "Add Property"
   - Configure name, data type, input type
   - Set validation rules
4. **Preview** how properties will render
5. **Save** your node type

## 📞 Still Having Issues?

Run the verification script:
```bash
cd /Users/eganpj/GitHub/semlayer
./verify-node-types.sh
```

This will check:
- ✅ All files are in place
- ✅ Routes are registered
- ✅ Servers are running
- ✅ Everything is properly configured

---

**Servers Running:**
- Frontend: http://localhost:5173
- Backend: http://localhost:8080

**Direct Link:** http://localhost:5173/core/node-types
