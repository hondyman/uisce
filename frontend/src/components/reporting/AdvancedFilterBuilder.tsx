import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  IconButton,
  Button,
  Chip,
  Select,
  MenuItem,
  TextField,
  Tooltip,
  Divider,
  useTheme,
} from '@mui/material';
import FunctionsIcon from '@mui/icons-material/Functions';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import CodeIcon from '@mui/icons-material/Code';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import AccountTreeIcon from '@mui/icons-material/AccountTree';

export interface FilterConditionModel {
  id: string;
  fieldKey: string;
  functionWrap?: string; // e.g. "SUBSTR", "UPPER", "TRIM", "NONE"
  functionArgs?: string; // e.g. "1, 3"
  operator: string;
  valueMode: 'LITERAL' | 'PARAMETER' | 'MACRO';
  value: string;
  paramKey?: string;
  macroName?: string;
  isEnabled: boolean;
}

export interface FilterGroupModel {
  id: string;
  combinator: 'AND' | 'OR';
  conditions: FilterConditionModel[];
}

export const AdvancedFilterBuilder: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  
  const [groups, setGroups] = useState<FilterGroupModel[]>([
    {
      id: 'grp_1',
      combinator: 'AND',
      conditions: [
        {
          id: 'cond_1',
          fieldKey: 'account_bk',
          functionWrap: 'SUBSTR',
          functionArgs: '1, 3',
          operator: 'EQUALS',
          valueMode: 'PARAMETER',
          value: '',
          paramKey: '@TargetPrefix',
          isEnabled: true,
        },
        {
          id: 'cond_2',
          fieldKey: 'trade_date',
          functionWrap: 'NONE',
          operator: 'GREATER_THAN_OR_EQUALS',
          valueMode: 'MACRO',
          value: '',
          macroName: 'PREVIOUS_BUSINESS_DAY (T-1)',
          isEnabled: true,
        },
      ],
    },
    {
      id: 'grp_2',
      combinator: 'OR',
      conditions: [
        {
          id: 'cond_3',
          fieldKey: 'total_nav',
          functionWrap: 'NONE',
          operator: 'GREATER_THAN',
          valueMode: 'LITERAL',
          value: '1000000',
          isEnabled: true,
        },
        {
          id: 'cond_4',
          fieldKey: 'account_subtype_cd',
          functionWrap: 'UPPER',
          operator: 'EQUALS',
          valueMode: 'LITERAL',
          value: 'ALT_INVESTMENT',
          isEnabled: true,
        },
      ],
    },
  ]);

  const [showSqlPreview, setShowSqlPreview] = useState(false);

  const toggleGroupCombinator = (groupId: string) => {
    setGroups((prev) =>
      prev.map((g) => (g.id === groupId ? { ...g, combinator: g.combinator === 'AND' ? 'OR' : 'AND' } : g))
    );
  };

  const toggleConditionState = (groupId: string, condId: string) => {
    setGroups((prev) =>
      prev.map((g) =>
        g.id === groupId
          ? {
              ...g,
              conditions: g.conditions.map((c) => (c.id === condId ? { ...c, isEnabled: !c.isEnabled } : c)),
            }
          : g
      )
    );
  };

  const bgColor = isDark ? '#050D1A' : '#f8fafc';
  const borderColor = isDark ? '#1E293B' : '#e2e8f0';
  const panelBg = isDark ? '#071526' : '#ffffff';
  const chipBg = isDark ? '#0B1E36' : '#f1f5f9';
  const accentColor = '#00D4FF';
  const mutedColor = isDark ? '#64748B' : '#64748B';
  const textColor = isDark ? '#fff' : '#111827';
  const subTextColor = isDark ? '#94A3B8' : '#4b5563';

  return (
    <Box sx={{ width: '100%', bgcolor: bgColor, color: textColor, borderRadius: 2, border: `1px solid ${borderColor}`, p: 3, fontFamily: 'sans-serif' }}>
      
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 2, borderBottom: `1px solid ${borderColor}`, mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <AccountTreeIcon sx={{ color: accentColor, fontSize: 24 }} />
          <Box>
            <Typography variant="subtitle1" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
              Enterprise Filter & Dynamic AST Expression Builder
            </Typography>
            <Typography variant="caption" sx={{ color: mutedColor }}>
              Scalar transformations (`SUBSTR`, `UPPER`), Relative Date Macros (`T-1`, `MTD`), and Parameter Bindings.
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', gap: 1.5 }}>
          <Button
            size="small"
            variant="outlined"
            startIcon={<CodeIcon />}
            onClick={() => setShowSqlPreview(!showSqlPreview)}
            sx={{ color: accentColor, borderColor: 'rgba(0, 212, 255, 0.3)', textTransform: 'none', fontSize: '11px' }}
          >
            {showSqlPreview ? 'Hide SQL' : 'View SQL WHERE'}
          </Button>
          <Button
            size="small"
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() =>
              setGroups((prev) => [
                ...prev,
                { id: `grp_${Date.now()}`, combinator: 'AND', conditions: [] },
              ])
            }
            sx={{ bgcolor: accentColor, color: isDark ? '#050D1A' : '#000', fontWeight: 700, fontSize: '11px', textTransform: 'none' }}
          >
            Add Filter Group
          </Button>
        </Box>
      </Box>

      {/* SQL Preview Banner */}
      {showSqlPreview && (
        <Paper sx={{ p: 2, mb: 3, bgcolor: panelBg, border: '1px solid rgba(0, 212, 255, 0.2)', borderRadius: 1.5, fontFamily: 'monospace', fontSize: '11px', color: '#A5F3FC' }}>
          <Typography variant="caption" sx={{ color: mutedColor, display: 'block', mb: 0.5 }}>Compiled AST Pushdown SQL:</Typography>
          WHERE (SUBSTR(account_bk, 1, 3) = $1 AND trade_date &gt;= $2) AND (total_nav &gt; $3 OR UPPER(account_subtype_cd) = $4)
        </Paper>
      )}

      {/* Filter Groups */}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
        {groups.map((group, gIdx) => (
          <React.Fragment key={group.id}>
            {gIdx > 0 && (
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Chip size="small" label="AND" sx={{ bgcolor: chipBg, color: accentColor, fontWeight: 700, fontSize: '10px', height: 20, border: `1px solid ${borderColor}` }} />
              </Box>
            )}

            <Paper sx={{ p: 2, bgcolor: panelBg, border: `1px solid ${borderColor}`, borderRadius: 2 }}>
              
              {/* Group Header */}
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                  <Typography variant="caption" fontWeight="700" sx={{ color: subTextColor }}>GROUP {gIdx + 1}</Typography>
                  <Button
                    size="small"
                    onClick={() => toggleGroupCombinator(group.id)}
                    sx={{ bgcolor: group.combinator === 'AND' ? '#0284C7' : '#D97706', color: '#fff', fontWeight: 700, fontSize: '10px', height: 20, minWidth: 44, px: 1 }}
                  >
                    {group.combinator}
                  </Button>
                </Box>

                <Button
                  size="small"
                  startIcon={<AddIcon />}
                  onClick={() => {
                    const next = [...groups];
                    next[gIdx].conditions.push({
                      id: `cond_${Date.now()}`,
                      fieldKey: 'nav_end',
                      operator: 'GREATER_THAN',
                      valueMode: 'LITERAL',
                      value: '0',
                      isEnabled: true,
                    });
                    setGroups(next);
                  }}
                  sx={{ color: accentColor, fontSize: '11px', textTransform: 'none' }}
                >
                  Add Condition
                </Button>
              </Box>

              {/* Conditions List */}
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                {group.conditions.map((cond) => (
                  <Box
                    key={cond.id}
                    sx={{
                      display: 'grid',
                      gridTemplateColumns: '180px 140px 180px 1fr 40px',
                      alignItems: 'center',
                      gap: 1.5,
                      p: 1.5,
                      bgcolor: bgColor,
                      borderRadius: 1.5,
                      border: '1px solid',
                      borderColor: cond.isEnabled ? borderColor : 'rgba(239, 68, 68, 0.2)',
                      opacity: cond.isEnabled ? 1 : 0.6,
                    }}
                  >
                    {/* Left: Function Wrap & Field */}
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '11px', color: isDark ? '#E2E8F0' : '#374151' }}>
                        {cond.functionWrap && cond.functionWrap !== 'NONE'
                          ? `${cond.functionWrap}(${cond.fieldKey}${cond.functionArgs ? `, ${cond.functionArgs}` : ''})`
                          : cond.fieldKey}
                      </Typography>
                      {cond.functionWrap && cond.functionWrap !== 'NONE' && (
                        <Typography variant="caption" sx={{ fontSize: '9px', color: '#22d3ee', fontFamily: 'monospace' }}>Scalar Function Wrapped</Typography>
                      )}
                    </Box>

                    {/* Operator */}
                    <Typography variant="caption" sx={{ color: accentColor, fontWeight: 600, fontFamily: 'monospace' }}>
                      {cond.operator}
                    </Typography>

                    {/* Value Mode Selector */}
                    <Box sx={{ display: 'flex', gap: 0.5 }}>
                      <Chip
                        size="small"
                        label={cond.valueMode}
                        sx={{
                          bgcolor: cond.valueMode === 'PARAMETER' ? 'rgba(168, 85, 247, 0.2)' : cond.valueMode === 'MACRO' ? 'rgba(245, 158, 11, 0.2)' : chipBg,
                          color: cond.valueMode === 'PARAMETER' ? '#C084FC' : cond.valueMode === 'MACRO' ? '#FBBF24' : subTextColor,
                          fontSize: '9px',
                          fontWeight: 700,
                          height: 18,
                        }}
                      />
                    </Box>

                    {/* Right: Value Expression */}
                    <Box>
                      {cond.valueMode === 'PARAMETER' ? (
                        <Typography variant="body2" sx={{ color: '#c084fc', fontFamily: 'monospace', fontSize: '12px', fontWeight: 700 }}>{cond.paramKey}</Typography>
                      ) : cond.valueMode === 'MACRO' ? (
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          <CalendarMonthIcon sx={{ fontSize: 13, color: '#fbbf24' }} />
                          <Typography variant="body2" sx={{ color: '#fbbf24', fontFamily: 'monospace', fontSize: '12px', fontWeight: 700 }}>{cond.macroName}</Typography>
                        </Box>
                      ) : (
                        <Typography variant="body2" sx={{ color: isDark ? '#e2e8f0' : '#1f2937', fontFamily: 'monospace', fontSize: '12px' }}>{cond.value}</Typography>
                      )}
                    </Box>

                    {/* Actions */}
                    <IconButton size="small" onClick={() => toggleConditionState(group.id, cond.id)} sx={{ color: cond.isEnabled ? '#10B981' : mutedColor }}>
                      {cond.isEnabled ? <VisibilityIcon fontSize="small" /> : <VisibilityOffIcon fontSize="small" />}
                    </IconButton>
                  </Box>
                ))}
              </Box>

            </Paper>
          </React.Fragment>
        ))}
      </Box>

    </Box>
  );
};

export default AdvancedFilterBuilder;
