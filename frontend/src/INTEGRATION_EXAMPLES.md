# Parameter Builder Integration Examples

This document shows how to use the `ParameterBuilder` component in different contexts across the platform.

---

## 1. Validation Rules Builder (✅ Already Implemented)

**File:** `frontend/src/pages/ValidationRulesBuilderPage.tsx`

```tsx
import ParameterBuilder from '../components/ParameterBuilder';
import { getParameterSchema } from '../lib/parameterSchemas';

export const ValidationRulesBuilderPage: React.FC = () => {
  const [formData, setFormData] = useState({
    ruleType: 'CONCENTRATION',
    parameters: {},
  });

  return (
    <div className="p-6">
      <h2>Create Validation Rule</h2>
      
      {/* ... other form fields ... */}

      {/* Parameter Builder */}
      {getParameterSchema(formData.ruleType) && (
        <ParameterBuilder
          schema={getParameterSchema(formData.ruleType)!}
          parameters={formData.parameters}
          onChange={(params) => setFormData({ ...formData, parameters: params })}
          showValidation={false}
        />
      )}

      {/* ... save button ... */}
    </div>
  );
};
```

---

## 2. Report Builder UI (Future Integration)

**File:** `frontend/src/components/ReportBuilderUI.tsx` (to be created)

```tsx
import ParameterBuilder from './ParameterBuilder';
import { getParameterSchema } from '../lib/parameterSchemas';

interface ReportConfig {
  type: 'SUMMARY' | 'DETAILED' | 'CUSTOM';
  parameters: Record<string, any>;
}

export const ReportBuilderUI: React.FC = () => {
  const [reportConfig, setReportConfig] = useState<ReportConfig>({
    type: 'SUMMARY',
    parameters: {},
  });

  const handleConfigChange = (params: Record<string, any>) => {
    setReportConfig({
      ...reportConfig,
      parameters: params,
    });
  };

  return (
    <div className="space-y-6">
      <h2>Build Custom Report</h2>

      {/* Report Type Selection */}
      <div>
        <label>Report Type</label>
        <select
          value={reportConfig.type}
          onChange={(e) => setReportConfig({ 
            ...reportConfig, 
            type: e.target.value as any,
            parameters: {} // Reset parameters when type changes
          })}
        >
          <option value="SUMMARY">Summary Report</option>
          <option value="DETAILED">Detailed Report</option>
          <option value="CUSTOM">Custom Report</option>
        </select>
      </div>

      {/* Dynamic Parameter Configuration */}
      {getParameterSchema(reportConfig.type) && (
        <ParameterBuilder
          schema={getParameterSchema(reportConfig.type)!}
          parameters={reportConfig.parameters}
          onChange={handleConfigChange}
          showValidation={false}
        />
      )}

      {/* Report Preview */}
      <ReportPreview config={reportConfig} />

      {/* Save Button */}
      <button onClick={() => saveReport(reportConfig)}>
        Save Report
      </button>
    </div>
  );
};
```

---

## 3. Rule Builder Component (Future Integration)

**File:** `frontend/src/components/RuleBuilder.tsx` (to be created)

```tsx
import ParameterBuilder from './ParameterBuilder';
import { getParameterSchema, validateParameters } from '../lib/parameterSchemas';

interface RuleConfig {
  id: string;
  name: string;
  type: string;
  parameters: Record<string, any>;
  enabled: boolean;
}

export const RuleBuilder: React.FC<{
  rule: RuleConfig;
  onSave: (rule: RuleConfig) => void;
}> = ({ rule, onSave }) => {
  const [editingRule, setEditingRule] = useState<RuleConfig>(rule);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);

  const handleSave = () => {
    // Validate using schema
    const paramErrors = validateParameters(
      editingRule.type,
      editingRule.parameters
    );

    if (Object.keys(paramErrors).length > 0) {
      setErrors(paramErrors);
      return;
    }

    setSaving(true);
    onSave(editingRule);
    setSaving(false);
  };

  return (
    <div className="space-y-6">
      <h2>Edit Rule: {editingRule.name}</h2>

      {/* Rule Name */}
      <div>
        <label>Rule Name</label>
        <input
          value={editingRule.name}
          onChange={(e) => setEditingRule({ 
            ...editingRule, 
            name: e.target.value 
          })}
        />
      </div>

      {/* Rule Type */}
      <div>
        <label>Rule Type</label>
        <select
          value={editingRule.type}
          onChange={(e) => setEditingRule({ 
            ...editingRule, 
            type: e.target.value,
            parameters: {} // Reset on type change
          })}
        >
          <option value="">Select Type</option>
          {/* Populate from getAvailableRuleTypes() */}
        </select>
      </div>

      {/* Parameter Configuration with Validation */}
      {getParameterSchema(editingRule.type) && (
        <ParameterBuilder
          schema={getParameterSchema(editingRule.type)!}
          parameters={editingRule.parameters}
          onChange={(params) => {
            setEditingRule({ ...editingRule, parameters: params });
            // Clear errors for this field when user changes it
            setErrors({});
          }}
          errors={errors}
          showValidation={saving}
        />
      )}

      {/* Save */}
      <button 
        onClick={handleSave}
        disabled={saving}
      >
        {saving ? 'Saving...' : 'Save Rule'}
      </button>
    </div>
  );
};
```

---

## 4. Dynamic Form Generator (Advanced)

**File:** `frontend/src/components/DynamicFormBuilder.tsx`

For scenarios where you need to generate forms from rule type definitions:

```tsx
import ParameterBuilder from './ParameterBuilder';
import { getAvailableRuleTypes, getParameterSchema } from '../lib/parameterSchemas';

interface DynamicFormBuilderProps {
  selectedRuleTypes: string[];
  onParametersChange: (ruleType: string, params: Record<string, any>) => void;
}

export const DynamicFormBuilder: React.FC<DynamicFormBuilderProps> = ({
  selectedRuleTypes,
  onParametersChange,
}) => {
  const [allParameters, setAllParameters] = useState<
    Record<string, Record<string, any>>
  >({});

  const handleParameterChange = (ruleType: string, params: Record<string, any>) => {
    setAllParameters({
      ...allParameters,
      [ruleType]: params,
    });
    onParametersChange(ruleType, params);
  };

  return (
    <div className="space-y-8">
      <h2>Configure Rules</h2>

      {selectedRuleTypes.map((ruleType) => {
        const schema = getParameterSchema(ruleType);
        if (!schema) return null;

        return (
          <div key={ruleType} className="border rounded-lg p-4">
            <h3>{schema.name}</h3>
            <ParameterBuilder
              schema={schema}
              parameters={allParameters[ruleType] || {}}
              onChange={(params) => handleParameterChange(ruleType, params)}
            />
          </div>
        );
      })}
    </div>
  );
};
```

---

## 5. Extending with Custom Schemas

For platform-specific use cases, you can extend the base schemas:

```tsx
// frontend/src/lib/customSchemas.ts
import { PARAMETER_SCHEMAS, ParameterSchema } from './parameterSchemas';

// Extend with custom rule types
export const EXTENDED_SCHEMAS: Record<string, ParameterSchema> = {
  ...PARAMETER_SCHEMAS,

  // Custom rule type specific to your domain
  CUSTOM_BUSINESS_RULE: {
    ruleType: 'CUSTOM_BUSINESS_RULE',
    name: 'Custom Business Rule',
    description: 'Your custom business logic',
    fields: [
      {
        name: 'businessLogic',
        label: 'Business Logic',
        type: 'textarea',
        placeholder: 'Enter your custom business logic...',
        rows: 10,
      },
      {
        name: 'executionFrequency',
        label: 'Execution Frequency',
        type: 'select',
        options: [
          { value: 'DAILY', label: 'Daily' },
          { value: 'WEEKLY', label: 'Weekly' },
          { value: 'MONTHLY', label: 'Monthly' },
        ],
      },
    ],
  },
};

// Use with ParameterBuilder
export function getExtendedSchema(ruleType: string): ParameterSchema | null {
  return EXTENDED_SCHEMAS[ruleType] || null;
}
```

---

## 6. With Error Handling and Validation

```tsx
import ParameterBuilder from './ParameterBuilder';
import { 
  getParameterSchema, 
  validateParameters 
} from '../lib/parameterSchemas';

export const ValidatedRuleForm: React.FC = () => {
  const [ruleType, setRuleType] = useState('CONCENTRATION');
  const [parameters, setParameters] = useState({});
  const [serverErrors, setServerErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    setSubmitting(true);

    try {
      // Client-side validation using schema
      const clientErrors = validateParameters(ruleType, parameters);
      if (Object.keys(clientErrors).length > 0) {
        setServerErrors(clientErrors);
        setSubmitting(false);
        return;
      }

      // Server-side validation
      const response = await fetch('/api/rules/validate', {
        method: 'POST',
        body: JSON.stringify({ ruleType, parameters }),
      });

      if (!response.ok) {
        const { errors } = await response.json();
        setServerErrors(errors);
        return;
      }

      // Success!
      console.log('Rule validated successfully');
      setServerErrors({});

    } catch (error) {
      setServerErrors({ 
        _general: 'Failed to validate rule' 
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div>
      {serverErrors._general && (
        <div className="alert alert-error">
          {serverErrors._general}
        </div>
      )}

      <ParameterBuilder
        schema={getParameterSchema(ruleType)!}
        parameters={parameters}
        onChange={setParameters}
        errors={serverErrors}
        showValidation={submitting}
      />

      <button 
        onClick={handleSubmit}
        disabled={submitting}
      >
        Validate & Save
      </button>
    </div>
  );
};
```

---

## 7. Multi-Step Wizard

```tsx
import ParameterBuilder from './ParameterBuilder';
import { getParameterSchema } from '../lib/parameterSchemas';

type Step = 'select-type' | 'configure-parameters' | 'review' | 'confirm';

export const RuleWizard: React.FC = () => {
  const [step, setStep] = useState<Step>('select-type');
  const [ruleType, setRuleType] = useState('');
  const [parameters, setParameters] = useState({});

  return (
    <div>
      {step === 'select-type' && (
        <div>
          <h2>Step 1: Select Rule Type</h2>
          {/* Rule type selection */}
          <button onClick={() => setStep('configure-parameters')}>
            Next
          </button>
        </div>
      )}

      {step === 'configure-parameters' && (
        <div>
          <h2>Step 2: Configure Parameters</h2>
          {getParameterSchema(ruleType) && (
            <ParameterBuilder
              schema={getParameterSchema(ruleType)!}
              parameters={parameters}
              onChange={setParameters}
            />
          )}
          <button onClick={() => setStep('review')}>Next</button>
        </div>
      )}

      {step === 'review' && (
        <div>
          <h2>Step 3: Review</h2>
          <RuleReview ruleType={ruleType} parameters={parameters} />
          <button onClick={() => setStep('confirm')}>Confirm</button>
        </div>
      )}

      {step === 'confirm' && (
        <div>
          <h2>Step 4: Complete</h2>
          <p>Rule created successfully!</p>
        </div>
      )}
    </div>
  );
};
```

---

## 8. Testing Integration

```tsx
// frontend/src/__tests__/ParameterBuilder.integration.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import ParameterBuilder from '../components/ParameterBuilder';
import { getParameterSchema } from '../lib/parameterSchemas';

describe('ParameterBuilder Integration Tests', () => {
  it('renders CONCENTRATION schema correctly', () => {
    const schema = getParameterSchema('CONCENTRATION')!;
    const { rerender } = render(
      <ParameterBuilder
        schema={schema}
        parameters={{}}
        onChange={jest.fn()}
      />
    );

    expect(screen.getByText('Max Position Percentage')).toBeInTheDocument();
    expect(screen.getByText('Warning Threshold')).toBeInTheDocument();
  });

  it('handles parameter changes', () => {
    const onChange = jest.fn();
    const schema = getParameterSchema('CONCENTRATION')!;

    render(
      <ParameterBuilder
        schema={schema}
        parameters={{ maxPositionPercentage: 10 }}
        onChange={onChange}
      />
    );

    const input = screen.getByDisplayValue('10');
    fireEvent.change(input, { target: { value: '20' } });

    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        maxPositionPercentage: 20,
      })
    );
  });

  it('validates parameters on submit', () => {
    const schema = getParameterSchema('CONCENTRATION')!;

    render(
      <ParameterBuilder
        schema={schema}
        parameters={{ maxPositionPercentage: 150 }} // Invalid
        onChange={jest.fn()}
        showValidation={true}
      />
    );

    expect(screen.getByText(/must be between 0 and 100/i)).toBeInTheDocument();
  });
});
```

---

## Usage Summary

| Context | Implementation | Complexity |
|---------|----------------|------------|
| **Validation Rules** | ✅ Done | Low |
| **Report Builder** | Ready to implement | Low |
| **Rule Builder** | Ready to implement | Low |
| **Dynamic Forms** | Template provided | Medium |
| **Custom Schemas** | Template provided | Medium |
| **Multi-step Wizard** | Template provided | High |
| **Advanced Validation** | Template provided | Medium |

---

## Common Patterns

### **Pattern 1: Type-driven Parameter Updates**
```tsx
onChange={(e) => setRuleType(e.target.value)}
// Reset parameters when type changes
setParameters({})
```

### **Pattern 2: Form Submission with Validation**
```tsx
const errors = validateParameters(ruleType, parameters);
if (Object.keys(errors).length > 0) return; // Has errors
submitForm(); // Proceed
```

### **Pattern 3: Conditional Parameter Display**
```tsx
{ruleType === 'CONCENTRATION' && (
  <ParameterBuilder ... />
)}
```

### **Pattern 4: Multi-rule Configuration**
```tsx
ruleTypes.map(type => (
  <ParameterBuilder key={type} schema={getParameterSchema(type)!} />
))
```

---

## Next Steps

1. ✅ **Validation Rules** - Implemented
2. ⏳ **Report Builder** - Use examples above to integrate
3. ⏳ **Rule Builder** - Use examples above to integrate
4. ⏳ **Tests** - Add integration tests using examples above
5. ⏳ **Documentation** - Update team wiki with this guide

---

**Last Updated:** October 30, 2025
**Status:** Ready for Integration
