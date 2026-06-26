import { useRef, useEffect, useState, useMemo } from 'react';
import type { FC } from 'react';
import {
  Paper,
  Box,
  Typography,
  Chip,
  ListItem,
  ListItemText,
  Stack,
  Tooltip,
  TextField,
  InputAdornment,
  Select,
  MenuItem,
  IconButton,
} from '@mui/material';
import { Search } from '@mui/icons-material';
import FilterTotals from './FilterTotals';
import { AvailableSource } from '../hooks/useAvailableSources';
import { getDatatypeIcon, getDimensionMeasureIcon } from '../utils/viewEditorUtils';
import ItemPath from '../../common/ItemPath';
import EditOutlined from '@mui/icons-material/EditOutlined';

interface ViewComponentRow {
  id: string;
  name: string;
  description?: string;
  datatype?: string;
  type: 'dimension' | 'measure';
  originalIndex: number;
  _sourceId?: string;
  _sourceName?: string;
  _sourceType: AvailableSource['type'];
}

interface ViewComponentsPanelProps {
  items: ViewComponentRow[];
  dimensionCount: number;
  measureCount: number;
  selectedItems: Set<string>;
  onItemClick: (itemId: string, index: number, modifiers: { additive?: boolean; range?: boolean }) => void;
  targetHighlightMap: Record<string, boolean>;
  availableSources?: { id: string | number; name: string }[];
  searchQuery?: string;
  onSearchChange?: (v: string) => void;
  sourceFilter?: string;
  onSourceFilterChange?: (v: string) => void;
  // handleRemoveViewItem removed — removals are done via center controls
  // handleRemoveViewItem?: (type: 'dimension' | 'measure', index: number) => void;
  onScrollStuckChange?: (stuck: boolean) => void;
  onEditClick?: (item: ViewComponentRow) => void;
  active?: boolean;
  typeFilter?: 'all' | 'dimension' | 'measure';
  setTypeFilter?: (v: 'all' | 'dimension' | 'measure') => void;
}

const SOURCE_TYPE_META: Record<AvailableSource['type'], { label: string; color: string }> = {
  cube: { label: 'Cube', color: 'warning.main' },
  // extended views should render with purple chips (secondary palette)
  extended_view: { label: 'View', color: 'secondary.main' },
};

export const ViewComponentsPanel: FC<ViewComponentsPanelProps> = ({
  items,
  dimensionCount,
  measureCount,
  selectedItems,
  onItemClick,
  targetHighlightMap,
  availableSources,
  // search/filter handled by parent
  searchQuery,
  onSearchChange,
  sourceFilter,
  onSourceFilterChange,
  // handleRemoveViewItem,
  onScrollStuckChange,
  active,
  onEditClick,
  typeFilter,
  setTypeFilter,
}) => {
  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    let rafId: number | null = null;
    let debounceTimer: number | null = null;
    const fire = (stuck: boolean) => {
      if (onScrollStuckChange) onScrollStuckChange(stuck);
    };
    const checkImmediate = () => {
      const stuck = el.scrollTop > 0;
      fire(stuck);
    };
    const check = () => {
      if (debounceTimer) window.clearTimeout(debounceTimer);
      debounceTimer = window.setTimeout(() => {
        if (rafId) cancelAnimationFrame(rafId);
        rafId = requestAnimationFrame(() => {
          const stuck = el.scrollTop > 0;
          fire(stuck);
        });
      }, 100);
    };
    checkImmediate();
    el.addEventListener('scroll', check, { passive: true });
    return () => {
      el.removeEventListener('scroll', check);
      if (debounceTimer) window.clearTimeout(debounceTimer);
      if (rafId) cancelAnimationFrame(rafId);
    };
  }, [onScrollStuckChange]);

  // Local search/source state for when the panel is used standalone (parent doesn't provide handlers)
  const [localSearch, setLocalSearch] = useState<string>('');
  const [localSource, setLocalSource] = useState<string>('all');
  // Local type filter when parent does not control it
  const [localTypeFilter, setLocalTypeFilter] = useState<'all' | 'dimension' | 'measure'>('all');

  const activeTypeFilter = typeFilter ?? localTypeFilter;

  const filteredItems = useMemo(() => items.filter((it) => activeTypeFilter === 'all' || it.type === activeTypeFilter), [items, activeTypeFilter]);

  return (
    <Paper
      elevation={0}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
      }}
    >
      <Box sx={{ p: 2, borderBottom: 1, borderColor: active ? 'grey.200' : 'divider', position: 'sticky', top: 0, zIndex: 5, backgroundColor: 'background.paper', boxShadow: active ? '0 10px 30px rgba(2,6,23,0.04)' : '0 1px 4px rgba(0,0,0,0.03)', transition: 'border-color 140ms ease, box-shadow 140ms ease' }}>
        {/* Render search/source controls only when parent hasn't provided handlers to avoid duplicates. */}
        {!(onSearchChange && onSourceFilterChange) ? (
          <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
            <TextField
              size="small"
              placeholder="Search components..."
              value={(onSearchChange ? (searchQuery || '') : localSearch)}
              onChange={(e) => {
                const v = e.target.value;
                if (onSearchChange) onSearchChange(v);
                else setLocalSearch(v);
              }}
              InputProps={{ startAdornment: <InputAdornment position="start"><Search sx={{ color: 'text.secondary' }} /></InputAdornment> }}
              fullWidth
            />

            <Select
              size="small"
              value={(onSourceFilterChange ? (sourceFilter || 'all') : localSource)}
              onChange={(e) => {
                const v = String(e.target.value);
                if (onSourceFilterChange) onSourceFilterChange(v);
                else setLocalSource(v);
              }}
              sx={{ minWidth: 160 }}
            >
              <MenuItem value="all">All Sources</MenuItem>
              {(availableSources || []).map((s) => (
                <MenuItem key={s.id} value={s.id}>{s.name}</MenuItem>
              ))}
            </Select>
          </Box>
        ) : null}

        <Typography variant="h6" sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
          View Components
        </Typography>

        <FilterTotals
          dimensionCount={dimensionCount}
          measureCount={measureCount}
          typeFilter={activeTypeFilter}
          setTypeFilter={(v) => {
            if (typeof setTypeFilter === 'function') setTypeFilter(v);
            else setLocalTypeFilter(v);
          }}
        />
      </Box>

      <Box ref={scrollRef} sx={{ flex: 1, overflow: 'auto', p: 1 }}>
        <Stack spacing={1}>
          {filteredItems.length === 0 ? (
            <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
              <Stack spacing={2} alignItems="center">
                <Typography variant="body2">No components added yet</Typography>
                <Typography variant="caption" textAlign="center">
                  Configure the view extension on the left, then select components from the Available Components panel
                </Typography>
              </Stack>
            </Box>
          ) : (
            filteredItems.map((item, index) => {
              const isSelected = selectedItems.has(item.id);
              const highlight = targetHighlightMap[item.id];
              const sourceMeta = SOURCE_TYPE_META[item._sourceType] || SOURCE_TYPE_META.cube;

              return (
                <ListItem
                  key={item.id}
                  onClick={(event) => {
                    event.preventDefault();
                    onItemClick(item.id, index, {
                      additive: event.ctrlKey || event.metaKey,
                      range: event.shiftKey,
                    });
                  }}
                  sx={{
                    border: 1,
                    borderColor: isSelected ? 'primary.main' : 'divider',
                    borderRadius: 1,
                    mb: 0.5,
                    cursor: 'pointer',
                    bgcolor: highlight ? 'success.50' : isSelected ? 'primary.50' : 'background.paper',
                    '&:hover': { bgcolor: highlight ? 'success.100' : 'action.hover' },
                    transition: 'background-color 150ms ease, border-color 150ms ease',
                  }}
                >
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                        <Box sx={{ minWidth: 0, flex: 1 }}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>
                            {item.name}
                          </Typography>
                          <ItemPath
                            id={item.id}
                            availableSources={availableSources}
                            noWrap
                          />
                        </Box>
                        <Box sx={{ ml: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Tooltip title={item.type === 'dimension' ? 'Dimension' : 'Measure'}>
                            <span>{getDimensionMeasureIcon(item.type)}</span>
                          </Tooltip>

                          {item.datatype && (
                            <Tooltip title={`Type: ${item.datatype}`}>
                              {getDatatypeIcon(item.datatype, item.type, item.name)}
                            </Tooltip>
                          )}

                          <Tooltip title={sourceMeta.label}>
                            <span>
                              <Chip
                                label={item._sourceName || 'View'}
                                size="small"
                                variant="outlined"
                                sx={{
                                  color: sourceMeta.color,
                                  borderColor: sourceMeta.color,
                                  backgroundColor: 'transparent',
                                }}
                              />
                            </span>
                          </Tooltip>
                        </Box>
                      </Box>
                    }
                    secondary={item.description}
                  />
                  {/* Edit button to open modal for editing view-level properties */}
                  <Box sx={{ ml: 1, display: 'flex', alignItems: 'center' }}>
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        if (typeof onEditClick === 'function') onEditClick(item);
                      }}
                      aria-label="edit"
                    >
                      <EditOutlined fontSize="small" sx={{ color: 'action.active' }} />
                    </IconButton>
                  </Box>
                </ListItem>
              );
            })
          )}
        </Stack>
      </Box>
    </Paper>
  );
};

export default ViewComponentsPanel;