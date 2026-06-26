import React, { useState, useCallback } from 'react';
import type { ZodError, ZodSchema } from 'zod';

// ---------- Types & helpers ----------
// helper types (kept for compatibility)
export type KeyPaths = string; // simplified for Vite perf
export type PathValue = unknown; // simplified for Vite perf

export function getIn(obj: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>(
    (acc, key) =>
      acc == null ? acc : (acc as Record<string, unknown>)[key],
    obj as unknown
  );
}

export function setIn(obj: unknown, path: string, value: unknown): unknown {
  const keys = path.split('.');
  const clone: unknown = Array.isArray(obj) ? [...(obj as unknown[])] : { ...(obj as Record<string, unknown>) };
  let cur: Record<string, unknown> = clone as Record<string, unknown>;
  
  for (let i = 0; i < keys.length - 1; i++) {
    const k = keys[i];
    const next = cur[k];
    cur[k] = Array.isArray(next) ? [...(next as unknown[])] : { ...(next as Record<string, unknown> ?? {}) };
    cur = cur[k] as Record<string, unknown>;
  }
  cur[keys[keys.length - 1]] = value;
  return clone;
}

export function zodIssuesToMap(issues: ZodError['issues']): Record<string, string> {
  const out: Record<string, string> = {};
  for (const issue of issues) {
    const path = issue.path.join('.');
    if (path && !out[path]) out[path] = issue.message;
  }
  return out;
}

// ---------- useMutationForm Hook ----------
export interface UseMutationFormOptions<TData, TVars> {
  mutationFn: (variables: TVars) => Promise<TData>;
  onSuccess?: (data: TData) => void;
  onError?: (error: Error) => void;
}

export interface UseMutationFormResult<TData, TVars> {
  submit: (variables: TVars) => Promise<void>;
  error: { message?: string } | null;
  submitting: boolean;
  data: TData | null;
}

export function useMutationForm<TData, TVars extends Record<string, any>>({
  mutationFn,
  onSuccess,
  onError,
}: UseMutationFormOptions<TData, TVars>): UseMutationFormResult<TData, TVars> {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [data, setData] = useState<TData | null>(null);

  const submit = useCallback(
    async (variables: TVars) => {
      setSubmitting(true);
      setError(null);
      
      try {
        const result = await mutationFn(variables);
        setData(result);
        onSuccess?.(result);
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Unknown error');
        setError({ message: error.message });
        onError?.(error);
      } finally {
        setSubmitting(false);
      }
    },
    [mutationFn, onSuccess, onError]
  );

  return {
    submit,
    error,
    submitting,
    data,
  };
}

// ---------- Form Types ----------
type FieldErrors<T> = Partial<Record<keyof T, string>>;

interface FormProps<TData, TVars extends Record<string, any>> {
  schema: ZodSchema<TVars>;
  useFormHook: UseMutationFormResult<TData, TVars>;
  initialValues: TVars;
  render: (props: {
    values: TVars;
    setValue: <K extends keyof TVars>(key: K, value: TVars[K]) => void;
    errors: FieldErrors<TVars>;
    submitting: boolean;
  }) => React.ReactNode;
}

/**
 * Generic form component that:
 *  - Controls state for all schema fields
 *  - Runs Zod validation on change and on submit
 *  - Delegates submit to `useMutationForm`
 */
export function Form<TData, TVars extends Record<string, any>>({
  schema,
  useFormHook,
  initialValues,
  render,
}: FormProps<TData, TVars>): React.ReactElement {
  const { submit, error, submitting } = useFormHook;

  const [values, setValues] = useState<TVars>(initialValues);
  const [errors, setErrors] = useState<FieldErrors<TVars>>({});

  function setValue<K extends keyof TVars>(key: K, value: TVars[K]) {
    const updated = { ...values, [key]: value };
    setValues(updated);

    // Per-field validation
    const parsed = schema.safeParse(updated);
    if (!parsed.success) {
      const fieldError =
        parsed.error.flatten().fieldErrors[key as string]?.[0] ?? '';
      setErrors((prev) => ({ ...prev, [key]: fieldError }));
    } else {
      setErrors((prev) => ({ ...prev, [key]: '' }));
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const parsed = schema.safeParse(values);
    if (!parsed.success) {
      const zErr: ZodError<TVars> = parsed.error;
      const newErrors: FieldErrors<TVars> = {};
      for (const [field, msgs] of Object.entries(
        zErr.flatten().fieldErrors
      )) {
        (newErrors as any)[field] = msgs?.[0] ?? '';
      }
      setErrors(newErrors);
      return;
    }
    submit(parsed.data);
  }

  return React.createElement(
    'form',
    { onSubmit: handleSubmit },
    render({ values, setValue, errors, submitting }),
    error && React.createElement(
      'div',
      { style: { color: 'red' } },
      error.message
    )
  );
}