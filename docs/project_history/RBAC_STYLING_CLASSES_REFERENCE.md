# RBAC Styling Guide - CSS Classes Reference

## Color System

### Text Colors
```css
.text-[#0d131c]      /* Primary Dark Blue - Main text */
.text-[#496a9c]      /* Secondary Blue - Secondary text */
.text-slate-50       /* Near white */
.text-gray-600       /* Fallback gray */
```

### Background Colors
```css
.bg-slate-50         /* Light gray-blue background */
.bg-[#e7ecf4]        /* Primary light blue */
.bg-[#d1dce8]        /* Slightly darker blue (hover) */
.bg-white            /* Pure white for cards */
.bg-gray-50          /* Light gray for alt rows */
```

### Border Colors
```css
.border-b-[#e7ecf4]    /* Light blue bottom border */
.border-[#ced9e8]      /* Gray border */
.border-t-[#ced9e8]    /* Gray top border */
.border-l-blue-600     /* Blue left accent */
.border-l-transparent  /* No left border */
.border-b-[3px]        /* Thick bottom border */
```

---

## Layout Classes

### Main Container
```css
.relative                    /* Position relative for children */
.flex                        /* Flexbox container */
.h-auto min-h-screen        /* Full height minimum */
.w-full                     /* Full width */
.flex-col                   /* Vertical flex direction */
.bg-slate-50               /* Light background */
.overflow-x-hidden         /* Hide horizontal scroll */
```

### Header
```css
.flex.items-center          /* Centered items */
.justify-between            /* Space between */
.whitespace-nowrap          /* No text wrapping */
.border-b.border-solid      /* Bottom border */
.px-10                      /* Padding: 40px horizontal */
.py-3                       /* Padding: 12px vertical */
```

### Content Area
```css
.px-40                      /* Padding: 160px horizontal (centered) */
.flex.flex-1.justify-center /* Flex center */
.py-5                       /* Padding: 20px vertical */
.layout-content-container   /* Max-width: 960px wrapper */
```

### Grid Layout
```css
.grid
.grid-cols-1               /* 1 column on small screens */
.lg:grid-cols-3            /* 3 columns on large screens */
.gap-6                     /* 24px gap between items */
.flex-1                    /* Equal width flex items */
```

---

## Typography Classes

### Headings
```css
.text-[32px]               /* Large heading size */
.font-bold                 /* Bold weight */
.leading-tight             /* Tight line height */
.tracking-[-0.015em]       /* Negative letter spacing */

.text-xl                   /* Medium heading */
.text-lg                   /* Slightly smaller heading */
.text-sm                   /* Small text */
.text-xs                   /* Extra small text */
```

### Font Weights
```css
.font-bold                 /* 700 weight */
.font-medium               /* 500 weight */
.font-normal               /* 400 weight */
```

### Text Alignment
```css
.text-left                 /* Left aligned */
.text-center               /* Center aligned */
.tracking-light            /* Light letter spacing */
.leading-normal            /* Normal line height: 1.5 */
```

---

## Component Styling

### Cards & Containers
```css
.bg-white                  /* White background */
.rounded-2xl               /* Large border radius: 16px */
.rounded-xl                /* Medium border radius: 12px */
.rounded-lg                /* Small border radius: 8px */
.shadow-xl                 /* Large shadow */
.shadow-lg                 /* Medium shadow */
.p-6                       /* Padding: 24px all sides */
.p-4                       /* Padding: 16px all sides */
.p-3                       /* Padding: 12px all sides */
.p-2                       /* Padding: 8px all sides */
```

### Input Fields
```css
.form-input                /* Input field styles */
.flex.w-full               /* Full width flex */
.flex-1                    /* Take remaining space */
.resize-none               /* Disable resize */
.overflow-hidden           /* Hide overflow */
.rounded-lg                /* Rounded corners */
.text-[#0d131c]            /* Dark blue text */
.focus:outline-0           /* Remove outline on focus */
.focus:ring-0              /* Remove ring on focus */
.border-none               /* No border */
.bg-[#e7ecf4]              /* Light blue background */
.focus:border-none         /* No border on focus */
.h-full                    /* Full height */
.placeholder:text-[#496a9c] /* Placeholder color */
.px-4                      /* Horizontal padding: 16px */
.pl-2                      /* Left padding: 8px */
.pl-10                     /* Left padding: 40px */
```

### Buttons
```css
.flex                      /* Flexbox */
.min-w-[84px]              /* Minimum width */
.max-w-[480px]             /* Maximum width */
.cursor-pointer            /* Hand cursor */
.items-center              /* Center items */
.justify-center            /* Center content */
.overflow-hidden           /* Hide overflow */
.rounded-lg                /* Rounded corners */
.h-8                       /* Height: 32px */
.h-10                      /* Height: 40px */
.px-4                      /* Padding horizontal: 16px */

/* States */
.hover:bg-blue-700         /* Hover background */
.transition-all            /* Smooth transition */
.disabled:opacity-50       /* Disabled state */
```

### Tables
```css
.overflow-hidden           /* Hide overflow */
.border.border-[#ced9e8]  /* Gray border */
.bg-slate-50              /* Light background */

/* Table header */
.bg-slate-50              /* Background */
.text-left                /* Left align */
.text-[#0d131c]           /* Dark text */
.w-[400px]                /* Column width */
.text-sm                  /* Small text */
.font-medium              /* Medium weight */
.leading-normal           /* Normal line height */

/* Table rows */
.border-t.border-t-[#ced9e8]  /* Top border */
.hover:bg-gray-50             /* Hover state */
.h-[72px]                     /* Row height */
.text-[#496a9c]               /* Secondary text */
```

### Icons
```css
.size-4                    /* 16px size (width: 4, height: 4) */
.size-10                   /* 40px size */
.w-4.h-4                   /* 16px width and height */
.w-5.h-5                   /* 20px width and height */
.w-6.h-6                   /* 24px width and height */
.w-8.h-8                   /* 32px width and height */
.w-12.h-12                 /* 48px width and height */
.w-16.h-16                 /* 64px width and height */
.w-24.h-24                 /* 96px width and height */

/* Icon colors */
.text-[#496a9c]            /* Blue icons */
.text-gray-400             /* Gray icons */
.text-gray-300             /* Lighter gray icons */

/* Icon states */
.animate-spin              /* Spinning animation */
.animate-pulse             /* Pulsing animation */
```

---

## Responsive Design

### Breakpoints
```css
.md:                    /* 768px and up */
.lg:                    /* 1024px and up */
.xl:                    /* 1280px and up */

/* Examples */
.lg:col-span-1          /* 1/3 width on large screens */
.lg:col-span-2          /* 2/3 width on large screens */
.md:grid-cols-2         /* 2 columns on medium+ screens */
```

### Display
```css
.flex                   /* Flex layout */
.grid                   /* Grid layout */
.block                  /* Block display */
.hidden                 /* Hidden element */
.overflow-y-auto        /* Vertical scroll */
.max-h-[calc(100vh-300px)]  /* Calculate max height */
.flex-1                 /* Equal flex space */
.shrink-0               /* Don't shrink */
```

---

## Spacing Reference

### Padding Classes
```
.p-2   = 8px    (0.5rem)
.p-3   = 12px   (0.75rem)
.p-4   = 16px   (1rem)
.p-6   = 24px   (1.5rem)
.p-8   = 32px   (2rem)
.p-12  = 48px   (3rem)

.px-2  = 8px horizontal
.px-4  = 16px horizontal
.px-6  = 24px horizontal
.px-10 = 40px horizontal
.px-40 = 160px horizontal

.py-2  = 8px vertical
.py-3  = 12px vertical
.py-5  = 20px vertical
```

### Margin Classes
```
.mb-2  = Margin bottom: 8px
.mb-4  = Margin bottom: 16px
.mb-6  = Margin bottom: 24px
.mb-8  = Margin bottom: 32px
.mt-1  = Margin top: 4px
.mt-2  = Margin top: 8px
```

### Gap Classes
```
.gap-2  = 8px gap
.gap-3  = 12px gap
.gap-4  = 16px gap
.gap-6  = 24px gap
.gap-8  = 32px gap
.gap-9  = 36px gap
.gap-x-2  = Horizontal gap
.gap-x-3  = Horizontal gap
```

---

## Custom Sizes

### Width/Height
```css
.size-4         /* 16px × 16px */
.size-10        /* 40px × 40px */
.w-full         /* 100% */
.w-80           /* 320px */
.h-auto         /* Auto */
.h-full         /* 100% */
.h-12           /* 48px */
.h-14           /* 56px */
.min-w-40       /* Minimum width: 160px */
.max-w-64       /* Maximum width: 256px */
.max-w-[960px]  /* Custom max width */
```

---

## Common Patterns

### Search Input Group
```html
<div class="flex w-full flex-1 items-stretch rounded-lg h-full">
  <!-- Icon -->
  <div class="text-[#496a9c] flex border-none bg-[#e7ecf4] items-center justify-center pl-4 rounded-l-lg border-r-0">
    <svg>...</svg>
  </div>
  <!-- Input -->
  <input class="form-input flex w-full ... bg-[#e7ecf4] rounded-l-none border-l-0 pl-2" />
</div>
```

### Filter Button
```html
<button class="flex h-8 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-[#e7ecf4] pl-4 pr-2">
  <p class="text-[#0d131c] text-sm font-medium leading-normal">Level</p>
  <svg><!-- Caret down --></svg>
</button>
```

### Table Structure
```html
<div class="flex overflow-hidden rounded-lg border border-[#ced9e8] bg-slate-50">
  <table class="flex-1">
    <thead>
      <tr class="bg-slate-50">
        <th class="px-4 py-3 text-left text-[#0d131c] w-[400px] text-sm font-medium">Name</th>
      </tr>
    </thead>
    <tbody>
      <tr class="border-t border-t-[#ced9e8]">
        <td class="h-[72px] px-4 py-2 w-[400px] text-[#0d131c]">Content</td>
      </tr>
    </tbody>
  </table>
</div>
```

---

## States & Interactions

### Hover States
```css
.hover:bg-[#d1dce8]     /* Darker blue on hover */
.hover:bg-gray-50       /* Light gray on hover */
.hover:bg-gray-200      /* Medium gray on hover */
.hover:border-blue-300  /* Blue border on hover */
.hover:shadow            /* Shadow on hover */
.hover:text-[#0d131c]   /* Change text color on hover */
.hover:bg-[#d1dce8]     /* Change background on hover */
```

### Focus States
```css
.focus:outline-0        /* Remove default outline */
.focus:ring-0           /* Remove default ring */
.focus:border-[#ced9e8] /* Custom focus border */
```

### Disabled States
```css
.disabled:opacity-50    /* 50% opacity when disabled */
.disabled:bg-gray-100   /* Gray background */
.disabled:cursor-not-allowed  /* Not-allowed cursor */
```

### Selected States
```css
.border-blue-500        /* Blue border */
.bg-blue-50             /* Light blue background */
.text-blue-600          /* Blue text */
```

---

## Animation & Transitions

```css
.transition-all         /* Smooth all transitions */
.animate-spin           /* Spinning animation */
.animate-pulse          /* Pulsing animation */
.duration-200           /* 200ms duration */
```

---

## Typography Sizes

```
.text-xs   = 12px   (0.75rem)
.text-sm   = 14px   (0.875rem)
.text-base = 16px   (1rem)
.text-lg   = 18px   (1.125rem)
.text-xl   = 20px   (1.25rem)
.text-2xl  = 24px   (1.5rem)
.text-[32px] = Custom 32px
```

---

## Example: Complete Component

```tsx
<div className="flex items-center gap-4 bg-slate-50 px-4 min-h-[72px] py-2 cursor-pointer border-l-4 border-l-transparent hover:bg-gray-100 hover:border-l-blue-600 transition-all">
  <div className="flex flex-col justify-center flex-1">
    <p className="text-[#0d131c] text-base font-medium leading-normal">Name</p>
    <p className="text-[#496a9c] text-sm font-normal leading-normal">Subtitle</p>
  </div>
  <button className="text-red-600 hover:text-red-800 p-2">
    <TrashIcon className="w-4 h-4" />
  </button>
</div>
```

This component demonstrates:
- Flexbox layout with gap
- Light background with hover state
- Left border accent with transition
- Multi-line text with proper colors
- Action button with hover state
- Proper spacing and sizing
