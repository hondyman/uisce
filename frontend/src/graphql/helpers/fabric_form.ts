import React, { useState, useCallback, useMemo, createContext, useContext } from 'react';
import type { ZodError, ZodSchema } from 'zod';

// ---------- Types & helpers ----------
type AnyObj = Record<string, unknown>;
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

function zodIssuesToMap(issues: ZodError['issues']): Record<string, string> {
  const out: Record<string, string> = {};
  for (const issue of issues) {
    const path = issue.path.join('.');
    if (path && !out[path]) out[path] = issue.message;
  }
  return out;
}

// ---------- Context ----------
// TVars is used as a generic for type inference across the form API. ESLint may
// incorrectly flag it as unused; silence that specific rule for this declaration.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export interface FormContextValue<_TVars extends AnyObj> {
  values: _TVars;
  setValue: <P extends KeyPaths>(
    name: P,
    value: PathValue,
    transform?: (val: unknown) => unknown
  ) => void;
  errors: Record<string, string>;
  touched: Record<string, boolean>;
  markTouched: (name: string) => void;
  submitting: boolean;
  submit: (values: _TVars) => void;
  reset: (next?: _TVars) => void;
}

const FormCtx = createContext<FormContextValue<AnyObj> | null>(null);

export function useFormContext<_TVars extends AnyObj>(): FormContextValue<_TVars> {
  const ctx = useContext(FormCtx);
  if (!ctx) throw new Error('useFormContext must be used within FormProvider');
  return ctx as FormContextValue<_TVars>;
}

// ---------- Hook Types ----------
export interface UseMutationFormResult {
  submit: (vars: Record<string, unknown>) => Promise<void>;
  error: { message?: string } | null;
  submitting: boolean;
}

// ---------- Provider ----------
interface FormProviderProps<TVars extends AnyObj> {
  schema: ZodSchema<TVars>;
  initialValues: TVars;
  hook: UseMutationFormResult;
  children: React.ReactNode;
}

export function FormProvider<TVars extends AnyObj>({
  schema,
  initialValues,
  hook,
  children,
}: FormProviderProps<TVars>): React.ReactElement {
  const [values, setValues] = useState<TVars>(initialValues);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  const setValue = useCallback(
    <P extends KeyPaths>(
      name: P,
      value: PathValue,
      transform?: (val: unknown) => unknown
    ) => {
      const finalVal = transform ? transform(value) : value;
      const next = setIn(values, name as string, finalVal) as TVars;
      setValues(next);
      
      const parsed = schema.safeParse(next);
      if (!parsed.success) {
        const map = zodIssuesToMap(parsed.error.issues);
        const msg = map[name as string] ?? '';
        setErrors((prev) => ({ ...prev, [name as string]: msg }));
      } else {
        setErrors((prev) => ({ ...prev, [name as string]: '' }));
      }
    },
    [values, schema]
  );

  const markTouched = useCallback(
    (name: string) => setTouched((t) => ({ ...t, [name]: true })),
    []
  );

  const submit = useCallback(
    (vals: TVars) => {
      const parsed = schema.safeParse(vals);
      if (!parsed.success) {
        const map = zodIssuesToMap(parsed.error.issues);
        setErrors(map);
        setTouched((t) => ({
          ...t,
          ...Object.fromEntries(Object.keys(map).map((k) => [k, true])),
        }));
        return;
      }
      hook.submit(parsed.data as Record<string, unknown>);
    },
    [schema, hook]
  );

  const reset = useCallback(
    (next?: TVars) => {
      setValues(next ?? initialValues);
      setErrors({});
      setTouched({});
    },
    [initialValues]
  );

  const ctx: FormContextValue<TVars> = useMemo(
    () => ({
      values,
      setValue,
      errors,
      touched,
      markTouched,
      submitting: hook.submitting,
      submit,
      reset,
    }),
    [values, setValue, errors, touched, markTouched, hook.submitting, submit, reset]
  );

  return React.createElement(FormCtx.Provider, { value: ctx as FormContextValue<AnyObj> }, children);
}

// ---------- Form wrapper ----------
interface FormProps<TVars extends AnyObj> {
  schema: ZodSchema<TVars>;
  initialValues: TVars;
  hook: UseMutationFormResult;
  render: (ctx: FormContextValue<TVars>) => React.ReactNode;
}

export function Form<TVars extends AnyObj>({
  schema,
  initialValues,
  hook,
  render,
}: FormProps<TVars>): React.ReactElement {
  const innerFormElement = React.createElement(InnerForm, { 
    render: render as (ctx: FormContextValue<AnyObj>) => React.ReactNode, 
    serverError: hook.error?.message 
  });

  return React.createElement(
    FormProvider,
    { 
      schema, 
      initialValues, 
      hook,
      children: innerFormElement
    }
  );
}

function InnerForm({
  render,
  serverError,
}: {
  render: (ctx: FormContextValue<AnyObj>) => React.ReactNode;
  serverError?: string;
}): React.ReactElement {
  const ctx = useFormContext<AnyObj>();
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    ctx.submit(ctx.values);
  };

  return React.createElement(
    'form',
    { onSubmit: handleSubmit },
    render(ctx),
    serverError && React.createElement(
      'div',
      { style: { color: 'red', marginTop: 8 } },
      serverError
    )
  );
}

// ---------- Base Field Props ----------
// Some generic type parameters are used for inference by consumers; ESLint may
// incorrectly report them as unused. Silence that rule for these declarations.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export interface BaseFieldProps<_TVars extends AnyObj, P extends KeyPaths> {
  name: P;
  label?: string;
  helpText?: string;
  required?: boolean;
  transform?: (val: unknown) => unknown;
  containerClassName?: string;
  labelClassName?: string;
  errorClassName?: string;
}

// ---------- Field shell ----------
function FieldShell({
  label,
  required,
  error,
  helpText,
  containerClassName,
  labelClassName,
  errorClassName,
  children,
}: {
  label?: string;
  required?: boolean;
  error?: string;
  helpText?: string;
  containerClassName?: string;
  labelClassName?: string;
  errorClassName?: string;
  children: React.ReactNode;
}): React.ReactElement {
  return React.createElement(
    'div',
    { className: containerClassName },
    label && React.createElement(
      'label',
      { className: labelClassName },
      label,
      required && React.createElement('span', null, '*')
    ),
    children,
    helpText && React.createElement(
      'div',
      { style: { fontSize: 12, opacity: 0.7 } },
      helpText
    ),
    error && React.createElement(
      'div',
      { 
        className: errorClassName, 
        style: { color: 'crimson', fontSize: 12 } 
      },
      error
    )
  );
}

// ---------- Input ----------
export function InputField<TVars extends AnyObj, P extends KeyPaths>(
  props: BaseFieldProps<TVars, P> & React.InputHTMLAttributes<HTMLInputElement>
): React.ReactElement {
  const { name, label, helpText, required, transform, containerClassName, labelClassName, errorClassName, ...rest } = props;
  const { values, setValue, errors, touched, markTouched } = useFormContext<TVars>();
  const val = getIn(values, name as string) ?? '';
  const error = touched[name as string] ? errors[name as string] : undefined;

  return React.createElement(
    FieldShell,
    {
      label,
      required,
      helpText,
      error,
      containerClassName,
      labelClassName,
      errorClassName,
      children: React.createElement('input', {
        ...rest,
        value: val as string,
        onChange: (e: React.ChangeEvent<HTMLInputElement>) => setValue(name, e.target.value, transform),
        onBlur: () => markTouched(name as string),
      })
    }
  );
}

// ---------- Number ----------
export function NumberField<TVars extends AnyObj, P extends KeyPaths>(
  props: BaseFieldProps<TVars, P> & React.InputHTMLAttributes<HTMLInputElement>
): React.ReactElement {
  const { name, label, helpText, required, transform, containerClassName, labelClassName, errorClassName, ...rest } = props;
  const { values, setValue, errors, touched, markTouched } = useFormContext<TVars>();
  const val = getIn(values, name as string) ?? '';
  const error = touched[name as string] ? errors[name as string] : undefined;

  return React.createElement(
    FieldShell,
    {
      label,
      required,
      helpText,
      error,
      containerClassName,
      labelClassName,
      errorClassName,
      children: React.createElement('input', {
        ...rest,
        type: 'number',
        value: val as string,
        onChange: (e: React.ChangeEvent<HTMLInputElement>) => {
          const raw = e.target.value;
          const parsed = raw === '' ? '' : Number(raw);
          setValue(name, parsed, transform);
        },
        onBlur: () => markTouched(name as string),
      })
    }
  );
}

// ---------- Select ----------
export interface SelectOption<T = string> {
  value: T;
  label: string;
}

export function SelectField<TVars extends AnyObj, P extends KeyPaths, T = string>(
  props: BaseFieldProps<TVars, P> &
    React.SelectHTMLAttributes<HTMLSelectElement> & { options: SelectOption<T>[] }
): React.ReactElement {
  const { name, label, helpText, required, transform, containerClassName, labelClassName, errorClassName, options, ...rest } =
    props;
  const { values, setValue, errors, touched, markTouched } = useFormContext<TVars>();
  const val = getIn(values, name as string) ?? '';
  const error = touched[name as string] ? errors[name as string] : undefined;

  return React.createElement(
    FieldShell,
    {
      label,
      required,
      helpText,
      error,
      containerClassName,
      labelClassName,
      errorClassName,
      children: React.createElement(
        'select',
        {
          ...rest,
          value: val as string,
          onChange: (e: React.ChangeEvent<HTMLSelectElement>) => setValue(name, e.target.value, transform),
          onBlur: () => markTouched(name as string),
        },
        React.createElement(
          'option',
          { value: '', disabled: true, hidden: true },
          'Select…'
        ),
        options.map((option) =>
          React.createElement(
            'option',
            { key: String(option.value), value: String(option.value) },
            option.label
          )
        )
      )
    }
  );
}

// ---------- TextArea ----------
export function TextAreaField<TVars extends AnyObj, P extends KeyPaths>(
  props: BaseFieldProps<TVars, P> & React.TextareaHTMLAttributes<HTMLTextAreaElement>
): React.ReactElement {
  const { name, label, helpText, required, transform, containerClassName, labelClassName, errorClassName, ...rest } = props;
  const { values, setValue, errors, touched, markTouched } = useFormContext<TVars>();
  const val = getIn(values, name as string) ?? '';
  const error = touched[name as string] ? errors[name as string] : undefined;

  return React.createElement(
    FieldShell,
    {
      label,
      required,
      helpText,
      error,
      containerClassName,
      labelClassName,
      errorClassName,
      children: React.createElement('textarea', {
        ...rest,
        value: val as string,
        onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => setValue(name, e.target.value, transform),
        onBlur: () => markTouched(name as string),
      })
    }
  );
}

// ---------- Checkbox ----------
export function CheckboxField<TVars extends AnyObj, P extends KeyPaths>(
  props: BaseFieldProps<TVars, P> & React.InputHTMLAttributes<HTMLInputElement>
): React.ReactElement {
  const { name, label, helpText, required, transform, containerClassName, labelClassName, errorClassName, ...rest } = props;
  const { values, setValue, errors, touched, markTouched } = useFormContext<TVars>();
  const val = !!getIn(values, name as string);
  const error = touched[name as string] ? errors[name as string] : undefined;

  return React.createElement(
    FieldShell,
    {
      label,
      required,
      helpText,
      error,
      containerClassName,
      labelClassName,
      errorClassName,
      children: React.createElement('input', {
        ...rest,
        type: 'checkbox',
        checked: val,
        onChange: (e: React.ChangeEvent<HTMLInputElement>) => setValue(name, e.target.checked, transform),
        onBlur: () => markTouched(name as string),
      })
    }
  );
}