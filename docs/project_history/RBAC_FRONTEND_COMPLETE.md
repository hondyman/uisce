# RBAC Frontend System - Complete Implementation

## Overview

This document describes the complete, production-ready RBAC (Role-Based Access Control) frontend system for the Semlayer platform. The system provides enterprise-grade security management with comprehensive user interfaces for roles, permissions, delegations, field-level security, and team management.

## Table of Contents

1. [Architecture](#architecture)
2. [Components](#components)
3. [Features](#features)
4. [User Guide](#user-guide)
5. [Technical Details](#technical-details)
6. [Integration](#integration)

---

## Architecture

### Frontend Stack

- **React 18** with TypeScript functional components
- **Tailwind CSS** for styling with gradient backgrounds
- **Lucide React** icons for visual consistency
- **React Hooks** for state management (useState, useEffect, useMemo)
- **Fetch API** for REST integration
- **Modal-based workflows** for creation and editing

### Component Structure

```
frontend/src/
├── components/RBAC/
│   ├── RoleManager.tsx                  # Role and permission management
│   ├── UserRoleAssignment.tsx           # User role assignment
│   ├── DelegationManager.tsx            # Approval delegations
│   ├── FieldPermissionEditor.tsx        # Field-level security
│   └── TeamManager.tsx                  # Team management
└── features/admin/pages/
    ├── RoleManagerPage.tsx              # Role manager page wrapper
    ├── UserRoleAssignmentPage.tsx       # User assignment page wrapper
    ├── DelegationManagerPage.tsx        # Delegation page wrapper
    ├── FieldPermissionEditorPage.tsx    # Field permission page wrapper
    └── TeamManagerPage.tsx              # Team manager page wrapper
```

### Routing

All RBAC features are accessible under `/admin/rbac/*`:

- `/admin/rbac/roles` - Roles & Permissions
- `/admin/rbac/users` - User Assignments
- `/admin/rbac/delegations` - Delegations
- `/admin/rbac/field-permissions` - Field Permissions
- `/admin/rbac/teams` - Teams

### Navigation

The RBAC menu is located in the **Setup > Security** section of the main navigation:

- **Roles & Permissions** - Manage roles and permissions
- **User Assignments** - Assign roles to users
- **Delegations** - Approval delegations
- **Field Permissions** - Field-level security
- **Teams** - Team management

---

## Components

### 1. RoleManager Component

**Purpose**: Enterprise role and permission management interface

**File**: `frontend/src/components/RBAC/RoleManager.tsx` (1000+ lines)

**Key Features**:
- Role CRUD operations (Create, Read, Update, Delete)
- Permission catalog with 7 resource types:
  - Process permissions
  - Step permissions
  - Field permissions
  - Document permissions
  - Report permissions
  - Admin permissions
  - Approval permissions
- Expandable permission groups with checkboxes
- Role cloning functionality
- System vs custom role distinction
- Role level filtering (viewer/editor/approver/admin/super_admin)
- Permission count badges
- Visual role level indicators with icons and colors

**State Management**:
- `roles`: All available roles
- `permissions`: Permission catalog
- `selectedRole`: Currently selected role
- `rolePermissions`: Set of selected permission IDs
- `isEditing`, `isCreating`: Mode control flags
- `expandedGroups`: Which permission groups are expanded
- `formData`: Role metadata (role_key, role_name, description, role_level)

**API Endpoints**:
- `GET /api/rbac/roles` - Fetch all roles
- `POST /api/rbac/roles` - Create new role
- `PUT /api/rbac/roles/:id` - Update role
- `DELETE /api/rbac/roles/:id` - Delete role
- `GET /api/rbac/permissions` - Fetch permission catalog

**Visual Design**:
- Two-column layout: role list (left) + role editor (right)
- Gradient background: slate → blue → indigo
- Role level colors: blue (viewer), green (editor), purple (approver), orange (admin), red (super_admin)
- Role level icons: Eye (viewer), Edit2 (editor), CheckCircle2 (approver), Shield (admin), Crown (super_admin)
- Permission groups with expand/collapse animations
- System role lock indicators

---

### 2. UserRoleAssignment Component

**Purpose**: User role assignment with scope and expiration management

**File**: `frontend/src/components/RBAC/UserRoleAssignment.tsx` (600+ lines)

**Key Features**:
- User search and filtering (active/inactive)
- Role assignment modal with scope configuration
- Scope types:
  - **Global**: Applies everywhere (no scope_id needed)
  - **Process**: Limited to specific process (requires process UUID)
  - **Step**: Limited to specific step (requires step UUID)
  - **Team**: Limited to team resources (requires team UUID)
- Expiration date picker (datetime-local input)
- Expired/expiring role warnings (7-day threshold)
- Active role visualization with status badges
- Bulk role display with permission counts

**Expiration Logic**:
- **No expiration**: Permanent assignment (green)
- **Expires soon**: Within 7 days (yellow warning)
- **Expired**: Past end date (red alert with warning message)

**State Management**:
- `users`: User list
- `selectedUser`: Currently selected user
- `userRoles`: Roles assigned to selected user
- `assignmentForm`: { role_id, scope_type, scope_id, expires_at }
- `loading`, `saving`: Operation states

**API Endpoints**:
- `GET /api/users` - Fetch user list
- `GET /api/rbac/users/:id/roles` - Fetch user's roles
- `POST /api/rbac/roles/:roleId/assign` - Assign role to user
- `DELETE /api/rbac/roles/:roleId/unassign/:userId` - Unassign role

**Visual Design**:
- Two-column layout: user list (left) + user role details (right)
- User cards with department/title information
- Active/inactive badges (green/red)
- Scope type icons: Globe (global), Folder (process), GitBranch (step), UsersIcon (team)
- Expiration warnings with Clock icon
- Role level badges with colors

---

### 3. DelegationManager Component

**Purpose**: Approval authority delegation management

**File**: `frontend/src/components/RBAC/DelegationManager.tsx` (600+ lines)

**Key Features**:
- Delegation CRUD operations
- Delegation types:
  - **Full**: Complete approval authority (green badge)
  - **Partial**: Limited by conditions/amounts (yellow badge)
  - **Backup**: Emergency backup only (blue badge)
- Date range configuration (start/end datetime)
- Reason field for audit trail
- Resource scoping (optional type/ID)
- Usage count display (number of delegation uses)
- Active status calculation
- Filter by type and status

**Date Logic**:
- **Active**: is_active && now >= start_date && (!end_date || now <= end_date)
- **Expired**: end_date < now
- Visual indicators: Active=green border, Expired=red border, Inactive=gray

**State Management**:
- `delegations`: All delegations
- `formData`: { delegator_user_id, delegate_user_id, delegation_type, resource_type, resource_id, start_date, end_date, reason }
- `typeFilter`: Filter by delegation type
- `loading`, `saving`: Operation states

**API Endpoints**:
- `GET /api/rbac/delegations` - Fetch delegations
- `POST /api/rbac/delegations` - Create delegation
- `DELETE /api/rbac/delegations/:id` - Delete delegation

**Visual Design**:
- Single column list with filtering
- Delegator → Delegate arrow display
- Delegation type badges (full/partial/backup)
- Active indicator with Activity icon
- Usage count with TrendingUp icon
- Date range display (Calendar + Clock icons)

---

### 4. FieldPermissionEditor Component

**Purpose**: Field-level security and PII masking configuration

**File**: `frontend/src/components/RBAC/FieldPermissionEditor.tsx` (700+ lines)

**Key Features**:
- Field permission level matrix (role × field)
- Permission levels:
  - **None**: No access (hidden) - Red
  - **Read**: Full visibility - Green
  - **Write**: Full access - Blue
  - **Mask**: Partial visibility - Yellow
- PII masking pattern management
- Masking types:
  - Full masking (****)
  - Partial masking (XXX-XX-1234)
  - Hash (SHA-256)
  - Tokenize (replace with token)
- Visual masking preview
- Resource type selector (process/step/document)
- Sensitive field catalog (SSN, Tax ID, Bank Account, Credit Card, etc.)
- Unmasked roles configuration (which roles see unmasked data)

**Sensitive Fields**:
- SSN (Social Security Number) - PII
- Tax ID - PII
- Bank Account - Financial
- Credit Card - Financial
- Email Address - PII
- Phone Number - PII
- Salary - Financial
- Account Balance - Financial

**Masking Patterns**:
- SSN: `XXX-XX-####` (shows last 4 digits)
- Tax ID: `XX-XXXXXXX` (shows last digit)
- Bank Account: `XXXX-####` (shows last 4 digits)
- Credit Card: `XXXX-XXXX-XXXX-####` (shows last 4 digits)
- Email: `X***@domain.com` (shows first letter and domain)
- Phone: `(XXX) XXX-####` (shows last 4 digits)

**API Endpoints**:
- `GET /api/rbac/field-permissions` - Fetch field permissions
- `POST /api/rbac/field-permissions` - Set field permission
- `POST /api/rbac/field-masking-rules` - Create masking rule

**Visual Design**:
- Permission matrix table (fields × roles)
- Permission level buttons with icons (EyeOff, Eye, Edit2, Lock)
- Color-coded permission levels
- Masking preview modal with before/after display
- Field category badges (PII/Financial)
- Sticky column headers for scrolling

---

### 5. TeamManager Component

**Purpose**: Team and department management with member roster

**File**: `frontend/src/components/RBAC/TeamManager.tsx` (600+ lines)

**Key Features**:
- Team CRUD operations
- Team types:
  - **Functional**: Department-based (blue badge)
  - **Project**: Temporary team (green badge)
  - **Cross-Functional**: Multi-department (purple badge)
- Member roster management
- Role-in-team assignments:
  - **Member**: Standard team member (blue badge)
  - **Lead**: Team lead (orange badge)
  - **Admin**: Team admin (red badge)
- Team manager assignment
- Member count tracking
- Add/remove member operations

**State Management**:
- `teams`: All teams
- `selectedTeam`: Currently selected team
- `teamMembers`: Members of selected team
- `users`: Available users
- `teamForm`: { team_key, team_name, description, team_type, manager_user_id }
- `memberForm`: { user_id, role_in_team }

**API Endpoints**:
- `GET /api/rbac/teams` - Fetch all teams
- `POST /api/rbac/teams` - Create team
- `DELETE /api/rbac/teams/:id` - Delete team
- `GET /api/rbac/teams/:id/members` - Fetch team members
- `POST /api/rbac/teams/:id/members` - Add member
- `DELETE /api/rbac/teams/:id/members/:memberId` - Remove member

**Visual Design**:
- Two-column layout: team list (left) + member roster (right)
- Team type icons: Building2 (functional), Target (project), Network (cross-functional)
- Team type color badges
- Role-in-team badges with icons: Crown (admin), Shield (lead), CheckCircle2 (member)
- Member cards with user initials avatar
- Member count badges

---

## Features

### Common Features Across All Components

1. **Tenant/Datasource Scoping**
   - All operations are scoped to selected tenant and datasource
   - Automatic scope injection via TenantContext
   - Missing scope detection with user-friendly error messages

2. **Search and Filtering**
   - Real-time search on all list views
   - Type/status/level filtering
   - Case-insensitive search

3. **Modal-based Workflows**
   - Create operations in modal dialogs
   - Form validation with required field indicators
   - Cancel/save actions with confirmation

4. **Loading States**
   - Animated loading indicators (spinning icons)
   - Skeleton screens for data loading
   - Disabled states during save operations

5. **Empty States**
   - Friendly messages when no data exists
   - Call-to-action buttons to create first item
   - Icon-based visual indicators

6. **Error Handling**
   - Try-catch blocks around all API calls
   - Console error logging for debugging
   - User-friendly error messages (coming soon)

7. **Visual Consistency**
   - Gradient backgrounds (slate → blue → indigo)
   - Rounded-2xl cards with shadow-xl
   - Transition-all animations
   - Color-coded status badges
   - Lucide icons throughout

---

## User Guide

### Accessing RBAC Features

1. Navigate to the main menu
2. Click **Setup** in the top navigation
3. Select **Security** from the dropdown
4. Choose your desired RBAC feature:
   - **Roles & Permissions**
   - **User Assignments**
   - **Delegations**
   - **Field Permissions**
   - **Teams**

### Managing Roles and Permissions

1. **Create a Role**:
   - Click "Create Role" button
   - Enter role key (unique identifier)
   - Enter role name (display name)
   - Add description
   - Select role level (viewer/editor/approver/admin/super_admin)
   - Click "Save Role"

2. **Assign Permissions**:
   - Select a role from the list
   - Expand permission groups (Process, Step, Field, etc.)
   - Check/uncheck permissions as needed
   - Permissions save automatically

3. **Clone a Role**:
   - Select existing role
   - Click "Clone Role" button
   - Enter new role key
   - Modify permissions as needed

4. **Delete a Role**:
   - Select role to delete
   - Click trash icon
   - Confirm deletion (warning if system role)

### Assigning Roles to Users

1. **Assign a Role**:
   - Select user from the list
   - Click "Assign Role" button
   - Select role from dropdown
   - Choose scope type:
     - Global (applies everywhere)
     - Process (specific process)
     - Step (specific step)
     - Team (specific team)
   - Enter scope ID if non-global
   - (Optional) Set expiration date
   - Click "Assign"

2. **Unassign a Role**:
   - Select user from the list
   - Find role in user's role list
   - Click trash icon
   - Confirm unassignment

3. **Filter Users**:
   - Use search bar to find users by name/email
   - Toggle "Active Users Only" filter

### Managing Delegations

1. **Create a Delegation**:
   - Click "Create Delegation" button
   - Select delegator (user giving authority)
   - Select delegate (user receiving authority)
   - Choose delegation type:
     - Full (complete authority)
     - Partial (limited authority)
     - Backup (emergency only)
   - Set start date
   - (Optional) Set end date
   - (Optional) Add reason
   - (Optional) Scope to resource
   - Click "Create"

2. **Delete a Delegation**:
   - Find delegation in list
   - Click trash icon
   - Confirm deletion

3. **Filter Delegations**:
   - Use search bar
   - Filter by delegation type
   - Toggle "Active Only" filter

### Configuring Field Permissions

1. **Set Permission Level**:
   - Select resource type (Process/Step/Document)
   - Find field in the list
   - For each role column, click desired permission level:
     - None (hidden)
     - Read (full access)
     - Write (full access)
     - Mask (partial visibility)
   - Permission saves automatically

2. **Add Masking Rule**:
   - Click "Add Masking Rule" button
   - Select field name
   - Choose masking type (Full/Partial/Hash/Tokenize)
   - Enter masking pattern (e.g., XXX-XX-####)
   - Preview masked output
   - Select roles that see unmasked data
   - Click "Save Rule"

3. **Search Fields**:
   - Use search bar to filter fields by name/category

### Managing Teams

1. **Create a Team**:
   - Click "Create Team" button
   - Enter team key (unique identifier)
   - Enter team name (display name)
   - Add description
   - Select team type:
     - Functional (department)
     - Project (temporary)
     - Cross-Functional (multi-department)
   - Select team manager
   - Click "Create Team"

2. **Add Team Member**:
   - Select team from the list
   - Click "Add Member" button
   - Select user
   - Choose role in team:
     - Member (standard)
     - Lead (team lead)
     - Admin (team admin)
   - Click "Add Member"

3. **Remove Team Member**:
   - Select team
   - Find member in roster
   - Click trash icon
   - Confirm removal

4. **Delete a Team**:
   - Select team to delete
   - Click trash icon
   - Confirm deletion

---

## Technical Details

### State Management Patterns

All components follow a consistent state management pattern:

```typescript
const [data, setData] = useState<Type[]>([]);
const [selectedItem, setSelectedItem] = useState<Type | null>(null);
const [loading, setLoading] = useState(true);
const [showModal, setShowModal] = useState(false);
const [saving, setSaving] = useState(false);
const [formData, setFormData] = useState<FormType>(initialState);
```

### API Integration Pattern

All API calls follow this pattern:

```typescript
const fetchData = async () => {
  try {
    setLoading(true);
    const response = await fetch(
      `/api/rbac/endpoint?tenant_id=${tenant.id}&datasource_id=${datasource.id}`
    );
    const data = await response.json();
    setData(data || []);
  } catch (error) {
    console.error('Failed to fetch data:', error);
  } finally {
    setLoading(false);
  }
};
```

### Component Lifecycle

All components follow this lifecycle:

1. **Mount**: Fetch initial data (roles, users, teams, etc.)
2. **Render**: Display loading state or data
3. **Interaction**: User actions trigger modal displays or state updates
4. **Update**: API calls to backend, then re-fetch data
5. **Unmount**: Clean up (automatic via React)

### Styling System

All components use Tailwind CSS with consistent patterns:

```tsx
// Backgrounds
className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50"

// Cards
className="bg-white rounded-2xl shadow-xl p-6"

// Buttons
className="px-6 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-all"

// Badges
className="px-3 py-1 rounded-full text-xs font-bold bg-blue-100 text-blue-700 border-2 border-blue-300"

// Inputs
className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
```

### Icon Usage

All components use Lucide React icons:

- `Shield` - Security, roles, admin
- `Users`, `UsersIcon` - Users, teams
- `UserCheck` - User assignments
- `Lock`, `LockOpen` - Permissions, field security
- `Eye`, `EyeOff` - Visibility control
- `Edit2` - Write access
- `CheckCircle2` - Member, completion
- `Crown` - Admin, super_admin
- `Target`, `Building2`, `Network` - Team types
- `Calendar`, `Clock` - Dates, expiration
- `Activity`, `TrendingUp` - Usage tracking

---

## Integration

### Backend Requirements

Each RBAC component requires corresponding backend API endpoints:

1. **RoleManager**:
   - `GET /api/rbac/roles`
   - `POST /api/rbac/roles`
   - `PUT /api/rbac/roles/:id`
   - `DELETE /api/rbac/roles/:id`
   - `GET /api/rbac/permissions`

2. **UserRoleAssignment**:
   - `GET /api/users`
   - `GET /api/rbac/users/:id/roles`
   - `POST /api/rbac/roles/:roleId/assign`
   - `DELETE /api/rbac/roles/:roleId/unassign/:userId`

3. **DelegationManager**:
   - `GET /api/rbac/delegations`
   - `POST /api/rbac/delegations`
   - `DELETE /api/rbac/delegations/:id`

4. **FieldPermissionEditor**:
   - `GET /api/rbac/field-permissions`
   - `POST /api/rbac/field-permissions`
   - `POST /api/rbac/field-masking-rules`

5. **TeamManager**:
   - `GET /api/rbac/teams`
   - `POST /api/rbac/teams`
   - `DELETE /api/rbac/teams/:id`
   - `GET /api/rbac/teams/:id/members`
   - `POST /api/rbac/teams/:id/members`
   - `DELETE /api/rbac/teams/:id/members/:memberId`

### Context Requirements

All components require the TenantContext to be configured:

```typescript
import { useTenant } from '../../../contexts/TenantContext';

const { tenant, datasource } = useTenant();
```

The TenantContext provides:
- `tenant`: { id, display_name }
- `datasource`: { id, source_name }

### Routing Configuration

The routing is configured in `AppRoutes.tsx`:

```typescript
import RoleManagerPage from "./features/admin/pages/RoleManagerPage";
import UserRoleAssignmentPage from "./features/admin/pages/UserRoleAssignmentPage";
// ... other imports

<Route path="/admin/rbac/roles" element={<ProtectedRoute><RoleManagerPage /></ProtectedRoute>} />
<Route path="/admin/rbac/users" element={<ProtectedRoute><UserRoleAssignmentPage /></ProtectedRoute>} />
// ... other routes
```

### Navigation Configuration

The navigation menu is configured in `MainNavigation.tsx`:

```typescript
{
  label: 'Security',
  icon: <SecurityIcon />,
  items: [
    { label: 'Roles & Permissions', path: '/admin/rbac/roles', icon: <ShieldIcon />, description: 'Manage roles & permissions' },
    { label: 'User Assignments', path: '/admin/rbac/users', icon: <PersonAddIcon />, description: 'Assign roles to users' },
    // ... other items
  ]
}
```

---

## Future Enhancements

### Planned Features

1. **Audit Trail**
   - Track all permission changes
   - Show who made changes and when
   - Rollback capability

2. **Permission Templates**
   - Pre-configured permission sets
   - Quick role creation from templates
   - Industry-specific templates

3. **Bulk Operations**
   - Bulk user role assignments
   - Bulk permission updates
   - CSV import/export

4. **Advanced Filters**
   - Filter by permission type
   - Filter by expiration date
   - Filter by delegation status

5. **Analytics Dashboard**
   - Permission usage metrics
   - User activity tracking
   - Role effectiveness analysis

6. **Real-time Updates**
   - WebSocket integration for live updates
   - Real-time permission changes
   - Live delegation status

7. **Mobile Optimization**
   - Responsive design improvements
   - Touch-friendly controls
   - Mobile-specific navigation

### Known Limitations

1. **No inline editing**: All edits require modal dialogs
2. **No undo/redo**: Changes are immediate and permanent
3. **No batch selection**: Multi-select not yet implemented
4. **No export**: No CSV/PDF export functionality
5. **Limited validation**: Client-side validation is basic

---

## Support and Troubleshooting

### Common Issues

1. **"No Tenant/Datasource Selected" Error**
   - **Cause**: TenantContext not initialized
   - **Solution**: Select tenant and datasource in the tenant picker

2. **API Calls Failing**
   - **Cause**: Backend not running or CORS issues
   - **Solution**: Verify backend is running on correct port

3. **Permissions Not Saving**
   - **Cause**: API endpoint not implemented
   - **Solution**: Check backend logs for errors

4. **Modal Not Closing**
   - **Cause**: Save operation failed silently
   - **Solution**: Check browser console for errors

### Debugging Tips

1. **Enable Console Logging**: All errors are logged to console
2. **Check Network Tab**: Verify API requests are being sent
3. **Verify Tenant Scope**: Check localStorage for selected_tenant and selected_datasource
4. **Test Backend Endpoints**: Use curl or Postman to test APIs directly

### Development Mode

To run in development mode:

```bash
cd frontend
npm install
npm start
```

The frontend will start on `http://localhost:3000` with hot reload enabled.

---

## Conclusion

This RBAC frontend system provides a complete, production-ready solution for enterprise security management. With comprehensive features for roles, permissions, delegations, field-level security, and team management, it offers everything needed to implement Fortune 500-level access control.

The system is designed for extensibility, with clear patterns for adding new features and integrating with additional backend services. All components follow consistent design principles, ensuring a cohesive user experience across the entire platform.

For technical support or feature requests, please contact the development team.

---

**Last Updated**: January 2025
**Version**: 1.0.0
**Maintainer**: Semlayer Development Team
