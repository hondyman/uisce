import React from 'react';
import { Stack } from '@mui/material';
import type { CorePageDefinition } from '../../../types/pageStudio';
import type { Action, FormSpec, TextSpec } from './appModel';
import { BindingField, ConditionEditor, SelectField, SwitchField, TextSpecField } from './editors';
import { FieldsEditor } from './fieldsEditor';

type RunOperation = Extract<Action, { kind: 'runOperation' }>;

/**
 * What a "run operation" action can ask first: a yes/no confirmation, or a
 * form (its title, a notice, its fields, the submit button). The values
 * typed are {{form.<name>}} in the action's params and On success actions.
 */

export function ConfirmEditor({ value, onChange, paths }: {
  value: RunOperation['confirm']; onChange: (v: RunOperation['confirm']) => void; paths: string[];
}) {
  return (
    <Stack spacing={1}>
      <SwitchField label="Ask to confirm first" checked={!!value} onChange={(on) => onChange(on ? { title: '' as TextSpec } : undefined)} />
      {value && (
        <>
          <TextSpecField label="Title" value={value.title} onChange={(v) => onChange({ ...value, title: v })} paths={paths} />
          <TextSpecField label="Question" value={value.text} onChange={(v) => onChange({ ...value, text: v || undefined })} paths={paths} />
          <TextSpecField label="Confirm button" value={value.confirmLabel} onChange={(v) => onChange({ ...value, confirmLabel: v || undefined })} paths={paths} />
        </>
      )}
    </Stack>
  );
}

export function FormSpecEditor({ value, onChange, draft, paths }: {
  value: FormSpec | undefined; onChange: (v: FormSpec | undefined) => void; draft: CorePageDefinition; paths: string[];
}) {
  const formPaths = [...paths, 'form'];
  return (
    <Stack spacing={1}>
      <SwitchField label="Ask for input first" checked={!!value} onChange={(on) => onChange(on ? { title: '' as TextSpec, fields: [] } : undefined)} />
      {value && (
        <>
          <TextSpecField label="Title" value={value.title} onChange={(v) => onChange({ ...value, title: v })} paths={paths} />
          <TextSpecField label="Introduction" value={value.intro} onChange={(v) => onChange({ ...value, intro: v || undefined })} paths={paths} />
          <SwitchField label="A notice (e.g. 'needs 2 approvals')" checked={!!value.notice}
            onChange={(on) => onChange({ ...value, notice: on ? { severity: 'info', text: '' as TextSpec } : undefined })} />
          {value.notice && (
            <>
              <BindingField label="Severity (info, warning, or a binding that gives one)" value={value.notice.severity} paths={paths}
                onChange={(v) => onChange({ ...value, notice: { ...value.notice!, severity: v as string } })} />
              <TextSpecField label="Notice text" value={value.notice.text} onChange={(v) => onChange({ ...value, notice: { ...value.notice!, text: v } })} paths={paths} />
              <ConditionEditor label="Show notice when" value={value.notice.visibleWhen} onChange={(v) => onChange({ ...value, notice: { ...value.notice!, visibleWhen: v } })} paths={paths} />
            </>
          )}
          <FieldsEditor fields={value.fields} onChange={(f) => onChange({ ...value, fields: f })} draft={draft} paths={formPaths} />
          <TextSpecField label="Submit button" value={value.submitLabel} onChange={(v) => onChange({ ...value, submitLabel: v || undefined })} paths={paths} />
          <SelectField label="Submit colour" value={value.submitColor ?? 'primary'} options={['primary', 'inherit', 'error'].map((x) => ({ value: x as 'primary', label: x }))}
            onChange={(v) => onChange({ ...value, submitColor: v })} />
        </>
      )}
    </Stack>
  );
}
