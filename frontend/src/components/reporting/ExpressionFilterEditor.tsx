import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import {
  Box, Typography, TextField, Select, MenuItem, Chip, IconButton,
  Tooltip, Button, Collapse, Autocomplete, Paper,
} from '@mui/material';
import CodeIcon from '@mui/icons-material/Code';
import CloseIcon from '@mui/icons-material/Close';
import CheckIcon from '@mui/icons-material/Check';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';

import { ExprNode, ExpressionFilter, FilterCategory, FuncNode, FieldNode, BinaryNode, LiteralStr, MacroNode, AggNode } from './filterTypes';

// ─────────────────────────────────────────────────────────────────────────────
// Design tokens (matches FilterBuilderPanel dark theme)
// ─────────────────────────────────────────────────────────────────────────────

const C = {
  bg:          '#050D1A',
  surface:     '#071526',
  surface2:    '#0B1E36',
  border:      '#1E293B',
  borderHover: '#334155',
  accent:      '#00D4FF',
  andColor:    '#00D4FF',
  orColor:     '#F59E0B',
  text:        '#E2E8F0',
  textMuted:   '#64748B',
  textDim:     '#94A3B8',
  green:       '#10B981',
  yellow:      '#F59E0B',
  red:         '#EF4444',
  purple:      '#A78BFA',
};

// ─────────────────────────────────────────────────────────────────────────────
// Known SQL functions with autocomplete metadata
// ─────────────────────────────────────────────────────────────────────────────

interface FnMeta {
  label: string;
  signature: string;
  category: string;
  description: string;
}

const SQL_FUNCTIONS: FnMeta[] = [
  // String
  { label: 'SUBSTR',             signature: 'SUBSTR(field, start, length)',          category: 'String',      description: 'Extract substring' },
  { label: 'UPPER',              signature: 'UPPER(field)',                           category: 'String',      description: 'Convert to upper case' },
  { label: 'LOWER',              signature: 'LOWER(field)',                           category: 'String',      description: 'Convert to lower case' },
  { label: 'TRIM',               signature: 'TRIM(field)',                            category: 'String',      description: 'Remove leading/trailing spaces' },
  { label: 'REPLACE',            signature: 'REPLACE(field, old, new)',              category: 'String',      description: 'Replace occurrences' },
  { label: 'LENGTH',             signature: 'LENGTH(field)',                          category: 'String',      description: 'String length' },
  { label: 'REGEXP_LIKE',        signature: 'REGEXP_LIKE(field, pattern)',            category: 'String',      description: 'Regex match' },
  // Date
  { label: 'DATE_TRUNC',         signature: "DATE_TRUNC('month', field)",            category: 'Date',        description: 'Truncate date to period' },
  { label: 'DATE_ADD',           signature: "DATE_ADD(field, n, 'day')",             category: 'Date',        description: 'Add interval to date' },
  { label: 'EXTRACT',            signature: "EXTRACT('year' FROM field)",            category: 'Date',        description: 'Extract date part' },
  // Macro
  { label: 'TODAY',              signature: 'TODAY()',                                category: 'Date Macro',  description: 'Current date (UTC)' },
  { label: 'YESTERDAY',          signature: 'YESTERDAY()',                            category: 'Date Macro',  description: 'Yesterday (UTC)' },
  { label: 'MTD',                signature: 'MTD()',                                  category: 'Date Macro',  description: 'Month-to-date start' },
  { label: 'QTD',                signature: 'QTD()',                                  category: 'Date Macro',  description: 'Quarter-to-date start' },
  { label: 'YTD',                signature: 'YTD()',                                  category: 'Date Macro',  description: 'Year-to-date start' },
  { label: 'LAST_N_DAYS',        signature: 'LAST_N_DAYS(30)',                        category: 'Date Macro',  description: 'N days ago' },
  { label: 'T_MINUS',            signature: 'T_MINUS(2)',                             category: 'Date Macro',  description: 'T-N settlement business days' },
  { label: 'BUSINESS_DAYS_AGO',  signature: 'BUSINESS_DAYS_AGO(5)',                   category: 'Date Macro',  description: 'N business days ago' },
  // Null / conditional
  { label: 'COALESCE',           signature: 'COALESCE(a, b, c)',                      category: 'Null',        description: 'First non-null value' },
  { label: 'NULLIF',             signature: 'NULLIF(field, value)',                   category: 'Null',        description: 'Return NULL if equal' },
  // Numeric
  { label: 'ROUND',              signature: 'ROUND(field, decimals)',                 category: 'Numeric',     description: 'Round to decimal places' },
  { label: 'ABS',                signature: 'ABS(field)',                             category: 'Numeric',     description: 'Absolute value' },
  // Array / JSON
  { label: 'ARRAY_CONTAINS',     signature: 'ARRAY_CONTAINS(field, value)',           category: 'Array/JSON',  description: 'Array membership test' },
  { label: 'JSON_EXTRACT_SCALAR',signature: "JSON_EXTRACT_SCALAR(field, '$.key')",   category: 'Array/JSON',  description: 'Extract JSON scalar' },
  // Aggregate (HAVING)
  { label: 'SUM',                signature: 'SUM(field)',                             category: 'Aggregate',   description: 'Sum — for HAVING groups' },
  { label: 'COUNT',              signature: 'COUNT(*)',                               category: 'Aggregate',   description: 'Count — for HAVING groups' },
  { label: 'AVG',                signature: 'AVG(field)',                             category: 'Aggregate',   description: 'Average — for HAVING groups' },
  { label: 'MAX',                signature: 'MAX(field)',                             category: 'Aggregate',   description: 'Maximum — for HAVING groups' },
  { label: 'MIN',                signature: 'MIN(field)',                             category: 'Aggregate',   description: 'Minimum — for HAVING groups' },
  // Window (QUALIFY)
  { label: 'ROW_NUMBER',         signature: 'ROW_NUMBER() OVER(PARTITION BY col ORDER BY col DESC)', category: 'Window', description: 'Row deduplication — for QUALIFY groups' },
  { label: 'RANK',               signature: 'RANK() OVER(PARTITION BY col ORDER BY col DESC)',       category: 'Window', description: 'Rank — for QUALIFY groups' },
];

const CATEGORY_COLORS: Record<string, string> = {
  'String':      '#10B981',
  'Date':        '#F59E0B',
  'Date Macro':  '#F59E0B',
  'Null':        '#64748B',
  'Numeric':     '#3B82F6',
  'Array/JSON':  '#06B6D4',
  'Aggregate':   '#A78BFA',
  'Window':      '#EC4899',
};

// ─────────────────────────────────────────────────────────────────────────────
// FilterCategory selector chips
// ─────────────────────────────────────────────────────────────────────────────

interface CategorySelectorProps {
  value: FilterCategory;
  onChange: (c: FilterCategory) => void;
}

const CATEGORY_META: Record<FilterCategory, { label: string; color: string; bg: string; hint: string }> = {
  WHERE:      { label: 'WHERE',      color: C.andColor, bg: '#00D4FF12', hint: 'Standard row-level predicate pushed to SQL WHERE clause' },
  HAVING:     { label: 'HAVING',     color: C.purple,   bg: '#A78BFA12', hint: 'Post-aggregation filter — use aggregate functions (SUM, COUNT…)' },
  QUALIFY:    { label: 'QUALIFY',    color: '#EC4899',  bg: '#EC489912', hint: 'Window deduplication — use ROW_NUMBER() OVER(…) = 1' },
  BITEMPORAL: { label: 'BITEMPORAL', color: C.yellow,   bg: '#F59E0B12', hint: 'System-period temporal guard — as-of and knowledge-date injection' },
};

export const CategorySelector: React.FC<CategorySelectorProps> = ({ value, onChange }) => (
  <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
    {(Object.keys(CATEGORY_META) as FilterCategory[]).map(cat => {
      const m = CATEGORY_META[cat];
      const active = value === cat;
      return (
        <Tooltip key={cat} title={m.hint} placement="top">
          <Chip
            label={m.label}
            size="small"
            onClick={() => onChange(cat)}
            sx={{
              height: 20, fontSize: '0.6rem', fontWeight: 800, fontFamily: 'monospace',
              letterSpacing: 0.8, cursor: 'pointer',
              bgcolor: active ? m.bg : 'transparent',
              color: active ? m.color : C.textMuted,
              border: `1px solid ${active ? m.color + '60' : C.border}`,
              '&:hover': { bgcolor: m.bg, color: m.color },
              '& .MuiChip-label': { px: 0.75 },
              transition: 'all 0.12s',
            }}
          />
        </Tooltip>
      );
    })}
  </Box>
);

// ─────────────────────────────────────────────────────────────────────────────
// Expression editor for a single ExpressionFilter
// ─────────────────────────────────────────────────────────────────────────────

interface ExpressionFilterEditorProps {
  filter: ExpressionFilter;
  fieldNames: string[];
  parameterNames: string[];
  onUpdate: (updates: Partial<ExpressionFilter>) => void;
  onClose: () => void;
}

export const ExpressionFilterEditor: React.FC<ExpressionFilterEditorProps> = ({
  filter, fieldNames, parameterNames, onUpdate, onClose,
}) => {
  const [expr, setExpr] = useState(filter.rawExpression || '');
  const [showFunctions, setShowFunctions] = useState(false);
  const [fnSearch, setFnSearch] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  // Debounce update
  useEffect(() => {
    const t = setTimeout(() => onUpdate({ rawExpression: expr }), 400);
    return () => clearTimeout(t);
  }, [expr, onUpdate]);

  const filteredFns = useMemo(() =>
    SQL_FUNCTIONS.filter(f =>
      fnSearch === '' ||
      f.label.toLowerCase().includes(fnSearch.toLowerCase()) ||
      f.category.toLowerCase().includes(fnSearch.toLowerCase())
    ), [fnSearch]);

  const insertSnippet = (snippet: string) => {
    const input = inputRef.current;
    if (!input) { setExpr(e => e + snippet); return; }
    const start = input.selectionStart ?? expr.length;
    const end = input.selectionEnd ?? expr.length;
    const next = expr.slice(0, start) + snippet + expr.slice(end);
    setExpr(next);
    setTimeout(() => {
      input.focus();
      input.setSelectionRange(start + snippet.length, start + snippet.length);
    }, 0);
  };
  // Live Intellisense State
  const [cursorWord, setCursorWord] = useState('');
  const [caretPos, setCaretPos] = useState(0);
  const [selectedSuggestIdx, setSelectedSuggestIdx] = useState(0);

  // Suggestions computation based on current token being typed
  const suggestions = useMemo(() => {
    if (!cursorWord || cursorWord.length < 1) return [];
    const query = cursorWord.toLowerCase().replace(/^@/, '');
    const isParamQuery = cursorWord.startsWith('@');

    const list: Array<{ label: string; insertText: string; category: string; signature?: string; description?: string; color: string }> = [];

    // Param suggestions
    if (isParamQuery || query.length >= 1) {
      parameterNames.forEach(p => {
        if (p.toLowerCase().includes(query)) {
          list.push({
            label: `@${p}`,
            insertText: `@${p}`,
            category: 'Parameter',
            description: `Report Parameter @${p}`,
            color: C.purple,
          });
        }
      });
      ['Session.TenantID', 'Session.UserID', 'Session.UserRoles', 'Session.AllowedDesks'].forEach(s => {
        if (s.toLowerCase().includes(query)) {
          list.push({
            label: `@${s}`,
            insertText: `@${s}`,
            category: 'Session',
            description: `Session ABAC Variable`,
            color: '#EC4899',
          });
        }
      });
    }

    // Function suggestions
    if (!isParamQuery) {
      SQL_FUNCTIONS.forEach(fn => {
        if (fn.label.toLowerCase().includes(query) || fn.signature.toLowerCase().includes(query)) {
          list.push({
            label: fn.label,
            insertText: fn.signature,
            category: fn.category,
            signature: fn.signature,
            description: fn.description,
            color: CATEGORY_COLORS[fn.category] || C.accent,
          });
        }
      });

      // Field suggestions
      fieldNames.forEach(f => {
        if (f.toLowerCase().includes(query)) {
          list.push({
            label: f,
            insertText: f,
            category: 'Field',
            description: `BO Field [${f}]`,
            color: C.green,
          });
        }
      });
    }

    return list.slice(0, 10);
  }, [cursorWord, parameterNames, fieldNames]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    const pos = e.target.selectionStart || 0;
    setExpr(val);
    setCaretPos(pos);

    // Extract word before cursor
    const textBeforeCursor = val.slice(0, pos);
    const match = textBeforeCursor.match(/([@a-zA-Z0-9_]+)$/);
    if (match) {
      setCursorWord(match[1]);
      setSelectedSuggestIdx(0);
    } else {
      setCursorWord('');
    }
  };

  const applySuggestion = (item: { insertText: string }) => {
    const input = inputRef.current;
    const textBefore = expr.slice(0, caretPos);
    const textAfter = expr.slice(caretPos);
    const wordStart = textBefore.length - cursorWord.length;
    const nextExpr = expr.slice(0, wordStart) + item.insertText + textAfter;
    setExpr(nextExpr);
    setCursorWord('');

    setTimeout(() => {
      if (input) {
        input.focus();
        const newPos = wordStart + item.insertText.length;
        input.setSelectionRange(newPos, newPos);
      }
    }, 0);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (suggestions.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedSuggestIdx(i => (i + 1) % suggestions.length);
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedSuggestIdx(i => (i - 1 + suggestions.length) % suggestions.length);
        return;
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault();
        if (suggestions[selectedSuggestIdx]) {
          applySuggestion(suggestions[selectedSuggestIdx]);
        }
        return;
      }
      if (e.key === 'Escape') {
        setCursorWord('');
        return;
      }
    }
  };

  const textSx = {
    '& .MuiOutlinedInput-root': {
      fontSize: '0.78rem', fontFamily: 'monospace', bgcolor: '#020810',
      '& fieldset': { borderColor: C.border },
      '&:hover fieldset': { borderColor: C.borderHover },
      '&.Mui-focused fieldset': { borderColor: C.accent },
    },
    '& .MuiOutlinedInput-input': { color: '#A5D8FF', py: '6px', px: 1 },
  };

  return (
    <Box sx={{ bgcolor: '#020810', border: `1px solid ${C.accent}30`, borderRadius: '8px', overflow: 'hidden', position: 'relative' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 0.75, borderBottom: `1px solid ${C.border}` }}>
        <CodeIcon sx={{ fontSize: 13, color: C.accent }} />
        <Typography sx={{ fontSize: '0.65rem', fontWeight: 700, color: C.accent, flex: 1, letterSpacing: 0.5 }}>
          EXPRESSION MODE (INTELLISENSE ACTIVE)
        </Typography>
        <Tooltip title="Browse functions">
          <IconButton size="small" onClick={() => setShowFunctions(v => !v)}
            sx={{ p: 0.4, color: showFunctions ? C.accent : C.textMuted }}>
            <AutoAwesomeIcon sx={{ fontSize: 13 }} />
          </IconButton>
        </Tooltip>
        <Tooltip title="Close expression editor">
          <IconButton size="small" onClick={onClose} sx={{ p: 0.4, color: C.textMuted, '&:hover': { color: C.red } }}>
            <CloseIcon sx={{ fontSize: 13 }} />
          </IconButton>
        </Tooltip>
      </Box>

      {/* Expression input */}
      <Box sx={{ px: 1.5, py: 1, position: 'relative' }}>
        <TextField
          inputRef={inputRef}
          fullWidth
          multiline
          minRows={2}
          maxRows={6}
          placeholder={`Type function, field, or @Param (e.g. SUBSTR, TODAY, @TargetPrefix)...`}
          value={expr}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          sx={textSx}
        />

        {/* Live Intellisense Autocomplete Menu */}
        {suggestions.length > 0 && (
          <Paper
            elevation={8}
            sx={{
              position: 'absolute',
              top: '60px',
              left: '12px',
              right: '12px',
              zIndex: 999,
              bgcolor: '#071526',
              border: `1px solid ${C.accent}60`,
              borderRadius: '6px',
              maxHeight: 200,
              overflowY: 'auto',
              boxShadow: '0 10px 25px -5px rgba(0,0,0,0.8)',
            }}
          >
            {suggestions.map((item, idx) => (
              <Box
                key={`${item.label}_${idx}`}
                onMouseDown={(e) => {
                  e.preventDefault();
                  applySuggestion(item);
                }}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1,
                  px: 1.25,
                  py: 0.5,
                  cursor: 'pointer',
                  bgcolor: idx === selectedSuggestIdx ? `${C.accent}25` : 'transparent',
                  borderLeft: idx === selectedSuggestIdx ? `3px solid ${item.color}` : '3px solid transparent',
                  '&:hover': { bgcolor: `${C.accent}15` },
                }}
              >
                <Chip
                  size="small"
                  label={item.category}
                  sx={{
                    height: 16,
                    fontSize: '0.52rem',
                    bgcolor: `${item.color}20`,
                    color: item.color,
                    border: `1px solid ${item.color}40`,
                    fontWeight: 700,
                    '& .MuiChip-label': { px: 0.5 },
                  }}
                />
                <Typography sx={{ fontFamily: 'monospace', fontSize: '0.72rem', fontWeight: 700, color: item.color }}>
                  {item.label}
                </Typography>
                <Typography sx={{ fontSize: '0.62rem', color: C.textMuted, flex: 1, noWrap: true }}>
                  {item.description || item.signature}
                </Typography>
                <Typography sx={{ fontSize: '0.55rem', color: C.textMuted, fontFamily: 'monospace' }}>
                  Tab/↵
                </Typography>
              </Box>
            ))}
          </Paper>
        )}

        {/* Quick-insert pills: fields + params */}
        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mt: 0.75 }}>
          {fieldNames.slice(0, 8).map(fn => (
            <Chip key={fn} label={fn} size="small" onClick={() => insertSnippet(fn)}
              sx={{ height: 18, fontSize: '0.58rem', fontFamily: 'monospace', cursor: 'pointer',
                bgcolor: '#10B98112', color: C.green, border: `1px solid ${C.green}30`,
                '& .MuiChip-label': { px: 0.6 }, '&:hover': { bgcolor: '#10B98122' } }} />
          ))}
          {parameterNames.map(pn => (
            <Chip key={pn} label={`@${pn}`} size="small" onClick={() => insertSnippet(`@${pn}`)}
              sx={{ height: 18, fontSize: '0.58rem', fontFamily: 'monospace', cursor: 'pointer',
                bgcolor: '#A78BFA12', color: C.purple, border: `1px solid ${C.purple}30`,
                '& .MuiChip-label': { px: 0.6 }, '&:hover': { bgcolor: '#A78BFA22' } }} />
          ))}
        </Box>

        {/* Hint */}
        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', mt: 0.5 }}>
          <InfoOutlinedIcon sx={{ fontSize: 11, color: C.textMuted }} />
          <Typography sx={{ fontSize: '0.6rem', color: C.textMuted, lineHeight: 1.4 }}>
            Type to trigger <strong>Intellisense Typeahead</strong> •{' '}
            <span style={{ color: C.purple }}>@ParamName</span> for report params •{' '}
            <span style={{ color: '#EC4899' }}>@Session.TenantID</span> for session vars
          </Typography>
        </Box>
      </Box>

      {/* Function browser */}
      <Collapse in={showFunctions}>
        <Box sx={{ borderTop: `1px solid ${C.border}`, px: 1.5, py: 1 }}>
          <TextField
            fullWidth size="small" placeholder="Search functions…" value={fnSearch}
            onChange={e => setFnSearch(e.target.value)}
            sx={{ ...textSx, mb: 1, '& .MuiOutlinedInput-input': { ...textSx['& .MuiOutlinedInput-input'], color: C.text } }}
          />
          <Box sx={{ maxHeight: 220, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 0.25 }}>
            {filteredFns.map(fn => (
              <Box key={fn.label}
                onClick={() => insertSnippet(fn.signature)}
                sx={{
                  display: 'flex', alignItems: 'baseline', gap: 1, px: 1, py: 0.4,
                  borderRadius: '4px', cursor: 'pointer',
                  '&:hover': { bgcolor: `${CATEGORY_COLORS[fn.category] || C.accent}15` },
                }}>
                <Box sx={{
                  minWidth: 110, fontSize: '0.68rem', fontFamily: 'monospace', fontWeight: 700,
                  color: CATEGORY_COLORS[fn.category] || C.accent,
                }}>
                  {fn.label}
                </Box>
                <Typography sx={{ fontSize: '0.6rem', color: C.textMuted, flex: 1 }}>
                  {fn.description}
                </Typography>
                <Chip label={fn.category} size="small" sx={{
                  height: 14, fontSize: '0.5rem', flexShrink: 0,
                  bgcolor: `${CATEGORY_COLORS[fn.category] || C.accent}18`,
                  color: CATEGORY_COLORS[fn.category] || C.accent,
                  '& .MuiChip-label': { px: 0.5 },
                }} />
              </Box>
            ))}
          </Box>
        </Box>
      </Collapse>
    </Box>
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// HAVING group builder — aggregate function + comparison + value
// ─────────────────────────────────────────────────────────────────────────────

interface HavingBuilderProps {
  fieldNames: string[];
  expression: ExpressionFilter;
  onUpdate: (updates: Partial<ExpressionFilter>) => void;
}

export const HavingBuilder: React.FC<HavingBuilderProps> = ({ fieldNames, expression, onUpdate }) => {
  const [aggFn, setAggFn] = useState('SUM');
  const [field, setField] = useState(fieldNames[0] || '');
  const [op, setOp] = useState('>');
  const [val, setVal] = useState('');

  const syncPredicate = useCallback((fn: string, f: string, operator: string, v: string) => {
    const predicate = BinaryNode(operator, AggNode(fn, FieldNode(f)), LiteralStr(v));
    const raw = `${fn}(${f}) ${operator} ${v}`;
    onUpdate({ predicate, rawExpression: raw });
  }, [onUpdate]);

  return (
    <Box sx={{ display: 'flex', gap: 0.75, alignItems: 'center', flexWrap: 'wrap', px: 0.5, py: 0.75 }}>
      {/* Aggregate function */}
      <Select size="small" value={aggFn} onChange={e => { setAggFn(e.target.value); syncPredicate(e.target.value, field, op, val); }}
        sx={{ fontSize: '0.72rem', bgcolor: '#0B1E36', color: C.purple, minWidth: 80,
          '.MuiOutlinedInput-notchedOutline': { borderColor: C.border },
          '.MuiSelect-icon': { color: C.textMuted }, '.MuiSelect-select': { py: '5px !important' } }}>
        {['SUM', 'COUNT', 'AVG', 'MAX', 'MIN'].map(f => (
          <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>
        ))}
      </Select>
      <Typography sx={{ fontSize: '0.72rem', color: C.textMuted }}>of</Typography>
      {/* Field */}
      <Select size="small" value={field} onChange={e => { setField(e.target.value); syncPredicate(aggFn, e.target.value, op, val); }}
        sx={{ fontSize: '0.72rem', bgcolor: '#0B1E36', color: C.text, minWidth: 110,
          '.MuiOutlinedInput-notchedOutline': { borderColor: C.border },
          '.MuiSelect-icon': { color: C.textMuted }, '.MuiSelect-select': { py: '5px !important' } }}>
        {fieldNames.map(f => <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>)}
      </Select>
      {/* Operator */}
      <Select size="small" value={op} onChange={e => { setOp(e.target.value); syncPredicate(aggFn, field, e.target.value, val); }}
        sx={{ fontSize: '0.72rem', bgcolor: '#0B1E36', color: C.text, minWidth: 60,
          '.MuiOutlinedInput-notchedOutline': { borderColor: C.border },
          '.MuiSelect-icon': { color: C.textMuted }, '.MuiSelect-select': { py: '5px !important' } }}>
        {['>', '<', '>=', '<=', '=', '!='].map(o => <MenuItem key={o} value={o} sx={{ fontSize: '0.72rem' }}>{o}</MenuItem>)}
      </Select>
      {/* Value */}
      <TextField size="small" placeholder="value" value={val}
        onChange={e => { setVal(e.target.value); syncPredicate(aggFn, field, op, e.target.value); }}
        sx={{ flex: 1, minWidth: 80,
          '& .MuiOutlinedInput-root': { fontSize: '0.72rem', bgcolor: '#0B1E36',
            '& fieldset': { borderColor: C.border }, '&.Mui-focused fieldset': { borderColor: C.purple } },
          '& .MuiOutlinedInput-input': { color: C.text, py: '5px' },
        }} />
    </Box>
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// QUALIFY group builder — ROW_NUMBER() OVER(PARTITION BY … ORDER BY … = 1)
// ─────────────────────────────────────────────────────────────────────────────

interface QualifyBuilderProps {
  fieldNames: string[];
  expression: ExpressionFilter;
  onUpdate: (updates: Partial<ExpressionFilter>) => void;
}

export const QualifyBuilder: React.FC<QualifyBuilderProps> = ({ fieldNames, expression, onUpdate }) => {
  const [partitionField, setPartitionField] = useState(fieldNames[0] || '');
  const [orderField, setOrderField] = useState(fieldNames[1] || fieldNames[0] || '');
  const [desc, setDesc] = useState(true);
  const [rankN, setRankN] = useState('1');

  const sync = useCallback((pf: string, of: string, d: boolean, n: string) => {
    const raw = `ROW_NUMBER() OVER(PARTITION BY ${pf} ORDER BY ${of}${d ? ' DESC' : ''}) = ${n}`;
    onUpdate({ rawExpression: raw });
  }, [onUpdate]);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75, px: 0.5, py: 0.75 }}>
      <Box sx={{ display: 'flex', gap: 0.75, alignItems: 'center', flexWrap: 'wrap' }}>
        <Typography sx={{ fontSize: '0.65rem', color: C.textMuted, minWidth: 80, fontFamily: 'monospace' }}>
          PARTITION BY
        </Typography>
        <Select size="small" value={partitionField} onChange={e => { setPartitionField(e.target.value); sync(e.target.value, orderField, desc, rankN); }}
          sx={{ fontSize: '0.72rem', bgcolor: '#0B1E36', color: C.text, flex: 1,
            '.MuiOutlinedInput-notchedOutline': { borderColor: C.border },
            '.MuiSelect-icon': { color: C.textMuted }, '.MuiSelect-select': { py: '5px !important' } }}>
          {fieldNames.map(f => <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>)}
        </Select>
      </Box>
      <Box sx={{ display: 'flex', gap: 0.75, alignItems: 'center', flexWrap: 'wrap' }}>
        <Typography sx={{ fontSize: '0.65rem', color: C.textMuted, minWidth: 80, fontFamily: 'monospace' }}>
          ORDER BY
        </Typography>
        <Select size="small" value={orderField} onChange={e => { setOrderField(e.target.value); sync(partitionField, e.target.value, desc, rankN); }}
          sx={{ fontSize: '0.72rem', bgcolor: '#0B1E36', color: C.text, flex: 1,
            '.MuiOutlinedInput-notchedOutline': { borderColor: C.border },
            '.MuiSelect-icon': { color: C.textMuted }, '.MuiSelect-select': { py: '5px !important' } }}>
          {fieldNames.map(f => <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>)}
        </Select>
        <Chip label={desc ? 'DESC' : 'ASC'} size="small" onClick={() => { setDesc(d => { sync(partitionField, orderField, !d, rankN); return !d; }); }}
          sx={{ height: 20, fontSize: '0.6rem', fontWeight: 700, cursor: 'pointer',
            bgcolor: desc ? '#EC489912' : '#10B98112', color: desc ? '#EC4899' : C.green,
            border: `1px solid ${desc ? '#EC489940' : C.green + '40'}`, '& .MuiChip-label': { px: 0.75 } }} />
      </Box>
      <Box sx={{ display: 'flex', gap: 0.75, alignItems: 'center' }}>
        <Typography sx={{ fontSize: '0.65rem', color: C.textMuted, minWidth: 80, fontFamily: 'monospace' }}>
          = rank
        </Typography>
        <TextField size="small" value={rankN} type="number"
          onChange={e => { setRankN(e.target.value); sync(partitionField, orderField, desc, e.target.value); }}
          inputProps={{ min: 1 }}
          sx={{ width: 70,
            '& .MuiOutlinedInput-root': { fontSize: '0.72rem', bgcolor: '#0B1E36',
              '& fieldset': { borderColor: C.border }, '&.Mui-focused fieldset': { borderColor: '#EC4899' } },
            '& .MuiOutlinedInput-input': { color: C.text, py: '5px' } }} />
      </Box>
      {/* Preview */}
      <Box sx={{ fontFamily: 'monospace', fontSize: '0.62rem', color: '#A5D8FF', bgcolor: '#020810',
        border: `1px solid ${C.border}`, borderRadius: '4px', px: 1, py: 0.5 }}>
        {`ROW_NUMBER() OVER(PARTITION BY ${partitionField} ORDER BY ${orderField}${desc ? ' DESC' : ''}) = ${rankN}`}
      </Box>
    </Box>
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// Bitemporal group builder — AS-OF and KNOWLEDGE-DATE pickers
// ─────────────────────────────────────────────────────────────────────────────

interface BitemporalBuilderProps {
  expression: ExpressionFilter;
  onUpdate: (updates: Partial<ExpressionFilter>) => void;
}

export const BitemporalBuilder: React.FC<BitemporalBuilderProps> = ({ expression, onUpdate }) => {
  const [mode, setMode] = useState<'session' | 'override'>('session');
  const [overrideDate, setOverrideDate] = useState('');

  const sync = (m: 'session' | 'override', d: string) => {
    const raw = m === 'session'
      ? `system_valid_from <= @Session.AsOfDate AND system_valid_to > @Session.AsOfDate`
      : `system_valid_from <= '${d}' AND system_valid_to > '${d}'`;
    onUpdate({ rawExpression: raw });
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, px: 0.5, py: 0.75 }}>
      <Box sx={{ display: 'flex', gap: 0.5 }}>
        {(['session', 'override'] as const).map(m => (
          <Chip key={m} label={m === 'session' ? 'JWT Session Date' : 'Override Date'} size="small"
            onClick={() => { setMode(m); sync(m, overrideDate); }}
            sx={{ height: 20, fontSize: '0.6rem', fontWeight: 700, cursor: 'pointer',
              bgcolor: mode === m ? '#F59E0B12' : 'transparent',
              color: mode === m ? C.yellow : C.textMuted,
              border: `1px solid ${mode === m ? C.yellow + '50' : C.border}`,
              '& .MuiChip-label': { px: 0.75 } }} />
        ))}
      </Box>
      {mode === 'override' && (
        <TextField size="small" type="date" value={overrideDate}
          onChange={e => { setOverrideDate(e.target.value); sync('override', e.target.value); }}
          InputLabelProps={{ shrink: true }}
          sx={{ width: 180,
            '& .MuiOutlinedInput-root': { fontSize: '0.72rem', bgcolor: '#0B1E36',
              '& fieldset': { borderColor: C.border }, '&.Mui-focused fieldset': { borderColor: C.yellow } },
            '& .MuiOutlinedInput-input': { color: C.text, py: '5px' } }} />
      )}
      <Box sx={{ fontFamily: 'monospace', fontSize: '0.62rem', color: '#A5D8FF', bgcolor: '#020810',
        border: `1px solid ${C.border}`, borderRadius: '4px', px: 1, py: 0.5 }}>
        {mode === 'session'
          ? 'system_valid_from ≤ @Session.AsOfDate AND system_valid_to > @Session.AsOfDate'
          : overrideDate
          ? `system_valid_from ≤ '${overrideDate}' AND system_valid_to > '${overrideDate}'`
          : '— pick a date —'}
      </Box>
      <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
        <InfoOutlinedIcon sx={{ fontSize: 11, color: C.textMuted }} />
        <Typography sx={{ fontSize: '0.6rem', color: C.textMuted }}>
          Injected automatically from <span style={{ color: C.yellow }}>X-Uisce-As-Of-Date</span> header.
          Report-level override takes precedence over the session value.
        </Typography>
      </Box>
    </Box>
  );
};

export default ExpressionFilterEditor;
