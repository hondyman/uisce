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
  // Date / Time — extract / truncate
  { label: 'YEAR',       signature: 'YEAR(field)',                          category: 'Date',    description: 'Extract year as integer',           returnType: 'number' },
  { label: 'MONTH',      signature: 'MONTH(field)',                         category: 'Date',    description: 'Extract month (1–12)',              returnType: 'number' },
  { label: 'DAY',        signature: 'DAY(field)',                            category: 'Date',    description: 'Extract day of month (1–31)',        returnType: 'number' },
  { label: 'DATE_TRUNC', signature: "DATE_TRUNC('month', field)",            category: 'Date',    description: 'Truncate to start of period',        returnType: 'date'   },
  { label: 'EXTRACT',    signature: "EXTRACT('year' FROM field)",            category: 'Date',    description: 'Extract date part (SQL standard)',   returnType: 'number' },
  { label: 'DATE_ADD',   signature: "DATE_ADD(field, n, 'day')",            category: 'Date',    description: 'Add N days to date',                 returnType: 'date'   },
  { label: 'DATESINPERIOD', signature: "DATESINPERIOD('date_col', START, -30, DAY)", category: 'Date', description: 'Dates in a rolling period',       returnType: 'date'   },
  // Date Macro
  { label: 'TODAY',              signature: 'TODAY()',                                category: 'Date Macro', description: 'Current date (UTC)',            returnType: 'date'   },
  { label: 'YESTERDAY',          signature: 'YESTERDAY()',                            category: 'Date Macro', description: 'Yesterday (UTC)',                returnType: 'date'   },
  { label: 'MTD',                signature: 'MTD()',                                  category: 'Date Macro', description: 'Month-to-date start',          returnType: 'date'   },
  { label: 'QTD',                signature: 'QTD()',                                  category: 'Date Macro', description: 'Quarter-to-date start',         returnType: 'date'   },
  { label: 'YTD',                signature: 'YTD()',                                  category: 'Date Macro', description: 'Year-to-date start',           returnType: 'date'   },
  { label: 'LAST_N_DAYS',        signature: 'LAST_N_DAYS(30)',                        category: 'Date Macro', description: 'Last N days including today',   returnType: 'date'   },
  { label: 'T_MINUS',            signature: 'T_MINUS(2)',                            category: 'Date Macro', description: 'T-N settlement days ago',      returnType: 'date'   },
  { label: 'BUSINESS_DAYS_AGO',  signature: 'BUSINESS_DAYS_AGO(5)',                  category: 'Date Macro', description: 'N business days ago (BD+)',    returnType: 'date'   },
  { label: 'PREVIOUS_BUSINESS_DAY', signature: 'PREVIOUS_BUSINESS_DAY(field)',        category: 'Date Macro', description: 'Prior business day',             returnType: 'date'   },
  // String
  { label: 'UPPER',      signature: 'UPPER(field)',                           category: 'String',  description: 'Convert to upper case',           returnType: 'string' },
  { label: 'LOWER',      signature: 'LOWER(field)',                           category: 'String',  description: 'Convert to lower case',           returnType: 'string' },
  { label: 'TRIM',       signature: 'TRIM(field)',                            category: 'String',  description: 'Remove leading / trailing spaces', returnType: 'string' },
  { label: 'LENGTH',    signature: 'LENGTH(field)',                          category: 'String',  description: 'Character count',                 returnType: 'number' },
  { label: 'SUBSTR',     signature: 'SUBSTR(field, start, length)',           category: 'String',  description: 'Extract substring',               returnType: 'string' },
  { label: 'REPLACE',    signature: 'REPLACE(field, old, new)',              category: 'String',  description: 'Replace occurrences',             returnType: 'string' },
  { label: 'CONCAT',     signature: 'CONCAT(field1, field2)',                category: 'String',  description: 'Concatenate strings',            returnType: 'string' },
  { label: 'REGEXP_LIKE',signature: 'REGEXP_LIKE(field, pattern)',            category: 'String',  description: 'POSIX regex match',               returnType: 'boolean'},
  // Null / conditional
  { label: 'COALESCE',   signature: 'COALESCE(field, default)',              category: 'Null',    description: 'First non-null argument',          returnType: 'any'    },
  { label: 'NULLIF',     signature: 'NULLIF(field, value)',                   category: 'Null',    description: 'Return NULL if equal',             returnType: 'any'    },
  // Numeric
  { label: 'ROUND',      signature: 'ROUND(field, decimals)',                 category: 'Numeric', description: 'Round to N decimal places',      returnType: 'number' },
  { label: 'ABS',        signature: 'ABS(field)',                             category: 'Numeric', description: 'Absolute value',                 returnType: 'number' },
  { label: 'FLOOR',      signature: 'FLOOR(field)',                           category: 'Numeric', description: 'Round down to integer',          returnType: 'number' },
  { label: 'CEIL',       signature: 'CEIL(field)',                            category: 'Numeric', description: 'Round up to integer',            returnType: 'number' },
  { label: 'MOD',        signature: 'MOD(field, divisor)',                     category: 'Numeric', description: 'Modulo remainder',               returnType: 'number' },
  // Cast
  { label: 'CAST',       signature: 'CAST(field AS TEXT)',                    category: 'Cast',    description: 'Convert to another type',          returnType: 'any'    },
  { label: 'TO_DATE',    signature: "TO_DATE(field, 'YYYY-MM-DD')",           category: 'Cast',    description: 'Parse string as date',            returnType: 'date'   },
  { label: 'TO_NUMBER',  signature: 'TO_NUMBER(field)',                       category: 'Cast',    description: 'Convert string to number',        returnType: 'number' },
  // Array / JSON
  { label: 'ARRAY_CONTAINS',     signature: 'ARRAY_CONTAINS(field, value)',            category: 'Array/JSON', description: 'Array membership test',   returnType: 'boolean' },
  { label: 'JSON_EXTRACT_SCALAR',signature: "JSON_EXTRACT_SCALAR(field, '$.key')",    category: 'Array/JSON', description: 'Extract JSON scalar',   returnType: 'string'  },
];

const CATEGORY_COLORS: Record<string, string> = {
  'Date':        '#F59E0B',
  'Date Macro':  '#FB923C',
  'String':      '#10B981',
  'Null':        '#64748B',
  'Numeric':     '#3B82F6',
  'Cast':        '#8B5CF6',
  'Array/JSON':  '#06B6D4',
};

const CATEGORY_ORDER = ['Date', 'Date Macro', 'String', 'Null', 'Numeric', 'Cast', 'Array/JSON'];

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
            width: 340,
            maxHeight: 420,
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
          placeholder="Search functions…"
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
                const sig = fn.signature.replace('field', fieldName);
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
                                border: `1px solid ${CATEGORY_COLORS[cat] || '#64748B'}40`,
                              }}
                            />
                          )}
                        </Box>
                        <Typography sx={{ fontSize: '0.65rem', color: '#00D4FF', fontFamily: 'monospace' }}>
                          {sig}
                        </Typography>
                        <Typography sx={{ fontSize: '0.6rem', color: '#64748B' }}>
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
