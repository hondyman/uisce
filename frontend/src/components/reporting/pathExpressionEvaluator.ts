/**
 * Expression evaluator and variable registry for dynamic report paths, filenames, and multi-tenant bursting routing.
 */

export interface PathEvaluationContext {
  tenant_code?: string;
  tenant_name?: string;
  tenant_id?: string;
  is_core?: boolean;
  gold_copy?: boolean;
  report_id?: string;
  report_name?: string;
  report_code?: string;
  executed_by?: string;

  // Slicing & Burst
  slice_key?: string;
  burst_key?: string;
  slice_name?: string;
  client_id?: string;
  client_name?: string;
  slice_dim?: string;
  seq?: string | number;
  seq_raw?: number;
  total_slices?: number;
  batch_id?: string;

  // Date & Calendar
  date?: string;
  today?: string;
  year?: string | number;
  YYYY?: string | number;
  month?: string | number;
  MM?: string | number;
  month_name?: string;
  day?: string | number;
  DD?: string | number;
  quarter?: string;
  period?: string;
  timestamp?: string;
  effective_date?: string;

  [key: string]: any;
}

export interface VariableDef {
  variable: string;
  tokenAlt?: string;
  category: 'system' | 'burst' | 'datetime' | 'functions';
  displayName: string;
  description: string;
  example: string;
}

export const SYSTEM_VARIABLES: VariableDef[] = [
  // System / Multi-tenant variables
  {
    variable: '@tenant_code',
    tokenAlt: '{tenant}',
    category: 'system',
    displayName: 'Tenant Code / Slug',
    description: 'Dynamic tenant slug for folder routing (e.g., acme_wealth, blackrock_wm)',
    example: 'acme_wealth',
  },
  {
    variable: '@tenant_name',
    tokenAlt: '{tenant_name}',
    category: 'system',
    displayName: 'Tenant Display Name',
    description: 'Full legal name of the tenant organization',
    example: 'Acme Wealth Partners LLC',
  },
  {
    variable: '@tenant_id',
    tokenAlt: '{tenant_id}',
    category: 'system',
    displayName: 'Tenant UUID',
    description: 'Unique database identifier of the tenant',
    example: '8f3a9e22-1d54-4f9e-a612-88231901df42',
  },
  {
    variable: '@is_core',
    tokenAlt: '{is_core}',
    category: 'system',
    displayName: 'Is Core / Gold Copy Flag',
    description: 'Boolean flag: true if running under the master core tenant, false if client tenant',
    example: 'false',
  },
  {
    variable: '@report_name',
    tokenAlt: '{report_name}',
    category: 'system',
    displayName: 'Report Title',
    description: 'Original human-readable report title',
    example: 'Daily Institutional Client Valuation',
  },
  {
    variable: '@report_code',
    tokenAlt: '{report_code}',
    category: 'system',
    displayName: 'Normalized Report Code',
    description: 'Safe sanitized snake_case report code',
    example: 'daily_institutional_client_valuation',
  },
  {
    variable: '@report_id',
    tokenAlt: '{report_id}',
    category: 'system',
    displayName: 'Report Identifier',
    description: 'System report key or UUID',
    example: 'rep-custom-001',
  },
  {
    variable: '@executed_by',
    tokenAlt: '{executed_by}',
    category: 'system',
    displayName: 'Executing User / Role',
    description: 'User email or automation daemon initiating the execution',
    example: 'admin@uuisce.internal',
  },

  // Slicing & Burst variables
  {
    variable: '@slice_key',
    tokenAlt: '{client_id}',
    category: 'burst',
    displayName: 'Slicing Key / Client Code',
    description: 'Current partition entity value (e.g. client_id, account_id)',
    example: 'client-001',
  },
  {
    variable: '@slice_name',
    tokenAlt: '{client_name}',
    category: 'burst',
    displayName: 'Sliced Entity Name',
    description: 'Resolved display name of the partition entity',
    example: 'Apex Global Alpha Fund',
  },
  {
    variable: '@slice_dim',
    tokenAlt: '{slice_dim}',
    category: 'burst',
    displayName: 'Slicing Dimension Name',
    description: 'Field name used to slice (e.g. client_id, account_id)',
    example: 'client_id',
  },
  {
    variable: '@seq',
    tokenAlt: '{seq}',
    category: 'burst',
    displayName: 'Sequence Number (3-Digit Padded)',
    description: 'Zero-padded index of sliced document in burst batch (001, 002, 003...)',
    example: '001',
  },
  {
    variable: '@seq_raw',
    tokenAlt: '{seq_raw}',
    category: 'burst',
    displayName: 'Raw Sequence Number',
    description: 'Unpadded integer index (1, 2, 3...)',
    example: '1',
  },
  {
    variable: '@total_slices',
    tokenAlt: '{total_slices}',
    category: 'burst',
    displayName: 'Total Slice Count',
    description: 'Total number of entities sliced in current burst batch',
    example: '48',
  },
  {
    variable: '@batch_id',
    tokenAlt: '{batch_id}',
    category: 'burst',
    displayName: 'Burst Batch ID',
    description: 'Unique execution batch identifier',
    example: 'btch-9a4f2e',
  },

  // Date & Calendar variables
  {
    variable: '@date',
    tokenAlt: '{date}',
    category: 'datetime',
    displayName: 'ISO Date (YYYY-MM-DD)',
    description: 'Full calendar date of execution',
    example: '2026-08-24',
  },
  {
    variable: '@year',
    tokenAlt: '{YYYY}',
    category: 'datetime',
    displayName: '4-Digit Year',
    description: 'Current execution calendar year',
    example: '2026',
  },
  {
    variable: '@month',
    tokenAlt: '{MM}',
    category: 'datetime',
    displayName: '2-Digit Month',
    description: 'Zero-padded execution month (01 to 12)',
    example: '08',
  },
  {
    variable: '@month_name',
    tokenAlt: '{month_name}',
    category: 'datetime',
    displayName: 'Month Name',
    description: 'Full month name in English',
    example: 'August',
  },
  {
    variable: '@day',
    tokenAlt: '{DD}',
    category: 'datetime',
    displayName: '2-Digit Day of Month',
    description: 'Zero-padded calendar day (01 to 31)',
    example: '24',
  },
  {
    variable: '@quarter',
    tokenAlt: '{period}',
    category: 'datetime',
    displayName: 'Accounting Quarter / Period',
    description: 'Calendar quarter designation (e.g. 2026-Q3)',
    example: '2026-Q3',
  },
  {
    variable: '@timestamp',
    tokenAlt: '{timestamp}',
    category: 'datetime',
    displayName: 'Compact Timestamp',
    description: 'Safe alphanumeric timestamp (YYYYMMDD_HHMMSS)',
    example: '20260824_140000',
  },
  {
    variable: '@effective_date',
    tokenAlt: '{effective_date}',
    category: 'datetime',
    displayName: 'Valuation / Effective Date',
    description: 'Accounting as-of date (incorporating calendar T-1 offsets)',
    example: '2026-08-23',
  },
];

export const PATH_EXPRESSION_PRESETS = [
  {
    name: 'Multi-Tenant Standard Route',
    description: 'Routes by tenant code into year/month directories with client-sliced files',
    folderExpr: '/tenants/@tenant_code/@year/@month/',
    fileExpr: '@report_code_@slice_key',
  },
  {
    name: 'Core Master Report Repository',
    description: 'Routes core reports to global repository, or client tenant reports to tenant directory',
    folderExpr: '=IIF(@is_core, "/core_reports/" + @report_code + "/" + @quarter + "/", "/tenants/" + @tenant_code + "/" + @year + "/" + @month + "/")',
    fileExpr: '=Concat(@report_code, "_", @tenant_code, "_", @slice_key, "_", @seq)',
  },
  {
    name: 'Sequenced Client Burst Packages',
    description: 'Includes 3-digit sequence number and ISO date to guarantee unique audit archives',
    folderExpr: '/tenants/@tenant_code/@year/@month/@slice_key/',
    fileExpr: '@report_code_@slice_key_@date_@seq',
  },
  {
    name: 'Compliance & Audit Vault',
    description: 'Secure partition organized by effective valuation date and execution batch ID',
    folderExpr: '/compliance_vault/@tenant_code/@effective_date/@batch_id/',
    fileExpr: '@report_code_@slice_key_@timestamp',
  },
];

/**
 * Creates default sample context for live evaluation simulation.
 */
export function getDefaultEvaluationContext(overrides?: Partial<PathEvaluationContext>): PathEvaluationContext {
  const now = new Date();
  const yyyy = String(now.getFullYear());
  const mm = String(now.getMonth() + 1).padStart(2, '0');
  const dd = String(now.getDate()).padStart(2, '0');
  const monthNames = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'
  ];
  const quarterMonth = Math.floor(now.getMonth() / 3) + 1;
  const quarter = `${yyyy}-Q${quarterMonth}`;
  const timestamp = `${yyyy}${mm}${dd}_${String(now.getHours()).padStart(2, '0')}${String(now.getMinutes()).padStart(2, '0')}${String(now.getSeconds()).padStart(2, '0')}`;

  return {
    tenant_code: 'acme_wealth',
    tenant_name: 'Acme Wealth Management',
    tenant_id: '8f3a9e22-1d54-4f9e-a612-88231901df42',
    is_core: false,
    gold_copy: false,
    report_id: 'rep-custom-001',
    report_name: 'Daily Institutional Client Valuation',
    report_code: 'daily_institutional_client_valuation',
    executed_by: 'admin@uuisce.internal',

    slice_key: 'client-001',
    burst_key: 'client-001',
    client_id: 'client-001',
    slice_name: 'Apex Global Alpha Fund',
    client_name: 'Apex Global Alpha Fund',
    slice_dim: 'client_id',
    seq: '001',
    seq_raw: 1,
    total_slices: 24,
    batch_id: 'btch-9a4f2e',

    date: `${yyyy}-${mm}-${dd}`,
    today: `${yyyy}-${mm}-${dd}`,
    year: yyyy,
    YYYY: yyyy,
    month: mm,
    MM: mm,
    month_name: monthNames[now.getMonth()],
    day: dd,
    DD: dd,
    quarter,
    period: quarter,
    timestamp,
    effective_date: `${yyyy}-${mm}-${dd}`,

    ...overrides,
  };
}

/**
 * Evaluates a path or filename expression against the provided context.
 * Supports:
 * 1. Formula Mode (starts with '=') e.g. =Concat("/tenants/", @tenant_code, "/", @year)
 * 2. Dynamic Variable Mode with @var or @{var} e.g. /tenants/@tenant_code/@year/@month/
 * 3. Legacy Token Mode with {token} e.g. /reports/{tenant}/{YYYY}/{MM}/
 */
export function evaluatePathExpression(
  expression: string,
  context: PathEvaluationContext
): { result: string; isFormula: boolean; error: string | null } {
  if (!expression || typeof expression !== 'string') {
    return { result: '', isFormula: false, error: null };
  }

  const trimmed = expression.trim();

  // 1. Formula Expression (starts with '=')
  if (trimmed.startsWith('=')) {
    try {
      const formulaBody = trimmed.slice(1);
      const evaluated = evaluateFormula(formulaBody, context);
      return { result: String(evaluated), isFormula: true, error: null };
    } catch (err: any) {
      return { result: '', isFormula: true, error: err.message || 'Formula Evaluation Error' };
    }
  }

  // 2. String Interpolation Mode (@variable, @{variable}, {variable})
  let output = expression;

  // Substitute all variables in context
  const fullContext = { ...getDefaultEvaluationContext(), ...context };

  // Sort keys by length descending to prevent sub-string collision (e.g. @tenant_name before @tenant)
  const keys = Object.keys(fullContext).sort((a, b) => b.length - a.length);

  for (const k of keys) {
    const rawVal = fullContext[k];
    const val = rawVal !== undefined && rawVal !== null ? String(rawVal) : '';

    // Replace @{key}
    output = output.split(`@{${k}}`).join(val);
    // Replace @key
    output = output.split(`@${k}`).join(val);
    // Replace {key}
    output = output.split(`{${k}}`).join(val);
  }

  // Common synonyms:
  output = output.split('{tenant}').join(String(fullContext.tenant_code || fullContext.tenant_id || ''));
  output = output.split('@tenant').join(String(fullContext.tenant_code || fullContext.tenant_id || ''));

  return { result: output, isFormula: false, error: null };
}

/**
 * Lightweight safe formula interpreter for SSRS/Crystal path expressions.
 */
function evaluateFormula(formula: string, context: PathEvaluationContext): any {
  // Built-in function environment
  const env: Record<string, any> = {
    IIF: (cond: any, a: any, b: any) => (Boolean(cond) ? a : b),
    Concat: (...args: any[]) => args.map((x) => (x !== null && x !== undefined ? String(x) : '')).join(''),
    Upper: (str: any) => String(str ?? '').toUpperCase(),
    Lower: (str: any) => String(str ?? '').toLowerCase(),
    PadLeft: (val: any, len: number, padChar = '0') => String(val ?? '').padStart(len, padChar),
    Replace: (str: any, search: string, replaceWith: string) => String(str ?? '').split(search).join(replaceWith),
    Coalesce: (...args: any[]) => args.find((x) => x !== null && x !== undefined && x !== '') ?? '',
    FormatDate: (dateVal: any, formatStr: string) => {
      const d = dateVal ? new Date(dateVal) : new Date();
      if (isNaN(d.getTime())) return String(dateVal);
      const yyyy = String(d.getFullYear());
      const mm = String(d.getMonth() + 1).padStart(2, '0');
      const dd = String(d.getDate()).padStart(2, '0');
      return (formatStr || 'YYYY-MM-DD')
        .replace(/YYYY/g, yyyy)
        .replace(/MM/g, mm)
        .replace(/DD/g, dd);
    },
  };

  // Replace variable identifiers like @tenant_code with env access
  const fullContext = { ...getDefaultEvaluationContext(), ...context };
  let jsExpr = formula;

  // Replace functions with env.FunctionName
  for (const fn of Object.keys(env)) {
    const fnRegex = new RegExp(`\\b${fn}\\s*\\(`, 'g');
    jsExpr = jsExpr.replace(fnRegex, `__env.${fn}(`);
  }

  // Replace @variables with __ctx.varName
  const varKeys = Object.keys(fullContext).sort((a, b) => b.length - a.length);
  for (const k of varKeys) {
    const varRegex = new RegExp(`@${k}\\b`, 'g');
    jsExpr = jsExpr.replace(varRegex, `__ctx['${k}']`);
  }

  // Replace any leftover @{key}
  for (const k of varKeys) {
    const varRegex = new RegExp(`@\\{${k}\\}`, 'g');
    jsExpr = jsExpr.replace(varRegex, `__ctx['${k}']`);
  }

  // Evaluate safely with restricted scope
  const evaluatorFn = new Function('__env', '__ctx', `return (${jsExpr});`);
  return evaluatorFn(env, fullContext);
}
