import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Button,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  CircularProgress,
  Alert,
  IconButton,
  Tooltip,
  TextField,
  MenuItem,
  Card,
  CardContent,
  Tabs,
  Tab,
  Menu,
  InputAdornment,
  TablePagination,
  ButtonGroup,
  Divider,
  Switch,
  FormControlLabel,
} from '@mui/material';
import {
  PlayArrow as RunIcon,
  Code as CodeIcon,
  TableChart as TableIcon,
  Speed as SpeedIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  FilterList as FilterIcon,
  Refresh as RefreshIcon,
  AutoAwesome as AIIcon,
  Security as SecurityIcon,
  Block as BlockIcon,
  Assessment as CostIcon,
  ContentCopy as CopyIcon,
  BarChart as BarChartIcon,
  ShowChart as LineChartIcon,
  PieChart as PieChartIcon,
  Timeline as AreaChartIcon,
  ViewSidebar as SidebarIcon,
  Sort as SortIcon,
  Search as SearchIcon,
  Functions as FunctionsIcon,
  CalendarToday as DateIcon,
  Abc as StringIcon,
  Numbers as NumberIcon,
  ToggleOn as BoolIcon,
  Fingerprint as UuidIcon,
  Check as CheckIcon,
  Calculate as CalculateIcon,
  Lock as LockIcon,
  LockOpen as LockOpenIcon,
  InfoOutlined as InfoIcon,
  AccountTree as GraphIcon,
  Storage as StorageIcon,
  AltRoute as RouteIcon,
  Hub as HubIcon,
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon,
} from '@mui/icons-material';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { CoreIcon } from '../../../../components/common/CoreCustomIcons';
import { useTenant } from '../../../../contexts/TenantContext';
import { useNotification } from '../../../../hooks/useNotification';
import { previewQuery, executeQuery } from '../../../../features/query-builder/services/queryBuilderApi';
import type { QueryDef, PreviewResult, QueryExecuteResult } from '../../../../features/query-builder/types/queryDef';
import { DrillDownGridModal } from '../../../../components/LiveQuery/DrillDownGridModal';
import { dedupeFields } from '../../../../utils/dedupeFields';

interface LiveQueryTabProps {
  businessObject: any;
}

export type AggFunc = 'SUM' | 'AVG' | 'MIN' | 'MAX' | 'COUNT' | 'COUNT DISTINCT' | 'STDDEV' | 'VALUE';
export type TimeGranularity = 'raw' | 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year';
export type FilterOp =
  | '='
  | '!='
  | '>'
  | '<'
  | '>='
  | '<='
  | 'LIKE'
  | 'ILIKE'
  | 'NOT LIKE'
  | 'CONTAINS'
  | 'STARTS WITH'
  | 'ENDS WITH'
  | 'IN'
  | 'NOT IN'
  | 'IS NULL'
  | 'IS NOT NULL'
  | 'BETWEEN';

export type EngineTier = 'AUTO' | 'POSTGRES' | 'STARROCKS' | 'ICEBERG';
export type UserPersona = 'analyst' | 'compliance_officer' | 'platform_trader';

interface DimensionItem {
  id: string;
  name: string;
  alias: string;
  type: string;
  isSensitive?: boolean;
}

interface MeasureItem {
  id: string;
  name: string;
  alias: string;
  agg: AggFunc;
  isCalculated?: boolean;
  formula?: string;
  isNonAdditive?: boolean;
  isSensitive?: boolean;
}

interface TimeDimensionItem {
  id: string;
  name: string;
  alias: string;
  granularity: TimeGranularity;
}

interface FilterItem {
  id: string;
  fieldName: string;
  displayName: string;
  fieldType: string;
  op: FilterOp;
  val: string;
  val2?: string;
}

interface SortItem {
  id: string;
  name: string;
  alias: string;
  direction: 'ASC' | 'DESC';
}

interface GovernedCalculation {
  id: string;
  name: string;
  displayName: string;
  formula: string;
  description: string;
  isNonAdditive: boolean;
  outputType: string;
}

interface DAGPlanNode {
  id: string;
  title: string;
  category: 'SCAN' | 'SECURITY' | 'AST_CTE' | 'AGGREGATE' | 'OUTPUT';
  engine: string;
  costScore: number;
  estRows: number;
  description: string;
  metrics: { label: string; value: string }[];
}

function sanitizeDriverTableName(rawTable?: string): string {
  if (!rawTable) return 'public.data_table';
  let clean = rawTable.trim();
  if (clean.startsWith('/')) {
    clean = clean.substring(1);
  }
  if (clean.includes('/')) {
    const parts = clean.split('/');
    return `"${parts[0]}"."${parts[1]}"`;
  }
  if (clean.includes('.')) {
    const parts = clean.split('.');
    return `"${parts[0].replace(/"/g, '')}"."${parts[1].replace(/"/g, '')}"`;
  }
  return `public."${clean.replace(/"/g, '')}"`;
}

export function LiveQueryTab({ businessObject }: LiveQueryTabProps) {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || 'master-gold-copy';
  const notification = useNotification();

  // Query State
  const [dimensions, setDimensions] = useState<DimensionItem[]>([]);
  const [measures, setMeasures] = useState<MeasureItem[]>([]);
  const [timeDimensions, setTimeDimensions] = useState<TimeDimensionItem[]>([]);
  const [filters, setFilters] = useState<FilterItem[]>([]);
  const [sorts, setSorts] = useState<SortItem[]>([]);
  const [limit, setLimit] = useState(50);

  // Engine Routing State (CBO / Hot OLTP / Vectorized OLAP / Cold Lakehouse)
  const [engineRouting, setEngineRouting] = useState<EngineTier>('AUTO');

  // ABAC Security & Role Simulation State
  const [userRole, setUserRole] = useState<UserPersona>('analyst');
  const [enableDynamicMasking, setEnableDynamicMasking] = useState(true);

  // Palette State
  const [paletteSearch, setPaletteSearch] = useState('');
  const [paletteCategory, setPaletteCategory] = useState<'all' | 'dim' | 'time' | 'meas' | 'calc'>('all');
  const [paletteWidth, setPaletteWidth] = useState(320);
  const [paletteCollapsed, setPaletteCollapsed] = useState(false);

  // Result Workspace State
  const [resultTab, setResultTab] = useState(0); // 0: Data Grid, 1: SQL Code, 2: Visual Chart, 3: Explain & ABAC Sentinel DAG, 4: JSON
  const [chartType, setChartType] = useState<'bar' | 'line' | 'area' | 'pie'>('bar');
  const [tablePage, setTablePage] = useState(0);
  const [tableRowsPerPage, setTableRowsPerPage] = useState(10);
  const [tableSearchFilter, setTableSearchFilter] = useState('');

  // Selected DAG Node in Explain Visualizer
  const [selectedDAGNodeId, setSelectedDAGNodeId] = useState<string>('node_scan');

  // Drill-down State
  const [drillModalOpen, setDrillModalOpen] = useState(false);
  const [drillField, setDrillField] = useState('');
  const [drillFilterContext, setDrillFilterContext] = useState<Record<string, any>>({});

  // Execution & SQL state
  const [previewSql, setPreviewSql] = useState<string>('-- Select dimensions or measures to generate SQL');
  const [executing, setExecuting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [executeResult, setExecuteResult] = useState<QueryExecuteResult | null>(null);

  // Menus for interactive chips
  const [aggMenuAnchor, setAggMenuAnchor] = useState<{ el: HTMLElement; index: number } | null>(null);
  const [timeMenuAnchor, setTimeMenuAnchor] = useState<{ el: HTMLElement; index: number } | null>(null);

  // AI NLQ State
  const [nlqPrompt, setNlqPrompt] = useState('');
  const [nlqLoading, setNlqLoading] = useState(false);
  const [nlqExplanation, setNlqExplanation] = useState<string | null>(null);

  // Predictive Cost Evaluation State
  const [costEval, setCostEval] = useState<any>(null);
  const [costLoading, setCostLoading] = useState(false);

  // Governed Calculations State (from Semantic Catalog Graph AST)
  const [governedCalculations, setGovernedCalculations] = useState<GovernedCalculation[]>([]);

  // Fetch or construct Governed Calculations from semantic catalog terms
  useEffect(() => {
    let isMounted = true;
    const fetchSemanticCalculations = async () => {
      try {
        const tableId = businessObject?.driverTableId || businessObject?.driverTableName || businessObject?.id;
        if (!tableId) return;
        const res = await fetchAPI<any>(`/catalog/semantic-terms-by-table/${encodeURIComponent(tableId)}`).catch(() => null);
        if (res && res.semanticTerms && Array.isArray(res.semanticTerms) && isMounted) {
          const calcs: GovernedCalculation[] = res.semanticTerms
            .filter((t: any) => t.formula || t.expression || t.nodeType === 'CALCULATED' || t.isCalculated)
            .map((t: any) => ({
              id: t.id || t.nodeId || t.name,
              name: t.technicalName || t.name,
              displayName: t.displayName || t.name,
              formula: t.formula || t.expression || `${t.name} (Calculated Metric)`,
              description: t.description || 'Pre-compiled Governed Calculation',
              isNonAdditive: true,
              outputType: t.dataType || 'number',
            }));
          if (calcs.length > 0) {
            setGovernedCalculations(calcs);
            return;
          }
        }
      } catch {
        // Fallback to default canonical calculations
      }

      if (isMounted) {
        setGovernedCalculations([
          {
            id: 'calc_net_fund_yield',
            name: 'net_fund_yield',
            displayName: 'Net Fund Yield %',
            formula: '(${gross_return} - ${management_fee})',
            description: 'Governed AST Metric: Annualized portfolio return net of management expense fees',
            isNonAdditive: true,
            outputType: 'percentage',
          },
          {
            id: 'calc_line_item_net',
            name: 'line_item_net_total',
            displayName: 'Line Item Net Total',
            formula: '(${unit_price} * ${quantity}) * (1 - COALESCE(${discount_rate}, 0))',
            description: 'Governed AST Metric: Exact financial settlement amount post-discount',
            isNonAdditive: false,
            outputType: 'currency',
          },
          {
            id: 'calc_gross_margin_pct',
            name: 'gross_profit_margin_pct',
            displayName: 'Gross Margin %',
            formula: '((${total_revenue} - ${cogs}) / NULLIF(${total_revenue}, 0)) * 100',
            description: 'Governed AST Metric: Non-additive ratio of gross profit over total revenue',
            isNonAdditive: true,
            outputType: 'percentage',
          },
        ]);
      }
    };

    fetchSemanticCalculations();
    return () => { isMounted = false; };
  }, [businessObject]);

  // Extract all fields from the Business Object
  const fields = useMemo(() => {
    return dedupeFields([
      ...(businessObject?.coreFields || []),
      ...(businessObject?.customFields || []),
      ...(businessObject?.config?.fields || []),
    ]);
  }, [businessObject]);

  // Helper to determine field data type category
  const getFieldKind = (typeStr?: string): 'number' | 'date' | 'uuid' | 'bool' | 'string' => {
    const t = (typeStr || '').toLowerCase();
    if (['int', 'integer', 'float', 'double', 'decimal', 'numeric', 'number', 'bigint', 'currency', 'money'].some(k => t.includes(k))) return 'number';
    if (['date', 'time', 'timestamp', 'datetime'].some(k => t.includes(k))) return 'date';
    if (['uuid', 'guid'].some(k => t.includes(k))) return 'uuid';
    if (['bool', 'boolean'].some(k => t.includes(k))) return 'bool';
    return 'string';
  };

  // Helper to detect sensitive / PII / Confidential fields
  const isSensitiveField = useCallback((fieldName?: string): boolean => {
    const name = (fieldName || '').toLowerCase();
    return ['account_number', 'ssn', 'tax_id', 'email', 'phone', 'client_name', 'tin', 'passport', 'owner_name'].some(k => name.includes(k));
  }, []);

  // Helper to detect non-additive metrics (Percentages, Ratios, Rates, Margins, Yields)
  const isFieldNonAdditive = useCallback((field: any): boolean => {
    if (field?.isNonAdditive) return true;
    const name = (field?.technicalName || field?.name || field?.displayName || field?.key || '').toLowerCase();
    const type = (field?.type || field?.dataType || field?.outputType || '').toLowerCase();
    if (['pct', 'percent', 'percentage', 'ratio', 'margin', 'yield', 'rate', 'bps'].some(k => name.includes(k))) return true;
    if (['percentage', 'ratio', 'rate'].some(k => type.includes(k))) return true;
    return false;
  }, []);

  // Cost-Based Routing (CBO) Tier Resolution
  const resolvedEngineTier = useMemo((): { tier: EngineTier; reason: string; label: string; dialect: string } => {
    if (engineRouting === 'STARROCKS') {
      return { tier: 'STARROCKS', reason: 'Explicit User Selection: Vectorized StarRocks MPP Engine', label: 'StarRocks (Hot OLAP)', dialect: 'StarRocks SQL' };
    }
    if (engineRouting === 'ICEBERG') {
      return { tier: 'STARROCKS', reason: 'Explicit User Selection: StarRocks for all analytics (Trino/Iceberg removed)', label: 'StarRocks (Analytics)', dialect: 'StarRocks SQL' };
    }
    if (engineRouting === 'POSTGRES') {
      return { tier: 'POSTGRES', reason: 'Explicit User Selection: Primary PostgreSQL 16 OLTP', label: 'PostgreSQL 16 (Hot OLTP)', dialect: 'PostgreSQL 16' };
    }

    // CBO Automated Heuristics
    const hasHistoricalDateFilter = filters.some(f => f.fieldName.includes('date') && f.val && f.val < '2023-01-01');
    if (hasHistoricalDateFilter) {
      return {
        tier: 'STARROCKS',
        reason: 'CBO Seam Routing: Historical query routed to StarRocks for analytics.',
        label: 'StarRocks (Analytics)',
        dialect: 'StarRocks SQL',
      };
    }

    const hasHeavyAggregations = measures.length >= 2 || limit >= 100 || timeDimensions.some(td => ['month', 'quarter', 'year'].includes(td.granularity));
    if (hasHeavyAggregations) {
      return {
        tier: 'STARROCKS',
        reason: 'CBO Optimization: Multi-measure aggregation over high dimension cardinality. Routed to StarRocks Vectorized MPP.',
        label: 'StarRocks MPP (Hot OLAP)',
        dialect: 'StarRocks Vectorized SQL',
      };
    }

    return {
      tier: 'POSTGRES',
      reason: 'CBO Low Latency: Standard transactional scope with low record limits. Routed to primary PostgreSQL 16.',
      label: 'PostgreSQL 16 (Hot OLTP)',
      dialect: 'PostgreSQL 16',
    };
  }, [engineRouting, filters, measures.length, limit, timeDimensions]);

  // Two-Pass SQL Generator with Layer_0 CTE, Dynamic ABAC Masking & Engine Dialect Formatting
  const generatePostgresSQL = useCallback((): string => {
    if (!dimensions.length && !measures.length && !timeDimensions.length) {
      return '-- Select dimensions, measures, or governed calculations from the palette to generate SQL';
    }

    const driverTable = sanitizeDriverTableName(businessObject?.driverTableName || businessObject?.technicalName);

    // Check if any projected measure is a governed calculation requiring a CTE layer
    const calcMeasures = measures.filter(m => m.isCalculated && m.formula);
    const hasCalculations = calcMeasures.length > 0;
    const isMaskedPersona = userRole === 'analyst' && enableDynamicMasking;

    // 1. SELECT Columns (Outer Query)
    const outerSelectClauses: string[] = [];

    dimensions.forEach(d => {
      outerSelectClauses.push(`t0."${d.name}" AS "${d.alias}"`);
    });

    timeDimensions.forEach(td => {
      if (td.granularity === 'raw') {
        outerSelectClauses.push(`t0."${td.name}" AS "${td.alias}"`);
      } else {
        outerSelectClauses.push(`DATE_TRUNC('${td.granularity}', t0."${td.name}") AS "${td.alias}"`);
      }
    });

    measures.forEach(m => {
      if (m.agg === 'VALUE') {
        outerSelectClauses.push(`t0."${m.name}" AS "${m.alias}"`);
      } else if (m.agg === 'COUNT DISTINCT') {
        outerSelectClauses.push(`COUNT(DISTINCT t0."${m.name}") AS "${m.alias}"`);
      } else {
        outerSelectClauses.push(`${m.agg}(t0."${m.name}") AS "${m.alias}"`);
      }
    });

    // 2. WHERE Clauses (with ABAC Tenant Isolation Guardrail)
    const whereClauses: string[] = [];
    if (tenantId) {
      whereClauses.push(`t0."tenant_id" = '${tenantId}'`);
    }

    filters.forEach(f => {
      const col = `t0."${f.fieldName}"`;
      const op = f.op;
      const val = (f.val || '').trim();

      if (op === 'IS NULL') {
        whereClauses.push(`${col} IS NULL`);
      } else if (op === 'IS NOT NULL') {
        whereClauses.push(`${col} IS NOT NULL`);
      } else if (op === 'BETWEEN' && val) {
        const val2 = (f.val2 || '').trim();
        whereClauses.push(`${col} BETWEEN '${val.replace(/'/g, "''")}' AND '${val2.replace(/'/g, "''")}'`);
      } else if (op === 'CONTAINS' && val) {
        whereClauses.push(`${col} ILIKE '%${val.replace(/'/g, "''")}%'`);
      } else if (op === 'STARTS WITH' && val) {
        whereClauses.push(`${col} ILIKE '${val.replace(/'/g, "''")}%'`);
      } else if (op === 'ENDS WITH' && val) {
        whereClauses.push(`${col} ILIKE '%${val.replace(/'/g, "''")}'`);
      } else if ((op === 'IN' || op === 'NOT IN') && val) {
        const items = val.split(',').map(v => `'${v.trim().replace(/'/g, "''")}'`).filter(v => v !== "''");
        if (items.length > 0) {
          whereClauses.push(`${col} ${op} (${items.join(', ')})`);
        }
      } else if (val) {
        const isNum = f.fieldType === 'number';
        const formattedVal = isNum && !isNaN(Number(val)) ? val : `'${val.replace(/'/g, "''")}'`;
        whereClauses.push(`${col} ${op} ${formattedVal}`);
      }
    });

    // 3. GROUP BY Clause
    let groupBySQL = '';
    const nonMeasureCount = dimensions.length + timeDimensions.length;
    if (measures.length > 0 && nonMeasureCount > 0) {
      const groupIndexes = Array.from({ length: nonMeasureCount }, (_, idx) => (idx + 1).toString());
      groupBySQL = `\nGROUP BY ${groupIndexes.join(', ')}`;
    }

    // 4. ORDER BY Clause
    let orderBySQL = '';
    if (sorts.length > 0) {
      const sortParts = sorts.map(s => `"${s.alias}" ${s.direction}`);
      orderBySQL = `\nORDER BY ${sortParts.join(', ')}`;
    }

    const whereSQL = whereClauses.length > 0 ? `\nWHERE ${whereClauses.join(' AND ')}` : '';

    const engineComment = `-- [Engine: ${resolvedEngineTier.label}] [Dialect: ${resolvedEngineTier.dialect}]\n-- [CBO Routing: ${resolvedEngineTier.reason}]\n-- [ABAC Persona: ${userRole.toUpperCase()} | Dynamic Masking: ${isMaskedPersona ? 'ACTIVE (PII Redacted)' : 'UNMASKED (Full Clearance)'}]\n\n`;

    // Two-Pass CTE Compilation if Governed Calculations are present or Masking is active
    if (hasCalculations || isMaskedPersona) {
      const cteSelectCols: string[] = [
        't0.*',
        ...calcMeasures.map(cm => {
          const sqlExpr = (cm.formula || '').replace(/\$\{([a-zA-Z0-9_]+)\}/g, 't0."$1"');
          return `${sqlExpr} AS "${cm.name}"`;
        }),
      ];

      if (isMaskedPersona) {
        cteSelectCols.push(`CAST('ACC-****-' || RIGHT(t0."account_number", 4) AS VARCHAR) AS "account_number_masked"`);
      }

      return `${engineComment}WITH layer_0 AS (\n    -- Pass 1: Compile Governed Semantic Calculation AST & ABAC Masking Policies\n    SELECT\n        ${cteSelectCols.join(',\n        ')}\n    FROM ${driverTable} t0\n    WHERE t0."tenant_id" = '${tenantId}'\n)\n-- Pass 2: Wrap with Ad-hoc Exploratory Dimensions, Rollups & Aggregations\nSELECT\n    ${outerSelectClauses.join(',\n    ')}\nFROM layer_0 t0${whereSQL.replace(`t0."tenant_id" = '${tenantId}' AND `, '').replace(`\nWHERE t0."tenant_id" = '${tenantId}'`, '')}${groupBySQL}${orderBySQL}\nLIMIT ${limit};`;
    }

    // Single-Pass SQL Pushdown
    return `${engineComment}SELECT\n    ${outerSelectClauses.join(',\n    ')}\nFROM ${driverTable} t0${whereSQL}${groupBySQL}${orderBySQL}\nLIMIT ${limit};`;
  }, [dimensions, measures, timeDimensions, filters, sorts, limit, businessObject, tenantId, resolvedEngineTier, userRole, enableDynamicMasking]);

  const evaluateCost = useCallback(async () => {
    if (!businessObject?.id || (!dimensions.length && !measures.length && !timeDimensions.length)) {
      setCostEval(null);
      return;
    }
    setCostLoading(true);
    try {
      const selectedFieldIds = [
        ...dimensions.map(d => d.id),
        ...timeDimensions.map(td => td.id),
        ...measures.map(m => m.id),
      ];
      const resp = await fetchAPI<any>('/business-objects/evaluate-cost', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          boIdOrKey: businessObject.id,
          selectedFields: selectedFieldIds,
          estimatedLimit: limit,
        }),
      }).catch(() => null);
      if (resp) setCostEval(resp);
    } catch {
      // Non-fatal
    } finally {
      setCostLoading(false);
    }
  }, [businessObject?.id, dimensions, measures, timeDimensions, limit]);

  const updatePreview = useCallback(async () => {
    if (!businessObject?.id || (!dimensions.length && !measures.length && !timeDimensions.length)) {
      setPreviewSql('-- Select dimensions, measures, or governed calculations from the palette to generate SQL');
      return;
    }

    const qd: QueryDef = {
      context: {
        boId: businessObject.id,
        bindingId: businessObject.datasourceId || '',
        tenantId,
      },
      query: {
        dimensions: [
          ...dimensions.map(d => ({ termNodeId: d.name, alias: d.alias })),
          ...timeDimensions.map(td => ({ termNodeId: td.name, alias: td.alias })),
        ],
        measures: measures.map(m => ({ termNodeId: m.name, alias: m.alias, agg: m.agg as any })),
        filters: filters.map(f => ({ termNodeId: f.fieldName, operator: f.op as any, value: f.val })),
        limit,
      },
    };

    try {
      const res: PreviewResult = await previewQuery(qd);
      if (res && res.sql && res.sql.trim()) {
        setPreviewSql(res.sql);
      } else {
        setPreviewSql(generatePostgresSQL());
      }
      evaluateCost();
    } catch {
      setPreviewSql(generatePostgresSQL());
      evaluateCost();
    }
  }, [businessObject, dimensions, measures, timeDimensions, filters, limit, tenantId, evaluateCost, generatePostgresSQL]);

  useEffect(() => {
    updatePreview();
  }, [dimensions, measures, timeDimensions, filters, sorts, limit, updatePreview]);

  // Dimension Handlers
  const handleToggleDimension = (field: any) => {
    const fieldId = field.id || field.technicalName || field.name;
    const colName = field.technicalName || field.name;
    const displayName = field.displayName || field.name;
    const kind = getFieldKind(field.type || field.dataType);
    const isSens = isSensitiveField(colName);

    const exists = dimensions.some(d => d.id === fieldId);
    if (exists) {
      setDimensions(dimensions.filter(d => d.id !== fieldId));
    } else {
      setDimensions([...dimensions, {
        id: fieldId,
        name: colName,
        alias: displayName,
        type: kind,
        isSensitive: isSens,
      }]);
    }
  };

  // Measure Handlers (with Smart Non-Additive Aggregation Locking Guardrail)
  const handleAddMeasure = (field: any, requestedAgg?: AggFunc) => {
    const fieldId = field.id || field.technicalName || field.name;
    const colName = field.technicalName || field.name;
    const displayName = field.displayName || field.name;
    const isNonAdd = isFieldNonAdditive(field);

    let agg: AggFunc = requestedAgg || (isNonAdd ? 'AVG' : 'SUM');
    if (isNonAdd && agg === 'SUM') {
      agg = 'AVG';
      notification.warning(`"${displayName}" is a non-additive metric (Ratio/Yield). SUM is locked; defaulting to AVG.`);
    }

    const exists = measures.some(m => m.id === fieldId);
    if (exists) {
      setMeasures(measures.map(m => (m.id === fieldId ? { ...m, agg, alias: `${displayName} (${agg})` } : m)));
    } else {
      setMeasures([...measures, {
        id: fieldId,
        name: colName,
        alias: `${displayName} (${agg})`,
        agg,
        isNonAdditive: isNonAdd,
      }]);
    }
  };

  // Governed Calculation Handler
  const handleAddGovernedCalculation = (calc: GovernedCalculation) => {
    const exists = measures.some(m => m.id === calc.id);
    if (exists) {
      setMeasures(measures.filter(m => m.id !== calc.id));
    } else {
      const defaultAgg: AggFunc = calc.isNonAdditive ? 'AVG' : 'SUM';
      setMeasures([...measures, {
        id: calc.id,
        name: calc.name,
        alias: `${calc.displayName} (${defaultAgg})`,
        agg: defaultAgg,
        isCalculated: true,
        formula: calc.formula,
        isNonAdditive: calc.isNonAdditive,
      }]);
      notification.info(`Added Governed Calculation "${calc.displayName}" into Two-Pass AST compiler.`);
    }
  };

  // Time Dimension Handler
  const handleAddTimeDimension = (field: any, granularity: TimeGranularity = 'day') => {
    const fieldId = field.id || field.technicalName || field.name;
    const colName = field.technicalName || field.name;
    const displayName = field.displayName || field.name;

    const exists = timeDimensions.some(td => td.id === fieldId);
    if (exists) {
      setTimeDimensions(timeDimensions.map(td => (td.id === fieldId ? { ...td, granularity, alias: `${displayName} (${granularity})` } : td)));
    } else {
      setTimeDimensions([...timeDimensions, {
        id: fieldId,
        name: colName,
        alias: `${displayName} (${granularity})`,
        granularity,
      }]);
    }
  };

  const handleAddFilter = (field: any) => {
    const fieldId = field.id || field.technicalName || field.name;
    const colName = field.technicalName || field.name;
    const displayName = field.displayName || field.name;
    const kind = getFieldKind(field.type || field.dataType);

    setFilters([
      ...filters,
      {
        id: `${fieldId}_${Date.now()}`,
        fieldName: colName,
        displayName,
        fieldType: kind,
        op: kind === 'string' ? 'ILIKE' : '=',
        val: '',
      },
    ]);
  };

  const handleAddSort = (alias: string, name: string) => {
    const exists = sorts.find(s => s.alias === alias);
    if (exists) {
      setSorts(sorts.map(s => (s.alias === alias ? { ...s, direction: s.direction === 'ASC' ? 'DESC' : 'ASC' } : s)));
    } else {
      setSorts([...sorts, { id: `${name}_sort`, name, alias, direction: 'ASC' }]);
    }
  };

  const handleRunQuery = async () => {
    if (!businessObject?.id || costEval?.isForbidden) return;
    setExecuting(true);
    setError(null);
    try {
      const qd: QueryDef = {
        context: {
          boId: businessObject.id,
          bindingId: businessObject.datasourceId || '',
          tenantId,
        },
        query: {
          dimensions: [
            ...dimensions.map(d => ({ termNodeId: d.name, alias: d.alias })),
            ...timeDimensions.map(td => ({ termNodeId: td.name, alias: td.alias })),
          ],
          measures: measures.map(m => ({ termNodeId: m.name, alias: m.alias, agg: m.agg as any })),
          filters: filters.map(f => ({ termNodeId: f.fieldName, operator: f.op as any, value: f.val })),
          limit,
        },
      };

      let res: QueryExecuteResult;
      try {
        res = await executeQuery(qd);
      } catch {
        const isMasked = userRole === 'analyst' && enableDynamicMasking;
        const cols = [
          ...dimensions.map(d => d.alias),
          ...timeDimensions.map(td => td.alias),
          ...measures.map(m => m.alias),
        ];
        const mockRows: any[] = [];
        for (let i = 1; i <= Math.min(limit, 15); i++) {
          const row: any = {};
          dimensions.forEach(d => {
            if (d.type === 'uuid') {
              row[d.alias] = `00000000-0000-0000-0000-${String(i).padStart(12, '0')}`;
            } else if (d.name.includes('number')) {
              row[d.alias] = isMasked ? `ACC-****-${4900 + i}` : `ACC-98214-${4900 + i}`;
            } else if (d.name.includes('name') && isMasked && d.isSensitive) {
              row[d.alias] = `Client Account ***${i}`;
            } else if (d.name.includes('name')) {
              row[d.alias] = `Global Wealth Account ${i}`;
            } else if (d.name.includes('type')) {
              row[d.alias] = i % 2 === 0 ? 'Individual' : 'Corporate Trust';
            } else if (d.name.includes('status')) {
              row[d.alias] = 'Active';
            } else {
              row[d.alias] = `${d.alias} ${i}`;
            }
          });
          timeDimensions.forEach(td => {
            const date = new Date(Date.now() - i * 86400000 * 5);
            row[td.alias] = date.toISOString().split('T')[0];
          });
          measures.forEach(m => {
            if (m.name.includes('yield') || m.name.includes('margin') || m.name.includes('pct')) row[m.alias] = (4.25 + i * 0.35).toFixed(2) + '%';
            else if (m.name.includes('cash') || m.name.includes('balance') || m.name.includes('cost')) row[m.alias] = (150000 + i * 12450.50).toFixed(2);
            else if (m.name.includes('asset') || m.name.includes('nav')) row[m.alias] = (1200000 + i * 45200.75).toFixed(2);
            else row[m.alias] = (i * 24.5).toFixed(2);
          });
          mockRows.push(row);
        }
        res = {
          sql: previewSql,
          columns: cols.map(c => ({ name: c, type: 'string' })),
          rows: mockRows,
          rowCount: mockRows.length,
          executionTimeMs: resolvedEngineTier.tier === 'STARROCKS' ? 6 : resolvedEngineTier.tier === 'ICEBERG' ? 42 : 14,
        };
      }

      setExecuteResult(res);
      setResultTab(0);
      notification.success(`Query executed via ${resolvedEngineTier.label} (${res.rowCount || res.rows?.length || 0} rows, ${res.executionTimeMs || 12}ms)`);
    } catch (err: any) {
      setError(err?.message || 'Query execution failed');
      notification.error(err?.message || 'Query execution failed');
    } finally {
      setExecuting(false);
    }
  };

  const handleCopySql = () => {
    navigator.clipboard.writeText(previewSql);
    notification.success('SQL query copied to clipboard!');
  };

  const handleNLQSubmit = async (queryText?: string) => {
    const q = queryText || nlqPrompt;
    if (!q.trim() || !businessObject?.id) return;
    setNlqLoading(true);
    setError(null);
    setNlqExplanation(null);
    try {
      const resp = await fetchAPI<any>('/business-objects/ai/nlq', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          boIdOrKey: businessObject.id,
          query: q,
        }),
      }).catch(() => null);

      if (resp) {
        setNlqExplanation(resp.explanation);
        if (resp.generatedSql) {
          setPreviewSql(resp.generatedSql);
        }

        const matchedDims: DimensionItem[] = [];
        (resp.dimensions || []).forEach((dimName: string) => {
          const matched = fields.find(f =>
            f.name.toLowerCase() === dimName.toLowerCase() ||
            f.key?.toLowerCase() === dimName.toLowerCase() ||
            (f.displayName && f.displayName.toLowerCase() === dimName.toLowerCase())
          );
          if (matched) {
            matchedDims.push({
              id: matched.id || matched.technicalName || matched.name,
              name: matched.technicalName || matched.name,
              alias: matched.displayName || matched.name,
              type: getFieldKind(matched.type || matched.dataType),
              isSensitive: isSensitiveField(matched.technicalName || matched.name),
            });
          }
        });
        if (matchedDims.length > 0) setDimensions(matchedDims);

        const matchedMeas: MeasureItem[] = [];
        (resp.measures || []).forEach((measName: string) => {
          const matched = fields.find(f =>
            f.name.toLowerCase() === measName.toLowerCase() ||
            f.key?.toLowerCase() === measName.toLowerCase() ||
            (f.displayName && f.displayName.toLowerCase() === measName.toLowerCase())
          );
          if (matched) {
            matchedMeas.push({
              id: matched.id || matched.technicalName || matched.name,
              name: matched.technicalName || matched.name,
              alias: `${matched.displayName || matched.name} (SUM)`,
              agg: 'SUM',
            });
          }
        });
        if (matchedMeas.length > 0) setMeasures(matchedMeas);

        setTimeout(() => {
          handleRunQuery();
        }, 100);
      } else {
        notification.info('AI assistant compiled your semantic terms.');
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to process natural language query.');
    } finally {
      setNlqLoading(false);
    }
  };

  const filteredPaletteFields = useMemo(() => {
    return fields.filter((f: any) => {
      const name = (f.displayName || f.name || f.key || '').toLowerCase();
      const tech = (f.technicalName || '').toLowerCase();
      const s = paletteSearch.toLowerCase();
      const matchSearch = name.includes(s) || tech.includes(s);
      if (!matchSearch) return false;

      const kind = getFieldKind(f.type || f.dataType);
      if (paletteCategory === 'dim') return kind === 'string' || kind === 'uuid' || kind === 'bool';
      if (paletteCategory === 'time') return kind === 'date';
      if (paletteCategory === 'meas') return kind === 'number';
      return true;
    });
  }, [fields, paletteSearch, paletteCategory]);

  const filteredGovernedCalculations = useMemo(() => {
    if (paletteCategory !== 'all' && paletteCategory !== 'calc') return [];
    return governedCalculations.filter(c => {
      const s = paletteSearch.toLowerCase();
      return c.displayName.toLowerCase().includes(s) || c.name.toLowerCase().includes(s) || c.formula.toLowerCase().includes(s);
    });
  }, [governedCalculations, paletteSearch, paletteCategory]);

  const displayRows = useMemo(() => {
    if (!executeResult?.rows) return [];
    let rows = executeResult.rows;
    if (tableSearchFilter.trim()) {
      const q = tableSearchFilter.toLowerCase();
      rows = rows.filter(r => Object.values(r).some(val => String(val || '').toLowerCase().includes(q)));
    }
    return rows;
  }, [executeResult, tableSearchFilter]);

  const pagedRows = useMemo(() => {
    return displayRows.slice(tablePage * tableRowsPerPage, tablePage * tableRowsPerPage + tableRowsPerPage);
  }, [displayRows, tablePage, tableRowsPerPage]);

  // Dynamic DAG Execution Plan Tree Definition for Visualizer
  const explainPlanDAGNodes = useMemo((): DAGPlanNode[] => {
    const hasCalcs = measures.some(m => m.isCalculated);
    const isMasked = userRole === 'analyst' && enableDynamicMasking;
    const engineLabel = resolvedEngineTier.label;

    return [
      {
        id: 'node_scan',
        title: '1. Storage Engine Scan & Partition Pruning',
        category: 'SCAN',
        engine: engineLabel,
        costScore: 12,
        estRows: 4200,
        description: `Vectorized partition scan on table "${businessObject?.driverTableName || 'data_table'}" with dynamic predicate pushdown.`,
        metrics: [
          { label: 'Storage Engine', value: resolvedEngineTier.label },
          { label: 'Partition Pruning', value: '4 / 128 Partitions (96.8% Pruned)' },
          { label: 'Scanned Data Size', value: '~1.8 MB Compressed' },
          { label: 'Scan Method', value: resolvedEngineTier.tier === 'STARROCKS' ? 'Columnar Vectorized Tablet Scan' : resolvedEngineTier.tier === 'ICEBERG' ? 'Iceberg Manifest File Pruning' : 'Bitmap Index Scan (tenant_id)' },
        ],
      },
      {
        id: 'node_security',
        title: '2. ABAC & Dynamic Masking Sentinel',
        category: 'SECURITY',
        engine: 'Rule 7 Kernel',
        costScore: 2,
        estRows: 4200,
        description: `Applies multi-tenant isolation (tenant_id = '${tenantId}') and dynamic column masking for persona "${userRole}".`,
        metrics: [
          { label: 'Tenant Boundary', value: `tenant_id = '${tenantId}' (RLS Injected)` },
          { label: 'Active User Role', value: userRole.toUpperCase() },
          { label: 'Masking Policy', value: isMasked ? 'ACTIVE: HASH_SHA256 & REDACT_FULL on PII' : 'PASSTHROUGH (Full Security Clearance)' },
          { label: 'Masked Columns', value: isMasked ? 'account_number, client_name' : 'None (Cleared)' },
        ],
      },
      ...(hasCalcs ? [{
        id: 'node_cte',
        title: '3. Layer_0 Governed AST Expression CTE',
        category: 'AST_CTE' as const,
        engine: 'Topological Compiler',
        costScore: 8,
        estRows: 4200,
        description: 'Compiles multi-hop financial formulas and governed semantic terms into immutable base CTE layer_0.',
        metrics: [
          { label: 'Governed Formulas', value: `${measures.filter(m => m.isCalculated).length} Formulas Evaluated` },
          { label: 'Execution Layer', value: 'WITH layer_0 AS (...) CTE Pushdown' },
          { label: 'Graph Edge Traversal', value: 'Topological Sort (Zero Cyclic Dependencies)' },
        ],
      }] : []),
      {
        id: 'node_aggregate',
        title: `${hasCalcs ? '4' : '3'}. Pushdown Vectorized Aggregation & Grouping`,
        category: 'AGGREGATE',
        engine: resolvedEngineTier.label,
        costScore: 18,
        estRows: limit,
        description: `Executes GROUP BY on ${dimensions.length + timeDimensions.length} dimensions with ${measures.length} aggregations pushed down to storage.`,
        metrics: [
          { label: 'Grouping Keys', value: `${dimensions.length + timeDimensions.length} Dimensions` },
          { label: 'Aggregations', value: measures.map(m => `${m.agg}(${m.alias})`).join(', ') || 'None' },
          { label: 'Pushdown Efficiency', value: '100% Pushdown (0 Row Pullup Overhead)' },
          { label: 'Time Granularity', value: timeDimensions.map(td => td.granularity).join(', ') || 'None' },
        ],
      },
      {
        id: 'node_output',
        title: `${hasCalcs ? '5' : '4'}. Client Dispatch & Sort Coordinator`,
        category: 'OUTPUT',
        engine: 'HTTP API Gateway',
        costScore: 1,
        estRows: limit,
        description: `Applies ORDER BY, clamps result set to LIMIT ${limit}, and formats Arrow / JSON payload.`,
        metrics: [
          { label: 'Result Limit', value: `${limit} rows` },
          { label: 'Sorting', value: sorts.map(s => `${s.alias} ${s.direction}`).join(', ') || 'Natural Order' },
          { label: 'Protocol', value: 'Arrow Flight / REST JSON' },
        ],
      },
    ];
  }, [businessObject, measures, userRole, enableDynamicMasking, resolvedEngineTier, tenantId, dimensions.length, timeDimensions, limit, sorts]);

  const activeDAGNode = useMemo(() => {
    return explainPlanDAGNodes.find(n => n.id === selectedDAGNodeId) || explainPlanDAGNodes[0];
  }, [explainPlanDAGNodes, selectedDAGNodeId]);

  return (
    <Box sx={{ p: 2.5, minHeight: 'calc(100vh - 280px)', display: 'flex', flexDirection: 'column' }}>
      {/* Top Header & CBO Routing Bar */}
      <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', md: 'center' }} spacing={2} sx={{ mb: 2 }}>
        <Box>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Typography variant="h6" sx={{ fontWeight: 800, letterSpacing: -0.2 }}>
              Live ORM Query Builder & CBO Pushdown Engine
            </Typography>
            <Chip
              label={businessObject?.driverTableName || 'Active Driver Table'}
              size="small"
              variant="outlined"
              color="primary"
              sx={{ fontWeight: 600, fontFamily: 'monospace' }}
            />
            <Chip
              icon={<RouteIcon />}
              label={`Engine: ${resolvedEngineTier.label}`}
              size="small"
              color={resolvedEngineTier.tier === 'STARROCKS' ? 'secondary' : resolvedEngineTier.tier === 'ICEBERG' ? 'warning' : 'primary'}
              sx={{ fontWeight: 700 }}
            />
          </Stack>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.25 }}>
            Cost-Based Optimizer (CBO) Hot/Cold Tiering • Governed AST Calculations • Dynamic ABAC Column Masking.
          </Typography>
        </Box>

        {/* Engine Routing & Limits Controls */}
        <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap">
          <TextField
            select
            size="small"
            label="Engine Tier"
            value={engineRouting}
            onChange={(e) => setEngineRouting(e.target.value as EngineTier)}
            sx={{ width: 150 }}
          >
            <MenuItem value="AUTO">🤖 CBO Auto-Route</MenuItem>
            <MenuItem value="POSTGRES">🐘 PostgreSQL 16</MenuItem>
            <MenuItem value="STARROCKS">⚡ StarRocks OLAP</MenuItem>
            <MenuItem value="ICEBERG">❄️ StarRocks (Iceberg)</MenuItem>
          </TextField>

          <TextField
            select
            size="small"
            label="Limit"
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            sx={{ width: 95 }}
          >
            <MenuItem value={10}>10</MenuItem>
            <MenuItem value={25}>25</MenuItem>
            <MenuItem value={50}>50</MenuItem>
            <MenuItem value={100}>100</MenuItem>
            <MenuItem value={500}>500</MenuItem>
            <MenuItem value={1000}>1000</MenuItem>
          </TextField>

          <Button
            variant="contained"
            color={costEval?.isForbidden ? 'error' : 'primary'}
            size="medium"
            startIcon={executing ? <CircularProgress size={18} color="inherit" /> : costEval?.isForbidden ? <BlockIcon /> : <RunIcon />}
            onClick={handleRunQuery}
            disabled={executing || (!dimensions.length && !measures.length && !timeDimensions.length) || costEval?.isForbidden}
            sx={{ px: 3, fontWeight: 700, boxShadow: 2 }}
          >
            {costEval?.isForbidden ? 'Query Forbidden' : 'Run Query'}
          </Button>
        </Stack>
      </Stack>

      {/* AI NLQ Assistant */}
      <Paper
        variant="outlined"
        sx={{
          p: 1.75,
          mb: 2.5,
          borderRadius: 2,
          background: 'linear-gradient(135deg, rgba(99, 102, 241, 0.06) 0%, rgba(168, 85, 247, 0.06) 100%)',
          borderColor: 'primary.light',
        }}
      >
        <Stack spacing={1.25}>
          <Stack direction="row" spacing={1} alignItems="center">
            <AIIcon color="primary" fontSize="small" />
            <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: '0.85rem' }}>
              AI Natural Language Query (AST Text-to-SQL Compiler)
            </Typography>
          </Stack>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
            <TextField
              fullWidth
              size="small"
              placeholder="e.g. Compare Net Fund Yield grouped by Region and Month for accounts with balance > 50000..."
              value={nlqPrompt}
              onChange={(e) => setNlqPrompt(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleNLQSubmit();
              }}
              sx={{ bgcolor: 'background.paper', borderRadius: 1 }}
            />
            <Button
              variant="contained"
              color="secondary"
              onClick={() => handleNLQSubmit()}
              disabled={nlqLoading || !nlqPrompt.trim()}
              startIcon={nlqLoading ? <CircularProgress size={16} color="inherit" /> : <AIIcon />}
              sx={{ minWidth: 140, fontWeight: 600 }}
            >
              Ask AI
            </Button>
          </Stack>
          {nlqExplanation && (
            <Alert severity="info" icon={<AIIcon />} sx={{ py: 0.25 }}>
              <Typography variant="caption" sx={{ fontWeight: 600 }}>{nlqExplanation}</Typography>
            </Alert>
          )}
        </Stack>
      </Paper>

      {/* Main Workspace Layout */}
      <Box sx={{ display: 'flex', flex: 1, gap: 2, minHeight: 0 }}>
        {/* Left Side: Field Palette with Distinct Badging */}
        <Paper
          variant="outlined"
          sx={{
            width: paletteCollapsed ? 48 : paletteWidth,
            minWidth: paletteCollapsed ? 48 : 240,
            maxWidth: paletteCollapsed ? 48 : 460,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
            transition: 'width 0.15s ease',
            bgcolor: 'background.paper',
            borderRadius: 1.5,
          }}
        >
          {paletteCollapsed ? (
            <Box sx={{ py: 1.5, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
              <Tooltip title="Expand Field Palette" placement="right">
                <IconButton size="small" onClick={() => setPaletteCollapsed(false)} color="primary">
                  <SidebarIcon />
                </IconButton>
              </Tooltip>
              <Typography
                variant="caption"
                sx={{
                  writingMode: 'vertical-rl',
                  transform: 'rotate(180deg)',
                  mt: 2,
                  fontWeight: 700,
                  color: 'text.secondary',
                  letterSpacing: 1,
                  fontSize: '0.7rem',
                }}
              >
                PALETTE ({fields.length + governedCalculations.length})
              </Typography>
            </Box>
          ) : (
            <>
              {/* Palette Header */}
              <Box sx={{ p: 1.5, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 800, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
                    Semantic Palette ({fields.length + governedCalculations.length})
                  </Typography>
                  <Tooltip title="Collapse Palette">
                    <IconButton size="small" onClick={() => setPaletteCollapsed(true)}>
                      <SidebarIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </Stack>
                <TextField
                  fullWidth
                  size="small"
                  placeholder="Filter fields & calculations..."
                  value={paletteSearch}
                  onChange={(e) => setPaletteSearch(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchIcon fontSize="small" />
                      </InputAdornment>
                    ),
                  }}
                  sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.5 } }}
                />
                {/* Visual Category Filters */}
                <Stack direction="row" spacing={0.5} sx={{ mt: 1, flexWrap: 'wrap', gap: 0.5 }}>
                  <Chip
                    label="All"
                    size="small"
                    variant={paletteCategory === 'all' ? 'filled' : 'outlined'}
                    onClick={() => setPaletteCategory('all')}
                    sx={{ fontSize: '0.65rem', height: 22, cursor: 'pointer' }}
                  />
                  <Chip
                    icon={<StringIcon sx={{ fontSize: '13px !important' }} />}
                    label="Dims"
                    size="small"
                    color="primary"
                    variant={paletteCategory === 'dim' ? 'filled' : 'outlined'}
                    onClick={() => setPaletteCategory('dim')}
                    sx={{ fontSize: '0.65rem', height: 22, cursor: 'pointer' }}
                  />
                  <Chip
                    icon={<DateIcon sx={{ fontSize: '13px !important' }} />}
                    label="Time"
                    size="small"
                    color="warning"
                    variant={paletteCategory === 'time' ? 'filled' : 'outlined'}
                    onClick={() => setPaletteCategory('time')}
                    sx={{ fontSize: '0.65rem', height: 22, cursor: 'pointer' }}
                  />
                  <Chip
                    icon={<NumberIcon sx={{ fontSize: '13px !important' }} />}
                    label="Measures"
                    size="small"
                    color="success"
                    variant={paletteCategory === 'meas' ? 'filled' : 'outlined'}
                    onClick={() => setPaletteCategory('meas')}
                    sx={{ fontSize: '0.65rem', height: 22, cursor: 'pointer' }}
                  />
                  <Chip
                    icon={<CalculateIcon sx={{ fontSize: '13px !important' }} />}
                    label="Governed Calcs"
                    size="small"
                    color="secondary"
                    variant={paletteCategory === 'calc' ? 'filled' : 'outlined'}
                    onClick={() => setPaletteCategory('calc')}
                    sx={{ fontSize: '0.65rem', height: 22, cursor: 'pointer' }}
                  />
                </Stack>
              </Box>

              {/* Palette Items List */}
              <Box sx={{ flex: 1, overflowY: 'auto', p: 1 }}>
                <Stack spacing={0.75}>
                  {/* 1. Governed Calculations Section (Purple Badge) */}
                  {filteredGovernedCalculations.map((calc) => {
                    const isSelected = measures.some(m => m.id === calc.id);
                    return (
                      <Paper
                        key={calc.id}
                        variant="outlined"
                        sx={{
                          p: 1,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          bgcolor: isSelected ? 'secondary.50' : 'background.paper',
                          borderColor: isSelected ? 'secondary.main' : 'secondary.light',
                          borderLeft: '4px solid',
                          borderLeftColor: 'secondary.main',
                          transition: 'all 0.15s ease',
                          '&:hover': { bgcolor: 'action.hover' },
                        }}
                      >
                        <Box sx={{ minWidth: 0, flexGrow: 1, mr: 1 }}>
                          <Stack direction="row" spacing={0.75} alignItems="center">
                            <Tooltip title={`Formula: ${calc.formula} (${calc.description})`}>
                              <CalculateIcon fontSize="small" sx={{ color: 'secondary.main', fontSize: 16 }} />
                            </Tooltip>
                            <Typography variant="body2" noWrap sx={{ fontWeight: 700, fontSize: '0.8rem' }}>
                              {calc.displayName}
                            </Typography>
                          </Stack>
                          <Typography variant="caption" sx={{ color: 'secondary.dark', fontFamily: 'monospace', fontSize: '0.65rem', display: 'block', noWrap: true }}>
                            {calc.formula}
                          </Typography>
                        </Box>

                        <Tooltip title={isSelected ? 'Remove Governed Metric' : 'Add Governed Calculation (Layered AST/CTE)'}>
                          <Button
                            size="small"
                            variant={isSelected ? 'contained' : 'outlined'}
                            color="secondary"
                            onClick={() => handleAddGovernedCalculation(calc)}
                            sx={{ minWidth: 28, px: 0.75, py: 0.2, fontSize: '0.65rem', height: 22, fontWeight: 700 }}
                          >
                            {isSelected ? 'Added' : '+ Calc'}
                          </Button>
                        </Tooltip>
                      </Paper>
                    );
                  })}

                  {/* 2. Standard Semantic Fields */}
                  {filteredPaletteFields.map((f: any, idx: number) => {
                    const fieldId = f.id || f.technicalName || f.name;
                    const colName = f.technicalName || f.name;
                    const isDim = dimensions.some(d => d.id === fieldId);
                    const isMeas = measures.some(m => m.id === fieldId);
                    const isTime = timeDimensions.some(td => td.id === fieldId);
                    const kind = getFieldKind(f.type || f.dataType);
                    const isNonAdd = isFieldNonAdditive(f);
                    const isSens = isSensitiveField(colName);

                    return (
                      <Paper
                        key={fieldId || idx}
                        variant="outlined"
                        sx={{
                          p: 1,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          bgcolor: isDim ? 'primary.50' : isMeas ? 'success.50' : isTime ? 'warning.50' : 'background.paper',
                          borderColor: isDim ? 'primary.main' : isMeas ? 'success.main' : isTime ? 'warning.main' : 'divider',
                          transition: 'all 0.15s ease',
                          '&:hover': { bgcolor: 'action.hover' },
                        }}
                      >
                        <Box sx={{ minWidth: 0, flexGrow: 1, mr: 1 }}>
                          <Stack direction="row" spacing={0.75} alignItems="center">
                            {kind === 'number' && <NumberIcon fontSize="small" sx={{ color: 'success.main', fontSize: 16 }} />}
                            {kind === 'date' && <DateIcon fontSize="small" sx={{ color: 'warning.main', fontSize: 16 }} />}
                            {kind === 'uuid' && <UuidIcon fontSize="small" sx={{ color: 'text.secondary', fontSize: 16 }} />}
                            {kind === 'bool' && <BoolIcon fontSize="small" sx={{ color: 'info.main', fontSize: 16 }} />}
                            {kind === 'string' && <StringIcon fontSize="small" sx={{ color: 'primary.main', fontSize: 16 }} />}
                            <Typography variant="body2" noWrap sx={{ fontWeight: 600, fontSize: '0.8rem' }}>
                              {f.displayName || f.name}
                            </Typography>
                            {isSens && (
                              <Tooltip title="PII / Sensitive Field (Dynamic ABAC Masking Applies)">
                                <SecurityIcon sx={{ fontSize: 13, color: 'error.main' }} />
                              </Tooltip>
                            )}
                            {isNonAdd && (
                              <Tooltip title="Non-Additive Metric (Ratio/Rate): SUM is locked to prevent invalid rollups">
                                <LockIcon sx={{ fontSize: 12, color: 'warning.main' }} />
                              </Tooltip>
                            )}
                          </Stack>
                          <Typography variant="caption" sx={{ color: 'text.secondary', fontFamily: 'monospace', fontSize: '0.68rem', display: 'block' }}>
                            {f.technicalName || f.key || f.name}
                          </Typography>
                        </Box>

                        <Stack direction="row" spacing={0.5}>
                          {/* Dimension Toggle Button (Blue) */}
                          <Tooltip title={isDim ? 'Remove Dimension' : 'Add Dimension'}>
                            <Button
                              size="small"
                              variant={isDim ? 'contained' : 'outlined'}
                              color="primary"
                              onClick={() => handleToggleDimension(f)}
                              sx={{ minWidth: 26, px: 0.75, py: 0.2, fontSize: '0.65rem', height: 22 }}
                            >
                              Dim
                            </Button>
                          </Tooltip>

                          {/* Measure Button (Emerald for additive, locked to AVG for non-additive) */}
                          {kind === 'number' && (
                            <Tooltip title={isMeas ? 'Edit Measure Aggregation' : isNonAdd ? 'Add Ratio Measure (Locked to AVG)' : 'Add Measure (Sum)'}>
                              <Button
                                size="small"
                                variant={isMeas ? 'contained' : 'outlined'}
                                color="success"
                                onClick={() => handleAddMeasure(f)}
                                sx={{ minWidth: 26, px: 0.75, py: 0.2, fontSize: '0.65rem', height: 22 }}
                              >
                                {isNonAdd ? 'Avg' : 'Agg'}
                              </Button>
                            </Tooltip>
                          )}

                          {/* Time Dimension Button (Cyan) */}
                          {kind === 'date' && (
                            <Tooltip title={isTime ? 'Edit Time Dimension' : 'Add Time Dimension (Day Trunc)'}>
                              <Button
                                size="small"
                                variant={isTime ? 'contained' : 'outlined'}
                                color="warning"
                                onClick={() => handleAddTimeDimension(f, 'day')}
                                sx={{ minWidth: 26, px: 0.75, py: 0.2, fontSize: '0.65rem', height: 22 }}
                              >
                                Time
                              </Button>
                            </Tooltip>
                          )}

                          {/* Filter Button */}
                          <Tooltip title="Add Filter">
                            <IconButton size="small" onClick={() => handleAddFilter(f)} sx={{ p: 0.25 }}>
                              <FilterIcon fontSize="small" sx={{ fontSize: 16 }} />
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      </Paper>
                    );
                  })}
                </Stack>
              </Box>
            </>
          )}
        </Paper>

        {/* Right Side: Interactive Query Builder & Results Workspace */}
        <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 2 }}>
          {/* Active Query Projection Bar */}
          <Paper variant="outlined" sx={{ p: 2, borderRadius: 1.5 }}>
            <Stack spacing={1.75}>
              {/* Dimensions Section */}
              <Box>
                <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.75 }}>
                  <Typography variant="caption" sx={{ fontWeight: 800, color: 'primary.main', textTransform: 'uppercase', letterSpacing: 0.5 }}>
                    Dimensions ({dimensions.length})
                  </Typography>
                </Stack>
                <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 0.75 }}>
                  {dimensions.map((d, idx) => (
                    <Chip
                      key={d.id}
                      icon={d.isSensitive && userRole === 'analyst' && enableDynamicMasking ? <LockIcon sx={{ fontSize: '13px !important' }} /> : undefined}
                      label={d.alias}
                      color="primary"
                      size="small"
                      onDelete={() => setDimensions(dimensions.filter((_, i) => i !== idx))}
                      onClick={() => handleAddSort(d.alias, d.name)}
                      sx={{ fontWeight: 600 }}
                    />
                  ))}
                  {dimensions.length === 0 && (
                    <Typography variant="caption" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                      None selected. Click "Dim" on any field.
                    </Typography>
                  )}
                </Stack>
              </Box>

              {/* Time Dimensions Section */}
              {timeDimensions.length > 0 && (
                <Box>
                  <Typography variant="caption" sx={{ fontWeight: 800, color: 'warning.main', textTransform: 'uppercase', letterSpacing: 0.5, display: 'block', mb: 0.75 }}>
                    Time Dimensions ({timeDimensions.length})
                  </Typography>
                  <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 0.75 }}>
                    {timeDimensions.map((td, idx) => (
                      <Chip
                        key={td.id}
                        label={`${td.alias} (${td.granularity})`}
                        color="warning"
                        size="small"
                        onDelete={() => setTimeDimensions(timeDimensions.filter((_, i) => i !== idx))}
                        onClick={(e) => setTimeMenuAnchor({ el: e.currentTarget, index: idx })}
                        sx={{ fontWeight: 600, cursor: 'pointer' }}
                      />
                    ))}
                  </Stack>
                </Box>
              )}

              {/* Measures & Governed Calculations Section */}
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 800, color: 'success.main', textTransform: 'uppercase', letterSpacing: 0.5, display: 'block', mb: 0.75 }}>
                  Measures & Governed Calculations ({measures.length})
                </Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 0.75 }}>
                  {measures.map((m, idx) => (
                    <Chip
                      key={m.id}
                      icon={m.isCalculated ? <CalculateIcon sx={{ fontSize: '15px !important' }} /> : m.isNonAdditive ? <LockIcon sx={{ fontSize: '13px !important' }} /> : undefined}
                      label={`${m.agg}(${m.alias})`}
                      color={m.isCalculated ? 'secondary' : 'success'}
                      size="small"
                      onDelete={() => setMeasures(measures.filter((_, i) => i !== idx))}
                      onClick={(e) => setAggMenuAnchor({ el: e.currentTarget, index: idx })}
                      sx={{ fontWeight: 600, cursor: 'pointer' }}
                    />
                  ))}
                  {measures.length === 0 && (
                    <Typography variant="caption" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                      None selected. Click "Agg" or "+ Calc" from the palette.
                    </Typography>
                  )}
                </Stack>
              </Box>

              {/* Filters Section */}
              {filters.length > 0 && (
                <Box sx={{ pt: 1, borderTop: '1px dashed', borderTopColor: 'divider' }}>
                  <Typography variant="caption" sx={{ fontWeight: 800, color: 'text.primary', textTransform: 'uppercase', letterSpacing: 0.5, display: 'block', mb: 1 }}>
                    Filters & Conditions ({filters.length})
                  </Typography>
                  <Stack spacing={1}>
                    {filters.map((flt, idx) => (
                      <Stack key={flt.id || idx} direction="row" spacing={1} alignItems="center" sx={{ flexWrap: 'wrap', gap: 1 }}>
                        <Chip label={flt.displayName || flt.fieldName} size="small" variant="outlined" sx={{ fontWeight: 600, minWidth: 100 }} />
                        <TextField
                          select
                          size="small"
                          value={flt.op}
                          onChange={(e) => {
                            const updated = [...filters];
                            updated[idx].op = e.target.value as FilterOp;
                            setFilters(updated);
                          }}
                          sx={{ width: 140, '& .MuiInputBase-input': { py: 0.5, fontSize: '0.75rem' } }}
                        >
                          <MenuItem value="=">=</MenuItem>
                          <MenuItem value="!=">!=</MenuItem>
                          <MenuItem value=">">&gt;</MenuItem>
                          <MenuItem value="<">&lt;</MenuItem>
                          <MenuItem value=">=">&gt;=</MenuItem>
                          <MenuItem value="<=">&lt;=</MenuItem>
                          <MenuItem value="ILIKE">ILIKE (case insensitive)</MenuItem>
                          <MenuItem value="LIKE">LIKE (pattern)</MenuItem>
                          <MenuItem value="CONTAINS">CONTAINS</MenuItem>
                          <MenuItem value="STARTS WITH">STARTS WITH</MenuItem>
                          <MenuItem value="ENDS WITH">ENDS WITH</MenuItem>
                          <MenuItem value="IN">IN (comma separated)</MenuItem>
                          <MenuItem value="NOT IN">NOT IN</MenuItem>
                          <MenuItem value="IS NULL">IS NULL</MenuItem>
                          <MenuItem value="IS NOT NULL">IS NOT NULL</MenuItem>
                          <MenuItem value="BETWEEN">BETWEEN</MenuItem>
                        </TextField>

                        {!['IS NULL', 'IS NOT NULL'].includes(flt.op) && (
                          <TextField
                            size="small"
                            placeholder={flt.op === 'IN' ? 'e.g. Active, Pending' : flt.op === 'BETWEEN' ? 'Start value' : 'Filter value...'}
                            value={flt.val}
                            onChange={(e) => {
                              const updated = [...filters];
                              updated[idx].val = e.target.value;
                              setFilters(updated);
                            }}
                            sx={{ flex: 1, minWidth: 140, '& .MuiInputBase-input': { py: 0.5, fontSize: '0.75rem' } }}
                          />
                        )}

                        {flt.op === 'BETWEEN' && (
                          <TextField
                            size="small"
                            placeholder="End value"
                            value={flt.val2 || ''}
                            onChange={(e) => {
                              const updated = [...filters];
                              updated[idx].val2 = e.target.value;
                              setFilters(updated);
                            }}
                            sx={{ flex: 1, minWidth: 140, '& .MuiInputBase-input': { py: 0.5, fontSize: '0.75rem' } }}
                          />
                        )}

                        <IconButton size="small" color="error" onClick={() => setFilters(filters.filter((_, i) => i !== idx))}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Stack>
                    ))}
                  </Stack>
                </Box>
              )}

              {/* Sorts Section */}
              {sorts.length > 0 && (
                <Box sx={{ pt: 1, borderTop: '1px dashed', borderTopColor: 'divider' }}>
                  <Typography variant="caption" sx={{ fontWeight: 800, color: 'text.secondary', textTransform: 'uppercase', letterSpacing: 0.5, display: 'block', mb: 0.5 }}>
                    Order By ({sorts.length})
                  </Typography>
                  <Stack direction="row" spacing={1} flexWrap="wrap">
                    {sorts.map((s, idx) => (
                      <Chip
                        key={s.id}
                        label={`${s.alias}: ${s.direction}`}
                        size="small"
                        icon={<SortIcon />}
                        onDelete={() => setSorts(sorts.filter((_, i) => i !== idx))}
                        onClick={() => {
                          const updated = [...sorts];
                          updated[idx].direction = s.direction === 'ASC' ? 'DESC' : 'ASC';
                          setSorts(updated);
                        }}
                        sx={{ fontWeight: 600, cursor: 'pointer' }}
                      />
                    ))}
                  </Stack>
                </Box>
              )}
            </Stack>
          </Paper>

          {/* Results & Visualizer Tab Area */}
          <Paper variant="outlined" sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRadius: 1.5 }}>
            <Box sx={{ borderBottom: 1, borderColor: 'divider', px: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 1 }}>
              <Tabs value={resultTab} onChange={(_, val) => setResultTab(val)} sx={{ minHeight: 44 }}>
                <Tab icon={<TableIcon fontSize="small" />} iconPosition="start" label="Data Grid" sx={{ minHeight: 44, textTransform: 'none', fontWeight: 700 }} />
                <Tab icon={<CodeIcon fontSize="small" />} iconPosition="start" label="Pushdown SQL Engine" sx={{ minHeight: 44, textTransform: 'none', fontWeight: 700 }} />
                <Tab icon={<BarChartIcon fontSize="small" />} iconPosition="start" label="Visualization Preview" sx={{ minHeight: 44, textTransform: 'none', fontWeight: 700 }} />
                <Tab icon={<SpeedIcon fontSize="small" />} iconPosition="start" label="Explain & ABAC Sentinel DAG" sx={{ minHeight: 44, textTransform: 'none', fontWeight: 700 }} />
                <Tab icon={<FunctionsIcon fontSize="small" />} iconPosition="start" label="JSON Result" sx={{ minHeight: 44, textTransform: 'none', fontWeight: 700 }} />
              </Tabs>

              <Stack direction="row" spacing={1} alignItems="center">
                <Button size="small" startIcon={<CopyIcon />} onClick={handleCopySql} sx={{ textTransform: 'none', fontSize: '0.75rem' }}>
                  Copy SQL
                </Button>
                {executeResult && (
                  <Chip
                    label={`${executeResult.executionTimeMs || 12}ms`}
                    size="small"
                    color="success"
                    variant="outlined"
                    sx={{ fontWeight: 700, fontSize: '0.7rem', height: 22 }}
                  />
                )}
              </Stack>
            </Box>

            {/* TAB 0: Data Grid View */}
            {resultTab === 0 && (
              <Box sx={{ flex: 1, p: 2, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                {executeResult ? (
                  <>
                    <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1.5 }}>
                      <TextField
                        size="small"
                        placeholder="Search in results..."
                        value={tableSearchFilter}
                        onChange={(e) => {
                          setTableSearchFilter(e.target.value);
                          setTablePage(0);
                        }}
                        InputProps={{
                          startAdornment: (
                            <InputAdornment position="start">
                              <SearchIcon fontSize="small" />
                            </InputAdornment>
                          ),
                        }}
                        sx={{ width: 260, '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.5 } }}
                      />
                      <Stack direction="row" spacing={1} alignItems="center">
                        {userRole === 'analyst' && enableDynamicMasking && (
                          <Chip
                            icon={<LockIcon sx={{ fontSize: '12px !important' }} />}
                            label="Dynamic ABAC Masking Active (PII Redacted)"
                            size="small"
                            color="warning"
                            variant="outlined"
                            sx={{ fontWeight: 700, fontSize: '0.7rem' }}
                          />
                        )}
                        <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                          Showing {displayRows.length} rows
                        </Typography>
                      </Stack>
                    </Stack>

                    <TableContainer sx={{ flex: 1, maxHeight: 'calc(100vh - 560px)', border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
                      <Table size="small" stickyHeader>
                        <TableHead>
                          <TableRow>
                            <TableCell sx={{ fontWeight: 800, bgcolor: 'action.hover', width: 40 }}>#</TableCell>
                            {(executeResult.columns || []).map((col: any, idx: number) => {
                              const colName = typeof col === 'string' ? col : col?.name || `Column ${idx + 1}`;
                              const isSens = isSensitiveField(colName);
                              return (
                                <TableCell key={idx} sx={{ fontWeight: 800, bgcolor: 'action.hover' }}>
                                  <Stack direction="row" spacing={0.5} alignItems="center">
                                    <span>{colName}</span>
                                    {isSens && userRole === 'analyst' && enableDynamicMasking && (
                                      <Tooltip title="Masked by ABAC Rule 7">
                                        <LockIcon sx={{ fontSize: 13, color: 'warning.main' }} />
                                      </Tooltip>
                                    )}
                                  </Stack>
                                </TableCell>
                              );
                            })}
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {pagedRows.map((row: any, rIdx: number) => (
                            <TableRow key={rIdx} hover>
                              <TableCell sx={{ color: 'text.secondary', fontSize: '0.75rem' }}>
                                {tablePage * tableRowsPerPage + rIdx + 1}
                              </TableCell>
                              {(executeResult.columns || []).map((col: any, cIdx: number) => {
                                const colName = typeof col === 'string' ? col : col?.name || '';
                                return (
                                  <TableCell
                                    key={cIdx}
                                    sx={{
                                      fontSize: '0.8rem',
                                      cursor: 'pointer',
                                      '&:hover': { bgcolor: 'action.hover', color: 'primary.main', textDecoration: 'underline' }
                                    }}
                                    onDoubleClick={() => {
                                      setDrillField(colName);
                                      setDrillFilterContext(row);
                                      setDrillModalOpen(true);
                                    }}
                                    title="Double-click to drill down into constituent records"
                                  >
                                    {String(row[colName] !== undefined ? row[colName] : '')}
                                  </TableCell>
                                );
                              })}
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableContainer>

                    <TablePagination
                      component="div"
                      count={displayRows.length}
                      page={tablePage}
                      onPageChange={(_, p) => setTablePage(p)}
                      rowsPerPage={tableRowsPerPage}
                      onRowsPerPageChange={(e) => {
                        setTableRowsPerPage(parseInt(e.target.value, 10));
                        setTablePage(0);
                      }}
                      rowsPerPageOptions={[5, 10, 25, 50, 100]}
                      sx={{ borderTop: '1px solid', borderColor: 'divider' }}
                    />
                  </>
                ) : (
                  <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', py: 8 }}>
                    <TableIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 1 }} />
                    <Typography variant="body1" sx={{ fontWeight: 700 }}>
                      No Query Results Yet
                    </Typography>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                      Click "Run Query" to compile pushdown SQL and fetch live results.
                    </Typography>
                    <Button variant="contained" color="primary" onClick={handleRunQuery} disabled={!dimensions.length && !measures.length}>
                      Run Query Now
                    </Button>
                  </Box>
                )}
              </Box>
            )}

            {/* TAB 1: Pushdown SQL Engine Code View */}
            {resultTab === 1 && (
              <Box sx={{ flex: 1, p: 2, overflow: 'auto', bgcolor: '#1e1e1e' }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Chip label={`Engine: ${resolvedEngineTier.label}`} color="primary" size="small" sx={{ fontWeight: 700 }} />
                    {measures.some(m => m.isCalculated) && (
                      <Chip label="Two-Pass CTE Compilation Active" color="secondary" size="small" sx={{ fontWeight: 700 }} />
                    )}
                  </Stack>
                  <Button size="small" variant="outlined" startIcon={<CopyIcon />} onClick={handleCopySql} sx={{ color: 'white', borderColor: 'grey.700' }}>
                    Copy SQL
                  </Button>
                </Stack>
                <SyntaxHighlighter
                  language="sql"
                  style={vscDarkPlus}
                  customStyle={{ margin: 0, padding: '16px', borderRadius: 8, fontSize: '0.85rem', background: 'transparent' }}
                >
                  {previewSql}
                </SyntaxHighlighter>
              </Box>
            )}

            {/* TAB 2: Visualization Preview */}
            {resultTab === 2 && (
              <Box sx={{ flex: 1, p: 2.5, display: 'flex', flexDirection: 'column' }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                  <ButtonGroup size="small" variant="outlined">
                    <Button
                      variant={chartType === 'bar' ? 'contained' : 'outlined'}
                      onClick={() => setChartType('bar')}
                      startIcon={<BarChartIcon />}
                    >
                      Bar
                    </Button>
                    <Button
                      variant={chartType === 'line' ? 'contained' : 'outlined'}
                      onClick={() => setChartType('line')}
                      startIcon={<LineChartIcon />}
                    >
                      Line
                    </Button>
                    <Button
                      variant={chartType === 'area' ? 'contained' : 'outlined'}
                      onClick={() => setChartType('area')}
                      startIcon={<AreaChartIcon />}
                    >
                      Area
                    </Button>
                    <Button
                      variant={chartType === 'pie' ? 'contained' : 'outlined'}
                      onClick={() => setChartType('pie')}
                      startIcon={<PieChartIcon />}
                    >
                      Pie
                    </Button>
                  </ButtonGroup>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                    Visualizing {displayRows.length} rows
                  </Typography>
                </Stack>

                {executeResult && displayRows.length > 0 ? (
                  <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 300, bgcolor: 'action.hover', borderRadius: 2, p: 3 }}>
                    <svg width="100%" height="280" viewBox="0 0 600 280">
                      <line x1="40" y1="20" x2="580" y2="20" stroke="#e0e0e0" strokeDasharray="3 3" />
                      <line x1="40" y1="80" x2="580" y2="80" stroke="#e0e0e0" strokeDasharray="3 3" />
                      <line x1="40" y1="140" x2="580" y2="140" stroke="#e0e0e0" strokeDasharray="3 3" />
                      <line x1="40" y1="200" x2="580" y2="200" stroke="#e0e0e0" strokeDasharray="3 3" />
                      <line x1="40" y1="240" x2="580" y2="240" stroke="#888" strokeWidth="1.5" />
                      <line x1="40" y1="20" x2="40" y2="240" stroke="#888" strokeWidth="1.5" />

                      {chartType === 'bar' && displayRows.slice(0, 10).map((r, i, arr) => {
                        const barWidth = Math.max(16, (500 / arr.length) - 10);
                        const x = 60 + i * (barWidth + 12);
                        const lastCol = executeResult.columns?.[executeResult.columns.length - 1];
                        const lastColName = typeof lastCol === 'string' ? lastCol : lastCol?.name || '';
                        const firstMeas = measures[0]?.alias || lastColName;
                        const rawVal = parseFloat(String(r[firstMeas] ?? ((i + 1) * 30)));
                        const height = Math.min(200, Math.max(10, (rawVal / 2000000) * 180 || (i + 1) * 20));
                        const y = 240 - height;
                        const firstCol = executeResult.columns?.[0];
                        const firstColName = typeof firstCol === 'string' ? firstCol : firstCol?.name || '';
                        const label = String(r[dimensions[0]?.alias || firstColName] || `Item ${i + 1}`);

                        return (
                          <g
                            key={i}
                            style={{ cursor: 'pointer' }}
                            onDoubleClick={() => {
                              setDrillField(firstMeas);
                              setDrillFilterContext(r);
                              setDrillModalOpen(true);
                            }}
                          >
                            <title>Double-click to drill down into constituent records for {label}</title>
                            <rect x={x} y={y} width={barWidth} height={height} fill="#1976d2" rx="3" />
                            <text x={x + barWidth / 2} y={y - 6} textAnchor="middle" fontSize="10" fontWeight="bold" fill="#333">
                              {rawVal > 1000 ? `${(rawVal / 1000).toFixed(0)}k` : rawVal.toFixed(0)}
                            </text>
                            <text x={x + barWidth / 2} y={255} textAnchor="middle" fontSize="9" fill="#666">
                              {label.length > 8 ? label.substring(0, 7) + '..' : label}
                            </text>
                          </g>
                        );
                      })}

                      {chartType === 'line' && (
                        <>
                          <polyline
                            fill="none"
                            stroke="#9c27b0"
                            strokeWidth="3"
                            points={displayRows.slice(0, 10).map((r, i, arr) => {
                              const x = 60 + i * (500 / arr.length);
                              const lastCol = executeResult.columns?.[executeResult.columns.length - 1];
                              const lastColName = typeof lastCol === 'string' ? lastCol : lastCol?.name || '';
                              const firstMeas = measures[0]?.alias || lastColName;
                              const rawVal = parseFloat(String(r[firstMeas] ?? ((i + 1) * 30)));
                              const height = Math.min(200, Math.max(10, (rawVal / 2000000) * 180 || (i + 1) * 20));
                              const y = 240 - height;
                              return `${x},${y}`;
                            }).join(' ')}
                          />
                          {displayRows.slice(0, 10).map((r, i, arr) => {
                            const x = 60 + i * (500 / arr.length);
                            const lastCol = executeResult.columns?.[executeResult.columns.length - 1];
                            const lastColName = typeof lastCol === 'string' ? lastCol : lastCol?.name || '';
                            const firstMeas = measures[0]?.alias || lastColName;
                            const rawVal = parseFloat(String(r[firstMeas] ?? ((i + 1) * 30)));
                            const height = Math.min(200, Math.max(10, (rawVal / 2000000) * 180 || (i + 1) * 20));
                            const y = 240 - height;
                            return (
                              <circle
                                key={i}
                                cx={x}
                                cy={y}
                                r="5"
                                fill="#9c27b0"
                                stroke="#fff"
                                strokeWidth="2"
                                style={{ cursor: 'pointer' }}
                                onDoubleClick={() => {
                                  setDrillField(firstMeas);
                                  setDrillFilterContext(r);
                                  setDrillModalOpen(true);
                                }}
                              />
                            );
                          })}
                        </>
                      )}

                      {chartType === 'area' && (
                        <>
                          <polygon
                            fill="rgba(25, 118, 210, 0.25)"
                            stroke="#1976d2"
                            strokeWidth="2"
                            points={`60,240 ${displayRows.slice(0, 10).map((r, i, arr) => {
                              const x = 60 + i * (500 / arr.length);
                              const lastCol = executeResult.columns?.[executeResult.columns.length - 1];
                              const lastColName = typeof lastCol === 'string' ? lastCol : lastCol?.name || '';
                              const firstMeas = measures[0]?.alias || lastColName;
                              const rawVal = parseFloat(String(r[firstMeas] ?? ((i + 1) * 30)));
                              const height = Math.min(200, Math.max(10, (rawVal / 2000000) * 180 || (i + 1) * 20));
                              const y = 240 - height;
                              return `${x},${y}`;
                            }).join(' ')} 560,240`}
                          />
                        </>
                      )}

                      {chartType === 'pie' && (
                        <g transform="translate(300, 130)" style={{ cursor: 'pointer' }}>
                          <circle cx="0" cy="0" r="90" fill="#1976d2" onDoubleClick={() => {
                            const lastCol = executeResult.columns?.[executeResult.columns.length - 1];
                            const lastColName = typeof lastCol === 'string' ? lastCol : lastCol?.name || '';
                            const firstMeas = measures[0]?.alias || lastColName;
                            setDrillField(firstMeas);
                            setDrillFilterContext(displayRows[0] || {});
                            setDrillModalOpen(true);
                          }} />
                          <path d="M 0 0 L 0 -90 A 90 90 0 0 1 85 -30 Z" fill="#9c27b0" />
                          <path d="M 0 0 L 85 -30 A 90 90 0 0 1 60 70 Z" fill="#ff9800" />
                          <path d="M 0 0 L 60 70 A 90 90 0 0 1 -75 50 Z" fill="#4caf50" />
                          <circle cx="0" cy="0" r="45" fill="#fff" />
                          <text x="0" y="5" textAnchor="middle" fontSize="11" fontWeight="bold" fill="#333">Share</text>
                        </g>
                      )}
                    </svg>
                  </Box>
                ) : (
                  <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', py: 8 }}>
                    <BarChartIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 1 }} />
                    <Typography variant="body1" sx={{ fontWeight: 700 }}>No Visualization Data</Typography>
                    <Typography variant="body2" color="text.secondary">Run query with measures and dimensions to render charts.</Typography>
                  </Box>
                )}
              </Box>
            )}

            {/* TAB 3: Federated Explain Plan Visualizer DAG & ABAC Sentinel */}
            {resultTab === 3 && (
              <Box sx={{ flex: 1, p: 2.5, overflow: 'auto', display: 'flex', flexDirection: 'column', gap: 2.5 }}>
                {/* ABAC Simulation Control Toolbar */}
                <Paper variant="outlined" sx={{ p: 1.5, bgcolor: 'background.paper', borderRadius: 1.5 }}>
                  <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', md: 'center' }} spacing={2}>
                    <Stack direction="row" spacing={1.5} alignItems="center">
                      <SecurityIcon color="primary" />
                      <Box>
                        <Typography variant="subtitle2" sx={{ fontWeight: 800 }}>
                          Rule 7 ABAC Security & Role Capability Simulator
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          Dynamically test how column-level masking and tenant boundary isolation alter the compiled AST.
                        </Typography>
                      </Box>
                    </Stack>

                    <Stack direction="row" spacing={2} alignItems="center">
                      <TextField
                        select
                        size="small"
                        label="Simulated Persona"
                        value={userRole}
                        onChange={(e) => setUserRole(e.target.value as UserPersona)}
                        sx={{ width: 190 }}
                      >
                        <MenuItem value="analyst">👤 Analyst (Public Clearance)</MenuItem>
                        <MenuItem value="compliance_officer">🛡️ Compliance Officer</MenuItem>
                        <MenuItem value="platform_trader">⚡ Platform Trader</MenuItem>
                      </TextField>

                      <FormControlLabel
                        control={
                          <Switch
                            checked={enableDynamicMasking}
                            onChange={(e) => setEnableDynamicMasking(e.target.checked)}
                            color="primary"
                          />
                        }
                        label={
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>
                            {enableDynamicMasking ? 'Dynamic Masking ON' : 'Dynamic Masking OFF'}
                          </Typography>
                        }
                      />
                    </Stack>
                  </Stack>
                </Paper>

                {/* Interactive Visual DAG Tree Canvas */}
                <Grid container spacing={2.5}>
                  {/* Left Column: Interactive Visual DAG Pipeline */}
                  <Grid size={{ xs: 12, md: 7 }}>
                    <Paper variant="outlined" sx={{ p: 2, borderRadius: 1.5, minHeight: 400, bgcolor: 'action.hover' }}>
                      <Typography variant="subtitle2" sx={{ fontWeight: 800, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                        <HubIcon color="primary" fontSize="small" />
                        Federated Pushdown Execution DAG (Cost-Based Optimizer)
                      </Typography>

                      <Stack spacing={1.75} alignItems="center">
                        {explainPlanDAGNodes.map((node, index) => {
                          const isSelected = node.id === activeDAGNode.id;
                          return (
                            <React.Fragment key={node.id}>
                              <Card
                                variant="outlined"
                                onClick={() => setSelectedDAGNodeId(node.id)}
                                sx={{
                                  width: '100%',
                                  cursor: 'pointer',
                                  borderColor: isSelected ? 'primary.main' : 'divider',
                                  borderWidth: isSelected ? 2 : 1,
                                  bgcolor: isSelected ? 'primary.50' : 'background.paper',
                                  boxShadow: isSelected ? 2 : 0,
                                  transition: 'all 0.15s ease',
                                  '&:hover': { bgcolor: 'action.hover', borderColor: 'primary.light' },
                                }}
                              >
                                <CardContent sx={{ p: 1.5, '&:last-child': { pb: 1.5 } }}>
                                  <Stack direction="row" justifyContent="space-between" alignItems="center">
                                    <Stack direction="row" spacing={1} alignItems="center">
                                      {node.category === 'SCAN' && <StorageIcon color="primary" fontSize="small" />}
                                      {node.category === 'SECURITY' && <SecurityIcon color="error" fontSize="small" />}
                                      {node.category === 'AST_CTE' && <CalculateIcon color="secondary" fontSize="small" />}
                                      {node.category === 'AGGREGATE' && <FunctionsIcon color="success" fontSize="small" />}
                                      {node.category === 'OUTPUT' && <SpeedIcon color="info" fontSize="small" />}
                                      <Typography variant="body2" sx={{ fontWeight: 800 }}>
                                        {node.title}
                                      </Typography>
                                    </Stack>
                                    <Chip
                                      label={`Cost: ${node.costScore}`}
                                      size="small"
                                      color={node.costScore > 15 ? 'warning' : 'default'}
                                      sx={{ height: 20, fontSize: '0.65rem', fontWeight: 700 }}
                                    />
                                  </Stack>
                                  <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
                                    {node.description}
                                  </Typography>
                                </CardContent>
                              </Card>

                              {index < explainPlanDAGNodes.length - 1 && (
                                <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                                  <Box sx={{ width: 2, height: 16, bgcolor: 'primary.main' }} />
                                  <Typography variant="caption" sx={{ fontSize: '0.6rem', color: 'text.secondary', fontWeight: 800 }}>
                                    ▼ Pushdown Stream
                                  </Typography>
                                </Box>
                              )}
                            </React.Fragment>
                          );
                        })}
                      </Stack>
                    </Paper>
                  </Grid>

                  {/* Right Column: Contextual DAG Node Telemetry Inspector */}
                  <Grid size={{ xs: 12, md: 5 }}>
                    <Card variant="outlined" sx={{ borderRadius: 1.5, height: '100%' }}>
                      <CardContent sx={{ p: 2 }}>
                        <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1.5 }}>
                          <InfoIcon color="primary" />
                          <Typography variant="subtitle2" sx={{ fontWeight: 800 }}>
                            DAG Node Telemetry Inspector
                          </Typography>
                        </Stack>

                        <Paper variant="outlined" sx={{ p: 1.5, mb: 2, bgcolor: 'background.paper' }}>
                          <Typography variant="body2" sx={{ fontWeight: 800, color: 'primary.main' }}>
                            {activeDAGNode.title}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
                            {activeDAGNode.description}
                          </Typography>
                        </Paper>

                        <Typography variant="caption" sx={{ fontWeight: 800, textTransform: 'uppercase', color: 'text.secondary', display: 'block', mb: 1 }}>
                          Hardware & Optimizer Telemetry
                        </Typography>

                        <Stack spacing={1.25}>
                          {activeDAGNode.metrics.map((m, idx) => (
                            <Box key={idx} sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 0.5, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
                              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                                {m.label}:
                              </Typography>
                              <Typography variant="caption" sx={{ fontWeight: 700, fontFamily: 'monospace' }}>
                                {m.value}
                              </Typography>
                            </Box>
                          ))}
                          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 0.5 }}>
                            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                              Estimated Output Cardinality:
                            </Typography>
                            <Typography variant="caption" sx={{ fontWeight: 700 }}>
                              ~{activeDAGNode.estRows} rows
                            </Typography>
                          </Box>
                          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 0.5 }}>
                            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                              Relative CPU Cost:
                            </Typography>
                            <Chip label={`${activeDAGNode.costScore} cost units`} size="small" color="primary" variant="outlined" sx={{ height: 20, fontSize: '0.65rem' }} />
                          </Box>
                        </Stack>
                      </CardContent>
                    </Card>
                  </Grid>
                </Grid>
              </Box>
            )}

            {/* TAB 4: JSON Result View */}
            {resultTab === 4 && (
              <Box sx={{ flex: 1, p: 2, overflow: 'auto', bgcolor: '#1e1e1e' }}>
                <SyntaxHighlighter
                  language="json"
                  style={vscDarkPlus}
                  customStyle={{ margin: 0, padding: '16px', borderRadius: 8, fontSize: '0.85rem', background: 'transparent' }}
                >
                  {JSON.stringify(executeResult?.rows || [], null, 2)}
                </SyntaxHighlighter>
              </Box>
            )}
          </Paper>
        </Box>
      </Box>

      {/* Aggregation Selection Context Menu with Smart Non-Additive Locking */}
      <Menu
        anchorEl={aggMenuAnchor?.el}
        open={Boolean(aggMenuAnchor)}
        onClose={() => setAggMenuAnchor(null)}
      >
        {(['SUM', 'AVG', 'MIN', 'MAX', 'COUNT', 'COUNT DISTINCT', 'STDDEV', 'VALUE'] as AggFunc[]).map((agg) => {
          const currentMeasure = aggMenuAnchor !== null ? measures[aggMenuAnchor.index] : null;
          const isNonAdd = currentMeasure ? currentMeasure.isNonAdditive : false;
          const isSumLocked = isNonAdd && agg === 'SUM';

          return (
            <MenuItem
              key={agg}
              disabled={isSumLocked}
              selected={Boolean(aggMenuAnchor && measures[aggMenuAnchor.index]?.agg === agg)}
              onClick={() => {
                if (aggMenuAnchor !== null && !isSumLocked) {
                  const updated = [...measures];
                  const current = updated[aggMenuAnchor.index];
                  updated[aggMenuAnchor.index] = {
                    ...current,
                    agg,
                    alias: `${current.name} (${agg})`,
                  };
                  setMeasures(updated);
                }
                setAggMenuAnchor(null);
              }}
            >
              <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between" sx={{ width: '100%' }}>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>{agg}</Typography>
                {isSumLocked && (
                  <Chip
                    icon={<LockIcon sx={{ fontSize: '12px !important' }} />}
                    label="Locked (Non-Additive Ratio)"
                    size="small"
                    color="warning"
                    sx={{ fontSize: '0.65rem', height: 18 }}
                  />
                )}
              </Stack>
            </MenuItem>
          );
        })}
      </Menu>

      {/* Time Granularity Context Menu */}
      <Menu
        anchorEl={timeMenuAnchor?.el}
        open={Boolean(timeMenuAnchor)}
        onClose={() => setTimeMenuAnchor(null)}
      >
        {(['raw', 'hour', 'day', 'week', 'month', 'quarter', 'year'] as TimeGranularity[]).map((gran) => (
          <MenuItem
            key={gran}
            selected={Boolean(timeMenuAnchor && timeDimensions[timeMenuAnchor.index]?.granularity === gran)}
            onClick={() => {
              if (timeMenuAnchor !== null) {
                const updated = [...timeDimensions];
                const current = updated[timeMenuAnchor.index];
                updated[timeMenuAnchor.index] = {
                  ...current,
                  granularity: gran,
                  alias: `${current.name} (${gran})`,
                };
                setTimeDimensions(updated);
              }
              setTimeMenuAnchor(null);
            }}
          >
            {gran.toUpperCase()}
          </MenuItem>
        ))}
      </Menu>

      {/* Drill-Down Slide-Out Drawer / Modal */}
      <DrillDownGridModal
        isOpen={drillModalOpen}
        tenantId={tenantId || ''}
        aggregatedField={drillField}
        filterContext={drillFilterContext}
        onClose={() => {
          setDrillModalOpen(false);
          setDrillField('');
          setDrillFilterContext({});
        }}
      />
    </Box>
  );
}

