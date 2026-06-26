# 🌙 Dark Mode Implementation Guide for All Pages

This guide provides step-by-step instructions to convert your React pages to support both light and dark modes using the pattern from your Entity Details examples.

## ✅ Quick Start Checklist

- [x] ✅ **Tailwind config** - `darkMode: 'class'` enabled with extended colors
- [x] ✅ **CSS variables** - Light/dark/high-contrast palettes in `index.css`
- [x] ✅ **ThemeContext** - Global theme state management
- [x] ✅ **ThemeToggleButton** - Component for toggling themes (in navbar)
- [x] ✅ **Dark Mode Helpers** - Utility functions in `src/utils/darkModeHelpers.ts`
- [ ] ⏳ **Convert Pages** - Update main pages with `dark:` classes

## 🎯 Available Color Palettes

### Light Mode (Default)
- **Background**: `bg-background-light` or `#f8fafc`
- **Surfaces**: `bg-white`
- **Text Primary**: `text-slate-900`
- **Text Secondary**: `text-slate-500`
- **Borders**: `border-slate-200`

### Dark Mode (Enabled by `.dark` class)
- **Background**: `dark:bg-background-dark` (`#0d1117`)
- **Surfaces**: `dark:bg-surface-dark` (`#161b22`)
- **Text Primary**: `dark:text-text-light` (`#e6edf3`)
- **Text Secondary**: `dark:text-text-dim` (`#8b949e`)
- **Borders**: `dark:border-border-dark` (`#30363d`)

## 📋 Conversion Pattern Template

Here's the pattern to apply to every component/page:

```tsx
// BEFORE (Light mode only):
<div className="bg-white text-slate-900 border border-slate-200">
  <h2 className="text-2xl font-bold text-slate-900">Title</h2>
  <p className="text-slate-500">Subtitle</p>
  <button className="bg-blue-500 text-white hover:bg-blue-600">
    Action
  </button>
</div>

// AFTER (Light + Dark mode):
<div className="bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light border border-slate-200 dark:border-border-dark">
  <h2 className="text-2xl font-bold text-slate-900 dark:text-text-light">Title</h2>
  <p className="text-slate-500 dark:text-text-dim">Subtitle</p>
  <button className="bg-blue-500 text-white hover:bg-blue-600 dark:bg-blue-600 dark:hover:bg-blue-700">
    Action
  </button>
</div>
```

## 🛠️ Using Helper Functions

For complex patterns, use the helper functions from `src/utils/darkModeHelpers.ts`:

```tsx
import { 
  getCardClasses, 
  getTextClasses, 
  getBadgeClasses,
  getSectionHeaderClasses,
  getButtonClasses 
} from '@/utils/darkModeHelpers';

export function MyComponent() {
  return (
    <div className={getCardClasses()}>
      <h3 className={getTextClasses('primary')}>Title</h3>
      <p className={getTextClasses('secondary')}>Subtitle</p>
      <span className={getBadgeClasses('success')}>Status</span>
      <button className={getButtonClasses('primary')}>Action</button>
    </div>
  );
}
```

## 🎨 Common Pattern Conversions

### Pattern 1: Card with Title and Description
```tsx
// Before
<div className="bg-white p-5 rounded-lg border border-slate-200">
  <h3 className="text-slate-900 font-bold">Title</h3>
  <p className="text-slate-500 text-sm">Description</p>
</div>

// After
<div className="bg-white dark:bg-surface-dark p-5 rounded-lg border border-slate-200 dark:border-border-dark">
  <h3 className="text-slate-900 dark:text-text-light font-bold">Title</h3>
  <p className="text-slate-500 dark:text-text-dim text-sm">Description</p>
</div>

// Or using helper:
<div className={getCardClasses()}>
  <h3 className={`font-bold ${getTextClasses('primary')}`}>Title</h3>
  <p className={getTextClasses('secondary')}>Description</p>
</div>
```

### Pattern 2: Colored Section Headers (like Entity Details)
```tsx
// Before
<div className="p-4 bg-amber-50 border-b border-amber-200">
  <div className="h-10 w-10 rounded-full bg-amber-100 text-amber-600">
    <Icon />
  </div>
  <h3 className="text-slate-900">Section Title</h3>
</div>

// After
<div className={getSectionHeaderClasses('amber').container}>
  <div className={getSectionHeaderClasses('amber').iconBg}>
    <div className={getSectionHeaderClasses('amber').iconText}>
      <Icon />
    </div>
  </div>
  <h3 className={getSectionHeaderClasses('amber').title}>Section Title</h3>
</div>
```

### Pattern 3: Status Badges
```tsx
// Before
<span className="px-2 py-1 rounded text-xs font-bold bg-red-100 text-red-700">
  Error
</span>

// After
<span className={getBadgeClasses('error')}>Error</span>
```

### Pattern 4: Forms & Inputs
```tsx
// Before
<input 
  className="w-full px-4 py-2 rounded border border-slate-300 bg-white text-slate-800 placeholder-slate-400"
  placeholder="Search..."
/>

// After
<input 
  className={getInputClasses()}
  placeholder="Search..."
/>
```

### Pattern 5: Buttons
```tsx
// Before (Primary)
<button className="px-4 py-2 rounded bg-blue-500 text-white hover:bg-blue-600">
  Click me
</button>

// After
<button className={getButtonClasses('primary')}>Click me</button>
```

### Pattern 6: Tables
```tsx
const tableClasses = getTableClasses();

<table className={tableClasses.table}>
  <thead className={tableClasses.thead}>
    <tr>
      <th className={tableClasses.th}>Column 1</th>
    </tr>
  </thead>
  <tbody className={tableClasses.tbody}>
    <tr className={tableClasses.tr}>
      <td className={tableClasses.td}>Data</td>
    </tr>
  </tbody>
</table>
```

## 📍 Key Pages to Convert (Priority Order)

1. **EntityDetailsPage.tsx** - Your main example page
2. **BundleListPage.tsx** - Catalog browsing
3. **BundleEditor.tsx** - Editing interface
4. **PolicyManagementPage.tsx** - Policy management
5. **DriftReportDashboard.tsx** - Dashboard
6. **SchemaExplorer.tsx** - Schema browsing
7. **ValidationRulesPage.tsx** - Validation rules
8. **RoleListPage.tsx** - Role management
9. **BusinessGlossaryPage.tsx** - Glossary management
10. **All remaining pages** - Systematic rollout

## 🔍 Search & Replace Tips

### Using VS Code Find & Replace with Regex

#### Replace all `text-slate-900` with dark variant:
```
Find: text-slate-900(?!['\"])
Replace: text-slate-900 dark:text-text-light
```

#### Replace all `bg-white` with dark variant:
```
Find: bg-white(?!['\"])
Replace: bg-white dark:bg-surface-dark
```

#### Replace all `border-slate-200` with dark variant:
```
Find: border-slate-200(?!['\"])
Replace: border-slate-200 dark:border-border-dark
```

## 💡 Implementation Tips

### 1. Start with Container Elements
Convert the outermost `<div>` elements first, then work inward.

### 2. Group Related Elements
Update parent and child colors together to ensure consistency.

### 3. Test Frequently
Use the toggle button to switch between light/dark modes while editing.

### 4. Follow the Pattern
Use the Tailwind pattern: `<light-class> dark:<dark-class>`

### 5. Use Helpers for Complex Components
For components with many color variations, use the helper functions to keep code DRY.

### 6. Handle Nested Components
Make sure each nested component properly inherits or applies dark mode classes.

### 7. Test Contrast
Ensure text color has sufficient contrast in both modes (WCAG AA: 4.5:1 for text).

## 🎯 Step-by-Step Example: Converting EntityDetailsPage

### Before (Selected snippet):
```tsx
<div className="flex flex-col gap-4 rounded-xl bg-white border border-slate-200 overflow-hidden">
  <div className="flex items-center gap-4 p-4 bg-amber-50 border-b border-amber-200">
    <h3 className="text-slate-900 text-base font-bold">Direct Assignment</h3>
    <p className="text-slate-500 text-sm">Rules applied directly</p>
  </div>
</div>
```

### After (With dark mode):
```tsx
<div className="flex flex-col gap-4 rounded-xl bg-white dark:bg-surface-dark border border-slate-200 dark:border-border-dark overflow-hidden">
  <div className={`flex items-center gap-4 p-4 ${getSectionHeaderClasses('amber').container}`}>
    <h3 className={getSectionHeaderClasses('amber').title}>Direct Assignment</h3>
    <p className={getTextClasses('secondary')}>Rules applied directly</p>
  </div>
</div>
```

## 🧪 Testing Your Changes

### 1. Toggle Theme Button
Click the theme toggle button in the navbar to switch between light and dark modes.

### 2. Check System Preference
- **Mac**: System Preferences > General > Appearance
- **Windows**: Settings > Personalization > Colors > Choose a color

### 3. Browser DevTools
```js
// In browser console, manually toggle dark mode:
document.documentElement.classList.toggle('dark');

// Check if dark mode is active:
document.documentElement.classList.contains('dark');
```

### 4. Visual Regression Testing
- Compare side-by-side: light mode on left, dark mode on right
- Check for text contrast issues
- Ensure all interactive elements are visible
- Verify icons and images look good in both modes

## 📚 Available Helper Functions

Quick reference of all helpers in `darkModeHelpers.ts`:

```tsx
// Containers
getCardClasses()
getBackgroundClasses()
getPageBackgroundClasses()

// Text
getTextClasses('primary' | 'secondary' | 'muted')
getHeaderClasses('h1' | 'h2' | 'h3')
getLabelClasses()
getHelpTextClasses()

// Interactive
getInputClasses()
getButtonClasses('primary' | 'secondary' | 'ghost')
getFormControlClasses()
getHoverClasses()
getFocusClasses()

// Visual Elements
getBadgeClasses('error' | 'warning' | 'info' | 'success')
getSectionHeaderClasses('amber' | 'emerald' | 'violet' | 'blue' | 'red')
getBorderClasses('default' | 'light' | 'subtle')
getDividerClasses()

// Complex
getTableClasses()
getTabClasses()
getModalClasses()
getAlertClasses('error' | 'warning' | 'success' | 'info')
getCodeBlockClasses()
getCodeTextClasses()

// Utilities
combineClasses(...classes)
getResponsiveClasses(mobileClass, desktopClass)
```

## 🚀 Deployment Checklist

- [ ] All pages converted to support `dark:` classes
- [ ] ThemeToggleButton visible and functional in navbar
- [ ] Theme preference persists across sessions (localStorage)
- [ ] System preference detected on first visit
- [ ] Text contrast meets WCAG AA standards in both modes
- [ ] All icons visible and properly colored in both modes
- [ ] Images have proper background fallbacks
- [ ] Modals and overlays properly styled
- [ ] Form inputs accessible in both modes
- [ ] Hover/focus states work in both modes
- [ ] No hardcoded color values remaining
- [ ] All color classes use CSS variables or helpers
- [ ] Tested on Chrome, Firefox, Safari, Edge
- [ ] Tested on light and dark system preferences

## 📞 Support & Reference

- **Pattern Reference**: See `ENTITY_DETAILS_DARK_MODE_PATTERN.md`
- **Example Component**: `src/components/ExampleThemeComponent.tsx`
- **Theme Context**: `src/contexts/ThemeContext.tsx`
- **Helpers**: `src/utils/darkModeHelpers.ts`

---

**Next Steps:**
1. Start with EntityDetailsPage.tsx (you already have the HTML example!)
2. Use the helper functions for consistency
3. Test light/dark toggle in navbar
4. Roll out to remaining pages systematically

Happy dark mode conversion! 🌙
