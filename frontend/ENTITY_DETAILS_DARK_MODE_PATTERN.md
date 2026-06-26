# Entity Details Dark Mode Pattern Reference

This document captures the **exact dark mode approach** used in your Entity Details page examples. Use this as your reference template for converting React components.

## 🎨 Color Strategy

Your dark mode uses a **semantic color system** with custom Tailwind extensions:

### Light Mode (Default)
```js
colors: {
  "primary": "#2762ec",           // Core brand blue
  "background-light": "#f8fafc",  // Page background
}
```

### Dark Mode Enhancements
```js
colors: {
  "primary": "#4f86f7",           // Lighter blue for dark bg
  "background-dark": "#0d1117",   // Very dark navy (almost black)
  "surface-dark": "#161b22",      // Slightly lighter surface
  "border-dark": "#30363d",       // Subtle dark borders
  "text-light": "#e6edf3",        // Main text (light)
  "text-dim": "#8b949e",          // Secondary text (dimmed)
}
```

## 🔄 HTML-to-Tailwind Mapping Patterns

### Pattern 1: Background Colors
**Light Mode:**
```html
<div class="bg-background-light"><!-- #f8fafc --></div>
<div class="bg-white"><!-- For cards/surfaces --></div>
```

**Dark Mode:**
```html
<div class="bg-background-dark"><!-- #0d1117 --></div>
<div class="bg-surface-dark"><!-- #161b22 for cards --></div>
```

**React (Tailwind with `dark:` prefix):**
```jsx
<div className="bg-background-light dark:bg-background-dark">
  {/* Switches automatically based on .dark class on <html> */}
</div>

<div className="bg-white dark:bg-surface-dark">
  {/* White in light mode, surface-dark in dark mode */}
</div>
```

---

### Pattern 2: Text Colors
**Light Mode:**
```html
<p class="text-slate-900"><!-- Dark text --></p>
<p class="text-slate-500"><!-- Muted text --></p>
```

**Dark Mode:**
```html
<p class="text-text-light"><!-- #e6edf3 --></p>
<p class="text-text-dim"><!-- #8b949e --></p>
```

**React (Tailwind with `dark:` prefix):**
```jsx
<p className="text-slate-900 dark:text-text-light">
  {/* Switches primary text color */}
</p>

<p className="text-slate-500 dark:text-text-dim">
  {/* Switches secondary/muted text */}
</p>
```

---

### Pattern 3: Borders
**Light Mode:**
```html
<div class="border border-slate-200"><!-- Subtle light border --></div>
```

**Dark Mode:**
```html
<div class="border border-border-dark"><!-- #30363d --></div>
```

**React:**
```jsx
<div className="border border-slate-200 dark:border-border-dark">
  {/* Border adapts to theme */}
</div>
```

---

### Pattern 4: Hover States
**Light Mode:**
```html
<button class="text-slate-500 hover:text-slate-900">
  {/* Gray → Dark gray on hover */}
</button>
```

**Dark Mode:**
```html
<button class="text-text-dim hover:text-text-light">
  {/* Dim → Light text on hover */}
</button>
```

**React:**
```jsx
<button className="text-slate-500 hover:text-slate-900 dark:text-text-dim dark:hover:text-text-light transition-colors">
  {/* All states adapt automatically */}
</button>
```

---

### Pattern 5: Status/Severity Badges

**Light Mode (Red Error Badge):**
```html
<span class="bg-red-100 text-red-700">Error</span>
```

**Dark Mode (Tinted Red Badge):**
```html
<span class="bg-red-900/50 text-red-300">Error</span>
```

**React with All Severities:**
```jsx
const getBadgeClasses = (severity) => {
  const baseClasses = "text-xs font-semibold uppercase px-2 py-1 rounded";
  
  switch(severity) {
    case 'error':
      return `${baseClasses} bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300`;
    case 'warning':
      return `${baseClasses} bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300`;
    case 'info':
      return `${baseClasses} bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300`;
    case 'success':
      return `${baseClasses} bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300`;
    default:
      return baseClasses;
  }
};

<span className={getBadgeClasses('error')}>Error</span>
```

---

### Pattern 6: Section Headers with Colored Backgrounds

**Light Mode (Amber Section):**
```html
<div class="bg-amber-50 border-b border-amber-200">
  <div class="bg-amber-100 text-amber-600"><!-- Icon circle --></div>
  <h3 class="text-slate-900">Direct Assignment</h3>
</div>
```

**Dark Mode (Same with Dark Tint):**
```html
<div class="bg-amber-900/20 border-b border-amber-500/30">
  <div class="bg-amber-500/20 text-amber-400"><!-- Icon circle --></div>
  <h3 class="text-text-light">Direct Assignment</h3>
</div>
```

**React Helper Function:**
```jsx
const getSectionHeaderClasses = (color) => {
  const colorMap = {
    amber: {
      light: 'bg-amber-50 border-amber-200',
      dark: 'dark:bg-amber-900/20 dark:border-amber-500/30',
      iconBg: 'bg-amber-100 dark:bg-amber-500/20',
      iconText: 'text-amber-600 dark:text-amber-400',
    },
    emerald: {
      light: 'bg-emerald-50 border-emerald-200',
      dark: 'dark:bg-emerald-900/20 dark:border-emerald-500/30',
      iconBg: 'bg-emerald-100 dark:bg-emerald-500/20',
      iconText: 'text-emerald-600 dark:text-emerald-400',
    },
    violet: {
      light: 'bg-violet-50 border-violet-200',
      dark: 'dark:bg-violet-900/20 dark:border-violet-500/30',
      iconBg: 'bg-violet-100 dark:bg-violet-500/20',
      iconText: 'text-violet-600 dark:text-violet-400',
    },
  };
  return colorMap[color] || colorMap.amber;
};

// Usage:
const header = getSectionHeaderClasses('amber');
<div className={`p-4 border-b ${header.light} ${header.dark}`}>
  <div className={`h-10 w-10 rounded-full ${header.iconBg} ${header.iconText}`}>
    {/* Icon */}
  </div>
</div>
```

---

### Pattern 7: Interactive Elements (Buttons, Inputs, Checkboxes)

**Light Mode Input:**
```html
<input 
  class="border border-slate-300 bg-white text-slate-800 placeholder-slate-400 focus:border-primary"
  placeholder="Search..."
/>
```

**Dark Mode Input:**
```html
<input 
  class="border border-border-dark bg-surface-dark text-text-light placeholder-text-dim focus:border-primary"
  placeholder="Search..."
/>
```

**React:**
```jsx
<input
  className="w-full h-12 rounded-lg border px-4 py-2
    border-slate-300 bg-white text-slate-800 placeholder-slate-400
    dark:border-border-dark dark:bg-surface-dark dark:text-text-light dark:placeholder-text-dim
    focus:border-primary dark:focus:border-primary
    transition-colors"
  placeholder="Search rules..."
  type="text"
/>
```

---

### Pattern 8: Hover States for Surface Elements

**Light Mode:**
```html
<div class="hover:bg-slate-50"><!-- White → Light gray --></div>
```

**Dark Mode:**
```html
<div class="hover:bg-slate-800/50"><!-- Surface → Darker surface --></div>
```

**React:**
```jsx
<div className="p-4 hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer">
  {/* Clickable item that dims/brightens on hover */}
</div>
```

---

## 📋 Quick Conversion Checklist

When converting a light-mode component to support dark mode:

### Step 1: Add Dark Mode Color Classes
- [ ] Replace all `bg-white` with `bg-white dark:bg-surface-dark`
- [ ] Replace all `text-slate-900` with `text-slate-900 dark:text-text-light`
- [ ] Replace all `text-slate-500` with `text-slate-500 dark:text-text-dim`
- [ ] Replace all `border-slate-200` with `border-slate-200 dark:border-border-dark`

### Step 2: Handle Status/Severity Colors
- [ ] Use helper functions for badge classes (error/warning/info/success)
- [ ] Apply `/50` opacity and lighter shades for dark mode backgrounds

### Step 3: Update Interactive States
- [ ] Add `dark:` variants for `hover:`, `focus:`, `active:` states
- [ ] Test keyboard navigation in both modes

### Step 4: Test Semantic Sections
- [ ] Colored section headers adapt using color map helpers
- [ ] Icon colors remain visible in both modes
- [ ] Text contrast is maintained (WCAG AA minimum)

---

## 🛠️ Integration with Your React Setup

### Step 1: Add Custom Colors to Tailwind Config
In `tailwind.config.js`:
```js
export default {
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'primary': '#2762ec',
        'background-light': '#f8fafc',
        'background-dark': '#0d1117',
        'surface-dark': '#161b22',
        'border-dark': '#30363d',
        'text-light': '#e6edf3',
        'text-dim': '#8b949e',
      },
    },
  },
};
```

### Step 2: Use in React Components
```jsx
export function EntityDetailsCard() {
  return (
    <div className="rounded-lg border bg-white p-5 border-slate-200 dark:border-border-dark dark:bg-surface-dark">
      <h3 className="text-slate-900 dark:text-text-light">Card Title</h3>
      <p className="text-slate-500 dark:text-text-dim">Secondary text</p>
    </div>
  );
}
```

### Step 3: Wrap App with Theme Provider
In `main.tsx`, ensure you have:
```jsx
import { ThemeProvider } from './contexts/ThemeContext';

function Main() {
  return (
    <ThemeProvider>
      <YourApp />
    </ThemeProvider>
  );
}
```

The `.dark` class will be added/removed on the `<html>` element automatically.

---

## 💡 Pro Tips

1. **Opacity for Colored Sections**: Use `bg-color-900/20` and `text-color-400` in dark mode for tinted backgrounds
2. **Borders**: Dark mode uses `border-dark` for consistency; avoid slate grays
3. **Text Hierarchy**: `text-light` for primary, `text-dim` for secondary
4. **Primary Button**: Stays same color but may need slight lightening (your example uses `#4f86f7` vs `#2762ec`)
5. **Always Pair**: Use `dark:` variant immediately after the light variant to stay organized

---

## 📸 Summary Table

| Element | Light Mode | Dark Mode | Tailwind Pattern |
|---------|-----------|-----------|-------------------|
| Page Background | `#f8fafc` | `#0d1117` | `bg-background-light dark:bg-background-dark` |
| Card/Surface | `white` | `#161b22` | `bg-white dark:bg-surface-dark` |
| Primary Text | `text-slate-900` | `text-text-light` | `text-slate-900 dark:text-text-light` |
| Secondary Text | `text-slate-500` | `text-text-dim` | `text-slate-500 dark:text-text-dim` |
| Borders | `border-slate-200` | `border-border-dark` | `border-slate-200 dark:border-border-dark` |
| Error Badge | `bg-red-100 text-red-700` | `bg-red-900/50 text-red-300` | Multi-class pattern |
| Hover Surface | `hover:bg-slate-50` | `hover:bg-slate-800/50` | `hover:bg-slate-50 dark:hover:bg-slate-800/50` |
