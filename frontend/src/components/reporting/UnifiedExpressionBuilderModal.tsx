import React, { useState, useMemo, useRef, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Box,
  Typography,
  Grid,
  TextField,
  Button,
  Chip,
  Paper,
  Tooltip,
  IconButton,
  Divider,
  InputAdornment,
} from '@mui/material';
import {
  Code2,
  Sparkles,
  Check,
  Search,
  Layers,
  Calendar,
  Database,
  Sliders,
  Globe,
  FunctionSquare,
  Bookmark,
  Play,
  RotateCcw,
  CheckCircle2,
  AlertCircle,
  FolderOpen,
} from 'lucide-react';
import {
  SYSTEM_VARIABLES,
  PATH_EXPRESSION_PRESETS,
  getDefaultEvaluationContext,
  evaluatePathExpression,
} from './pathExpressionEvaluator';

export interface ExpressionItemDef {
  code: string;
  category: 'system' | 'burst' | 'datetime' | 'fields' | 'parameters' | 'globals' | 'functions' | 'presets';
  displayName: string;
  description: string;
  syntax: string;
  example: string;
}

export const UNIFIED_EXPRESSION_ITEMS: ExpressionItemDef[] = [
  // ── 1. System / Multi-tenant variables ──────────────────────────────
  {
    code: '@tenant_code',
    category: 'system',
    displayName: '@tenant_code',
    description: 'Dynamic tenant slug for routing (e.g. acme_wealth, blackrock_wm)',
    syntax: '@tenant_code',
    example: 'acme_wealth',
  },
  {
    code: '@is_core',
    category: 'system',
    displayName: '@is_core',
    description: 'True if executing under master core tenant, false if client tenant',
    syntax: '@is_core',
    example: 'false',
  },
  {
    code: '@tenant_name',
    category: 'system',
    displayName: '@tenant_name',
    description: 'Full legal name of the active tenant organization',
    syntax: '@tenant_name',
    example: 'Acme Wealth Management LLC',
  },
  {
    code: '@tenant_id',
    category: 'system',
    displayName: '@tenant_id',
    description: 'Database UUID of the tenant',
    syntax: '@tenant_id',
    example: '8f3a9e22-1d54-4f9e-a612-88231901df42',
  },
  {
    code: '@report_name',
    category: 'system',
    displayName: '@report_name',
    description: 'Current report display title',
    syntax: '@report_name',
    example: 'Daily Institutional Client Valuation',
  },
  {
    code: '@report_code',
    category: 'system',
    displayName: '@report_code',
    description: 'Sanitized safe snake_case report code',
    syntax: '@report_code',
    example: 'daily_institutional_client_valuation',
  },
  {
    code: '@report_id',
    category: 'system',
    displayName: '@report_id',
    description: 'Report definition key or UUID',
    syntax: '@report_id',
    example: 'rep-custom-001',
  },
  {
    code: '@executed_by',
    category: 'system',
    displayName: '@executed_by',
    description: 'User email or background scheduler identity',
    syntax: '@executed_by',
    example: 'admin@uuisce.internal',
  },

  // ── 2. Slicing & Burst Partitioning ──────────────────────────────────
  {
    code: '@slice_key',
    category: 'burst',
    displayName: '@slice_key',
    description: 'Active burst slicing partition value (e.g. client_id, account_id)',
    syntax: '@slice_key',
    example: 'client-001',
  },
  {
    code: '@slice_name',
    category: 'burst',
    displayName: '@slice_name',
    description: 'Display name of sliced client or account entity',
    syntax: '@slice_name',
    example: 'Apex Global Alpha Fund',
  },
  {
    code: '@seq',
    category: 'burst',
    displayName: '@seq',
    description: 'Zero-padded 3-digit burst sequence index (001, 002, 003...)',
    syntax: '@seq',
    example: '001',
  },
  {
    code: '@seq_raw',
    category: 'burst',
    displayName: '@seq_raw',
    description: 'Raw integer sequence number',
    syntax: '@seq_raw',
    example: '1',
  },
  {
    code: '@total_slices',
    category: 'burst',
    displayName: '@total_slices',
    description: 'Total number of entities sliced in current burst batch',
    syntax: '@total_slices',
    example: '48',
  },
  {
    code: '@batch_id',
    category: 'burst',
    displayName: '@batch_id',
    description: 'Unique execution batch run UUID',
    syntax: '@batch_id',
    example: 'btch-9a4f2e',
  },

  // ── 3. Date & Time ──────────────────────────────────────────────────
  {
    code: '@date',
    category: 'datetime',
    displayName: '@date',
    description: 'Current execution date in ISO format (YYYY-MM-DD)',
    syntax: '@date',
    example: '2026-08-24',
  },
  {
    code: '@year',
    category: 'datetime',
    displayName: '@year',
    description: '4-Digit calendar year',
    syntax: '@year',
    example: '2026',
  },
  {
    code: '@month',
    category: 'datetime',
    displayName: '@month',
    description: '2-Digit zero-padded month (01 to 12)',
    syntax: '@month',
    example: '08',
  },
  {
    code: '@month_name',
    category: 'datetime',
    displayName: '@month_name',
    description: 'Full English month name',
    syntax: '@month_name',
    example: 'August',
  },
  {
    code: '@day',
    category: 'datetime',
    displayName: '@day',
    description: '2-Digit calendar day of month',
    syntax: '@day',
    example: '24',
  },
  {
    code: '@quarter',
    category: 'datetime',
    displayName: '@quarter',
    description: 'Accounting quarter period string',
    syntax: '@quarter',
    example: '2026-Q3',
  },
  {
    code: '@timestamp',
    category: 'datetime',
    displayName: '@timestamp',
    description: 'Compact alphanumeric timestamp',
    syntax: '@timestamp',
    example: '20260824_140000',
  },
  {
    code: '@effective_date',
    category: 'datetime',
    displayName: '@effective_date',
    description: 'Valuation as-of date (with calendar non-trading offset)',
    syntax: '@effective_date',
    example: '2026-08-23',
  },

  // ── 4. Dataset Fields ────────────────────────────────────────────────
  {
    code: 'Fields!nav.Value',
    category: 'fields',
    displayName: 'Fields!nav.Value',
    description: 'Net Asset Value (NAV) of portfolio / account',
    syntax: 'Fields!nav.Value',
    example: '1450000',
  },
  {
    code: 'Fields!amount.Value',
    category: 'fields',
    displayName: 'Fields!amount.Value',
    description: 'Transaction or ledger amount',
    syntax: 'Fields!amount.Value',
    example: '52000.50',
  },
  {
    code: 'Fields!status.Value',
    category: 'fields',
    displayName: 'Fields!status.Value',
    description: 'Entity lifecycle or compliance status (e.g. Active, Pending)',
    syntax: 'Fields!status.Value',
    example: '"Active"',
  },
  {
    code: 'Fields!sector.Value',
    category: 'fields',
    displayName: 'Fields!sector.Value',
    description: 'GICS Sector classification',
    syntax: 'Fields!sector.Value',
    example: '"Technology"',
  },
  {
    code: 'Fields!asset_class.Value',
    category: 'fields',
    displayName: 'Fields!asset_class.Value',
    description: 'Asset class (Equities, Fixed Income, Alternatives)',
    syntax: 'Fields!asset_class.Value',
    example: '"Equities"',
  },
  {
    code: 'Fields!return_ytd.Value',
    category: 'fields',
    displayName: 'Fields!return_ytd.Value',
    description: 'Year-to-date portfolio percentage return',
    syntax: 'Fields!return_ytd.Value',
    example: '12.45',
  },
  {
    code: 'Fields!client_id.Value',
    category: 'fields',
    displayName: 'Fields!client_id.Value',
    description: 'Institutional client account identifier',
    syntax: 'Fields!client_id.Value',
    example: '"client-001"',
  },

  // ── 5. Parameters ────────────────────────────────────────────────────
  {
    code: 'Parameters!StartDate.Value',
    category: 'parameters',
    displayName: 'Parameters!StartDate',
    description: 'Report query range start date',
    syntax: 'Parameters!StartDate.Value',
    example: '"2026-01-01"',
  },
  {
    code: 'Parameters!EndDate.Value',
    category: 'parameters',
    displayName: 'Parameters!EndDate',
    description: 'Report query range end date',
    syntax: 'Parameters!EndDate.Value',
    example: '"2026-08-24"',
  },
  {
    code: 'Parameters!Benchmark.Value',
    category: 'parameters',
    displayName: 'Parameters!Benchmark',
    description: 'Selected market benchmark index',
    syntax: 'Parameters!Benchmark.Value',
    example: '"S&P 500 Total Return"',
  },
  {
    code: 'Parameters!Currency.Value',
    category: 'parameters',
    displayName: 'Parameters!Currency',
    description: 'Base reporting presentation currency',
    syntax: 'Parameters!Currency.Value',
    example: '"USD"',
  },

  // ── 6. Globals ───────────────────────────────────────────────────────
  {
    code: 'Globals!PageNumber',
    category: 'globals',
    displayName: 'Globals!PageNumber',
    description: 'Current page number in rendered document',
    syntax: '{PageNumber}',
    example: '1',
  },
  {
    code: 'Globals!TotalPages',
    category: 'globals',
    displayName: 'Globals!TotalPages',
    description: 'Total page count of rendered document',
    syntax: '{TotalPages}',
    example: '12',
  },
  {
    code: 'Globals!ExecutionTime',
    category: 'globals',
    displayName: 'Globals!ExecutionTime',
    description: 'Exact UTC timestamp of report execution',
    syntax: '{ExecutionTime}',
    example: '"2026-08-24T14:00:00Z"',
  },
  {
    code: 'Globals!ReportName',
    category: 'globals',
    displayName: 'Globals!ReportName',
    description: 'Active report name',
    syntax: '{ReportName}',
    example: '"Daily Institutional Client Valuation"',
  },

  // ── 7. Built-in Functions ────────────────────────────────────────────
  {
    code: 'IIF(condition, truePart, falsePart)',
    category: 'functions',
    displayName: 'IIF(condition, true, false)',
    description: 'Returns truePart if condition is true, otherwise falsePart',
    syntax: '=IIF(Fields!nav.Value > 1000000, "#10B981", "#EF4444")',
    example: '#10B981',
  },
  {
    code: 'Switch(cond1, val1, cond2, val2, ...)',
    category: 'functions',
    displayName: 'Switch(cond1, val1, ...)',
    description: 'Evaluates multiple condition pairs in sequence',
    syntax: '=Switch(Fields!status.Value == "Active", "#10B981", Fields!status.Value == "Pending", "#F59E0B", true, "#94A3B8")',
    example: '#10B981',
  },
  {
    code: 'Concat(str1, str2, ...)',
    category: 'functions',
    displayName: 'Concat(str1, str2, ...)',
    description: 'Concatenates multiple strings together',
    syntax: '=Concat(@report_code, "_", @tenant_code, "_", @slice_key, "_", @seq)',
    example: 'daily_valuation_acme_client-001_001',
  },
  {
    code: 'FormatDate(date, format)',
    category: 'functions',
    displayName: 'FormatDate(date, format)',
    description: 'Formats a date according to standard format mask',
    syntax: '=FormatDate(@date, "YYYY/MM/DD")',
    example: '2026/08/24',
  },
  {
    code: 'PadLeft(value, length, padChar)',
    category: 'functions',
    displayName: 'PadLeft(val, len, char)',
    description: 'Pads a string/number on the left to target length',
    syntax: '=PadLeft(@seq_raw, 4, "0")',
    example: '0001',
  },
  {
    code: 'Upper(string)',
    category: 'functions',
    displayName: 'Upper(string)',
    description: 'Converts string to uppercase',
    syntax: '=Upper(@tenant_code)',
    example: 'ACME_WEALTH',
  },
  {
    code: 'Lower(string)',
    category: 'functions',
    displayName: 'Lower(string)',
    description: 'Converts string to lowercase',
    syntax: '=Lower(@report_name)',
    example: 'daily institutional client valuation',
  },
  {
    code: 'Coalesce(val1, val2, ...)',
    category: 'functions',
    displayName: 'Coalesce(val1, val2, ...)',
    description: 'Returns the first non-null/non-empty value',
    syntax: '=Coalesce(Fields!display_name.Value, @slice_key, "Unknown")',
    example: 'Apex Global Alpha',
  },

  // ── 8. Common Quick Presets ──────────────────────────────────────────
  {
    code: '=IIF(Fields!return_ytd.Value >= 0, "#10B981", "#EF4444")',
    category: 'presets',
    displayName: 'Positive / Negative Color KPI',
    description: 'Green for positive returns, red for negative returns',
    syntax: '=IIF(Fields!return_ytd.Value >= 0, "#10B981", "#EF4444")',
    example: '#10B981',
  },
  {
    code: '=IIF(@is_core, "/core_reports/" + @report_code + "/" + @quarter + "/", "/tenants/" + @tenant_code + "/" + @year + "/" + @month + "/")',
    category: 'presets',
    displayName: 'Core vs Tenant Routing Path',
    description: 'Routes core reports to global master directory or tenant folder',
    syntax: '=IIF(@is_core, "/core_reports/" + @report_code + "/" + @quarter + "/", "/tenants/" + @tenant_code + "/" + @year + "/" + @month + "/")',
    example: '/tenants/acme_wealth/2026/08/',
  },
  {
    code: '=Concat(@report_code, "_", @slice_key, "_", @date, "_", @seq)',
    category: 'presets',
    displayName: 'Sequenced Client Package Filename',
    description: 'Generates uniquely indexed burst audit file name',
    syntax: '=Concat(@report_code, "_", @slice_key, "_", @date, "_", @seq)',
    example: 'daily_valuation_client-001_2026-08-24_001',
  },
];

export interface UnifiedExpressionBuilderModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  label?: string;
  initialFormula: string;
  onApply: (formula: string) => void;
  reportName?: string;
  reportId?: string;
  tenantId?: string;
  isPathMode?: boolean;
}

export const UnifiedExpressionBuilderModal: React.FC<UnifiedExpressionBuilderModalProps> = ({
  open,
  onClose,
  title = 'Unified Dynamic Expression Builder',
  label = 'Formula / Expression',
  initialFormula,
  onApply,
  reportName,
  reportId,
  tenantId,
  isPathMode = false,
}) => {
  const [formula, setFormula] = useState(initialFormula);
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const textFieldRef = useRef<HTMLInputElement>(null);

  // Sync initial formula when dialog opens
  useEffect(() => {
    if (open) {
      setFormula(initialFormula || (isPathMode ? '/tenants/@tenant_code/@year/@month/' : '=IIF(Fields!nav.Value > 1000000, "#10B981", "#EF4444")'));
    }
  }, [open, initialFormula, isPathMode]);

  // Realistic sample evaluation context
  const sampleContext = useMemo(() => {
    return getDefaultEvaluationContext({
      tenant_code: 'acme_wealth',
      tenant_name: 'Acme Wealth Management LLC',
      tenant_id: tenantId || '8f3a9e22-1d54-4f9e-a612-88231901df42',
      is_core: false,
      gold_copy: false,
      report_name: reportName || 'Daily Institutional Client Valuation',
      report_code: (reportName || 'report').toLowerCase().replace(/\s+/g, '_'),
      report_id: reportId || 'rep-custom-001',
      slice_key: 'client-001',
      client_id: 'client-001',
      slice_name: 'Apex Global Alpha Fund',
      seq: '001',
      seq_raw: 1,
      // Sample dataset fields
      nav: 1450000,
      amount: 52000.5,
      status: 'Active',
      sector: 'Technology',
      asset_class: 'Equities',
      return_ytd: 12.45,
    });
  }, [tenantId, reportName, reportId]);

  // Live evaluated result
  const evaluationResult = useMemo(() => {
    return evaluatePathExpression(formula, sampleContext);
  }, [formula, sampleContext]);

  // Filtered items
  const filteredItems = useMemo(() => {
    return UNIFIED_EXPRESSION_ITEMS.filter((item) => {
      const matchesCat = selectedCategory === 'all' || item.category === selectedCategory;
      const matchesSearch =
        item.displayName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        item.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
        item.code.toLowerCase().includes(searchQuery.toLowerCase());
      return matchesCat && matchesSearch;
    });
  }, [selectedCategory, searchQuery]);

  const handleInsert = (item: ExpressionItemDef) => {
    setFormula((prev) => {
      if (!prev) return item.syntax || item.code;
      // If it's a preset, replace whole formula
      if (item.category === 'presets') {
        return item.code;
      }
      return `${prev} ${item.code}`;
    });
  };

  const handleApply = () => {
    onApply(formula);
    onClose();
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: {
          bgcolor: '#0B132B',
          color: '#E2E8F0',
          borderRadius: 2.5,
          border: '1px solid rgba(255,255,255,0.1)',
        },
      }}
    >
      {/* Title */}
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.08)', pb: 1.5 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Code2 size={22} color="#2DD4BF" />
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 800, color: '#F8FAFC' }}>
              {title}
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              SSRS, Crystal Reports &amp; Multi-Tenant System Variable Formula Engine
            </Typography>
          </Box>
        </Box>
        <Chip
          size="small"
          label={evaluationResult.error ? 'Formula Error' : 'Valid Expression'}
          color={evaluationResult.error ? 'error' : 'success'}
          sx={{ fontWeight: 700, fontSize: '0.72rem' }}
        />
      </DialogTitle>

      <DialogContent sx={{ p: 3 }}>
        <Grid container spacing={3}>
          
          {/* Left Column: Expression Editor & Real-Time Evaluator */}
          <Grid size={{ xs: 12, md: 7 }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              
              {/* Formula Input */}
              <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.8 }}>
                  <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8' }}>
                    {label}:
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>
                    Start with <code>=</code> for calculations or use <code>@variable</code> tags
                  </Typography>
                </Box>

                <TextField
                  inputRef={textFieldRef}
                  fullWidth
                  multiline
                  minRows={4}
                  maxRows={8}
                  value={formula}
                  onChange={(e) => setFormula(e.target.value)}
                  placeholder='=IIF(Fields!nav.Value > 1000000, "#10B981", "#EF4444") or /tenants/@tenant_code/@year/@month/'
                  InputProps={{
                    sx: {
                      fontFamily: 'monospace',
                      fontSize: '0.88rem',
                      bgcolor: 'rgba(15, 23, 42, 0.7)',
                      color: '#38BDF8',
                      fontWeight: 700,
                      borderRadius: 1.5,
                      border: '1px solid rgba(255,255,255,0.08)',
                    },
                  }}
                />
              </Box>

              {/* Live Evaluated Output Preview */}
              <Paper sx={{ p: 2, bgcolor: 'rgba(15, 23, 42, 0.8)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Sparkles size={14} color="#2DD4BF" /> Real-Time Evaluated Result:
                  </Typography>
                  <Chip
                    size="small"
                    label={evaluationResult.isFormula ? 'Dynamic Formula' : 'Variable Interpolated'}
                    sx={{ height: 18, fontSize: '0.65rem', bgcolor: 'rgba(13, 148, 136, 0.2)', color: '#2DD4BF', fontWeight: 700 }}
                  />
                </Box>
                <Typography
                  variant="body2"
                  sx={{
                    fontFamily: 'monospace',
                    color: evaluationResult.error ? '#EF4444' : '#2DD4BF',
                    fontWeight: 800,
                    fontSize: '0.9rem',
                    wordBreak: 'break-all',
                  }}
                >
                  {evaluationResult.error ? evaluationResult.error : String(evaluationResult.result)}
                </Typography>
              </Paper>

              {/* Context Preview Banner */}
              <Box sx={{ p: 1.5, bgcolor: 'rgba(0,0,0,0.25)', borderRadius: 1.5, border: '1px solid rgba(255,255,255,0.05)' }}>
                <Typography variant="caption" sx={{ color: '#64748B', display: 'block', mb: 0.5, fontWeight: 700 }}>
                  Live Evaluator Test Context:
                </Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.8 }}>
                  <Chip size="small" label="@tenant_code: acme_wealth" sx={{ fontSize: '0.68rem', bgcolor: 'rgba(255,255,255,0.05)', color: '#94A3B8' }} />
                  <Chip size="small" label="@slice_key: client-001" sx={{ fontSize: '0.68rem', bgcolor: 'rgba(255,255,255,0.05)', color: '#94A3B8' }} />
                  <Chip size="small" label="@seq: 001" sx={{ fontSize: '0.68rem', bgcolor: 'rgba(255,255,255,0.05)', color: '#94A3B8' }} />
                  <Chip size="small" label="Fields!nav: $1,450,000" sx={{ fontSize: '0.68rem', bgcolor: 'rgba(255,255,255,0.05)', color: '#94A3B8' }} />
                  <Chip size="small" label="Fields!status: Active" sx={{ fontSize: '0.68rem', bgcolor: 'rgba(255,255,255,0.05)', color: '#94A3B8' }} />
                </Box>
              </Box>

            </Box>
          </Grid>

          {/* Right Column: Unified Variable & Function Palette */}
          <Grid size={{ xs: 12, md: 5 }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, height: '100%' }}>
              
              {/* Search Bar */}
              <TextField
                fullWidth
                size="small"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search variables, fields, functions..."
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <Search size={15} color="#94A3B8" />
                    </InputAdornment>
                  ),
                  sx: {
                    bgcolor: 'rgba(15, 23, 42, 0.6)',
                    color: '#F8FAFC',
                    fontSize: '0.8rem',
                    borderRadius: 1.5,
                  },
                }}
              />

              {/* Category Pills */}
              <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                {[
                  { id: 'all', label: 'All' },
                  { id: 'system', label: '@System' },
                  { id: 'burst', label: '@Burst' },
                  { id: 'datetime', label: '@Date' },
                  { id: 'fields', label: 'Fields!' },
                  { id: 'parameters', label: 'Parameters!' },
                  { id: 'globals', label: 'Globals!' },
                  { id: 'functions', label: 'Functions' },
                  { id: 'presets', label: 'Presets' },
                ].map((cat) => (
                  <Chip
                    key={cat.id}
                    size="small"
                    label={cat.label}
                    onClick={() => setSelectedCategory(cat.id)}
                    sx={{
                      fontSize: '0.68rem',
                      fontWeight: 700,
                      cursor: 'pointer',
                      bgcolor: selectedCategory === cat.id ? '#0D9488' : 'rgba(255,255,255,0.06)',
                      color: selectedCategory === cat.id ? '#FFF' : '#94A3B8',
                      '&:hover': { bgcolor: '#0D9488', color: '#FFF' },
                    }}
                  />
                ))}
              </Box>

              {/* Item Palette List */}
              <Box sx={{ maxHeight: 330, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 0.8, pr: 0.5 }}>
                {filteredItems.map((item) => (
                  <Box
                    key={`${item.category}_${item.code}`}
                    onClick={() => handleInsert(item)}
                    sx={{
                      p: 1.2,
                      bgcolor: 'rgba(15, 23, 42, 0.6)',
                      border: '1px solid rgba(255,255,255,0.06)',
                      borderRadius: 1.5,
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      transition: 'all 0.15s',
                      '&:hover': {
                        bgcolor: 'rgba(13, 148, 136, 0.15)',
                        borderColor: '#2DD4BF',
                        transform: 'translateX(2px)',
                      },
                    }}
                  >
                    <Box sx={{ overflow: 'hidden', mr: 1 }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8 }}>
                        <Typography variant="caption" sx={{ fontFamily: 'monospace', fontWeight: 800, color: '#2DD4BF', fontSize: '0.78rem' }}>
                          {item.displayName}
                        </Typography>
                        <Chip
                          size="small"
                          label={item.category}
                          sx={{ height: 16, fontSize: '0.58rem', bgcolor: 'rgba(0,0,0,0.3)', color: '#94A3B8' }}
                        />
                      </Box>
                      <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: '0.68rem', display: 'block', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {item.description}
                      </Typography>
                    </Box>
                    <Chip
                      size="small"
                      label={`e.g. ${item.example}`}
                      sx={{ height: 18, fontSize: '0.62rem', fontFamily: 'monospace', bgcolor: 'rgba(255,255,255,0.05)', color: '#CBD5E1', flexShrink: 0 }}
                    />
                  </Box>
                ))}
              </Box>

            </Box>
          </Grid>
        </Grid>
      </DialogContent>

      <DialogActions sx={{ p: 2, borderTop: '1px solid rgba(255,255,255,0.08)', justifyContent: 'space-between' }}>
        <Button onClick={onClose} sx={{ color: '#94A3B8', textTransform: 'none', fontWeight: 700 }}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={handleApply}
          startIcon={<Check size={16} />}
          sx={{
            bgcolor: '#0D9488',
            color: '#FFF',
            fontWeight: 800,
            textTransform: 'none',
            px: 3,
            '&:hover': { bgcolor: '#0F766E' },
          }}
        >
          Apply Expression
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default UnifiedExpressionBuilderModal;
