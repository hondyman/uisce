import React, { useEffect, useState } from 'react';
import {
  Autocomplete, Box, Button, Divider, IconButton, MenuItem, Paper, Stack, Switch, FormControlLabel, TextField, ToggleButton,
  ToggleButtonGroup, Tooltip, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/DeleteOutline';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import type { CorePageDefinition } from '../../../types/pageStudio';
import { getOperation, listOperations } from '../../../studio-core/operations/registry';
import type { Action, ConditionNode, FormSpec, TextSpec } from './appModel';

type RunOperation = Extract<Action, { kind: 'runOperation' }>;

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
    <Autocomplete freeSolo size="small" options={options} value={str} inputValue={str}
      onInputChange={(_, v, reason) => { if (reason !== 'reset') onChange(v); }}
      onChange={(_, v) => onChange(typeof v === 'string' ? v : '')}
      filterOptions={(opts, s) => {
        // Suggest while typing inside {{ ... }}.
        const m = /\{\{\s*([\w.]*)$/.exec(s.inputValue);
        return m ? opts.filter((o) => o.includes(m[1])) : s.inputValue ? opts.filter((o) => o.includes(s.inputValue)) : opts;
      }}
      renderInput={(params) => <TextField {...params} label={label} helperText={helperText} multiline={multiline} />} />
  );
}

/** Display text: a literal, an i18n key, or a template. */
export function TextSpecField({ label, value, onChange, paths }: { label: string; value: TextSpec | undefined; onChange: (v: TextSpec) => void; paths: string[] }) {
  if (value && typeof value === 'object') {
    return <JsonField label={`${label} (i18n key + params)`} value={value} onChange={(v) => onChange(v as TextSpec)} />;
  }
  return <BindingField label={label} value={value} onChange={(v) => onChange(String(v ?? ''))} paths={paths} helperText="Text, an i18n key (mastering.title) or {{template}}" />;
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

/** Operators the rule engine implements (backend/internal/rules/vm) that make sense for page state. */
const OPERATORS: { value: string; label: string; needsValue: boolean }[] = [
  { value: 'equals', label: 'equals', needsValue: true },
  { value: 'not_equals', label: 'does not equal', needsValue: true },
  { value: 'in', label: 'is one of', needsValue: true },
  { value: 'not_in', label: 'is not one of', needsValue: true },
  { value: 'is_empty', label: 'is empty', needsValue: false },
  { value: 'is_not_empty', label: 'is not empty', needsValue: false },
  { value: 'is_true', label: 'is true', needsValue: false },
  { value: 'is_false', label: 'is false', needsValue: false },
  { value: 'greater_than', label: '>', needsValue: true },
  { value: 'less_than', label: '<', needsValue: true },
  { value: 'contains', label: 'contains', needsValue: true },
];

type Leaf = Extract<ConditionNode, { type: 'condition' }>;

const parseValue = (s: string, op: string): unknown => {
  if (op === 'in' || op === 'not_in') return s.split(',').map((x) => x.trim());
  if (s === 'true') return true;
  if (s === 'false') return false;
  if (s !== '' && !Number.isNaN(Number(s))) return Number(s);
  return s;
};
const showValue = (v: unknown) => (Array.isArray(v) ? v.join(', ') : v === undefined || v === null ? '' : String(v));

function LeafEditor({ leaf, onChange, paths }: { leaf: Leaf; onChange: (l: Leaf) => void; paths: string[] }) {
  const op = OPERATORS.find((o) => o.value === leaf.operator);
  return (
    <Stack spacing={1}>
      <Autocomplete freeSolo size="small" options={paths} value={leaf.field} inputValue={leaf.field}
        onInputChange={(_, v) => onChange({ ...leaf, field: v })} renderInput={(p) => <TextField {...p} label="Field (scope path)" />} />
      <Stack direction="row" spacing={1}>
        <TextField select size="small" label="Operator" value={leaf.operator} sx={{ minWidth: 140 }}
          onChange={(e) => onChange({ ...leaf, operator: e.target.value })}>
          {OPERATORS.map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
        </TextField>
        {op?.needsValue !== false && (
          <TextField size="small" label="Value" value={showValue(leaf.value)} fullWidth
            helperText={leaf.operator === 'in' || leaf.operator === 'not_in' ? 'Comma-separated' : undefined}
            onChange={(e) => onChange({ ...leaf, value: parseValue(e.target.value, leaf.operator) })} />
        )}
      </Stack>
    </Stack>
  );
}

/**
 * Edits a condition as the rule engine's own RuleNode JSON: one condition,
 * or an AND/OR group of conditions. Nothing here evaluates anything.
 */
export function ConditionEditor({ label, value, onChange, paths }: {
  label: string; value: ConditionNode | undefined; onChange: (v: ConditionNode | undefined) => void; paths: string[];
}) {
  const leaves: Leaf[] = !value ? [] : value.type === 'condition' ? [value] : value.conditions.filter((c): c is Leaf => c.type === 'condition');
  const nested = value?.type === 'group' && value.conditions.some((c) => c.type === 'group');
  const joiner = value?.type === 'group' ? value.operator : 'AND';
  const emit = (ls: Leaf[], op: 'AND' | 'OR' = joiner) =>
    onChange(ls.length === 0 ? undefined : ls.length === 1 ? ls[0] : { type: 'group', operator: op, conditions: ls });
  if (nested) return <JsonField label={`${label} (nested groups)`} value={value} onChange={(v) => onChange(v as ConditionNode)} />;
  return (
    <Box>
      <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
        <Typography variant="caption" color="text.secondary" sx={{ flex: 1 }}>{label}{leaves.length === 0 ? ': always' : ''}</Typography>
        {leaves.length > 1 && (
          <ToggleButtonGroup size="small" exclusive value={joiner} onChange={(_, v) => v && emit(leaves, v)}>
            <ToggleButton value="AND" sx={{ py: 0 }}>all</ToggleButton>
            <ToggleButton value="OR" sx={{ py: 0 }}>any</ToggleButton>
          </ToggleButtonGroup>
        )}
      </Stack>
      <ListEditor items={leaves} onChange={(ls) => emit(ls)} addLabel="Add condition"
        create={(): Leaf => ({ type: 'condition', field: paths[0] ?? '', operator: 'is_not_empty' })}
        render={(l, update) => <LeafEditor leaf={l} onChange={update} paths={paths} />} />
    </Box>
  );
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
      <JsonField label="Ask first: form (optional)" value={a.form ?? null} minRows={2} onChange={(v) => onChange({ ...a, form: (v || undefined) as FormSpec | undefined })} />
      <JsonField label="Ask first: confirm (optional)" value={a.confirm ?? null} minRows={2} onChange={(v) => onChange({ ...a, confirm: (v || undefined) as RunOperation['confirm'] })} />
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
