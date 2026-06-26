# Integration Guide - Adding the Wizard to ValidationRulesWithFacets

**Quick Setup:** Copy-paste the code snippets below into your existing component.

---

## 1. Add Import at Top of File

```typescript
import { ValidationRuleCreator } from './ValidationRuleCreator';
```

**Location:** Near other component imports in `ValidationRulesWithFacets.tsx`

---

## 2. Add State for Creator Modal

```typescript
const [creatorOpen, setCreatorOpen] = useState(false);
```

**Location:** Add this line with your other useState declarations, after the existing state variables.

**Example:**
```typescript
const [searchTerm, setSearchTerm] = useState('');
const [selectedFilters, setSelectedFilters] = useState<SelectedFilters>({});
const [creatorOpen, setCreatorOpen] = useState(false);  // ← Add here
const [totalFacetCounts, setTotalFacetCounts] = useState<FacetCounts | null>(null);
```

---

## 3. Add the "+ Add Rule" Button

**Location:** In the search bar area, typically next to the search input

**Code:**
```typescript
<button
  onClick={() => setCreatorOpen(true)}
  className="add-rule-btn"
  title="Create a new validation rule"
>
  + Add Rule
</button>
```

**Example (Full Search Bar):**
```typescript
<div className="search-bar">
  <input
    type="text"
    placeholder="Search validation rules..."
    value={searchTerm}
    onChange={(e) => setSearchTerm(e.target.value)}
    className="search-input"
  />
  <button
    onClick={() => setCreatorOpen(true)}
    className="add-rule-btn"
    title="Create a new validation rule"
  >
    + Add Rule
  </button>
</div>
```

---

## 4. Add CSS for Button (if not already present)

**File:** `ValidationRulesWithFacets.css`

**Add this to your CSS file:**

```css
.add-rule-btn {
  padding: 0.5rem 1rem;
  background-color: #2563eb;
  color: white;
  border: none;
  border-radius: 0.375rem;
  font-weight: 500;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.add-rule-btn:hover {
  background-color: #1d4ed8;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.add-rule-btn:active {
  background-color: #1e40af;
  transform: translateY(1px);
}

.add-rule-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Mobile responsive */
@media (max-width: 640px) {
  .add-rule-btn {
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
  }
}
```

---

## 5. Add the Component Instance

**Location:** At the end of your component's JSX, typically before the closing return statement

**Code:**
```typescript
<ValidationRuleCreator
  isOpen={creatorOpen}
  onClose={() => setCreatorOpen(false)}
  onSave={(newRule) => {
    // Refresh the rules list after successful creation
    fetchRules();
    setCreatorOpen(false);
  }}
  tenantId={tenantId}
  datasourceId={datasourceId}
  availableEntities={Object.keys(totalFacetCounts?.entities || {})}
/>
```

**Example (Full Return JSX):**
```typescript
return (
  <div className="validation-rules-container">
    {/* Existing content */}
    <div className="search-bar">
      <input
        type="text"
        placeholder="Search..."
        value={searchTerm}
        onChange={(e) => setSearchTerm(e.target.value)}
      />
      <button onClick={() => setCreatorOpen(true)} className="add-rule-btn">
        + Add Rule
      </button>
    </div>

    {/* Existing rules list */}
    <div className="rules-list">
      {rules.map(rule => (
        // ... existing rule items
      ))}
    </div>

    {/* Add the creator modal here */}
    <ValidationRuleCreator
      isOpen={creatorOpen}
      onClose={() => setCreatorOpen(false)}
      onSave={(newRule) => {
        fetchRules();
        setCreatorOpen(false);
      }}
      tenantId={tenantId}
      datasourceId={datasourceId}
      availableEntities={Object.keys(totalFacetCounts?.entities || {})}
    />
  </div>
);
```

---

## Complete Integration Example

Here's what your updated component structure should look like:

```typescript
// At top of file
import React, { useState, useEffect } from 'react';
import { ValidationRuleCreator } from './ValidationRuleCreator'; // ← ADD THIS
import './ValidationRulesWithFacets.css';

// Component
export const ValidationRulesWithFacets: React.FC<Props> = ({
  tenantId,
  datasourceId,
}) => {
  // Existing state
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedFilters, setSelectedFilters] = useState<SelectedFilters>({});
  const [totalFacetCounts, setTotalFacetCounts] = useState<FacetCounts | null>(null);
  const [rules, setRules] = useState<ValidationRule[]>([]);

  // ADD NEW STATE FOR CREATOR
  const [creatorOpen, setCreatorOpen] = useState(false);

  // Existing functions
  const fetchRules = async () => {
    // ... existing fetch logic
  };

  // Existing useEffect
  useEffect(() => {
    // ... existing effect logic
  }, [dependencies]);

  return (
    <div className="validation-rules-container">
      {/* Search bar with button */}
      <div className="search-bar">
        <input
          type="text"
          placeholder="Search validation rules..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="search-input"
        />
        {/* ADD THIS BUTTON */}
        <button
          onClick={() => setCreatorOpen(true)}
          className="add-rule-btn"
          title="Create a new validation rule"
        >
          + Add Rule
        </button>
      </div>

      {/* Existing facets and rules list */}
      <div className="facets-section">
        {/* ... existing facets code ... */}
      </div>

      <div className="rules-list">
        {rules.map(rule => (
          // ... existing rule items
        ))}
      </div>

      {/* ADD THIS COMPONENT */}
      <ValidationRuleCreator
        isOpen={creatorOpen}
        onClose={() => setCreatorOpen(false)}
        onSave={(newRule) => {
          // After successful creation, refresh the list
          fetchRules();
          setCreatorOpen(false);
        }}
        tenantId={tenantId}
        datasourceId={datasourceId}
        availableEntities={Object.keys(totalFacetCounts?.entities || {})}
      />
    </div>
  );
};
```

---

## Integration Verification Checklist

After adding the code, verify:

- [ ] **Import Added** - `ValidationRuleCreator` imported at top
- [ ] **State Added** - `creatorOpen` state variable created
- [ ] **Button Added** - "+ Add Rule" button visible in UI
- [ ] **Button Works** - Clicking button opens modal
- [ ] **CSS Added** - Button has proper styling
- [ ] **Component Added** - Modal renders when `isOpen` is true
- [ ] **Props Passed** - All required props provided to component
- [ ] **onSave Handler** - Calls `fetchRules()` to refresh list
- [ ] **onClose Handler** - Sets `creatorOpen` to false
- [ ] **TypeScript** - No compilation errors
- [ ] **Build** - `npm run build` succeeds
- [ ] **Runtime** - Modal opens and closes without errors

---

## Testing the Integration

### Manual Testing Steps

1. **Start Backend**
   ```bash
   cd backend
   PORT=29080 go run ./cmd/server
   ```

2. **Build Frontend**
   ```bash
   cd frontend
   npm run build
   ```

3. **Start Frontend (if using dev server)**
   ```bash
   npm run dev
   ```

4. **Test in Browser**
   - Navigate to validation rules page
   - Verify "+ Add Rule" button is visible
   - Click the button
   - Modal should open
   - Try filling out the form
   - Submit should work (or show validation errors)

5. **Check Console**
   - No TypeScript errors
   - No JavaScript errors
   - API calls visible in Network tab
   - Successful POST should show in Network

---

## Troubleshooting Integration

| Issue | Solution |
|-------|----------|
| Button not visible | Check CSS is imported and className matches |
| Modal won't open | Verify `isOpen` prop binding is correct |
| Modal won't close | Check `onClose` handler is updating state correctly |
| Form won't submit | Check console for validation errors |
| Rules don't appear | Check `fetchRules()` is being called in `onSave` |
| Styling looks wrong | Clear browser cache, rebuild frontend |
| TypeScript errors | Check all props are passed correctly |
| API error | Verify `tenantId` and `datasourceId` are valid UUIDs |

---

## Alternative Integrations

### If using different button placement

```typescript
// In a toolbar
<div className="toolbar">
  <button onClick={() => setCreatorOpen(true)}>
    Create Rule
  </button>
</div>

// In a floating action button
<button 
  className="fab"
  onClick={() => setCreatorOpen(true)}
  title="Create new validation rule"
>
  ➕
</button>

// In a header section
<header>
  <h1>Validation Rules</h1>
  <button onClick={() => setCreatorOpen(true)}>
    + New Rule
  </button>
</header>
```

### If using different styling

```typescript
// Bootstrap style
<button 
  onClick={() => setCreatorOpen(true)}
  className="btn btn-primary"
>
  + Add Rule
</button>

// Material-UI style
<Button
  variant="contained"
  color="primary"
  onClick={() => setCreatorOpen(true)}
>
  + Add Rule
</Button>

// Tailwind style
<button 
  onClick={() => setCreatorOpen(true)}
  className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
>
  + Add Rule
</button>
```

---

## Integration Complete! ✅

Once you've completed all the steps above:
1. Your "+ Add Rule" button will be visible
2. Clicking it opens the beautiful 4-step wizard
3. Users can create validation rules with the guided interface
4. Created rules automatically appear in your list
5. Full form validation and error handling working

Enjoy your new validation rules wizard! 🎉
