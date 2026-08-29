import React, { useState } from 'react';
import {
  Box, Typography, TextField, InputAdornment, Popover,
  ListItem, ListItemButton, Chip,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';

export interface FnMeta {
  label: string;
  signature: string;
  category: string;
  description: string;
  returnType?: string;
}

export const FILTER_FUNCTIONS: FnMeta[] = [
  // SQL Aggregates
  { label: 'SUM',        signature: 'SUM(field)',                             category: 'Aggregate', description: 'Sum total of numeric values',        returnType: 'number' },
  { label: 'AVG',        signature: 'AVG(field)',                             category: 'Aggregate', description: 'Arithmetic mean average',            returnType: 'number' },
  { label: 'COUNT',      signature: 'COUNT(field)',                           category: 'Aggregate', description: 'Count of non-null entries',          returnType: 'number' },
  { label: 'COUNT_DISTINCT', signature: 'COUNT(DISTINCT field)',              category: 'Aggregate', description: 'Count of unique non-null values',     returnType: 'number' },
  { label: 'MIN',        signature: 'MIN(field)',                             category: 'Aggregate', description: 'Minimum value in group',             returnType: 'number' },
  { label: 'MAX',        signature: 'MAX(field)',                             category: 'Aggregate', description: 'Maximum value in group',             returnType: 'number' },
  { label: 'STDDEV',     signature: 'STDDEV(field)',                          category: 'Aggregate', description: 'Statistical standard deviation',     returnType: 'number' },
  { label: 'VARIANCE',   signature: 'VARIANCE(field)',                        category: 'Aggregate', description: 'Statistical variance',               returnType: 'number' },
  { label: 'PERCENTILE_CONT', signature: 'PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY field)', category: 'Aggregate', description: 'Continuous percentile / Median', returnType: 'number' },

  // Window Functions
  { label: 'ROW_NUMBER', signature: 'ROW_NUMBER() OVER (ORDER BY field)',    category: 'Window',    description: 'Unique sequential row number in partition', returnType: 'number' },
  { label: 'RANK',       signature: 'RANK() OVER (ORDER BY field DESC)',      category: 'Window',    description: 'Rank with gaps for identical values', returnType: 'number' },
  { label: 'DENSE_RANK', signature: 'DENSE_RANK() OVER (ORDER BY field DESC)',category: 'Window',    description: 'Rank without gaps',                  returnType: 'number' },
  { label: 'RUNNING_TOTAL', signature: 'SUM(field) OVER (ORDER BY field ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW)', category: 'Window', description: 'Cumulative rolling sum', returnType: 'number' },
  { label: 'LEAD',       signature: 'LEAD(field, 1) OVER (ORDER BY field)',   category: 'Window',    description: 'Value from subsequent row',          returnType: 'any'    },
  { label: 'LAG',        signature: 'LAG(field, 1) OVER (ORDER BY field)',    category: 'Window',    description: 'Value from preceding row',           returnType: 'any'    },
  { label: 'PERCENT_OF_TOTAL', signature: '100.0 * field / NULLIF(SUM(field) OVER (), 0)', category: 'Window', description: 'Percentage contribution to partition sum', returnType: 'number' },

  // PostgreSQL JSON & JSONB Functions & Operators
  { label: 'JSON_KEY_TEXT',    signature: "field->>'key'",                    category: 'JSON / JSONB', description: 'Postgres: Extract JSON field as text (->>)', returnType: 'string'  },
  { label: 'JSON_PATH_TEXT',   signature: "field#>>'{path,key}'",             category: 'JSON / JSONB', description: 'Postgres: Extract nested JSON path as text (#>>)', returnType: 'string' },
  { label: 'JSON_EXTRACT_PATH', signature: "jsonb_extract_path_text(field, 'key')", category: 'JSON / JSONB', description: 'Postgres: Extract path as text', returnType: 'string' },
  { label: 'JSON_HAS_KEY',     signature: "field ? 'key'",                    category: 'JSON / JSONB', description: 'Postgres: Check if top-level key exists (?)', returnType: 'boolean' },
  { label: 'JSONB_CONTAINS',   signature: "field @> '{\"key\": \"val\"}'::jsonb", category: 'JSON / JSONB', description: 'Postgres: JSONB containment match (@>)', returnType: 'boolean' },
  { label: 'JSON_ARRAY_LENGTH', signature: 'jsonb_array_length(field)',       category: 'JSON / JSONB', description: 'Postgres: Number of elements in JSON array', returnType: 'number' },
  { label: 'JSON_TYPEOF',      signature: 'jsonb_typeof(field)',              category: 'JSON / JSONB', description: 'Postgres: JSON type (string, number, object, array)', returnType: 'string' },

  // Date / Time — extract / truncate
  { label: 'YEAR',       signature: 'EXTRACT(YEAR FROM field)',               category: 'Date',    description: 'Extract year as integer',           returnType: 'number' },
  { label: 'MONTH',      signature: 'EXTRACT(MONTH FROM field)',              category: 'Date',    description: 'Extract month (1–12)',              returnType: 'number' },
  { label: 'DAY',        signature: 'EXTRACT(DAY FROM field)',                category: 'Date',    description: 'Extract day of month (1–31)',        returnType: 'number' },
  { label: 'DATE_TRUNC', signature: "DATE_TRUNC('month', field)",            category: 'Date',    description: 'Truncate to start of period',        returnType: 'date'   },
  { label: 'EXTRACT',    signature: "EXTRACT('year' FROM field)",            category: 'Date',    description: 'Extract date part (SQL standard)',   returnType: 'number' },
  { label: 'DATE_ADD',   signature: "field + INTERVAL '30 days'",             category: 'Date',    description: 'Add interval days to date',          returnType: 'date'   },
  { label: 'DATESINPERIOD', signature: "DATESINPERIOD(field, CURRENT_DATE, -30, 'DAY')", category: 'Date', description: 'Dates in a rolling period', returnType: 'date'   },

  // Date Macro
  { label: 'CURRENT_DATE',       signature: 'CURRENT_DATE',                           category: 'Date Macro', description: 'Current date (UTC)',            returnType: 'date'   },
  { label: 'CURRENT_TIMESTAMP',  signature: 'CURRENT_TIMESTAMP',                      category: 'Date Macro', description: 'Current timestamp (UTC)',       returnType: 'date'   },
  { label: 'TODAY',              signature: 'CURRENT_DATE',                           category: 'Date Macro', description: 'Current date (UTC)',            returnType: 'date'   },
  { label: 'YESTERDAY',          signature: "CURRENT_DATE - INTERVAL '1 day'",        category: 'Date Macro', description: 'Yesterday (UTC)',                returnType: 'date'   },
  { label: 'MTD',                signature: "DATE_TRUNC('month', CURRENT_DATE)",      category: 'Date Macro', description: 'Month-to-date start',          returnType: 'date'   },
  { label: 'QTD',                signature: "DATE_TRUNC('quarter', CURRENT_DATE)",    category: 'Date Macro', description: 'Quarter-to-date start',         returnType: 'date'   },
  { label: 'YTD',                signature: "DATE_TRUNC('year', CURRENT_DATE)",       category: 'Date Macro', description: 'Year-to-date start',           returnType: 'date'   },
  { label: 'LAST_N_DAYS',        signature: "field >= CURRENT_DATE - INTERVAL '30 days'", category: 'Date Macro', description: 'Last N days including today', returnType: 'boolean' },

  // String
  { label: 'UPPER',      signature: 'UPPER(field)',                           category: 'String',  description: 'Convert to upper case',           returnType: 'string' },
  { label: 'LOWER',      signature: 'LOWER(field)',                           category: 'String',  description: 'Convert to lower case',           returnType: 'string' },
  { label: 'TRIM',       signature: 'TRIM(field)',                            category: 'String',  description: 'Remove leading / trailing spaces', returnType: 'string' },
  { label: 'LENGTH',    signature: 'LENGTH(field)',                          category: 'String',  description: 'Character count',                 returnType: 'number' },
  { label: 'SUBSTR',     signature: 'SUBSTR(field, 1, 5)',                    category: 'String',  description: 'Extract substring: SUBSTR(field, start, length)', returnType: 'string' },
  { label: 'REPLACE',    signature: "REPLACE(field, 'old', 'new')",          category: 'String',  description: 'Replace occurrences',             returnType: 'string' },
  { label: 'CONCAT',     signature: "CONCAT(field, ' - suffix')",             category: 'String',  description: 'Concatenate strings',            returnType: 'string' },
  { label: 'REGEXP_LIKE',signature: "field ~ '^[A-Z0-9]'",                    category: 'String',  description: 'POSIX regex match (~)',            returnType: 'boolean'},

  // Null / conditional
  { label: 'COALESCE',   signature: "COALESCE(field, 'N/A')",                 category: 'Null',    description: 'First non-null argument',          returnType: 'any'    },
  { label: 'NULLIF',     signature: 'NULLIF(field, 0)',                       category: 'Null',    description: 'Return NULL if equal',             returnType: 'any'    },

  // Numeric
  { label: 'ROUND',      signature: 'ROUND(field, 2)',                        category: 'Numeric', description: 'Round to 2 decimal places',        returnType: 'number' },
  { label: 'ABS',        signature: 'ABS(field)',                             category: 'Numeric', description: 'Absolute value',                 returnType: 'number' },
  { label: 'FLOOR',      signature: 'FLOOR(field)',                           category: 'Numeric', description: 'Round down to integer',          returnType: 'number' },
  { label: 'CEIL',       signature: 'CEIL(field)',                            category: 'Numeric', description: 'Round up to integer',            returnType: 'number' },
  { label: 'MOD',        signature: 'MOD(field, 2)',                          category: 'Numeric', description: 'Modulo remainder',               returnType: 'number' },

  // Cast
  { label: 'CAST_TEXT',   signature: 'CAST(field AS TEXT)',                    category: 'Cast',    description: 'Convert to Postgres text',        returnType: 'string' },
  { label: 'CAST_NUMERIC',signature: 'CAST(field AS NUMERIC(18,2))',           category: 'Cast',    description: 'Convert to numeric decimal',      returnType: 'number' },
  { label: 'TO_DATE',    signature: "TO_DATE(field, 'YYYY-MM-DD')",           category: 'Cast',    description: 'Parse string as date',            returnType: 'date'   },
  { label: 'TO_TIMESTAMP', signature: "TO_TIMESTAMP(field, 'YYYY-MM-DD HH24:MI:SS')", category: 'Cast', description: 'Parse string as timestamp', returnType: 'date' },
];

const CATEGORY_COLORS: Record<string, string> = {
  'Aggregate':   '#10B981',
  'Window':      '#8B5CF6',
  'JSON / JSONB':'#06B6D4',
  'Date':        '#F59E0B',
  'Date Macro':  '#FB923C',
  'String':      '#3B82F6',
  'Null':        '#64748B',
  'Numeric':     '#EC4899',
  'Cast':        '#A855F7',
};

const CATEGORY_ORDER = ['Aggregate', 'Window', 'JSON / JSONB', 'Date', 'Date Macro', 'String', 'Null', 'Numeric', 'Cast'];

function categorize(fns: FnMeta[]): Record<string, FnMeta[]> {
  const out: Record<string, FnMeta[]> = {};
  fns.forEach(f => { if (!out[f.category]) out[f.category] = []; out[f.category].push(f); });
  return out;
}

interface Props {
  anchorEl: HTMLElement | null;
  onClose: () => void;
  fieldName: string;
  onSelect: (signature: string, returnType?: string) => void;
}

const FunctionPickerMenu: React.FC<Props> = ({ anchorEl, onClose, fieldName, onSelect }) => {
  const [search, setSearch] = useState('');

  const filtered = search.trim()
    ? FILTER_FUNCTIONS.filter(f =>
        f.label.toLowerCase().includes(search.toLowerCase()) ||
        f.signature.toLowerCase().includes(search.toLowerCase()) ||
        f.description.toLowerCase().includes(search.toLowerCase())
      )
    : FILTER_FUNCTIONS;

  const grouped = categorize(filtered);
  const open = Boolean(anchorEl);

  return (
    <Popover
      open={open}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      transformOrigin={{ vertical: 'top', horizontal: 'left' }}
      slotProps={{
        paper: {
          sx: {
            bgcolor: '#071526',
            border: '1px solid #1E293B',
            borderRadius: 2,
            width: 360,
            maxHeight: 460,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          },
        },
      }}
    >
      <Box sx={{ p: 1.5, borderBottom: '1px solid #1E293B', flexShrink: 0 }}>
        <TextField
          size="small"
          autoFocus
          placeholder="Search SQL functions, aggregates, JSON, windows…"
          value={search}
          onChange={e => setSearch(e.target.value)}
          fullWidth
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ fontSize: 16, color: '#64748B' }} />
              </InputAdornment>
            ),
          }}
          sx={{
            '& .MuiOutlinedInput-root': {
              fontSize: '0.75rem',
              bgcolor: '#0B1E36',
              color: '#E2E8F0',
              '& fieldset': { borderColor: '#1E293B' },
              '&:hover fieldset': { borderColor: '#334155' },
              '&.Mui-focused fieldset': { borderColor: '#00D4FF' },
            },
          }}
        />
      </Box>

      <Box sx={{ flex: 1, overflowY: 'auto' }}>
        <ListItem disablePadding>
          <ListItemButton
            dense
            onClick={() => { onSelect('', undefined); onClose(); }}
            sx={{
              py: 0.8, px: 2,
              borderBottom: '1px solid #1E293B',
              '&:hover': { bgcolor: '#0B1E36' },
            }}
          >
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.2, width: '100%' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Typography sx={{ fontSize: '0.72rem', fontWeight: 700, color: '#94A3B8', fontFamily: 'monospace' }}>
                  None (Raw column)
                </Typography>
                <Chip
                  label="Clear"
                  size="small"
                  sx={{
                    height: 14, fontSize: '0.55rem', fontWeight: 700,
                    bgcolor: 'rgba(148, 163, 184, 0.1)',
                    color: '#94A3B8',
                    border: '1px solid rgba(148, 163, 184, 0.3)',
                  }}
                />
              </Box>
              <Typography sx={{ fontSize: '0.65rem', color: '#64748B', fontFamily: 'monospace' }}>
                {fieldName}
              </Typography>
            </Box>
          </ListItemButton>
        </ListItem>

        {CATEGORY_ORDER.map(cat => {
          const fns = grouped[cat];
          if (!fns?.length) return null;
          return (
            <Box key={cat}>
              <Box sx={{
                px: 2, py: 0.5,
                bgcolor: '#0B1E36',
                borderTop: '1px solid #1E293B',
                borderBottom: '1px solid #1E293B40',
                display: 'flex', alignItems: 'center', gap: 0.75,
              }}>
                <Box sx={{
                  width: 8, height: 8, borderRadius: '50%',
                  bgcolor: CATEGORY_COLORS[cat] || '#64748B',
                  flexShrink: 0,
                }} />
                <Typography sx={{ fontSize: '0.6rem', fontWeight: 700, color: CATEGORY_COLORS[cat] || '#64748B', textTransform: 'uppercase', letterSpacing: 1 }}>
                  {cat}
                </Typography>
              </Box>
              {fns.map(fn => {
                const label = fn.label;
                const sig = fn.signature.replace(/\bfield\b/g, fieldName || 'field');
                return (
                  <ListItem disablePadding key={label}>
                    <ListItemButton
                      dense
                      onClick={() => { onSelect(sig, fn.returnType); onClose(); }}
                      sx={{
                        py: 0.6, px: 2,
                        '&:hover': { bgcolor: '#0B1E36' },
                      }}
                    >
                      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.2, width: '100%' }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                          <Typography sx={{ fontSize: '0.72rem', fontWeight: 700, color: '#E2E8F0', fontFamily: 'monospace' }}>
                            {label}
                          </Typography>
                          {fn.returnType && (
                            <Chip
                              label={fn.returnType}
                              size="small"
                              sx={{
                                height: 14, fontSize: '0.55rem', fontWeight: 700,
                                bgcolor: CATEGORY_COLORS[cat] ? `${CATEGORY_COLORS[cat]}20` : '#64748B20',
                                color: CATEGORY_COLORS[cat] || '#64748B',
                                border: `1px solid ${CATEGORY_COLORS[cat] ? `${CATEGORY_COLORS[cat]}40` : '#64748B40'}`,
                              }}
                            />
                          )}
                        </Box>
                        <Typography sx={{ fontSize: '0.68rem', color: '#00D4FF', fontFamily: 'monospace' }}>
                          {sig}
                        </Typography>
                        <Typography sx={{ fontSize: '0.62rem', color: '#64748B' }}>
                          {fn.description}
                        </Typography>
                      </Box>
                    </ListItemButton>
                  </ListItem>
                );
              })}
            </Box>
          );
        })}
        {filtered.length === 0 && (
          <Box sx={{ py: 4, textAlign: 'center' }}>
            <Typography sx={{ fontSize: '0.72rem', color: '#64748B' }}>No functions match "{search}"</Typography>
          </Box>
        )}
      </Box>
    </Popover>
  );
};

export default FunctionPickerMenu;
