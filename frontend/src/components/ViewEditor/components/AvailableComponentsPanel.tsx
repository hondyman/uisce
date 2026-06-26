import { useRef } from 'react';
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
  IconButton,
} from '@mui/material';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import FilterTotals from './FilterTotals';
import { AvailableSource, AvailableItem } from '../hooks/useAvailableSources';
import { getDatatypeIcon, getDimensionMeasureIcon } from '../utils/viewEditorUtils';
// renderCoreCustomChips was used by the filter UI which is now moved to the parent.
import ItemPath from '../../common/ItemPath';

type FlattenedAvailableItem = AvailableItem & {
  _sourceId: string;
  _sourceName: string;
  _sourceType: AvailableSource['type'];
};

interface AvailableComponentsPanelProps {
  filteredAvailableSources: AvailableSource[];
  items: FlattenedAvailableItem[];
  selectedAvailableItems: Set<string>;
  onItemClick: (itemId: string, index: number, modifiers: { additive?: boolean; range?: boolean }) => void;
  highlightMap: Record<string, 'added' | 'exists'>;
  // search and filter are controlled by the parent but rendered here above the header
  searchQuery: string;
  onSearchChange: (v: string) => void;
  sourceFilter: string;
  onSourceFilterChange: (v: string) => void;
  availableSourceSummaries?: { id: string | number; name: string }[];
  onScrollStuckChange?: (stuck: boolean) => void;
  active?: boolean;
  onInfoClick?: (item: FlattenedAvailableItem) => void;
  typeFilter?: 'all' | 'dimension' | 'measure';
  setTypeFilter?: (v: 'all' | 'dimension' | 'measure') => void;
  availableTotals?: { dimensionCount: number; measureCount: number };
}

const SOURCE_TYPE_META: Record<AvailableSource['type'], { label: string; color: string }> = {
  cube: { label: 'Cube', color: 'warning.main' },
  // extended views should render with purple chips (secondary palette)
  extended_view: { label: 'View', color: 'secondary.main' },
};

export const AvailableComponentsPanel: FC<AvailableComponentsPanelProps> = ({
  filteredAvailableSources,
  items,
  selectedAvailableItems,
  onItemClick,
  highlightMap,
  // search/filter handled by parent
  onScrollStuckChange: _onScrollStuckChange,
  active,
  onInfoClick,
  typeFilter,
  setTypeFilter,
  availableTotals,
}) => {
  const scrollRef = useRef<HTMLDivElement | null>(null);

  return (
    <Paper sx={{
      border: 1,
      borderColor: 'divider',
      display: 'flex',
      flexDirection: 'column',
    }}>
      <Box sx={{ p: 2, borderBottom: 1, borderColor: active ? 'grey.200' : 'divider', position: 'sticky', top: 0, zIndex: 5, backgroundColor: 'background.paper', boxShadow: active ? '0 10px 30px rgba(2,6,23,0.04)' : '0 1px 4px rgba(0,0,0,0.03)', transition: 'border-color 140ms ease, box-shadow 140ms ease' }}>
        <Typography variant="h6" sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
          Available Components
        </Typography>
        <FilterTotals
          dimensionCount={availableTotals?.dimensionCount ?? 0}
          measureCount={availableTotals?.measureCount ?? 0}
          typeFilter={typeFilter || 'all'}
          setTypeFilter={(v) => setTypeFilter && setTypeFilter(v)}
        />
      </Box>

  <Box ref={scrollRef} sx={{ flex: 1, overflow: 'auto', p: 1 }}>
        <Stack spacing={1}>
          {items.length === 0 ? (
            <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
              <Stack spacing={2} alignItems="center">
                <Typography variant="body2">No available components</Typography>
                <Typography variant="caption" textAlign="center">
                  Try changing the filter or extending a different view
                </Typography>
              </Stack>
            </Box>
          ) : (
            items.map((item, index) => {
              const isSelected = selectedAvailableItems.has(item.id);
              const status = highlightMap[item.id];
              const isAdded = status === 'added';
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
                    bgcolor: isAdded ? 'grey.200' : isSelected ? 'primary.50' : 'background.paper',
                    '&:hover': { bgcolor: isAdded ? 'grey.200' : 'action.hover' },
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
                            availableSources={filteredAvailableSources.map((s) => ({ id: s.id, name: s.name }))}
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
                                label={item._sourceName}
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
                  {status && (
                    <Box sx={{ ml: 1, display: 'flex', alignItems: 'center' }}>
                      <Chip
                        label={status === 'added' ? 'Added' : 'Exists'}
                        size="small"
                        color={status === 'added' ? 'success' : 'warning'}
                        variant={status === 'added' ? 'filled' : 'outlined'}
                      />
                    </Box>
                  )}
                  {/* info button (opens read-only modal) */}
                  <Box sx={{ ml: 1, display: 'flex', alignItems: 'center' }}>
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        if (typeof onInfoClick === 'function') onInfoClick(item);
                      }}
                      aria-label="info"
                    >
                      <InfoOutlined fontSize="small" />
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