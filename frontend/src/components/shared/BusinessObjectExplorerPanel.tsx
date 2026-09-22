/**
 * BusinessObjectExplorerPanel - THE centralized Business Object
 * selection + field catalog control (panes 1 + 2 of the query/report/page
 * "3-pane explorer" UX). Built on useBusinessObjectSelector, the same
 * headless hook every BO-picking surface in the app already shares - this
 * component is the visual layer on top, meant to be dropped into Query
 * Builder, Report Builder, and Page Studio alike instead of each growing
 * its own picker UI.
 *
 * Pane 1 (left, fixed 270px): primary object (locked) + any related
 * objects added so far, each clickable to become "active" for pane 2;
 * a "Related Business Objects to Add" section with one-click Join buttons.
 * Pane 2 (middle, fixed 320px): searchable, ALL/DIM/MEAS/CALC-filterable
 * field list for whichever object is active, grouped into Dimensions and
 * Measures & Metrics, each row showing a drag handle, type chip, and
 * either a checkmark (already in the query) or quick-add actions.
 */
import React, { useMemo, useState } from 'react';
import {
  Box, Paper, Typography, TextField, InputAdornment, Chip, Stack, Button,
  List, ListItemButton, ListItemText, CircularProgress, Alert, Tooltip, IconButton,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import AddCircleIcon from '@mui/icons-material/AddCircle';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import LockIcon from '@mui/icons-material/Lock';
import DatasetIcon from '@mui/icons-material/Dataset';
import AbcIcon from '@mui/icons-material/Abc';
import NumbersIcon from '@mui/icons-material/Numbers';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import ToggleOnIcon from '@mui/icons-material/ToggleOn';
import FingerprintIcon from '@mui/icons-material/Fingerprint';
import CategoryIcon from '@mui/icons-material/Category';
import PaidIcon from '@mui/icons-material/Paid';
import DataObjectIcon from '@mui/icons-material/DataObject';
import ViewListIcon from '@mui/icons-material/ViewList';
import type { BusinessObjectSelector } from './BusinessObjectSelectorControl';
import { looksLikeUuid, type SemanticTermView } from '../../studio-core/binding/businessObjectApi';

/** Never render a raw id where a name belongs - shows "Loading…" instead
 * while the display name is still resolving (see useBusinessObjectSelector's
 * boOptions backfill effect). */
function displayName(name: string): string {
  return looksLikeUuid(name) ? 'Loading…' : name;
}

type FieldFilter = 'ALL' | 'DIMENSION' | 'MEASURE' | 'CALCULATED';
const FILTER_TABS: { value: FieldFilter; label: string }[] = [
  { value: 'ALL', label: 'ALL' },
  { value: 'DIMENSION', label: 'DIM' },
  { value: 'MEASURE', label: 'MEAS' },
  { value: 'CALCULATED', label: 'CALC' },
];

export interface BusinessObjectExplorerPanelProps {
  selector: BusinessObjectSelector;
  activeBoId: string;
  onSelectActive: (boId: string) => void;
  onToggleField: (boId: string, termNodeId: string) => void;
  onAddFilter?: (boId: string, term: SemanticTermView) => void;
}

export default function BusinessObjectExplorerPanel({
  selector, activeBoId, onSelectActive, onToggleField, onAddFilter,
}: BusinessObjectExplorerPanelProps) {
  const { primary, objects } = selector;
  if (!primary) return null;

  const active = objects.find((o) => o.boId === activeBoId) || primary;

  return (
    <Stack direction="row" spacing={1.5} sx={{ height: '100%', minHeight: 0 }}>
      <NavigatorPane selector={selector} activeBoId={activeBoId} onSelectActive={onSelectActive} />
      <FieldsPane active={active} onToggleField={onToggleField} onAddFilter={onAddFilter} />
    </Stack>
  );
}

// ---------- Pane 1: Business Object Selection ----------

function NavigatorPane({ selector, activeBoId, onSelectActive }: {
  selector: BusinessObjectSelector; activeBoId: string; onSelectActive: (boId: string) => void;
}) {
  const { primary, related, availableRelationships, relationshipsLoading, addRelated, removeRelated } = selector;
  if (!primary) return null;

  return (
    <Paper variant="outlined" sx={{ width: 270, flexShrink: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <PaneHeader icon={<DatasetIcon sx={{ fontSize: 17 }} color="secondary" />} title="Business Object Selection" />
      <Box sx={{ flex: 1, overflowY: 'auto', p: 1.25, display: 'flex', flexDirection: 'column', gap: 1 }}>
        <Typography variant="overline" color="text.secondary" sx={{ fontSize: '0.62rem' }}>Primary Business Object</Typography>
        <BoNavItem
          name={displayName(primary.boName)} sub={primary.bindingId}
          active={activeBoId === primary.boId} locked
          onClick={() => onSelectActive(primary.boId)}
        />
        {related.map((r) => (
          <BoNavItem
            key={r.boId} name={displayName(r.boName)} sub={r.bindingId}
            active={activeBoId === r.boId}
            onClick={() => onSelectActive(r.boId)}
            onRemove={() => removeRelated(r.boId)}
          />
        ))}

        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mt: 1.5 }}>
          <Typography variant="overline" color="secondary.main" sx={{ fontSize: '0.62rem' }}>
            Related Objects to Add
          </Typography>
        </Stack>
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.68rem' }}>
          Click Join to append a related object.
        </Typography>
        <Stack spacing={0.75}>
          {relationshipsLoading ? (
            <Box display="flex" justifyContent="center" p={1}><CircularProgress size={16} /></Box>
          ) : availableRelationships.length === 0 ? (
            <Typography variant="caption" color="text.disabled">No more related objects</Typography>
          ) : (
            availableRelationships.map((r) => (
              <Stack key={r.targetObjectId} direction="row" alignItems="center" justifyContent="space-between"
                sx={{ px: 1, py: 0.5, bgcolor: 'action.hover', border: 1, borderColor: 'divider', borderRadius: 1 }}>
                <Typography variant="caption" fontWeight={500} sx={{ fontFamily: 'monospace' }} noWrap>{r.relatedObjectName}</Typography>
                <Button size="small" variant="outlined" sx={{ minWidth: 0, px: 1, py: 0, fontSize: '0.65rem', height: 22 }}
                  onClick={() => addRelated(r.targetObjectId, r.relatedObjectName).then(() => onSelectActive(r.targetObjectId))}>
                  + Join
                </Button>
              </Stack>
            ))
          )}
        </Stack>
      </Box>
    </Paper>
  );
}

function BoNavItem({ name, sub, active, locked, onClick, onRemove }: {
  name: string; sub: string; active: boolean; locked?: boolean; onClick: () => void; onRemove?: () => void;
}) {
  return (
    <Box
      onClick={onClick}
      sx={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 1.25, py: 1,
        borderRadius: 1, cursor: 'pointer', bgcolor: active ? 'primary.main' : 'action.hover',
        color: active ? 'primary.contrastText' : 'text.primary',
        border: 1, borderColor: active ? 'primary.main' : 'transparent',
        '&:hover': { borderColor: active ? 'primary.main' : 'secondary.main' },
      }}
    >
      <Stack direction="row" spacing={1} alignItems="center" sx={{ minWidth: 0 }}>
        {locked && <LockIcon sx={{ fontSize: 15, opacity: 0.7 }} />}
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="body2" fontWeight={600} noWrap>{name}</Typography>
          <Typography variant="caption" sx={{ fontFamily: 'monospace', opacity: 0.75, display: 'block' }} noWrap>{sub}</Typography>
        </Box>
      </Stack>
      <Stack direction="row" alignItems="center" spacing={0.5}>
        {active ? <CheckCircleIcon sx={{ fontSize: 17 }} /> : <RadioButtonUncheckedIcon sx={{ fontSize: 16, opacity: 0.4 }} />}
        {onRemove && (
          <IconButton size="small" onClick={(e) => { e.stopPropagation(); onRemove(); }} sx={{ color: 'inherit', p: 0.25 }}>
            <Typography sx={{ fontSize: 13, lineHeight: 1 }}>×</Typography>
          </IconButton>
        )}
      </Stack>
    </Box>
  );
}

// ---------- Pane 2: BO Fields ----------

function FieldsPane({ active, onToggleField, onAddFilter }: {
  active: { boId: string; boName: string; terms: SemanticTermView[]; termsLoading: boolean; termsError: string | null };
  onToggleField: (boId: string, termNodeId: string) => void;
  onAddFilter?: (boId: string, term: SemanticTermView) => void;
}) {
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<FieldFilter>('ALL');

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    return active.terms.filter((t) => {
      if (filter !== 'ALL' && t.role !== filter) return false;
      if (!term) return true;
      return t.displayName.toLowerCase().includes(term) || t.termKey.toLowerCase().includes(term);
    });
  }, [active.terms, search, filter]);

  const dims = filtered.filter((t) => t.role !== 'MEASURE');
  const measures = filtered.filter((t) => t.role === 'MEASURE');

  return (
    <Paper variant="outlined" sx={{ width: 320, flexShrink: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <PaneHeader
        icon={<ViewListIcon sx={{ fontSize: 17 }} color="warning" />}
        title="BO Fields"
        endAdornment={
          <Stack direction="row" spacing={0.5}>
            <Chip size="small" label={`${active.terms.length} Fields`} sx={{ height: 18, fontSize: '0.62rem', fontFamily: 'monospace' }} />
            <Chip size="small" color="secondary" label={displayName(active.boName)} sx={{ height: 18, fontSize: '0.6rem', maxWidth: 90 }} />
          </Stack>
        }
      />
      <Box sx={{ p: 1.25, pb: 1, borderBottom: 1, borderColor: 'divider' }}>
        <TextField
          size="small" fullWidth placeholder="Search fields, attributes…"
          value={search} onChange={(e) => setSearch(e.target.value)}
          InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }}
          sx={{ mb: 1 }}
        />
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 0.5, bgcolor: 'background.default', p: 0.25, borderRadius: 1 }}>
          {FILTER_TABS.map((f) => (
            <Box
              key={f.value} onClick={() => setFilter(f.value)}
              sx={{
                textAlign: 'center', py: 0.4, fontSize: '0.65rem', fontWeight: 600, borderRadius: 0.5, cursor: 'pointer',
                color: filter === f.value ? 'secondary.main' : 'text.secondary',
                bgcolor: filter === f.value ? 'action.selected' : 'transparent',
              }}
            >
              {f.label}
            </Box>
          ))}
        </Box>
      </Box>

      <Box sx={{ flex: 1, overflowY: 'auto', p: 1.25 }}>
        {active.termsLoading ? (
          <Box display="flex" justifyContent="center" p={3}><CircularProgress size={20} /></Box>
        ) : active.termsError ? (
          <Alert severity="warning">{active.termsError}</Alert>
        ) : (
          <>
            {dims.length > 0 && (
              <FieldGroup label="Dimensions" caption="QUALITATIVE" color="secondary.main">
                {dims.map((t) => (
                  <FieldRow key={t.termNodeId} term={t}
                    onClick={() => onToggleField(active.boId, t.termNodeId)}
                    onAddFilter={onAddFilter ? () => onAddFilter(active.boId, t) : undefined} />
                ))}
              </FieldGroup>
            )}
            {measures.length > 0 && (
              <FieldGroup label="Measures & Metrics" caption="AGGREGATES" color="warning.main">
                {measures.map((t) => (
                  <FieldRow key={t.termNodeId} term={t}
                    onClick={() => onToggleField(active.boId, t.termNodeId)}
                    onAddFilter={onAddFilter ? () => onAddFilter(active.boId, t) : undefined} />
                ))}
              </FieldGroup>
            )}
            {filtered.length === 0 && (
              <Typography variant="caption" color="text.disabled">No matching fields.</Typography>
            )}
          </>
        )}
      </Box>
    </Paper>
  );
}

function FieldGroup({ label, caption, color, children }: { label: string; caption: string; color: string; children: React.ReactNode }) {
  return (
    <Box sx={{ mb: 1.5 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 0.5 }}>
        <Typography variant="overline" sx={{ color, fontSize: '0.65rem', fontWeight: 700 }}>{label}</Typography>
        <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace', fontSize: '0.6rem' }}>{caption}</Typography>
      </Stack>
      <List dense disablePadding>{children}</List>
    </Box>
  );
}

/** One glyph per data type, so pane 2 reads as a scan-able icon column
 * instead of a wall of type-name chips; the actual type string still
 * shows up on hover, via the Tooltip wrapping the icon, rather than being
 * dropped. */
function dataTypeIcon(dataType: string | undefined) {
  const t = (dataType || '').toLowerCase();
  if (t.includes('uuid') || t.includes('id')) return FingerprintIcon;
  if (t.includes('date') || t.includes('time')) return CalendarMonthIcon;
  if (t.includes('bool')) return ToggleOnIcon;
  if (t.includes('curr') || t.includes('money') || t.includes('decimal')) return PaidIcon;
  if (t.includes('num') || t.includes('int') || t.includes('float') || t.includes('rate')) return NumbersIcon;
  if (t.includes('cat') || t.includes('enum')) return CategoryIcon;
  if (t.includes('text') || t.includes('string') || t.includes('varchar')) return AbcIcon;
  return DataObjectIcon;
}

function FieldRow({ term, onClick, onAddFilter }: { term: SemanticTermView; onClick: () => void; onAddFilter?: () => void }) {
  const isMeasure = term.role === 'MEASURE';
  const TypeIcon = dataTypeIcon(term.dataType);
  return (
    <ListItemButton
      dense disableRipple
      sx={{
        borderRadius: 1, mb: 0.4, py: 0.5, px: 0.75, cursor: 'default',
        bgcolor: term.selected ? 'action.selected' : 'background.default',
        border: 1, borderColor: term.selected ? (isMeasure ? 'warning.main' : 'secondary.main') : 'divider',
        '&:hover .field-row-actions': { opacity: 1 },
      }}
    >
      <DragIndicatorIcon sx={{ fontSize: 15, color: 'text.disabled', mr: 0.5, cursor: 'grab' }} />
      <Tooltip title={term.dataType || 'text'}>
        <TypeIcon sx={{ fontSize: 16, mr: 1, color: isMeasure ? 'warning.main' : 'secondary.main' }} />
      </Tooltip>
      <ListItemText
        onClick={onClick}
        primary={term.displayName}
        secondary={term.termKey}
        primaryTypographyProps={{ variant: 'body2', fontWeight: term.selected ? 600 : 500, noWrap: true }}
        secondaryTypographyProps={{ variant: 'caption', sx: { fontFamily: 'monospace' }, noWrap: true }}
        sx={{ cursor: 'pointer' }}
      />
      {term.selected ? (
        <CheckCircleIcon sx={{ fontSize: 16 }} color={isMeasure ? 'warning' : 'secondary'} />
      ) : (
        <Box className="field-row-actions" sx={{ display: 'flex', gap: 0.25, opacity: 0, transition: 'opacity 120ms' }}>
          {onAddFilter && (
            <Tooltip title="Add as filter">
              <IconButton size="small" onClick={(e) => { e.stopPropagation(); onAddFilter(); }}>
                <FilterAltIcon sx={{ fontSize: 15 }} />
              </IconButton>
            </Tooltip>
          )}
          <Tooltip title={isMeasure ? 'Add as measure' : 'Add as dimension'}>
            <IconButton size="small" onClick={(e) => { e.stopPropagation(); onClick(); }}>
              <AddCircleIcon sx={{ fontSize: 16 }} color="primary" />
            </IconButton>
          </Tooltip>
        </Box>
      )}
    </ListItemButton>
  );
}

function PaneHeader({ icon, title, endAdornment }: { icon: React.ReactNode; title: string; endAdornment?: React.ReactNode }) {
  return (
    <Stack direction="row" alignItems="center" justifyContent="space-between"
      sx={{ px: 1.5, py: 1, borderBottom: 1, borderColor: 'divider', bgcolor: 'background.default' }}>
      <Stack direction="row" spacing={0.75} alignItems="center">
        {icon}
        <Typography variant="overline" sx={{ fontSize: '0.68rem' }}>{title}</Typography>
      </Stack>
      {endAdornment}
    </Stack>
  );
}
