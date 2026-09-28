import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert, Box, Button, Chip, InputAdornment, LinearProgress, MenuItem, Paper, Stack, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, TextField, Tooltip, Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import { CatalogErrorAlert } from '../../../features/message-catalog/parts';
import { getDomainComponent } from '../../../studio-core/components/registry';
import type { ComponentDefinition } from '../../../types/pageStudio';
import type { Action, Binding, CellSpec, ChipColor, ColumnDef, ConditionNode, TextSpec } from './appModel';
import { getPath, resolve, resolveAll, text, type Scope } from './bindings';
import { evaluateCondition, useCondition } from './conditions';
import { useAppRuntime, type QueryState } from './AppRuntime';
import { Cell } from './cells';
import { PageIcon } from './icons';

/** Component types rendered by this module (the page application widgets). */
export const APP_WIDGET_TYPES = ['PageHeader', 'VariableSelect', 'SearchInput', 'ActionButton', 'DataGrid', 'AlertBanner', 'DomainComponent'] as const;
export type AppWidgetType = typeof APP_WIDGET_TYPES[number];
export const isAppWidget = (type: string): type is AppWidgetType => (APP_WIDGET_TYPES as readonly string[]).includes(type);

/** Props a freshly dropped widget starts with. */
export const APP_WIDGET_DEFAULTS: Record<AppWidgetType, Record<string, unknown>> = {
  PageHeader: { icon: 'dashboard', title: 'Page title', subtitle: '' },
  VariableSelect: { variable: '', label: 'Choose', options: [] },
  SearchInput: { variable: '', placeholder: 'Search' },
  ActionButton: { label: 'Action', variant: 'contained', onClick: [] },
  DataGrid: { query: '', columns: [], rowKey: 'id' },
  AlertBanner: { severity: 'info', text: 'Message' },
  DomainComponent: { component: '', inputs: {}, events: {} },
};

// --- Props per widget (component.props) ------------------------------------

export interface PageHeaderProps { icon?: string; title?: TextSpec; subtitle?: TextSpec }
export interface VariableSelectProps {
  variable: string;
  label?: TextSpec;
  options?: { value: string; label: TextSpec }[];
  /** Options from a query: rows at rowsPath, value/label read from each row (omit = the row itself). */
  optionsFrom?: { query: string; rowsPath?: string; valueField?: string; labelField?: string };
  /** Adds a leading "" option with this label (All). */
  emptyLabel?: TextSpec;
  minWidth?: number;
  /** Runs after the variable is set; {{value}} is the new value. */
  onChange?: Action[];
  disabledWhen?: ConditionNode;
}
export interface SearchInputProps { variable: string; placeholder?: TextSpec; debounceMs?: number; maxWidth?: number }
export interface ActionButtonProps {
  label: TextSpec;
  icon?: string;
  variant?: 'text' | 'outlined' | 'contained';
  color?: 'primary' | 'secondary' | 'inherit' | 'error';
  disabledWhen?: ConditionNode;
  onClick: Action[];
}
export interface DataGridProps {
  query: string;
  /** Path to the row array inside the query's data (omit when data is the array). */
  rowsPath?: string;
  rowKey?: string;
  columns: ColumnDef[];
  onRowClick?: Action[];
  emptyText?: TextSpec;
  /** Show progress while refetching (filters), not only on first load. */
  progressOnFetch?: boolean;
  maxHeight?: number;
}
export interface AlertBannerProps {
  severity?: Binding;
  /** Maps a resolved severity value (a run status) to success/info/warning/error; '*' is the fallback. */
  severityMap?: Record<string, 'success' | 'info' | 'warning' | 'error'>;
  text: TextSpec;
  chips?: { value: Binding; labelKey?: string; colorMap?: Record<string, ChipColor>; keys?: string[] };
  detail?: TextSpec;
  /** Makes the alert closable; runs these on close. */
  onClose?: Action[];
}
export interface DomainComponentProps {
  component: string;
  inputs?: Record<string, Binding>;
  events?: Record<string, Action[]>;
}

// --- Widgets ----------------------------------------------------------------

function PageHeader({ p, scope }: { p: PageHeaderProps; scope: Scope }) {
  return (
    <Stack direction="row" spacing={2} alignItems="center">
      {p.icon && <PageIcon name={p.icon} color="primary" fontSize="large" />}
      <Box>
        <Typography variant="h5" fontWeight={700}>{text(p.title, scope)}</Typography>
        {p.subtitle && <Typography color="text.secondary">{text(p.subtitle, scope)}</Typography>}
      </Box>
    </Stack>
  );
}

function VariableSelect({ p, scope }: { p: VariableSelectProps; scope: Scope }) {
  const { setVariable, runActions } = useAppRuntime();
  const gated = useCondition(p.disabledWhen, scope, false);
  const disabled = !!p.disabledWhen && gated;
  const current = getPath(scope, `vars.${p.variable}`);
  const options = useMemo(() => {
    if (p.optionsFrom) {
      const q = getPath(scope, `queries.${p.optionsFrom.query}.data`);
      const rows = (p.optionsFrom.rowsPath ? getPath(q, p.optionsFrom.rowsPath) : q) as unknown[] | undefined;
      return (Array.isArray(rows) ? rows : []).map((r) => ({
        value: String(p.optionsFrom!.valueField ? getPath(r, p.optionsFrom!.valueField) : r),
        label: String(p.optionsFrom!.labelField ? getPath(r, p.optionsFrom!.labelField) : p.optionsFrom!.valueField ? getPath(r, p.optionsFrom!.valueField) : r),
      }));
    }
    return (p.options ?? []).map((o) => ({ value: o.value, label: text(o.label, scope) }));
  }, [p.optionsFrom, p.options, scope]);
  const value = current === null || current === undefined ? '' : String(current);
  // A value not (yet) among the options would make MUI warn; show it empty until options load.
  const shown = value === '' || options.some((o) => o.value === value) ? value : '';
  const hasEmpty = p.emptyLabel !== undefined || options.some((o) => o.value === '');
  return (
    <TextField select size="small" sx={{ minWidth: p.minWidth ?? 180 }} label={p.label ? text(p.label, scope) : undefined} value={shown} disabled={disabled}
      SelectProps={{ displayEmpty: hasEmpty }} InputLabelProps={hasEmpty ? { shrink: true } : undefined}
      onChange={(e) => { setVariable(p.variable, e.target.value); void runActions(p.onChange, { ...scope, value: e.target.value }); }}>
      {p.emptyLabel !== undefined && <MenuItem value="">{text(p.emptyLabel, scope)}</MenuItem>}
      {options.map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
    </TextField>
  );
}

function SearchInput({ p, scope }: { p: SearchInputProps; scope: Scope }) {
  const { setVariable } = useAppRuntime();
  const committed = getPath(scope, `vars.${p.variable}`);
  const [draft, setDraft] = useState(committed === null || committed === undefined ? '' : String(committed));
  useEffect(() => {
    const h = setTimeout(() => setVariable(p.variable, draft), p.debounceMs ?? 300);
    return () => clearTimeout(h);
  }, [draft, p.variable, p.debounceMs, setVariable]);
  return (
    <TextField size="small" sx={{ flex: 1, maxWidth: p.maxWidth ?? 420 }} placeholder={p.placeholder ? text(p.placeholder, scope) : undefined} value={draft}
      onChange={(e) => setDraft(e.target.value)}
      InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }} />
  );
}

function ActionButton({ p, scope }: { p: ActionButtonProps; scope: Scope }) {
  const { runActions, mode } = useAppRuntime();
  const gated = useCondition(p.disabledWhen, scope, true);
  const disabled = !!p.disabledWhen && gated;
  return (
    <Button variant={p.variant ?? 'contained'} color={p.color ?? 'primary'} startIcon={p.icon ? <PageIcon name={p.icon} /> : undefined}
      disabled={disabled || mode === 'design'} onClick={() => void runActions(p.onClick, scope)} sx={{ whiteSpace: 'nowrap' }}>
      {text(p.label, scope)}
    </Button>
  );
}

/** Which gated columns show now - column conditions read page scope, not rows. */
function useVisibleColumns(cols: ColumnDef[], scope: Scope): ColumnDef[] {
  const [visible, setVisible] = useState<Record<string, boolean>>({});
  const gated = cols.filter((c) => c.visibleWhen);
  const key = JSON.stringify(gated.map((c) => [c.id, c.visibleWhen]));
  useEffect(() => {
    let live = true;
    void Promise.all(gated.map(async (c) => [c.id, await evaluateCondition(c.visibleWhen, scope)] as const))
      .then((pairs) => { if (live) setVisible(Object.fromEntries(pairs)); });
    return () => { live = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, scope]);
  return cols.filter((c) => !c.visibleWhen || visible[c.id]);
}

const cellOf = (c: ColumnDef): CellSpec[] =>
  c.stack ?? (c.cell ? [c.cell] : [{ kind: 'text', value: `{{row.${c.field ?? c.id}}}` }]);

function GridRow({ row, index, p, scope, visibleCols }: { row: unknown; index: number; p: DataGridProps; scope: Scope; visibleCols: ColumnDef[] }) {
  const { runActions } = useAppRuntime();
  const [rowState, setRowState] = useState<Record<string, unknown>>({});
  const rowScope = useMemo<Scope>(() => ({
    ...scope, row, index, rowState,
    setRowState: (n: string, v: unknown) => setRowState((s) => ({ ...s, [n]: v })),
  }), [scope, row, index, rowState]);
  const clickable = !!p.onRowClick?.length;
  return (
    <TableRow hover sx={clickable ? { cursor: 'pointer' } : undefined} onClick={clickable ? () => void runActions(p.onRowClick, rowScope) : undefined}>
      {visibleCols.map((c) => {
        const tip = c.tooltip !== undefined ? resolve(c.tooltip, rowScope) : undefined;
        const body = cellOf(c).map((spec, i) => <Cell key={i} spec={spec} scope={rowScope} />);
        const content = c.stack
          ? <Stack direction={c.stackDirection ?? 'column'} spacing={0.5} useFlexGap flexWrap="wrap"
              alignItems={c.stackDirection === 'row' ? 'center' : c.align === 'right' ? 'flex-end' : 'flex-start'}>{body}</Stack>
          : body;
        return (
          <TableCell key={c.id} align={c.align} sx={{ whiteSpace: c.nowrap ? 'nowrap' : undefined, minWidth: c.minWidth }}>
            {tip ? <Tooltip title={String(tip)} componentsProps={{ tooltip: { sx: { whiteSpace: 'pre-line' } } }}><span>{content}</span></Tooltip> : content}
          </TableCell>
        );
      })}
    </TableRow>
  );
}

function DataGrid({ p, scope }: { p: DataGridProps; scope: Scope }) {
  const q = getPath(scope, `queries.${p.query}`) as QueryState | undefined;
  const data = q?.data;
  const rowsRaw = p.rowsPath ? getPath(data, p.rowsPath) : data;
  const rows = Array.isArray(rowsRaw) ? rowsRaw : [];
  // Column visibility is page-level (e.g. an actions column only for open items).
  const visibleCols = useVisibleColumns(p.columns ?? [], scope);
  const busy = q ? (p.progressOnFetch ? q.isFetching : q.isLoading) : false;
  return (
    <>
      {busy && <LinearProgress />}
      {!!q?.error && <Box sx={{ mb: 1 }}><CatalogErrorAlert error={q.error} /></Box>}
      {!q && <Alert severity="info">Choose a query for this grid.</Alert>}
      <TableContainer component={Paper} variant="outlined" sx={p.maxHeight ? { maxHeight: p.maxHeight } : undefined}>
        <Table size="small" stickyHeader={!!p.maxHeight}>
          <TableHead>
            <TableRow>
              {visibleCols.map((c) => <TableCell key={c.id} align={c.align}>{c.header ? text(c.header, scope) : ''}</TableCell>)}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row, i) => (
              <GridRow key={String(getPath(row, p.rowKey ?? 'id') ?? i)} row={row} index={i} p={p} scope={scope} visibleCols={visibleCols} />
            ))}
            {q && !q.isLoading && !q.error && rows.length === 0 && (
              <TableRow><TableCell colSpan={Math.max(1, visibleCols.length)}>
                <Typography color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>{text(p.emptyText ?? 'studioApp.empty', scope)}</Typography>
              </TableCell></TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
}

function AlertBanner({ p, scope }: { p: AlertBannerProps; scope: Scope }) {
  const { runActions } = useAppRuntime();
  const raw = resolve(p.severity ?? 'info', scope);
  const sev = p.severityMap ? (p.severityMap[String(raw)] ?? p.severityMap['*'] ?? 'info') : (String(raw) as 'info');
  const detail = p.detail ? text(p.detail, scope) : '';
  return (
    <Alert severity={sev} onClose={p.onClose ? () => void runActions(p.onClose, scope) : undefined}>
      <Typography variant="body2" sx={{ mb: p.chips ? 1 : 0 }}>{text(p.text, scope)}</Typography>
      {p.chips && <Cell spec={{ kind: 'chips', ...p.chips }} scope={scope} />}
      {detail && <Typography variant="caption" component="p" sx={{ mt: 1 }}>{detail}</Typography>}
    </Alert>
  );
}

function DomainComponentView({ p, scope }: { p: DomainComponentProps; scope: Scope }) {
  const { runActions, mode } = useAppRuntime();
  const def = getDomainComponent(p.component);
  const inputs = useMemo(() => resolveAll(p.inputs, scope), [p.inputs, scope]);
  if (!def) return <Alert severity="warning">Unknown domain component: {p.component || '(none)'}</Alert>;
  if (mode === 'design' && def.overlay) {
    return <Chip size="small" variant="outlined" label={`${def.label} · overlay`} />;
  }
  const R = def.render;
  return <R inputs={inputs} emit={(event, payload) => void runActions(p.events?.[event], { ...scope, event: payload ?? {} })} />;
}

/** Renders one application widget against the page scope. */
export function AppWidget({ component }: { component: ComponentDefinition }) {
  const { scope } = useAppRuntime();
  const p = (component.props ?? {}) as never;
  switch (component.type as AppWidgetType) {
    case 'PageHeader': return <PageHeader p={p} scope={scope} />;
    case 'VariableSelect': return <VariableSelect p={p} scope={scope} />;
    case 'SearchInput': return <SearchInput p={p} scope={scope} />;
    case 'ActionButton': return <ActionButton p={p} scope={scope} />;
    case 'DataGrid': return <DataGrid p={p} scope={scope} />;
    case 'AlertBanner': return <AlertBanner p={p} scope={scope} />;
    case 'DomainComponent': return <DomainComponentView p={p} scope={scope} />;
    default: return null;
  }
}

/** Generic widget visibility (any widget type): hidden at runtime, dimmed in design. */
export function VisibleWhen({ when, children }: { when?: ConditionNode; children: React.ReactNode }) {
  const { scope, mode } = useAppRuntime();
  const visible = useCondition(when, scope, false);
  if (!when || visible) return <>{children}</>;
  if (mode === 'design') return <Box sx={{ opacity: 0.45 }} title="Hidden now: its condition is false">{children}</Box>;
  return null;
}
