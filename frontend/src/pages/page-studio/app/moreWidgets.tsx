import React, { useEffect, useRef } from 'react';
import { Alert, Box, Button, LinearProgress, List, ListItemButton, Paper, Stack, Typography } from '@mui/material';
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
  /** Seeds the values when it changes (e.g. the row being edited); field defaults fill the rest. */
  initFrom?: Binding;
  columns?: number;
  submitLabel?: TextSpec;
  /** Runs on submit with {{form}} = the values. */
  onSubmit?: Action[];
  submitDisabledWhen?: ConditionNode;
}

export function FormWidget({ p, scope }: { p: FormWidgetProps; scope: Scope }) {
  const { setVariable, runActions } = useAppRuntime();
  const values = ((scope.vars as Record<string, unknown>)?.[p.variable] ?? {}) as Record<string, unknown>;
  const init = p.initFrom !== undefined ? resolve(p.initFrom, scope) : undefined;
  const initKey = JSON.stringify(init ?? null);
  const seeded = useRef<string | null>(null);
  useEffect(() => {
    if (!p.variable || seeded.current === initKey) return;
    seeded.current = initKey;
    const base: Record<string, unknown> = {};
    for (const f of p.fields ?? []) {
      const d = f.default !== undefined ? resolve(f.default, scope) : undefined;
      if (d !== undefined) base[f.name] = d;
    }
    setVariable(p.variable, { ...base, ...(init && typeof init === 'object' ? (init as Record<string, unknown>) : {}) });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [p.variable, initKey]);
  const blocked = useCondition(p.submitDisabledWhen, { ...scope, form: values }, false);
  if (!p.variable) return <Alert severity="info">Choose the variable this form edits.</Alert>;
  const ready = requiredFilled(p.fields ?? [], values) && !(p.submitDisabledWhen && blocked);
  return (
    <Stack spacing={2}>
      <FormFields fields={p.fields ?? []} values={values} scope={scope} columns={p.columns}
        onChange={(name, v) => setVariable(p.variable, { ...values, [name]: v })} />
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
