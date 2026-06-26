# Quick Implementation Guide: Add Test Button to Calculation Cards

## Current State

**File:** `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx`
**Line:** ~553 (CardActions section)

### Current Code:
```tsx
<CardActions>
  <Button
    size="small"
    startIcon={<EditIcon />}
    onClick={() => handleOpenEditor(calculation)}
    color="secondary"
  >
    Edit
  </Button>
</CardActions>
```

---

## Implementation Steps

### Step 1: Import Required Icons

Add to imports at top of file (around line 35):
```tsx
import {
  // ... existing imports ...
  PlayArrow as PlayArrowIcon,  // ADD THIS LINE
  // ... rest of imports ...
} from '@mui/icons-material';
```

### Step 2: Add State for Test Dialog

In the component state section (around line 95-110), add:
```tsx
const [testDialogOpen, setTestDialogOpen] = useState(false);
const [selectedForTest, setSelectedForTest] = useState<CalculationOption | null>(null);
const [testResults, setTestResults] = useState<any>(null);
const [testLoading, setTestLoading] = useState(false);
const [testError, setTestError] = useState<string | null>(null);
```

### Step 3: Add Test Handler Function

Add this function after the `handleSaveCalculation` function (around line 300):
```tsx
const handleTestCalculation = (calculation: CalculationOption) => {
  setSelectedForTest(calculation);
  setTestDialogOpen(true);
  setTestResults(null);
  setTestError(null);
};

const handleRunTest = async (sampleData: Record<string, any>) => {
  if (!selectedForTest) return;
  
  setTestLoading(true);
  setTestError(null);
  
  try {
    const response = await fetch(
      import.meta.env.VITE_API_URL || 'http://localhost:8082',
      {
        method: 'POST',
        path: '/api/calc/run',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': localStorage.getItem('selected_tenant') ? 
            JSON.parse(localStorage.getItem('selected_tenant') || '{}').id : 'default',
        },
        body: JSON.stringify({
          financial: {
            type: selectedForTest.financial_calc?.type || 'sql',
            formula: selectedForTest.sql || selectedForTest.financial_calc?.formula,
            arguments: sampleData,
          }
        })
      }
    );

    if (!response.ok) {
      throw new Error(`Test failed: ${response.statusText}`);
    }

    const result = await response.json();
    setTestResults(result);
    setSnackbarMessage('✅ Calculation test successful!');
    setSnackbarOpen(true);
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : 'Unknown error';
    setTestError(errorMsg);
    setSnackbarMessage(`❌ Test failed: ${errorMsg}`);
    setSnackbarOpen(true);
  } finally {
    setTestLoading(false);
  }
};
```

### Step 4: Update CardActions

Replace the CardActions section (around line 553) with:
```tsx
<CardActions sx={{ display: 'flex', gap: 1 }}>
  <Button
    size="small"
    startIcon={<PlayArrowIcon />}
    onClick={() => handleTestCalculation(calculation)}
    color="success"
    variant="outlined"
  >
    Test
  </Button>
  <Button
    size="small"
    startIcon={<EditIcon />}
    onClick={() => handleOpenEditor(calculation)}
    color="secondary"
  >
    Edit
  </Button>
</CardActions>
```

### Step 5: Add Test Dialog Component

Add this new component before the final return statement (around line 590):
```tsx
<Dialog 
  open={testDialogOpen} 
  onClose={() => setTestDialogOpen(false)} 
  maxWidth="sm" 
  fullWidth
>
  <ModalHeader 
    title={`Test: ${selectedForTest?.title}`} 
    onClose={() => setTestDialogOpen(false)} 
  />
  <DialogContent>
    <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Alert severity="info">
        Enter sample data to test this calculation. Results will be returned in real-time.
      </Alert>

      {/* Formula Display */}
      <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50' }}>
        <Typography variant="subtitle2" gutterBottom>
          Formula
        </Typography>
        <Typography 
          variant="body2" 
          sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}
        >
          {selectedForTest?.sql || selectedForTest?.financial_calc?.formula}
        </Typography>
      </Paper>

      {/* Sample Input */}
      <TextField
        label="Sample Data (JSON)"
        multiline
        rows={6}
        defaultValue={JSON.stringify(
          {
            cash_flows: [100, -50, 75, -120],
            dates: ['2023-01-01', '2023-06-01', '2024-01-01', '2024-06-01']
          },
          null,
          2
        )}
        placeholder='{\n  "field1": "value1",\n  "field2": 123\n}'
        fullWidth
        variant="outlined"
        sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}
      />

      {/* Results */}
      {testResults && (
        <Paper variant="outlined" sx={{ p: 2, bgcolor: 'success.50' }}>
          <Typography variant="subtitle2" gutterBottom color="success.dark">
            ✓ Test Results
          </Typography>
          <Typography 
            variant="body2" 
            sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}
          >
            {JSON.stringify(testResults, null, 2)}
          </Typography>
        </Paper>
      )}

      {/* Errors */}
      {testError && (
        <Alert severity="error">
          {testError}
        </Alert>
      )}
    </Box>
  </DialogContent>
  <DialogActions>
    <Button onClick={() => setTestDialogOpen(false)}>Close</Button>
    <Button 
      onClick={() => handleRunTest(JSON.parse(
        (document.querySelector('textarea[placeholder*="field1"]') as HTMLTextAreaElement)?.value || '{}'
      ))}
      variant="contained"
      disabled={testLoading}
    >
      {testLoading ? 'Testing...' : 'Run Test'}
    </Button>
  </DialogActions>
</Dialog>
```

---

## Full File Structure Overview

After implementation, your component structure will be:

```
CalculationsLibraryPage.tsx
├── Imports (add PlayArrowIcon)
├── Types
├── Component State
│   ├── searchTerm, selectedCategories, etc.
│   └── ✨ NEW: testDialogOpen, selectedForTest, testResults, testLoading, testError
├── Handlers
│   ├── handleCategoryToggle()
│   ├── handleOpenEditor()
│   ├── handleSaveCalculation()
│   └── ✨ NEW: handleTestCalculation(), handleRunTest()
├── JSX Return
│   ├── Left Sidebar (Filters)
│   ├── Main Content (Calculation Cards)
│   │   └── CardActions with Edit + Test buttons
│   ├── CalculationEditorModal
│   └── ✨ NEW: Test Results Dialog
└── Export
```

---

## Testing the Implementation

### Test Scenario 1: IRR Calculation

1. Navigate to: `http://localhost:5173/fabric/calculations`
2. Filter: Performance → IRR
3. Click "Test" on "Investment XIRR" card
4. Sample data popup appears
5. Modify sample data (or use defaults)
6. Click "Run Test"
7. See results or error message

**Sample Test Data:**
```json
{
  "cash_flows": [100, -50, 75, -120],
  "dates": ["2023-01-01", "2023-06-01", "2024-01-01", "2024-06-01"]
}
```

**Expected Result:**
```json
{
  "result": {
    "type": "percentage",
    "value": 0.1245,
    "display": "12.45%",
    "metadata": {
      "calculation_type": "xirr",
      "calculation_time_ms": 125
    }
  }
}
```

### Test Scenario 2: Custom Calculation

1. Click "Add Calculation" button
2. Create test calculation:
   - Name: `test_sum`
   - Title: `Sum Test`
   - Formula: `SUM(values)`
   - Category: `Testing`
3. Click "Save" then immediately click "Test"
4. Enter sample data: `{"values": [10, 20, 30]}`
5. Verify result: `60`

---

## File Locations Summary

| Component | File | Lines |
|-----------|------|-------|
| Main Page | `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx` | Full file |
| Imports | Same file | ~35-50 |
| State | Same file | ~95-110 |
| Handlers | Same file | ~280-320 + NEW |
| CardActions | Same file | ~553-563 |
| Test Dialog | Same file | ~590-680 + NEW |
| Modal Header | `frontend/src/components/ModalHeader.tsx` | Import only |

---

## API Endpoint Being Called

```
POST http://localhost:8082/api/calc/run

Required Headers:
- Content-Type: application/json
- X-Tenant-ID: <tenant-id>
- X-Tenant-Datasource-ID: <datasource-id>

Request Body:
{
  "financial": {
    "type": "calculation_type",
    "formula": "calculation_formula",
    "arguments": { "sample": "data" }
  }
}

Response:
{
  "result": {
    "type": "result_type",
    "value": calculated_value,
    "display": "formatted_display",
    "metadata": { "calculation_type": "...", "calculation_time_ms": 123 }
  }
}
```

---

## Next Steps After Implementation

1. ✅ Test basic calculations (IRR, sums, averages)
2. ✅ Add error handling for invalid formulas
3. ✅ Add test data templates for each calculation type
4. ✅ Add result formatting/visualization
5. ✅ Add ability to save test runs as documentation
6. ⏭️ Integrate with external service calls (see CALC_ENGINE_EXTENSIONS_GUIDE.md)

