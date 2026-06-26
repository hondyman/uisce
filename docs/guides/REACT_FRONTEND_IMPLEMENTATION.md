# 🎨 React Frontend Implementation - Workday-Style Dynamic Forms

## Overview

Build a complete React frontend that:
- Loads form definitions from backend
- Renders dynamic sections, fields, and actions
- Validates in real-time (client-side)
- Submits with server-side validation
- Triggers business processes

---

## 🗂️ Project Structure

```
frontend/src/
├── hooks/
│   └── useFormDefinition.ts          # React hooks for form operations
├── components/
│   ├── DynamicForm.tsx               # Main form wrapper
│   ├── DynamicFormGenerator.tsx       # Form rendering engine
│   ├── FormField.tsx                 # Single field component
│   ├── FormSection.tsx               # Section container
│   ├── FormActions.tsx               # Action buttons
│   └── ValidationMessages.tsx        # Error/warning display
├── types/
│   └── form.ts                       # TypeScript interfaces
├── utils/
│   └── validation.ts                 # Client-side validation
└── pages/
    └── FormPage.tsx                  # Page component example
```

---

## 📦 Installation

```bash
cd frontend

# Install dependencies
npm install @tanstack/react-query axios react-hook-form zod
npm install -D @types/react @types/react-dom
```

---

## 🏗️ TypeScript Types

### `frontend/src/types/form.ts`

```typescript
// ============================================================================
// FORM DEFINITION TYPES (from backend)
// ============================================================================

export interface FormDefinition {
  id: string;
  business_object: BusinessObject;
  sections: FormSection[];
  actions: FormAction[];
  validations?: Map<string, ValidationRule[]>;
}

export interface BusinessObject {
  id: string;
  tenant_id: string;
  bo_name: string;
  bo_description?: string;
  entity_type: string;
  allow_custom_fields: boolean;
  allow_field_deletion: boolean;
  is_system_bo: boolean;
  is_active: boolean;
  fields: BOField[];
}

export interface BOField {
  id: string;
  bo_id: string;
  field_name: string;
  field_type: 'string' | 'number' | 'date' | 'boolean' | 'reference' | 'picklist' | 'decimal';
  display_label: string;
  is_required: boolean;
  is_readonly: boolean;
  help_text?: string;
  display_order: number;
  section_name?: string;
  validation_rule_ids: string[];
  picklist_values?: string[];
  target_bo_id?: string;
}

export interface FormSection {
  id: string;
  section_title: string;
  columns: number;
  is_collapsible?: boolean;
  field_ids: string[];
  is_collapsed?: boolean;
}

export interface FormAction {
  id: string;
  action_label: string;
  action_type: 'save' | 'submit' | 'cancel' | 'custom';
  requires_validation: boolean;
  triggers_bp_id?: string;
  requires_confirmation?: boolean;
  action_order: number;
}

// ============================================================================
// VALIDATION TYPES
// ============================================================================

export interface ValidationRule {
  id: string;
  rule_name: string;
  condition_type: 'regex' | 'compare' | 'unique_check' | 'range' | 'cross_field' | 'field_type';
  condition_json: Record<string, any>;
  severity: 'error' | 'warning';
  message: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: FieldError[];
  warnings: FieldError[];
}

export interface FieldError {
  field_id: string;
  field_name: string;
  severity: 'error' | 'warning';
  message: string;
}

// ============================================================================
// FORM STATE TYPES
// ============================================================================

export interface FormSubmission {
  record_id: string;
  workflow_id?: string;
  status: 'saved' | 'submitted' | 'error';
  message: string;
}

export interface FieldValidationState {
  [fieldName: string]: {
    errors: FieldError[];
    warnings: FieldError[];
    validating: boolean;
    touched: boolean;
  };
}
```

---

## 🪝 React Hooks

### `frontend/src/hooks/useFormDefinition.ts`

```typescript
import { useQuery, useMutation } from '@tanstack/react-query';
import { FormDefinition, ValidationResult, FormSubmission } from '../types/form';

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

function getTenantContext() {
  const tenant = localStorage.getItem('selected_tenant');
  const product = localStorage.getItem('selected_product');
  const datasource = localStorage.getItem('selected_datasource');

  if (!tenant || !product || !datasource) {
    throw new Error('Tenant context not set. Please select a tenant first.');
  }

  return {
    tenant_id: JSON.parse(tenant).id,
    datasource_id: JSON.parse(datasource).id,
  };
}

function getHeaders() {
  const context = getTenantContext();
  return {
    'Content-Type': 'application/json',
    'X-Tenant-ID': context.tenant_id,
    'X-Tenant-Datasource-ID': context.datasource_id,
  };
}

// ============================================================================
// HOOKS
// ============================================================================

/**
 * Hook to load form definition from backend
 * @param layoutId - Page layout ID
 * @returns Query object with form definition
 */
export function useFormDefinition(layoutId: string) {
  return useQuery({
    queryKey: ['form-definition', layoutId],
    queryFn: async (): Promise<FormDefinition> => {
      const { tenant_id, datasource_id } = getTenantContext();

      const url = new URL(`${import.meta.env.VITE_API_BASE_URL}/api/ui/forms/${layoutId}`);
      url.searchParams.append('tenant_id', tenant_id);
      url.searchParams.append('datasource_id', datasource_id);

      const response = await fetch(url.toString(), {
        method: 'GET',
        headers: getHeaders(),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to load form definition');
      }

      return response.json();
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
    retry: 3,
  });
}

/**
 * Hook to validate form data against all validation rules
 * @param boId - Business Object ID
 * @returns Mutation object for form validation
 */
export function useFormValidation(boId: string) {
  return useMutation({
    mutationFn: async (data: Record<string, any>): Promise<ValidationResult> => {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/ui/validate`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          bo_id: boId,
          data,
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Validation failed');
      }

      return response.json();
    },
  });
}

/**
 * Hook to save form data (without triggering BP)
 * @returns Mutation object for form save
 */
export function useFormSave() {
  return useMutation({
    mutationFn: async (payload: {
      bo_id: string;
      data: Record<string, any>;
    }): Promise<FormSubmission> => {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/ui/save`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Save failed');
      }

      return response.json();
    },
  });
}

/**
 * Hook to submit form data and trigger business process
 * @returns Mutation object for form submission
 */
export function useFormSubmit() {
  return useMutation({
    mutationFn: async (payload: {
      bo_id: string;
      bp_id: string;
      data: Record<string, any>;
    }): Promise<FormSubmission> => {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/ui/submit`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Submission failed');
      }

      return response.json();
    },
  });
}
```

---

## 🎨 Component: ValidationMessages

### `frontend/src/components/ValidationMessages.tsx`

```typescript
import React from 'react';
import { FieldError } from '../types/form';

interface ValidationMessagesProps {
  errors: FieldError[];
  warnings: FieldError[];
}

export const ValidationMessages: React.FC<ValidationMessagesProps> = ({
  errors,
  warnings,
}) => {
  if (errors.length === 0 && warnings.length === 0) {
    return null;
  }

  return (
    <div className="validation-messages">
      {/* Error Messages */}
      {errors.map((error, idx) => (
        <div key={`error-${idx}`} className="validation-message error">
          <span className="icon">❌</span>
          <span className="message">{error.message}</span>
        </div>
      ))}

      {/* Warning Messages */}
      {warnings.map((warning, idx) => (
        <div key={`warning-${idx}`} className="validation-message warning">
          <span className="icon">⚠️</span>
          <span className="message">{warning.message}</span>
        </div>
      ))}

      <style jsx>{`
        .validation-messages {
          display: flex;
          flex-direction: column;
          gap: 8px;
          margin-top: 6px;
        }

        .validation-message {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 13px;
          padding: 6px 8px;
          border-radius: 4px;
        }

        .validation-message.error {
          background-color: #fee;
          color: #c00;
          border-left: 3px solid #c00;
        }

        .validation-message.warning {
          background-color: #ffe;
          color: #990;
          border-left: 3px solid #990;
        }

        .icon {
          font-size: 14px;
          flex-shrink: 0;
        }

        .message {
          flex: 1;
        }
      `}</style>
    </div>
  );
};
```

---

## 🎨 Component: FormField

### `frontend/src/components/FormField.tsx`

```typescript
import React, { useState, useCallback } from 'react';
import { BOField, ValidationRule, FieldError } from '../types/form';
import { ValidationMessages } from './ValidationMessages';
import { validateField } from '../utils/validation';

interface FormFieldProps {
  field: BOField;
  value: any;
  onChange: (fieldName: string, value: any) => void;
  onValidate: (fieldName: string, value: any) => void;
  validationRules: ValidationRule[];
  errors: FieldError[];
  warnings: FieldError[];
  picklistOptions?: Record<string, string[]>;
  lookupOptions?: Record<string, any[]>;
}

export const FormField: React.FC<FormFieldProps> = ({
  field,
  value,
  onChange,
  onValidate,
  validationRules,
  errors,
  warnings,
  picklistOptions,
  lookupOptions,
}) => {
  const [focused, setFocused] = useState(false);

  const handleChange = useCallback(
    (e: React.ChangeEvent<any>) => {
      const newValue =
        field.field_type === 'number' || field.field_type === 'decimal'
          ? parseFloat(e.target.value) || null
          : e.target.value;

      onChange(field.field_name, newValue);
    },
    [field, onChange]
  );

  const handleBlur = useCallback(() => {
    setFocused(false);
    onValidate(field.field_name, value);
  }, [field, value, onValidate]);

  const hasError = errors.length > 0;
  const hasWarning = warnings.length > 0;

  return (
    <div className="form-field">
      <label htmlFor={field.id}>
        {field.display_label}
        {field.is_required && <span className="required">*</span>}
      </label>

      {field.help_text && <div className="help-text">{field.help_text}</div>}

      {/* TEXT INPUT */}
      {field.field_type === 'string' && (
        <input
          id={field.id}
          type="text"
          value={value || ''}
          onChange={handleChange}
          onBlur={handleBlur}
          onFocus={() => setFocused(true)}
          disabled={field.is_readonly}
          required={field.is_required}
          className={`input ${hasError ? 'error' : ''} ${hasWarning ? 'warning' : ''}`}
          placeholder={field.display_label}
        />
      )}

      {/* NUMBER INPUT */}
      {field.field_type === 'number' && (
        <input
          id={field.id}
          type="number"
          value={value || ''}
          onChange={handleChange}
          onBlur={handleBlur}
          onFocus={() => setFocused(true)}
          disabled={field.is_readonly}
          required={field.is_required}
          className={`input ${hasError ? 'error' : ''} ${hasWarning ? 'warning' : ''}`}
          placeholder={field.display_label}
        />
      )}

      {/* DECIMAL INPUT */}
      {field.field_type === 'decimal' && (
        <input
          id={field.id}
          type="number"
          step="0.01"
          value={value || ''}
          onChange={handleChange}
          onBlur={handleBlur}
          onFocus={() => setFocused(true)}
          disabled={field.is_readonly}
          required={field.is_required}
          className={`input ${hasError ? 'error' : ''} ${hasWarning ? 'warning' : ''}`}
          placeholder={field.display_label}
        />
      )}

      {/* DATE INPUT */}
      {field.field_type === 'date' && (
        <input
          id={field.id}
          type="date"
          value={value || ''}
          onChange={handleChange}
          onBlur={handleBlur}
          onFocus={() => setFocused(true)}
          disabled={field.is_readonly}
          required={field.is_required}
          className={`input ${hasError ? 'error' : ''} ${hasWarning ? 'warning' : ''}`}
        />
      )}

      {/* PICKLIST SELECT */}
      {field.field_type === 'picklist' && (
        <select
          id={field.id}
          value={value || ''}
          onChange={handleChange}
          onBlur={handleBlur}
          onFocus={() => setFocused(true)}
          disabled={field.is_readonly}
          required={field.is_required}
          className={`select ${hasError ? 'error' : ''} ${hasWarning ? 'warning' : ''}`}
        >
          <option value="">Select {field.display_label}</option>
          {field.picklist_values?.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      )}

      {/* REFERENCE (LOOKUP) */}
      {field.field_type === 'reference' && (
        <select
          id={field.id}
          value={value?.id || ''}
          onChange={(e) => {
            const selectedId = e.target.value;
            const selectedOption = lookupOptions?.[field.target_bo_id || '']?.find(
              (opt: any) => opt.id === selectedId
            );
            onChange(field.field_name, selectedOption);
          }}
          onBlur={handleBlur}
          onFocus={() => setFocused(true)}
          disabled={field.is_readonly}
          required={field.is_required}
          className={`select ${hasError ? 'error' : ''} ${hasWarning ? 'warning' : ''}`}
        >
          <option value="">Select {field.display_label}</option>
          {lookupOptions?.[field.target_bo_id || '']?.map((option: any) => (
            <option key={option.id} value={option.id}>
              {option.display_text}
            </option>
          ))}
        </select>
      )}

      {/* BOOLEAN CHECKBOX */}
      {field.field_type === 'boolean' && (
        <input
          id={field.id}
          type="checkbox"
          checked={value || false}
          onChange={(e) => onChange(field.field_name, e.target.checked)}
          onBlur={handleBlur}
          onFocus={() => setFocused(true)}
          disabled={field.is_readonly}
          className={`checkbox ${hasError ? 'error' : ''} ${hasWarning ? 'warning' : ''}`}
        />
      )}

      {/* VALIDATION MESSAGES */}
      <ValidationMessages errors={errors} warnings={warnings} />

      <style jsx>{`
        .form-field {
          display: flex;
          flex-direction: column;
          gap: 4px;
          margin-bottom: 16px;
        }

        label {
          font-weight: 500;
          font-size: 14px;
          color: #333;
        }

        .required {
          color: #c00;
          margin-left: 4px;
        }

        .help-text {
          font-size: 12px;
          color: #666;
          font-style: italic;
        }

        input.input,
        select.select {
          padding: 8px 12px;
          border: 1px solid #ccc;
          border-radius: 4px;
          font-size: 14px;
          font-family: inherit;
          transition: border-color 0.2s;
        }

        input.input:focus,
        select.select:focus {
          outline: none;
          border-color: #0066cc;
          box-shadow: 0 0 0 3px rgba(0, 102, 204, 0.1);
        }

        input.input.error,
        select.select.error {
          border-color: #c00;
          background-color: #fee;
        }

        input.input.warning,
        select.select.warning {
          border-color: #990;
          background-color: #ffe;
        }

        input.input:disabled,
        select.select:disabled {
          background-color: #f0f0f0;
          color: #999;
          cursor: not-allowed;
        }

        checkbox {
          width: 18px;
          height: 18px;
          cursor: pointer;
        }

        checkbox:disabled {
          cursor: not-allowed;
          opacity: 0.5;
        }
      `}</style>
    </div>
  );
};
```

---

## 🎨 Component: FormSection

### `frontend/src/components/FormSection.tsx`

```typescript
import React, { useState } from 'react';
import { FormSection as IFormSection, BOField, ValidationRule } from '../types/form';
import { FormField } from './FormField';

interface FormSectionProps {
  section: IFormSection;
  fields: BOField[];
  formData: Record<string, any>;
  onChange: (fieldName: string, value: any) => void;
  onValidate: (fieldName: string, value: any) => void;
  validationMap: Record<string, any>;
  validationRulesMap: Record<string, ValidationRule[]>;
  picklistOptions?: Record<string, string[]>;
  lookupOptions?: Record<string, any[]>;
}

export const FormSection: React.FC<FormSectionProps> = ({
  section,
  fields,
  formData,
  onChange,
  onValidate,
  validationMap,
  validationRulesMap,
  picklistOptions,
  lookupOptions,
}) => {
  const [isCollapsed, setIsCollapsed] = useState(section.is_collapsible ? true : false);

  // Get fields for this section, ordered by display_order
  const sectionFields = section.field_ids
    .map((fieldId) => fields.find((f) => f.id === fieldId))
    .filter((f): f is BOField => Boolean(f))
    .sort((a, b) => a.display_order - b.display_order);

  const gridColsClass = `grid-cols-${section.columns}`;

  return (
    <div className="form-section">
      {section.is_collapsible && (
        <button
          type="button"
          onClick={() => setIsCollapsed(!isCollapsed)}
          className="section-header"
        >
          <span className="toggle-icon">{isCollapsed ? '▶' : '▼'}</span>
          <h3>{section.section_title}</h3>
        </button>
      )}

      {!section.is_collapsible && <h3 className="section-title">{section.section_title}</h3>}

      {!isCollapsed && (
        <div className={`fields-grid ${gridColsClass}`}>
          {sectionFields.map((field) => (
            <FormField
              key={field.id}
              field={field}
              value={formData[field.field_name]}
              onChange={onChange}
              onValidate={onValidate}
              validationRules={validationRulesMap[field.id] || []}
              errors={validationMap[field.field_name]?.errors || []}
              warnings={validationMap[field.field_name]?.warnings || []}
              picklistOptions={picklistOptions}
              lookupOptions={lookupOptions}
            />
          ))}
        </div>
      )}

      <style jsx>{`
        .form-section {
          margin-bottom: 24px;
          padding: 16px;
          background-color: #f9f9f9;
          border-radius: 6px;
          border: 1px solid #e0e0e0;
        }

        .section-header {
          display: flex;
          align-items: center;
          gap: 8px;
          width: 100%;
          padding: 0;
          margin: 0;
          background: none;
          border: none;
          cursor: pointer;
          font-size: 16px;
          font-weight: 600;
          text-align: left;
          color: #333;
          transition: color 0.2s;
        }

        .section-header:hover {
          color: #0066cc;
        }

        .toggle-icon {
          display: inline-block;
          width: 16px;
          text-align: center;
        }

        .section-title {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
          color: #333;
        }

        .fields-grid {
          display: grid;
          gap: 16px;
        }

        .grid-cols-1 {
          grid-template-columns: 1fr;
        }

        .grid-cols-2 {
          grid-template-columns: 1fr 1fr;
        }

        .grid-cols-3 {
          grid-template-columns: 1fr 1fr 1fr;
        }

        @media (max-width: 768px) {
          .grid-cols-2,
          .grid-cols-3 {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  );
};
```

---

## 🎨 Component: FormActions

### `frontend/src/components/FormActions.tsx`

```typescript
import React from 'react';
import { FormAction } from '../types/form';

interface FormActionsProps {
  actions: FormAction[];
  onAction: (actionType: string, bpId?: string) => Promise<void>;
  isLoading: boolean;
  isDisabled: boolean;
}

export const FormActions: React.FC<FormActionsProps> = ({
  actions,
  onAction,
  isLoading,
  isDisabled,
}) => {
  // Sort by action_order
  const sortedActions = [...actions].sort((a, b) => a.action_order - b.action_order);

  return (
    <div className="form-actions">
      {sortedActions.map((action) => (
        <button
          key={action.id}
          type="button"
          onClick={() => onAction(action.action_type, action.triggers_bp_id)}
          disabled={isLoading || isDisabled}
          className={`action-button ${action.action_type}`}
          title={action.action_label}
        >
          {isLoading ? '⏳ Loading...' : action.action_label}
        </button>
      ))}

      <style jsx>{`
        .form-actions {
          display: flex;
          gap: 12px;
          justify-content: flex-start;
          margin-top: 24px;
          padding-top: 16px;
          border-top: 1px solid #e0e0e0;
        }

        .action-button {
          padding: 10px 20px;
          font-size: 14px;
          font-weight: 500;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          transition: all 0.2s;
          font-family: inherit;
        }

        .action-button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .action-button.save {
          background-color: #f0f0f0;
          color: #333;
          border: 1px solid #ccc;
        }

        .action-button.save:hover:not(:disabled) {
          background-color: #e0e0e0;
        }

        .action-button.submit {
          background-color: #0066cc;
          color: white;
        }

        .action-button.submit:hover:not(:disabled) {
          background-color: #0052a3;
        }

        .action-button.cancel {
          background-color: #f5f5f5;
          color: #666;
          border: 1px solid #ddd;
        }

        .action-button.cancel:hover:not(:disabled) {
          background-color: #e0e0e0;
        }

        .action-button.custom {
          background-color: #6c757d;
          color: white;
        }

        .action-button.custom:hover:not(:disabled) {
          background-color: #5a6268;
        }
      `}</style>
    </div>
  );
};
```

---

## 🎨 Component: DynamicFormGenerator

### `frontend/src/components/DynamicFormGenerator.tsx`

```typescript
import React, { useState, useCallback } from 'react';
import { FormDefinition, FieldValidationState, FormAction } from '../types/form';
import { useFormValidation, useFormSave, useFormSubmit } from '../hooks/useFormDefinition';
import { FormSection } from './FormSection';
import { FormActions } from './FormActions';

interface DynamicFormGeneratorProps {
  formDefinition: FormDefinition;
}

export const DynamicFormGenerator: React.FC<DynamicFormGeneratorProps> = ({
  formDefinition,
}) => {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [validationState, setValidationState] = useState<FieldValidationState>({});

  const validateMutation = useFormValidation(formDefinition.business_object.id);
  const saveMutation = useFormSave();
  const submitMutation = useFormSubmit();

  // ============================================================================
  // FIELD CHANGE HANDLER
  // ============================================================================

  const handleFieldChange = useCallback((fieldName: string, value: any) => {
    setFormData((prev) => ({
      ...prev,
      [fieldName]: value,
    }));

    // Clear validation state when user modifies field
    setValidationState((prev) => ({
      ...prev,
      [fieldName]: {
        ...(prev[fieldName] || {}),
        touched: true,
      },
    }));
  }, []);

  // ============================================================================
  // FIELD VALIDATION HANDLER (real-time, on blur)
  // ============================================================================

  const handleFieldValidation = useCallback(
    async (fieldName: string, value: any) => {
      // Mark field as touched
      setValidationState((prev) => ({
        ...prev,
        [fieldName]: {
          ...(prev[fieldName] || {}),
          touched: true,
          validating: true,
        },
      }));

      // Call backend validation for this field
      try {
        const result = await validateMutation.mutateAsync({
          ...formData,
          [fieldName]: value,
        });

        // Filter errors/warnings for this field only
        const fieldErrors = result.errors.filter((e) => e.field_name === fieldName);
        const fieldWarnings = result.warnings.filter((w) => w.field_name === fieldName);

        setValidationState((prev) => ({
          ...prev,
          [fieldName]: {
            errors: fieldErrors,
            warnings: fieldWarnings,
            validating: false,
            touched: true,
          },
        }));
      } catch (error) {
        console.error('Validation error:', error);
        setValidationState((prev) => ({
          ...prev,
          [fieldName]: {
            ...(prev[fieldName] || {}),
            validating: false,
          },
        }));
      }
    },
    [formData, validateMutation]
  );

  // ============================================================================
  // FULL FORM VALIDATION (before submit)
  // ============================================================================

  const validateForm = useCallback(async (): Promise<boolean> => {
    try {
      const result = await validateMutation.mutateAsync(formData);

      const newValidationState: FieldValidationState = {};
      formDefinition.business_object.fields.forEach((field) => {
        const fieldErrors = result.errors.filter((e) => e.field_name === field.field_name);
        const fieldWarnings = result.warnings.filter((w) => w.field_name === field.field_name);

        newValidationState[field.field_name] = {
          errors: fieldErrors,
          warnings: fieldWarnings,
          validating: false,
          touched: true,
        };
      });

      setValidationState(newValidationState);
      return result.valid;
    } catch (error) {
      console.error('Form validation error:', error);
      return false;
    }
  }, [formData, validateMutation, formDefinition]);

  // ============================================================================
  // ACTION HANDLERS
  // ============================================================================

  const handleAction = useCallback(
    async (actionType: string, bpId?: string) => {
      // For save/submit, validate first
      if (actionType === 'save' || actionType === 'submit') {
        const isValid = await validateForm();

        if (!isValid && actionType === 'submit') {
          // Show error message
          alert('Please fix validation errors before submitting.');
          return;
        }
      }

      if (actionType === 'cancel') {
        // Reset form
        setFormData({});
        setValidationState({});
        return;
      }

      if (actionType === 'save') {
        try {
          const result = await saveMutation.mutateAsync({
            bo_id: formDefinition.business_object.id,
            data: formData,
          });

          alert(`Form saved successfully. Record ID: ${result.record_id}`);
          setFormData({});
          setValidationState({});
        } catch (error) {
          alert(`Save failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
        return;
      }

      if (actionType === 'submit' && bpId) {
        try {
          const result = await submitMutation.mutateAsync({
            bo_id: formDefinition.business_object.id,
            bp_id: bpId,
            data: formData,
          });

          alert(
            `Form submitted successfully!\n\nRecord ID: ${result.record_id}\nWorkflow ID: ${result.workflow_id}`
          );
          setFormData({});
          setValidationState({});
        } catch (error) {
          alert(`Submission failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
        return;
      }
    },
    [formData, validateForm, saveMutation, submitMutation, formDefinition]
  );

  const isLoading =
    validateMutation.isPending || saveMutation.isPending || submitMutation.isPending;

  return (
    <div className="dynamic-form-generator">
      <div className="form-header">
        <h1>{formDefinition.business_object.bo_name} Entry Form</h1>
        {formDefinition.business_object.bo_description && (
          <p className="description">{formDefinition.business_object.bo_description}</p>
        )}
      </div>

      <form className="form-content">
        {formDefinition.sections.map((section) => (
          <FormSection
            key={section.id}
            section={section}
            fields={formDefinition.business_object.fields}
            formData={formData}
            onChange={handleFieldChange}
            onValidate={handleFieldValidation}
            validationMap={validationState}
            validationRulesMap={
              formDefinition.validations || new Map()
            }
          />
        ))}

        <FormActions
          actions={formDefinition.actions}
          onAction={handleAction}
          isLoading={isLoading}
          isDisabled={false}
        />
      </form>

      <style jsx>{`
        .dynamic-form-generator {
          max-width: 900px;
          margin: 0 auto;
          padding: 24px;
        }

        .form-header {
          margin-bottom: 32px;
        }

        .form-header h1 {
          margin: 0 0 8px 0;
          font-size: 28px;
          font-weight: 600;
          color: #333;
        }

        .description {
          margin: 0;
          font-size: 14px;
          color: #666;
        }

        .form-content {
          display: flex;
          flex-direction: column;
        }
      `}</style>
    </div>
  );
};
```

---

## 🎨 Component: DynamicForm (Wrapper)

### `frontend/src/components/DynamicForm.tsx`

```typescript
import React from 'react';
import { useFormDefinition } from '../hooks/useFormDefinition';
import { DynamicFormGenerator } from './DynamicFormGenerator';

interface DynamicFormProps {
  layoutId: string;
}

export const DynamicForm: React.FC<DynamicFormProps> = ({ layoutId }) => {
  const { data: formDefinition, isLoading, error } = useFormDefinition(layoutId);

  if (isLoading) {
    return (
      <div className="loading-container">
        <div className="spinner"></div>
        <p>Loading form...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="error-container">
        <h2>❌ Error Loading Form</h2>
        <p>{error instanceof Error ? error.message : 'Unknown error occurred'}</p>
      </div>
    );
  }

  if (!formDefinition) {
    return (
      <div className="error-container">
        <h2>Form not found</h2>
        <p>The requested form definition could not be found.</p>
      </div>
    );
  }

  return <DynamicFormGenerator formDefinition={formDefinition} />;
};
```

---

## 📄 Usage Example

### `frontend/src/pages/FormPage.tsx`

```typescript
import React from 'react';
import { useParams } from 'react-router-dom';
import { DynamicForm } from '../components/DynamicForm';

export const FormPage: React.FC = () => {
  const { layoutId } = useParams<{ layoutId: string }>();

  if (!layoutId) {
    return <div>Layout ID not provided</div>;
  }

  return (
    <div style={{ minHeight: '100vh', backgroundColor: '#fff' }}>
      <DynamicForm layoutId={layoutId} />
    </div>
  );
};
```

### In your router:

```typescript
<Route path="/forms/:layoutId" element={<FormPage />} />

// Usage:
// <Link to="/forms/layout_employee_entry">Hire Employee</Link>
```

---

## 🧪 Testing

### Unit Test Example

```typescript
// __tests__/FormField.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { FormField } from '../components/FormField';
import { BOField } from '../types/form';

describe('FormField', () => {
  const mockField: BOField = {
    id: 'field_1',
    bo_id: 'bo_1',
    field_name: 'email',
    field_type: 'string',
    display_label: 'Email Address',
    is_required: true,
    is_readonly: false,
    display_order: 1,
    validation_rule_ids: [],
  };

  it('renders field label and input', () => {
    render(
      <FormField
        field={mockField}
        value=""
        onChange={jest.fn()}
        onValidate={jest.fn()}
        validationRules={[]}
        errors={[]}
        warnings={[]}
      />
    );

    expect(screen.getByLabelText(/email address/i)).toBeInTheDocument();
  });

  it('calls onChange when input changes', () => {
    const onChange = jest.fn();
    render(
      <FormField
        field={mockField}
        value=""
        onChange={onChange}
        onValidate={jest.fn()}
        validationRules={[]}
        errors={[]}
        warnings={[]}
      />
    );

    const input = screen.getByDisplayValue('');
    fireEvent.change(input, { target: { value: 'test@example.com' } });

    expect(onChange).toHaveBeenCalledWith('email', 'test@example.com');
  });

  it('displays error messages', () => {
    const errors = [
      {
        field_id: 'field_1',
        field_name: 'email',
        severity: 'error' as const,
        message: 'Invalid email format',
      },
    ];

    render(
      <FormField
        field={mockField}
        value="invalid"
        onChange={jest.fn()}
        onValidate={jest.fn()}
        validationRules={[]}
        errors={errors}
        warnings={[]}
      />
    );

    expect(screen.getByText('Invalid email format')).toBeInTheDocument();
  });
});
```

---

## 🚀 Summary

You now have:

✅ **Complete React Implementation**:
- Form definition loading
- Real-time validation (on blur)
- Full form validation (before submit)
- Multi-field forms with sections
- Support for all field types (string, date, picklist, reference, etc.)
- Error and warning messages
- Save and submit actions with BP triggering

✅ **TypeScript Interfaces** for all data types

✅ **React Hooks** for backend integration

✅ **Reusable Components** for sections, fields, and actions

✅ **Example usage** in FormPage

**Next Steps**:
1. Install dependencies
2. Create the component files
3. Update your router to include form pages
4. Test with backend endpoints
5. Customize styling to match your design system
