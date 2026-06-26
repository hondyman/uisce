# BP Builder - Design System & UX Guide

## 🎨 Visual Design System

### Color Palette

**Step Types** (Color-Coded for Quick Identification):
```
Data Entry       → Blue       (#3B82F6)  - Input/Collection
Validation       → Green      (#22C55E)  - Verification/Checks
Approval         → Purple     (#A855F7)  - Authorization/Decision
Notification     → Orange     (#F97316)  - Communication
Integration      → Indigo     (#6366F1)  - External Systems
Conditional      → Yellow     (#EAB308)  - Logic/Branching
```

**UI Elements**:
```
Primary Action   → Blue-600    (#2563EB)  - Save, Publish, Simulate
Success          → Green-500   (#22C55E)  - Completed actions
Error            → Red-500     (#EF4444)  - Failures, alerts
Warning          → Yellow-500  (#EAB308)  - Escalations, notices
Neutral          → Gray-500    (#6B7280)  - Secondary actions
```

---

## 📐 Component Layout

### Master Layout (BP Builder Page)
```
┌────────────────────────────────────────────────────────────┐
│ Header: Gradient (Blue→Indigo)                     [⛶] ◥   │  ← Maximize button
├──────────────────────────┬──────────────────────────────────┤
│                          │                                  │
│  Left Panel              │  Right Panel                     │
│  (380px)                 │  (Flex)                          │
│                          │                                  │
│  • Process Config        │  • View Mode Selector            │
│  • Entity Selection      │  • Canvas/Timeline/JSON          │
│  • Metadata              │  • Visual Workflow               │
│  • Statistics            │  • Step Listing                  │
│  • Action Buttons        │  • Add Step Button               │
│  (S)ave (S)imulate       │                                  │
│  (P)ublish (E)xport      │                                  │
│                          │                                  │
└──────────────────────────┴──────────────────────────────────┘
```

### Step Card (Canvas View)
```
┌─────────────────────────────────────────────────────────────────┐
│  [Icon] Step Name              [Badge: Step N]                  │
│  ─────────────────────────────────────────────────────────────  │
│  Description text appears here with full context info           │
│                                                                  │
│  🕐 1.5h  ⚠️ Escalate: 4h  👤 Manager  ✓ 3 rules  [⚙] [✕]    │
└─────────────────────────────────────────────────────────────────┘
```

### Step Editor Modal
```
┌──────────────────────────────────────────────────────────────────┐
│ [Icon] Edit Step - Data Entry              [×]                   │
│ Collect information from user                                    │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Step Type Selection (6 type buttons)                            │
│  ──────────────────────────────────────────────────────────────  │
│  [📋] [✓] [👤] [📧] [⚙] [🔀]                                    │
│                                                                   │
│  Step Name *                                                     │
│  [_____________________________________]                        │
│                                                                   │
│  Description                                                     │
│  [_____________________________________________]                │
│  [_____________________________________________]                │
│  [_____________________________________________]                │
│                                                                   │
│  Duration (hours) | Escalation Threshold (hours)               │
│  [_____________]  | [_____________]                             │
│                                                                   │
│  [Type-Specific Fields]  ← Changes based on step type           │
│                                                                   │
├──────────────────────────────────────────────────────────────────┤
│                                                    [Cancel] [Save]│
└──────────────────────────────────────────────────────────────────┘
```

---

## 🎭 State Management

### Visual State Transitions

```
Draft Process
    │
    ├─ Edit: Add/Remove/Reorder Steps
    │  (Visual feedback: highlight, drag preview)
    │
    ├─ Save
    │  (Button becomes loading spinner, success toast)
    │
    ↓
Saved Process (Version 1)
    │
    ├─ Publish
    │  (Button disabled until saved)
    │  (Status badge: Draft → Published)
    │
    ↓
Published Process (Active)
    │
    ├─ Simulate
    │  (Shows execution flow, timing, any errors)
    │
    └─ Can be executed by workflows
```

### Loading States

```
Button States:
┌──────────────────────────┐
│ Save Process             │  ← Idle
└──────────────────────────┘
        ↓ Click
┌──────────────────────────┐
│ ⟳ Saving...              │  ← Loading (spinner, disabled)
└──────────────────────────┘
        ↓ 1-2s
┌──────────────────────────┐
│ Save Process             │  ← Success (return to idle)
└──────────────────────────┘
   + Toast: "Saved successfully"
```

### Error States

```
Invalid State:
┌──────────────────────────┐
│ Process name is required │  ← Inline error
│ [_________________]      │
│ ✓ Check ❌ Reset        │
└──────────────────────────┘

API Error:
┌──────────────────────────────────────────┐
│ ✗ Failed to save: Connection timeout     │  ← Toast error
│   [Retry]                                 │
└──────────────────────────────────────────┘
```

---

## ♿ Accessibility Features

### Keyboard Navigation

```
Tab Flow:
Process Name Input
    ↓ Tab
Entity Dropdown
    ↓ Tab
Description Textarea
    ↓ Tab
Add Step Button
    ↓ Tab
Step Cards (Sortable)
    ↓ Arrow Keys to reorder
    ↓ Tab through Edit/Delete buttons
    ↓ Enter to activate
```

### Screen Reader Optimizations

```html
<!-- All buttons have aria-labels -->
<button aria-label="Save process">
  <SaveIcon />
</button>

<!-- Form fields properly labeled -->
<label htmlFor="processName">Process Name *</label>
<input id="processName" type="text" />

<!-- Form sections have headings -->
<h3 aria-level="3">Step Configuration</h3>

<!-- Interactive lists marked -->
<div role="list">
  <div role="listitem" tabIndex={0}>Step 1</div>
</div>
```

### Color Contrast

```
Text on colored backgrounds meets WCAG AAA:
- Text color: #ffffff or #000000
- Minimum contrast ratio: 7:1
- Large text: 4.5:1 acceptable
```

---

## 📱 Responsive Behavior

### Desktop (1920px)
```
3-column layout:
[Left Config (380px)] [Divider] [Canvas + Timeline (flex)]
```

### Laptop (1366px)
```
2-column layout:
[Left Config (320px)] [Divider] [Canvas (flex)]
```

### Tablet (768px)
```
Stacked:
[Config Panel (full width)]
[Canvas Panel (full width, below)]
```

### Mobile (375px)
```
Single column:
[Maximized canvas only]
[Config in drawer/modal]
[Buttons: ↑ Config ↓ Canvas]
```

---

## 🎬 Animation & Interaction

### Micro-Interactions

```css
/* Hover states */
.step-card:hover {
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
  transition: all 0.2s ease;
}

/* Drag preview */
.step-card.dragging {
  opacity: 0.5;
  transform: scale(0.95);
}

/* Toast entrance */
@keyframes slideIn {
  from { 
    transform: translateX(400px);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

/* Loading spinner */
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
```

### Transition Timings

```
User Action → Feedback Time
────────────────────────────
Click button       → 0-100ms   (Visual feedback)
API request        → 100-500ms (Loading state)
Page transition    → 200-300ms (Slide/fade)
Toast dismiss      → 3000ms    (Auto-close)
Hover effect       → 150ms     (Smooth transition)
```

---

## 🎯 Typography Hierarchy

### Font Sizes & Weights

```
Headings:
H1 (Page Title)      24px / Bold / Letter-spacing: 0.5px
H2 (Section)         20px / Semibold
H3 (Subsection)      16px / Semibold
H4 (Label)           14px / Semibold

Body Text:
Paragraph            14px / Regular / Line-height: 1.6
Small                12px / Regular / Color: Gray-600
Code                 12px / Monospace / Font: 'Monaco'
Error Message        13px / Regular / Color: Red-600
```

### Line Heights

```
Headings:             1.2x
Body text:            1.6x
Tight list:           1.4x
Loose list:           1.8x
Code blocks:          1.5x
```

---

## 🎨 Icon Usage

### Icon Set (Lucide React)

```typescript
// Step type icons
FileText    → Data Entry
CheckCircle → Validation
User        → Approval
Send        → Notification
Settings    → Integration
GitBranch   → Conditional

// Action icons
Plus        → Add item
Trash2      → Delete
Save        → Save/Persist
Upload      → Publish
Play        → Simulate
Download    → Export
Clock       → Duration
AlertTriangle → Escalation

// UI icons
ChevronUp   → Collapse
ChevronDown → Expand
Maximize2   → Fullscreen
Minimize2   → Exit fullscreen
X           → Close/Cancel
```

---

## 📊 Statistics Display

### Dashboard Card

```
┌──────────────────────────┐
│ Total Steps:         5   │  ← Font-size: 18px, Bold
│ Total Duration:     12.5h│  ← Font-size: 16px, Semibold
│ Status:     Published ✓  │  ← Badge: Green bg, green text
└──────────────────────────┘
```

### Timeline View

```
0h  │
    ├─ [Step 1] 1h
    │
1h  │
    ├─ [Step 2] 2h
    │
3h  │  ← Cumulative time markers
    ├─ [Step 3] 4h
    │
7h  │
    └─ [Step 4] 1h
    │
8h  │ ← Final completion time
```

---

## 🚨 Error & Success Patterns

### Success Pattern
```
✓ Green background
✓ White text
✓ Check icon
✓ Clear message: "Process saved successfully"
✓ Auto-dismiss after 3s
✓ Optional [Dismiss] button
```

### Error Pattern
```
✗ Red background
✗ White text
✗ Alert icon
✗ Error message: "Failed to save: Network error"
✗ Optional [Retry] or [Details] button
✗ Manual dismiss required
```

### Warning Pattern
```
⚠ Yellow/Orange background
⚠ Dark text
⚠ Warning icon
⚠ Message: "Process has unsaved changes"
⚠ Call-to-action buttons
⚠ Manual dismiss
```

---

## 📐 Spacing & Sizing

### Margins & Padding (px)
```
Page padding:         24px
Section spacing:      16px
Card padding:         16px
Button padding:       10px 16px
Input padding:        8px 12px
Icon spacing:         8px
Component gap:        12px
```

### Border Radius (px)
```
Buttons:              8px
Cards:                8px
Modals:              12px
Badges:               20px (pill-shaped)
Inputs:               6px
```

### Widths
```
Left panel:          380px
Modal max-width:     600px (desktop), 90vw (mobile)
Input min-width:     200px
Button min-width:    120px
```

---

## 🌈 Theme Support (Ready for Light/Dark)

### Light Mode (Current)
```
Background:          #FFFFFF
Text:                #1F2937
Border:              #E5E7EB
Hover overlay:       rgba(0,0,0,0.05)
```

### Dark Mode (Future)
```
Background:          #1F2937
Text:                #F3F4F6
Border:              #374151
Hover overlay:       rgba(255,255,255,0.1)
```

---

## 📸 Screenshot Reference

### Canvas View
- [Visual step cards arranged vertically]
- [Arrows between steps showing flow]
- [Color-coded badges on each step]
- [Edit/Delete buttons on hover]
- [Add Step button at bottom]

### Timeline View
- [Horizontal timeline with time markers]
- [Cumulative duration on left]
- [Step cards positioned along timeline]
- [Connection lines between steps]

### Editor Modal
- [Step type selector at top]
- [Form fields for configuration]
- [Type-specific fields below]
- [Save/Cancel buttons at bottom]

---

## ✅ Design Validation Checklist

- [ ] All buttons have hover states
- [ ] All text meets contrast requirements
- [ ] All inputs have associated labels
- [ ] All icons have meaningful titles
- [ ] Loading states are visible
- [ ] Error messages are clear and actionable
- [ ] Color is not the only differentiator
- [ ] Transitions are smooth but not distracting
- [ ] Mobile layout is tested
- [ ] Keyboard navigation works

---

**Design System Version**: 1.0  
**Last Updated**: October 21, 2025  
**Status**: ✅ Production Ready
