import React, { useState } from 'react';
import { Accordion, AccordionDetails, AccordionSummary, Box, MenuItem, Paper, Stack, Tab, Tabs, TextField, Typography } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { ComponentDefinition, CorePageDefinition } from '../../../types/pageStudio';
import { getDomainComponent, listDomainComponents } from '../../../studio-core/components/registry';
import type { CellSpec, ColumnDef, ConditionNode, TextSpec } from './appModel';
import type { AppWidgetType, CanvasProps, ChatProps, DataGridProps, DomainComponentProps, FormWidgetProps, KeyValueProps, TimelineProps } from './AppWidgets';
import { FieldsEditor } from './fieldsEditor';
import { PAGE_ICONS } from './icons';
import {
  ActionsEditor, BindingField, ConditionEditor, JsonField, ListEditor, Section, SelectField, SwitchField, TextSpecField, scopePaths,
} from './editors';

const CELL_KINDS: { value: CellSpec['kind']; label: string }[] = [
  { value: 'text', label: 'Text' }, { value: 'twoLine', label: 'Two lines' }, { value: 'number', label: 'Number' },
  { value: 'percent', label: 'Percent (0-1)' }, { value: 'delta', label: 'Change % (coloured)' }, { value: 'datetime', label: 'Date/time' },
  { value: 'chip', label: 'Status chip' }, { value: 'chips', label: 'Chips (counts/list)' }, { value: 'diff', label: 'Change (old → new)' },
  { value: 'link', label: 'Link button' }, { value: 'actions', label: 'Row buttons' }, { value: 'input', label: 'Inline input' },
  { value: 'list', label: 'List of values' }, { value: 'toggle', label: 'On/off switch' },
];

const COLORS = ['default', 'primary', 'secondary', 'success', 'warning', 'error', 'info'];

/** One cell's structured fields; anything richer (colour maps, buttons) edits as JSON beside it. */
function CellEditor({ cell, onChange, paths, draft }: { cell: CellSpec; onChange: (c: CellSpec) => void; paths: string[]; draft: CorePageDefinition }) {
  const set = (patch: Record<string, unknown>) => onChange({ ...cell, ...patch } as CellSpec);
  const c = cell as Record<string, unknown>;
  const bind = (name: string, label: string) => <BindingField key={name} label={label} value={c[name]} onChange={(v) => set({ [name]: v })} paths={paths} />;
  const fields: React.ReactNode[] = [];
  switch (cell.kind) {
    case 'text':
      fields.push(bind('value', 'Value'), <SwitchField key="mono" label="Monospace" checked={!!cell.mono} onChange={(v) => set({ mono: v })} />,
        <SwitchField key="caption" label="Caption style" checked={!!cell.caption} onChange={(v) => set({ caption: v })} />,
        bind('tone', 'Tone (success|warning|error|info - tints the value)'), bind('strike', 'Struck through when (truthy)'), bind('tooltip', 'Tooltip'));
      break;
    case 'list':
      fields.push(bind('value', 'Values (a list)'),
        <SelectField key="d" label="Direction" value={cell.direction ?? 'column'} options={[{ value: 'column', label: 'Stacked' }, { value: 'row', label: 'In a row' }]} onChange={(v) => set({ direction: v })} />,
        <Box key="i"><Typography variant="caption" color="text.secondary">Each item ({'{{item}}'})</Typography>
          <CellEditor cell={cell.item ?? { kind: 'text', value: '{{item}}' }} onChange={(c) => set({ item: c })} paths={[...paths, 'item']} draft={draft} /></Box>);
      break;
    case 'twoLine': fields.push(bind('primary', 'Primary'), bind('secondary', 'Secondary'),
      <SwitchField key="m" label="Secondary monospace" checked={!!cell.secondaryMono} onChange={(v) => set({ secondaryMono: v })} />); break;
    case 'number': fields.push(bind('value', 'Value'), bind('suffix', 'Suffix (currency)')); break;
    case 'percent': case 'datetime': fields.push(bind('value', 'Value')); break;
    case 'delta': fields.push(bind('value', 'Value (%)'),
      <Stack key="t" direction="row" spacing={1}>
        <TextField size="small" type="number" label="Warn at" value={cell.warn ?? 1} onChange={(e) => set({ warn: Number(e.target.value) })} />
        <TextField size="small" type="number" label="Bad at" value={cell.bad ?? 5} onChange={(e) => set({ bad: Number(e.target.value) })} />
      </Stack>); break;
    case 'diff': fields.push(bind('before', 'Before'), bind('after', 'After')); break;
    case 'chip':
      fields.push(bind('value', 'Value'),
        <TextField key="lk" size="small" label="Label i18n prefix" value={cell.labelKey ?? ''} onChange={(e) => set({ labelKey: e.target.value || undefined })} helperText="e.g. mastering.goldenStatus." />,
        bind('colorBy', 'Colour by (default: value)'), bind('tooltip', 'Tooltip'), bind('caption', 'Caption after'),
        <SelectField key="v" label="Variant" value={cell.variant} options={[{ value: 'filled', label: 'Filled' }, { value: 'outlined', label: 'Outlined' }]} onChange={(v) => set({ variant: v })} />,
        <JsonField key="cm" label={`Colour map (value → ${COLORS.join('|')}; * = otherwise)`} value={cell.colorMap ?? {}} minRows={2} onChange={(v) => set({ colorMap: v })} />);
      break;
    case 'chips':
      fields.push(bind('value', 'Value (object of counts, or list)'),
        <TextField key="lk" size="small" label="Label i18n prefix" value={cell.labelKey ?? ''} onChange={(e) => set({ labelKey: e.target.value || undefined })} />,
        <JsonField key="k" label="Keys, in order (optional)" value={cell.keys ?? null} minRows={1} onChange={(v) => set({ keys: v || undefined })} />,
        <JsonField key="cm" label="Colour map" value={cell.colorMap ?? {}} minRows={2} onChange={(v) => set({ colorMap: v })} />);
      break;
    case 'link':
      fields.push(<TextSpecField key="l" label="Label" value={cell.label} onChange={(v) => set({ label: v })} paths={paths} />, bind('after', 'Text after'),
        <ActionsEditor key="a" label="On click" value={cell.onClick} onChange={(v) => set({ onClick: v ?? [] })} draft={draft} paths={paths} />);
      break;
    case 'input': fields.push(
      <TextField key="n" size="small" label="Name ({{rowState.<name>}})" value={cell.name} onChange={(e) => set({ name: e.target.value })} />,
      <TextSpecField key="p" label="Placeholder" value={cell.placeholder} onChange={(v) => set({ placeholder: v })} paths={paths} />); break;
    case 'toggle':
      fields.push(bind('value', 'On when'), <TextSpecField key="l" label="Label (for screen readers)" value={cell.label} onChange={(v) => set({ label: v })} paths={paths} />,
        <JsonField key="o" label="On change ({{row}}, {{value}}) - actions" value={cell.onChange ?? []} minRows={3} onChange={(v) => set({ onChange: v ?? [] })} />);
      break;
    case 'actions':
      fields.push(<JsonField key="b" label="Buttons [{label, icon, variant, color, visibleWhen, onClick}]" value={cell.buttons} onChange={(v) => set({ buttons: v })} minRows={6} />,
        <JsonField key="c" label="Caption {text, visibleWhen} (optional)" value={cell.caption ?? null} minRows={2} onChange={(v) => set({ caption: v || undefined })} />);
      break;
  }
  return (
    <>
      <TextField select size="small" label="Cell" value={cell.kind} onChange={(e) => onChange({ kind: e.target.value, value: c.value } as CellSpec)}>
        {CELL_KINDS.map((k) => <MenuItem key={k.value} value={k.value}>{k.label}</MenuItem>)}
      </TextField>
      {fields}
      <ConditionEditor label="Show when (row)" value={cell.visibleWhen} onChange={(v) => set({ visibleWhen: v })} paths={paths} />
    </>
  );
}

function ColumnEditor({ col, onChange, paths, draft }: { col: ColumnDef; onChange: (c: ColumnDef) => void; paths: string[]; draft: CorePageDefinition }) {
  const cells: CellSpec[] = col.stack ?? (col.cell ? [col.cell] : []);
  const setCells = (cs: CellSpec[]) => onChange({ ...col, cell: cs.length === 1 ? cs[0] : undefined, stack: cs.length > 1 ? cs : undefined });
  const title = typeof col.header === 'string' ? col.header : col.header?.t;
  return (
    <Accordion disableGutters variant="outlined" sx={{ '&:before': { display: 'none' } }}>
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Typography variant="body2" fontWeight={600} noWrap>{title || col.id}</Typography>
        <Typography variant="caption" color="text.secondary" sx={{ ml: 1 }} noWrap>{cells.map((c) => c.kind).join(' + ') || col.field}</Typography>
      </AccordionSummary>
      <AccordionDetails>
        <Stack spacing={1.25}>
          <TextField size="small" label="Id" value={col.id} onChange={(e) => onChange({ ...col, id: e.target.value })} />
          <TextSpecField label="Header" value={col.header} onChange={(v) => onChange({ ...col, header: v })} paths={paths} />
          {cells.length === 0 && (
            <TextField size="small" label="Field (plain text)" value={col.field ?? ''} onChange={(e) => onChange({ ...col, field: e.target.value })} />
          )}
          <Stack direction="row" spacing={1}>
            <SelectField label="Align" value={col.align} options={[{ value: 'left', label: 'Left' }, { value: 'right', label: 'Right' }, { value: 'center', label: 'Center' }]} onChange={(v) => onChange({ ...col, align: v })} />
            <SwitchField label="No wrap" checked={!!col.nowrap} onChange={(v) => onChange({ ...col, nowrap: v || undefined })} />
          </Stack>
          <BindingField label="Tooltip" value={col.tooltip} onChange={(v) => onChange({ ...col, tooltip: v || undefined })} paths={paths} />
          <Typography variant="caption" color="text.secondary">Cells (several stack in one column)</Typography>
          <ListEditor items={cells} onChange={setCells} addLabel="Add cell" create={(): CellSpec => ({ kind: 'text', value: '' })}
            render={(cell, update) => <CellEditor cell={cell} onChange={update} paths={paths} draft={draft} />} />
          {cells.length > 1 && (
            <SelectField label="Stack" value={col.stackDirection} options={[{ value: 'column', label: 'Vertical' }, { value: 'row', label: 'Inline' }]} onChange={(v) => onChange({ ...col, stackDirection: v })} />
          )}
          <ConditionEditor label="Show column when (page)" value={col.visibleWhen} onChange={(v) => onChange({ ...col, visibleWhen: v })} paths={scopePaths(draft)} />
        </Stack>
      </AccordionDetails>
    </Accordion>
  );
}

/**
 * Properties for page application widgets (app/AppWidgets.tsx): structured
 * editors per type, the widget's visibility condition, and a JSON view of
 * the whole widget for anything not yet covered by a form.
 */
export default function AppWidgetInspector({ component, draft, setDraft }: {
  component: ComponentDefinition; draft: CorePageDefinition; setDraft: React.Dispatch<React.SetStateAction<CorePageDefinition>>;
}) {
  const [view, setView] = useState<'form' | 'json'>('form');
  const type = component.type as AppWidgetType;
  const props = (component.props ?? {}) as Record<string, unknown>;
  const update = (next: Partial<ComponentDefinition>) => setDraft((prev) => ({
    ...prev, components: { ...prev.components, [component.id]: { ...prev.components[component.id], ...next } },
  }));
  const setProps = (patch: Record<string, unknown>) => update({ props: { ...props, ...patch } });
  // Row paths for per-row editors: a grid's or a timeline's query.
  const gridQuery = type === 'DataGrid' || type === 'Timeline' ? (props.query as string | undefined) : undefined;
  const paths = scopePaths(draft);
  const rowPaths = scopePaths(draft, gridQuery);
  const vars = (draft.app?.variables ?? []).map((v) => ({ value: v.name, label: v.name }));
  const queries = (draft.app?.queries ?? []).map((q) => ({ value: q.id, label: `${q.id} (${q.operation})` }));
  const text = (name: string, label: string) => (
    <TextSpecField key={name} label={label} value={props[name] as TextSpec | undefined} onChange={(v) => setProps({ [name]: v })} paths={paths} />
  );
  const actions = (name: string, label: string, extra: string[] = []) => (
    <ActionsEditor key={name} label={label} value={props[name] as never} onChange={(v) => setProps({ [name]: v })} draft={draft} paths={[...paths, ...extra]} />
  );
  const condition = (name: string, label: string) => (
    <ConditionEditor key={name} label={label} value={props[name] as ConditionNode | undefined} onChange={(v) => setProps({ [name]: v })} paths={paths} />
  );

  let body: React.ReactNode = null;
  switch (type) {
    case 'PageHeader':
      body = (
        <Section title="Header">
          <SelectField label="Icon" value={props.icon as string} options={Object.keys(PAGE_ICONS).map((k) => ({ value: k, label: k }))} onChange={(v) => setProps({ icon: v })} allowEmpty="None" />
          {text('title', 'Title')}{text('subtitle', 'Subtitle')}
        </Section>
      );
      break;
    case 'VariableSelect': {
      const from = props.optionsFrom as { query: string; rowsPath?: string; valueField?: string; labelField?: string } | undefined;
      body = (
        <>
          <Section title="Select">
            <SelectField label="Sets variable" value={props.variable as string} options={vars} onChange={(v) => setProps({ variable: v })} />
            {text('label', 'Label')}{text('emptyLabel', 'Empty option label (All)')}
            <SelectField label="Style" value={(props.variant as 'select') ?? 'select'} options={[{ value: 'select', label: 'Dropdown' }, { value: 'toggle', label: 'Toggle buttons' }]} onChange={(v) => setProps({ variant: v === 'select' ? undefined : v })} />
            <TextField size="small" type="number" label="Min width" value={props.minWidth ?? ''} onChange={(e) => setProps({ minWidth: e.target.value ? Number(e.target.value) : undefined })} />
          </Section>
          <Section title="Options">
            <SwitchField label="From a query" checked={!!from} onChange={(v) => setProps({ optionsFrom: v ? { query: queries[0]?.value ?? '' } : undefined, options: v ? undefined : [] })} />
            {from ? (
              <>
                <SelectField label="Query" value={from.query} options={queries} onChange={(v) => setProps({ optionsFrom: { ...from, query: v ?? '' } })} />
                <TextField size="small" label="Rows path (optional)" value={from.rowsPath ?? ''} onChange={(e) => setProps({ optionsFrom: { ...from, rowsPath: e.target.value || undefined } })} />
                <TextField size="small" label="Value field" value={from.valueField ?? ''} onChange={(e) => setProps({ optionsFrom: { ...from, valueField: e.target.value || undefined } })} />
                <TextField size="small" label="Label field" value={from.labelField ?? ''} onChange={(e) => setProps({ optionsFrom: { ...from, labelField: e.target.value || undefined } })} />
              </>
            ) : (
              <ListEditor items={(props.options as { value: string; label: TextSpec }[]) ?? []} onChange={(v) => setProps({ options: v })} addLabel="Add option"
                create={() => ({ value: '', label: '' })}
                render={(o, set) => (
                  <>
                    <TextField size="small" label="Value" value={o.value} onChange={(e) => set({ ...o, value: e.target.value })} />
                    <TextSpecField label="Label" value={o.label} onChange={(v) => set({ ...o, label: v })} paths={paths} />
                  </>
                )} />
            )}
          </Section>
          <Section title="Behaviour">{actions('onChange', 'After change ({{value}})', ['value'])}{condition('disabledWhen', 'Disabled when')}</Section>
        </>
      );
      break;
    }
    case 'SearchInput':
      body = (
        <Section title="Search">
          <SelectField label="Sets variable" value={props.variable as string} options={vars} onChange={(v) => setProps({ variable: v })} />
          {text('label', 'Label')}
          {text('placeholder', 'Placeholder')}
          <SelectField label="Variant" value={(props.variant as string) ?? 'search'}
            options={[{ value: 'search', label: 'Search input' }, { value: 'typeahead', label: 'Typeahead / Autocomplete' }, { value: 'plain', label: 'Plain input' }, { value: 'title', label: 'Title style' }, { value: 'date', label: 'Date' }]}
            onChange={(v) => setProps({ variant: v as never })} />
          <TextField size="small" label="Options from query path (typeahead)" value={props.optionsFrom as string ?? ''} onChange={(e) => setProps({ optionsFrom: e.target.value || undefined })} />
          <TextField size="small" label="Options field in rows (typeahead)" value={props.optionsField as string ?? ''} onChange={(e) => setProps({ optionsField: e.target.value || undefined })} />
          <TextField size="small" type="number" label="Debounce (ms)" value={props.debounceMs ?? 300} onChange={(e) => setProps({ debounceMs: Number(e.target.value) })} />
        </Section>
      );
      break;
    case 'ActionButton':
      body = (
        <>
          <Section title="Button">
            {text('label', 'Label')}
            <SelectField label="Icon" value={props.icon as string} options={Object.keys(PAGE_ICONS).map((k) => ({ value: k, label: k }))} onChange={(v) => setProps({ icon: v })} allowEmpty="None" />
            <SelectField label="Style" value={props.variant as 'contained'} options={['contained', 'outlined', 'text'].map((v) => ({ value: v as 'contained', label: v }))} onChange={(v) => setProps({ variant: v })} />
            <SelectField label="Colour" value={props.color as 'primary'} options={['primary', 'secondary', 'inherit', 'error'].map((v) => ({ value: v as 'primary', label: v }))} onChange={(v) => setProps({ color: v })} />
          </Section>
          <Section title="Behaviour">{actions('onClick', 'On click')}{condition('disabledWhen', 'Disabled when')}</Section>
        </>
      );
      break;
    case 'DataGrid': {
      const p = props as unknown as DataGridProps;
      body = (
        <>
          <Section title="Data">
            <SelectField label="Query" value={p.query} options={queries} onChange={(v) => setProps({ query: v })} />
            <TextField size="small" label="Rows path (when data is not the list)" value={p.rowsPath ?? ''} onChange={(e) => setProps({ rowsPath: e.target.value || undefined })} />
            <TextField size="small" label="Row key" value={p.rowKey ?? 'id'} onChange={(e) => setProps({ rowKey: e.target.value })} />
            {text('emptyText', 'When empty')}
            <SwitchField label="Show progress while refetching" checked={!!p.progressOnFetch} onChange={(v) => setProps({ progressOnFetch: v })} />
          </Section>
          <Section title="Filtering & Search">
            <SwitchField label="Enable built-in table search toolbar" checked={!!p.enableSearch} onChange={(v) => setProps({ enableSearch: v || undefined })} />
            {p.enableSearch && (
              <>
                <SelectField label="Search style" value={p.searchVariant ?? 'typeahead'}
                  options={[{ value: 'typeahead', label: 'Typeahead / Autocomplete' }, { value: 'text', label: 'Standard text input' }]}
                  onChange={(v) => setProps({ searchVariant: v as never })} />
                <TextField size="small" label="Placeholder" value={typeof p.searchPlaceholder === 'string' ? p.searchPlaceholder : ''} onChange={(e) => setProps({ searchPlaceholder: e.target.value || undefined })} />
                <TextField size="small" label="Typeahead option field in rows (e.g. attribute_code, vendor_name)" value={p.searchOptionsField ?? ''} onChange={(e) => setProps({ searchOptionsField: e.target.value || undefined })} />
                <TextField size="small" type="number" label="Search width (px)" value={typeof p.searchWidth === 'number' ? p.searchWidth : 400} onChange={(e) => setProps({ searchWidth: e.target.value ? Number(e.target.value) : undefined })} />
              </>
            )}
            <SelectField label="Sync with page search variable (optional)" value={p.searchVariable ?? ''} options={[{ value: '', label: '(None / Standalone)' }, ...vars]} onChange={(v) => setProps({ searchVariable: v || undefined })} />
            <TextField size="small" label="Search fields (comma-separated, empty = all)" value={(p.searchFields ?? []).join(', ')} onChange={(e) => setProps({ searchFields: e.target.value ? e.target.value.split(',').map((s) => s.trim()).filter(Boolean) : undefined })} />
            <JsonField label="Variable filters { rowField: varName }" value={p.filters ?? null} minRows={2} onChange={(v) => setProps({ filters: v || undefined })} />
          </Section>
          <Section title="Columns">
            <ListEditor items={p.columns ?? []} onChange={(v) => setProps({ columns: v })} addLabel="Add column"
              create={() => ({ id: `col_${Math.random().toString(36).slice(2, 6)}`, header: '', field: '' })}
              render={(col, set) => <ColumnEditor col={col} onChange={set} paths={rowPaths} draft={draft} />} />
          </Section>
          <Section title="Behaviour">{actions('onRowClick', 'On row click ({{row}})', rowPaths)}</Section>
          <Section title="Wide and nested">
            <SwitchField label="Freeze the first column" checked={!!p.stickyFirstColumn} onChange={(v) => setProps({ stickyFirstColumn: v || undefined })} />
            <JsonField label="Columns from data {from, idField, header, valuePath, cell, insertAt} ({{col}}, {{value}})" value={p.dynamicColumns ?? null} minRows={4}
              onChange={(v) => setProps({ dynamicColumns: v || undefined })} />
            <JsonField label="Row detail {rows, columns, text, when} - an expandable nested table ({{row}} = detail row, {{parent}})" value={p.rowDetail ?? null} minRows={4}
              onChange={(v) => setProps({ rowDetail: v || undefined })} />
          </Section>
        </>
      );
      break;
    }
    case 'FacetFilter':
      body = (
        <>
          <Section title="Facet Filter">
            <SelectField label="Target variable" value={props.variable as string} options={vars} onChange={(v) => setProps({ variable: v })} />
            {text('label', 'Label')}{text('allLabel', 'All options label')}
            <SelectField label="Count from query" value={props.query as string ?? ''} options={[{ value: '', label: '(None)' }, ...queries]} onChange={(v) => setProps({ query: v || undefined })} />
            <TextField size="small" label="Rows path in query" value={props.rowsPath as string ?? ''} onChange={(e) => setProps({ rowsPath: e.target.value || undefined })} />
            <TextField size="small" label="Facet field in row" value={props.facetField as string ?? ''} onChange={(e) => setProps({ facetField: e.target.value || undefined })} />
          </Section>
          <Section title="Facet Options">
            <JsonField label="Options [{ value, label, color }]" value={props.options ?? []} minRows={3} onChange={(v) => setProps({ options: v || [] })} />
            <JsonField label="Color map { value: color }" value={props.colorMap ?? null} minRows={2} onChange={(v) => setProps({ colorMap: v || undefined })} />
          </Section>
        </>
      );
      break;
    case 'ProgressBar':
      body = (
        <Section title="Progress Bar">
          <SelectField label="Message variable" value={props.variable as string ?? ''} options={[{ value: '', label: '(None)' }, ...vars]} onChange={(v) => setProps({ variable: v || undefined })} />
          {text('label', 'Static label / template')}
          <SwitchField label="Show pulsing signal" checked={props.showSignal !== false} onChange={(v) => setProps({ showSignal: v })} />
          <SelectField label="Color" value={(props.color as string) ?? 'primary'}
            options={['primary', 'secondary', 'success', 'warning', 'info'].map((c) => ({ value: c, label: c }))}
            onChange={(v) => setProps({ color: v as never })} />
          <TextField size="small" type="number" label="Progress (0-100, blank = indeterminate)" value={props.progress ?? ''} onChange={(e) => setProps({ progress: e.target.value ? Number(e.target.value) : undefined })} />
        </Section>
      );
      break;
    case 'AlertBanner':
      body = (
        <Section title="Alert">
          <BindingField label="Severity (or a value to map)" value={props.severity} onChange={(v) => setProps({ severity: v })} paths={paths} />
          <JsonField label="Severity map (value → success|info|warning|error)" value={props.severityMap ?? null} minRows={2} onChange={(v) => setProps({ severityMap: v || undefined })} />
          {text('text', 'Text')}{text('detail', 'Detail')}
          <JsonField label="Chips {value, labelKey, colorMap} (optional)" value={props.chips ?? null} minRows={2} onChange={(v) => setProps({ chips: v || undefined })} />
          {actions('onClose', 'On close (makes it closable)')}
          <SwitchField label="Action button" checked={!!props.action} onChange={(v) => setProps({ action: v ? { label: 'Action', onClick: [] } : undefined })} />
          {!!props.action && (
            <>
              <TextSpecField label="Button label" value={(props.action as { label: TextSpec }).label} onChange={(v) => setProps({ action: { ...(props.action as object), label: v } })} paths={paths} />
              <ActionsEditor label="On click" value={(props.action as { onClick: never }).onClick} onChange={(v) => setProps({ action: { ...(props.action as object), onClick: v ?? [] } })} draft={draft} paths={paths} />
            </>
          )}
        </Section>
      );
      break;
    case 'TextBlock':
      body = (
        <Section title="Text">
          {text('text', 'Text')}
          <SelectField label="Style" value={(props.variant as 'body2') ?? 'body2'}
            options={['h4', 'h5', 'h6', 'subtitle1', 'subtitle2', 'body1', 'body2', 'caption', 'overline'].map((v) => ({ value: v as 'body2', label: v }))} onChange={(v) => setProps({ variant: v })} />
          <SelectField label="Colour" value={props.color as 'text.primary'} allowEmpty="Default"
            options={['text.primary', 'text.secondary', 'primary.main', 'error.main', 'warning.main', 'success.main', 'info.main'].map((v) => ({ value: v as 'text.primary', label: v }))}
            onChange={(v) => setProps({ color: v || undefined })} />
        </Section>
      );
      break;
    case 'Chat': {
      const p = props as unknown as ChatProps;
      const msgPaths = [...paths, 'message', 'index'];
      const setMessage = (patch: Partial<NonNullable<ChatProps['message']>>) => setProps({ message: { ...p.message, ...patch } });
      body = (
        <>
          <Section title="Conversation">
            <SelectField label="Messages in variable" value={p.variable} options={vars} onChange={(v) => setProps({ variable: v })} />
            {text('title', 'Title')}
            <SelectField label="Icon" value={p.icon} options={Object.keys(PAGE_ICONS).map((k) => ({ value: k, label: k }))} allowEmpty="None" onChange={(v) => setProps({ icon: v || undefined })} />
            {text('intro', 'When empty')}
            <JsonField label="Starters (a list of texts, or a binding)" value={p.starters ?? []} minRows={3} onChange={(v) => setProps({ starters: (v ?? undefined) as ChatProps['starters'] })} />
            {text('placeholder', 'Placeholder')}{text('busyText', 'While waiting')}
          </Section>
          <Section title="Sending">
            {actions('onSend', 'On send ({{text}}, {{messages}})', ['text', 'messages'])}
          </Section>
          <Section title="Assistant messages ({{message}}, {{index}})">
            <BindingField label="Detail lines" value={p.message?.details} onChange={(v) => setMessage({ details: v || undefined })} paths={msgPaths} />
            <BindingField label="Warning lines" value={p.message?.warning} onChange={(v) => setMessage({ warning: v || undefined })} paths={msgPaths} />
            <TextSpecField label="Warning title" value={p.message?.warningTitle} onChange={(v) => setMessage({ warningTitle: v || undefined })} paths={msgPaths} />
            <SwitchField label="An action on each message" checked={!!p.message?.action}
              onChange={(v) => setMessage({ action: v ? { label: 'Apply', doneLabel: 'Applied', onClick: [] } : undefined })} />
            {p.message?.action && (
              <>
                <TextSpecField label="Action label" value={p.message.action.label} onChange={(v) => setMessage({ action: { ...p.message!.action!, label: v } })} paths={msgPaths} />
                <TextSpecField label="Once done" value={p.message.action.doneLabel} onChange={(v) => setMessage({ action: { ...p.message!.action!, doneLabel: v || undefined } })} paths={msgPaths} />
                <ConditionEditor label="Show when" value={p.message.action.visibleWhen} onChange={(v) => setMessage({ action: { ...p.message!.action!, visibleWhen: v } })} paths={msgPaths} />
                <ActionsEditor label="On click" value={p.message.action.onClick} onChange={(v) => setMessage({ action: { ...p.message!.action!, onClick: v ?? [] } })} draft={draft} paths={msgPaths} />
              </>
            )}
          </Section>
          <Section title="Closing">{actions('onClose', 'On close (makes it closable)')}</Section>
        </>
      );
      break;
    }
    case 'Canvas': {
      const p = props as unknown as CanvasProps;
      const nodePaths = [...paths, 'node', 'extra'];
      const setNode = (patch: Partial<CanvasProps['node']>) => setProps({ node: { ...p.node, ...patch } });
      body = (
        <>
          <Section title="Graph">
            <SelectField label="Graph in variable" value={p.variable} options={vars} onChange={(v) => setProps({ variable: v })} />
            <Stack direction="row" spacing={1}>
              <TextField size="small" label="Nodes at" value={p.nodesPath ?? 'nodes'} onChange={(e) => setProps({ nodesPath: e.target.value || undefined })} />
              <TextField size="small" label="Edges at" value={p.edgesPath ?? 'edges'} onChange={(e) => setProps({ edgesPath: e.target.value || undefined })} />
            </Stack>
            <Stack direction="row" spacing={1}>
              <TextField size="small" label="Edge from field" value={p.edgeFrom ?? 'from'} onChange={(e) => setProps({ edgeFrom: e.target.value || undefined })} />
              <TextField size="small" label="Edge to field" value={p.edgeTo ?? 'to'} onChange={(e) => setProps({ edgeTo: e.target.value || undefined })} />
            </Stack>
            <SelectField label="Selected node id in variable" value={p.selectedVariable} options={vars} allowEmpty="None" onChange={(v) => setProps({ selectedVariable: v || undefined })} />
            <SwitchField label="One input per node (a new connection replaces it)" checked={!!p.singleInput} onChange={(v) => setProps({ singleInput: v || undefined })} />
            <Stack direction="row" spacing={1} alignItems="center">
              <TextField size="small" type="number" label="Height" value={p.height ?? 520} onChange={(e) => setProps({ height: Number(e.target.value) || undefined })} />
              <SwitchField label="Minimap" checked={p.minimap !== false} onChange={(v) => setProps({ minimap: v ? undefined : false })} />
            </Stack>
            {text('emptyText', 'When empty')}{text('emptyHint', 'Hint when empty')}
          </Section>
          <Section title="Nodes ({{node}}, {{extra}})">
            <TextSpecField label="Title" value={p.node?.title} onChange={(v) => setNode({ title: v })} paths={nodePaths} />
            <TextSpecField label="Subtitle" value={p.node?.subtitle} onChange={(v) => setNode({ subtitle: v || undefined })} paths={nodePaths} />
            <TextSpecField label="When the subtitle is empty" value={p.node?.placeholder} onChange={(v) => setNode({ placeholder: v || undefined })} paths={nodePaths} />
            <BindingField label="Icon (a name)" value={p.node?.icon} onChange={(v) => setNode({ icon: v })} paths={nodePaths} helperText={Object.keys(PAGE_ICONS).join(', ')} />
            <BindingField label="Category" value={p.node?.category} onChange={(v) => setNode({ category: v })} paths={nodePaths} />
            <BindingField label="Problems (a message or list)" value={p.node?.errors} onChange={(v) => setNode({ errors: v || undefined })} paths={nodePaths} />
            <BindingField label="Chips [{label, color, variant}]" value={p.node?.chips} onChange={(v) => setNode({ chips: v || undefined })} paths={nodePaths} />
            <JsonField label="Categories {name: {color, inputs, outputs}}" value={p.categories ?? {}} minRows={3} onChange={(v) => setProps({ categories: (v || undefined) as CanvasProps['categories'] })} />
            <BindingField label="Extra data per node id" value={p.nodeData} onChange={(v) => setProps({ nodeData: v || undefined })} paths={paths} helperText="e.g. {{queries.issues.data.by_node}}" />
            {condition('animatedWhen', 'Edges animate when')}
          </Section>
          <Section title="Palette ({{item}})">
            <SwitchField label="Show a palette" checked={!!p.palette} onChange={(v) => setProps({ palette: v ? { query: queries[0]?.value ?? '', label: '{{item.label}}' } : undefined })} />
            {p.palette && (
              <>
                <SelectField label="Items from query" value={p.palette.query} options={queries} onChange={(v) => setProps({ palette: { ...p.palette!, query: v ?? '' } })} />
                <Stack direction="row" spacing={1}>
                  <TextField size="small" label="Rows path" value={p.palette.rowsPath ?? ''} onChange={(e) => setProps({ palette: { ...p.palette!, rowsPath: e.target.value || undefined } })} />
                  <TextField size="small" label="Group by field" value={p.palette.groupBy ?? ''} onChange={(e) => setProps({ palette: { ...p.palette!, groupBy: e.target.value || undefined } })} />
                </Stack>
                <JsonField label="Groups [{id, label}] (order and titles)" value={p.palette.groups ?? []} minRows={2} onChange={(v) => setProps({ palette: { ...p.palette!, groups: (v || undefined) as never } })} />
                <TextSpecField label="Label" value={p.palette.label} onChange={(v) => setProps({ palette: { ...p.palette!, label: v } })} paths={[...paths, 'item']} />
                <TextSpecField label="Description" value={p.palette.description} onChange={(v) => setProps({ palette: { ...p.palette!, description: v || undefined } })} paths={[...paths, 'item']} />
                <BindingField label="Icon" value={p.palette.icon} onChange={(v) => setProps({ palette: { ...p.palette!, icon: v } })} paths={[...paths, 'item']} />
                <BindingField label="Category" value={p.palette.category} onChange={(v) => setProps({ palette: { ...p.palette!, category: v } })} paths={[...paths, 'item']} />
                <ConditionEditor label="Disabled when" value={p.palette.disabledWhen} onChange={(v) => setProps({ palette: { ...p.palette!, disabledWhen: v } })} paths={[...paths, 'item']} />
              </>
            )}
          </Section>
          <Section title="Actions">
            {actions('onAdd', 'Add ({{item}}, {{graph}}, {{selected}})', ['item', 'graph', 'selected'])}
            {actions('onChange', 'After the graph changes ({{graph}})', ['graph'])}
          </Section>
        </>
      );
      break;
    }
    case 'KeyValue': {
      const p = props as unknown as KeyValueProps;
      const dataPaths = [...paths, 'data'];
      body = (
        <>
          <Section title="Record">
            <BindingField label="Record (templates below see {{data}})" value={p.source} onChange={(v) => setProps({ source: v })} paths={paths}
              helperText="e.g. {{queries.golden.data}}" />
            <TextField size="small" type="number" label="Columns" value={p.columns ?? 2} onChange={(e) => setProps({ columns: Math.max(1, Number(e.target.value) || 1) })} />
            {text('emptyText', 'When empty')}
          </Section>
          <Section title="Items">
            <SwitchField label="Pairs from data (list or object)" checked={!!p.pairsFrom} onChange={(v) => setProps({ pairsFrom: v ? { value: '{{data}}' } : undefined })} />
            {p.pairsFrom ? (
              <>
                <BindingField label="Pairs" value={p.pairsFrom.value} onChange={(v) => setProps({ pairsFrom: { ...p.pairsFrom!, value: v } })} paths={dataPaths} />
                <Stack direction="row" spacing={1}>
                  <TextField size="small" label="Label field" value={p.pairsFrom.labelField ?? ''} onChange={(e) => setProps({ pairsFrom: { ...p.pairsFrom!, labelField: e.target.value || undefined } })} />
                  <TextField size="small" label="Value field" value={p.pairsFrom.valueField ?? ''} onChange={(e) => setProps({ pairsFrom: { ...p.pairsFrom!, valueField: e.target.value || undefined } })} />
                </Stack>
              </>
            ) : (
              <ListEditor items={p.items ?? []} onChange={(v) => setProps({ items: v })} addLabel="Add item" create={() => ({ label: 'Label', value: '{{data.field}}' })}
                render={(it, up) => (
                  <>
                    <TextSpecField label="Label" value={it.label} onChange={(v) => up({ ...it, label: v })} paths={dataPaths} />
                    <SwitchField label="Rich cell (chip, number, date...)" checked={!!it.cell} onChange={(v) => up({ ...it, cell: v ? { kind: 'text', value: it.value } : undefined })} />
                    {it.cell
                      ? <CellEditor cell={it.cell} onChange={(c) => up({ ...it, cell: c })} paths={dataPaths} draft={draft} />
                      : <BindingField label="Value" value={it.value} onChange={(v) => up({ ...it, value: v })} paths={dataPaths} />}
                    <ConditionEditor label="Show when" value={it.visibleWhen} onChange={(v) => up({ ...it, visibleWhen: v })} paths={dataPaths} />
                  </>
                )} />
            )}
          </Section>
        </>
      );
      break;
    }
    case 'Timeline': {
      const p = props as unknown as TimelineProps;
      body = (
        <>
          <Section title="Items">
            <SelectField label="Query" value={p.query} options={queries} allowEmpty="From a binding" onChange={(v) => setProps({ query: v || undefined })} />
            {p.query
              ? <TextField size="small" label="Rows path (optional)" value={p.rowsPath ?? ''} onChange={(e) => setProps({ rowsPath: e.target.value || undefined })} />
              : <BindingField label="Items (a list)" value={p.items} onChange={(v) => setProps({ items: v })} paths={paths} />}
            {text('emptyText', 'When empty')}
            <TextField size="small" type="number" label="Max height (px, scrolls)" value={p.maxHeight ?? ''} onChange={(e) => setProps({ maxHeight: e.target.value ? Number(e.target.value) : undefined })} />
          </Section>
          <Section title="Each item ({{row}})">
            <TextSpecField label="Title" value={p.title} onChange={(v) => setProps({ title: v })} paths={rowPaths} />
            <TextSpecField label="Subtitle" value={p.subtitle} onChange={(v) => setProps({ subtitle: v || undefined })} paths={rowPaths} />
            <BindingField label="Time" value={p.time} onChange={(v) => setProps({ time: v || undefined })} paths={rowPaths} />
            <JsonField label="Chip {value, labelKey, colorMap} (optional)" value={p.chip ?? null} minRows={2} onChange={(v) => setProps({ chip: v || undefined })} />
          </Section>
          <Section title="Behaviour">
            <ConditionEditor label="Selected when" value={p.selectedWhen} onChange={(v) => setProps({ selectedWhen: v })} paths={rowPaths} />
            {actions('onItemClick', 'On click ({{row}})', rowPaths)}
          </Section>
        </>
      );
      break;
    }
    case 'Form': {
      const p = props as unknown as FormWidgetProps;
      body = (
        <>
          <Section title="Form">
            <SelectField label="Values in variable" value={p.variable} options={vars} onChange={(v) => setProps({ variable: v })} />
            <BindingField label="Start from (optional)" value={p.initFrom} onChange={(v) => setProps({ initFrom: v || undefined })} paths={paths}
              helperText="e.g. {{vars.editRow}} - re-seeds when it changes" />
            <BindingField label="Fields from data (optional)" value={p.fieldsFrom} onChange={(v) => setProps({ fieldsFrom: v || undefined })} paths={paths}
              helperText="A list of field specs an operation shapes (e.g. from a table's columns); replaces the fields below" />
            <TextField size="small" type="number" label="Columns" value={p.columns ?? 1} onChange={(e) => setProps({ columns: Math.max(1, Number(e.target.value) || 1) })} />
          </Section>
          <Section title="Fields">
            <FieldsEditor fields={p.fields ?? []} onChange={(v) => setProps({ fields: v })} draft={draft} paths={paths} />
          </Section>
          <Section title="Submit (optional - or use a dialog's buttons)">
            {text('submitLabel', 'Button label')}
            {actions('onSubmit', 'On submit ({{form}})', ['form'])}
            {condition('submitDisabledWhen', 'Disabled when')}
          </Section>
          <Section title="On change">
            {actions('onChange', 'After a field changes ({{form}})', ['form'])}
          </Section>
        </>
      );
      break;
    }
    case 'DomainComponent': {
      const p = props as unknown as DomainComponentProps;
      const def = getDomainComponent(p.component);
      body = (
        <>
          <Section title="Component">
            <SelectField label="Domain component" value={p.component} options={listDomainComponents().map((d) => ({ value: d.id, label: `${d.label} (${d.id})` }))} onChange={(v) => setProps({ component: v, inputs: {}, events: {} })} />
            {def?.description && <Typography variant="caption" color="text.secondary">{def.description}</Typography>}
          </Section>
          {def && (
            <Section title="Inputs">
              {def.inputs.map((i) => (
                <BindingField key={i.name} label={`${i.label ?? i.name}${i.required ? ' *' : ''}`} value={p.inputs?.[i.name]} helperText={i.description} paths={paths}
                  onChange={(v) => setProps({ inputs: { ...p.inputs, [i.name]: v } })} />
              ))}
            </Section>
          )}
          {def && def.events.length > 0 && (
            <Section title="Events">
              {def.events.map((e) => (
                <ActionsEditor key={e.name} label={`On ${e.label ?? e.name}${e.payload ? ` ({{event.${e.payload.join('}}, {{event.')}}})` : ''}`} value={p.events?.[e.name]}
                  onChange={(v) => setProps({ events: { ...p.events, [e.name]: v ?? [] } })} draft={draft} paths={[...paths, ...(e.payload ?? []).map((x) => `event.${x}`)]} />
              ))}
            </Section>
          )}
        </>
      );
      break;
    }
  }

  return (
    <Paper elevation={0} sx={{ height: '100%', bgcolor: 'background.paper', overflowY: 'auto' }}>
      <Box sx={{ p: 2 }}>
        <Typography variant="overline" color="text.secondary" fontWeight="bold">{type}</Typography>
        <Typography variant="caption" color="text.secondary" component="div" fontFamily="monospace">{component.id}</Typography>
        <Tabs value={view} onChange={(_, v) => setView(v)} sx={{ minHeight: 32, mt: 1, '& .MuiTab-root': { minHeight: 32, py: 0 } }}>
          <Tab value="form" label="Properties" />
          <Tab value="json" label="JSON" />
        </Tabs>
        {view === 'json' ? (
          <Box sx={{ mt: 2 }}>
            <JsonField label="Widget" value={{ label: component.label, props: component.props, style: component.style, visibleWhen: component.visibleWhen }} minRows={16}
              onChange={(v) => { const x = (v ?? {}) as Partial<ComponentDefinition>; update({ label: x.label, props: x.props, style: x.style, visibleWhen: x.visibleWhen }); }} />
          </Box>
        ) : (
          <>
            {body}
            <Section title="Layout">
              <SelectField label="Width in its row" value={component.style?.flex === '0 0 auto' ? 'auto' : 'fill'}
                options={[{ value: 'fill', label: 'Fill remaining space' }, { value: 'auto', label: 'Fit content' }]}
                onChange={(v) => update({ style: { ...component.style, flex: v === 'auto' ? '0 0 auto' : '1' } })} />
            </Section>
            <Section title="Visibility">
              <ConditionEditor label="Show when" value={component.visibleWhen} onChange={(v) => update({ visibleWhen: v })} paths={paths} />
            </Section>
          </>
        )}
      </Box>
    </Paper>
  );
}
