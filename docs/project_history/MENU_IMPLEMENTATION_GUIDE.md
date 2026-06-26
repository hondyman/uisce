# Menu Restructure - Implementation & Customization Guide

## Implementation Summary

Your menu structure has been successfully restructured in `MainNavigation.tsx` with the following changes:

### What Changed

**Before:**
```
SemLayer [Config ▼] [Fabric ▼] [Governance ▼] [Analytics ▼] [Upgrade ▼]
```

**After:**
```
SemLayer [Catalog ▼] [Weave ▼] [Entity ▼]
```

### File Modified
- `frontend/src/components/MainNavigation.tsx`

---

## Code Structure

### NavigationCategory Interface
```typescript
interface NavigationCategory {
  label: string;                    // Display name: "Catalog", "Weave", "Entity"
  key: 'catalog' | 'weave' | 'entity';  // Unique identifier
  icon: React.ReactNode;            // Category icon (from @mui/icons-material)
  description: string;              // Tagline shown in mega menu header
  color: {                          // Color scheme for category
    primary: string;                // Main color (#2196F3, #9C27B0, #FF9800)
    light: string;                  // Light variant for hover states
    dark: string;                   // Dark variant for active states
    background: string;             // Background color for headers
  };
  groups: NavigationGroup[];        // Subcategories within this category
}
```

### NavigationGroup Interface
```typescript
interface NavigationGroup {
  label: string;                    // Subcategory name (e.g., "Bundles & Models")
  icon: React.ReactNode;            // Group icon
  items: NavigationItem[];          // Menu items in this group
}
```

### NavigationItem Interface
```typescript
interface NavigationItem {
  label: string;                    // Display name
  path: string;                     // Route path
  icon: React.ReactNode;            // Item icon
  description?: string;             // Tooltip/description text
  badge?: {                         // Optional badge (AI, Updated, New)
    label: string;
    color?: MuiColorType;
  };
}
```

---

## How to Customize

### 1. Add a New Item to an Existing Category

Find the category in `navigationCategories` and locate the group you want to add to:

```typescript
{
  label: 'Weave',
  key: 'weave',
  groups: [
    {
      label: 'Bundles & Models',
      icon: <BuildIcon />,
      items: [
        // Add your new item here
        { 
          label: 'Your New Feature', 
          path: '/your-new-route', 
          icon: <YourIcon />, 
          description: 'What this feature does',
          badge: { label: 'New', color: 'warning' }
        }
      ]
    }
  ]
}
```

### 2. Add a New Subcategory Group

Add a new group object to any category's `groups` array:

```typescript
{
  label: 'Your New Group Name',
  icon: <YourGroupIcon />,
  items: [
    { label: 'Item 1', path: '/item-1', icon: <Icon1 />, description: 'Description' },
    { label: 'Item 2', path: '/item-2', icon: <Icon2 />, description: 'Description' }
  ]
}
```

### 3. Create a New Top-Level Category

Add a new category to the `navigationCategories` array:

```typescript
{
  label: 'NewCategory',
  key: 'newcategory',              // Must be unique
  icon: <NewCategoryIcon />,
  description: 'What this category contains',
  color: {
    primary: '#673AB7',             // Choose your color
    light: '#EDE7F6',
    dark: '#512DA8',
    background: 'rgba(103, 58, 183, 0.08)'
  },
  groups: [
    {
      label: 'Group 1',
      icon: <Group1Icon />,
      items: [
        // Your items here
      ]
    }
  ]
}
```

### 4. Change Colors for a Category

Update the `color` object in any category:

```typescript
{
  label: 'Catalog',
  key: 'catalog',
  color: {
    primary: '#2196F3',      // ← Change this
    light: '#E3F2FD',        // ← And this
    dark: '#1976D2',         // ← And this
    background: 'rgba(33, 150, 243, 0.08)'  // ← And this
  }
}
```

**Color Picker Tips:**
- Primary: The main color (used for selected items, headers)
- Light: 50% tint (used for hover backgrounds)
- Dark: 20% darker (used for active backgrounds)
- Background: Transparent version for subtle backgrounds

### 5. Update Item Descriptions or Badges

```typescript
{ 
  label: 'API Catalog', 
  path: '/api-catalog', 
  icon: <ApiIcon />, 
  description: 'Browse and manage APIs',  // ← Update this
  badge: { label: 'Updated', color: 'success' }  // ← Or this
}
```

### 6. Reorganize Items Between Groups

Simply cut the item from one group's `items` array and paste it in another group's `items` array.

---

## Available MUI Icons

Common icons used in the menu (all from `@mui/icons-material`):

```typescript
import {
  // Categories/Organization
  Category as CategoryIcon,
  Business as BusinessIcon,
  
  // Technical
  Schema as SchemaIcon,
  Api as ApiIcon,
  
  // Actions
  Build as BuildIcon,
  Settings as SettingsIcon,
  Security as SecurityIcon,
  
  // Data/Analytics
  Assessment as AssessmentIcon,
  QueryStats as QueryStatsIcon,
  Timeline as TimelineIcon,
  
  // Communication
  Notifications as NotificationsIcon,
  
  // Utilities
  Policy as PolicyIcon,
  AutoFixHigh as AutoFixHighIcon,
  ManageAccounts as ManageAccountsIcon,
  CheckCircle as CheckCircleIcon,
  SystemUpdateAlt as SystemUpdateAltIcon,
  KeyboardArrowDown as KeyboardArrowDownIcon,
} from '@mui/icons-material';
```

To use other icons, just add the import and use it in your item.

---

## Color Combinations (Ready to Use)

### Blue Palette (Catalog - Current)
```
Primary: #2196F3
Light: #E3F2FD
Dark: #1976D2
Background: rgba(33, 150, 243, 0.08)
```

### Purple Palette (Weave - Current)
```
Primary: #9C27B0
Light: #F3E5F5
Dark: #7B1FA2
Background: rgba(156, 39, 176, 0.08)
```

### Orange Palette (Entity - Current)
```
Primary: #FF9800
Light: #FFF3E0
Dark: #F57C00
Background: rgba(255, 152, 0, 0.08)
```

### Alternative Color Palettes

**Green:**
```
Primary: #4CAF50
Light: #E8F5E9
Dark: #388E3C
Background: rgba(76, 175, 80, 0.08)
```

**Red:**
```
Primary: #F44336
Light: #FFEBEE
Dark: #D32F2F
Background: rgba(244, 67, 54, 0.08)
```

**Teal:**
```
Primary: #009688
Light: #E0F2F1
Dark: #00796B
Background: rgba(0, 150, 136, 0.08)
```

**Indigo:**
```
Primary: #3F51B5
Light: #E8EAF6
Dark: #303F9F
Background: rgba(63, 81, 181, 0.08)
```

---

## Badge Options

Common badge configurations:

```typescript
// AI Feature
badge: { label: 'AI', color: 'info' }

// New Feature
badge: { label: 'New', color: 'warning' }

// Updated Feature
badge: { label: 'Updated', color: 'success' }

// Beta/Experimental
badge: { label: 'Beta', color: 'warning' }

// Deprecated
badge: { label: 'Deprecated', color: 'error' }

// Enterprise Only
badge: { label: 'Enterprise', color: 'primary' }
```

---

## Responsive Behavior

The mega menu automatically adapts to different screen sizes:

### Desktop (md and up)
```
Grid Layout: 2 columns per group
Full descriptions visible
All badges displayed
```

### Tablet (sm)
```
Grid Layout: 2 columns per group
Full descriptions visible
Optimized padding
```

### Mobile (xs)
```
Grid Layout: 1 column per group
Full descriptions visible
Touch-friendly sizing
```

---

## Styling Deep Dive

### Category Header Styling
Located in the mega menu header:
- Uses category's `background` color
- Icon box uses transparent version of `primary` color
- Title uses category's `primary` color
- Description uses muted `text.secondary` color

### Menu Item Card Styling

**Default State:**
- Border: 1px divider gray
- Background: white (or dark background in dark mode)
- Shadow: elevation 1

**Hover State:**
- Border: category `primary` color
- Background: category `light` color
- Shadow: elevation 4
- Transform: translateY(-2px) for lift effect

**Active/Selected State:**
- Border: category `primary` color
- Background: category `primary` color
- Text color: white
- Icon background: semi-transparent white
- Shadow: elevation 4

### Icon Styling
- Icon box padding: 1 unit (8px)
- Icon box radius: 4px
- Color: category `primary` in default state
- Color: white in selected state
- Background in default: category `light` color
- Background in selected: semi-transparent white

---

## Animation & Interaction

### Menu Open Animation
- Smooth fade and position animation
- Originates from clicked category button
- Auto-closes when navigating

### Item Hover Effects
- 120ms smooth transition for:
  - Border color change
  - Background color change
  - Box shadow elevation
  - 2px upward transform

### Category Button States
- Default: transparent background
- Hover: 10% white overlay
- Active: 10% white overlay + bold text

---

## Performance Considerations

1. **Icon Rendering**: All icons are pre-imported at the top of the file. No dynamic icon loading.

2. **Menu State**: Minimal state management (only `activeCategory` and menu positioning).

3. **Grid Layout**: CSS Grid with responsive columns (efficient layout).

4. **Re-render Optimization**: Menu only re-renders on category change or navigation.

---

## Testing Checklist

After making changes, verify:

- [ ] All routes are correct (no typos in paths)
- [ ] All icons are imported and render properly
- [ ] Colors display correctly in light and dark modes
- [ ] Descriptions fit in the available space
- [ ] Badges don't overflow on mobile
- [ ] Menu closes after navigation
- [ ] Active items highlight correctly
- [ ] Hover effects work smoothly
- [ ] Responsive layouts work on xs, sm, md screens
- [ ] No console errors

---

## Common Issues & Solutions

### Issue: Menu items not appearing
**Solution**: Verify the item is in the `groups[n].items` array and the path is correct.

### Issue: Colors look different in dark mode
**Solution**: Dark mode colors are auto-handled by Material-UI. Ensure you're using theme colors or semi-transparent rgba values.

### Issue: Icon not showing
**Solution**: Verify the icon is imported at the top of the file and spelled correctly.

### Issue: Menu doesn't close after navigation
**Solution**: Ensure `onClick={handleMenuClose}` is on the MenuItem or BlockableLink component.

### Issue: Item highlights in wrong color
**Solution**: Verify the category's `color.primary` is set correctly.

---

## Advanced Customization

### Custom Color for Single Item

If you need a specific item to have a different color scheme, you would need to modify the menu item rendering logic. Currently, all items use the category's color scheme.

### Dynamic Item Visibility

To conditionally show/hide items based on user role or permissions:

```typescript
// In the groups definition
const items = [
  { label: 'Admin Feature', path: '/admin', icon: <AdminIcon />, description: 'Admin only' },
  ...(userRole === 'admin' ? [yourAdminItem] : [])
];
```

### Category-Specific Footers

Currently only the Weave category has quick action buttons. To add these to other categories, look for the pattern near line 600 and duplicate it for other categories.

---

## Support & Questions

For implementation help:
1. Review the interfaces at the top of `MainNavigation.tsx`
2. Check the migration map in `MENU_ITEM_MIGRATION_MAP.md`
3. Reference existing items for patterns
4. Test changes incrementally

For design questions:
1. Review `MENU_VISUAL_GUIDE.md` for visual patterns
2. Check `MENU_RESTRUCTURE_SUMMARY.md` for philosophy
3. Consider the user's mental model and workflow
