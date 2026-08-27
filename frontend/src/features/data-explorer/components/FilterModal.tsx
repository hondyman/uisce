import React, { useEffect, useMemo, useState } from 'react';
import {
  Box,
  Typography,
  Button,
  TextField,
  IconButton,
  Paper,
  Chip,
  Stack,
  FormControlLabel,
  Switch,
  MenuItem,
  Select,
} from '@mui/material';
import {
  Close as CloseIcon,
  FilterAlt as FilterIcon,
  Tune as TuneIcon,
} from '@mui/icons-material';
import {
  EXPLORER_ACCENT,
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';
import type {
  ExplorerField,
  ExplorerSource,
  FilterOperator,
  FilterSelection,
  QueryParameter,
} from '../types/dataExplorerTypes';

interface FilterModalProps {
  open: boolean;
  onClose: () => void;
  source: ExplorerSource;
  parameters?: QueryParameter[];
  initialFieldId?: string;
  initialFilter?: FilterSelection | null;
  onApply: (filter: FilterSelection) => void;
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

const OPERATORS_BY_TYPE: Record<ExplorerField['type'], FilterOperator[]> = {
  string: ['equals', 'not_equals', 'contains', 'starts_with', 'ends_with', 'in', 'not_in', 'is_set', 'is_not_set'],
  number: ['equals', 'not_equals', 'gt', 'gte', 'lt', 'lte', 'between', 'is_set', 'is_not_set'],
  date: ['equals', 'not_equals', 'gt', 'gte', 'lt', 'lte', 'between', 'is_set', 'is_not_set'],
  boolean: ['equals', 'not_equals', 'is_set', 'is_not_set'],
  unknown: ['equals', 'not_equals', 'contains', 'in', 'not_in', 'is_set', 'is_not_set'],
};

const defaultOperatorForType = (type: ExplorerField['type']): FilterOperator => {
  if (type === 'date') return 'between';
  return 'equals';
};

const isFieldMatch = (f: ExplorerField, key?: string): boolean => {
  if (!key) return false;
  return (
    f.id === key ||
    f.name === key ||
    f.technicalName === key ||
    f.displayName === key
  );
};

export const FilterModal: React.FC<FilterModalProps> = ({
  open,
  onClose,
  source,
  parameters = [],
  initialFieldId,
  initialFilter,
  onApply,
}) => {
  const [fieldId, setFieldId] = useState<string>(initialFieldId || initialFilter?.fieldId || '');
  const [operator, setOperator] = useState<FilterOperator>(initialFilter?.operator || 'equals');
  const [values, setValues] = useState<string[]>(initialFilter?.values || ['']);
  const [isParamBound, setIsParamBound] = useState<boolean>(false);
  const [selectedParamName, setSelectedParamName] = useState<string>('');

  useEffect(() => {
    if (!open) return;
    setFieldId(initialFieldId || initialFilter?.fieldId || '');
    setOperator(initialFilter?.operator || 'equals');
    const initVals = initialFilter?.values || [''];
    setValues(initVals);
    
    // Check if initial filter is bound to a parameter (e.g. @ParamName)
    const firstVal = initVals[0] || '';
    if (firstVal.startsWith('@') && parameters.some(p => `@${p.name}` === firstVal)) {
      setIsParamBound(true);
      setSelectedParamName(firstVal);
    } else {
      setIsParamBound(false);
      setSelectedParamName(parameters[0] ? `@${parameters[0].name}` : '');
    }
  }, [open, initialFieldId, initialFilter, parameters]);

  const sortedFields = useMemo(
    () => [...source.fields].sort((a, b) => a.displayName.localeCompare(b.displayName)),
    [source.fields]
  );

  const selectedField = useMemo(
    () => sortedFields.find((f) => isFieldMatch(f, fieldId)),
    [sortedFields, fieldId]
  );

  const allowedOperators = useMemo(
    () => OPERATORS_BY_TYPE[selectedField?.type || 'unknown'],
    [selectedField]
  );

  useEffect(() => {
    if (!selectedField) return;
    if (!allowedOperators.includes(operator)) {
      setOperator(allowedOperators[0]);
    }
  }, [selectedField, allowedOperators, operator]);

  useEffect(() => {
    if (!selectedField || initialFilter) return;
    setOperator((prev) => (allowedOperators.includes(prev) ? prev : defaultOperatorForType(selectedField.type)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedField?.id, selectedField?.type]);

  if (!open) return null;

  const requiresValue = !['is_set', 'is_not_set'].includes(operator);
  const canApply =
    !!fieldId &&
    (operator === 'is_set' || operator === 'is_not_set' ||
      (isParamBound && !!selectedParamName) ||
      (values.length > 0 && values.some((v) => v.trim() !== '')));

  const handleApply = () => {
    if (!canApply) return;
    let finalValues: string[];
    if (operator === 'is_set' || operator === 'is_not_set') {
      finalValues = [];
    } else if (isParamBound && selectedParamName) {
      finalValues = [selectedParamName];
    } else if (operator === 'between') {
      finalValues = values.slice(0, 2);
    } else {
      finalValues = values.filter((v) => v.trim() !== '');
    }

    onApply({
      fieldId,
      operator,
      values: finalValues.length > 0 ? finalValues : [''],
    });
    onClose();
  };

  return (
    <Box
      sx={{
        position: 'fixed',
        inset: 0,
        zIndex: 1300,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        p: 2,
      }}
    >
      <Box
        sx={{ position: 'absolute', inset: 0, bgcolor: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(2px)' }}
        onClick={onClose}
      />
      <Paper
        elevation={24}
        sx={{
          position: 'relative',
          width: '100%',
          maxWidth: 540,
          borderRadius: 4,
          bgcolor: 'white',
          border: `1px solid ${EXPLORER_BORDER}`,
        }}
      >
        <Box
          sx={{
            px: 3,
            py: 2,
            borderBottom: `1px solid ${EXPLORER_BORDER}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box
              sx={{
                width: 40,
                height: 40,
                borderRadius: '50%',
                bgcolor: EXPLORER_BG,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <FilterIcon sx={{ color: EXPLORER_MUTED }} />
            </Box>
            <Box>
              <Typography variant="h6" fontWeight={700} sx={{ lineHeight: 1.2, color: EXPLORER_TEXT }}>
                {initialFilter ? 'Edit Filter' : 'Add Filter'}
              </Typography>
              <Typography variant="caption" sx={{ color: EXPLORER_MUTED }}>
                Refine query results or bind to workbook parameters
              </Typography>
            </Box>
          </Box>
          <IconButton onClick={onClose}>
            <CloseIcon />
          </IconButton>
        </Box>

        <Box sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2.5 }}>
          <Box>
            <Typography
              variant="caption"
              fontWeight={700}
              sx={{
                color: EXPLORER_MUTED,
                textTransform: 'uppercase',
                letterSpacing: 1,
                mb: 1,
                display: 'block',
              }}
            >
              Field
            </Typography>
            <TextField
              select
              fullWidth
              value={fieldId}
              onChange={(e) => setFieldId(e.target.value)}
              SelectProps={{ native: true }}
              sx={{
                '& .MuiOutlinedInput-root': {
                  borderRadius: 3,
                  bgcolor: EXPLORER_BG,
                  '& fieldset': { borderColor: EXPLORER_BORDER },
                },
              }}
            >
              <option value="" disabled>
                Select a field…
              </option>
              {sortedFields.map((f) => {
                const optValue = f.technicalName || f.name || f.id;
                return (
                  <option key={optValue} value={optValue}>
                    {f.displayName} ({f.category})
                  </option>
                );
              })}
            </TextField>
          </Box>

          {selectedField && (
            <Box sx={{ display: 'flex', gap: 2 }}>
              <Box sx={{ flex: 1 }}>
                <Typography
                  variant="caption"
                  fontWeight={700}
                  sx={{
                    color: EXPLORER_MUTED,
                    textTransform: 'uppercase',
                    letterSpacing: 1,
                    mb: 1,
                    display: 'block',
                  }}
                >
                  Operator
                </Typography>
                <TextField
                  select
                  fullWidth
                  value={operator}
                  onChange={(e) => setOperator(e.target.value as FilterOperator)}
                  SelectProps={{ native: true }}
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: 3,
                      bgcolor: EXPLORER_BG,
                      '& fieldset': { borderColor: EXPLORER_BORDER },
                    },
                  }}
                >
                  {allowedOperators.map((op) => (
                    <option key={op} value={op}>
                      {OPERATOR_LABELS[op]}
                    </option>
                  ))}
                </TextField>
              </Box>

              {requiresValue && (
                <Box sx={{ flex: 1 }}>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                    <Typography
                      variant="caption"
                      fontWeight={700}
                      sx={{
                        color: EXPLORER_MUTED,
                        textTransform: 'uppercase',
                        letterSpacing: 1,
                      }}
                    >
                      {operator === 'between' ? 'From' : (isParamBound ? 'Workbook Parameter' : 'Value')}
                    </Typography>
                    {parameters.length > 0 && (
                      <FormControlLabel
                        control={
                          <Switch
                            size="small"
                            checked={isParamBound}
                            onChange={(e) => setIsParamBound(e.target.checked)}
                          />
                        }
                        label={<Typography variant="caption" sx={{ fontSize: 10, fontWeight: 700 }}>Param</Typography>}
                        sx={{ m: 0 }}
                      />
                    )}
                  </Stack>

                  {isParamBound ? (
                    <TextField
                      select
                      fullWidth
                      value={selectedParamName}
                      onChange={(e) => setSelectedParamName(e.target.value)}
                      SelectProps={{ native: true }}
                      sx={{
                        '& .MuiOutlinedInput-root': {
                          borderRadius: 3,
                          bgcolor: EXPLORER_BG,
                          '& fieldset': { borderColor: EXPLORER_BORDER },
                        },
                      }}
                    >
                      {parameters.map((p) => (
                        <option key={p.id} value={`@${p.name}`}>
                          @{p.displayName || p.name} ({p.type})
                        </option>
                      ))}
                    </TextField>
                  ) : (
                    <TextField
                      fullWidth
                      placeholder={operator === 'in' || operator === 'not_in' ? 'value1, value2' : 'Enter value…'}
                      value={values[0] || ''}
                      onChange={(e) => setValues([e.target.value])}
                      sx={{
                        '& .MuiOutlinedInput-root': {
                          borderRadius: 3,
                          bgcolor: EXPLORER_BG,
                          '& fieldset': { borderColor: EXPLORER_BORDER },
                        },
                      }}
                    />
                  )}
                </Box>
              )}
            </Box>
          )}

          {selectedField && operator === 'between' && !isParamBound && (
            <Box>
              <Typography
                variant="caption"
                fontWeight={700}
                sx={{
                  color: EXPLORER_MUTED,
                  textTransform: 'uppercase',
                  letterSpacing: 1,
                  mb: 1,
                  display: 'block',
                }}
              >
                To
              </Typography>
              <TextField
                fullWidth
                placeholder="Enter end value…"
                value={values[1] || ''}
                onChange={(e) => setValues([values[0] || '', e.target.value])}
                sx={{
                  '& .MuiOutlinedInput-root': {
                    borderRadius: 3,
                    bgcolor: EXPLORER_BG,
                    '& fieldset': { borderColor: EXPLORER_BORDER },
                  },
                }}
              />
            </Box>
          )}

          {selectedField && (
            <Box
              sx={{
                p: 1.5,
                borderRadius: 2,
                bgcolor: 'rgba(249, 245, 6, 0.1)',
                border: '1px solid rgba(249, 245, 6, 0.2)',
                display: 'flex',
                gap: 1,
              }}
            >
              <Typography sx={{ fontSize: 16 }}>💡</Typography>
              <Typography variant="caption" sx={{ color: EXPLORER_TEXT }}>
                Filtering by <strong>{selectedField.displayName}</strong> {isParamBound ? `will dynamically evaluate against ${selectedParamName}.` : 'will be applied to query runs.'}
              </Typography>
            </Box>
          )}
        </Box>

        <Box
          sx={{
            p: 2,
            bgcolor: EXPLORER_BG,
            borderTop: `1px solid ${EXPLORER_BORDER}`,
            display: 'flex',
            justifyContent: 'flex-end',
            gap: 1.5,
          }}
        >
          <Button
            onClick={onClose}
            sx={{
              color: EXPLORER_MUTED,
              fontWeight: 700,
              textTransform: 'none',
              borderRadius: 999,
              px: 3,
              '&:hover': { bgcolor: 'white' },
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleApply}
            variant="contained"
            disabled={!canApply}
            sx={{
              bgcolor: EXPLORER_ACCENT,
              color: EXPLORER_TEXT,
              borderRadius: 999,
              fontWeight: 700,
              textTransform: 'none',
              px: 3,
              boxShadow: 'none',
              '&:hover': { bgcolor: '#e6e205', boxShadow: 'none' },
              '&:disabled': { bgcolor: EXPLORER_BORDER, color: EXPLORER_MUTED },
            }}
          >
            {initialFilter ? 'Save Filter' : 'Apply Filter'}
          </Button>
        </Box>
      </Paper>
    </Box>
  );
};

export default FilterModal;
