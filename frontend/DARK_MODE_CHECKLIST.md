# Dark Mode Implementation Checklist

Follow this checklist to fully implement dark mode across your platform.

## Phase 1: Foundation (✅ Complete - Already Done)

- [x] Create ThemeContext with persistence
- [x] Create ThemeToggleButton component
- [x] Update main.tsx with theme provider
- [x] Enhance CSS variables in index.css
- [x] Create documentation

## Phase 2: Navigation & Header (1 Hour)

Navigate to your main navigation components and add the theme toggle:

```tsx
import { ThemeToggleButton } from './ThemeToggleButton';
```

**Files to update:**
- [ ] `src/components/MainNavigation.tsx` - Add `<ThemeToggleButton />` to header
- [ ] `src/components/MobileResponsiveNavigation.tsx` - If you have mobile nav
- [ ] Any other header/navbar components

**Example location in header:**
```tsx
<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
  <ThemeToggleButton />  {/* Add this */}
  <IconButton>...</IconButton>
  {/* Other buttons */}
</Box>
```

## Phase 3: Core Pages (2-3 Hours)

Update the main pages your users see most:

### Authentication & Layout Pages
- [ ] `src/pages/AuthPage.tsx`
- [ ] `src/components/ErrorBoundary.tsx`
- [ ] `src/components/ProtectedRoute.tsx`

### Dashboard Pages
- [ ] `src/features/upgrade/pages/VersionsPage.tsx`
- [ ] `src/features/fabric/pages/PolicyManagementPage.tsx`
- [ ] Any custom dashboard components

### List & Detail Pages
- [ ] `src/pages/EntityDetailsPage.tsx` ⭐ (Start here - file you're editing)
- [ ] `src/features/catalog/pages/APICatalogPage.tsx`
- [ ] Any browse/search pages

**For each page:**
1. Find all elements with hardcoded colors
2. Add `dark:` Tailwind classes or use CSS variables
3. Test in both light and dark modes

**Common patterns to update:**
```tsx
// Badge styling
<span className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-200">
  {badge}
</span>

// Card styling
<div className="bg-white dark:bg-slate-800 border border-gray-200 dark:border-gray-700">
  {content}
</div>

// Text styling
<h1 className="text-gray-900 dark:text-white">Title</h1>
<p className="text-gray-600 dark:text-gray-400">Description</p>
```

## Phase 4: Component Library (1-2 Hours)

Update your reusable components:

### UI Components
- [ ] Buttons
- [ ] Cards
- [ ] Input fields
- [ ] Dropdowns/Selects
- [ ] Modals/Dialogs
- [ ] Notifications/Toasts

### Form Components
- [ ] FormControl
- [ ] TextField variations
- [ ] Checkboxes/Radio buttons
- [ ] Switches

### Data Display
- [ ] Tables
- [ ] Lists
- [ ] Trees
- [ ] Data grids

**Create a shared pattern for each component type:**

```tsx
// Button variant
className="bg-blue-600 dark:bg-blue-700 text-white hover:bg-blue-700 dark:hover:bg-blue-800"

// Card variant
className="bg-white dark:bg-slate-900 border border-gray-200 dark:border-gray-800"

// Input variant
className="bg-white dark:bg-slate-800 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white"
```

## Phase 5: CSS Files (1-2 Hours)

Update component-specific CSS files:

- [ ] `src/Explorer.css`
- [ ] `src/pages/TabbedModal/tabs/BusinessTermTree.css`
- [ ] `src/pages/TabbedModal/tabs/DataCatalogTree.css`
- [ ] `src/components/UnifiedSemanticBuilder/ColumnActionsPanel.css`
- [ ] Any other `.css` files

**Pattern for CSS dark mode:**
```css
/* Light mode (default) */
.my-component {
  background: white;
  color: black;
  border: 1px solid #e5e7eb;
}

/* Dark mode */
.dark .my-component {
  background: #1f2937;
  color: white;
  border: 1px solid #374151;
}

/* Even better - use CSS variables */
.my-component {
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  border: 1px solid hsl(var(--border));
}
```

## Phase 6: Advanced Features (Optional - 1-2 Hours)

- [ ] Add theme preference to user settings
- [ ] Add keyboard shortcut for theme toggle (e.g., `Ctrl+Shift+T`)
- [ ] Add theme preview in settings
- [ ] Create custom theme builder
- [ ] Add theme to export/import functionality

## Phase 7: Testing & QA (2-3 Hours)

### Manual Testing
- [ ] Test light mode thoroughly
- [ ] Test dark mode thoroughly
- [ ] Test system preference detection
- [ ] Test theme persistence (refresh page)
- [ ] Test on mobile devices
- [ ] Test in different browsers

### Accessibility Testing
- [ ] Check contrast ratio (minimum 4.5:1)
- [ ] Test with color vision simulator
- [ ] Test with screen readers
- [ ] Test keyboard navigation
- [ ] Run Lighthouse audit

### Browser Testing
- [ ] Chrome/Chromium
- [ ] Firefox
- [ ] Safari
- [ ] Edge
- [ ] Mobile Safari
- [ ] Chrome Mobile

**Debugging commands:**
```javascript
// Check current theme
document.documentElement.className

// Check stored preference
localStorage.getItem('app-theme-preference')

// Manually apply dark mode
document.documentElement.classList.add('dark')

// Check CSS variables
getComputedStyle(document.documentElement).getPropertyValue('--background')
```

## Phase 8: Deployment (30 Minutes)

- [ ] Run lint checks: `npm run lint`
- [ ] Build project: `npm run build`
- [ ] Test build output
- [ ] Deploy to staging
- [ ] Test on staging environment
- [ ] Get stakeholder approval
- [ ] Deploy to production
- [ ] Monitor for issues

## Phase 9: Documentation & Training (1 Hour)

- [ ] Share `DARK_MODE_IMPLEMENTATION.md` with team
- [ ] Share `DARK_MODE_QUICK_START.md` with team
- [ ] Show example at `src/components/ExampleThemeComponent.tsx`
- [ ] Hold brief team training session
- [ ] Add to team wiki/documentation
- [ ] Update contribution guidelines

## Estimated Timeline

| Phase | Time | Priority |
|-------|------|----------|
| Phase 1 | Done ✅ | Complete |
| Phase 2 | 1 hr | 🔴 High |
| Phase 3 | 3 hrs | 🔴 High |
| Phase 4 | 2 hrs | 🟠 Medium |
| Phase 5 | 2 hrs | 🟠 Medium |
| Phase 6 | 2 hrs | 🟡 Low |
| Phase 7 | 3 hrs | 🔴 High |
| Phase 8 | 30m | 🔴 High |
| Phase 9 | 1 hr | 🟡 Low |
| **Total** | **~14 hours** | |

## Priority Implementation Order

**Must Do First (Phase 2):**
- Add toggle button to navigation
- Users need a way to switch themes!

**Should Do Soon (Phase 3):**
- Update main dashboard pages
- Update EntityDetailsPage (you're editing this!)
- Update frequently used pages

**Nice to Have (Phases 4-6):**
- Polish remaining components
- Update custom CSS files
- Add advanced features

**Important (Phase 7):**
- Test everything
- Verify accessibility
- Check all browsers

## Quick Reference: Common Updates

### Change from:
```tsx
<div className="bg-white text-black">
```

### Change to:
```tsx
<div className="bg-white dark:bg-slate-900 text-black dark:text-white">
```

### Or better:
```tsx
<div className="bg-background text-foreground">
```

## Success Criteria

Your dark mode implementation is successful when:

✅ Users see theme toggle in navigation  
✅ Toggle switches between light and dark  
✅ App correctly detects system preference  
✅ User choice persists after page reload  
✅ All pages look good in both modes  
✅ Text contrast meets WCAG AA (4.5:1)  
✅ No console errors  
✅ Works on all major browsers  
✅ Mobile responsive  
✅ Team has documentation  

## Troubleshooting During Implementation

**Issue: Colors not changing**
- Solution: Check that `.dark` class is on `<html>` element
- Debug: `document.documentElement.className`

**Issue: Some components still look wrong**
- Solution: Add `dark:` classes to remaining elements
- Debug: Use browser inspector to check applied classes

**Issue: Performance degradation**
- Solution: Ensure you're using CSS variables, not JS calculations
- Debug: Check DevTools performance tab

**Issue: Mobile theme not switching**
- Solution: Ensure ThemeProvider wraps entire app, not just navigation
- Debug: Test on actual mobile device, not just DevTools

## Getting Help

1. **Check documentation:** `DARK_MODE_IMPLEMENTATION.md`
2. **Review examples:** `src/components/ExampleThemeComponent.tsx`
3. **Search codebase:** Look for existing `dark:` usage
4. **Browser DevTools:** Inspect `.dark` class and CSS variables
5. **Console debugging:** Run commands from section above

## Team Coordination

- [ ] Assign team member to lead implementation
- [ ] Create GitHub issue/Jira ticket
- [ ] Set deadline (suggest 2 weeks for full completion)
- [ ] Schedule code review
- [ ] Plan QA testing
- [ ] Communicate launch to users

---

## Progress Tracking

As you complete each phase, check it off here:

- [ ] Phase 1: Foundation ✅
- [ ] Phase 2: Navigation & Header 🔄 
- [ ] Phase 3: Core Pages 🔄
- [ ] Phase 4: Component Library ⏳
- [ ] Phase 5: CSS Files ⏳
- [ ] Phase 6: Advanced Features ⏳
- [ ] Phase 7: Testing & QA ⏳
- [ ] Phase 8: Deployment ⏳
- [ ] Phase 9: Documentation ⏳

**Overall Progress:** Phase 1 Complete! 👏

---

**Last Updated:** November 2024  
**Framework:** React + Tailwind + Material-UI  
**Status:** Foundation Complete, Ready for Rollout
