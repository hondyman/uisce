import React, { useState, useMemo, useEffect } from 'react';
import {
  Box,
  Card,
  Typography,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Checkbox,
  ListItemText,
  TextField,
  Chip,
  Tooltip,
  Stack,
  OutlinedInput,
  Button,
} from '@mui/material';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import FilterListIcon from '@mui/icons-material/FilterList';
import type { SlicerTileConfig } from '../../types/pageStudio';
import { useCrossFilterBus, useActiveCrossFilters } from '../../features/query-builder/utils/crossFilterBus';

export interface SlicerWidgetProps {
  id: string;
  title?: string;
  config: SlicerTileConfig;
  data?: {
    rows?: Record<string, unknown>[];
    columns?: { name: string; type: string }[];
  };
  mode?: 'design' | 'preview';
  onUpdatePageVar?: (varName: string, value: any) => void;
}

export const SlicerWidget: React.FC<SlicerWidgetProps> = ({
  id,
  title,
  config,
  data,
  mode = 'preview',
  onUpdatePageVar,
}) => {
  const bus = useCrossFilterBus();
  const activeFilters = useActiveCrossFilters(bus);

  const [selectedValues, setSelectedValues] = useState<string[]>(() => {
    if (config.defaultSelection?.value) {
      return Array.isArray(config.defaultSelection.value)
        ? config.defaultSelection.value
        : [config.defaultSelection.value];
    }
    return [];
  });

  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  // Authoring validation: dateRange requires a bound parameter
  const dateRangeParamWarning = useMemo(() => {
    if (config.display === 'dateRange' && !config.paramBinding?.varName) {
      return 'Date range slicer requires a bound parameter (paramBinding)';
    }
    return null;
  }, [config.display, config.paramBinding]);

  // Extract raw member options from query data
  const rawMembers = useMemo(() => {
    if (!data?.rows || !config.dimensionAlias) return [];
    const set = new Set<string>();
    const list: string[] = [];
    for (const r of data.rows) {
      const val = String(r[config.dimensionAlias] ?? '');
      if (val && !set.has(val)) {
        set.add(val);
        list.push(val);
      }
    }
    if (config.sortBy === 'label') {
      list.sort((a, b) => a.localeCompare(b));
    }
    return list;
  }, [data?.rows, config.dimensionAlias, config.sortBy]);

  // Cascading Slicer Filtering: if cascadingFrom is configured, filter members by upstream active filters
  const members = useMemo(() => {
    let filtered = rawMembers;
    if (config.cascadingFrom && config.cascadingFrom.length > 0) {
      const upstreamFilters = activeFilters.filter((f) =>
        config.cascadingFrom!.includes(f.sourceWidgetId)
      );
      if (upstreamFilters.length > 0 && data?.rows) {
        const allowedRows = data.rows.filter((r) => {
          return upstreamFilters.every((uf) => {
            if (uf.termNodeId in r) {
              const valInRow = r[uf.termNodeId];
              return Array.isArray(uf.value)
                ? (uf.value as any[]).includes(valInRow)
                : String(valInRow) === String(uf.value);
            }
            return Object.values(r).some((v) =>
              Array.isArray(uf.value)
                ? (uf.value as any[]).includes(v)
                : String(v) === String(uf.value)
            );
          });
        });
        const cascadeSet = new Set(allowedRows.map((r) => String(r[config.dimensionAlias] ?? '')));
        filtered = filtered.filter((m) => cascadeSet.has(m));
      }
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      filtered = filtered.filter((m) => m.toLowerCase().includes(q));
    }
    return filtered;
  }, [rawMembers, config.cascadingFrom, activeFilters, data?.rows, config.dimensionAlias, searchQuery]);

  // Categorical emission to CrossFilterBus
  const handleCategoricalChange = (newValues: string[]) => {
    let finalValues = newValues;
    if (config.maxSelections && finalValues.length > config.maxSelections) {
      finalValues = finalValues.slice(-config.maxSelections);
    }
    setSelectedValues(finalValues);

    if (finalValues.length === 0) {
      bus.remove(config.termNodeId);
    } else if (finalValues.length === 1) {
      bus.emit(id, config.termNodeId, finalValues[0], 'eq');
    } else {
      bus.emit(id, config.termNodeId, finalValues, 'in');
    }
  };

  // Date Range emission to Page Variables (parameter binding)
  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
    if (config.paramBinding?.varName && onUpdatePageVar) {
      onUpdatePageVar(config.paramBinding.varName, { start, end });
    }
  };

  const handleClear = () => {
    setSelectedValues([]);
    setStartDate('');
    setEndDate('');
    bus.remove(config.termNodeId);
    if (config.paramBinding?.varName && onUpdatePageVar) {
      onUpdatePageVar(config.paramBinding.varName, null);
    }
  };

  return (
    <Card variant="outlined" sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
        <Stack direction="row" alignItems="center" spacing={0.5}>
          <FilterListIcon fontSize="small" color="action" />
          <Typography variant="subtitle2" fontWeight={600}>
            {title || config.dimensionAlias || 'Slicer'}
          </Typography>
        </Stack>

        <Stack direction="row" spacing={1} alignItems="center">
          {dateRangeParamWarning && mode === 'design' && (
            <Tooltip title={dateRangeParamWarning}>
              <Chip
                size="small"
                color="warning"
                icon={<WarningAmberIcon fontSize="small" />}
                label="Param Required"
                sx={{ height: 20, fontSize: '0.7rem' }}
              />
            </Tooltip>
          )}
          {(selectedValues.length > 0 || startDate || endDate) && (
            <Button size="small" onClick={handleClear} sx={{ minWidth: 0, p: 0.5, fontSize: '0.75rem' }}>
              Clear
            </Button>
          )}
        </Stack>
      </Stack>

      {config.showSearch && config.display !== 'dateRange' && (
        <TextField
          size="small"
          placeholder="Search..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          sx={{ mb: 1 }}
        />
      )}

      <Box sx={{ flex: 1, minHeight: 0 }}>
        {/* Dropdown Display */}
        {config.display === 'dropdown' && (
          <FormControl fullWidth size="small">
            <InputLabel>{title || 'Select...'}</InputLabel>
            <Select
              value={selectedValues[0] || ''}
              label={title || 'Select...'}
              onChange={(e) => handleCategoricalChange(e.target.value ? [String(e.target.value)] : [])}
            >
              <MenuItem value="">
                <em>All</em>
              </MenuItem>
              {members.map((m) => (
                <MenuItem key={m} value={m}>
                  {m}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        )}

        {/* Multi-Select Dropdown */}
        {config.display === 'multiSelect' && (
          <FormControl fullWidth size="small">
            <InputLabel>{title || 'Select multiple...'}</InputLabel>
            <Select
              multiple
              value={selectedValues}
              onChange={(e) => {
                const v = typeof e.target.value === 'string' ? e.target.value.split(',') : (e.target.value as string[]);
                handleCategoricalChange(v);
              }}
              input={<OutlinedInput label={title || 'Select multiple...'} />}
              renderValue={(selected) => (
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {selected.map((val) => (
                    <Chip key={val} label={val} size="small" />
                  ))}
                </Box>
              )}
            >
              {members.map((m) => (
                <MenuItem key={m} value={m}>
                  <Checkbox checked={selectedValues.indexOf(m) > -1} size="small" />
                  <ListItemText primary={m} />
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        )}

        {/* List Checklist Display */}
        {config.display === 'list' && (
          <Box sx={{ maxHeight: 180, overflowY: 'auto', pr: 0.5 }}>
            {members.map((m) => {
              const checked = selectedValues.includes(m);
              return (
                <Stack
                  key={m}
                  direction="row"
                  alignItems="center"
                  spacing={1}
                  sx={{
                    py: 0.25,
                    cursor: 'pointer',
                    '&:hover': { bgcolor: 'action.hover' },
                    borderRadius: 0.5,
                  }}
                  onClick={() => {
                    const next = checked
                      ? selectedValues.filter((v) => v !== m)
                      : [...selectedValues, m];
                    handleCategoricalChange(next);
                  }}
                >
                  <Checkbox checked={checked} size="small" sx={{ p: 0.5 }} />
                  <Typography variant="body2">{m}</Typography>
                </Stack>
              );
            })}
          </Box>
        )}

        {/* Date Range Display */}
        {config.display === 'dateRange' && (
          <Stack spacing={1.5} sx={{ mt: 0.5 }}>
            <TextField
              size="small"
              type="date"
              label="Start Date"
              InputLabelProps={{ shrink: true }}
              value={startDate}
              onChange={(e) => handleDateChange(e.target.value, endDate)}
              fullWidth
            />
            <TextField
              size="small"
              type="date"
              label="End Date"
              InputLabelProps={{ shrink: true }}
              value={endDate}
              onChange={(e) => handleDateChange(startDate, e.target.value)}
              fullWidth
            />
          </Stack>
        )}
      </Box>
    </Card>
  );
};

export default SlicerWidget;
