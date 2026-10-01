import React, { useEffect, useState } from 'react';
import {
  Autocomplete, Box, Button, Divider, IconButton, MenuItem, Paper, Stack, Switch, FormControlLabel, TextField, Tooltip, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/DeleteOutline';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import type { CorePageDefinition } from '../../../types/pageStudio';
import { getOperation, listOperations } from '../../../studio-core/operations/registry';
import type { Action, ConditionNode, TextSpec } from './appModel';
import { BindingPicker } from './bindingPicker';
import { ConditionBuilder } from './conditionBuilder';
import { ConfirmEditor, FormSpecEditor } from './actionExtras';
import { TextKeyParamsEditor } from './structuredEditors';

/**
 * The editors every app-model surface shares: bindings with scope
 * suggestions, rule-engine conditions, action lists, and a JSON escape
 * hatch. Kept generic so a new widget's inspector is a few lines of
 * field declarations, not a new editor.
 */

// --- Scope suggestions -------------------------------------------------------

/** Paths an author can bind to on this page: variables, query results, and (inside a grid) row fields. */
export function scopePaths(draft: CorePageDefinition, rowQuery?: string): string[] {
  const out: string[] = [];
  for (const v of draft.app?.variables ?? []) out.push(`vars.${v.name}`);
  for (const q of draft.app?.queries ?? []) {
    out.push(`queries.${q.id}.data`);
    out.push(`queries.${q.id}.data.length`);
    for (const f of getOperation(q.operation)?.fields ?? []) out.push(`queries.${q.id}.data.${f.name}`);
  }
  if (rowQuery) {
    const q = (draft.app?.queries ?? []).find((x) => x.id === rowQuery);
    for (const f of (q && getOperation(q.operation)?.fields) ?? []) out.push(`row.${f.name}`);
    out.push('rowState.comment', 'index');
  }
  return out;
}

// --- Primitive fields ---------------------------------------------------------

export function Section({ title, children, action }: { title: string; children: React.ReactNode; action?: React.ReactNode }) {
  return (
    <Box sx={{ mt: 2 }}>
      <Stack direction="row" alignItems="center" sx={{ mb: 1 }}>
        <Typography variant="overline" color="text.secondary" fontWeight="bold" sx={{ flex: 1 }}>{title}</Typography>
        {action}
      </Stack>
      <Stack spacing={1.25}>{children}</Stack>
    </Box>
  );
}

/** A value or {{template}}; suggestions insert scope paths. */
export function BindingField({ label, value, onChange, paths, helperText, multiline }: {
  label: string; value: unknown; onChange: (v: unknown) => void; paths: string[]; helperText?: string; multiline?: boolean;
}) {
  const str = value === undefined || value === null ? '' : typeof value === 'string' ? value : JSON.stringify(value);
  const options = paths.map((p) => `{{${p}}}`);
  return (
    <Stack direction="row" spacing={0.5} alignItems="flex-start">
      <Autocomplete freeSolo size="small" sx={{ flex: 1, minWidth: 0 }} options={options} value={str} inputValue={str}
        onInputChange={(_, v, reason) => { if (reason !== 'reset') onChange(v); }}
        onChange={(_, v) => onChange(typeof v === 'string' ? v : '')}
        filterOptions={(opts, s) => {
          // Suggest while typing inside {{ ... }}.
          const m = /\{\{\s*([\w.]*)$/.exec(s.inputValue);
          return m ? opts.filter((o) => o.includes(m[1])) : s.inputValue ? opts.filter((o) => o.includes(s.inputValue)) : opts;
        }}
        renderInput={(params) => <TextField {...params} label={label} helperText={helperText} multiline={multiline} />} />
      {/* Browse the page's live data; the pick goes in as {{path}} (after any text already there). */}
      <BindingPicker contextPaths={paths} label={`Browse data for ${label}`} onPick={(path) => onChange(`${str}{{${path}}}`)} />
    </Stack>
  );
}

/** Display text: a literal, an i18n key, or a template. */
export function TextSpecField({ label, value, onChange, paths }: { label: string; value: TextSpec | undefined; onChange: (v: TextSpec) => void; paths: string[] }) {
  if (value && typeof value === 'object') {
    return <TextKeyParamsEditor label={label} value={value as { t: string; params?: Record<string, unknown> }} onChange={onChange} paths={paths} />;
  }
  const key = typeof value === 'string' && /^[a-z][A-Za-z0-9_]*(\.[A-Za-z0-9_]+)+$/.test(value);
  return (
    <>
      <BindingField label={label} value={value} onChange={(v) => onChange(String(v ?? ''))} paths={paths} helperText="Text, an i18n key (mastering.title) or {{template}}" />
      {/* A key that needs values filled in (a count, a name) becomes {t, params}. */}
      {key && <Button size="small" sx={{ alignSelf: 'flex-start', mt: -0.5 }} onClick={() => onChange({ t: value as string, params: {} })}>Fill in values</Button>}
    </>
  );
}

/** Raw JSON editing for anything the structured editors do not cover yet. Commits on valid JSON. */
export function JsonField({ label, value, onChange, minRows = 3 }: { label: string; value: unknown; onChange: (v: unknown) => void; minRows?: number }) {
  const [draft, setDraft] = useState(() => JSON.stringify(value ?? null, null, 2));
  const [error, setError] = useState<string | null>(null);
  useEffect(() => { setDraft(JSON.stringify(value ?? null, null, 2)); }, [value]);
  return (
    <TextField label={label} value={draft} multiline minRows={minRows} maxRows={24} fullWidth error={!!error}
      helperText={error ?? 'JSON - applied when valid'} InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12 } }}
      onChange={(e) => {
        setDraft(e.target.value);
        try { const v = JSON.parse(e.target.value); setError(null); onChange(v); } catch (err) { setError((err as Error).message); }
      }} />
  );
}

export function SelectField<T extends string>({ label, value, options, onChange, allowEmpty }: {
  label: string; value: T | undefined; options: { value: T; label: string }[]; onChange: (v: T | undefined) => void; allowEmpty?: string;
}) {
  return (
    <TextField select size="small" label={label} value={value ?? ''} onChange={(e) => onChange((e.target.value || undefined) as T | undefined)}
      SelectProps={{ displayEmpty: !!allowEmpty }} InputLabelProps={allowEmpty ? { shrink: true } : undefined}>
      {allowEmpty && <MenuItem value=""><em>{allowEmpty}</em></MenuItem>}
      {options.map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
    </TextField>
  );
}

export function SwitchField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return <FormControlLabel control={<Switch size="small" checked={checked} onChange={(e) => onChange(e.target.checked)} />} label={<Typography variant="body2">{label}</Typography>} />;
}

/** Ordered list editing: add, remove, move. */
export function ListEditor<T>({ items, onChange, render, create, addLabel }: {
  items: T[]; onChange: (items: T[]) => void; render: (item: T, update: (next: T) => void, index: number) => React.ReactNode; create: () => T; addLabel: string;
}) {
  const move = (i: number, d: number) => {
    const next = [...items];
    const [x] = next.splice(i, 1);
    next.splice(i + d, 0, x);
    onChange(next);
  };
  return (
    <Stack spacing={1}>
      {items.map((item, i) => (
        <Paper key={i} variant="outlined" sx={{ p: 1.25 }}>
          <Stack direction="row" justifyContent="flex-end" sx={{ mt: -0.5, mb: 0.5 }}>
            <IconButton size="small" disabled={i === 0} onClick={() => move(i, -1)}><ArrowUpwardIcon fontSize="inherit" /></IconButton>
            <IconButton size="small" disabled={i === items.length - 1} onClick={() => move(i, 1)}><ArrowDownwardIcon fontSize="inherit" /></IconButton>
            <IconButton size="small" onClick={() => onChange(items.filter((_, j) => j !== i))}><DeleteIcon fontSize="inherit" /></IconButton>
          </Stack>
          <Stack spacing={1.25}>{render(item, (next) => onChange(items.map((x, j) => (j === i ? next : x))), i)}</Stack>
        </Paper>
      ))}
      <Button size="small" startIcon={<AddIcon />} onClick={() => onChange([...items, create()])} sx={{ alignSelf: 'flex-start' }}>{addLabel}</Button>
    </Stack>
  );
}

// --- Conditions (rule-engine RuleNode JSON) --------------------------------------

/**
 * Edits a condition as a sentence: field, operator and value, nested in
 * all/any groups (conditionBuilder.tsx). It stores the rule engine's own
 * RuleNode JSON; nothing here evaluates anything.
 */
export function ConditionEditor({ label, value, onChange, paths }: {
  label: string; value: ConditionNode | undefined; onChange: (v: ConditionNode | undefined) => void; paths: string[];
}) {
  return <ConditionBuilder label={label} value={value} onChange={onChange} paths={paths} />;
}

// --- Actions -----------------------------------------------------------------

const ACTION_KINDS: { value: Action['kind']; label: string }[] = [
  { value: 'setVariable', label: 'Set a variable' },
  { value: 'runOperation', label: 'Run an operation' },
  { value: 'navigate', label: 'Go to a page' },
  { value: 'notify', label: 'Show a message' },
];

function newAction(kind: Action['kind']): Action {
  switch (kind) {
    case 'setVariable': return { kind, name: '', value: '' };
    case 'runOperation': return { kind, operation: '', params: {} };
    case 'navigate': return { kind, to: '' };
    default: return { kind: 'notify', severity: 'success', text: '' };
  }
}

function ActionEditor({ a, onChange, draft, paths }: { a: Action; onChange: (a: Action) => void; draft: CorePageDefinition; paths: string[] }) {
  const kind = (
    <TextField select size="small" label="Do" value={a.kind} onChange={(e) => onChange(newAction(e.target.value as Action['kind']))}>
      {ACTION_KINDS.map((k) => <MenuItem key={k.value} value={k.value}>{k.label}</MenuItem>)}
    </TextField>
  );
  if (a.kind === 'setVariable') {
    return (
      <>
        {kind}
        <SelectField label="Variable" value={a.name} options={(draft.app?.variables ?? []).map((v) => ({ value: v.name, label: v.name }))} onChange={(v) => onChange({ ...a, name: v ?? '' })} />
        <BindingField label="To" value={a.value} onChange={(v) => onChange({ ...a, value: v })} paths={[...paths, 'value', 'event', 'result']} helperText="Empty sets it to nothing (closes a drawer)" />
      </>
    );
  }
  if (a.kind === 'navigate') {
    return <>{kind}<BindingField label="Path" value={a.to} onChange={(v) => onChange({ ...a, to: v })} paths={paths} /></>;
  }
  if (a.kind === 'notify') {
    return (
      <>
        {kind}
        <SelectField label="Severity" value={a.severity} options={['success', 'info', 'warning', 'error'].map((s) => ({ value: s as 'info', label: s }))} onChange={(v) => onChange({ ...a, severity: v ?? 'info' })} />
        <TextSpecField label="Message" value={a.text} onChange={(v) => onChange({ ...a, text: v })} paths={paths} />
      </>
    );
  }
  const op = getOperation(a.operation);
  return (
    <>
      {kind}
      <SelectField label="Operation" value={a.operation} options={listOperations('mutation').map((o) => ({ value: o.id, label: `${o.label} (${o.id})` }))}
        onChange={(v) => onChange({ ...a, operation: v ?? '' })} />
      {op?.description && <Typography variant="caption" color="text.secondary">{op.description}</Typography>}
      {op?.params.map((p) => (
        <BindingField key={p.name} label={`${p.label ?? p.name}${p.required ? ' *' : ''}`} value={a.params?.[p.name]} paths={[...paths, 'form.note']}
          onChange={(v) => onChange({ ...a, params: { ...a.params, [p.name]: v } })} helperText={p.description} />
      ))}
      <TextSpecField label="Success message" value={a.successMessage} onChange={(v) => onChange({ ...a, successMessage: v || undefined })} paths={[...paths, 'result.message']} />
      <SelectField label="Progress into variable (optional)" value={a.progressVariable} allowEmpty="None"
        options={(draft.app?.variables ?? []).map((v) => ({ value: v.name, label: v.name }))} onChange={(v) => onChange({ ...a, progressVariable: v || undefined })} />
      <ConfirmEditor value={a.confirm} onChange={(v) => onChange({ ...a, confirm: v })} paths={paths} />
      <FormSpecEditor value={a.form} onChange={(v) => onChange({ ...a, form: v })} draft={draft} paths={paths} />
      <Divider textAlign="left"><Typography variant="caption">Then</Typography></Divider>
      <ActionsEditor label="On success" value={a.onSuccess} onChange={(v) => onChange({ ...a, onSuccess: v })} draft={draft} paths={[...paths, 'result']} />
    </>
  );
}

export function ActionsEditor({ label, value, onChange, draft, paths }: {
  label: string; value: Action[] | undefined; onChange: (v: Action[] | undefined) => void; draft: CorePageDefinition; paths: string[];
}) {
  return (
    <Box>
      <Tooltip title="Actions run in order; an operation's result is {{result}} for the actions after it.">
        <Typography variant="caption" color="text.secondary">{label}</Typography>
      </Tooltip>
      <ListEditor items={value ?? []} onChange={(v) => onChange(v.length ? v : undefined)} addLabel="Add action" create={() => newAction('setVariable')}
        render={(a, update) => <ActionEditor a={a} onChange={update} draft={draft} paths={paths} />} />
    </Box>
  );
}
