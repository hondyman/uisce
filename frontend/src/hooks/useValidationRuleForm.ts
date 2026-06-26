/**
 * useValidationRuleForm Hook
 * Manages form state and validation for validation rules
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import {
  ValidationRuleFormData,
  createDefaultRuleFormData,
  buildRuleFormDataFromRule,
  validateRuleForm,
  hasRuleChanged,
} from '../lib/ruleUtils';

export interface FormErrors {
  [key: string]: string;
}

export interface UseValidationRuleFormOptions {
  onSubmit?: (formData: ValidationRuleFormData) => Promise<void>;
  initialRule?: any;
}

export const useValidationRuleForm = (options: UseValidationRuleFormOptions = {}) => {
  const { onSubmit, initialRule } = options;

  const [formData, setFormData] = useState<ValidationRuleFormData>(
    initialRule ? buildRuleFormDataFromRule(initialRule) : createDefaultRuleFormData()
  );

  const [errors, setErrors] = useState<FormErrors>({});
  const [touched, setTouched] = useState<Set<string>>(new Set());
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [hasChanges, setHasChanges] = useState(false);

  const initialDataRef = useRef(
    initialRule ? buildRuleFormDataFromRule(initialRule) : createDefaultRuleFormData()
  );

  // Track changes
  useEffect(() => {
    const changed = hasRuleChanged(initialDataRef.current, formData);
    setHasChanges(changed);
  }, [formData]);

  /**
   * Update a single form field
   */
  const updateField = useCallback((field: keyof ValidationRuleFormData, value: any) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // Clear error for this field when user starts typing
    if (touched.has(field)) {
      validateField(field, value);
    }
  }, [touched]);

  /**
   * Validate a single field
   */
  const validateField = useCallback((field: keyof ValidationRuleFormData, value?: any) => {
    const valueToValidate = value ?? formData[field];
    let fieldError = '';

    switch (field) {
      case 'name':
        if (!valueToValidate || !valueToValidate.trim()) {
          fieldError = 'Rule name is required';
        } else if (valueToValidate.length > 100) {
          fieldError = 'Rule name must be 100 characters or less';
        }
        break;

      case 'ruleType':
        if (!valueToValidate) {
          fieldError = 'Rule type is required';
        }
        break;

      case 'accountTypes':
        if (!valueToValidate || valueToValidate.length === 0) {
          fieldError = 'At least one account type must be selected';
        }
        break;

      case 'evaluationOrder':
        if (valueToValidate < 0) {
          fieldError = 'Evaluation order must be non-negative';
        }
        break;

      case 'description':
        if (valueToValidate && valueToValidate.length > 500) {
          fieldError = 'Description must be 500 characters or less';
        }
        break;

      default:
        break;
    }

    setErrors((prev) => {
      if (fieldError) {
        return { ...prev, [field]: fieldError };
      }
      const updated = { ...prev };
      delete updated[field];
      return updated;
    });

    return !fieldError;
  }, [formData]);

  /**
   * Mark field as touched and validate
   */
  const touchField = useCallback((field: keyof ValidationRuleFormData) => {
    setTouched((prev) => new Set([...prev, field]));
    validateField(field);
  }, [validateField]);

  /**
   * Validate entire form
   */
  const validateForm = useCallback((): boolean => {
    const validationErrors = validateRuleForm(formData);
    const errorMap: FormErrors = {};

    validationErrors.forEach((error) => {
      // Map error messages to fields
      if (error.includes('name')) errorMap.name = error;
      else if (error.includes('type')) errorMap.ruleType = error;
      else if (error.includes('account')) errorMap.accountTypes = error;
      else if (error.includes('Order')) errorMap.evaluationOrder = error;
      else if (error.includes('Description')) errorMap.description = error;
    });

    setErrors(errorMap);
    // Mark all fields as touched
    setTouched(new Set(['name', 'ruleType', 'accountTypes', 'description', 'evaluationOrder']));
    return Object.keys(errorMap).length === 0;
  }, [formData]);

  /**
   * Handle form submission
   */
  const handleSubmit = useCallback(
    async (e?: React.FormEvent) => {
      if (e) e.preventDefault();

      // Validate form
      if (!validateForm()) {
        setSubmitError('Please fix the errors above');
        return false;
      }

      setIsSubmitting(true);
      setSubmitError(null);

      try {
        if (onSubmit) {
          await onSubmit(formData);
        }

        // Reset to new state after successful submission
        initialDataRef.current = { ...formData };
        setHasChanges(false);
        return true;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to save rule';
        setSubmitError(message);
        return false;
      } finally {
        setIsSubmitting(false);
      }
    },
    [formData, validateForm, onSubmit]
  );

  /**
   * Reset form to initial state
   */
  const reset = useCallback(() => {
    setFormData({ ...initialDataRef.current });
    setErrors({});
    setTouched(new Set());
    setSubmitError(null);
    setHasChanges(false);
  }, []);

  /**
   * Reset to new blank form
   */
  const resetToBlank = useCallback(() => {
    setFormData(createDefaultRuleFormData());
    setErrors({});
    setTouched(new Set());
    setSubmitError(null);
    setHasChanges(false);
    initialDataRef.current = createDefaultRuleFormData();
  }, []);

  /**
   * Get field error (only show if touched)
   */
  const getFieldError = useCallback(
    (field: keyof ValidationRuleFormData): string | undefined => {
      return touched.has(field) ? errors[field] : undefined;
    },
    [errors, touched]
  );

  /**
   * Check if field has error
   */
  const hasFieldError = useCallback(
    (field: keyof ValidationRuleFormData): boolean => {
      return touched.has(field) && !!errors[field];
    },
    [errors, touched]
  );

  /**
   * Get all validation errors
   */
  const getAllErrors = useCallback((): string[] => {
    return Object.values(errors).filter(Boolean);
  }, [errors]);

  /**
   * Check if form is valid
   */
  const isValid = useCallback((): boolean => {
    return Object.keys(errors).length === 0;
  }, [errors]);

  return {
    // State
    formData,
    errors,
    touched,
    isSubmitting,
    submitError,
    hasChanges,

    // Methods
    updateField,
    validateField,
    touchField,
    validateForm,
    handleSubmit,
    reset,
    resetToBlank,
    getFieldError,
    hasFieldError,
    getAllErrors,
    isValid,

    // Utilities
    setErrors,
    setFormData,
    setSubmitError,
  };
};
