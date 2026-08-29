import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Divider,
  Select,
  MenuItem,
} from '@mui/material';
import {
  Numbers as NumberIcon,
  Abc as StringIcon,
  CalendarToday as DateIcon,
  ArrowUpward as AscIcon,
  ArrowDownward as DescIcon,
  FilterAlt as FilterAltIcon,
  Tune as TuneIcon,
  Functions as FunctionsIcon,
} from '@mui/icons-material';
import FunctionPickerMenu from '../../../components/reporting/filter/FunctionPickerMenu';
import type {
  DimensionSelection,
  ExplorerField,
  ExplorerSource,
  ExplorerQueryState,
  AggFn,
} from '../types/dataExplorerTypes';
import {
  EXPLORER_ACCENT,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';

interface QueryDefinitionBarProps {
  source: ExplorerSource;
  state: ExplorerQueryState;
  onToggleDimension?: (fieldId: string) => void;
  onToggleMeasure?: (fieldId: string, agg?: AggFn) => void;
  onAddTimeDimension?: (fieldId: string) => void;
  onRemoveDimension: (fieldId: string) => void;
  onRemoveMeasure: (fieldId: string) => void;
  onRemoveTimeDimension: (fieldId: string) => void;
  onUpdateMeasureAgg: (fieldId: string, agg: AggFn) => void;
  onUpdateDimensionExpression?: (fieldId: string, expression?: string) => void;
  onUpdateMeasureExpression?: (fieldId: string, expression?: string) => void;
  onToggleSort: (fieldId: string) => void;
  onLimitChange: (limit: number) => void;
  onOpenFilterModal?: () => void;
  onOpenParameterModal?: () => void;
}

function fieldIcon(type: ExplorerField['type']) {
  if (type === 'number') return <NumberIcon sx={{ fontSize: 14, color: '#f97316' }} />;
  if (type === 'date') return <DateIcon sx={{ fontSize: 14, color: '#a855f7' }} />;
  return <StringIcon sx={{ fontSize: 14, color: '#3b82f6' }} />;
}

const AGG_OPTIONS: AggFn[] = [
  'SUM',
  'AVG',
  'MIN',
  'MAX',
  'COUNT',
  'COUNT_DISTINCT',
  'ROW_NUMBER',
  'RANK',
  'DENSE_RANK',
  'LEAD',
  'LAG',
  'RUNNING_TOTAL',
  'PERCENT_OF_TOTAL',
  'HAVING',
  'NONE',
];

const SORT_ICON: Record<'asc' | 'desc', React.ReactNode> = {
  asc: <AscIcon sx={{ fontSize: 14 }} />,
  desc: <DescIcon sx={{ fontSize: 14 }} />,
};

export const QueryDefinitionBar: React.FC<QueryDefinitionBarProps> = ({
  source,
  state,
  onToggleDimension,
  onToggleMeasure,
  onAddTimeDimension,
  onRemoveDimension,
  onRemoveMeasure,
  onRemoveTimeDimension,
  onUpdateMeasureAgg,
  onUpdateDimensionExpression,
  onUpdateMeasureExpression,
  onToggleSort,
  onLimitChange,
  onOpenFilterModal,
  onOpenParameterModal,
}) => {
  const [fnMenuAnchor, setFnMenuAnchor] = React.useState<{ el: HTMLElement; fieldId: string; kind: 'dimension' | 'measure' } | null>(null);

  const fieldById = (() => {
    const m = new Map<string, ExplorerField>();
    source.fields.forEach((f) => {
      if (f.id) m.set(f.id, f);
      if (f.technicalName) m.set(f.technicalName, f);
      if (f.name) m.set(f.name, f);
      if (f.displayName) m.set(f.displayName, f);
    });
    return m;
  })();

  const handleDrop = (e: React.DragEvent) => {
    const rawData = e.dataTransfer.getData('application/json') || e.dataTransfer.getData('bo-field-bundle');
    const textPlain = e.dataTransfer.getData('text/plain');
    if (rawData || textPlain) {
      try {
        let fields: any[] = [];
        if (rawData) {
          const parsed = JSON.parse(rawData);
          if (Array.isArray(parsed)) {
            fields = parsed;
          } else if (parsed.type === 'bofield_batch' || parsed.type === 'bo-field-bundle') {
            fields = parsed.fields || [];
          } else if (parsed.type === 'bofield' && parsed.field) {
            fields = [parsed.field];
          } else if (parsed.fieldKey || parsed.name) {
            fields = [parsed];
          }
        }
        if (fields.length === 0 && textPlain) {
          const keys = textPlain.split(',').map(s => s.trim()).filter(Boolean);
          fields = keys.map(k => source.fields.find(f => f.name === k || f.id === k || f.technicalName === k) || { id: k, name: k, technicalName: k, displayName: k, type: 'string' });
        }

        if (fields.length > 0) {
          e.preventDefault();
          e.stopPropagation();
          fields.forEach((f) => {
            const fieldName = f.technicalName || f.name || f.id;
            const normType = (f.dataType || f.type || 'string').toLowerCase();
            if (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some((k) => normType.includes(k))) {
              if (onToggleMeasure) {
                onToggleMeasure(fieldName, 'SUM');
              } else {
                onUpdateMeasureAgg(fieldName, 'SUM');
              }
            } else if (['date', 'time', 'timestamp', 'datetime'].some((k) => normType.includes(k))) {
              if (onAddTimeDimension) {
                onAddTimeDimension(fieldName);
              } else if (onToggleDimension) {
                onToggleDimension(fieldName);
              }
            } else {
              if (onToggleDimension) {
                onToggleDimension(fieldName);
              }
            }
          });
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

  interface DimensionChip {
    itemFieldId: string;
    field: ExplorerField;
    granularity: DimensionSelection['granularity'];
    expression?: string;
  }
  interface TimeDimChip {
    itemFieldId: string;
    field: ExplorerField;
    granularity: import('../types/dataExplorerTypes').Granularity;
    expression?: string;
  }
  interface MeasureChip {
    itemFieldId: string;
    field: ExplorerField;
    agg: AggFn;
    expression?: string;
  }

  const dimensionChips: DimensionChip[] = state.dimensions
    .map((d) => {
      const f = fieldById.get(d.fieldId) || {
        id: d.fieldId,
        name: d.fieldId,
        displayName: d.fieldId,
        technicalName: d.fieldId,
        category: 'dimension' as const,
        type: 'string' as const,
      };
      return { itemFieldId: d.fieldId, field: f, granularity: d.granularity, expression: d.expression };
    });

  const timeDimChips: TimeDimChip[] = state.timeDimensions
    .map((t) => {
      const f = fieldById.get(t.fieldId) || {
        id: t.fieldId,
        name: t.fieldId,
        displayName: t.fieldId,
        technicalName: t.fieldId,
        category: 'time' as const,
        type: 'date' as const,
      };
      return { itemFieldId: t.fieldId, field: f, granularity: t.granularity ?? 'month', expression: t.expression };
    });

  const measureChips: MeasureChip[] = state.measures
    .map((m) => {
      const f = fieldById.get(m.fieldId) || {
        id: m.fieldId,
        name: m.fieldId,
        displayName: m.fieldId,
        technicalName: m.fieldId,
        category: 'measure' as const,
        type: 'number' as const,
      };
      return { itemFieldId: m.fieldId, field: f, agg: m.agg, expression: m.expression };
    });

  const sortByField = new Map(state.sorts.map((s) => [s.fieldId, s.direction]));

  const renderChip = (
    key: string,
    label: React.ReactNode,
    onRemove: () => void,
    sortDir?: 'asc' | 'desc',
    onSortClick?: () => void,
    onFnClick?: (e: React.MouseEvent<HTMLElement>) => void,
    expression?: string
  ) => {
    return (
      <Chip
        key={key}
        size="small"
        label={
          <Stack direction="row" alignItems="center" spacing={0.5}>
            <span>{label}</span>
            {expression && (
              <span style={{ fontSize: 10, opacity: 0.8, color: '#38BDF8', fontFamily: 'monospace' }}>
                {expression}
              </span>
            )}
            {onFnClick && (
              <Box
                component="span"
                onClick={(e) => {
                  e.stopPropagation();
                  onFnClick(e);
                }}
                sx={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  cursor: 'pointer',
                  p: 0.2,
                  borderRadius: 0.5,
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.2)' },
                }}
              >
                <FunctionsIcon sx={{ fontSize: 13, color: expression ? '#38BDF8' : 'inherit' }} />
              </Box>
            )}
            {sortDir && SORT_ICON[sortDir]}
          </Stack>
        }
        onDelete={onRemove}
        onClick={onSortClick}
        sx={{
          bgcolor: EXPLORER_ACCENT,
          color: EXPLORER_TEXT,
          fontWeight: 600,
          '& .MuiChip-deleteIcon': { color: EXPLORER_TEXT, opacity: 0.7 },
        }}
      />
    );
  };

  return (
    <Paper
      elevation={0}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      sx={{
        p: 1.5,
        borderBottom: `1px solid ${EXPLORER_BORDER}`,
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
      }}
    >
      <Stack direction="row" flexWrap="wrap" alignItems="center" gap={1}>
        {dimensionChips.length === 0 && (
          <Typography variant="caption" sx={{ color: EXPLORER_MUTED }}>
            Pick dimensions, measures, and time fields from the left to build a query.
          </Typography>
        )}

        {dimensionChips.map(({ itemFieldId, field, expression }) => {
          const sortDir = sortByField.get(itemFieldId) || sortByField.get(field.id);
          return renderChip(
            `d-${itemFieldId}`,
            <Stack direction="row" spacing={0.5} alignItems="center">
              {fieldIcon(field.type)}
              <span>{field.displayName}</span>
              {field.category === 'time' && (
                <span style={{ opacity: 0.6, fontWeight: 400 }}>({field.category})</span>
              )}
            </Stack>,
            () => onRemoveDimension(itemFieldId),
            sortDir,
            () => onToggleSort(itemFieldId),
            onUpdateDimensionExpression
              ? (e) => setFnMenuAnchor({ el: e.currentTarget, fieldId: itemFieldId, kind: 'dimension' })
              : undefined,
            expression
          );
        })}

        {timeDimChips.length > 0 && <Divider orientation="vertical" flexItem />}

        {timeDimChips.map(({ itemFieldId, field, granularity, expression }) => {
          const sortDir = sortByField.get(itemFieldId) || sortByField.get(field.id);
          return renderChip(
            `t-${itemFieldId}`,
            <Stack direction="row" spacing={0.5} alignItems="center">
              {fieldIcon(field.type)}
              <span>{field.displayName}</span>
              <span style={{ opacity: 0.6, fontWeight: 400 }}>({granularity})</span>
            </Stack>,
            () => onRemoveTimeDimension(itemFieldId),
            sortDir,
            () => onToggleSort(itemFieldId),
            onUpdateDimensionExpression
              ? (e) => setFnMenuAnchor({ el: e.currentTarget, fieldId: itemFieldId, kind: 'dimension' })
              : undefined,
            expression
          );
        })}

        {measureChips.length > 0 && (
          <>
            {dimensionChips.length + timeDimChips.length > 0 && <Divider orientation="vertical" flexItem />}
            <Typography variant="caption" sx={{ color: EXPLORER_MUTED, fontWeight: 700 }}>
              MEASURES
            </Typography>
            {measureChips.map(({ itemFieldId, field, agg, expression }) => {
              const sortDir = sortByField.get(itemFieldId) || sortByField.get(field.id);
              return (
                <Stack
                  key={`m-${itemFieldId}`}
                  direction="row"
                  alignItems="center"
                  sx={{
                    bgcolor: EXPLORER_ACCENT,
                    color: EXPLORER_TEXT,
                    borderRadius: 999,
                    px: 0.5,
                  }}
                >
                  <Select
                    size="small"
                    value={agg}
                    onChange={(e) => onUpdateMeasureAgg(itemFieldId, e.target.value as AggFn)}
                    variant="standard"
                    disableUnderline
                    sx={{
                      fontSize: 12,
                      fontWeight: 700,
                      color: EXPLORER_TEXT,
                      '& .MuiSelect-select': { padding: '4px 18px 4px 8px' },
                      '& .MuiSelect-icon': { color: EXPLORER_TEXT },
                    }}
                  >
                    {AGG_OPTIONS.map((opt) => (
                      <MenuItem key={opt} value={opt} sx={{ fontSize: 12 }}>
                        {opt}
                      </MenuItem>
                    ))}
                  </Select>
                  <Chip
                    size="small"
                    label={
                      <Stack direction="row" alignItems="center" spacing={0.5}>
                        <span>{field.displayName}</span>
                        {expression && (
                          <span style={{ fontSize: 10, opacity: 0.8, color: '#38BDF8', fontFamily: 'monospace' }}>
                            {expression}
                          </span>
                        )}
                        {onUpdateMeasureExpression && (
                          <Box
                            component="span"
                            onClick={(e) => {
                              e.stopPropagation();
                              setFnMenuAnchor({ el: e.currentTarget, fieldId: itemFieldId, kind: 'measure' });
                            }}
                            sx={{
                              display: 'inline-flex',
                              alignItems: 'center',
                              cursor: 'pointer',
                              p: 0.2,
                              borderRadius: 0.5,
                              '&:hover': { bgcolor: 'rgba(255,255,255,0.2)' },
                            }}
                          >
                            <FunctionsIcon sx={{ fontSize: 13, color: expression ? '#38BDF8' : 'inherit' }} />
                          </Box>
                        )}
                        {sortDir && SORT_ICON[sortDir]}
                      </Stack>
                    }
                    onClick={() => onToggleSort(itemFieldId)}
                    onDelete={() => onRemoveMeasure(itemFieldId)}
                    sx={{
                      ml: 0.5,
                      bgcolor: EXPLORER_ACCENT,
                      color: EXPLORER_TEXT,
                      fontWeight: 600,
                      '& .MuiChip-deleteIcon': { color: EXPLORER_TEXT, opacity: 0.7 },
                    }}
                  />
                </Stack>
              );
            })}
          </>
        )}

        {state.calculations && state.calculations.length > 0 && (
          <>
            <Divider orientation="vertical" flexItem />
            <Typography variant="caption" sx={{ color: '#0D9488', fontWeight: 700 }}>
              CALCULATIONS
            </Typography>
            {state.calculations.map((calc) => (
              <Chip
                key={`c-${calc.id}`}
                size="small"
                label={
                  <Stack direction="row" alignItems="center" spacing={0.5}>
                    <span>{calc.displayName}</span>
                    <span style={{ fontSize: 10, opacity: 0.7 }}>({calc.formula})</span>
                  </Stack>
                }
                sx={{
                  bgcolor: '#CCFBF1',
                  color: '#0F766E',
                  fontWeight: 600,
                  border: '1px solid #99F6E4',
                }}
              />
            ))}
          </>
        )}

        <Box sx={{ flexGrow: 1 }} />

        {onOpenFilterModal && (
          <Chip
            icon={<FilterAltIcon sx={{ fontSize: 14 }} />}
            label={`Filters (${state.filters.length})`}
            onClick={onOpenFilterModal}
            size="small"
            variant={state.filters.length > 0 ? 'filled' : 'outlined'}
            sx={{
              fontWeight: 600,
              cursor: 'pointer',
              bgcolor: state.filters.length > 0 ? 'rgba(59, 130, 246, 0.15)' : 'transparent',
              color: state.filters.length > 0 ? '#3b82f6' : EXPLORER_MUTED,
              borderColor: EXPLORER_BORDER,
            }}
          />
        )}

        {onOpenParameterModal && (
          <Chip
            icon={<TuneIcon sx={{ fontSize: 14 }} />}
            label={`Parameters (${(state.parameters || []).length})`}
            onClick={onOpenParameterModal}
            size="small"
            variant={(state.parameters || []).length > 0 ? 'filled' : 'outlined'}
            sx={{
              fontWeight: 600,
              cursor: 'pointer',
              bgcolor: (state.parameters || []).length > 0 ? 'rgba(168, 85, 247, 0.15)' : 'transparent',
              color: (state.parameters || []).length > 0 ? '#a855f7' : EXPLORER_MUTED,
              borderColor: EXPLORER_BORDER,
            }}
          />
        )}

        <Stack direction="row" alignItems="center" spacing={0.5} sx={{ ml: 1 }}>
          <Typography variant="caption" sx={{ color: EXPLORER_MUTED, fontWeight: 700 }}>
            LIMIT
          </Typography>
          <Select
            size="small"
            value={state.limit}
            onChange={(e) => onLimitChange(Number(e.target.value))}
            variant="standard"
            disableUnderline
            sx={{ fontSize: 13, fontWeight: 600, minWidth: 60 }}
          >
            {[100, 250, 500, 1000, 2500, 5000].map((opt) => (
              <MenuItem key={opt} value={opt} sx={{ fontSize: 13 }}>
                {opt.toLocaleString()}
              </MenuItem>
            ))}
          </Select>
        </Stack>
      </Stack>

      {/* Function Picker Menu popup */}
      {fnMenuAnchor && (
        <FunctionPickerMenu
          anchorEl={fnMenuAnchor.el}
          onClose={() => setFnMenuAnchor(null)}
          fieldName={(() => {
            const f = fieldById.get(fnMenuAnchor.fieldId);
            return f?.technicalName || f?.name || fnMenuAnchor.fieldId;
          })()}
          onSelect={(signature) => {
            if (fnMenuAnchor.kind === 'dimension' && onUpdateDimensionExpression) {
              onUpdateDimensionExpression(fnMenuAnchor.fieldId, signature);
            } else if (fnMenuAnchor.kind === 'measure' && onUpdateMeasureExpression) {
              onUpdateMeasureExpression(fnMenuAnchor.fieldId, signature);
            }
            setFnMenuAnchor(null);
          }}
        />
      )}
    </Paper>
  );
};

export default QueryDefinitionBar;
