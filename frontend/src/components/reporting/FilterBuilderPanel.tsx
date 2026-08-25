import React, { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import {
  Box, Typography, TextField,
  Chip, IconButton, Button, Tooltip, Collapse,
  Autocomplete,
} from '@mui/material';
import { useTheme } from '@mui/material/styles';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import CodeIcon from '@mui/icons-material/Code';
import FolderSpecialOutlinedIcon from '@mui/icons-material/FolderSpecialOutlined';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined';
import CloseIcon from '@mui/icons-material/Close';
import FunctionsIcon from '@mui/icons-material/Functions';
import FunctionPickerMenu from './filter/FunctionPickerMenu';
import ParameterCreatorPopover from './filter/ParameterCreatorPopover';
import { getOperatorsForFieldType, needsValue, needsListValues } from './operatorMetadata';
import { loadFilterModel, saveFilterModel, loadTenantDefaults } from './filterApi';
import { FilterModel, FilterCategory } from './filterTypes';
import {
  CategorySelector,
  ExpressionFilterEditor,
  HavingBuilder,
  QualifyBuilder,
  BitemporalBuilder,
} from './ExpressionFilterEditor';

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface BOField {
  name: string;
  label?: string;
  type?: string;
  technicalName?: string;
  _scope?: string;
  _subtypeKey?: string;
}

export interface FilterCondition {
  id: string;
  field: string;
  fieldLabel: string;
  dataType: string;
  operator: string;
  value: string;
  values: string[];
  parameter?: string;
  isParamBound?: boolean;
  enabled: boolean;
  exprMode?: boolean;
  rawExpression?: string;
  // Function-wrapped field expression, e.g. "YEAR(birthdate)"
  // When set, overrides `field` in generated SQL
  fieldExpr?: string;
  // Effective data type after function wrapping, drives operator selection
  effectiveDataType?: string;
}

export interface FilterGroup {
  id: string;
  combinator: 'AND' | 'OR';
  category?: FilterCategory;
  conditions: FilterCondition[];
  // optional sub-groups (one level of nesting for now)
  groups?: FilterGroup[];
}

interface FilterBuilderPanelProps {
  selectedBO: any;
  reportId?: string;
  parameters?: any[];
  isReadOnly?: boolean;
  onClone?: () => void;
  onChange?: (groups: FilterGroup[]) => void;
  onParametersChange?: (parameters: any[]) => void;
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

const uid = () => `${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;

const TYPE_META: Record<string, { label: string; color: string; bg: string }> = {
  string:    { label: 'Abc', color: '#10B981', bg: '#10B98118' },
  number:    { label: '#',   color: '#3B82F6', bg: '#3B82F618' },
  date:      { label: 'D',   color: '#F59E0B', bg: '#F59E0B18' },
  boolean:   { label: 'T/F', color: '#8B5CF6', bg: '#8B5CF618' },
  enum:      { label: 'E',   color: '#EC4899', bg: '#EC489918' },
};

const typeMeta = (t: string) => {
  const k = Object.keys(TYPE_META).find(k => (t || '').toLowerCase().includes(k));
  return k ? TYPE_META[k] : { label: '?', color: '#64748B', bg: '#64748B18' };
};

export const buildGroupSQL = (group: FilterGroup): string => {
  const parts = group.conditions.filter(c => c.enabled).map(c => {
    if (c.rawExpression) {
      return c.rawExpression;
    }
    if (!c.field && !c.fieldExpr) return '';
    const fieldRef = c.fieldExpr || `[${c.fieldLabel || c.field}]`;

    // Check if bound to parameter
    const isParam = c.isParamBound || Boolean(c.parameter) || (c.value && c.value.startsWith('@'));
    const paramRef = isParam ? (c.parameter ? (c.parameter.startsWith('@') ? c.parameter : `@${c.parameter}`) : c.value) : null;

    if (paramRef) {
      switch (c.operator) {
        case 'equals': return `${fieldRef} = ${paramRef}`;
        case 'not_equals': return `${fieldRef} != ${paramRef}`;
        case 'greater_than': return `${fieldRef} > ${paramRef}`;
        case 'less_than': return `${fieldRef} < ${paramRef}`;
        case 'greater_equal': return `${fieldRef} >= ${paramRef}`;
        case 'less_equal': return `${fieldRef} <= ${paramRef}`;
        case 'in': return `${fieldRef} IN (${paramRef})`;
        case 'not_in': return `${fieldRef} NOT IN (${paramRef})`;
        case 'contains': return `${fieldRef} LIKE '%' || ${paramRef} || '%'`;
        case 'starts_with': return `${fieldRef} LIKE ${paramRef} || '%'`;
        case 'ends_with': return `${fieldRef} LIKE '%' || ${paramRef}`;
        default: return `${fieldRef} = ${paramRef}`;
      }
    }

    switch (c.operator) {
      case 'is_null': return `${fieldRef} IS NULL`;
      case 'is_not_null': return `${fieldRef} IS NOT NULL`;
      case 'between': return `${fieldRef} BETWEEN '${c.values[0] || ''}' AND '${c.values[1] || ''}'`;
      case 'in': case 'not_in': {
        const vals = (c.values || []).map(v => `'${v}'`).join(', ');
        return `${fieldRef} ${c.operator === 'in' ? 'IN' : 'NOT IN'} (${vals})`;
      }
      case 'contains': return `${fieldRef} LIKE '%${c.value}%'`;
      case 'starts_with': return `${fieldRef} LIKE '${c.value}%'`;
      case 'ends_with': return `${fieldRef} LIKE '%${c.value}'`;
      case 'equals': return `${fieldRef} = '${c.value}'`;
      case 'not_equals': return `${fieldRef} != '${c.value}'`;
      case 'greater_than': return `${fieldRef} > ${c.value}`;
      case 'less_than': return `${fieldRef} < ${c.value}`;
      case 'greater_equal': return `${fieldRef} >= ${c.value}`;
      case 'less_equal': return `${fieldRef} <= ${c.value}`;
      case 'regexp_like': return `REGEXP_LIKE(${fieldRef}, '${c.value}')`;
      case 'array_contains': return `ARRAY_CONTAINS(${fieldRef}, '${c.value}')`;
      case 'json_extract': return `JSON_EXTRACT_SCALAR(${fieldRef}, '$.${c.value}')`;
      default: return `${fieldRef} ${c.operator.toUpperCase()} '${c.value}'`;
    }
  }).filter(Boolean);
  if (parts.length === 0) return '';
  const prefix = group.category && group.category !== 'WHERE' ? `${group.category} ` : '';
  return parts.length === 1 ? `${prefix}${parts[0]}` : `${prefix}(${parts.join(` ${group.combinator} `)})`;
};

export const buildSQL = (groups: FilterGroup[]): string => {
  const whereParts: string[] = [];
  const havingParts: string[] = [];
  const qualifyParts: string[] = [];
  const bitemporalParts: string[] = [];

  groups.forEach(g => {
    const s = buildGroupSQL(g);
    if (!s) return;
    const cat = g.category || 'WHERE';
    if (cat === 'HAVING') havingParts.push(s.replace(/^HAVING\s+/, ''));
    else if (cat === 'QUALIFY') qualifyParts.push(s.replace(/^QUALIFY\s+/, ''));
    else if (cat === 'BITEMPORAL') bitemporalParts.push(s.replace(/^BITEMPORAL\s+/, ''));
    else whereParts.push(s);
  });

  const lines: string[] = [];
  if (whereParts.length > 0) lines.push(`WHERE ${whereParts.join('\n  AND ')}`);
  if (bitemporalParts.length > 0) lines.push(`/* BITEMPORAL GUARDS */\n  AND ${bitemporalParts.join('\n  AND ')}`);
  if (havingParts.length > 0) lines.push(`HAVING ${havingParts.join('\n  AND ')}`);
  if (qualifyParts.length > 0) lines.push(`QUALIFY ${qualifyParts.join('\n  AND ')}`);

  return lines.join('\n');
};

const emptyGroup = (combinator: 'AND' | 'OR' = 'AND', category: FilterCategory = 'WHERE'): FilterGroup => ({
  id: uid(),
  combinator,
  category,
  conditions: [],
  groups: [],
});

const emptyCondition = (field?: BOField): FilterCondition => ({
  id: uid(),
  field: field?.name || '',
  fieldLabel: field?.label || field?.name || '',
  dataType: field?.type || 'string',
  operator: 'equals',
  value: '',
  values: [],
  enabled: true,
});

// ─────────────────────────────────────────────────────────────────────────────
// Combinator pill — click to toggle AND ↔ OR
// ─────────────────────────────────────────────────────────────────────────────

const CombinatorPill: React.FC<{
  value: 'AND' | 'OR';
  onChange: (v: 'AND' | 'OR') => void;
  vertical?: boolean;
  C: Record<string, string>;
}> = ({ value, onChange, vertical, C }) => (
  <Tooltip title={`Click to switch to ${value === 'AND' ? 'OR' : 'AND'}`} placement="right">
    <Box
      onClick={() => onChange(value === 'AND' ? 'OR' : 'AND')}
      sx={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        minWidth: vertical ? 28 : 40, minHeight: vertical ? 40 : 22,
        borderRadius: 1,
        bgcolor: value === 'AND' ? C.andBg : C.orBg,
        border: `1px solid ${value === 'AND' ? C.andColor + '50' : C.orColor + '50'}`,
        color: value === 'AND' ? C.andColor : C.orColor,
        fontSize: '0.6rem', fontWeight: 800, fontFamily: 'monospace',
        cursor: 'pointer', userSelect: 'none', letterSpacing: 1,
        writingMode: vertical ? 'vertical-rl' : 'horizontal-tb',
        transition: 'all 0.15s',
        '&:hover': {
          bgcolor: value === 'AND' ? C.andBg : C.orBg,
          boxShadow: `0 0 0 2px ${value === 'AND' ? C.andColor : C.orColor}40`,
        },
      }}
    >
      {value}
    </Box>
  </Tooltip>
);

// ─────────────────────────────────────────────────────────────────────────────
// Type badge
// ─────────────────────────────────────────────────────────────────────────────

const TypeBadge: React.FC<{ type: string }> = ({ type }) => {
  const meta = typeMeta(type);
  return (
    <Box sx={{
      minWidth: 26, height: 18, borderRadius: '3px', px: 0.4,
      bgcolor: meta.bg, border: `1px solid ${meta.color}40`,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      flexShrink: 0,
    }}>
      <Typography sx={{ fontSize: '0.55rem', fontWeight: 800, color: meta.color, fontFamily: 'monospace', lineHeight: 1 }}>
        {meta.label}
      </Typography>
    </Box>
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// Value input — smart based on operator type + @-aware parameter typeahead
// ─────────────────────────────────────────────────────────────────────────────

interface ValueInputProps {
  condition: FilterCondition;
  parameters?: any[];
  onUpdate: (u: Partial<FilterCondition>) => void;
  onOpenParamCreator?: (anchor: HTMLElement) => void;
  fieldLabel?: string;
  C: Record<string, string>;
}

const ValueInput: React.FC<ValueInputProps> = ({ condition, parameters = [], onUpdate, onOpenParamCreator, fieldLabel, C }) => {
  const noVal = !needsValue(condition.operator as any);
  const isList = needsListValues(condition.operator as any);
  const isDate = ['date', 'time', 'timestamp', 'datetime'].some(k => (condition.dataType || '').toLowerCase().includes(k));
  const isNum = ['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some(k => (condition.dataType || '').toLowerCase().includes(k));

  const isParamBound = condition.isParamBound || Boolean(condition.parameter) || (condition.value && condition.value.startsWith('@'));

  const inputSx = {
    '& .MuiOutlinedInput-root': {
      fontSize: '0.72rem', bgcolor: C.bg,
      '& fieldset': { borderColor: C.border },
      '&:hover fieldset': { borderColor: C.borderHover },
      '&.Mui-focused fieldset': { borderColor: C.accent },
    },
    '& .MuiOutlinedInput-input': { py: '5px', px: 1, color: C.text },
  };

  if (noVal) return (
    <Box sx={{ flex: 1, display: 'flex', alignItems: 'center' }}>
      <Typography sx={{ fontSize: '0.68rem', color: C.textMuted, fontStyle: 'italic' }}>— no value required —</Typography>
    </Box>
  );

  // ── Parameter typeahead options ────────────────────────────────────────────
  const paramOptions = parameters.map((p: any) => ({
    label: `@${p.name}`,
    sublabel: p.prompt || p.type,
    name: p.name,
  }));

  // Check if current value looks like a param reference
  const currentParamName = condition.value?.startsWith('@')
    ? condition.value.slice(1)
    : condition.parameter;

  // ── Inline param-creator option always available when field is selected ────
  interface ParamOption { label: string; sublabel?: string; name?: string; isCreateNew?: boolean; }

  const allParamOptions: ParamOption[] = [
    ...(onOpenParamCreator ? [{ label: '+ Create new parameter…', sublabel: `Based on: ${fieldLabel || condition.field}`, isCreateNew: true }] : []),
    ...paramOptions,
  ];

  // ── Between / not_between ─────────────────────────────────────────────────
  if (condition.operator === 'between' || condition.operator === 'not_between') {
    return (
      <Box sx={{ flex: 1, display: 'flex', gap: 0.5, alignItems: 'center' }}>
        <TextField size="small" placeholder="From" value={condition.values[0] || ''}
          type={isNum ? 'number' : isDate ? 'date' : 'text'}
          onChange={e => onUpdate({ values: [e.target.value, condition.values[1] || ''] })}
          sx={{ ...inputSx, flex: 1 }} InputLabelProps={isDate ? { shrink: true } : undefined} />
        <Typography sx={{ color: C.textMuted, fontSize: '0.65rem' }}>to</Typography>
        <TextField size="small" placeholder="To" value={condition.values[1] || ''}
          type={isNum ? 'number' : isDate ? 'date' : 'text'}
          onChange={e => onUpdate({ values: [condition.values[0] || '', e.target.value] })}
          sx={{ ...inputSx, flex: 1 }} InputLabelProps={isDate ? { shrink: true } : undefined} />
      </Box>
    );
  }

  // ── List (IN / NOT IN) ───────────────────────────────────────────────────
  if (isList) {
    return (
      <TextField size="small" placeholder="value1, value2, value3" value={condition.values.join(', ')}
        onChange={e => onUpdate({ values: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })}
        sx={{ ...inputSx, flex: 1 }}
        helperText={<span style={{ fontSize: '0.55rem', color: C.textMuted }}>comma-separated list</span>}
      />
    );
  }

  // ── Parameter-bound mode ──────────────────────────────────────────────────
  if (isParamBound) {
    return (
      <Autocomplete
        size="small"
        options={allParamOptions}
        value={allParamOptions.find(o => o.name === currentParamName) || null}
        onChange={(_, v) => {
          if (!v) return;
          if (v.isCreateNew) {
            const anchor = document.activeElement as HTMLElement;
            onOpenParamCreator?.(anchor);
            return;
          }
          onUpdate({ parameter: v.name, isParamBound: true, value: `@${v.name}` });
        }}
        getOptionLabel={o => o.label}
        isOptionEqualToValue={(o, v) => o.name === v.name && o.isCreateNew === v.isCreateNew}
        filterOptions={(options, state) => {
          const q = (state.inputValue || '').toLowerCase();
          if (!q) return options;
          return options.filter(o =>
            o.label.toLowerCase().includes(q) || (o.sublabel || '').toLowerCase().includes(q)
          );
        }}
        sx={{ flex: 1 }}
        renderInput={params => (
          <TextField
            {...params}
            placeholder="Select or search @param…"
            sx={inputSx}
            InputProps={{
              ...params.InputProps,
              startAdornment: (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mr: 0.5 }}>
                  <Chip
                    size="small"
                    label="@"
                    sx={{ height: 16, fontSize: '0.65rem', fontWeight: 800, bgcolor: 'rgba(13, 148, 136, 0.12)', color: '#0D9488' }}
                  />
                </Box>
              ),
            }}
          />
        )}
        renderOption={(props, o) => (
          <Box component="li" {...props} sx={{ fontSize: '0.72rem', py: 0.5, px: 1, '&.MuiAutocomplete-option': { py: 0 }, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography sx={{ fontSize: '0.72rem', fontFamily: 'monospace', color: o.isCreateNew ? '#00D4FF' : '#0D9488', fontWeight: 700 }}>
                {o.label}
              </Typography>
              {o.sublabel && (
                <Typography sx={{ fontSize: '0.6rem', color: C.textMuted }}>{o.sublabel}</Typography>
              )}
            </Box>
          </Box>
        )}
      />
    );
  }

  // ── Free-text mode with @ detection ───────────────────────────────────────
  return (
    <Autocomplete
      size="small"
      freeSolo
      options={allParamOptions}
      value={condition.value || ''}
      onChange={(_, v) => {
        if (typeof v === 'string') {
          onUpdate({ value: v, isParamBound: false, parameter: undefined });
          return;
        }
        if (!v) return;
        if (v.isCreateNew) {
          const anchor = document.activeElement as HTMLElement;
          onOpenParamCreator?.(anchor);
          return;
        }
        onUpdate({ parameter: v.name, isParamBound: true, value: `@${v.name}` });
      }}
      inputValue={condition.value || ''}
      onInputChange={(_, v) => {
        if (!v.startsWith('@')) {
          onUpdate({ value: v, isParamBound: false, parameter: undefined });
        }
      }}
      filterOptions={(options, state) => {
        const raw = state.inputValue || '';
        if (!raw.startsWith('@')) return [];
        const q = raw.slice(1).toLowerCase();
        if (!q) return options;
        return options.filter(o =>
          (o.label || '').toLowerCase().includes(`@${q}`) ||
          (o.name || '').toLowerCase().includes(q) ||
          (o.sublabel || '').toLowerCase().includes(q)
        );
      }}
      getOptionLabel={o => (typeof o === 'string' ? o : o.label)}
      sx={{ flex: 1 }}
      renderInput={params => (
        <TextField
          {...params}
          placeholder="Value or @param"
          type={isNum ? 'number' : isDate ? 'date' : 'text'}
          sx={inputSx}
          InputLabelProps={isDate ? { shrink: true } : undefined}
        />
      )}
      renderOption={(props, o) => {
        if (typeof o === 'string') return null;
        return (
          <Box component="li" {...props} sx={{ fontSize: '0.72rem', py: 0.5, px: 1, '&.MuiAutocomplete-option': { py: 0 }, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography sx={{ fontSize: '0.72rem', fontFamily: 'monospace', color: o.isCreateNew ? '#00D4FF' : '#0D9488', fontWeight: 700 }}>
                {o.label}
              </Typography>
              {o.sublabel && (
                <Typography sx={{ fontSize: '0.6rem', color: C.textMuted }}>{o.sublabel}</Typography>
              )}
            </Box>
          </Box>
        );
      }}
    />
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// Condition Row
// ─────────────────────────────────────────────────────────────────────────────

const ConditionRow: React.FC<{
  condition: FilterCondition;
  allFields: BOField[];
  parameters?: any[];
  isFirst: boolean;
  combinator: 'AND' | 'OR';
  onUpdate: (u: Partial<FilterCondition>) => void;
  onDelete: () => void;
  onDuplicate: () => void;
  onCreateParameter?: (param: { name: string; type: string; prompt: string; sourceType: string; defaultValue?: string }) => void;
  C: Record<string, string>;
}> = ({ condition, allFields, parameters = [], isFirst, combinator, onUpdate, onDelete, onDuplicate, onCreateParameter, C }) => {
  const effectiveDataType = condition.effectiveDataType || condition.dataType || 'string';
  const operators = getOperatorsForFieldType(effectiveDataType);

  const handleFieldChange = (name: string) => {
    const fd = allFields.find(f => f.name === name);
    const newOps = getOperatorsForFieldType(fd?.type || 'string');
    const newOp = newOps.some(o => o.id === condition.operator) ? condition.operator : (newOps[0]?.id || 'equals');
    onUpdate({ field: name, fieldLabel: fd?.label || fd?.name || name, dataType: fd?.type || 'string', operator: newOp, fieldExpr: undefined, effectiveDataType: undefined });
  };

  const fieldInputSx = {
    '& .MuiOutlinedInput-root': {
      fontSize: '0.72rem',
      color: C.text,
      bgcolor: C.bg,
      borderColor: C.border,
      '& fieldset': { borderColor: C.border },
      '&:hover fieldset': { borderColor: C.borderHover },
      '&.Mui-focused fieldset': { borderColor: C.accent },
    },
  };

  // Function picker state
  const [fxAnchor, setFxAnchor] = useState<HTMLElement | null>(null);

  // Parameter creator state
  const [paramCreatorAnchor, setParamCreatorAnchor] = useState<HTMLElement | null>(null);

  const handleFxSelect = (signature: string, returnType?: string) => {
    const fieldName = condition.field;
    const displayExpr = signature.includes('field')
      ? signature.replace(/field/gi, fieldName || 'field')
      : `${signature.split('(')[0]}(${fieldName})`;
    const ops = getOperatorsForFieldType(returnType || 'number');
    const newOp = ops.some(o => o.id === condition.operator) ? condition.operator : (ops[0]?.id || 'equals');
    onUpdate({ fieldExpr: displayExpr, effectiveDataType: returnType, operator: newOp });
  };

  const handleClearFx = () => {
    onUpdate({ fieldExpr: undefined, effectiveDataType: undefined });
  };

  const handleCreateParameter = (param: { name: string; type: string; prompt: string; sourceType: string; defaultValue?: string }) => {
    onCreateParameter?.(param);
    onUpdate({ parameter: param.name, isParamBound: true, value: `@${param.name}` });
  };

  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 0 }}>
      {/* Left combinator / first-row spacer */}
      <Box sx={{ width: 44, display: 'flex', flexDirection: 'column', alignItems: 'center', pt: 1, flexShrink: 0, gap: 0 }}>
        {isFirst ? (
          <Box sx={{ width: 1, height: 36 }} />
        ) : (
          <Box sx={{
            fontSize: '0.55rem', fontWeight: 800, fontFamily: 'monospace', letterSpacing: 0.8,
            color: combinator === 'AND' ? C.andColor : C.orColor,
            bgcolor: combinator === 'AND' ? C.andBg : C.orBg,
            border: `1px solid ${combinator === 'AND' ? C.andColor + '40' : C.orColor + '40'}`,
            borderRadius: '3px', px: 0.5, py: 0.2, mt: 1, lineHeight: 1,
          }}>
            {combinator}
          </Box>
        )}
      </Box>

      {/* Condition card */}
      <Box
        sx={{
          flex: 1, display: 'flex', alignItems: 'center', gap: 0.5,
          px: 1.25, py: 0.75, mb: 0.5,
          bgcolor: condition.enabled ? C.conditionBg : C.surface,
          border: `1px solid ${condition.enabled ? C.conditionBorder : C.border + '60'}`,
          borderRadius: '8px',
          opacity: condition.enabled ? 1 : 0.45,
          transition: 'all 0.15s',
          '&:hover': {
            borderColor: C.borderHover,
            bgcolor: condition.enabled ? C.conditionHover : C.surface,
            '& .row-actions': { opacity: 1 },
          },
        }}
      >
        <DragIndicatorIcon sx={{ fontSize: 14, color: C.textMuted, cursor: 'grab', flexShrink: 0 }} />

        {/* Effective type badge — shows wrapped type when fieldExpr is set */}
        <TypeBadge type={effectiveDataType || 'string'} />

        {/* Field typeahead */}
        <Autocomplete
          size="small"
          options={allFields}
          value={allFields.find(f => f.name === condition.field) || null}
          onChange={(_, v) => { if (v) handleFieldChange(v.name); }}
          getOptionLabel={f => f.label || f.name}
          isOptionEqualToValue={(o, v) => o.name === v.name}
          filterOptions={(options, state) => {
            const q = (state.inputValue || '').toLowerCase();
            if (!q) return options;
            return options.filter(f =>
              (f.label || '').toLowerCase().includes(q) ||
              (f.name || '').toLowerCase().includes(q) ||
              (f.technicalName || '').toLowerCase().includes(q)
            );
          }}
          renderInput={params => (
            <TextField
              {...params}
              placeholder="Select field…"
              sx={{ ...fieldInputSx, minWidth: 130, flex: '1 1 26%' }}
              InputProps={{
                ...params.InputProps,
                startAdornment: condition.fieldExpr ? (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mr: 0.5 }}>
                    <Chip
                      size="small"
                      label={condition.fieldExpr}
                      sx={{
                        height: 16, fontSize: '0.6rem', fontWeight: 700,
                        fontFamily: 'monospace', color: '#00D4FF',
                        bgcolor: 'rgba(0, 212, 255, 0.08)',
                        border: '1px solid rgba(0, 212, 255, 0.3)',
                        maxWidth: 160,
                        '& .MuiChip-label': { px: 0.75, overflow: 'hidden', textOverflow: 'ellipsis' },
                      }}
                    />
                  </Box>
                ) : null,
              }}
            />
          )}
          renderOption={(props, f) => (
            <Box component="li" {...props} sx={{ fontSize: '0.72rem', display: 'flex', gap: 0.75, alignItems: 'center', py: 0.5, px: 1, '&.MuiAutocomplete-option': { py: 0 } }}>
              <TypeBadge type={f.type || 'string'} />
              {f.label || f.name}
              {f.technicalName && f.technicalName !== f.name && (
                <Typography component="span" sx={{ fontSize: '0.6rem', color: C.textMuted, fontFamily: 'monospace' }}>
                  {f.technicalName}
                </Typography>
              )}
            </Box>
          )}
        />

        {/* fx button — wraps field in a function */}
        <Tooltip title="Apply a function to this field">
          <IconButton
            size="small"
            onClick={e => setFxAnchor(e.currentTarget)}
            disabled={!condition.field}
            sx={{
              p: 0.4,
              color: condition.fieldExpr ? '#00D4FF' : C.textMuted,
              bgcolor: condition.fieldExpr ? 'rgba(0, 212, 255, 0.08)' : 'transparent',
              border: condition.fieldExpr ? '1px solid rgba(0, 212, 255, 0.3)' : '1px solid transparent',
              '&:hover': { bgcolor: 'rgba(0, 212, 255, 0.12)', color: '#00D4FF' },
              '&.Mui-disabled': { color: C.textMuted, opacity: 0.4 },
            }}
          >
            <FunctionsIcon sx={{ fontSize: 14 }} />
          </IconButton>
        </Tooltip>

        {/* Clear fx button when fieldExpr is set */}
        {condition.fieldExpr && (
          <Tooltip title="Remove function wrapper">
            <IconButton
              size="small"
              onClick={handleClearFx}
              sx={{ p: 0.4, color: '#EF4444', '&:hover': { bgcolor: 'rgba(239, 68, 68, 0.1)' } }}
            >
              <CloseIcon sx={{ fontSize: 12 }} />
            </IconButton>
          </Tooltip>
        )}

        {/* Operator */}
        <Autocomplete
          size="small"
          options={operators}
          value={operators.find(o => o.id === condition.operator) || operators[0]}
          onChange={(_, v) => { if (v) onUpdate({ operator: v.id }); }}
          getOptionLabel={o => o.label}
          isOptionEqualToValue={(o, v) => o.id === v.id}
          disableClearable
          sx={{ minWidth: 130, flex: '0 1 auto' }}
          renderInput={params => (
            <TextField {...params} sx={fieldInputSx} />
          )}
          renderOption={(props, o) => (
            <Box component="li" {...props} sx={{ fontSize: '0.72rem', py: 0.5, px: 1, '&.MuiAutocomplete-option': { py: 0 } }}>
              {o.label}
            </Box>
          )}
        />

        {/* Value */}
        <Box sx={{ flex: '1 1 30%', minWidth: 80 }}>
          <ValueInput
            condition={condition}
            parameters={parameters}
            onUpdate={onUpdate}
            onOpenParamCreator={condition.field ? (anchor: HTMLElement) => setParamCreatorAnchor(anchor) : undefined}
            fieldLabel={condition.fieldLabel || condition.field}
            C={C}
          />
        </Box>

        {/* Actions */}
        <Box className="row-actions" sx={{ display: 'flex', gap: 0.25, flexShrink: 0, opacity: 0, transition: 'opacity 0.15s' }}>
          <Tooltip title={condition.exprMode ? 'Switch to builder' : 'Switch to expression code'}>
            <IconButton size="small" onClick={() => onUpdate({ exprMode: !condition.exprMode })}
              sx={{ p: 0.4, color: condition.exprMode ? C.accent : C.textMuted }}>
              <CodeIcon sx={{ fontSize: 13 }} />
            </IconButton>
          </Tooltip>
          <Tooltip title={condition.enabled ? 'Disable' : 'Enable'}>
            <IconButton size="small" onClick={() => onUpdate({ enabled: !condition.enabled })}
              sx={{ p: 0.4, color: condition.enabled ? '#10B981' : C.textMuted }}>
              {condition.enabled
                ? <VisibilityOutlinedIcon sx={{ fontSize: 13 }} />
                : <VisibilityOffOutlinedIcon sx={{ fontSize: 13 }} />}
            </IconButton>
          </Tooltip>
          <Tooltip title="Duplicate">
            <IconButton size="small" onClick={onDuplicate}
              sx={{ p: 0.4, color: C.textMuted, '&:hover': { color: C.text } }}>
              <ContentCopyIcon sx={{ fontSize: 12 }} />
            </IconButton>
          </Tooltip>
          <Tooltip title="Remove">
            <IconButton size="small" onClick={onDelete}
              sx={{ p: 0.4, color: C.textMuted, '&:hover': { color: '#EF4444' } }}>
              <DeleteOutlineIcon sx={{ fontSize: 14 }} />
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {/* Function picker menu */}
      <FunctionPickerMenu
        anchorEl={fxAnchor}
        onClose={() => setFxAnchor(null)}
        fieldName={condition.field}
        onSelect={handleFxSelect}
      />

      {/* Parameter creator popover */}
      <ParameterCreatorPopover
        anchorEl={paramCreatorAnchor}
        onClose={() => setParamCreatorAnchor(null)}
        field={{
          fieldLabel: condition.fieldLabel || condition.field || 'value',
          fieldName: condition.field || 'value',
          fieldType: effectiveDataType || 'string',
          fieldExpr: condition.fieldExpr,
        }}
        onSave={handleCreateParameter}
      />

      {/* Inline Expression Editor when in expression mode */}
      {condition.exprMode && (
        <Box sx={{ ml: 5, mb: 1, mr: 0.5 }}>
          <ExpressionFilterEditor
            filter={{
              id: condition.id,
              category: 'WHERE',
              enabled: condition.enabled,
              rawExpression: condition.rawExpression || (condition.field ? `${condition.field} = '${condition.value}'` : ''),
            }}
            fieldNames={allFields.map(f => f.name)}
            parameterNames={parameters.map((p: any) => p.name || p)}
            onUpdate={updates => onUpdate({ rawExpression: updates.rawExpression })}
            onClose={() => onUpdate({ exprMode: false })}
          />
        </Box>
      )}
    </Box>
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// Group Block
// ─────────────────────────────────────────────────────────────────────────────

const GroupBlock: React.FC<{
  group: FilterGroup;
  allFields: BOField[];
  parameters?: any[];
  isTopLevel?: boolean;
  onGroupChange: (g: FilterGroup) => void;
  onDelete?: () => void;
  onAddConditionFromDrop?: (field: BOField) => void;
  onCreateParameter?: (param: { name: string; type: string; prompt: string; sourceType: string; defaultValue?: string }) => void;
  C: Record<string, string>;
}> = ({ group, allFields, parameters = [], isTopLevel, onGroupChange, onDelete, onCreateParameter, C }) => {
  const [isDragOver, setIsDragOver] = useState(false);
  const [sqlOpen, setSqlOpen] = useState(false);

  const sql = useMemo(() => buildGroupSQL(group), [group]);
  const activeCount = group.conditions.filter(c => c.enabled).length;

  const updateCondition = useCallback((id: string, updates: Partial<FilterCondition>) => {
    onGroupChange({
      ...group,
      conditions: group.conditions.map(c => c.id === id ? { ...c, ...updates } : c),
    });
  }, [group, onGroupChange]);

  const deleteCondition = useCallback((id: string) => {
    onGroupChange({ ...group, conditions: group.conditions.filter(c => c.id !== id) });
  }, [group, onGroupChange]);

  const duplicateCondition = useCallback((id: string) => {
    const idx = group.conditions.findIndex(c => c.id === id);
    if (idx < 0) return;
    const copy = { ...group.conditions[idx], id: uid() };
    const next = [...group.conditions];
    next.splice(idx + 1, 0, copy);
    onGroupChange({ ...group, conditions: next });
  }, [group, onGroupChange]);

  const addCondition = useCallback((field?: BOField) => {
    onGroupChange({ ...group, conditions: [...group.conditions, emptyCondition(field)] });
  }, [group, onGroupChange]);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);
    try {
      const raw = e.dataTransfer.getData('application/json') || e.dataTransfer.getData('text/plain');
      if (!raw) return;
      const data = JSON.parse(raw);
      if (data?.type === 'bo-field-bundle' && Array.isArray(data.fields)) {
        const next = data.fields.map((f: BOField) => emptyCondition(f));
        onGroupChange({ ...group, conditions: [...group.conditions, ...next] });
      } else if (data?.type === 'bo-field' && data.field) {
        addCondition(data.field as BOField);
      } else if (Array.isArray(data)) {
        const next = data.map((f: BOField) => emptyCondition(f));
        onGroupChange({ ...group, conditions: [...group.conditions, ...next] });
      } else if (data && (data.name || data.technicalName)) {
        addCondition(data as BOField);
      }
    } catch {}
  }, [group, onGroupChange, addCondition]);

  const borderColor = isTopLevel ? C.border : (group.combinator === 'AND' ? C.andColor + '30' : C.orColor + '30');
  const accentColor = group.combinator === 'AND' ? C.andColor : C.orColor;

  return (
    <Box
      onDragOver={e => { e.preventDefault(); e.stopPropagation(); setIsDragOver(true); }}
      onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setIsDragOver(false); }}
      onDrop={handleDrop}
      sx={{
        border: `1px solid ${isDragOver ? accentColor : borderColor}`,
        borderRadius: '10px',
        bgcolor: isDragOver ? `${accentColor}08` : (isTopLevel ? 'transparent' : C.surface + '80'),
        transition: 'border-color 0.15s, background-color 0.15s',
        overflow: 'hidden',
      }}
    >
      {/* Group header */}
      <Box sx={{
        display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 0.75,
        borderBottom: `1px solid ${borderColor}`,
        bgcolor: isTopLevel ? C.surface : `${accentColor}08`,
      }}>
        {!isTopLevel && <FolderSpecialOutlinedIcon sx={{ fontSize: 13, color: accentColor, flexShrink: 0 }} />}

        {/* Combinator pill */}
        <CombinatorPill
          value={group.combinator}
          onChange={v => onGroupChange({ ...group, combinator: v })}
          C={C}
        />

        {/* Category selector */}
        <CategorySelector
          value={group.category || 'WHERE'}
          onChange={c => onGroupChange({ ...group, category: c })}
        />

        <Typography sx={{ fontSize: '0.65rem', color: C.textMuted, flex: 1 }}>
          {group.conditions.length === 0
            ? 'Drop fields or add condition'
            : `${activeCount} of ${group.conditions.length} active`}
        </Typography>

        {/* Add condition */}
        <Tooltip title="Add condition">
          <Button size="small" onClick={() => addCondition()}
            startIcon={<AddIcon sx={{ fontSize: 11 }} />}
            sx={{
              textTransform: 'none', fontSize: '0.65rem', py: 0.25, px: 1,
              color: C.accent, border: `1px solid ${C.accent}30`,
              '&:hover': { bgcolor: C.accent + '15' }, lineHeight: 1.5,
            }}>
            Add
          </Button>
        </Tooltip>

        {/* SQL preview toggle */}
        {sql && (
          <Tooltip title="Toggle SQL preview">
            <IconButton size="small" onClick={() => setSqlOpen(v => !v)}
              sx={{ p: 0.4, color: sqlOpen ? C.accent : C.textMuted }}>
              <VisibilityOutlinedIcon sx={{ fontSize: 13 }} />
            </IconButton>
          </Tooltip>
        )}

        {/* Delete group */}
        {!isTopLevel && onDelete && (
          <Tooltip title="Remove group">
            <IconButton size="small" onClick={onDelete}
              sx={{ p: 0.4, color: C.textMuted, '&:hover': { color: '#EF4444' } }}>
              <DeleteOutlineIcon sx={{ fontSize: 14 }} />
            </IconButton>
          </Tooltip>
        )}
      </Box>

      {/* Specialized Category Builders */}
      {group.category === 'HAVING' && (
        <Box sx={{ px: 1.5, py: 1, borderBottom: `1px solid ${borderColor}`, bgcolor: '#A78BFA06' }}>
          <HavingBuilder
            fieldNames={allFields.map(f => f.name)}
            expression={{
              id: `having_${group.id}`,
              category: 'HAVING',
              enabled: true,
              rawExpression: group.conditions[0]?.rawExpression,
            }}
            onUpdate={updates => {
              if (group.conditions.length === 0) {
                onGroupChange({ ...group, conditions: [{ ...emptyCondition(), ...updates, enabled: true }] });
              } else {
                updateCondition(group.conditions[0].id, updates);
              }
            }}
          />
        </Box>
      )}

      {group.category === 'QUALIFY' && (
        <Box sx={{ px: 1.5, py: 1, borderBottom: `1px solid ${borderColor}`, bgcolor: '#EC489906' }}>
          <QualifyBuilder
            fieldNames={allFields.map(f => f.name)}
            expression={{
              id: `qualify_${group.id}`,
              category: 'QUALIFY',
              enabled: true,
              rawExpression: group.conditions[0]?.rawExpression,
            }}
            onUpdate={updates => {
              if (group.conditions.length === 0) {
                onGroupChange({ ...group, conditions: [{ ...emptyCondition(), ...updates, enabled: true }] });
              } else {
                updateCondition(group.conditions[0].id, updates);
              }
            }}
          />
        </Box>
      )}

      {group.category === 'BITEMPORAL' && (
        <Box sx={{ px: 1.5, py: 1, borderBottom: `1px solid ${borderColor}`, bgcolor: '#F59E0B06' }}>
          <BitemporalBuilder
            expression={{
              id: `bitemporal_${group.id}`,
              category: 'BITEMPORAL',
              enabled: true,
              rawExpression: group.conditions[0]?.rawExpression,
            }}
            onUpdate={updates => {
              if (group.conditions.length === 0) {
                onGroupChange({ ...group, conditions: [{ ...emptyCondition(), ...updates, enabled: true }] });
              } else {
                updateCondition(group.conditions[0].id, updates);
              }
            }}
          />
        </Box>
      )}

      {/* SQL preview */}
      <Collapse in={sqlOpen}>
        <Box sx={{ px: 2, py: 1, bgcolor: '#020810', borderBottom: `1px solid ${C.border}` }}>
          <Typography sx={{ fontSize: '0.68rem', fontFamily: 'monospace', color: '#A5D8FF', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            {sql || '-- no conditions'}
          </Typography>
        </Box>
      </Collapse>

      {/* Empty drop zone */}
      {group.conditions.length === 0 && (
        <Box sx={{
          mx: 1.5, my: 1.5,
          border: `2px dashed ${isDragOver ? accentColor : C.border}`,
          borderRadius: '8px',
          p: 3, textAlign: 'center',
          bgcolor: isDragOver ? `${accentColor}06` : 'transparent',
          transition: 'all 0.15s',
        }}>
          <FilterAltIcon sx={{ fontSize: 24, color: isDragOver ? accentColor : C.textMuted, mb: 0.5 }} />
          <Typography sx={{ fontSize: '0.72rem', color: isDragOver ? accentColor : C.textMuted }}>
            {isDragOver ? 'Release to add filter' : 'Drag a field here or click Add'}
          </Typography>
        </Box>
      )}

      {/* Conditions */}
      {group.conditions.length > 0 && (
        <Box sx={{ px: 1, pt: 1, pb: 0.5 }}>
          {group.conditions.map((cond, idx) => (
            <ConditionRow
              key={cond.id}
              condition={cond}
              allFields={allFields}
              parameters={parameters}
              isFirst={idx === 0}
              combinator={group.combinator}
              onUpdate={updates => updateCondition(cond.id, updates)}
              onDelete={() => deleteCondition(cond.id)}
              onDuplicate={() => duplicateCondition(cond.id)}
              onCreateParameter={onCreateParameter}
              C={C}
            />
          ))}
        </Box>
      )}

      {/* Nested sub-groups */}
      {(group.groups || []).length > 0 && (
        <Box sx={{ px: 1.5, pb: 1.5, display: 'flex', flexDirection: 'column', gap: 1 }}>
          {(group.groups || []).map((subGroup, si) => (
            <Box key={subGroup.id} sx={{ display: 'flex', alignItems: 'flex-start', gap: 0 }}>
              {/* Nested combinator label */}
              <Box sx={{
                width: 44, display: 'flex', flexDirection: 'column', alignItems: 'center',
                pt: 1.5, flexShrink: 0,
              }}>
                {si > 0 && (
                  <Box sx={{
                    fontSize: '0.55rem', fontWeight: 800, fontFamily: 'monospace',
                    color: group.combinator === 'AND' ? C.andColor : C.orColor,
                    bgcolor: group.combinator === 'AND' ? C.andBg : C.orBg,
                    border: `1px solid ${group.combinator === 'AND' ? C.andColor + '40' : C.orColor + '40'}`,
                    borderRadius: '3px', px: 0.5, py: 0.2, letterSpacing: 0.8, lineHeight: 1,
                  }}>{group.combinator}</Box>
                )}
              </Box>
              <Box sx={{ flex: 1 }}>
                <GroupBlock
                  group={subGroup}
                  allFields={allFields}
                  parameters={parameters}
                  isTopLevel={false}
                  onGroupChange={updated => {
                    const next = [...(group.groups || [])];
                    next[si] = updated;
                    onGroupChange({ ...group, groups: next });
                  }}
                  onDelete={() => {
                    const next = (group.groups || []).filter((_, i) => i !== si);
                    onGroupChange({ ...group, groups: next });
                  }}
                  onCreateParameter={onCreateParameter}
                  C={C}
                />
              </Box>
            </Box>
          ))}
        </Box>
      )}
    </Box>
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// Main FilterBuilderPanel
// ─────────────────────────────────────────────────────────────────────────────

const FilterBuilderPanel: React.FC<FilterBuilderPanelProps> = ({
  selectedBO, reportId, parameters = [], isReadOnly = false, onClone, onChange, onParametersChange,
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const C = {
    bg: isDark ? '#050D1A' : theme.palette.background.paper,
    surface: isDark ? '#071526' : theme.palette.background.default,
    surface2: isDark ? '#0B1E36' : theme.palette.background.default,
    border: isDark ? '#1E293B' : theme.palette.divider,
    borderHover: isDark ? '#334155' : theme.palette.action.hover,
    accent: '#00D4FF',
    andBg: isDark ? 'rgba(0,212,255,0.12)' : 'rgba(0,212,255,0.08)',
    andColor: '#00D4FF',
    orBg: isDark ? 'rgba(245,158,11,0.14)' : 'rgba(245,158,11,0.1)',
    orColor: '#F59E0B',
    text: isDark ? '#E2E8F0' : theme.palette.text.primary,
    textMuted: isDark ? '#64748B' : theme.palette.text.disabled,
    textDim: isDark ? '#94A3B8' : theme.palette.text.secondary,
    dropZone: isDark ? 'rgba(0,212,255,0.06)' : 'rgba(0,212,255,0.04)',
    dropZoneBorder: isDark ? 'rgba(0,212,255,0.35)' : 'rgba(0,212,255,0.25)',
    conditionBg: isDark ? '#0B1E36' : theme.palette.background.paper,
    conditionBorder: isDark ? '#1E293B' : theme.palette.divider,
    conditionHover: isDark ? '#112236' : theme.palette.action.hover,
    pillBg: isDark ? '#0F2038' : theme.palette.background.paper,
  };

  const [groups, setGroups] = useState<FilterGroup[]>([emptyGroup('AND')]);
  const [showSQL, setShowSQL] = useState(false);
  const [isDragOver, setIsDragOver] = useState(false);
  const saveTimerRef = useRef<NodeJS.Timeout | null>(null);

  const handleCreateParameter = useCallback((param: { name: string; type: string; prompt: string; sourceType: string; defaultValue?: string }) => {
    const newParam = {
      id: `param_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
      name: param.name,
      type: param.type,
      prompt: param.prompt,
      sourceType: param.sourceType,
      defaultValue: param.defaultValue,
    };
    onParametersChange?.([...(parameters || []), newParam]);
  }, [parameters, onParametersChange]);

  // Load from backend
  useEffect(() => {
    if (!reportId) return;
    loadFilterModel(reportId).then(model => {
      if (!model?.groups?.length) return;
      const g: FilterGroup[] = model.groups.map((mg: any, gi: number) => ({
        id: mg.id || uid(),
        combinator: mg.combinator || 'AND',
        conditions: (mg.filters || []).map((f: any, fi: number) => ({
          id: f.id || `c_${gi}_${fi}`,
          field: f.field || '',
          fieldLabel: f.field || '',
          dataType: 'string',
          operator: f.operator || 'equals',
          value: f.valueSource?.value || '',
          values: f.values || [],
          enabled: f.enabled ?? true,
          fieldExpr: f.fieldExpr,
        })),
        groups: [],
      }));
      setGroups(g);
    }).catch(() => {});
  }, [reportId]);

  const schedulePersist = useCallback((g: FilterGroup[]) => {
    if (!reportId) return;
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveTimerRef.current = setTimeout(async () => {
      try {
        const defaults = await loadTenantDefaults();
        const model: FilterModel = {
          groupCombinator: 'AND',
          groups: g.map(grp => ({
            id: grp.id,
            combinator: grp.combinator,
            filters: grp.conditions.map(c => ({
              id: c.id,
              field: c.field,
              fieldScope: 'root',
              operator: c.operator as any,
              valueSource: { kind: 'constant' as const, value: c.value },
              values: c.values,
              enabled: c.enabled,
              fieldExpr: c.fieldExpr,
            })),
          })),
        };
        await saveFilterModel(reportId, model, parameters as any[], defaults);
      } catch {}
    }, 800);
  }, [reportId, parameters]);

  useEffect(() => () => { if (saveTimerRef.current) clearTimeout(saveTimerRef.current); }, []);

  const allFields = useMemo(() => {
    if (!selectedBO) return [];
    const fields: BOField[] = [];
    const seen = new Set<string>();
    const add = (f: any, scope = 'root', subtypeKey?: string) => {
      const k = `${scope}:${subtypeKey || ''}:${f.name || f.technicalName}`;
      if (!k || seen.has(k)) return;
      seen.add(k);
      fields.push({
        name: f.name || f.technicalName,
        label: f.label || f.displayName || f.name || f.technicalName,
        type: f.dataType || f.type || 'string',
        technicalName: f.technicalName,
        _scope: scope,
        _subtypeKey: subtypeKey,
      });
    };
    (selectedBO.coreFields || []).forEach((f: any) => add(f));
    (selectedBO.customFields || []).forEach((f: any) => add(f));
    (selectedBO.fields || []).forEach((f: any) => add(f));
    (selectedBO.config?.fields || []).forEach((f: any) => add(f));
    if (selectedBO.subtypes) {
      Object.entries(selectedBO.subtypes).forEach(([stKey, st]: [string, any]) => {
        (st.subtypeFields || []).forEach((f: any) => add(f, 'subtype', stKey));
      });
    }
    return fields;
  }, [selectedBO]);

  const updateGroups = useCallback((next: FilterGroup[]) => {
    setGroups(next);
    onChange?.(next);
    schedulePersist(next);
  }, [onChange, schedulePersist]);

  const addGroup = () => updateGroups([...groups, emptyGroup('OR')]);

  const totalActive = groups.reduce((s, g) => s + g.conditions.filter(c => c.enabled).length, 0);
  const totalAll = groups.reduce((s, g) => s + g.conditions.length, 0);

  const fullSQL = useMemo(() => buildSQL(groups), [groups]);

  // Top-level drag handler (drops onto the main panel → first group)
  const handleTopDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    // Route to first group if it exists
    if (groups.length === 0) return;
    try {
      const raw = e.dataTransfer.getData('application/json') || e.dataTransfer.getData('text/plain');
      if (!raw) return;
      const data = JSON.parse(raw);
      let fields: BOField[] = [];
      if (data?.type === 'bo-field-bundle' && Array.isArray(data.fields)) fields = data.fields;
      else if (data?.type === 'bo-field' && data.field) fields = [data.field];
      else if (Array.isArray(data)) fields = data;
      else if (data?.name || data?.technicalName) fields = [data];
      if (!fields.length) return;
      const newConds = fields.map(f => emptyCondition(f));
      const next = [...groups];
      next[0] = { ...next[0], conditions: [...next[0].conditions, ...newConds] };
      updateGroups(next);
    } catch {}
  }, [groups, updateGroups]);

  return (
    <Box
      sx={{ display: 'flex', width: '100%', height: '100%', flexDirection: 'column', overflow: 'hidden', bgcolor: C.bg }}
      onDragOver={e => { e.preventDefault(); setIsDragOver(true); }}
      onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setIsDragOver(false); }}
      onDrop={handleTopDrop}
    >
      {/* ── Header ─────────────────────────────────────────────────────── */}
      <Box sx={{
        display: 'flex', alignItems: 'center', gap: 1.5,
        px: 2, py: 1.25,
        borderBottom: `1px solid ${C.border}`,
        bgcolor: C.surface, flexShrink: 0,
      }}>
        <FilterAltIcon sx={{ fontSize: 16, color: C.accent }} />
        <Typography sx={{ fontSize: '0.78rem', fontWeight: 700, color: C.text, flex: 1 }}>
          Filter Builder
        </Typography>

        {totalAll > 0 && (
          <Chip
            size="small"
            label={`${totalActive} / ${totalAll} active`}
            sx={{
              height: 20, fontSize: '0.62rem', fontWeight: 700,
              bgcolor: C.andBg, color: C.accent,
              border: `1px solid ${C.accent}30`,
              '& .MuiChip-label': { px: 1 },
            }}
          />
        )}

        {/* SQL preview */}
        <Tooltip title="Preview SQL WHERE clause">
          <IconButton size="small" onClick={() => setShowSQL(v => !v)}
            sx={{ p: 0.5, color: showSQL ? C.accent : C.textMuted }}>
            <VisibilityOutlinedIcon sx={{ fontSize: 14 }} />
          </IconButton>
        </Tooltip>

        {/* Add group */}
        <Tooltip title="Add filter group">
          <Button size="small" onClick={addGroup}
            startIcon={<FolderSpecialOutlinedIcon sx={{ fontSize: 12 }} />}
            sx={{
              textTransform: 'none', fontSize: '0.65rem', py: 0.3, px: 1,
              color: C.textDim, border: `1px solid ${C.border}`,
              '&:hover': { bgcolor: C.surface2, borderColor: C.borderHover, color: C.text },
              lineHeight: 1.5,
            }}>
            Group
          </Button>
        </Tooltip>

        {/* Clear all */}
        {totalAll > 0 && (
          <Tooltip title="Clear all filters">
            <IconButton size="small" onClick={() => updateGroups([emptyGroup('AND')])}
              sx={{ p: 0.5, color: C.textMuted, '&:hover': { color: '#EF4444' } }}>
              <DeleteOutlineIcon sx={{ fontSize: 14 }} />
            </IconButton>
          </Tooltip>
        )}
      </Box>

      {/* ── SQL preview strip ──────────────────────────────────────────── */}
      <Collapse in={showSQL}>
        <Box sx={{
          px: 2, py: 1.25, bgcolor: '#020810',
          borderBottom: `1px solid ${C.border}`,
          flexShrink: 0,
        }}>
          <Typography sx={{ fontSize: '0.62rem', color: C.textMuted, mb: 0.5, letterSpacing: 0.5, textTransform: 'uppercase', fontWeight: 700 }}>
            Generated SQL Preview
          </Typography>
          <Typography sx={{
            fontSize: '0.7rem', fontFamily: 'monospace', color: '#A5D8FF',
            whiteSpace: 'pre-wrap', wordBreak: 'break-all', lineHeight: 1.7,
          }}>
            {fullSQL || '-- No filters defined'}
          </Typography>
        </Box>
      </Collapse>

      {/* ── Core Read-Only Banner ────────────────────────────────────────── */}
      {isReadOnly && (
        <Paper sx={{ p: 1.2, px: 2, m: 1.5, mb: 0, bgcolor: isDark ? 'rgba(245, 158, 11, 0.12)' : '#FEF3C7', border: '1px solid rgba(245, 158, 11, 0.35)', borderRadius: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
          <Typography variant="caption" sx={{ color: isDark ? '#FCD34D' : '#92400E', fontWeight: 600, fontSize: '0.75rem' }}>
            <strong>Core Report Template (Read-Only Filters):</strong> Filters cannot be changed directly on master reports. Click <strong>Clone</strong> to create a customizable copy for your tenant.
          </Typography>
          {onClone && (
            <Button
              size="small"
              variant="contained"
              onClick={onClone}
              sx={{ bgcolor: '#0D9488', color: '#FFF', fontSize: '0.7rem', fontWeight: 700, textTransform: 'none', py: 0.2, px: 1.2, borderRadius: 1, '&:hover': { bgcolor: '#0F766E' } }}
            >
              Clone Report
            </Button>
          )}
        </Paper>
      )}

      {/* ── Groups ────────────────────────────────────────────────────── */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 1.5, display: 'flex', flexDirection: 'column', gap: 1.5 }}>

        {/* Global drop hint when dragging */}
        {isDragOver && totalAll === 0 && (
          <Box sx={{
            border: `2px dashed ${C.accent}`,
            borderRadius: '10px', p: 3, textAlign: 'center',
            bgcolor: C.dropZone, pointerEvents: 'none',
          }}>
            <FilterAltIcon sx={{ fontSize: 28, color: C.accent, mb: 0.5 }} />
            <Typography sx={{ fontSize: '0.75rem', color: C.accent, fontWeight: 600 }}>
              Release to create filter
            </Typography>
          </Box>
        )}

        {groups.map((group, gi) => (
          <Box key={group.id} sx={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
            {/* Between-group combinator */}
            {gi > 0 && (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 1.5, py: 0.5 }}>
                <Box sx={{ flex: 1, height: '1px', bgcolor: C.border }} />
                <Chip
                  label="AND"
                  size="small"
                  sx={{
                    height: 20, fontSize: '0.6rem', fontWeight: 800, fontFamily: 'monospace',
                    bgcolor: C.andBg, color: C.andColor,
                    border: `1px solid ${C.andColor}40`,
                    '& .MuiChip-label': { px: 1 },
                  }}
                />
                <Box sx={{ flex: 1, height: '1px', bgcolor: C.border }} />
              </Box>
            )}

            <GroupBlock
              group={group}
              allFields={allFields}
              parameters={parameters}
              isTopLevel
              onGroupChange={updated => {
                const next = [...groups];
                next[gi] = updated;
                updateGroups(next);
              }}
              onDelete={groups.length > 1 ? () => updateGroups(groups.filter((_, i) => i !== gi)) : undefined}
              onCreateParameter={handleCreateParameter}
              C={C}
            />
          </Box>
        ))}

        {/* Footer spacer for scroll room */}
        <Box sx={{ minHeight: 32 }} />
      </Box>

      {/* ── Status bar ─────────────────────────────────────────────────── */}
      <Box sx={{
        px: 2, py: 0.75, borderTop: `1px solid ${C.border}`,
        bgcolor: C.surface, flexShrink: 0,
        display: 'flex', alignItems: 'center', gap: 1,
      }}>
        <Box sx={{
          width: 6, height: 6, borderRadius: '50%',
          bgcolor: totalActive > 0 ? '#10B981' : C.textMuted,
        }} />
        <Typography sx={{ fontSize: '0.65rem', color: C.textMuted }}>
          {totalActive > 0
            ? `${totalActive} active filter${totalActive !== 1 ? 's' : ''} across ${groups.length} group${groups.length !== 1 ? 's' : ''}`
            : 'No active filters — all records will be returned'}
        </Typography>
      </Box>
    </Box>
  );
};

export default FilterBuilderPanel;
