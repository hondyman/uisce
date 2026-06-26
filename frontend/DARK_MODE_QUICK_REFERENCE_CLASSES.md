# Dark Mode Tailwind Classes - Quick Reference

**Copy and paste these patterns directly into your components!**

## 🎨 Most Used Color Classes

### Backgrounds
```
Light Only:        bg-white / bg-slate-50 / bg-slate-100
Light + Dark:      bg-white dark:bg-surface-dark
                   bg-slate-50 dark:bg-slate-900/20
                   bg-background-light dark:bg-background-dark
```

### Text Colors
```
Primary Text:      text-slate-900 dark:text-text-light
Secondary Text:    text-slate-500 dark:text-text-dim
Muted Text:        text-slate-400 dark:text-text-dim/70
Hover Text:        hover:text-slate-700 dark:hover:text-text-light
```

### Borders
```
Default Border:    border border-slate-200 dark:border-border-dark
Light Border:      border border-slate-100 dark:border-border-dark/50
Subtle Border:     border-t border-slate-200 dark:border-border-dark
```

## 🏗️ Common Component Patterns

### Card/Box
```
<div className="rounded-lg border bg-white dark:bg-surface-dark p-5 border-slate-200 dark:border-border-dark">
```

### Header/Title
```
<h2 className="text-2xl font-bold text-slate-900 dark:text-text-light">
<h3 className="text-lg font-semibold text-slate-900 dark:text-text-light">
```

### Subtitle/Help Text
```
<p className="text-slate-500 dark:text-text-dim text-sm">
```

### Input/Textarea
```
<input className="w-full rounded-lg border px-4 py-2 border-slate-300 bg-white text-slate-800 placeholder-slate-400 dark:border-border-dark dark:bg-surface-dark dark:text-text-light dark:placeholder-text-dim focus:border-primary dark:focus:border-primary transition-colors" />
```

### Button (Primary)
```
<button className="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 dark:hover:bg-primary/80 font-medium transition-colors">
```

### Button (Secondary)
```
<button className="px-4 py-2 rounded-lg bg-slate-200 text-slate-900 hover:bg-slate-300 dark:bg-surface-dark dark:text-text-light dark:hover:bg-slate-700 transition-colors">
```

### Badge/Status Chip
```
Error:       <span className="px-2 py-1 rounded text-xs font-bold bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300">
Warning:     <span className="px-2 py-1 rounded text-xs font-bold bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300">
Success:     <span className="px-2 py-1 rounded text-xs font-bold bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300">
Info:        <span className="px-2 py-1 rounded text-xs font-bold bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300">
```

### Hover State
```
<div className="rounded p-3 hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer">
```

### Divider/Separator
```
<div className="border-t border-slate-200 dark:border-border-dark my-4">
```

### Section Header (Colored)
```
Amber:   <div className="p-4 bg-amber-50 border-b border-amber-200 dark:bg-amber-900/20 dark:border-amber-500/30">
Emerald: <div className="p-4 bg-emerald-50 border-b border-emerald-200 dark:bg-emerald-900/20 dark:border-emerald-500/30">
Violet:  <div className="p-4 bg-violet-50 border-b border-violet-200 dark:bg-violet-900/20 dark:border-violet-500/30">
```

### Icon Container (Colored)
```
Amber:   <div className="h-10 w-10 rounded-full bg-amber-100 dark:bg-amber-500/20 text-amber-600 dark:text-amber-400">
Emerald: <div className="h-10 w-10 rounded-full bg-emerald-100 dark:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400">
Violet:  <div className="h-10 w-10 rounded-full bg-violet-100 dark:bg-violet-500/20 text-violet-600 dark:text-violet-400">
```

### Table Header
```
<thead className="bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-border-dark">
  <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-text-light">
```

### Table Row
```
<tr className="border-b border-slate-200 dark:border-border-dark hover:bg-slate-50 dark:hover:bg-slate-800/30 transition-colors">
  <td className="px-4 py-3 text-sm text-slate-700 dark:text-text-light">
```

### Alert/Notification Box
```
Error:   <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-500/30 text-red-800 dark:text-red-200 px-4 py-3 rounded-lg">
Warning: <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-500/30 text-amber-800 dark:text-amber-200 px-4 py-3 rounded-lg">
Success: <div className="bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-500/30 text-emerald-800 dark:text-emerald-200 px-4 py-3 rounded-lg">
Info:    <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-500/30 text-blue-800 dark:text-blue-200 px-4 py-3 rounded-lg">
```

### Code Block
```
<div className="bg-slate-100 dark:bg-background-dark border border-slate-200 dark:border-border-dark p-4 rounded-lg overflow-x-auto">
  <code className="text-slate-800 dark:text-slate-200 font-mono text-sm">
```

### Form Label
```
<label className="text-sm font-medium text-slate-700 dark:text-text-light">
```

### Form Help Text
```
<p className="text-xs text-slate-500 dark:text-text-dim">
```

### Checkbox/Radio
```
<input className="h-4 w-4 rounded border-slate-300 dark:border-border-dark dark:bg-surface-dark text-primary dark:text-primary focus:ring-primary/50 dark:focus:ring-primary/50" type="checkbox" />
```

### Dropdown/Select
```
<select className="rounded-lg border px-3 py-2 border-slate-300 bg-white text-slate-800 dark:border-border-dark dark:bg-surface-dark dark:text-text-light focus:border-primary dark:focus:border-primary">
```

## 🔗 Using Helper Functions

Instead of writing long class strings, use these helpers:

```tsx
import { 
  getCardClasses,
  getTextClasses,
  getButtonClasses,
  getBadgeClasses,
  getSectionHeaderClasses,
  getInputClasses,
  getBorderClasses
} from '@/utils/darkModeHelpers';

// Card
<div className={getCardClasses()}>

// Text
<p className={getTextClasses('primary')}>  // 'primary' | 'secondary' | 'muted'

// Button
<button className={getButtonClasses('primary')}>  // 'primary' | 'secondary' | 'ghost'

// Badge
<span className={getBadgeClasses('error')}>  // 'error' | 'warning' | 'info' | 'success'

// Section Header
<div className={getSectionHeaderClasses('amber').container}>  // 'amber' | 'emerald' | 'violet' | 'blue' | 'red'

// Input
<input className={getInputClasses()} />

// Border
<div className={getBorderClasses()}>  // 'default' | 'light' | 'subtle'
```

## 📋 Conversion Checklist

When updating a component, check these classes:

- [ ] `bg-white` → add `dark:bg-surface-dark`
- [ ] `bg-slate-*` → add `dark:bg-slate-*` (adjusted shade)
- [ ] `text-slate-900` → add `dark:text-text-light`
- [ ] `text-slate-500` → add `dark:text-text-dim`
- [ ] `border-slate-200` → add `dark:border-border-dark`
- [ ] `hover:bg-*` → add `dark:hover:bg-*`
- [ ] `hover:text-*` → add `dark:hover:text-*`
- [ ] Badges → add dark opacity variants (`/50`, `/30`)
- [ ] Colored sections → add dark tinted backgrounds

## 🎯 Color Value Reference

### Custom Colors (from tailwind.config.js)
```
background-light: #f8fafc
background-dark: #0d1117
surface-dark: #161b22
border-dark: #30363d
text-light: #e6edf3
text-dim: #8b949e
```

### Standard Tailwind Colors with Dark Mode
```
slate-50:    #f8fafc         dark: use background-dark variants
slate-100:   #f1f5f9         dark: use surface-dark (slightlybrighter)
slate-200:   #e2e8f0         dark: use border-dark
slate-500:   #64748b         dark: use text-dim
slate-900:   #0f172a         dark: use text-light

red-100:     #fee2e2         dark: red-900/50
amber-100:   #fef3c7         dark: amber-900/50
emerald-100: #d1fae5         dark: emerald-900/50
blue-100:    #dbeafe         dark: blue-900/50
```

## 🚀 Pro Tips

1. **Use CSS Variables**: For consistent theming, use the HSL variables from `index.css`
2. **Group Conversions**: Convert parent + all children at once
3. **Test Both Modes**: Always toggle and verify
4. **Use Helpers**: For complex components, helpers reduce code duplication
5. **Opacity Variants**: Use `/20`, `/30`, `/50` for tinted colored sections in dark mode
6. **Contrast Check**: Ensure WCAG AA contrast (4.5:1 for text)

## 📞 Examples

See these files for working examples:
- `src/components/ExampleThemeComponent.tsx` - Full component examples
- `ENTITY_DETAILS_DARK_MODE_PATTERN.md` - Detailed patterns
- Your light/dark HTML examples from Entity Details page

---

**Copy-paste these patterns and modify as needed. Dark mode will work automatically! 🌙**
