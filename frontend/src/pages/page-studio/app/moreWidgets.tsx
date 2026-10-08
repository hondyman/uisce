import React, { useEffect, useRef } from 'react';
import { Alert, Box, Button, LinearProgress, List, ListItemButton, Paper, Stack, Typography } from '@mui/material';
import Editor from '@monaco-editor/react';
import { CatalogErrorAlert } from '../../../features/message-catalog/parts';
import type { Action, Binding, CellSpec, ChipColor, ConditionNode, FormFieldSpec, TextSpec } from './appModel';
import { getPath, resolve, text, type Scope } from './bindings';
import { useCondition } from './conditions';
import { useAppRuntime, type QueryState } from './AppRuntime';
import { Cell } from './cells';
import { FormFields, isBlank, requiredFilled } from './formFields';

/**
 * Record-display and editing widgets: KeyValue (label/value pairs, e.g. a
 * golden record's header), Timeline (versions, runs, events - selectable),
 * and Form (fields bound to a page variable, submitted by actions). With the
 * Drawer/Dialog/TabSet containers these build record drawers and editors by
 * configuration - see docs/page-studio-app-model.md.
 */

// --- KeyValue ------------------------------------------------------------------

export interface KeyValueProps {
  /** The record the items read ({{data.x}}). */
  source?: Binding;
  /** Fixed items; value is a binding (templates see {{data}}), or a cell for rich display. */
  items?: { label: TextSpec; value?: Binding; cell?: CellSpec; visibleWhen?: ConditionNode }[];
  /** Or pairs from data: a list of rows (labelField/valueField) or an object's entries. */
  pairsFrom?: { value: Binding; labelField?: string; valueField?: string };
  columns?: number;
  emptyText?: TextSpec;
}

const show = (v: unknown) => (v === undefined || v === null || v === '' ? '—' : typeof v === 'object' ? JSON.stringify(v) : String(v));

function KeyValueItem({ item, scope }: { item: NonNullable<KeyValueProps['items']>[number]; scope: Scope }) {
  const visible = useCondition(item.visibleWhen, scope, true);
  if (item.visibleWhen && !visible) return null;
  return (
    <Box sx={{ minWidth: 0 }}>
      <Typography variant="caption" color="text.secondary" component="div">{text(item.label, scope)}</Typography>
      {item.cell ? <Cell spec={item.cell} scope={scope} /> : <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>{show(resolve(item.value, scope))}</Typography>}
    </Box>
  );
}

export function KeyValue({ p, scope }: { p: KeyValueProps; scope: Scope }) {
  const data = p.source !== undefined ? resolve(p.source, scope) : undefined;
  const local: Scope = { ...scope, data, row: data };
  const grid = { display: 'grid', gridTemplateColumns: { xs: '1fr', sm: `repeat(${p.columns ?? 2}, minmax(0, 1fr))` }, gap: 2 };
  if (p.pairsFrom) {
    const v = resolve(p.pairsFrom.value, local);
    const pairs: [string, unknown][] = Array.isArray(v)
      ? v.map((r) => [show(p.pairsFrom!.labelField ? getPath(r, p.pairsFrom!.labelField) : r), p.pairsFrom!.valueField ? getPath(r, p.pairsFrom!.valueField) : r])
      : v && typeof v === 'object' ? Object.entries(v as Record<string, unknown>) : [];
    if (pairs.length === 0) return <Typography variant="body2" color="text.secondary">{text(p.emptyText ?? 'studioApp.empty', scope)}</Typography>;
    return (
      <Box sx={grid}>
        {pairs.map(([k, val]) => (
          <Box key={k} sx={{ minWidth: 0 }}>
            <Typography variant="caption" color="text.secondary" component="div">{k}</Typography>
            <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>{show(val)}</Typography>
          </Box>
        ))}
      </Box>
    );
  }
  if (p.source !== undefined && isBlank(data)) {
    return <Typography variant="body2" color="text.secondary">{text(p.emptyText ?? 'studioApp.empty', scope)}</Typography>;
  }
  return <Box sx={grid}>{(p.items ?? []).map((it, i) => <KeyValueItem key={i} item={it} scope={local} />)}</Box>;
}

// --- Timeline --------------------------------------------------------------------

export interface TimelineProps {
  /** Items from a query (rows at rowsPath) or from a binding. */
  query?: string;
  rowsPath?: string;
  items?: Binding;
  title: TextSpec;
  subtitle?: TextSpec;
  time?: Binding;
  chip?: { value: Binding; labelKey?: string; colorMap?: Record<string, ChipColor>; variant?: 'filled' | 'outlined' };
  /** Highlights an item (e.g. the version being viewed). */
  selectedWhen?: ConditionNode;
  onItemClick?: Action[];
  emptyText?: TextSpec;
  maxHeight?: number;
}

function TimelineItem({ p, row, scope }: { p: TimelineProps; row: unknown; scope: Scope }) {
  const { runActions } = useAppRuntime();
  const s: Scope = { ...scope, row };
  const selected = useCondition(p.selectedWhen, s, false) && !!p.selectedWhen;
  const body = (
    <Stack direction="row" spacing={1.5} alignItems="flex-start" sx={{ width: '100%', minWidth: 0 }}>
      <Box sx={{ width: 10, height: 10, mt: 0.75, borderRadius: '50%', flex: '0 0 auto', bgcolor: selected ? 'primary.main' : 'divider' }} />
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Stack direction="row" spacing={1} alignItems="center" useFlexGap flexWrap="wrap">
          <Typography variant="body2" fontWeight={selected ? 700 : 600}>{text(p.title, s)}</Typography>
          {p.chip && <Cell spec={{ kind: 'chip', value: p.chip.value, labelKey: p.chip.labelKey, colorMap: p.chip.colorMap, variant: p.chip.variant }} scope={s} />}
        </Stack>
        {p.subtitle && <Typography variant="caption" color="text.secondary" component="div">{text(p.subtitle, s)}</Typography>}
      </Box>
      {p.time !== undefined && <Box sx={{ flex: '0 0 auto' }}><Cell spec={{ kind: 'datetime', value: p.time, caption: true }} scope={s} /></Box>}
    </Stack>
  );
  if (!p.onItemClick?.length) return <Box sx={{ px: 1.5, py: 1 }}>{body}</Box>;
  return <ListItemButton selected={selected} onClick={() => void runActions(p.onItemClick, s)} sx={{ borderRadius: 1 }}>{body}</ListItemButton>;
}

export function Timeline({ p, scope }: { p: TimelineProps; scope: Scope }) {
  const q = p.query ? (getPath(scope, `queries.${p.query}`) as QueryState | undefined) : undefined;
  const raw = p.query ? (p.rowsPath ? getPath(q?.data, p.rowsPath) : q?.data) : resolve(p.items, scope);
  const rows = Array.isArray(raw) ? raw : [];
  return (
    <Paper variant="outlined" sx={{ p: 0.5, maxHeight: p.maxHeight, overflowY: p.maxHeight ? 'auto' : undefined }}>
      {q?.isLoading && <LinearProgress />}
      {!!q?.error && <CatalogErrorAlert error={q.error} />}
      {rows.length === 0 && !q?.isLoading && (
        <Typography variant="body2" color="text.secondary" sx={{ p: 2, textAlign: 'center' }}>{text(p.emptyText ?? 'studioApp.empty', scope)}</Typography>
      )}
      <List dense disablePadding>
        {rows.map((row, i) => <TimelineItem key={String(getPath(row, 'id') ?? i)} p={p} row={row} scope={scope} />)}
      </List>
    </Paper>
  );
}

// --- Form ------------------------------------------------------------------------

export interface FormWidgetProps {
  /** The page variable holding the values (an object: {{vars.<variable>.<field>}}). */
  variable: string;
  fields: FormFieldSpec[];
  /** Or fields from data - a binding to a list of field specs (an operation shapes them, e.g. from a table's columns). */
  fieldsFrom?: Binding;
  /** Reseed only when this changes (e.g. the selected step) - not every time initFrom does. */
  seedKey?: Binding;
  /** Seeds the values when it changes (e.g. the row being edited); field defaults fill the rest. Omitted, existing values are kept. */
  initFrom?: Binding;
  columns?: number;
  submitLabel?: TextSpec;
  /** Runs on submit with {{form}} = the values. */
  onSubmit?: Action[];
  submitDisabledWhen?: ConditionNode;
  /** Runs after the user changes a field, with {{form}} = the new values (e.g. clear a stale preview). */
  onChange?: Action[];
}

export function FormWidget({ p, scope }: { p: FormWidgetProps; scope: Scope }) {
  const { setVariable, runActions } = useAppRuntime();
  const values = ((scope.vars as Record<string, unknown>)?.[p.variable] ?? {}) as Record<string, unknown>;
  const generated = p.fieldsFrom !== undefined ? resolve(p.fieldsFrom, scope) : undefined;
  const fields = (Array.isArray(generated) ? generated : p.fields ?? []) as FormFieldSpec[];
  const init = p.initFrom !== undefined ? resolve(p.initFrom, scope) : undefined;
  // With a seed key the form reseeds when the key changes (once its start value is there), not on every change of it.
  const initKey = p.seedKey !== undefined
    ? `key:${JSON.stringify(resolve(p.seedKey, scope) ?? null)}${init === undefined || init === null ? ':pending' : ''}`
    : JSON.stringify(init ?? null);
  const seeded = useRef<string | null>(null);
  useEffect(() => {
    if (!p.variable || seeded.current === initKey) return;
    // A start value that goes away (its query refetching) keeps what was typed.
    if (seeded.current !== null && p.initFrom !== undefined && (init === undefined || init === null)) return;
    seeded.current = initKey;
    const base: Record<string, unknown> = {};
    for (const f of fields) {
      const d = f.default !== undefined ? resolve(f.default, scope) : undefined;
      if (d !== undefined) base[f.name] = d;
    }
    // Without a start value the form only fills its defaults in, so two forms can share a variable.
    setVariable(p.variable, p.initFrom === undefined
      ? { ...base, ...values }
      : { ...base, ...(init && typeof init === 'object' ? (init as Record<string, unknown>) : {}) });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [p.variable, initKey]);
  const blocked = useCondition(p.submitDisabledWhen, { ...scope, form: values }, false);
  if (!p.variable) return <Alert severity="info">Choose the variable this form edits.</Alert>;
  const ready = requiredFilled(fields, values) && !(p.submitDisabledWhen && blocked);
  return (
    <Stack spacing={2}>
      <FormFields fields={fields} values={values} scope={scope} columns={p.columns}
        onChange={(name, v) => {
          const next = { ...values, [name]: v };
          // Dependent fields start over when what they depend on changes.
          for (const f of fields) if (f.resetOn?.includes(name) && f.name !== name) delete next[f.name];
          setVariable(p.variable, next);
          if (p.onChange?.length) void runActions(p.onChange, { form: next });
        }} />
      {p.onSubmit && p.onSubmit.length > 0 && (
        <Box>
          <Button variant="contained" disabled={!ready} onClick={() => void runActions(p.onSubmit, { form: values })}>
            {text(p.submitLabel ?? 'studioApp.ok', scope)}
          </Button>
        </Box>
      )}
    </Stack>
  );
}

// --- TreeView -------------------------------------------------------------------

export interface TreeViewProps {
  /** Items from a query (rows at rowsPath) or from a binding. */
  query?: string;
  rowsPath?: string;
  items?: Binding;
  /** Property on each node providing its unique identifier (default: "id" or "nodeKey"). */
  idField?: string;
  /** Property or template for the node's label (default: "{{row.label}}" or "{{row.name}}"). */
  label?: TextSpec;
  /** Property containing child nodes array (default: "children"). */
  childrenField?: string;
  /** Variable to store selected node or node id. */
  selectedVariable?: string;
  /** If true, stores the entire selected node object instead of just its id. */
  selectFullNode?: boolean;
  onNodeSelect?: Action[];
  emptyText?: TextSpec;
  maxHeight?: number;
}

export function TreeViewWidget({ p, scope }: { p: TreeViewProps; scope: Scope }) {
  const { setVariable, runActions } = useAppRuntime();
  const q = p.query ? (getPath(scope, `queries.${p.query}`) as QueryState | undefined) : undefined;
  const raw = p.query ? (p.rowsPath ? getPath(q?.data, p.rowsPath) : q?.data) : resolve(p.items, scope);
  const rootNodes = Array.isArray(raw) ? raw : [];

  const idKey = p.idField || 'id';
  const childKey = p.childrenField || 'children';
  const selectedVal = p.selectedVariable ? (getPath(scope, `vars.${p.selectedVariable}`) as unknown) : undefined;
  const selectedItemId = typeof selectedVal === 'object' && selectedVal !== null
    ? String(getPath(selectedVal, idKey) ?? '')
    : (selectedVal ? String(selectedVal) : null);

  const renderNode = (node: Record<string, unknown>, index: number) => {
    const rawId = getPath(node, idKey) ?? getPath(node, 'nodeKey') ?? getPath(node, 'key') ?? index;
    const itemId = String(rawId);
    const nodeScope: Scope = { ...scope, row: node, node };
    const labelStr = p.label ? text(p.label, nodeScope) : String(getPath(node, 'label') ?? getPath(node, 'name') ?? itemId);
    const children = getPath(node, childKey);
    const childArr = Array.isArray(children) ? (children as Record<string, unknown>[]) : [];

    return (
      <Box
        key={itemId}
        onClick={(e) => {
          e.stopPropagation();
          if (p.selectedVariable) {
            setVariable(p.selectedVariable, p.selectFullNode ? node : itemId);
          }
          if (p.onNodeSelect?.length) {
            void runActions(p.onNodeSelect, { ...nodeScope, selectedId: itemId, selectedNode: node });
          }
        }}
        sx={{
          py: 0.5,
          px: 1,
          borderRadius: 1,
          cursor: 'pointer',
          bgcolor: selectedItemId === itemId ? 'action.selected' : 'transparent',
          '&:hover': { bgcolor: 'action.hover' },
        }}
      >
        <Typography variant="body2" fontWeight={selectedItemId === itemId ? 600 : 400}>
          {labelStr}
        </Typography>
        {childArr.length > 0 && (
          <Box sx={{ pl: 2, borderLeft: '1px solid', borderColor: 'divider', mt: 0.5 }}>
            {childArr.map((child, i) => renderNode(child, i))}
          </Box>
        )}
      </Box>
    );
  };

  return (
    <Paper variant="outlined" sx={{ p: 1, maxHeight: p.maxHeight ?? 400, overflowY: 'auto' }}>
      {q?.isLoading && <LinearProgress sx={{ mb: 1 }} />}
      {!!q?.error && <CatalogErrorAlert error={q.error} />}
      {rootNodes.length === 0 && !q?.isLoading && (
        <Typography variant="body2" color="text.secondary" sx={{ p: 2, textAlign: 'center' }}>
          {text(p.emptyText ?? 'studioApp.empty', scope)}
        </Typography>
      )}
      <Box>{rootNodes.map((node, i) => renderNode(node as Record<string, unknown>, i))}</Box>
    </Paper>
  );
}

// --- CodeEditor -----------------------------------------------------------------

export interface CodeEditorProps {
  /** Page variable holding the code string. */
  variable?: string;
  /** Language syntax (e.g. 'sql', 'json', 'yaml', 'typescript', 'python'). Default: 'sql'. */
  language?: string;
  /** Editor height (number in px or CSS string). Default: 300. */
  height?: number | string;
  /** Editor theme ('vs-dark' | 'light'). Default: 'vs-dark'. */
  theme?: string;
  /** Whether editor is read-only. */
  readOnly?: boolean;
  /** Initial code from a binding if variable is unseeded. */
  initFrom?: Binding;
  onChange?: Action[];
}

export function CodeEditorWidget({ p, scope }: { p: CodeEditorProps; scope: Scope }) {
  const { setVariable, runActions } = useAppRuntime();
  const val = p.variable ? (getPath(scope, `vars.${p.variable}`) as unknown) : undefined;
  const initial = p.initFrom !== undefined ? resolve(p.initFrom, scope) : undefined;
  const codeValue = typeof val === 'string' ? val : (typeof initial === 'string' ? initial : '');

  const handleChange = (newValue: string | undefined) => {
    const textVal = newValue ?? '';
    if (p.variable) {
      setVariable(p.variable, textVal);
    }
    if (p.onChange?.length) {
      void runActions(p.onChange, { ...scope, code: textVal });
    }
  };

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden', borderRadius: 1 }}>
      <Editor
        height={p.height ?? 300}
        language={p.language || 'sql'}
        theme={p.theme || 'vs-dark'}
        value={codeValue}
        onChange={handleChange}
        options={{
          readOnly: !!p.readOnly,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          fontSize: 13,
          tabSize: 2,
          automaticLayout: true,
        }}
      />
    </Paper>
  );
}


