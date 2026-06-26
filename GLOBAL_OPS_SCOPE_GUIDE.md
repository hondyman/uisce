# Using Operating Scope as Global Ops

## Problem Solved ✅

When you logged in as `global_ops`, the Glossary page showed no semantic terms because **no operating scope (tenant + datasource) was selected**. 

We've added a helpful UI that guides you to select your scope.

---

## How to Use

### Step 1: Open Glossary Page
Navigate to the **Glossary** section in the left sidebar.

### Step 2: See the Empty State Message
You'll see a message:
```
Select Operating Scope
Please select a tenant and datasource to view and manage semantic terms.
```

With a button: **"Select Tenant & Datasource"**

### Step 3: Click the Button
Click **"Select Tenant & Datasource"** to open the scope selector dialog.

### Step 4: Select Your Scope

#### In the Dialog:
1. **Select Tenant**: Click on "Uiscé" (or your desired tenant)
2. **Select Instance**: Choose the appropriate instance
3. **Select Product**: Choose the product
4. **Select Datasource**: Choose "Northwinds" (or your datasource)
5. **Confirm**: The dialog will close and apply your selection

### Step 5: View Semantic Terms
The Glossary page will now show:
- ✅ All Semantic Terms for your selected scope
- ✅ Mapping statistics (Total, Mapped, Unmapped)
- ✅ Filter by mapping status
- ✅ Ability to create/edit/delete terms

---

## Where to Find the Scope Selector (Alternative Access)

If you want to change your scope later, click the **"Scope Badge"** in the top navigation bar:
- Look for a button showing your current tennant/datasource
- Click it to re-open the scope selector

---

## For Global Ops Users

As a `global_ops` user, you can:
- ✅ Switch between different tenants and datasources freely
- ✅ Manage semantic terms for any combination
- ✅ View all accessible tenants
- ✅ Create/edit/delete business objects across scopes

---

## Behavior

**When you select a scope:**
- Your selection is **saved in browser storage** (persists across page refreshes)
- The selection applies to **all pages** (Glossary, Business Objects, etc.)
- You can **change scope anytime** by clicking the scope selector

**If you clear your selection:**
- All pages will show the empty state again
- You'll need to select a scope to continue

---

## Tech Details

**Implementation:**
- Added scope selector UI to `SemanticTermsTab` component
- Integrated with `ScopeSelectorDialog` (existing component)
- Uses `AccessContext` for platform operator detection
- Persists scope in localStorage

**Files Modified:**
- `frontend/src/pages/glossary/SemanticTermsTab.tsx`

---

## Next: Try It Out!

1. Refresh your browser: **Cmd+R** (or **Ctrl+R** on Windows/Linux)
2. Go to **Glossary** page
3. Click **"Select Tenant & Datasource"**
4. Choose **uisce** tenant and **northwinds** datasource
5. View your semantic terms! ✨

