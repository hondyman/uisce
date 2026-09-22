/**
 * QueryShelves - "Query Definition Shelves": three independently
 * collapsible accordion cards (Selected Fields & Measures, Filters, Group
 * By & Ordering) matching the reference workbench layout. Dimensions/
 * measures read straight from useBusinessObjectSelector (removing a chip
 * here is the same as un-checking the field in the catalog); filters are
 * a separate, page-owned list (the selector hook has no concept of
 * filters) passed in as props, editable inline via a small popover.
 * Group By is read-only here - SavedQueryState has no explicit orderBy or
 * group-by-override concept yet, so this shows the derived default
 * (group by every selected dimension) rather than fabricating editable
 * controls for something the backend doesn't support.
 */
import React, { useMemo, useState } from 'react';
import {
  Box, Paper, Typography, Chip, Stack, Button, Popover,
  Select, MenuItem, TextField, ToggleButtonGroup, ToggleButton, FormControlLabel, Checkbox,
} from '@mui/material';
import ViewColumnIcon from '@mui/icons-material/ViewColumn';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import SchemaIcon from '@mui/icons-material/Schema';
import TuneIcon from '@mui/icons-material/Tune';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import LockIcon from '@mui/icons-material/Lock';
import AddIcon from '@mui/icons-material/Add';
import type { BusinessObjectSelector } from './BusinessObjectSelectorControl';
import type { FilterOperator, SavedQueryParameter } from '../../features/query-builder/types/queryDef';
import { looksLikeUuid } from '../../studio-core/binding/businessObjectApi';

/** Qualifies a field name with its object's name only when that's actually
 * informative - never for the primary object (redundant, it's implied),
 * and never with a raw id (see looksLikeUuid) while the name is still
 * resolving. */
function qualifiedFieldLabel(boName: string, isPrimary: boolean, termKey: string): string {
  if (isPrimary || looksLikeUuid(boName)) return termKey;
  return `${boName}.${termKey}`;
}

export interface EditableFilter {
  termNodeId: string;
  boId?: string;
  operator: FilterOperator;
  value: string;
  /** When set, this filter's value is resolved at run time from a named
   * parameter (see QueryShelvesProps.parameters) instead of the literal
   * `value` above - editable here just like the literal-value case, via
   * the Literal/Parameter toggle in FilterChip's popover. */
  paramRef?: string;
}

const OPERATOR_LABEL: Record<FilterOperator, string> = {
  eq: '=', neq: '≠', gt: '>', gte: '≥', lt: '<', lte: '≤',
  contains: 'contains', starts_with: 'starts with', ends_with: 'ends with',
  in: 'in', not_in: 'not in', is_null: 'is null', is_not_null: 'is not null', between: 'between',
};
const NO_VALUE_OPERATORS: FilterOperator[] = ['is_null', 'is_not_null'];

export interface QueryShelvesProps {
  selector: BusinessObjectSelector;
  filters: EditableFilter[];
  onUpdateFilter: (index: number, patch: Partial<EditableFilter>) => void;
  onRemoveFilter: (index: number) => void;
  /** Named parameters a filter can bind to instead of a literal value -
   * the same paramRef mechanism Report Builder's FilterBuilderPanel uses
   * ("map a query parameter" instead of hardcoding a value), standardized
   * here rather than reinvented: a filter with paramRef set is resolved
   * from `?<paramName>=value` at run time (see saved_query_handler.go's
   * resolveParams), same contract on both sides of the app. */
  parameters: SavedQueryParameter[];
  onAddParameter: (param: SavedQueryParameter) => void;
  onRemoveParameter: (name: string) => void;
}

export default function QueryShelves({
  selector, filters, onUpdateFilter, onRemoveFilter, parameters, onAddParameter, onRemoveParameter,
}: QueryShelvesProps) {
  const { objects, toggleField } = selector;
  const [openFields, setOpenFields] = useState(true);
  const [openFilters, setOpenFilters] = useState(true);
  const [openGroup, setOpenGroup] = useState(true);
  const allOpen = openFields && openFilters && openGroup;

  const dimensions = useMemo(() => objects.flatMap((o) =>
    o.terms.filter((t) => t.selected && t.role !== 'MEASURE').map((t) => ({ ...t, boId: o.boId, boName: o.boName, isPrimary: o.isPrimary }))), [objects]);
  const measures = useMemo(() => objects.flatMap((o) =>
    o.terms.filter((t) => t.selected && t.role === 'MEASURE').map((t) => ({ ...t, boId: o.boId, boName: o.boName, isPrimary: o.isPrimary }))), [objects]);

  const toggleAll = () => {
    const next = !allOpen;
    setOpenFields(next); setOpenFilters(next); setOpenGroup(next);
  };

  return (
    <Paper variant="outlined" sx={{ p: 1.5, bgcolor: 'background.default' }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
        <Stack direction="row" spacing={0.75} alignItems="center">
          <TuneIcon sx={{ fontSize: 18 }} color="secondary" />
          <Typography variant="overline" sx={{ fontSize: '0.72rem' }}>Query Definition Shelves</Typography>
        </Stack>
        <Button size="small" onClick={toggleAll} sx={{ fontSize: '0.68rem', color: 'text.secondary' }}>
          {allOpen ? 'Collapse All' : 'Expand All'}
        </Button>
      </Stack>

      <ShelfAccordion
        open={openFields} onToggle={() => setOpenFields((v) => !v)}
        icon={<ViewColumnIcon sx={{ fontSize: 16 }} color="secondary" />}
        title="Selected Fields & Measures"
        summary={<>
          <Chip size="small" color="secondary" variant="outlined" label={`${dimensions.length} Dim`} sx={{ height: 18, fontSize: '0.62rem' }} />
          <Chip size="small" color="warning" variant="outlined" label={`${measures.length} Mes`} sx={{ height: 18, fontSize: '0.62rem' }} />
        </>}
      >
        <Stack spacing={1}>
          <ShelfRow label="Dimensions:" color="secondary.main" empty="Add a dimension from the catalog">
            {dimensions.map((d) => (
              <Chip key={`${d.boId}:${d.termNodeId}`} size="small" color="secondary" variant="outlined"
                label={qualifiedFieldLabel(d.boName, d.isPrimary, d.termKey)}
                onDelete={() => toggleField(d.boId, d.termNodeId)} />
            ))}
          </ShelfRow>
          <ShelfRow label="Measures:" color="warning.main" empty="Add a measure from the catalog">
            {measures.map((m) => (
              <Chip key={`${m.boId}:${m.termNodeId}`} size="small" color="warning" variant="outlined"
                label={`${m.defaultAggregation && m.defaultAggregation !== 'NONE' ? `${m.defaultAggregation}(` : ''}${qualifiedFieldLabel(m.boName, m.isPrimary, m.termKey)}${m.defaultAggregation && m.defaultAggregation !== 'NONE' ? ')' : ''}`}
                onDelete={() => toggleField(m.boId, m.termNodeId)} />
            ))}
          </ShelfRow>
        </Stack>
      </ShelfAccordion>

      <ShelfAccordion
        open={openFilters} onToggle={() => setOpenFilters((v) => !v)}
        icon={<FilterAltIcon sx={{ fontSize: 16 }} color="primary" />}
        title="Filters"
        summary={<Chip size="small" color="primary" variant="outlined" label={`${filters.length} Active`} sx={{ height: 18, fontSize: '0.62rem' }} />}
      >
        <Stack spacing={1.25}>
          <Stack direction="row" alignItems="center" spacing={0.75} flexWrap="wrap" useFlexGap>
            {filters.length === 0 && <Typography variant="caption" color="text.disabled">Add a filter from the catalog</Typography>}
            {filters.map((f, i) => {
              const owner = objects.find((o) => (f.boId ? o.boId === f.boId : o.isPrimary));
              const term = owner?.terms.find((t) => t.termNodeId === f.termNodeId);
              const label = term ? qualifiedFieldLabel(owner?.boName || '', !!owner?.isPrimary, term.termKey) : f.termNodeId.slice(0, 8);
              return (
                <FilterChip key={`${f.boId}:${f.termNodeId}:${i}`} filter={f} label={label} parameters={parameters}
                  onChange={(patch) => onUpdateFilter(i, patch)} onDelete={() => onRemoveFilter(i)} />
              );
            })}
          </Stack>
          <ParametersRow parameters={parameters} onAdd={onAddParameter} onRemove={onRemoveParameter} />
        </Stack>
      </ShelfAccordion>

      <ShelfAccordion
        open={openGroup} onToggle={() => setOpenGroup((v) => !v)}
        icon={<SchemaIcon sx={{ fontSize: 16 }} color="primary" />}
        title="Group By"
        summary={<Chip size="small" label={`GROUP BY ${dimensions.length}`} sx={{ height: 18, fontSize: '0.62rem', fontFamily: 'monospace' }} />}
        last
      >
        <Stack direction="row" spacing={1} alignItems="center">
          <Typography variant="overline" color="text.secondary" sx={{ fontSize: '0.65rem' }}>GROUP BY:</Typography>
          <Typography variant="caption" sx={{ fontFamily: 'monospace', bgcolor: 'action.hover', px: 1, py: 0.25, borderRadius: 0.5 }}>
            {dimensions.length > 0 ? dimensions.map((d) => d.termKey).join(', ') : '(none - add a dimension)'}
          </Typography>
        </Stack>
      </ShelfAccordion>
    </Paper>
  );
}

function ShelfAccordion({ open, onToggle, icon, title, summary, children, last }: {
  open: boolean; onToggle: () => void; icon: React.ReactNode; title: string;
  summary?: React.ReactNode; children: React.ReactNode; last?: boolean;
}) {
  return (
    <Paper variant="outlined" sx={{ mb: last ? 0 : 1, overflow: 'hidden' }}>
      <Stack
        direction="row" justifyContent="space-between" alignItems="center"
        onClick={onToggle}
        sx={{ px: 1.5, py: 0.75, cursor: 'pointer', bgcolor: 'action.hover', '&:hover': { bgcolor: 'action.selected' } }}
      >
        <Stack direction="row" spacing={1} alignItems="center">
          {icon}
          <Typography variant="subtitle2" sx={{ fontSize: '0.78rem' }}>{title}</Typography>
          {summary}
        </Stack>
        {open ? <ExpandLessIcon fontSize="small" color="disabled" /> : <ExpandMoreIcon fontSize="small" color="disabled" />}
      </Stack>
      {open && <Box sx={{ px: 1.5, py: 1.25, borderTop: 1, borderColor: 'divider' }}>{children}</Box>}
    </Paper>
  );
}

function ShelfRow({ label, color, empty, children }: { label: string; color: string; empty: string; children: React.ReactNode }) {
  const hasContent = React.Children.count(children) > 0;
  return (
    <Stack direction="row" spacing={1.5} alignItems="center">
      <Typography variant="overline" sx={{ width: 85, flexShrink: 0, fontSize: '0.65rem', color }}>{label}</Typography>
      <Box sx={{
        flex: 1, minHeight: 32, bgcolor: 'background.default', borderRadius: 1, px: 1, py: 0.5,
        display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap',
      }}>
        {hasContent ? children : <Typography variant="caption" color="text.disabled">{empty}</Typography>}
      </Box>
    </Stack>
  );
}

function ParametersRow({ parameters, onAdd, onRemove }: {
  parameters: SavedQueryParameter[]; onAdd: (p: SavedQueryParameter) => void; onRemove: (name: string) => void;
}) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [name, setName] = useState('');
  const [type, setType] = useState<SavedQueryParameter['type']>('string');
  const [required, setRequired] = useState(false);

  const submit = () => {
    const trimmed = name.trim();
    if (!trimmed || parameters.some((p) => p.name === trimmed)) return;
    onAdd({ name: trimmed, type, required });
    setName(''); setType('string'); setRequired(false); setAnchor(null);
  };

  return (
    <Stack direction="row" spacing={1.5} alignItems="center">
      <Typography variant="overline" sx={{ width: 85, flexShrink: 0, fontSize: '0.65rem', color: 'text.secondary' }}>Parameters:</Typography>
      <Stack direction="row" spacing={0.75} alignItems="center" flexWrap="wrap" useFlexGap>
        {parameters.map((p) => (
          <Chip key={p.name} size="small" variant="outlined" label={`:${p.name}${p.required ? ' *' : ''}`} onDelete={() => onRemove(p.name)} />
        ))}
        <Button size="small" startIcon={<AddIcon sx={{ fontSize: 14 }} />} onClick={(e) => setAnchor(e.currentTarget)}
          sx={{ fontSize: '0.65rem', color: 'text.secondary' }}>
          Add Parameter
        </Button>
      </Stack>
      <Popover open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}>
        <Stack spacing={1} sx={{ p: 1.5, minWidth: 220 }}>
          <TextField size="small" autoFocus label="Parameter name" value={name} onChange={(e) => setName(e.target.value)} />
          <Select size="small" value={type} onChange={(e) => setType(e.target.value as SavedQueryParameter['type'])}>
            <MenuItem value="string">string</MenuItem>
            <MenuItem value="number">number</MenuItem>
            <MenuItem value="date">date</MenuItem>
            <MenuItem value="boolean">boolean</MenuItem>
          </Select>
          <FormControlLabel control={<Checkbox size="small" checked={required} onChange={(e) => setRequired(e.target.checked)} />} label="Required" />
          <Button size="small" variant="contained" onClick={submit} disabled={!name.trim()}>Add</Button>
        </Stack>
      </Popover>
    </Stack>
  );
}

function FilterChip({ filter, label, parameters, onChange, onDelete }: {
  filter: EditableFilter; label: string; parameters: SavedQueryParameter[];
  onChange: (patch: Partial<EditableFilter>) => void; onDelete: () => void;
}) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const isBound = !!filter.paramRef;

  const needsValue = !NO_VALUE_OPERATORS.includes(filter.operator);
  const summary = isBound
    ? `${label} ← :${filter.paramRef}`
    : needsValue
      ? `${label} ${OPERATOR_LABEL[filter.operator]} ${filter.value || '…'}`
      : `${label} ${OPERATOR_LABEL[filter.operator]}`;

  return (
    <>
      <Chip size="small" color={isBound ? 'default' : 'primary'} variant="outlined" label={summary}
        icon={isBound ? <LockIcon sx={{ fontSize: 14 }} /> : undefined}
        onClick={(e) => setAnchor(e.currentTarget)} onDelete={onDelete} />
      <Popover
        open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      >
        <Stack spacing={1} sx={{ p: 1.5, minWidth: 240 }}>
          <Typography variant="caption" color="text.secondary">{label}</Typography>
          <Select
            size="small" value={filter.operator}
            onChange={(e) => onChange({ operator: e.target.value as FilterOperator })}
          >
            {Object.entries(OPERATOR_LABEL).map(([op, lbl]) => (
              <MenuItem key={op} value={op}>{lbl}</MenuItem>
            ))}
          </Select>

          <ToggleButtonGroup size="small" exclusive value={isBound ? 'param' : 'literal'}
            onChange={(_, v) => {
              if (v === 'literal') onChange({ paramRef: undefined });
              else if (v === 'param') onChange({ paramRef: parameters[0]?.name, value: '' });
            }}>
            <ToggleButton value="literal" sx={{ fontSize: '0.65rem', px: 1 }}>Literal value</ToggleButton>
            <ToggleButton value="param" sx={{ fontSize: '0.65rem', px: 1 }} disabled={parameters.length === 0}>Parameter</ToggleButton>
          </ToggleButtonGroup>

          {isBound ? (
            <Select size="small" value={filter.paramRef} onChange={(e) => onChange({ paramRef: e.target.value })}>
              {parameters.map((p) => <MenuItem key={p.name} value={p.name}>:{p.name}</MenuItem>)}
            </Select>
          ) : needsValue && (
            <TextField
              size="small" autoFocus label="Value" value={filter.value}
              onChange={(e) => onChange({ value: e.target.value })}
            />
          )}
          <Button size="small" onClick={() => setAnchor(null)}>Done</Button>
        </Stack>
      </Popover>
    </>
  );
}
