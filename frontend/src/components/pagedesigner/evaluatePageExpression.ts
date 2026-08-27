import { DynamicProperty } from './DynamicPropertyTypes';

export interface PageExpressionContext {
  rowData: Record<string, any>;
  parameters: Record<string, any>; // Page Event Bus state
  globalContext: {
    userName: string;
    executionTime: Date;
    tenantId: string;
  };
}

const DANGEROUS_KEYWORDS = [
  'window', 'document', 'eval', 'fetch', 'import', 'require',
  'process', 'globalThis', 'global', 'Function', 'constructor',
  '__proto__', 'prototype',
];

function isExpressionSafe(expr: string): boolean {
  const lower = expr.toLowerCase();
  return !DANGEROUS_KEYWORDS.some((kw) => lower.includes(kw));
}

// Splits IIF arguments respecting nested parens and string literals
function splitIIFArgs(argStr: string): [string, string, string] | null {
  const args: string[] = [];
  let current = '';
  let depth = 0;
  let inString: string | null = null;

  for (let i = 0; i < argStr.length; i++) {
    const char = argStr[i];
    if (inString) {
      current += char;
      if (char === inString && argStr[i - 1] !== '\\') {
        inString = null;
      }
    } else if (char === "'" || char === '"') {
      inString = char;
      current += char;
    } else if (char === '(') {
      depth++;
      current += char;
    } else if (char === ')') {
      depth--;
      current += char;
    } else if (char === ',' && depth === 0) {
      args.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }
  if (current) {
    args.push(current.trim());
  }

  if (args.length === 3) {
    return [args[0], args[1], args[2]];
  }
  return null;
}

function transformIIF(expr: string): string {
  let result = expr;
  let match: RegExpExecArray | null;
  const iifRegex = /IIF\s*\(/gi;

  while ((match = iifRegex.exec(result)) !== null) {
    const startIdx = match.index;
    let depth = 1;
    let endIdx = startIdx + match[0].length;
    let inString: string | null = null;

    while (endIdx < result.length && depth > 0) {
      const char = result[endIdx];
      if (inString) {
        if (char === inString && result[endIdx - 1] !== '\\') {
          inString = null;
        }
      } else if (char === "'" || char === '"') {
        inString = char;
      } else if (char === '(') {
        depth++;
      } else if (char === ')') {
        depth--;
      }
      endIdx++;
    }

    if (depth === 0) {
      const fullCall = result.substring(startIdx, endIdx);
      const innerArgs = fullCall.substring(match[0].length, fullCall.length - 1);
      const parsedArgs = splitIIFArgs(innerArgs);
      if (parsedArgs) {
        const replacement = `(${parsedArgs[0]} ? ${parsedArgs[1]} : ${parsedArgs[2]})`;
        result = result.substring(0, startIdx) + replacement + result.substring(endIdx);
        iifRegex.lastIndex = startIdx + replacement.length;
      }
    }
  }

  return result;
}

export function evaluatePageExpression<T>(
  prop: DynamicProperty<T> | undefined,
  context: PageExpressionContext
): T {
  if (!prop) return undefined as unknown as T;
  if (!prop.isExpression || !prop.formula) {
    return prop.value;
  }

  let expr = prop.formula.trim();

  // 1. Strip leading '=' if present
  if (expr.startsWith('=')) {
    expr = expr.substring(1).trim();
  }

  // 2. Replace Global Tokens ({UserName}, {ExecutionTime}, {TenantId})
  expr = expr
    .replace(/{UserName}/g, JSON.stringify(context.globalContext.userName || 'Current User'))
    .replace(/{TenantId}/g, JSON.stringify(context.globalContext.tenantId || 'active-tenant'))
    .replace(/{ExecutionTime}/g, JSON.stringify((context.globalContext.executionTime || new Date()).toISOString()));

  // 3. Replace Event Bus Parameters (Parameters!SelectedRegion.Value or @SelectedRegion)
  expr = expr.replace(/Parameters!([a-zA-Z0-9_]+)\.Value/g, (_, paramKey) => {
    const val = context.parameters ? context.parameters[paramKey] : undefined;
    return typeof val === 'string' ? JSON.stringify(val) : String(val ?? 'null');
  });
  expr = expr.replace(/@([a-zA-Z0-9_]+)/g, (_, paramKey) => {
    const val = context.parameters ? context.parameters[paramKey] : undefined;
    return typeof val === 'string' ? JSON.stringify(val) : String(val ?? 'null');
  });

  // 4. Replace BO Field Values (Fields!nav_end.Value or [nav_end])
  expr = expr.replace(/Fields!([a-zA-Z0-9_]+)\.Value/g, (_, fieldKey) => {
    const val = context.rowData ? context.rowData[fieldKey] : undefined;
    return typeof val === 'string' ? JSON.stringify(val) : String(val ?? 'null');
  });
  expr = expr.replace(/\[(?:[a-zA-Z0-9_]+\.)?([a-zA-Z0-9_]+)\]/g, (_, fieldKey) => {
    const val = context.rowData ? context.rowData[fieldKey] : undefined;
    return typeof val === 'string' ? JSON.stringify(val) : String(val ?? 'null');
  });

  // 5. SSRS IIF / Switch Function Translation
  expr = transformIIF(expr);

  if (!isExpressionSafe(expr)) {
    console.warn('[PageExpressionEngine] Blocked dangerous expression:', expr);
    return prop.value;
  }

  try {
    // Isolated evaluation scope with context.rowData
    const evalFn = new Function('context', `with(context) { return ${expr}; }`);
    return evalFn(context.rowData || {});
  } catch (err) {
    console.warn(`[PageExpressionEngine] Evaluation failed for "${prop.formula}":`, err);
    return prop.value;
  }
}
