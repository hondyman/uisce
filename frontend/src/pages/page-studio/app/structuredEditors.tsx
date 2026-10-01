import React, { useEffect, useRef, useState } from 'react';
import { Box, Button, IconButton, MenuItem, Stack, TextField, Typography } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/DeleteOutline';
import type { CorePageDefinition } from '../../../types/pageStudio';
import type { Binding, ConditionNode, MapFieldSpec, OptionsFrom, RowButton, RowFieldSpec, TextSpec } from './appModel';
import type { CanvasCategory } from './canvas';
import { PAGE_ICONS } from './icons';
import {
  ActionsEditor, BindingField, ConditionEditor, ListEditor, SelectField, SwitchField, TextSpecField,
} from './editors';

/**
 * Structured editors for the settings that used to be raw JSON: key -> value
 * tables (colour maps, variable filters, text params), lists of things with
 * their own fields (row buttons, options, palette groups, mapping groups,
 * row fields, canvas categories). Each edits the same stored shape the JSON
 * did, so nothing already saved changes; they only make it a form.
 */

export const COLORS = ['default', 'primary', 'secondary', 'success', 'warning', 'error', 'info'];
export const colorOptions = COLORS.map((c) => ({ value: c, label: c }));

// --- key -> value tables ------------------------------------------------------------------------------

/**
 * Rows of [key, value] held locally, so a half-typed or blank key does not
 * vanish while editing; every change applies to the rows and emits the object
 * they make (blank keys left out). The rows reset only when the stored object
 * changes from somewhere else.
 */
function useKeyedRows<T>(value: Record<string, T> | undefined, fresh: () => T, onChange: (v: Record<string, T> | undefined) => void): {
  rows: [string, T][]; setRow: (i: number, row: [string, T]) => void; add: () => void; remove: (i: number) => void;
} {
  const toRows = (v: Record<string, T> | undefined): [string, T][] => Object.entries(v ?? {});
  const [rows, setRows] = useState<[string, T][]>(() => toRows(value));
  const emitted = useRef(JSON.stringify(value ?? {}));
  const external = JSON.stringify(value ?? {});
  useEffect(() => {
    if (external !== emitted.current) { emitted.current = external; setRows(toRows(value)); }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [external]);
  const apply = (next: [string, T][]) => {
    setRows(next);
    const obj = objectOf(next);
    emitted.current = JSON.stringify(obj ?? {});
    onChange(obj);
  };
  return {
    rows,
    setRow: (i, row) => apply(rows.map((r, j) => (j === i ? row : r))),
    add: () => apply([...rows, ['', fresh()]]),
    remove: (i) => apply(rows.filter((_, j) => j !== i)),
  };
}

/** The object a set of rows stands for: blank keys left out, a repeated key keeps its last value. */
export const objectOf = <T,>(rows: [string, T][]): Record<string, T> | undefined => {
  const out: Record<string, T> = {};
  for (const [k, v] of rows) if (k.trim() !== '') out[k] = v;
  return Object.keys(out).length ? out : undefined;
};

/** A table of key -> value: a colour map (value -> colour), variable filters (row field -> variable), text params. */
export function StringMapEditor({ label, value, onChange, keyLabel = 'When it is', valueLabel = 'Use', valueOptions, keyHelp, addLabel = 'Add row' }: {
  label: string; value: Record<string, string> | undefined; onChange: (v: Record<string, string> | undefined) => void;
  keyLabel?: string; valueLabel?: string; valueOptions?: { value: string; label: string }[]; keyHelp?: string; addLabel?: string;
}) {
  const { rows, setRow, add, remove } = useKeyedRows<string>(value, () => valueOptions?.[0]?.value ?? '', onChange);
  return (
    <Box>
      <Typography variant="caption" color="text.secondary">{label}</Typography>
      <Stack spacing={1} sx={{ mt: 0.5 }}>
        {rows.map(([k, v], i) => (
          <Stack key={i} direction="row" spacing={1} alignItems="flex-start">
            <TextField size="small" label={keyLabel} value={k} sx={{ flex: 1, minWidth: 0 }} helperText={i === 0 ? keyHelp : undefined}
              onChange={(e) => setRow(i, [e.target.value, v])} />
            {valueOptions ? (
              <TextField select size="small" label={valueLabel} value={v} sx={{ flex: 1, minWidth: 0 }} onChange={(e) => setRow(i, [k, e.target.value])}>
                {valueOptions.map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
              </TextField>
            ) : (
              <TextField size="small" label={valueLabel} value={v} sx={{ flex: 1, minWidth: 0 }} onChange={(e) => setRow(i, [k, e.target.value])} />
            )}
            <IconButton size="small" aria-label={`Remove ${k || 'row'}`} onClick={() => remove(i)}><DeleteIcon fontSize="small" /></IconButton>
          </Stack>
        ))}
        <Button size="small" startIcon={<AddIcon />} onClick={add} sx={{ alignSelf: 'flex-start' }}>{addLabel}</Button>
      </Stack>
    </Box>
  );
}

/** A colour map: what each value looks like (`*` = anything else). */
export const ColorMapEditor = ({ label = 'Colours', value, onChange }: { label?: string; value: Record<string, string> | undefined; onChange: (v: Record<string, string> | undefined) => void }) => (
  <StringMapEditor label={label} value={value} onChange={onChange} keyLabel="When the value is" valueLabel="Colour" valueOptions={colorOptions}
    keyHelp="Use * for any other value" addLabel="Add colour" />
);

// --- simple lists ----------------------------------------------------------------------------------------------

/** Words separated by commas (keys in order, search fields). */
export function WordsField({ label, value, onChange, helperText }: { label: string; value: string[] | undefined; onChange: (v: string[] | undefined) => void; helperText?: string }) {
  return (
    <TextField size="small" label={label} value={(value ?? []).join(', ')} helperText={helperText ?? 'Separated by commas'}
      onChange={(e) => { const v = e.target.value.split(',').map((x) => x.trim()).filter(Boolean); onChange(v.length ? v : undefined); }} />
  );
}

/** A list of texts (starter prompts), or one binding that supplies the list. */
export function TextListEditor({ label, value, onChange, paths, addLabel = 'Add' }: {
  label: string; value: TextSpec[] | Binding | undefined; onChange: (v: TextSpec[] | Binding | undefined) => void; paths: string[]; addLabel?: string;
}) {
  const bound = typeof value === 'string';
  return (
    <Box>
      <Stack direction="row" alignItems="center">
        <Typography variant="caption" color="text.secondary" sx={{ flex: 1 }}>{label}</Typography>
        <Button size="small" onClick={() => onChange(bound ? [] : '')}>{bound ? 'Use a list' : 'Use a binding'}</Button>
      </Stack>
      {bound
        ? <BindingField label="List from" value={value} onChange={(v) => onChange(v as Binding)} paths={paths} />
        : <ListEditor items={(value as TextSpec[] | undefined) ?? []} onChange={(v) => onChange(v.length ? v : undefined)} addLabel={addLabel} create={() => '' as TextSpec}
          render={(t, up) => <TextSpecField label="Text" value={t} onChange={up} paths={paths} />} />}
    </Box>
  );
}

/** An i18n key with params ({t, params}): the key, and a table of param -> value. */
export function TextKeyParamsEditor({ label, value, onChange, paths }: {
  label: string; value: { t: string; params?: Record<string, unknown> }; onChange: (v: TextSpec) => void; paths: string[];
}) {
  const params = Object.fromEntries(Object.entries(value.params ?? {}).map(([k, v]) => [k, typeof v === 'string' ? v : JSON.stringify(v)]));
  return (
    <Stack spacing={1}>
      <Stack direction="row" spacing={1} alignItems="center">
        <TextField size="small" label={`${label} (i18n key)`} value={value.t} sx={{ flex: 1 }} onChange={(e) => onChange({ ...value, t: e.target.value })} />
        <Button size="small" onClick={() => onChange(value.t)}>Plain text</Button>
      </Stack>
      <StringMapEditor label="Values filled in" value={params} keyLabel="Name" valueLabel="Value ({{…}} allowed)"
        onChange={(v) => onChange({ ...value, params: v })} addLabel="Add value" />
      <Typography variant="caption" color="text.secondary">Available: {paths.slice(0, 3).join(', ')}{paths.length > 3 ? '…' : ''}</Typography>
    </Stack>
  );
}

/** Options of a select/toggle: value, label and an optional colour. */
type Option = { value: string; label: TextSpec; color?: string };
export function OptionListEditor({ value, onChange, paths, withColor }: {
  value: Option[] | undefined; onChange: (v: Option[]) => void; paths: string[]; withColor?: boolean;
}) {
  return (
    <ListEditor items={value ?? []} onChange={onChange} addLabel="Add option" create={(): Option => ({ value: '', label: '' as TextSpec })}
      render={(o, up) => (
        <>
          <TextField size="small" label="Value" value={o.value} onChange={(e) => up({ ...o, value: e.target.value })} />
          <TextSpecField label="Label" value={o.label} onChange={(v) => up({ ...o, label: v })} paths={paths} />
          {withColor && <SelectField label="Colour" value={o.color} allowEmpty="None" options={colorOptions} onChange={(v) => up({ ...o, color: v })} />}
        </>
      )} />
  );
}

// --- row buttons -----------------------------------------------------------------------------------------------------

/** A grid cell's buttons: label, icon, style, colour, when to show, and what a click does. */
export function RowButtonsEditor({ value, onChange, draft, paths }: {
  value: RowButton[] | undefined; onChange: (v: RowButton[]) => void; draft: CorePageDefinition; paths: string[];
}) {
  return (
    <ListEditor items={value ?? []} onChange={onChange} addLabel="Add button" create={(): RowButton => ({ label: 'Button' as TextSpec, onClick: [] })}
      render={(b, up) => (
        <>
          <TextSpecField label="Label" value={b.label} onChange={(v) => up({ ...b, label: v })} paths={paths} />
          <Stack direction="row" spacing={1}>
            <SelectField label="Icon (label becomes its tooltip)" value={b.icon} allowEmpty="None" options={Object.keys(PAGE_ICONS).map((k) => ({ value: k, label: k }))} onChange={(v) => up({ ...b, icon: v })} />
            <SelectField label="Style" value={b.variant ?? 'text'} options={['text', 'outlined', 'contained'].map((x) => ({ value: x as 'text', label: x }))} onChange={(v) => up({ ...b, variant: v })} />
            <SelectField label="Colour" value={b.color ?? 'primary'} options={['primary', 'secondary', 'inherit', 'error'].map((x) => ({ value: x as 'primary', label: x }))} onChange={(v) => up({ ...b, color: v })} />
          </Stack>
          <ConditionEditor label="Show when" value={b.visibleWhen} onChange={(v) => up({ ...b, visibleWhen: v })} paths={paths} />
          <ActionsEditor label="On click" value={b.onClick} onChange={(v) => up({ ...b, onClick: v ?? [] })} draft={draft} paths={paths} />
        </>
      )} />
  );
}

/** A small line under a cell's buttons (e.g. "You voted"), shown conditionally. */
export function CaptionEditor({ value, onChange, paths }: {
  value: { text: TextSpec; visibleWhen?: ConditionNode } | undefined; onChange: (v: { text: TextSpec; visibleWhen?: ConditionNode } | undefined) => void; paths: string[];
}) {
  return (
    <Box>
      <SwitchField label="A caption under the buttons" checked={!!value} onChange={(on) => onChange(on ? { text: '' as TextSpec } : undefined)} />
      {value && (
        <Stack spacing={1} sx={{ mt: 1 }}>
          <TextSpecField label="Caption" value={value.text} onChange={(v) => onChange({ ...value, text: v })} paths={paths} />
          <ConditionEditor label="Show when" value={value.visibleWhen} onChange={(v) => onChange({ ...value, visibleWhen: v })} paths={paths} />
        </Stack>
      )}
    </Box>
  );
}

// --- canvas ---------------------------------------------------------------------------------------------------------------

/** Canvas node categories: a name, its colour, and whether nodes take an input and give an output. */
export function CanvasCategoriesEditor({ value, onChange }: { value: Record<string, CanvasCategory> | undefined; onChange: (v: Record<string, CanvasCategory> | undefined) => void }) {
  const { rows, setRow, add, remove } = useKeyedRows<CanvasCategory>(value, () => ({ color: 'primary' }), onChange);
  return (
    <Box>
      <Typography variant="caption" color="text.secondary">Categories (a node's category picks its colour and handles)</Typography>
      <Stack spacing={1} sx={{ mt: 0.5 }}>
        {rows.map(([name, c], i) => (
          <Box key={i} sx={{ border: 1, borderColor: 'divider', borderRadius: 1, p: 1 }}>
            <Stack direction="row" spacing={1} alignItems="flex-start">
              <TextField size="small" label="Category" value={name} sx={{ flex: 1 }} onChange={(e) => setRow(i, [e.target.value, c])} />
              <SelectField label="Colour" value={c.color ?? 'primary'} options={colorOptions as { value: NonNullable<CanvasCategory['color']>; label: string }[]} onChange={(v) => setRow(i, [name, { ...c, color: v }])} />
              <IconButton size="small" aria-label={`Remove ${name || 'category'}`} onClick={() => remove(i)}><DeleteIcon fontSize="small" /></IconButton>
            </Stack>
            <Stack direction="row" spacing={2}>
              <SwitchField label="Takes an input" checked={c.inputs !== false} onChange={(on) => setRow(i, [name, { ...c, inputs: on ? undefined : false }])} />
              <SwitchField label="Gives an output" checked={c.outputs !== false} onChange={(on) => setRow(i, [name, { ...c, outputs: on ? undefined : false }])} />
            </Stack>
          </Box>
        ))}
        <Button size="small" startIcon={<AddIcon />} onClick={add} sx={{ alignSelf: 'flex-start' }}>Add category</Button>
      </Stack>
    </Box>
  );
}

/** The palette's groups, in order, each with its title. */
export function PaletteGroupsEditor({ value, onChange, paths }: {
  value: { id: string; label: TextSpec }[] | undefined; onChange: (v: { id: string; label: TextSpec }[] | undefined) => void; paths: string[];
}) {
  return (
    <ListEditor items={value ?? []} onChange={(v) => onChange(v.length ? v : undefined)} addLabel="Add group" create={() => ({ id: '', label: '' as TextSpec })}
      render={(g, up) => (
        <>
          <TextField size="small" label="Group value (the field it groups by)" value={g.id} onChange={(e) => up({ ...g, id: e.target.value })} />
          <TextSpecField label="Title" value={g.label} onChange={(v) => up({ ...g, label: v })} paths={paths} />
        </>
      )} />
  );
}

// --- where options come from -------------------------------------------------------------------------------------------

/** "Options from a query": which query, where its rows are, which fields hold value / label / caption. */
export function OptionsFromEditor({ value, onChange, queries, captions }: {
  value: OptionsFrom; onChange: (v: OptionsFrom) => void; queries: { value: string; label: string }[]; captions?: boolean;
}) {
  const field = (key: keyof OptionsFrom, label: string) => (
    <TextField size="small" label={label} value={(value[key] as string | undefined) ?? ''} onChange={(e) => onChange({ ...value, [key]: e.target.value || undefined })} />
  );
  return (
    <Stack spacing={1}>
      <SelectField label="Query" value={value.query} options={queries} onChange={(v) => onChange({ ...value, query: v ?? '' })} />
      {field('rowsPath', 'Rows path (optional)')}
      <Stack direction="row" spacing={1}>{field('valueField', 'Value field')}{field('labelField', 'Label field')}</Stack>
      {captions && field('captionField', 'Caption field (optional)')}
    </Stack>
  );
}

// --- row fields (a list of rows inside a form) ---------------------------------------------------------------

const ROW_KINDS: { value: NonNullable<RowFieldSpec['kind']>; label: string }[] = [
  { value: 'text', label: 'Text' }, { value: 'number', label: 'Number' }, { value: 'select', label: 'Choice' },
  { value: 'switch', label: 'On / off' }, { value: 'chips', label: 'List of values' },
];

/** The columns of a "list of rows" form field. */
export function RowFieldsEditor({ value, onChange, paths, queries }: {
  value: RowFieldSpec[] | undefined; onChange: (v: RowFieldSpec[] | undefined) => void; paths: string[]; queries: { value: string; label: string }[];
}) {
  return (
    <ListEditor items={value ?? []} onChange={(v) => onChange(v.length ? v : undefined)} addLabel="Add column" create={(): RowFieldSpec => ({ name: '', label: '' as TextSpec })}
      render={(f, up) => {
        const choice = f.kind === 'select' || f.kind === 'chips';
        return (
          <>
            <Stack direction="row" spacing={1}>
              <TextField size="small" label="Name" value={f.name} onChange={(e) => up({ ...f, name: e.target.value })} helperText="{{row.<name>}}" />
              <SelectField label="Kind" value={f.kind ?? 'text'} options={ROW_KINDS} onChange={(v) => up({ ...f, kind: v })} />
            </Stack>
            <TextSpecField label="Label" value={f.label} onChange={(v) => up({ ...f, label: v })} paths={paths} />
            <Stack direction="row" spacing={1} alignItems="center">
              <SwitchField label="Read-only" checked={!!f.readOnly} onChange={(on) => up({ ...f, readOnly: on || undefined })} />
              <TextField size="small" type="number" label="Width weight" value={f.flex ?? ''} sx={{ width: 120 }} onChange={(e) => up({ ...f, flex: e.target.value ? Number(e.target.value) : undefined })} />
            </Stack>
            <TextSpecField label="Placeholder" value={f.placeholder} onChange={(v) => up({ ...f, placeholder: v || undefined })} paths={paths} />
            <BindingField label="Caption under it" value={f.caption} onChange={(v) => up({ ...f, caption: v || undefined })} paths={[...paths, 'row']} />
            {choice && (
              <>
                <SwitchField label="Options from a query" checked={!!f.optionsFrom} onChange={(on) => up({ ...f, optionsFrom: on ? { query: queries[0]?.value ?? '' } : undefined, options: on ? undefined : f.options ?? [] })} />
                {f.optionsFrom
                  ? <OptionsFromEditor value={f.optionsFrom} onChange={(v) => up({ ...f, optionsFrom: v })} queries={queries} />
                  : <OptionListEditor value={f.options} onChange={(v) => up({ ...f, options: v })} paths={paths} />}
              </>
            )}
            <ConditionEditor label="Show when" value={f.visibleWhen} onChange={(v) => up({ ...f, visibleWhen: v })} paths={[...paths, 'row']} />
          </>
        );
      }} />
  );
}

// --- mapping table (map form field) -------------------------------------------------------------------------------

/** A mapping table: the rows, headers, groups (each with a title and an action), hints, and add controls. */
export function MapSpecEditor({ value, onChange, draft, paths }: {
  value: MapFieldSpec; onChange: (v: MapFieldSpec) => void; draft: CorePageDefinition; paths: string[];
}) {
  const set = (patch: Partial<MapFieldSpec>) => onChange({ ...value, ...patch });
  const groups = value.groups ?? [];
  const adds = value.add ?? [];
  const mapPaths = [...paths, 'map'];
  type Group = NonNullable<MapFieldSpec['groups']>[number];
  type Add = NonNullable<MapFieldSpec['add']>[number];
  const addOptions = (a: Add) => (a.options ?? []).join(', ');
  return (
    <Stack spacing={1.5}>
      <BindingField label="Rows (a list of {key, label, caption, group})" value={value.rows} onChange={(v) => set({ rows: v as Binding })} paths={paths}
        helperText="e.g. {{queries.fieldRows.data.rows}}" />
      <Stack direction="row" spacing={1}>
        <TextSpecField label="Key column header" value={value.keyHeader} onChange={(v) => set({ keyHeader: v || undefined })} paths={paths} />
        <TextSpecField label="Value column header" value={value.valueHeader} onChange={(v) => set({ valueHeader: v || undefined })} paths={paths} />
      </Stack>
      <TextSpecField label="Empty value shows" value={value.placeholder} onChange={(v) => set({ placeholder: v || undefined })} paths={paths} />
      <BindingField label="Hints (per key: {value, label, color, tooltip}) - a chip while the value is still the hint" value={value.hints} onChange={(v) => set({ hints: (v || undefined) as Binding | undefined })} paths={paths} />
      <TextField size="small" type="number" label="Max height (px)" value={value.maxHeight ?? ''} onChange={(e) => set({ maxHeight: e.target.value ? Number(e.target.value) : undefined })} />
      <Box>
        <Typography variant="caption" color="text.secondary">Groups (each its own table; rows pick one by their group)</Typography>
        <ListEditor items={groups} onChange={(v) => set({ groups: v.length ? v : undefined })} addLabel="Add group" create={(): Group => ({ id: '' })}
          render={(g, up) => (
            <>
              <TextField size="small" label="Group id" value={g.id} onChange={(e) => up({ ...g, id: e.target.value })} />
              <TextSpecField label="Title ({{map.bound}} of {{map.total}} available)" value={g.title} onChange={(v) => up({ ...g, title: v || undefined })} paths={mapPaths} />
              <TextSpecField label="Help text" value={g.help} onChange={(v) => up({ ...g, help: v || undefined })} paths={mapPaths} />
              <SwitchField label="Show column headers" checked={!!g.headers} onChange={(on) => up({ ...g, headers: on || undefined })} />
              <SwitchField label="A button in the group's title bar" checked={!!g.action} onChange={(on) => up({ ...g, action: on ? { label: 'Action' as TextSpec, onClick: [] } : undefined })} />
              {g.action && (
                <>
                  <TextSpecField label="Button label" value={g.action.label} onChange={(v) => up({ ...g, action: { ...g.action!, label: v } })} paths={mapPaths} />
                  <ConditionEditor label="Disabled when" value={g.action.disabledWhen} onChange={(v) => up({ ...g, action: { ...g.action!, disabledWhen: v } })} paths={mapPaths} />
                  <ActionsEditor label="On click" value={g.action.onClick} onChange={(v) => up({ ...g, action: { ...g.action!, onClick: v ?? [] } })} draft={draft} paths={[...mapPaths, 'form']} />
                </>
              )}
            </>
          )} />
      </Box>
      <Box>
        <Typography variant="caption" color="text.secondary">Add controls (add a row whose key starts with a prefix, e.g. id:ISIN)</Typography>
        <ListEditor items={adds} onChange={(v) => set({ add: v.length ? v : undefined })} addLabel="Add control" create={(): Add => ({ prefix: '', label: '' as TextSpec, inputLabel: '' as TextSpec })}
          render={(a, up) => (
            <>
              <Stack direction="row" spacing={1}>
                <TextField size="small" label="Key prefix" value={a.prefix} onChange={(e) => up({ ...a, prefix: e.target.value })} />
                <TextField size="small" label="Goes in group" value={a.group ?? ''} onChange={(e) => up({ ...a, group: e.target.value || undefined })} />
              </Stack>
              <TextSpecField label="Button label" value={a.label} onChange={(v) => up({ ...a, label: v })} paths={mapPaths} />
              <TextSpecField label="Input label" value={a.inputLabel} onChange={(v) => up({ ...a, inputLabel: v })} paths={mapPaths} />
              <TextSpecField label="Help text" value={a.helperText} onChange={(v) => up({ ...a, helperText: v || undefined })} paths={mapPaths} />
              <TextField size="small" label="Suggestions" value={addOptions(a)} helperText="Separated by commas"
                onChange={(e) => { const v = e.target.value.split(',').map((x) => x.trim()).filter(Boolean); up({ ...a, options: v.length ? v : undefined }); }} />
              <SwitchField label="Tidy the typed code (UPPER_CASE)" checked={!!a.code} onChange={(on) => up({ ...a, code: on || undefined })} />
              <TextSpecField label="Added row's label ({{item.type}} is the code)" value={a.rowLabel} onChange={(v) => up({ ...a, rowLabel: v || undefined })} paths={[...mapPaths, 'item']} />
              <TextSpecField label="Remove button label" value={a.removeLabel} onChange={(v) => up({ ...a, removeLabel: v || undefined })} paths={mapPaths} />
              <ConditionEditor label="Show when" value={a.visibleWhen} onChange={(v) => up({ ...a, visibleWhen: v })} paths={paths} />
            </>
          )} />
      </Box>
    </Stack>
  );
}

/** A chip (or a few) read from a value: the value, how its label is translated, its colours, an optional order. */
export function ChipSpecEditor({ label, value, onChange, paths, withKeys }: {
  label: string; value: { value?: Binding; labelKey?: string; colorMap?: Record<string, string>; keys?: string[]; variant?: 'filled' | 'outlined' } | undefined;
  onChange: (v: Record<string, unknown> | undefined) => void; paths: string[]; withKeys?: boolean;
}) {
  const set = (patch: Record<string, unknown>) => onChange({ ...value, ...patch });
  return (
    <Box>
      <SwitchField label={label} checked={!!value} onChange={(on) => onChange(on ? { value: '' } : undefined)} />
      {value && (
        <Stack spacing={1} sx={{ mt: 1 }}>
          <BindingField label={withKeys ? 'Value (an object of counts, or a list)' : 'Value'} value={value.value} onChange={(v) => set({ value: v })} paths={paths} />
          <TextField size="small" label="Label translation prefix" value={value.labelKey ?? ''} helperText="e.g. mastering.counts. - the value is added to it"
            onChange={(e) => set({ labelKey: e.target.value || undefined })} />
          {withKeys && <WordsField label="Show these keys, in this order (optional)" value={value.keys} onChange={(v) => set({ keys: v })} />}
          <ColorMapEditor value={value.colorMap} onChange={(v) => set({ colorMap: v })} />
        </Stack>
      )}
    </Box>
  );
}
