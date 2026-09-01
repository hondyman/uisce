import { ReportParameter } from './builderSerialization';

export interface UserSessionProfile {
  id?: string;
  name?: string;
  email?: string;
  tenantId?: string;
  tenantCode?: string;
  accountId?: string;
  clientId?: string;
  branchId?: string;
  region?: string;
  roles?: string[];
  department?: string;
  [key: string]: any;
}

/**
 * Resolves dynamic relative date keywords into ISO format YYYY-MM-DD
 */
export function resolveRelativeDate(keyword: string): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = now.getMonth(); // 0-indexed
  const date = now.getDate();

  const pad = (n: number) => String(n).padStart(2, '0');
  const formatDate = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

  const upper = (keyword || '').trim().toUpperCase();

  switch (upper) {
    case 'TODAY':
    case 'NOW':
    case 'CURRENT_DATE':
      return formatDate(now);

    case 'YESTERDAY': {
      const y = new Date(now);
      y.setDate(date - 1);
      return formatDate(y);
    }

    case 'TOMORROW': {
      const t = new Date(now);
      t.setDate(date + 1);
      return formatDate(t);
    }

    case 'START_OF_YEAR':
    case 'YTD':
    case 'YEAR_TO_DATE':
      return `${year}-01-01`;

    case 'END_OF_YEAR':
      return `${year}-12-31`;

    case 'START_OF_MONTH':
    case 'MTD':
    case 'MONTH_TO_DATE':
      return `${year}-${pad(month + 1)}-01`;

    case 'END_OF_MONTH': {
      const lastDay = new Date(year, month + 1, 0).getDate();
      return `${year}-${pad(month + 1)}-${pad(lastDay)}`;
    }

    case 'PREV_MONTH':
    case 'START_OF_PREV_MONTH': {
      const prevMonthDate = new Date(year, month - 1, 1);
      return `${prevMonthDate.getFullYear()}-${pad(prevMonthDate.getMonth() + 1)}-01`;
    }

    case 'END_OF_PREV_MONTH': {
      const prevMonthEnd = new Date(year, month, 0);
      return formatDate(prevMonthEnd);
    }

    case 'START_OF_QUARTER': {
      const quarterStartMonth = Math.floor(month / 3) * 3;
      return `${year}-${pad(quarterStartMonth + 1)}-01`;
    }

    case 'PREV_QUARTER':
    case 'START_OF_PREV_QUARTER': {
      const currentQuarter = Math.floor(month / 3);
      const prevQuarterStartMonth = ((currentQuarter - 1 + 4) % 4) * 3;
      const prevQuarterYear = currentQuarter === 0 ? year - 1 : year;
      return `${prevQuarterYear}-${pad(prevQuarterStartMonth + 1)}-01`;
    }

    default:
      return keyword;
  }
}

/**
 * Resolves user context variables, relative date expressions, 
 * and multi-value arrays to personalize report parameters automatically.
 */
export function resolveParameterDefaults(
  parameters: ReportParameter[],
  userProfile?: UserSessionProfile
): Record<string, any> {
  const resolved: Record<string, any> = {};

  parameters.forEach((param) => {
    let val: any = param.defaultValue || '';

    // 1. Contextual Personalization & Auto-Binding
    const contextKey = param.userContextKey || param.contextKey;
    if (param.sourceType === 'context' || contextKey) {
      if (userProfile) {
        if (contextKey === 'user.id' || contextKey === 'id' || contextKey === 'userId') {
          val = userProfile.id || val;
        } else if (contextKey === 'tenant.id' || contextKey === 'tenantId') {
          val = userProfile.tenantId || userProfile.tenantCode || val;
        } else if (contextKey === 'accountId' || contextKey === 'user.accountId') {
          val = userProfile.accountId || val;
        } else if (contextKey === 'clientId' || contextKey === 'user.clientId') {
          val = userProfile.clientId || val;
        } else if (contextKey === 'branchId' || contextKey === 'user.branchId') {
          val = userProfile.branchId || val;
        } else if (contextKey === 'user.region' || contextKey === 'region') {
          val = userProfile.region || val;
        } else if (contextKey && userProfile[contextKey] !== undefined) {
          val = userProfile[contextKey];
        }
      }
    }

    // 2. Relative Date Keywords
    if (param.type === 'date' && typeof val === 'string') {
      val = resolveRelativeDate(val);
    }

    // 3. Multi-Value Array handling
    if (param.allowMultiple && typeof val === 'string' && val.includes(',')) {
      val = val.split(',').map((s) => s.trim()).filter(Boolean);
    }

    resolved[param.name] = val;
  });

  return resolved;
}
