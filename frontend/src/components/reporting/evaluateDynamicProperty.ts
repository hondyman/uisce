export type DynamicProperty<T> = {
  isExpression: boolean;
  value: T;
  formula?: string;
};

export interface InstitutionalTableColumn {
  id: string;
  fieldKey: string;
  headerText: DynamicProperty<string>;
  textColor: DynamicProperty<string>;
  backgroundColor: DynamicProperty<string>;
  fontWeight: DynamicProperty<number | string>;
  fontStyle: DynamicProperty<'normal' | 'italic'>;
  fontSize: DynamicProperty<string>;
  formatMask: DynamicProperty<string>;
  isVisible: DynamicProperty<boolean>;
  widthPx: number;
}

const DANGEROUS_KEYWORDS = [
  'window', 'document', 'eval', 'fetch', 'import', 'require',
  'process', 'globalThis', 'global', 'Function', 'constructor',
  '__proto__', 'prototype',
];

function isExpressionSafe(expr: string): boolean {
  const lower = expr.toLowerCase();
  return !DANGEROUS_KEYWORDS.some(kw => lower.includes(kw));
}

export function evaluateDynamicProperty<T>(
  prop: DynamicProperty<T> | undefined | any,
  rowContext: Record<string, any> = {},
  globalContext: { pageNumber?: number; totalPages?: number; userName?: string; executionTime?: Date } = {}
): T {
  if (prop === undefined || prop === null) return undefined as unknown as T;

  if (typeof prop !== 'object' || !('isExpression' in prop)) {
    return prop as T;
  }

  if (!prop.isExpression || !prop.formula) {
    return prop.value;
  }

  const expr = prop.formula.trim();
  const pageNumber = globalContext.pageNumber ?? 1;
  const totalPages = globalContext.totalPages ?? 1;
  const userName = globalContext.userName ?? 'User';
  const executionTime = globalContext.executionTime ? globalContext.executionTime.toISOString() : new Date().toISOString();

  let resolvedExpr = expr
    .replace(/{PageNumber}/g, String(pageNumber))
    .replace(/{TotalPages}/g, String(totalPages))
    .replace(/{UserName}/g, `"${userName}"`)
    .replace(/{ExecutionTime}/g, `"${executionTime}"`);

  resolvedExpr = resolvedExpr.replace(/Fields!([a-zA-Z0-9_]+)\.Value/g, (_: string, fieldName: string) => {
    const val = rowContext[fieldName] ?? rowContext[fieldName.toLowerCase()];
    return typeof val === 'string' ? JSON.stringify(val) : String(val ?? 0);
  });

  resolvedExpr = resolvedExpr.replace(/\[(?:[a-zA-Z0-9_]+\.)?([a-zA-Z0-9_]+)\]/g, (_: string, fieldName: string) => {
    const val = rowContext[fieldName] ?? rowContext[fieldName.toLowerCase()];
    return typeof val === 'string' ? JSON.stringify(val) : String(val ?? 0);
  });

  resolvedExpr = resolvedExpr.replace(/IIF\s*\((.*?),(.*?),(.*?)\)/gi, '($1 ? $2 : $3)');

  if (resolvedExpr.startsWith('=')) {
    resolvedExpr = resolvedExpr.substring(1).trim();
  }

  if (!isExpressionSafe(resolvedExpr)) {
    console.warn('[evaluateDynamicProperty] Blocked dangerous expression:', resolvedExpr);
    return prop.value;
  }

  try {
    const evalFn = new Function('context', `with(context) { return ${resolvedExpr}; }`);
    return evalFn(rowContext);
  } catch (err) {
    return prop.value;
  }
}

export function evaluateCustomAggregate(
  rows: Record<string, unknown>[],
  expression: string,
  fieldContext: Record<string, unknown> = {}
): number {
  if (!isExpressionSafe(expression)) {
    console.warn('[evaluateCustomAggregate] Blocked dangerous expression:', expression);
    return 0;
  }

  try {
    const vals = rows.map(row => {
      const ctx = { ...row, ...fieldContext };
      const resolved = expression
        .replace(/Fields!([a-zA-Z0-9_]+)\.Value/g, (_, fn) => {
          const v = ctx[fn] ?? ctx[fn.toLowerCase()];
          return typeof v === 'string' ? JSON.stringify(v) : String(v ?? 0);
        })
        .replace(/\[([a-zA-Z0-9_]+)\]/g, (_, fn) => {
          const v = ctx[fn] ?? ctx[fn.toLowerCase()];
          return typeof v === 'string' ? JSON.stringify(v) : String(v ?? 0);
        });
      return new Function('ctx', `with(ctx) { return ${resolved}; }`)(ctx);
    }).filter(v => typeof v === 'number' && Number.isFinite(v as number));

    return vals.length ? vals.reduce((a: number, b: number) => a + b, 0) / vals.length : 0;
  } catch {
    return 0;
  }
}

export interface FormatMaskResult {
  format: (value: number) => string;
  supportedTokens: string[];
  unsupportedTokens: string[];
  isSupported: boolean;
}

export function parseFormatMask(mask: string): FormatMaskResult {
  if (!mask) {
    return {
      format: (v: number) => String(v),
      supportedTokens: [],
      unsupportedTokens: [],
      isSupported: false,
    };
  }

  const supportedTokens: string[] = [];
  const unsupportedTokens: string[] = [];

  const knownTokens = ['0', '00', '#', ',', '.', '%', 'E+', 'E-', 'e+', 'e-'];
  const knownSeparators = [' ', ',', '.', ';', ':', '/', '-', '(', ')', '€', '£', '$'];
  const knownColors = ['[Red]', '[Blue]', '[Green]', '[Yellow]', '[Magenta]', '[Cyan]'];

  const sections = mask.split(';');
  const positiveSection = sections[0] || mask;
  const negativeSection = sections.length > 1 ? sections[1] : `-${positiveSection.replace(/-/g, '')}`;
  const hasColorNegatives = sections.length > 1 && /\[Red\]/i.test(negativeSection);

  const numMatch = positiveSection.match(/([#,]*[0#]+(?:\.[0#]+)?)/);
  const hasPercent = mask.includes('%');
  const hasThousands = mask.includes(',');
  const hasDecimal = mask.includes('.');
  const decPart = numMatch && numMatch[1].includes('.') ? numMatch[1].split('.')[1] : '';

  if (numMatch) {
    supportedTokens.push(numMatch[1]);
    if (hasPercent) supportedTokens.push('%');
    if (hasThousands) supportedTokens.push(',');
    if (hasDecimal) supportedTokens.push('.');
  }

  knownColors.forEach(c => {
    if (mask.includes(c)) supportedTokens.push(c);
  });

  const format = (value: number): string => {
    const absVal = Math.abs(value);
    const isNeg = value < 0;
    const section = isNeg ? negativeSection : positiveSection;

    let formatted: string;
    if (hasPercent) {
      formatted = `${(absVal * 100).toFixed(decPart.length)}%`;
    } else {
      const decimals = decPart ? decPart.length : 2;
      formatted = absVal.toLocaleString('en-US', {
        minimumFractionDigits: hasDecimal ? decimals : 0,
        maximumFractionDigits: hasDecimal ? decimals : 0,
        useGrouping: hasThousands,
      });
    }

    if (isNeg && !section.startsWith('-')) {
      formatted = `-${formatted}`;
    }

    return formatted;
  };

  return {
    format,
    supportedTokens,
    unsupportedTokens,
    isSupported: true,
  };
}
