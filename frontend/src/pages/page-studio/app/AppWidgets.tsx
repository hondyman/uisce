import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert, Autocomplete, Box, Button, Chip, CircularProgress, Collapse, IconButton, InputAdornment, LinearProgress, MenuItem, Paper, Stack, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, TextField, ToggleButton, ToggleButtonGroup, Tooltip, Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import SearchIcon from '@mui/icons-material/Search';
import ClearIcon from '@mui/icons-material/Clear';
import { CatalogErrorAlert } from '../../../features/message-catalog/parts';
import { getDomainComponent } from '../../../studio-core/components/registry';
import type { ComponentDefinition } from '../../../types/pageStudio';
import type { Action, Binding, CellSpec, ChipColor, ColumnDef, ConditionNode, TextSpec } from './appModel';
import { getPath, resolve, resolveAll, text, type Scope } from './bindings';
import { evaluateCondition, useCondition } from './conditions';
import { useAppRuntime, type QueryState } from './AppRuntime';
import { Cell } from './cells';
import { PageIcon } from './icons';
import { FormWidget, KeyValue, Timeline } from './moreWidgets';
import { Canvas, DEFAULT_CANVAS_PROPS } from './canvas';
import { Chat, DEFAULT_CHAT_PROPS } from './chat';
export type { FormWidgetProps, KeyValueProps, TimelineProps } from './moreWidgets';
export type { CanvasProps } from './canvas';
export type { ChatProps } from './chat';

/** Component types rendered by this module (the page application widgets). */
export const APP_WIDGET_TYPES = ['PageHeader', 'VariableSelect', 'SearchInput', 'ActionButton', 'DataGrid', 'AlertBanner', 'DomainComponent', 'KeyValue', 'Timeline', 'Form', 'TextBlock', 'Canvas', 'Chat', 'FacetFilter', 'ProgressBar'] as const;
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
  KeyValue: { source: '', items: [{ label: 'Label', value: '{{data.field}}' }], columns: 2 },
  Timeline: { query: '', title: '{{row.title}}', subtitle: '', time: '{{row.at}}', onItemClick: [] },
  Form: { variable: 'draft', fields: [{ name: 'name', label: 'Name', kind: 'text', required: true }], columns: 1 },
  TextBlock: { text: 'Section title', variant: 'subtitle2' },
  Canvas: DEFAULT_CANVAS_PROPS as unknown as Record<string, unknown>,
  Chat: DEFAULT_CHAT_PROPS as unknown as Record<string, unknown>,
  FacetFilter: { variable: '', label: 'Filter', allLabel: 'All', options: [] },
  ProgressBar: { variable: '', label: 'Processing...', showSignal: true },
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
  /** toggle: a row of toggle buttons (Values | Side by side) instead of a dropdown. */
  variant?: 'select' | 'toggle';
}
export interface SearchInputProps {
  variable: string;
  placeholder?: TextSpec;
  debounceMs?: number;
  maxWidth?: number;
  /** search (default, with an icon), plain (a labelled text field), title, date (a date filter), or typeahead (autocomplete with suggestions). */
  variant?: 'search' | 'plain' | 'title' | 'date' | 'typeahead';
  label?: TextSpec;
  /** After the value is committed (e.g. mark the page unsaved), with {{value}}. */
  onChange?: Action[];
  /** Static options or path to dynamic items array (e.g. 'queries.scorecard.data.substitution_matrix') */
  optionsFrom?: string;
  /** If optionsFrom is an array of objects, the field to pull options from (e.g. 'attribute_code') */
  optionsField?: string;
  options?: { value: string; label?: TextSpec }[];
}
export interface FacetFilterProps {
  variable: string;
  label?: TextSpec;
  allLabel?: TextSpec;
  query?: string;
  rowsPath?: string;
  facetField?: string;
  options?: { value: string; label: TextSpec; color?: ChipColor }[];
  colorMap?: Record<string, ChipColor>;
}
export interface ProgressBarProps {
  variable?: string;
  label?: TextSpec;
  progress?: number;
  showSignal?: boolean;
  color?: 'primary' | 'secondary' | 'success' | 'warning' | 'info';
  height?: number;
}
export interface ActionButtonProps {
  label: TextSpec;
  icon?: string;
  /** chip: a small clickable status chip (colour from chipColor). */
  variant?: 'text' | 'outlined' | 'contained' | 'chip';
  color?: 'primary' | 'secondary' | 'inherit' | 'error' | 'success';
  /** chip: its colour, a binding resolving to default|primary|success|warning|error|info. */
  chipColor?: Binding;
  disabledWhen?: ConditionNode;
  /** Hover text (why it is disabled, what it does). */
  tooltip?: TextSpec;
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
  /**
   * Expandable detail under a row (the values that competed for an attribute):
   * a nested table over `rows` (resolved with {{row}}), shown for rows where `when` holds.
   */
  /** An expandable detail under a row: text (in an alert when severity is set) and/or a nested table of rows. */
  rowDetail?: { rows?: Binding; columns?: ColumnDef[]; text?: TextSpec; severity?: 'error' | 'warning' | 'info'; when?: ConditionNode; emptyText?: TextSpec };
  /**
   * Columns generated from data (one per source): for each entry of `from`
   * ({{col}}), a column headed `header` whose cell sees {{col}} and {{value}} =
   * row[valuePath][col[idField]]. Inserted at `insertAt` (default: after the static columns).
   */
  dynamicColumns?: { from: Binding; idField?: string; header: TextSpec; valuePath?: string; cell: CellSpec; insertAt?: number; minWidth?: number };
  /** Freeze the first column while scrolling sideways (wide matrices). */
  stickyFirstColumn?: boolean;
  /** Client-side search: page variable name to match against row fields */
  searchVariable?: string;
  /** Limit search to these fields (default: all string/number fields) */
  searchFields?: string[];
  /** Client-side filters: map of rowField -> variableName (e.g. { vendor_id: 'vendor_filter', tier: 'tier_filter' }) */
  filters?: Record<string, string>;
  /** Enable built-in table search toolbar directly above the table */
  enableSearch?: boolean;
  /** Search variant: 'typeahead' (MUI Autocomplete) or 'text' (standard TextField) */
  searchVariant?: 'typeahead' | 'text';
  /** Placeholder for built-in table search */
  searchPlaceholder?: TextSpec;
  /** Field to extract typeahead options from rows (e.g. attribute_code, vendor_name) */
  searchOptionsField?: string;
  /** Max width of the search toolbar input (default: 400) */
  searchWidth?: number | string;
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
  /** A button in the alert (Back to latest). */
  action?: { label: TextSpec; onClick: Action[] };
}
export interface TextBlockProps {
  text: TextSpec;
  variant?: 'h4' | 'h5' | 'h6' | 'subtitle1' | 'subtitle2' | 'body1' | 'body2' | 'caption' | 'overline';
  color?: 'text.primary' | 'text.secondary' | 'primary.main' | 'error.main' | 'warning.main' | 'success.main' | 'info.main';
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
  const choose = (v: string) => { setVariable(p.variable, v); void runActions(p.onChange, { ...scope, value: v }); };
  if (p.variant === 'toggle') {
    return (
      <ToggleButtonGroup size="small" exclusive value={shown} disabled={disabled} onChange={(_, v) => v !== null && choose(v)} aria-label={p.label ? text(p.label, scope) : undefined}>
        {options.map((o) => <ToggleButton key={o.value} value={o.value} sx={{ textTransform: 'none' }}>{o.label}</ToggleButton>)}
      </ToggleButtonGroup>
    );
  }
  return (
    <TextField select size="small" sx={{ minWidth: p.minWidth ?? 180 }} label={p.label ? text(p.label, scope) : undefined} value={shown} disabled={disabled}
      SelectProps={{ displayEmpty: hasEmpty }} InputLabelProps={hasEmpty ? { shrink: true } : undefined}
      onChange={(e) => choose(e.target.value)}>
      {p.emptyLabel !== undefined && <MenuItem value="">{text(p.emptyLabel, scope)}</MenuItem>}
      {options.map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
    </TextField>
  );
}

function SearchInput({ p, scope }: { p: SearchInputProps; scope: Scope }) {
  const { setVariable, runActions } = useAppRuntime();
  const raw = getPath(scope, `vars.${p.variable}`);
  const committed = raw === null || raw === undefined ? '' : String(raw);
  const [draft, setDraft] = useState(committed);
  const sent = useRef(committed);
  // The variable set from elsewhere (a record loaded) shows here.
  useEffect(() => {
    if (committed !== sent.current) { sent.current = committed; setDraft(committed); }
  }, [committed]);
  useEffect(() => {
    if (draft === sent.current) return;
    const h = setTimeout(() => {
      sent.current = draft;
      setVariable(p.variable, draft);
      if (p.onChange?.length) void runActions(p.onChange, { value: draft });
    }, p.debounceMs ?? 300);
    return () => clearTimeout(h);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft, p.variable, p.debounceMs, setVariable]);
  const label = p.label ? text(p.label, scope) : undefined;

  const typeaheadOptions = useMemo<string[]>(() => {
    if (p.variant !== 'typeahead') return [];
    if (Array.isArray(p.options) && p.options.length > 0) {
      return p.options.map((o) => o.value);
    }
    if (p.optionsFrom) {
      const source = getPath(scope, p.optionsFrom);
      if (Array.isArray(source)) {
        const seen = new Set<string>();
        for (const item of source) {
          if (!item) continue;
          const val = p.optionsField && typeof item === 'object'
            ? String((item as Record<string, unknown>)[p.optionsField] ?? '').trim()
            : String(item).trim();
          if (val) seen.add(val);
        }
        return Array.from(seen).sort();
      }
    }
    return [];
  }, [p.variant, p.options, p.optionsFrom, p.optionsField, scope]);

  if (p.variant === 'date') {
    // A date filter: committed as picked (YYYY-MM-DD).
    return (
      <TextField size="small" type="date" label={label} value={committed} InputLabelProps={{ shrink: true }}
        onChange={(e) => { sent.current = e.target.value; setDraft(e.target.value); setVariable(p.variable, e.target.value); if (p.onChange?.length) void runActions(p.onChange, { value: e.target.value }); }} />
    );
  }
  if (p.variant === 'title') {
    return (
      <TextField variant="standard" value={draft} onChange={(e) => setDraft(e.target.value)} placeholder={p.placeholder ? text(p.placeholder, scope) : undefined}
        inputProps={{ 'aria-label': label ?? p.variable, style: { fontSize: 20, fontWeight: 600 } }} sx={{ minWidth: 220, flex: '1 1 260px', maxWidth: p.maxWidth ?? 480 }} />
    );
  }

  if (p.variant === 'typeahead') {
    return (
      <Autocomplete
        freeSolo
        size="small"
        sx={{ flex: 1, maxWidth: p.maxWidth ?? 420, minWidth: 220 }}
        options={typeaheadOptions}
        value={draft}
        onInputChange={(_event, newInputValue) => {
          setDraft(newInputValue);
        }}
        renderInput={(params) => (
          <TextField
            {...params}
            label={label}
            placeholder={p.placeholder ? text(p.placeholder, scope) : undefined}
            InputProps={{
              ...params.InputProps,
              startAdornment: (
                <>
                  <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment>
                  {params.InputProps.startAdornment}
                </>
              ),
            }}
          />
        )}
      />
    );
  }

  return (
    <TextField size="small" sx={{ flex: 1, maxWidth: p.maxWidth ?? 420 }} label={label} placeholder={p.placeholder ? text(p.placeholder, scope) : undefined} value={draft}
      onChange={(e) => setDraft(e.target.value)}
      InputProps={p.variant === 'plain' ? undefined : { startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }} />
  );
}

function FacetFilter({ p, scope }: { p: FacetFilterProps; scope: Scope }) {
  const { setVariable } = useAppRuntime();
  const currentVal = String(getPath(scope, `vars.${p.variable}`) ?? '');

  // Calculate item counts if query and facetField are given
  const counts = useMemo<Record<string, number>>(() => {
    if (!p.query || !p.facetField) return {};
    const q = getPath(scope, `queries.${p.query}`) as QueryState | undefined;
    const raw = p.rowsPath ? getPath(q?.data, p.rowsPath) : q?.data;
    if (!Array.isArray(raw)) return {};
    const res: Record<string, number> = {};
    for (const item of raw) {
      if (item && typeof item === 'object') {
        const val = String((item as Record<string, unknown>)[p.facetField] ?? '');
        if (val) {
          res[val] = (res[val] || 0) + 1;
        }
      }
    }
    return res;
  }, [p.query, p.rowsPath, p.facetField, scope]);

  const totalCount = useMemo(() => {
    return Object.values(counts).reduce((sum, n) => sum + n, 0);
  }, [counts]);

  const allLabelText = p.allLabel ? text(p.allLabel, scope) : 'All';

  return (
    <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
      {p.label && <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, mr: 0.5 }}>{text(p.label, scope)}:</Typography>}
      <Chip
        size="small"
        label={totalCount > 0 ? `${allLabelText} (${totalCount})` : allLabelText}
        variant={currentVal === '' ? 'filled' : 'outlined'}
        color={currentVal === '' ? 'primary' : 'default'}
        onClick={() => setVariable(p.variable, '')}
        sx={{ cursor: 'pointer', fontWeight: currentVal === '' ? 600 : 400 }}
      />
      {(p.options ?? []).map((opt) => {
        const selected = currentVal.toLowerCase() === opt.value.toLowerCase();
        const cnt = counts[opt.value];
        const lbl = text(opt.label, scope);
        const labelWithCount = cnt !== undefined ? `${lbl} (${cnt})` : lbl;
        const color = opt.color ?? (p.colorMap?.[opt.value] as ChipColor) ?? 'default';

        return (
          <Chip
            key={opt.value}
            size="small"
            label={labelWithCount}
            variant={selected ? 'filled' : 'outlined'}
            color={selected ? (color !== 'default' ? color : 'primary') : 'default'}
            onClick={() => setVariable(p.variable, selected ? '' : opt.value)}
            sx={{ cursor: 'pointer', fontWeight: selected ? 600 : 400 }}
          />
        );
      })}
    </Box>
  );
}

function ProgressBarWidget({ p, scope }: { p: ProgressBarProps; scope: Scope }) {
  const rawMsg = p.variable ? getPath(scope, `vars.${p.variable}`) : null;
  const msg = rawMsg !== undefined && rawMsg !== null && String(rawMsg).trim() !== ''
    ? String(rawMsg)
    : p.label ? text(p.label, scope) : '';

  if (!msg && p.progress === undefined) {
    return null;
  }

  const isDeterminate = typeof p.progress === 'number' && !isNaN(p.progress);
  const color = p.color ?? 'primary';

  return (
    <Paper
      variant="outlined"
      sx={{
        p: 1.5,
        bgcolor: 'action.hover',
        borderColor: `${color}.light`,
        borderRadius: 1.5,
        position: 'relative',
        overflow: 'hidden',
        width: '100%',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1, gap: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {p.showSignal && (
            <Box
              sx={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                bgcolor: `${color}.main`,
                boxShadow: (theme) => `0 0 0 3px ${theme.palette[color]?.light || '#90caf9'}55`,
                animation: 'studioPulse 1.5s infinite ease-in-out',
                '@keyframes studioPulse': {
                  '0%': { transform: 'scale(0.95)', opacity: 0.7 },
                  '50%': { transform: 'scale(1.2)', opacity: 1 },
                  '100%': { transform: 'scale(0.95)', opacity: 0.7 },
                },
              }}
            />
          )}
          <Typography variant="body2" sx={{ fontWeight: 600, color: 'text.primary' }}>
            {msg}
          </Typography>
        </Box>
        {isDeterminate && (
          <Typography variant="caption" sx={{ fontWeight: 700, color: `${color}.main` }}>
            {Math.round(p.progress!)}%
          </Typography>
        )}
      </Box>
      <LinearProgress
        variant={isDeterminate ? 'determinate' : 'indeterminate'}
        value={isDeterminate ? p.progress : undefined}
        color={color}
        sx={{ height: p.height ?? 6, borderRadius: 3 }}
      />
    </Paper>
  );
}

function ActionButton({ p, scope }: { p: ActionButtonProps; scope: Scope }) {
  const { runActions, mode } = useAppRuntime();
  const gated = useCondition(p.disabledWhen, scope, true);
  const disabled = !!p.disabledWhen && gated;
  // Waits (spinner, disabled) while its actions run.
  const [busy, setBusy] = useState(false);
  const click = async () => {
    setBusy(true);
    try { await runActions(p.onClick, scope); } finally { setBusy(false); }
  };
  const label = text(p.label, scope);
  let control: React.ReactElement;
  if (p.variant === 'chip') {
    const c = String(resolve(p.chipColor ?? 'default', scope) ?? 'default') as ChipColor;
    control = <Chip size="small" color={c} label={label} onClick={mode === 'design' ? undefined : () => void click()} />;
  } else {
    control = (
      <Button variant={p.variant ?? 'contained'} color={p.color ?? 'primary'}
        startIcon={busy ? <CircularProgress size={16} /> : p.icon ? <PageIcon name={p.icon} /> : undefined}
        disabled={disabled || busy || mode === 'design'} onClick={() => void click()} sx={{ whiteSpace: 'nowrap' }}>
        {label}
      </Button>
    );
  }
  const tip = p.tooltip ? text(p.tooltip, scope) : '';
  // A disabled button fires no hover events; the span carries the tooltip.
  return tip ? <Tooltip title={tip}><span>{control}</span></Tooltip> : control;
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

/** A column made from data (DataGridProps.dynamicColumns): its cell sees {{col}} and {{value}}. */
type GridColumn = ColumnDef & { dyn?: { col: unknown; value: (row: unknown) => unknown } };

function useGridColumns(p: DataGridProps, scope: Scope): GridColumn[] {
  const staticCols = useVisibleColumns(p.columns ?? [], scope);
  const d = p.dynamicColumns;
  const from = d ? resolve(d.from, scope) : undefined;
  const fromKey = JSON.stringify(from ?? null);
  const dyn = useMemo<GridColumn[]>(() => {
    if (!d || !Array.isArray(from)) return [];
    return from.map((col) => {
      const id = String(d.idField ? getPath(col, d.idField) : col);
      return {
        id: `dyn_${id}`, header: text(d.header, { ...scope, col }), minWidth: d.minWidth, cell: d.cell,
        dyn: { col, value: (row: unknown) => getPath(d.valuePath ? getPath(row, d.valuePath) : row, id) },
      };
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fromKey, d]);
  if (!dyn.length) return staticCols;
  const at = Math.min(d?.insertAt ?? staticCols.length, staticCols.length);
  return [...staticCols.slice(0, at), ...dyn, ...staticCols.slice(at)];
}

function GridCell({ c, rowScope, first, sticky, extra }: { c: GridColumn; rowScope: Scope; first: boolean; sticky?: boolean; extra?: React.ReactNode }) {
  const s = c.dyn ? { ...rowScope, col: c.dyn.col, value: c.dyn.value(rowScope.row) } : rowScope;
  const tip = c.tooltip !== undefined ? resolve(c.tooltip, s) : undefined;
  const body = cellOf(c).map((spec, i) => <Cell key={i} spec={spec} scope={s} />);
  const content = c.stack
    ? <Stack direction={c.stackDirection ?? 'column'} spacing={0.5} useFlexGap flexWrap="wrap"
        alignItems={c.stackDirection === 'row' ? 'center' : c.align === 'right' ? 'flex-end' : 'flex-start'}>{body}</Stack>
    : body;
  return (
    <TableCell align={c.align} sx={{
      whiteSpace: c.nowrap ? 'nowrap' : undefined, minWidth: c.minWidth, verticalAlign: 'top',
      ...(first && sticky ? { position: 'sticky', left: 0, zIndex: 1, bgcolor: 'background.paper' } : {}),
    }}>
      {tip ? <Tooltip title={String(tip)} componentsProps={{ tooltip: { sx: { whiteSpace: 'pre-line' } } }}><span>{content}</span></Tooltip> : content}
      {extra}
    </TableCell>
  );
}

function RowDetail({ p, rowScope, span }: { p: DataGridProps; rowScope: Scope; span: number }) {
  const d = p.rowDetail!;
  const raw = resolve(d.rows, rowScope);
  const rows = Array.isArray(raw) ? raw : [];
  const cols = useVisibleColumns(d.columns ?? [], rowScope);
  const body = d.text ? text(d.text, rowScope) : '';
  return (
    <Box sx={{ py: 1.5 }}>
      {body && (d.severity
        ? <Alert severity={d.severity} sx={{ mb: d.rows !== undefined ? 1 : 0, whiteSpace: 'pre-line' }}>{body}</Alert>
        : <Typography variant="body2" sx={{ mb: d.rows !== undefined ? 1 : 0, whiteSpace: 'pre-line' }}>{body}</Typography>)}
      {/* Text-only detail (a failed run's reason) has no table. */}
      {d.rows !== undefined && (
      <Table size="small">
        <TableHead>
          <TableRow>{cols.map((c) => <TableCell key={c.id} align={c.align}>{c.header ? text(c.header, rowScope) : ''}</TableCell>)}</TableRow>
        </TableHead>
        <TableBody>
          {rows.map((r, i) => (
            <TableRow key={i}>
              {cols.map((c, j) => <GridCell key={c.id} c={c} rowScope={{ ...rowScope, row: r, parent: rowScope.row, index: i }} first={j === 0} />)}
            </TableRow>
          ))}
          {rows.length === 0 && (
            <TableRow><TableCell colSpan={Math.max(1, cols.length || span)}>
              <Typography variant="body2" color="text.secondary">{text(d.emptyText ?? 'studioApp.empty', rowScope)}</Typography>
            </TableCell></TableRow>
          )}
        </TableBody>
      </Table>
      )}
    </Box>
  );
}

function GridRow({ row, index, p, scope, cols }: { row: unknown; index: number; p: DataGridProps; scope: Scope; cols: GridColumn[] }) {
  const { runActions } = useAppRuntime();
  const [rowState, setRowState] = useState<Record<string, unknown>>({});
  const [open, setOpen] = useState(false);
  const rowScope = useMemo<Scope>(() => ({
    ...scope, row, index, rowState,
    setRowState: (n: string, v: unknown) => setRowState((s) => ({ ...s, [n]: v })),
  }), [scope, row, index, rowState]);
  const expandable = useCondition(p.rowDetail?.when, rowScope, true);
  const hasDetail = !!p.rowDetail && (!p.rowDetail.when || expandable);
  const clickable = !!p.onRowClick?.length;
  const span = Math.max(1, cols.length);
  // The detail toggle sits at the end of the last column, beside any row buttons.
  const toggle = hasDetail ? (
    <IconButton size="small" aria-label={open ? 'Hide detail' : 'Show detail'} onClick={(e) => { e.stopPropagation(); setOpen(!open); }}>
      {open ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
    </IconButton>
  ) : undefined;
  return (
    <>
      <TableRow hover sx={clickable ? { cursor: 'pointer' } : undefined} onClick={clickable ? () => void runActions(p.onRowClick, rowScope) : undefined}>
        {cols.map((c, i) => <GridCell key={c.id} c={c} rowScope={rowScope} first={i === 0} sticky={p.stickyFirstColumn} extra={i === cols.length - 1 ? toggle : undefined} />)}
      </TableRow>
      {hasDetail && (
        <TableRow>
          <TableCell colSpan={span} sx={{ py: 0, borderBottom: open ? undefined : 0 }}>
            <Collapse in={open} unmountOnExit><RowDetail p={p} rowScope={rowScope} span={span} /></Collapse>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function DataGrid({ p, scope }: { p: DataGridProps; scope: Scope }) {
  const { setVariable } = useAppRuntime();
  const [localSearch, setLocalSearch] = useState('');

  const q = getPath(scope, `queries.${p.query}`) as QueryState | undefined;
  const data = q?.data;
  const rowsRaw = p.rowsPath ? getPath(data, p.rowsPath) : data;
  let rows = Array.isArray(rowsRaw) ? rowsRaw : [];

  // Effective search string (bound variable or internal local state)
  const boundSearch = p.searchVariable ? String(getPath(scope, `vars.${p.searchVariable}`) ?? '') : null;
  const currentSearch = boundSearch !== null ? boundSearch : localSearch;

  const handleSearchChange = (val: string) => {
    if (p.searchVariable) {
      setVariable(p.searchVariable, val);
    } else {
      setLocalSearch(val);
    }
  };

  // Typeahead options computed dynamically from rows
  const typeaheadOptions = useMemo<string[]>(() => {
    if (!Array.isArray(rowsRaw)) return [];
    const seen = new Set<string>();
    for (const row of rowsRaw) {
      if (!row || typeof row !== 'object') continue;
      const r = row as Record<string, unknown>;
      if (p.searchOptionsField && r[p.searchOptionsField] !== undefined) {
        const val = String(r[p.searchOptionsField]).trim();
        if (val) seen.add(val);
      } else if (p.searchFields && p.searchFields.length > 0) {
        for (const f of p.searchFields) {
          const val = String(r[f] ?? '').trim();
          if (val) seen.add(val);
        }
      } else {
        for (const val of Object.values(r)) {
          if (typeof val === 'string' && val.length > 0 && val.length < 50) {
            seen.add(val.trim());
          }
        }
      }
    }
    return Array.from(seen).sort();
  }, [rowsRaw, p.searchOptionsField, p.searchFields]);

  // Client-side search filtering
  const searchStr = currentSearch.trim().toLowerCase();
  if (searchStr) {
    rows = rows.filter((row: unknown) => {
      if (!row || typeof row !== 'object') return false;
      const r = row as Record<string, unknown>;
      if (p.searchFields && p.searchFields.length > 0) {
        return p.searchFields.some((f) => String(r[f] ?? '').toLowerCase().includes(searchStr));
      }
      return Object.values(r).some((val) =>
        typeof val === 'string' || typeof val === 'number' ? String(val).toLowerCase().includes(searchStr) : false
      );
    });
  }

  // Client-side variable filter mapping (e.g. { vendor_id: 'vendor_filter', tier: 'tier_filter' })
  if (p.filters) {
    for (const [rowField, varName] of Object.entries(p.filters)) {
      const filterVal = getPath(scope, `vars.${varName}`);
      if (filterVal !== undefined && filterVal !== null && String(filterVal).trim() !== '') {
        const strVal = String(filterVal).toLowerCase();
        rows = rows.filter((row: unknown) => {
          if (!row || typeof row !== 'object') return false;
          const val = (row as Record<string, unknown>)[rowField];
          if (val === undefined || val === null) return false;
          return String(val).toLowerCase() === strVal;
        });
      }
    }
  }

  // Column visibility is page-level (e.g. an actions column only for open items); some columns come from data.
  const cols = useGridColumns(p, scope);
  const busy = q ? (p.progressOnFetch ? q.isFetching : q.isLoading) : false;
  const span = Math.max(1, cols.length);
  return (
    <>
      {busy && <LinearProgress />}
      {!!q?.error && <Box sx={{ mb: 1 }}><CatalogErrorAlert error={q.error} /></Box>}
      {!q && <Alert severity="info">Choose a query for this grid.</Alert>}
      <TableContainer component={Paper} variant="outlined" sx={p.maxHeight ? { maxHeight: p.maxHeight } : undefined}>
        {p.enableSearch && (
          <Box sx={{ p: 1.25, px: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1.5, borderBottom: 1, borderColor: 'divider', bgcolor: 'action.hover' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: 1, maxWidth: p.searchWidth ?? 400 }}>
              {p.searchVariant === 'text' ? (
                <TextField
                  size="small"
                  fullWidth
                  placeholder={p.searchPlaceholder ? text(p.searchPlaceholder, scope) : 'Search table...'}
                  value={currentSearch}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchIcon fontSize="small" />
                      </InputAdornment>
                    ),
                    endAdornment: currentSearch ? (
                      <InputAdornment position="end">
                        <IconButton size="small" onClick={() => handleSearchChange('')} aria-label="Clear search">
                          <ClearIcon fontSize="small" />
                        </IconButton>
                      </InputAdornment>
                    ) : undefined,
                  }}
                />
              ) : (
                <Autocomplete
                  freeSolo
                  size="small"
                  fullWidth
                  options={typeaheadOptions}
                  value={currentSearch}
                  onInputChange={(_event, newInputValue) => {
                    handleSearchChange(newInputValue);
                  }}
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      placeholder={p.searchPlaceholder ? text(p.searchPlaceholder, scope) : 'Typeahead filter table...'}
                      InputProps={{
                        ...params.InputProps,
                        startAdornment: (
                          <>
                            <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment>
                            {params.InputProps.startAdornment}
                          </>
                        ),
                      }}
                    />
                  )}
                />
              )}
            </Box>
            <Typography variant="caption" color="text.secondary" sx={{ whiteSpace: 'nowrap', fontWeight: 500 }}>
              Showing {rows.length} {rows.length === 1 ? 'row' : 'rows'}
            </Typography>
          </Box>
        )}
        <Table size="small" stickyHeader={!!p.maxHeight}>
          <TableHead>
            <TableRow>
              {cols.map((c, i) => (
                <TableCell key={c.id} align={c.align} sx={{ minWidth: c.minWidth, ...(i === 0 && p.stickyFirstColumn ? { position: 'sticky', left: 0, zIndex: 3 } : {}) }}>
                  {c.header ? (c.dyn ? String(c.header) : text(c.header, scope)) : ''}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {(() => {
              const seen = new Set<string>();
              return rows.map((row, i) => {
                let k: string;
                if (p.rowKey && p.rowKey.includes(',')) {
                  k = p.rowKey
                    .split(',')
                    .map((f) => String(getPath(row, f.trim()) ?? ''))
                    .join('::');
                } else {
                  const raw = getPath(row, p.rowKey ?? 'id') ?? getPath(row, 'key');
                  k = raw !== undefined && raw !== null ? String(raw) : String(i);
                }
                if (!k || seen.has(k)) {
                  k = `${k || 'row'}__${i}`;
                }
                seen.add(k);
                return <GridRow key={k} row={row} index={i} p={p} scope={scope} cols={cols} />;
              });
            })()}
            {q && !q.isLoading && !q.error && rows.length === 0 && (
              <TableRow><TableCell colSpan={span}>
                <Typography color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>{text(p.emptyText ?? 'studioApp.empty', scope)}</Typography>
              </TableCell></TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
}

function TextBlockView({ p, scope }: { p: TextBlockProps; scope: Scope }) {
  const v = p.variant ?? 'body2';
  return <Typography variant={v} color={p.color} component={v.startsWith('h') || v.startsWith('subtitle') ? 'h3' : 'div'} sx={{ whiteSpace: 'pre-line' }}>{text(p.text, scope)}</Typography>;
}

function AlertBanner({ p, scope }: { p: AlertBannerProps; scope: Scope }) {
  const { runActions } = useAppRuntime();
  const raw = resolve(p.severity ?? 'info', scope);
  const sev = p.severityMap ? (p.severityMap[String(raw)] ?? p.severityMap['*'] ?? 'info') : (String(raw) as 'info');
  const detail = p.detail ? text(p.detail, scope) : '';
  return (
    <Alert severity={sev} onClose={p.onClose ? () => void runActions(p.onClose, scope) : undefined}
      action={p.action ? <Button color="inherit" size="small" onClick={() => void runActions(p.action!.onClick, scope)}>{text(p.action.label, scope)}</Button> : undefined}>
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
    case 'KeyValue': return <KeyValue p={p} scope={scope} />;
    case 'Timeline': return <Timeline p={p} scope={scope} />;
    case 'Form': return <FormWidget p={p} scope={scope} />;
    case 'TextBlock': return <TextBlockView p={p} scope={scope} />;
    case 'Canvas': return <Canvas p={p} scope={scope} />;
    case 'Chat': return <Chat p={p} scope={scope} />;
    case 'FacetFilter': return <FacetFilter p={p} scope={scope} />;
    case 'ProgressBar': return <ProgressBarWidget p={p} scope={scope} />;
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
