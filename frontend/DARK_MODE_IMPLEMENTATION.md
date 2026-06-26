# Dark Mode Implementation Guide

Your platform now has complete dark mode support! Here's how it works and how to use it.

## 🎨 How It Works

The dark mode system consists of three main components:

### 1. **ThemeContext** (`src/contexts/ThemeContext.tsx`)
- Manages theme state (light, dark, or system preference)
- Persists user preference to localStorage
- Automatically applies theme to the DOM by adding/removing the `.dark` class
- Provides hooks: `useTheme()` to access theme state

### 2. **Theme Setup** (`src/main.tsx`)
- Wraps the entire app with `CustomThemeProvider`
- Integrates with Material-UI's theme system
- Reads the effective theme and passes it to MUI's ThemeProvider

### 3. **CSS Variables** (`src/index.css`)
- Defines HSL color variables for both light and dark modes
- Variables automatically switch when `.dark` class is applied to `<html>`
- All components use these variables through Tailwind CSS

## 🔌 Using the Theme Toggle

### In Navigation/Header Components

```tsx
import { ThemeToggleButton } from './components/ThemeToggleButton';

// Simple toggle (Light ↔ Dark):
<ThemeToggleButton showMenu={false} />

// With menu (Light / Dark / System):
<ThemeToggleButton showMenu={true} />
<ThemeToggleButton /> {/* showMenu={true} is default */}
```

### Update Your Main Navigation

If you have a custom navigation component, add the toggle button:

```tsx
import { ThemeToggleButton } from '../components/ThemeToggleButton';

export const MainNavigation: React.FC = () => {
  return (
    <header>
      {/* ... other nav items ... */}
      <ThemeToggleButton />
    </header>
  );
};
```

### Manual Theme Control in Components

```tsx
import { useTheme } from '../contexts/ThemeContext';

export function MyComponent() {
  const { theme, effectiveTheme, setTheme } = useTheme();

  return (
    <div>
      <p>Current theme: {effectiveTheme}</p>
      <p>Preference: {theme}</p>
      
      <button onClick={() => setTheme('light')}>Light</button>
      <button onClick={() => setTheme('dark')}>Dark</button>
      <button onClick={() => setTheme('system')}>System</button>
    </div>
  );
}
```

## 🎨 Styling Components for Dark Mode

### Using Tailwind Dark Mode Classes

The Tailwind CSS `dark:` prefix automatically applies to elements when `.dark` is on the `<html>`:

```tsx
// React JSX example
<div className="bg-white dark:bg-slate-900 text-black dark:text-white">
  <h1 className="text-gray-900 dark:text-gray-50">Title</h1>
  <p className="text-gray-600 dark:text-gray-400">Description</p>
</div>
```

### Using CSS Variables

All color variables defined in `src/index.css` automatically switch themes:

```tsx
// Using Tailwind's color system
<div className="bg-background text-foreground border border-border">
  <h1 className="text-primary">Title</h1>
  <p className="text-muted-foreground">Muted text</p>
</div>
```

### Custom CSS Files

For component-specific CSS files, use the `.dark` class selector:

```css
/* MyComponent.css */
.my-component {
  background: white;
  color: black;
}

.dark .my-component {
  background: #1a1a1a;
  color: white;
}

/* Or with CSS variables */
.my-component {
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  border: 1px solid hsl(var(--border));
}
```

### Using Material-UI with Theme

Material-UI components automatically respond to the theme set in `main.tsx`:

```tsx
import { useTheme } from '@mui/material/styles';
import { Box, Typography } from '@mui/material';

export function MyComponent() {
  const theme = useTheme();

  return (
    <Box sx={{
      backgroundColor: theme.palette.background.paper,
      color: theme.palette.text.primary,
      borderColor: theme.palette.divider,
    }}>
      <Typography variant="h6">
        This automatically responds to the theme!
      </Typography>
    </Box>
  );
}
```

## 🎯 Available Colors

### Primary Colors
- `background` - Main background
- `foreground` - Main text color
- `card` - Card/panel backgrounds
- `card-foreground` - Card text

### Semantic Colors
- `primary` / `primary-foreground` - Primary action color
- `secondary` / `secondary-foreground` - Secondary actions
- `accent` / `accent-foreground` - Accent color
- `destructive` / `destructive-foreground` - Dangerous actions (delete, etc.)
- `muted` / `muted-foreground` - Disabled/secondary text

### UI Colors
- `border` - Border color
- `input` - Input field backgrounds
- `ring` - Focus ring color

## 🌓 System Preference

The theme respects the user's OS preference:

- **Light mode** on your system → Light theme in the app
- **Dark mode** on your system → Dark theme in the app
- **User overrides** → Saved to localStorage and always takes precedence

Users can toggle between light, dark, or system preference using the `ThemeToggleButton`.

## 📋 Migration Checklist

To update your existing components:

- [ ] Add `dark:` Tailwind classes to light-mode-only styles
- [ ] Update custom CSS files to include dark mode selectors
- [ ] Replace hardcoded colors with CSS variable references
- [ ] Add the `ThemeToggleButton` to your main navigation
- [ ] Test in both light and dark modes
- [ ] Check text contrast (should be WCAG AA compliant)

## 🔍 Testing Dark Mode

### In Browser DevTools
```javascript
// Apply dark mode
document.documentElement.classList.add('dark');

// Remove dark mode
document.documentElement.classList.remove('dark');

// Check current theme preference
localStorage.getItem('app-theme-preference');

// Set preference
localStorage.setItem('app-theme-preference', 'dark');
```

### In Your App
1. Click the theme toggle button in the navigation
2. Refresh the page - your preference persists
3. Change your OS theme - if set to "system", the app responds

## 🚀 Performance Tips

1. **Use CSS variables** instead of hardcoded colors - they're automatically swapped
2. **Use Tailwind's `dark:` prefix** - it's the most performant
3. **Avoid `@media (prefers-color-scheme)` queries** - the CSS variables handle it
4. **Memoize styled components** to prevent unnecessary re-renders

## 📚 File Reference

| File | Purpose |
|------|---------|
| `src/contexts/ThemeContext.tsx` | Theme state management |
| `src/components/ThemeToggleButton.tsx` | Toggle button component |
| `src/main.tsx` | App setup with theme provider |
| `src/index.css` | CSS variables for all themes |
| `src/tailwind.config.js` | Tailwind color configuration |

## 🐛 Troubleshooting

### Theme not applying?
1. Verify `ThemeProvider` wraps your entire app in `main.tsx`
2. Check that `src/index.css` is imported
3. Clear browser cache and localStorage

### Toggle button not working?
1. Ensure `ThemeToggleButton` is inside `CustomThemeProvider`
2. Check console for errors
3. Verify `useTheme` hook is accessible

### Colors look wrong in dark mode?
1. Check CSS specificity - inline styles override CSS variables
2. Verify the `.dark` class is on `<html>` element
3. Use browser DevTools to inspect computed styles

## 💡 Best Practices

✅ **Do:**
- Use CSS variables for colors
- Use Tailwind's `dark:` prefix
- Test both modes regularly
- Ensure sufficient contrast in dark mode
- Save theme preference to localStorage

❌ **Don't:**
- Hardcode colors (use variables instead)
- Use inline styles for theme-dependent colors
- Forget to test dark mode
- Ignore accessibility contrast ratios
- Rely only on `@media (prefers-color-scheme)`

## 🎓 Examples

### Simple Card Component
```tsx
export function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-card text-card-foreground border border-border rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-2">{title}</h2>
      {children}
    </div>
  );
}
```

### Button Component with Theme
```tsx
export function Button({ children, variant = 'primary' }: { children: React.ReactNode; variant?: 'primary' | 'secondary' }) {
  const baseClasses = 'px-4 py-2 rounded-lg font-medium transition-colors';
  const variants = {
    primary: 'bg-primary text-primary-foreground hover:opacity-90',
    secondary: 'bg-secondary text-secondary-foreground hover:opacity-90',
  };

  return <button className={`${baseClasses} ${variants[variant]}`}>{children}</button>;
}
```

### Status Badge with Theme
```tsx
export function StatusBadge({ status }: { status: 'success' | 'error' | 'pending' }) {
  const colors = {
    success: 'bg-green-100 dark:bg-green-950 text-green-700 dark:text-green-200',
    error: 'bg-red-100 dark:bg-red-950 text-red-700 dark:text-red-200',
    pending: 'bg-yellow-100 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-200',
  };

  return <span className={`px-2 py-1 rounded-full text-sm ${colors[status]}`}>{status}</span>;
}
```

---

**Last Updated:** November 2024  
**Maintainers:** Your Team
