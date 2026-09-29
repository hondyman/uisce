import React from 'react';
import { Stack, TextField } from '@mui/material';
import type { CorePageDefinition } from '../../../types/pageStudio';
import type { FormFieldSpec, TextSpec } from './appModel';
import { BindingField, ConditionEditor, JsonField, ListEditor, SelectField, SwitchField, TextSpecField } from './editors';

const KINDS: { value: FormFieldSpec['kind']; label: string }[] = [
  { value: 'text', label: 'Text' }, { value: 'multiline', label: 'Long text' }, { value: 'number', label: 'Number' },
  { value: 'date', label: 'Date' }, { value: 'time', label: 'Time of day' }, { value: 'select', label: 'Choice (list)' }, { value: 'radio', label: 'Choice (radio)' },
  { value: 'switch', label: 'On / off' }, { value: 'chips', label: 'List of values (chips)' },
  { value: 'rows', label: 'List of rows' }, { value: 'json', label: 'JSON' }, { value: 'map', label: 'Mapping table (key -> value)' },
];

/** One form field: name, label, kind, options (static or from a query), conditions. */
function FieldEditor({ f, onChange, paths, queries }: { f: FormFieldSpec; onChange: (f: FormFieldSpec) => void; paths: string[]; queries: { value: string; label: string }[] }) {
  const set = (patch: Partial<FormFieldSpec>) => onChange({ ...f, ...patch });
  const hasOptions = f.kind === 'select' || f.kind === 'radio' || f.kind === 'chips' || f.kind === 'map';
  return (
    <>
      <Stack direction="row" spacing={1}>
        <TextField size="small" label="Name" value={f.name} onChange={(e) => set({ name: e.target.value })} helperText="{{form.<name>}}" />
        <SelectField label="Kind" value={f.kind} options={KINDS} onChange={(v) => set({ kind: (v ?? 'text') as FormFieldSpec['kind'] })} />
      </Stack>
      <TextSpecField label="Label" value={f.label} onChange={(v) => set({ label: v })} paths={paths} />
      <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
        <SwitchField label="Required" checked={!!f.required} onChange={(v) => set({ required: v || undefined })} />
        <SwitchField label="Always read-only" checked={!!f.readOnly} onChange={(v) => set({ readOnly: v || undefined })} />
        <SwitchField label="Full width" checked={!!f.wide} onChange={(v) => set({ wide: v || undefined })} />
      </Stack>
      {f.kind === 'number' && (
        <TextField size="small" type="number" label="Step (1 = whole numbers)" value={f.step ?? ''} onChange={(e) => set({ step: e.target.value === '' ? undefined : Number(e.target.value) })} />
      )}
      {f.kind === 'rows' && (
        <JsonField label="Row fields [{name, label, kind: text|number}]" value={f.rowFields ?? []} minRows={3} onChange={(v) => set({ rowFields: (v || undefined) as FormFieldSpec['rowFields'] })} />
      )}
      {f.kind === 'map' && (
        <JsonField label="Mapping (rows, groups, hints, add)" value={f.map ?? { rows: '' }} minRows={6} onChange={(v) => set({ map: (v || undefined) as FormFieldSpec['map'] })} />
      )}
      <TextField size="small" label="Clear when these fields change" value={(f.resetOn ?? []).join(', ')} helperText="Field names, comma separated"
        onChange={(e) => { const v = e.target.value.split(',').map((x) => x.trim()).filter(Boolean); set({ resetOn: v.length ? v : undefined }); }} />
      <BindingField label="Default" value={f.default} onChange={(v) => set({ default: v })} paths={paths} />
      {hasOptions && (
        <>
          {f.kind !== 'chips' && (
            <SwitchField label="Store as a number" checked={f.valueType === 'number'} onChange={(v) => set({ valueType: v ? 'number' : undefined })} />
          )}
          <SwitchField label="Options from a query" checked={!!f.optionsFrom} onChange={(v) => set({ optionsFrom: v ? { query: queries[0]?.value ?? '' } : undefined, options: v ? undefined : f.options ?? [] })} />
          {f.optionsFrom ? (
            <>
              <SelectField label="Query" value={f.optionsFrom.query} options={queries} onChange={(v) => set({ optionsFrom: { ...f.optionsFrom!, query: v ?? '' } })} />
              <TextField size="small" label="Rows path (optional)" value={f.optionsFrom.rowsPath ?? ''} onChange={(e) => set({ optionsFrom: { ...f.optionsFrom!, rowsPath: e.target.value || undefined } })} />
              <Stack direction="row" spacing={1}>
                <TextField size="small" label="Value field" value={f.optionsFrom.valueField ?? ''} onChange={(e) => set({ optionsFrom: { ...f.optionsFrom!, valueField: e.target.value || undefined } })} />
                <TextField size="small" label="Label field" value={f.optionsFrom.labelField ?? ''} onChange={(e) => set({ optionsFrom: { ...f.optionsFrom!, labelField: e.target.value || undefined } })} />
              </Stack>
              <TextField size="small" label="Caption field (optional)" value={f.optionsFrom.captionField ?? ''} helperText="A second line under each option"
                onChange={(e) => set({ optionsFrom: { ...f.optionsFrom!, captionField: e.target.value || undefined } })} />
            </>
          ) : (
            <ListEditor items={f.options ?? []} onChange={(v) => set({ options: v })} addLabel="Add option" create={() => ({ value: '', label: '' as TextSpec })}
              render={(o, up) => (
                <>
                  <TextField size="small" label="Value" value={o.value} onChange={(e) => up({ ...o, value: e.target.value })} />
                  <TextSpecField label="Label" value={o.label} onChange={(v) => up({ ...o, label: v })} paths={paths} />
                </>
              )} />
          )}
        </>
      )}
      <TextSpecField label="Help text" value={f.helperText} onChange={(v) => set({ helperText: v || undefined })} paths={paths} />
      <ConditionEditor label="Show when" value={f.visibleWhen} onChange={(v) => set({ visibleWhen: v })} paths={[...paths, 'form']} />
      <ConditionEditor label="Read-only when" value={f.readOnlyWhen} onChange={(v) => set({ readOnlyWhen: v })} paths={[...paths, 'form']} />
    </>
  );
}

/** A form's fields (the Form widget; action forms). */
export function FieldsEditor({ fields, onChange, draft, paths }: {
  fields: FormFieldSpec[]; onChange: (f: FormFieldSpec[]) => void; draft: CorePageDefinition; paths: string[];
}) {
  const queries = (draft.app?.queries ?? []).map((q) => ({ value: q.id, label: `${q.id} (${q.operation})` }));
  return (
    <ListEditor items={fields} onChange={onChange} addLabel="Add field"
      create={(): FormFieldSpec => ({ name: `field${fields.length + 1}`, label: 'Field', kind: 'text' })}
      render={(f, up) => <FieldEditor f={f} onChange={up} paths={paths} queries={queries} />} />
  );
}
