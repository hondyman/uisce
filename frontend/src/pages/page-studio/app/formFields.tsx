import React, { useEffect, useState } from 'react';
import { Alert, Autocomplete, Box, Button, Checkbox, Chip, CircularProgress, FormControlLabel, FormLabel, IconButton, MenuItem, Radio, RadioGroup, Stack, Switch, TextField, Typography } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import type { FormFieldSpec, OptionsFrom, RowFieldSpec } from './appModel';
import { getPath, resolve, text, type Scope } from './bindings';
import { useCondition } from './conditions';
import { MapField } from './mapField';
import { useAppRuntime } from './AppRuntime';
import { getOperation } from '../../../studio-core/operations/registry';
import { PageIcon } from './icons';

/** A field's options: its static list, or rows of a query. */
export type Badge = { label: string; color?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info'; variant?: 'filled' | 'outlined' };

export function fieldOptions(f: { options?: FormFieldSpec['options']; optionsFrom?: OptionsFrom }, scope: Scope): { value: string; label: string; caption?: string; badges?: Badge[] }[] {
  if (f.optionsFrom) {
    const { query, rowsPath, valueField, labelField, captionField, badgesField } = f.optionsFrom;
    const data = getPath(scope, `queries.${query}.data`);
    const rows = (rowsPath ? getPath(data, rowsPath) : data) as unknown[] | undefined;
    return (Array.isArray(rows) ? rows : []).map((r) => {
      const value = valueField ? getPath(r, valueField) : r;
      const label = labelField ? getPath(r, labelField) : value;
      const caption = captionField ? getPath(r, captionField) : undefined;
      const badges = badgesField ? getPath(r, badgesField) : undefined;
      return {
        value: String(value ?? ''), label: String(label ?? ''), ...(caption ? { caption: String(caption) } : {}),
        ...(Array.isArray(badges) ? { badges: badges as Badge[] } : {}),
      };
    });
  }
  return (f.options ?? []).map((o) => ({ value: o.value, label: text(o.label, scope), ...(o.caption ? { caption: text(o.caption, scope) } : {}), ...(o.badges ? { badges: o.badges } : {}) }));
}

export const isBlank = (v: unknown) => v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0);

/** Whether every required, visible field has a value (visibility is re-checked by the caller's own conditions). */
export function requiredFilled(fields: FormFieldSpec[], values: Record<string, unknown>): boolean {
  return fields.every((f) => !f.required || f.visibleWhen || !isBlank(typeof values[f.name] === 'string' ? (values[f.name] as string).trim() : values[f.name]));
}

type Row = Record<string, unknown>;

/** One cell of a row: its own kind, options, read-only, caption, visibility (all seeing {{row}}). */
function RowCell({ c, row, onChange, disabled, scope }: { c: RowFieldSpec; row: Row; onChange: (v: unknown) => void; disabled: boolean; scope: Scope }) {
  const s: Scope = { ...scope, row };
  const visible = useCondition(c.visibleWhen, s, true);
  if (c.visibleWhen && !visible) return <Box sx={{ flex: c.flex ?? 1 }} />;
  const v = row[c.name];
  const label = text(c.label, s);
  const off = disabled || !!c.readOnly;
  const caption = c.caption !== undefined ? resolve(c.caption, s) : undefined;
  let input: React.ReactNode;
  switch (c.kind) {
    case 'select':
      input = (
        <TextField select size="small" fullWidth label={label} value={v === undefined || v === null ? '' : String(v)} disabled={off} onChange={(e) => onChange(e.target.value)}>
          {fieldOptions(c, s).map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
          {v !== undefined && v !== null && v !== '' && !fieldOptions(c, s).some((o) => o.value === String(v)) && <MenuItem value={String(v)}>{String(v)} (missing)</MenuItem>}
        </TextField>
      );
      break;
    case 'switch':
      input = <FormControlLabel control={<Checkbox size="small" checked={!!v} disabled={off} onChange={(e) => onChange(e.target.checked)} />} label={label} />;
      break;
    case 'chips':
      input = (
        <Autocomplete multiple freeSolo size="small" options={fieldOptions(c, s).map((o) => o.value)} value={Array.isArray(v) ? v.map(String) : []} disabled={off}
          onChange={(_, n) => onChange(n)} renderInput={(p) => <TextField {...p} label={label} placeholder={c.placeholder ? text(c.placeholder, s) : undefined} />} />
      );
      break;
    default:
      input = (
        <TextField size="small" fullWidth label={label} value={v === undefined || v === null ? '' : String(v)} disabled={off}
          placeholder={c.placeholder ? text(c.placeholder, s) : undefined}
          onChange={(e) => {
            const n = Number(e.target.value);
            onChange(c.kind === 'number' && e.target.value !== '' && !Number.isNaN(n) ? n : e.target.value);
          }} />
      );
  }
  return (
    <Box sx={{ flex: c.flex ?? 1, minWidth: 0 }}>
      {input}
      {caption !== undefined && caption !== null && caption !== '' && <Typography variant="caption" color="text.secondary" noWrap component="div">{String(caption)}</Typography>}
    </Box>
  );
}

/** A list of objects edited as rows (a match rule's fuzzy keys, a map step's field maps, a file's columns). */
function RowsField({ f, value, onChange, disabled, label, scope }: { f: FormFieldSpec; value: unknown; onChange: (v: unknown) => void; disabled: boolean; label: string; scope: Scope }) {
  const rows = (Array.isArray(value) ? value : []) as Row[];
  const cols: RowFieldSpec[] = f.rowFields ?? Array.from(new Set(rows.flatMap((r) => Object.keys(r)))).map((name) => ({ name, label: name, kind: 'text' as const }));
  const set = (i: number, name: string, v: unknown) => onChange(rows.map((r, j) => (j === i ? { ...r, [name]: v } : r)));
  return (
    <Box>
      {label && <Typography variant="caption" color="text.secondary">{label}</Typography>}
      <Stack spacing={1}>
        {rows.map((r, i) => (
          <Stack key={i} direction="row" spacing={1} alignItems="flex-start">
            {cols.map((c) => <RowCell key={c.name} c={c} row={r} disabled={disabled} scope={scope} onChange={(v) => set(i, c.name, v)} />)}
            {!f.fixedRows && (
              <IconButton size="small" aria-label="Remove" disabled={disabled} onClick={() => onChange(rows.filter((_, j) => j !== i))}><DeleteIcon fontSize="small" /></IconButton>
            )}
          </Stack>
        ))}
        {!f.fixedRows && (
          <Box>
            <Button size="small" startIcon={<AddIcon />} disabled={disabled}
              onClick={() => onChange([...rows, Object.fromEntries(cols.map((c) => [c.name, c.default !== undefined ? c.default : c.kind === 'number' ? 0 : c.kind === 'switch' ? false : c.kind === 'chips' ? [] : '']))])}>
              {f.addLabel ? text(f.addLabel, scope) : 'Add'}
            </Button>
          </Box>
        )}
      </Stack>
    </Box>
  );
}

/** A button inside a form: runs actions with {{form}} (Read file, Suggest mappings, Remove this step). */
function FormButton({ f, scope, disabled, label }: { f: FormFieldSpec; scope: Scope; disabled: boolean; label: string }) {
  const { runActions } = useAppRuntime();
  const [busy, setBusy] = useState(false);
  const click = async () => {
    setBusy(true);
    try { await runActions(f.onClick, scope); } finally { setBusy(false); }
  };
  return (
    <Box>
      <Button size="small" variant={f.buttonVariant ?? 'outlined'} disabled={disabled || busy} onClick={() => void click()}
        startIcon={busy ? <CircularProgress size={16} /> : f.icon ? <PageIcon name={f.icon} fontSize="small" /> : undefined}>
        {label}
      </Button>
    </Box>
  );
}

/** Upload a file through an operation ({file}), then run actions with {{result}}. */
function UploadButton({ f, scope, disabled, label }: { f: FormFieldSpec; scope: Scope; disabled: boolean; label: string }) {
  const { runActions } = useAppRuntime();
  const ref = React.useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pick = async (file: File) => {
    const op = f.uploadOperation ? getOperation(f.uploadOperation) : undefined;
    if (!op) { setError(`Unknown operation ${f.uploadOperation}`); return; }
    setBusy(true); setError(null);
    try {
      const result = await op.run({ file });
      await runActions(f.onClick, { ...scope, result });
    } catch (e) { setError(e instanceof Error ? e.message : String(e)); } finally { setBusy(false); }
  };
  return (
    <Box>
      <input ref={ref} type="file" hidden accept={f.accept} aria-label={label} onChange={(e) => { const x = e.target.files?.[0]; if (x) void pick(x); e.target.value = ''; }} />
      <Button size="small" variant={f.buttonVariant ?? 'outlined'} disabled={disabled || busy} onClick={() => ref.current?.click()}
        startIcon={busy ? <CircularProgress size={16} /> : <PageIcon name={f.icon ?? 'export'} fontSize="small" />}>
        {label}
      </Button>
      {error && <Alert severity="error" sx={{ mt: 1 }}>{error}</Alert>}
    </Box>
  );
}

/** Several options picked with checkboxes, each with a caption and badges (rules with their severity). */
function Checklist({ f, value, onChange, disabled, label, scope }: { f: FormFieldSpec; value: unknown; onChange: (v: unknown) => void; disabled: boolean; label: string; scope: Scope }) {
  const picked = new Set(Array.isArray(value) ? value.map(String) : []);
  return (
    <Box>
      {label && <Typography variant="caption" color="text.secondary">{label}</Typography>}
      <Stack>
        {fieldOptions(f, scope).map((o) => (
          <FormControlLabel key={o.value} sx={{ alignItems: 'flex-start' }} disabled={disabled}
            control={<Checkbox checked={picked.has(o.value)} onChange={(e) => onChange(e.target.checked ? [...picked, o.value] : [...picked].filter((x) => x !== o.value))} />}
            label={
              <Box>
                <Stack direction="row" spacing={1} alignItems="center">
                  <Typography variant="body2">{o.label}</Typography>
                  {(o.badges ?? []).map((b, i) => <Chip key={i} size="small" label={b.label} color={b.color ?? 'default'} variant={b.variant ?? 'filled'} />)}
                </Stack>
                {o.caption && <Typography variant="caption" color="text.secondary">{o.caption}</Typography>}
              </Box>
            } />
        ))}
      </Stack>
    </Box>
  );
}

/** Any JSON value as text; kept as text until it parses. */
function JsonText({ value, onChange, ...rest }: { value: unknown; onChange: (v: unknown) => void; label: string; disabled: boolean; required?: boolean; helperText?: string }) {
  const [draft, setDraft] = useState(() => (typeof value === 'string' ? value : value === undefined || value === null ? '' : JSON.stringify(value, null, 2)));
  useEffect(() => {
    if (typeof value !== 'string') setDraft((d) => { try { return JSON.stringify(JSON.parse(d)) === JSON.stringify(value) ? d : JSON.stringify(value ?? null, null, 2); } catch { return JSON.stringify(value ?? null, null, 2); } });
  }, [value]);
  return (
    <TextField size="small" fullWidth multiline minRows={2} {...rest} value={draft}
      onChange={(e) => { setDraft(e.target.value); try { onChange(JSON.parse(e.target.value)); } catch { onChange(e.target.value); } }} />
  );
}

function Field({ f, value, onChange, scope }: { f: FormFieldSpec; value: unknown; onChange: (v: unknown) => void; scope: Scope }) {
  const body = <FieldBody f={f} value={value} onChange={onChange} scope={scope} />;
  return f.wide ? <Box sx={{ gridColumn: '1 / -1' }}>{body}</Box> : body;
}

/** Kinds that hold no value (never seeded, never required). */
export const ACTION_KINDS: FormFieldSpec['kind'][] = ['note', 'button', 'upload'];

function FieldBody({ f, value, onChange, scope }: { f: FormFieldSpec; value: unknown; onChange: (v: unknown) => void; scope: Scope }) {
  const visible = useCondition(f.visibleWhen, scope, true);
  const readOnly = useCondition(f.readOnlyWhen, scope, false);
  if (f.visibleWhen && !visible) return null;
  const disabled = !!f.readOnly || (!!f.readOnlyWhen && readOnly);
  const label = text(f.label, scope);
  const helper = f.helperText ? text(f.helperText, scope) || undefined : undefined;
  // Values show as text whatever they are stored as (a policy's 2 selects option "2").
  const shown = value === undefined || value === null ? '' : String(value);
  const choose = (v: string) => onChange(f.valueType === 'number' && v !== '' ? Number(v) : v);
  switch (f.kind) {
    case 'radio':
      return (
        <Stack>
          {label && <FormLabel>{label}</FormLabel>}
          <RadioGroup value={shown} onChange={(e) => choose(e.target.value)}>
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
        <TextField select size="small" fullWidth label={label} value={shown} disabled={disabled} required={f.required} helperText={helper}
          onChange={(e) => choose(e.target.value)}>
          {!f.required && <MenuItem value=""><em>—</em></MenuItem>}
          {fieldOptions(f, scope).map((o) => (
            <MenuItem key={o.value} value={o.value}>
              {o.caption ? (
                <Box>
                  <Typography variant="body2">{o.label}</Typography>
                  <Typography variant="caption" color="text.secondary">{o.caption}</Typography>
                </Box>
              ) : o.label}
            </MenuItem>
          ))}
        </TextField>
      );
    case 'rows':
      return <RowsField f={f} value={value} onChange={onChange} disabled={disabled} label={label} scope={scope} />;
    case 'note':
      return f.severity
        ? <Alert severity={f.severity}>{label}</Alert>
        : <Typography variant="body2" color="text.secondary" sx={{ whiteSpace: 'pre-line' }}>{label}</Typography>;
    case 'button':
      return <FormButton f={f} scope={scope} disabled={disabled} label={label} />;
    case 'upload':
      return <UploadButton f={f} scope={scope} disabled={disabled} label={label} />;
    case 'checklist':
      return <Checklist f={f} value={value} onChange={onChange} disabled={disabled} label={label} scope={scope} />;
    case 'json':
      return <JsonText value={value} onChange={onChange} label={label} disabled={disabled} required={f.required} helperText={helper} />;
    case 'map':
      return f.map ? <MapField spec={f.map} value={value} onChange={onChange} disabled={disabled} options={fieldOptions(f, scope)} scope={scope} /> : null;
    case 'number':
      return (
        <TextField size="small" fullWidth type="number" label={label} value={value ?? ''} disabled={disabled} required={f.required} helperText={helper}
          inputProps={{ step: f.step ?? 'any' }}
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
