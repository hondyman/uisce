# 🎯 API Endpoint Catalog Page - Quick Access Guide

## How to Access the Page

### Option 1: Via Menu Navigation
1. Look for **"Core"** section in main navigation menu
2. Click on **"API Endpoint Catalog"** submenu item
3. Page loads at `/core/api-endpoint-catalog`

### Option 2: Direct URL
Navigate directly to:
```
http://localhost:5173/core/api-endpoint-catalog
```

### Option 3: From Connections
1. Go to **Connections**
2. Select a **Tenant** and **Datasource**
3. Click **"View API Catalog"** (if available in connections page)
4. Navigates to catalog page with scope pre-selected

## 📋 Page Layout

```
┌─────────────────────────────────────────────────────────────┐
│  API Endpoint Catalog                                       │
│  Browse and manage all available API endpoints              │
└─────────────────────────────────────────────────────────────┘

┌─ Tenant Scope Alert (if not selected) ──────────────────────┐
│ ⚠️  Please select a tenant and datasource...               │
└────────────────────────────────────────────────────────────┘

┌─ Toolbar ──────────────────────────────────────────────────┐
│ 🔍 [Search endpoints...]  📁 [Category ▼]  🔄 Refresh    │
└────────────────────────────────────────────────────────────┘

Showing 45 of 48 endpoints

┌─ Endpoints Table ──────────────────────────────────────────┐
│ Method │ Endpoint          │ Description      │ ... │ Actions│
├────────┼───────────────────┼──────────────────┼─────┼────────┤
│ GET    │ /api/entities     │ Get all entities │ ... │ Details│
│ POST   │ /api/entities     │ Create entity    │ ... │ Details│
│ PUT    │ /api/entities/:id │ Update entity    │ ... │ Details│
│ DELETE │ /api/entities/:id │ Delete entity    │ ... │ Details│
├────────┼───────────────────┼──────────────────┼─────┼────────┤
│ ... more rows ...                                           │
└────────────────────────────────────────────────────────────┘
```

## 🔍 Using Search

**Search by Endpoint Name:**
```
Type: "entity" 
Matches: "/api/entities", "/api/entity-mappings", etc.
```

**Search by Path:**
```
Type: "/api/users"
Matches: All paths containing "/api/users"
```

**Search by Description:**
```
Type: "create"
Matches: All endpoints with "create" in description
```

## 🎯 Using Category Filter

1. Click the **"Category"** dropdown
2. Select a category to filter
3. Table updates to show only matching endpoints
4. Select **"All Categories"** to clear filter

**Common Categories:**
- Validation
- Entities
- Datasources
- Mappings
- Semantic Objects

## 📖 Viewing Endpoint Details

### Step 1: Click Details Button
```
Click [Details] button on any row
↓
Modal opens
```

### Step 2: View Information

The modal shows 4 sections:

**1. Basic Information**
- Path: `/api/entities`
- Method: GET
- Version: v1.0
- Category: Entities
- Description: Full endpoint description

**2. Status**
- Active/Inactive badge
- Deprecated warning (if applicable)
- Auth Required indicator

**3. Request & Response Schemas**
- Full JSON schema definitions
- Scrollable area for large schemas

**4. Metadata**
- Rate limit: X requests/min
- Subcategory: List, Create, Update, etc.
- Created date
- Updated date

### Step 3: Close Modal
- Click **"Close"** button
- Or press **Escape** key
- Or click outside modal

## 🔄 Refresh Data

Click the **🔄 Refresh** button to:
- Fetch latest endpoints from backend
- Clear search and filters
- Reload data

Use when:
- New endpoints added
- Endpoint definitions changed
- Want fresh data

## ⚙️ Prerequisites

### Before You Can View Endpoints

✅ **Select Tenant and Datasource**
1. Go to **Connections** page
2. Select a **Tenant** from dropdown
3. Select a **Datasource** for that tenant
4. Status shows "Connected"
5. Return to API Catalog page

### If You See "No Data"

Check these:
1. ✅ Tenant and datasource selected in Connections
2. ✅ User has permission to view catalog
3. ✅ Backend API is running and healthy
4. ✅ Endpoints exist in database for this tenant

## 💡 Tips & Tricks

### Search Multiple Terms
```
Search: "entity list" 
Shows endpoints containing both "entity" AND "list"
```

### Filter by Status
Use filter dropdown to find:
- **Active endpoints**: Currently in use
- **Deprecated endpoints**: Being phased out
- **All endpoints**: See everything

### View Request Examples
Expand schemas in details modal to see:
- Request examples (example payloads)
- Response examples (what API returns)

### Copy Endpoint Path
Hover over the endpoint path in table → Click to copy to clipboard

## 🚀 Common Workflows

### Workflow 1: Find All Create Endpoints
1. Search: "create"
2. View results showing creation endpoints
3. Click Details on one to see schema

### Workflow 2: Find Endpoints in Category
1. Open Category dropdown
2. Select "Entities"
3. See all entity-related endpoints

### Workflow 3: Check Endpoint Requirements
1. Search for specific endpoint
2. Click Details
3. Check "Auth Required" status
4. Check "Rate Limit" in metadata

### Workflow 4: Explore New Endpoints
1. Sort by "Recently Updated" (if available)
2. View endpoints added/modified recently
3. Click Details to understand what's new

## 🎨 Color Reference

### HTTP Method Colors
- 🔵 **GET** = Blue (retrieval)
- 🟢 **POST** = Green (creation)
- 🟠 **PUT** = Orange (replacement)
- 🔴 **DELETE** = Red (removal)
- 🟠 **PATCH** = Orange (modification)

### Status Colors
- 🟢 **Active** = Green (in use)
- ⚫ **Inactive** = Gray (not in use)
- 🔴 **Deprecated** = Red (being phased out)

## 📱 Mobile Usage

### On Mobile Devices
- Table scrolls horizontally ↔️
- Toolbar items stack vertically
- Modal takes full screen height
- Touch-friendly buttons and dropdowns

### Viewing Details on Mobile
1. Tap [Details] button
2. Full-screen modal appears
3. Swipe to scroll through details
4. Tap [Close] or tap outside

## ⌨️ Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `/` | Focus search box |
| `Esc` | Close modal / Clear search |
| `Tab` | Navigate between elements |
| `Enter` | Open Details, Refresh data |
| `↑↓` | Navigate dropdown options |

## 🆘 Quick Troubleshooting

| Problem | Solution |
|---------|----------|
| **"Select tenant" warning** | Go to Connections, select tenant + datasource |
| **Table is empty** | Check if tenant has endpoints; refresh page |
| **Search not working** | Verify endpoint names match search term |
| **Modal won't open** | Click [Details] button, check browser console |
| **Page loads slow** | May need to scroll - table shows all endpoints |
| **Refresh doesn't work** | Check network connection; backend may be down |

## 📊 Example Usage Scenarios

### Scenario 1: API Documentation
**Goal**: Find all available endpoints for integration

```
1. Open API Endpoint Catalog
2. Review all endpoints in table
3. Click Details on each to understand requirements
4. Note rate limits and auth requirements
5. Use endpoint paths in your integration
```

### Scenario 2: Data Quality Check
**Goal**: Verify entity endpoints are properly documented

```
1. Search: "entity"
2. Filter: Category = "Entities"
3. Check each endpoint has:
   - Description ✓
   - Request/Response schemas ✓
   - Examples ✓
   - Appropriate version ✓
```

### Scenario 3: Monitoring Deprecated APIs
**Goal**: Track which endpoints are being phased out

```
1. Open API Catalog
2. Look for "Deprecated" badges
3. Click Details to understand replacement
4. Plan migration timeline
```

## 🔐 Permissions & Scope

**Who can access?**
- All authenticated users with:
  - Tenant selection permission
  - Datasource access permission

**What can they see?**
- Only endpoints for selected tenant/datasource
- Endpoints they have permission to view

**Data isolation:**
- Cannot see endpoints from other tenants
- Automatic scope enforcement
- Secure by default

## 📞 Getting Help

**Page Not Loading?**
1. Check browser console for errors (F12)
2. Verify you're logged in
3. Go to Connections and select tenant
4. Refresh page (Ctrl+R)

**Missing Endpoints?**
1. Check right tenant selected
2. Verify endpoints in database
3. Check backend logs
4. Try Refresh button

**Details Modal Issues?**
1. Refresh page
2. Try clicking Details on different endpoint
3. Check browser console
4. Report issue with error message

---

**Quick Links:**
- 📖 Full Guide: `API_ENDPOINT_CATALOG_PAGE_GUIDE.md`
- 📊 Delivery Summary: `API_ENDPOINT_CATALOG_PAGE_DELIVERY.md`
- 🔗 Backend Integration: `BACKEND_API_CATALOG_INTEGRATION.md`
- 🚀 Roadmap: `FRONTEND_BACKEND_INTEGRATION_ROADMAP.md`
