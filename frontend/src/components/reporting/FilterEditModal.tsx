import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions, Button,
  Box, Typography, TextField, Select, MenuItem, FormControl,
  InputLabel, Radio, RadioGroup, FormControlLabel, Autocomplete,
  Chip, Tooltip, Divider,
} from '@mui/material';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import CloseIcon from '@mui/icons-material/Close';
import { Filter, FilterOperator, ValueSource, TenantDefaults, TenantCalendar, ReportParameter } from './filterTypes';
import { OPERATOR_GROUPS, getOperatorsForFieldType, needsValue, needsListValues, getOperatorById, OperatorDef } from './operatorMetadata';

interface FilterEditModalProps {
  open: boolean;
  onClose: () => void;
  onSave: (filter: Partial<Filter>) => void;
  initialFilter?: Partial<Filter>;
  fields: Array<{ name: string; label: string; dataType: string; _scope?: string; _subtypeKey?: string }>;
  parameters: ReportParameter[];
  tenantDefaults: TenantDefaults;
  calendars: TenantCalendar[];
  allFieldsLabel?: string;
}

const VALUE_SOURCE_OPTIONS: { value: string; label: string }[] = [
  { value: 'constant', label: 'Constant' },
  { value: 'parameter', label: 'Parameter' },
  { value: 'function', label: 'Function Expression' },
  { value: 'tenant_default', label: 'Tenant Default' },
  { value: 'calendar', label: 'Calendar' },
];

const TENANT_DEFAULT_OPTIONS: { value: string; label: string }[] = [
  { value: 'default_calendar', label: 'Default Calendar' },
  { value: 'default_fiscal_year', label: 'Default Fiscal Year' },
  { value: 'default_region', label: 'Default Region' },
];

export const FilterEditModal: React.FC<FilterEditModalProps> = ({
  open, onClose, onSave, initialFilter,
  fields, parameters, tenantDefaults, calendars,
}) => {
  const [filter, setFilter] = useState<Partial<Filter>>({
    operator: 'equals',
    valueSource: { kind: 'constant', value: '' },
    values: [],
    enabled: true,
    ...initialFilter,
  });

  useEffect(() => {
    setFilter({ operator: 'equals', valueSource: { kind: 'constant', value: '' }, values: [], enabled: true, ...initialFilter });
  }, [initialFilter, open]);

  const selectedField = useMemo(
    () => fields.find(f => f.name === filter.field),
    [fields, filter.field]
  );

  const availableOperators = useMemo(() => {
    const dataType = selectedField?.dataType || 'string';
    return getOperatorsForFieldType(dataType);
  }, [selectedField]);

  const opDef = getOperatorById(filter.operator as FilterOperator);

  const handleFieldChange = (name: string) => {
    const field = fields.find(f => f.name === name);
    const ops = getOperatorsForFieldType(field?.dataType || 'string');
    const currentOpValid = ops.some(o => o.id === filter.operator);
    const newOp = currentOpValid ? filter.operator : (ops[0]?.id || 'equals');
    setFilter(prev => ({
      ...prev,
      field: name,
      operator: newOp as FilterOperator,
    }));
  };

  const handleSourceChange = (kind: string) => {
    const vs: ValueSource = { kind: kind as ValueSource['kind'] };
    if (kind === 'constant') vs.value = '';
    if (kind === 'parameter') { vs.parameterId = ''; vs.parameterName = ''; }
    if (kind === 'function') vs.expression = '';
    if (kind === 'tenant_default') vs.defaultKey = 'default_calendar';
    if (kind === 'calendar') vs.calendarCode = calendars[0]?.code || 'US';
    setFilter(prev => ({ ...prev, valueSource: vs }));
  };

  const handleSave = () => {
    if (!filter.field || !filter.operator) return;
    onSave(filter);
    onClose();
  };

  const isValid = !!filter.field && !!filter.operator && needsValue(filter.operator as FilterOperator)
    ? (filter.valueSource?.kind === 'constant' ? !!filter.valueSource?.value : true)
    : true;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <FilterAltIcon color="primary" />
          <Typography variant="h6" fontWeight={700}>
            {initialFilter?.id ? 'Edit Filter' : 'Add Filter'}
          </Typography>
        </Box>
        <Button onClick={onClose} size="small" sx={{ minWidth: 0 }}>
          <CloseIcon fontSize="small" />
        </Button>
      </DialogTitle>

      <Divider />

      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, pt: 2 }}>
        {/* Field Selector */}
        <FormControl fullWidth size="small">
          <InputLabel>Field</InputLabel>
          <Select
            value={filter.field || ''}
            label="Field"
            onChange={e => handleFieldChange(e.target.value)}
          >
            {fields.map(f => (
              <MenuItem key={`${f._scope || 'root'}-${f._subtypeKey || 'root'}-${f.name}`} value={f.name}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                    {f.label || f.name}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">{f.name}</Typography>
                </Box>
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        {/* Operator Selector */}
        <FormControl fullWidth size="small">
          <InputLabel>Operator</InputLabel>
          <Select
            value={filter.operator || ''}
            label="Operator"
            onChange={e => setFilter(prev => ({ ...prev, operator: e.target.value as FilterOperator }))}
          >
            {OPERATOR_GROUPS.map(group => {
              const groupOps = availableOperators.filter(o => o.groupLabel === group.label);
              if (groupOps.length === 0) return null;
              return (
                <MenuItem key={group.label} value={group.label} disabled sx={{ fontStyle: 'italic', opacity: 0.6, fontSize: '0.7rem' }}>
                  — {group.label} —
                </MenuItem>
              );
            })}
            {availableOperators.map(op => (
              <MenuItem key={op.id} value={op.id}>
                {op.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        {/* Value Source */}
        <Box>
          <Typography variant="caption" fontWeight={700} sx={{ textTransform: 'uppercase', letterSpacing: 1, color: 'text.secondary', mb: 0.5, display: 'block' }}>
            Value Source
          </Typography>
          <RadioGroup
            row
            value={filter.valueSource?.kind || 'constant'}
            onChange={e => handleSourceChange(e.target.value)}
          >
            {VALUE_SOURCE_OPTIONS.map(opt => (
              <FormControlLabel key={opt.value} value={opt.value} control={<Radio size="small" />} label={opt.label} sx={{ fontSize: '0.8rem' }} />
            ))}
          </RadioGroup>
        </Box>

        {/* Value Input based on source */}
        {filter.valueSource?.kind === 'constant' && needsValue(filter.operator as FilterOperator) && (
          <Box>
            <Typography variant="caption" fontWeight={700} sx={{ textTransform: 'uppercase', letterSpacing: 1, color: 'text.secondary', mb: 0.5, display: 'block' }}>
              Value
            </Typography>
            {needsListValues(filter.operator as FilterOperator) ? (
              <Box sx={{ display: 'flex', gap: 1 }}>
                {filter.operator === 'between' || filter.operator === 'not_between' ? (
                  <>
                    <TextField
                      size="small"
                      label="From"
                      value={filter.values?.[0] || ''}
                      onChange={e => setFilter(prev => ({ ...prev, values: [e.target.value, prev.values?.[1] || ''] }))}
                      fullWidth
                      type={selectedField?.dataType?.includes('date') ? 'date' : 'text'}
                    />
                    <TextField
                      size="small"
                      label="To"
                      value={filter.values?.[1] || ''}
                      onChange={e => setFilter(prev => ({ ...prev, values: [prev.values?.[0] || '', e.target.value] }))}
                      fullWidth
                      type={selectedField?.dataType?.includes('date') ? 'date' : 'text'}
                    />
                  </>
                ) : (
                  <TextField
                    size="small"
                    label="Values (comma-separated)"
                    value={(filter.values || []).join(', ')}
                    onChange={e => setFilter(prev => ({ ...prev, values: e.target.value.split(',').map(s => s.trim()).filter(Boolean) }))}
                    fullWidth
                    placeholder="e.g. APAC, EMEA, Americas"
                  />
                )}
              </Box>
            ) : (
              <TextField
                size="small"
                label="Value"
                value={filter.valueSource?.value || ''}
                onChange={e => setFilter(prev => ({ ...prev, valueSource: { ...prev.valueSource!, value: e.target.value } }))}
                fullWidth
                type={selectedField?.dataType?.includes('number') ? 'number' : 'text'}
              />
            )}
          </Box>
        )}

        {filter.valueSource?.kind === 'parameter' && (
          <FormControl fullWidth size="small">
            <InputLabel>Parameter</InputLabel>
            <Select
              value={filter.valueSource?.parameterId || ''}
              label="Parameter"
              onChange={e => setFilter(prev => ({
                ...prev,
                valueSource: { ...prev.valueSource!, parameterId: e.target.value, parameterName: `@${e.target.value}` },
              }))}
            >
              {parameters.map(p => (
                <MenuItem key={p.id} value={p.name}>
                  @{p.name} <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 1 }}>({p.type})</Typography>
                </MenuItem>
              ))}
              {parameters.length === 0 && (
                <MenuItem value="" disabled>No parameters defined</MenuItem>
              )}
            </Select>
            <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5 }}>
              Maps to the parameter value at report runtime
            </Typography>
          </FormControl>
        )}

        {filter.valueSource?.kind === 'function' && (
          <Box>
            <Typography variant="caption" fontWeight={700} sx={{ textTransform: 'uppercase', letterSpacing: 1, color: 'text.secondary', mb: 0.5, display: 'block' }}>
              Expression
            </Typography>
            <TextField
              size="small"
              label="Function Expression"
              value={filter.valueSource?.expression || ''}
              onChange={e => setFilter(prev => ({ ...prev, valueSource: { ...prev.valueSource!, expression: e.target.value } }))}
              fullWidth
              placeholder="e.g. DATESINPERIOD('trade_date', NOW(), -30, DAY)"
              multiline
              minRows={2}
              sx={{ fontFamily: 'monospace', '& .MuiInputBase-input': { fontFamily: 'monospace', fontSize: '0.8rem' } }}
            />
          </Box>
        )}

        {filter.valueSource?.kind === 'tenant_default' && (
          <FormControl fullWidth size="small">
            <InputLabel>Tenant Default</InputLabel>
            <Select
              value={filter.valueSource?.defaultKey || 'default_calendar'}
              label="Tenant Default"
              onChange={e => setFilter(prev => ({ ...prev, valueSource: { ...prev.valueSource!, defaultKey: e.target.value as any } }))}
            >
              {TENANT_DEFAULT_OPTIONS.map(opt => (
                <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
              ))}
            </Select>
            <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5 }}>
              Resolves to: {filter.valueSource?.defaultKey === 'default_calendar' ? tenantDefaults.defaultCalendarCode : filter.valueSource?.defaultKey === 'default_fiscal_year' ? String(tenantDefaults.defaultFiscalYear) : tenantDefaults.defaultRegion}
            </Typography>
          </FormControl>
        )}

        {filter.valueSource?.kind === 'calendar' && (
          <FormControl fullWidth size="small">
            <InputLabel>Calendar</InputLabel>
            <Select
              value={filter.valueSource?.calendarCode || calendars[0]?.code || 'US'}
              label="Calendar"
              onChange={e => setFilter(prev => ({ ...prev, valueSource: { ...prev.valueSource!, calendarCode: e.target.value } }))}
            >
              {calendars.map(c => (
                <MenuItem key={c.code} value={c.code}>{c.name} ({c.code})</MenuItem>
              ))}
              {calendars.length === 0 && (
                <>
                  <MenuItem value="US">US (Weekends only)</MenuItem>
                  <MenuItem value="NYSE">NYSE</MenuItem>
                  <MenuItem value="LSE">LSE</MenuItem>
                  <MenuItem value="TARGET2">TARGET2</MenuItem>
                </>
              )}
            </Select>
          </FormControl>
        )}

        {/* Enabled toggle */}
        <FormControlLabel
          control={
            <Radio
              checked={filter.enabled !== false}
              onChange={(_, checked) => setFilter(prev => ({ ...prev, enabled: checked }))}
              size="small"
            />
          }
          label="Filter enabled"
        />
      </DialogContent>

      <Divider />

      <DialogActions sx={{ px: 2, py: 1.5, justifyContent: 'space-between' }}>
        <Button onClick={onClose} size="small">Cancel</Button>
        <Button
          onClick={handleSave}
          variant="contained"
          disabled={!isValid}
          size="small"
        >
          {initialFilter?.id ? 'Update Filter' : 'Add Filter'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default FilterEditModal;
