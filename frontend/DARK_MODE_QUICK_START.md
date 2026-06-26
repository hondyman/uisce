# Dark Mode - Quick Start Guide

Get dark mode working in your app in **5 minutes**! 🌙

## Step 1: Add Toggle Button to Navigation (2 min)

Find your main navigation component (likely in `src/components/MainNavigation.tsx` or similar):

```tsx
import { ThemeToggleButton } from './ThemeToggleButton';

export function MainNavigation() {
  return (
    <nav>
      {/* ...other nav items... */}
      <ThemeToggleButton /> {/* Add this line */}
    </nav>
  );
}
```

## Step 2: Update Component Styles (3 min)

For any component that needs dark mode support, add `dark:` classes:

**Before:**
```tsx
<div className="bg-white text-black">
  <p className="text-gray-600">Text</p>
</div>
```

**After:**
```tsx
<div className="bg-white dark:bg-slate-900 text-black dark:text-white">
  <p className="text-gray-600 dark:text-gray-400">Text</p>
</div>
```

## That's It! 🎉

Your app now supports:
- ✅ Dark mode toggle
- ✅ Light mode toggle
- ✅ System preference detection
- ✅ Preference persistence (survives page reload)

## Quick Reference

| Want... | Use... | Example |
|---------|--------|---------|
| Background | `bg-background dark:bg-...` | `className="bg-background dark:bg-slate-900"` |
| Text | `text-foreground dark:text-...` | `className="text-foreground"` |
| Cards | `bg-card dark:...` | `className="bg-card border border-border"` |
| Buttons | `bg-primary dark:...` | `className="bg-primary text-primary-foreground"` |
| Borders | `border-border dark:...` | `className="border border-border"` |

## Testing

1. **Click the theme toggle** in your navigation bar
2. **Refresh the page** - theme preference is saved!
3. **Change OS theme** - if set to "system", app responds automatically

## Troubleshooting

**Not seeing dark mode?**
- Clear browser cache (`Cmd+Shift+Delete`)
- Check console for errors
- Verify you have the `.dark` CSS classes in your components

**Colors look weird?**
- Check text contrast
- Use the CSS variables from `src/index.css` instead of hardcoded colors
- Example: `bg-background` instead of `bg-white`

## Common Patterns

### Card Component
```tsx
<div className="bg-card text-card-foreground border border-border rounded-lg p-4">
  Content here
</div>
```

### Button Component
```tsx
<button className="bg-primary text-primary-foreground hover:opacity-90 px-4 py-2 rounded">
  Click me
</button>
```

### Status Indicator
```tsx
<span className="bg-green-100 dark:bg-green-950 text-green-700 dark:text-green-200">
  Active
</span>
```

## Next Steps

- Read `DARK_MODE_IMPLEMENTATION.md` for comprehensive guide
- Audit your pages and add `dark:` classes where needed
- Test text contrast in dark mode
- Get user feedback on theme appearance

---

**Need help?** Check `src/contexts/ThemeContext.tsx` or `src/components/ThemeToggleButton.tsx` for the implementation.
