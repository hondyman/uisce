import React from 'react';
import { Autocomplete, Chip, FormControlLabel, FormLabel, MenuItem, Radio, RadioGroup, Stack, Switch, TextField } from '@mui/material';
import type { FormFieldSpec, OptionsFrom } from './appModel';
import { getPath, text, type Scope } from './bindings';
import { useCondition } from './conditions';

/** A field's options: its static list, or rows of a query. */
export function fieldOptions(f: { options?: FormFieldSpec['options']; optionsFrom?: OptionsFrom }, scope: Scope): { value: string; label: string }[] {
  if (f.optionsFrom) {
    const { query, rowsPath, valueField, labelField } = f.optionsFrom;
    const data = getPath(scope, `queries.${query}.data`);
    const rows = (rowsPath ? getPath(data, rowsPath) : data) as unknown[] | undefined;
    return (Array.isArray(rows) ? rows : []).map((r) => {
      const value = valueField ? getPath(r, valueField) : r;
      const label = labelField ? getPath(r, labelField) : value;
      return { value: String(value ?? ''), label: String(label ?? '') };
    });
  }
  return (f.options ?? []).map((o) => ({ value: o.value, label: text(o.label, scope) }));
}

export const isBlank = (v: unknown) => v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0);

/** Whether every required, visible field has a value (visibility is re-checked by the caller's own conditions). */
export function requiredFilled(fields: FormFieldSpec[], values: Record<string, unknown>): boolean {
  return fields.every((f) => !f.required || f.visibleWhen || !isBlank(typeof values[f.name] === 'string' ? (values[f.name] as string).trim() : values[f.name]));
}

function Field({ f, value, onChange, scope }: { f: FormFieldSpec; value: unknown; onChange: (v: unknown) => void; scope: Scope }) {
  const visible = useCondition(f.visibleWhen, scope, true);
  const readOnly = useCondition(f.readOnlyWhen, scope, false);
  if (f.visibleWhen && !visible) return null;
  const disabled = !!f.readOnlyWhen && readOnly;
  const label = text(f.label, scope);
  const helper = f.helperText ? text(f.helperText, scope) : undefined;
  switch (f.kind) {
    case 'radio':
      return (
        <Stack>
          {label && <FormLabel>{label}</FormLabel>}
          <RadioGroup value={value ?? ''} onChange={(e) => onChange(e.target.value)}>
            {fieldOptions(f, scope).map((o) => <FormControlLabel key={o.value} value={o.value} control={<Radio size="small" disabled={disabled} />} label={o.label} />)}
          </RadioGroup>
        </Stack>
      );
    case 'switch':
      return <FormControlLabel control={<Switch checked={!!value} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />} label={label} />;
    case 'chips':
      return (
        <Autocomplete multiple freeSolo options={fieldOptions(f, scope).map((o) => o.value)} value={Array.isArray(value) ? value.map(String) : []} disabled={disabled}
          onChange={(_, v) => onChange(v)}
          renderTags={(vals, getTagProps) => vals.map((x, i) => <Chip size="small" label={x} {...getTagProps({ index: i })} key={x} />)}
          renderInput={(p) => <TextField {...p} size="small" label={label} required={f.required} helperText={helper ?? 'Type a value and press Enter'} />} />
      );
    case 'select':
      return (
        <TextField select size="small" fullWidth label={label} value={value ?? ''} disabled={disabled} required={f.required} helperText={helper}
          onChange={(e) => onChange(e.target.value)}>
          {!f.required && <MenuItem value=""><em>—</em></MenuItem>}
          {fieldOptions(f, scope).map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
        </TextField>
      );
    case 'number':
      return (
        <TextField size="small" fullWidth type="number" label={label} value={value ?? ''} disabled={disabled} required={f.required} helperText={helper}
          onChange={(e) => onChange(e.target.value === '' ? null : Number(e.target.value))} />
      );
    case 'date':
      return (
        <TextField size="small" fullWidth type="date" label={label} value={value ?? ''} disabled={disabled} required={f.required} helperText={helper}
          InputLabelProps={{ shrink: true }} onChange={(e) => onChange(e.target.value)} />
      );
    default:
      return (
        <TextField size="small" fullWidth label={label} value={value ?? ''} disabled={disabled} required={f.required} helperText={helper}
          multiline={f.kind === 'multiline'} minRows={f.kind === 'multiline' ? 2 : undefined} onChange={(e) => onChange(e.target.value)} />
      );
  }
}

/**
 * The fields of a form - shared by action forms (runOperation.form, a dialog)
 * and the Form widget (bound to a page variable), so the two never drift.
 */
export function FormFields({ fields, values, onChange, scope, columns = 1 }: {
  fields: FormFieldSpec[];
  values: Record<string, unknown>;
  onChange: (name: string, v: unknown) => void;
  scope: Scope;
  columns?: number;
}) {
  return (
    <Stack spacing={2} sx={columns > 1 ? { display: 'grid', gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`, gap: 2, '& > *': { m: '0 !important' } } : undefined}>
      {fields.map((f) => <Field key={f.name} f={f} value={values[f.name]} onChange={(v) => onChange(f.name, v)} scope={{ ...scope, form: values }} />)}
    </Stack>
  );
}
