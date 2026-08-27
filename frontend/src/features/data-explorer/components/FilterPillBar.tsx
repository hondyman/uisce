import React from 'react';
import {
  Box,
  Chip,
  Stack,
  Typography,
  Button,
} from '@mui/material';
import {
  Add as AddIcon,
  FilterAlt as FilterIcon,
} from '@mui/icons-material';
import type {
  ExplorerSource,
  FilterSelection,
  FilterOperator,
} from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface FilterPillBarProps {
  source: ExplorerSource;
  filters: FilterSelection[];
  onAddFilter: () => void;
  onRemoveFilter: (index: number) => void;
  onEditFilter?: (index: number) => void;
  onDropField?: (fieldKey: string) => void;
}

const OPERATOR_LABELS: Record<FilterOperator, string> = {
  equals: 'is',
  not_equals: 'is not',
  contains: 'contains',
  starts_with: 'starts with',
  ends_with: 'ends with',
  gt: '>',
  gte: '>=',
  lt: '<',
  lte: '<=',
  in: 'is in',
  not_in: 'is not in',
  is_set: 'is set',
  is_not_set: 'is not set',
  between: 'between',
};

function renderValues(operator: FilterOperator, values: string[]) {
  if (operator === 'is_set' || operator === 'is_not_set') return '';
  if (operator === 'between') return `${values[0]} → ${values[1] ?? ''}`;
  if (operator === 'in' || operator === 'not_in') return values.join(', ');
  return values[0] ?? '';
}

export const FilterPillBar: React.FC<FilterPillBarProps> = ({
  source,
  filters,
  onAddFilter,
  onRemoveFilter,
  onEditFilter,
  onDropField,
}) => {
  const theme = useExplorerTheme();

  const handleDrop = (e: React.DragEvent) => {
    const rawData = e.dataTransfer.getData('application/json') || e.dataTransfer.getData('bo-field-bundle');
    const textPlain = e.dataTransfer.getData('text/plain');
    if ((rawData || textPlain) && onDropField) {
      try {
        let fieldNames: string[] = [];
        if (rawData) {
          const parsed = JSON.parse(rawData);
          if (Array.isArray(parsed)) {
            fieldNames = parsed.map((p) => p.name || p.technicalName || p.fieldKey);
          } else if (parsed.type === 'bofield_batch' || parsed.type === 'bo-field-bundle') {
            fieldNames = (parsed.fields || []).map((p: any) => p.name || p.technicalName || p.fieldKey);
          } else if (parsed.type === 'bofield' && parsed.field) {
            fieldNames = [parsed.field.name || parsed.field.technicalName];
          } else if (parsed.fieldKey || parsed.name) {
            fieldNames = [parsed.fieldKey || parsed.name];
          }
        }
        if (fieldNames.length === 0 && textPlain) {
          fieldNames = [textPlain];
        }

        if (fieldNames.length > 0) {
          e.preventDefault();
          e.stopPropagation();
          fieldNames.filter(Boolean).forEach((name) => onDropField(name));
        }
      } catch {
        // ignore
      }
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  };

  if (filters.length === 0) {
    return (
      <Box
        onDragOver={handleDragOver}
        onDrop={handleDrop}
        sx={{
          px: 3,
          py: 1,
          borderBottom: `1px solid ${theme.border}`,
          bgcolor: theme.background,
        }}
      >
        <Stack direction="row" alignItems="center" spacing={1}>
          <FilterIcon sx={{ fontSize: 16, color: theme.textMuted }} />
          <Typography variant="caption" sx={{ color: theme.textMuted }}>
            Drop terms here or click Add Filter
          </Typography>
          <Button
            size="small"
            startIcon={<AddIcon />}
            onClick={onAddFilter}
            sx={{ ml: 'auto', color: theme.textMuted, textTransform: 'none', fontWeight: 600 }}
          >
            Add Filter
          </Button>
        </Stack>
      </Box>
    );
  }

  const fieldById = useMemo(() => {
    const fields = source?.fields ?? [];
    const m = new Map<string, typeof fields[number]>();
    fields.forEach((f) => {
      m.set(f.id, f);
      if (f.technicalName) m.set(f.technicalName, f);
      if (f.name) m.set(f.name, f);
      if (f.displayName) m.set(f.displayName, f);
    });
    return m;
  }, [source]);

  return (
    <Box
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      sx={{
        px: 3,
        py: 1.5,
        borderBottom: `1px solid ${theme.border}`,
        display: 'flex',
        flexWrap: 'wrap',
        gap: 1,
        bgcolor: theme.backgroundElevated,
      }}
    >
      {filters.map((filter, index) => {
        const field = fieldById.get(filter.fieldId);
        return (
          <Chip
            key={`${filter.fieldId}-${index}`}
            label={
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Typography variant="caption" fontWeight={700} sx={{ color: theme.text }}>
                  {field?.displayName ?? filter.fieldId}
                </Typography>
                <Typography variant="caption" sx={{ color: theme.textMuted }}>
                  {OPERATOR_LABELS[filter.operator]}
                </Typography>
                <Typography variant="caption" fontWeight={600} sx={{ color: theme.textMuted }}>
                  {renderValues(filter.operator, filter.values)}
                </Typography>
              </Box>
            }
            onDelete={() => onRemoveFilter(index)}
            onClick={() => onEditFilter?.(index)}
            sx={{
              bgcolor: theme.backgroundElevated,
              border: `1px solid ${theme.border}`,
              borderRadius: 2,
              '& .MuiChip-deleteIcon': { fontSize: 16, color: theme.textMuted },
              boxShadow: '0 1px 1px rgba(0,0,0,0.04)',
              ...(filter.values.includes(theme.accent)
                ? { borderColor: theme.accent }
                : {}),
            }}
          />
        );
      })}
      <Button
        size="small"
        startIcon={<AddIcon />}
        onClick={onAddFilter}
        sx={{ color: theme.textMuted, textTransform: 'none', fontWeight: 600 }}
      >
        Add Filter
      </Button>
    </Box>
  );
};

export default FilterPillBar;
