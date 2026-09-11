import { evaluateRuleWasm } from './wasmRuntime';

export interface ValidationRuleDescriptor {
  id: string;
  name: string;
  description?: string;
  bo_name: string;
  severity: string;
  timing: string;
  category?: string;
  domain?: string;
  rule_ast: unknown;
  is_active: boolean;
  governance_status?: string;
}

export interface RuleEvaluationResult {
  ruleId: string;
  ruleName: string;
  ruleDescription?: string;
  passed: boolean;
  isServerSideOnly: boolean;
  serverSideReason?: string;
  error?: string;
}

export interface RuleGridEvaluation {
  ruleResults: RuleEvaluationResult[];
  allPassed: boolean;
  summary: string;
}

interface RulesCache {
  rules: ValidationRuleDescriptor[];
  fetchedAt: number;
}

const rulesCache: Record<string, RulesCache> = {};
const CACHE_TTL_MS = 5 * 60 * 1000;

export function boHasCollectionKeys(boKey: string): boolean {
  const collectionKeys: Record<string, string[]> = {
    order: ['OrderAllocations'],
  };
  return (collectionKeys[boKey]?.length ?? 0) > 0;
}

export function getCollectionKeysForBO(boKey: string): string[] {
  const collectionKeys: Record<string, string[]> = {
    order: ['OrderAllocations'],
  };
  return collectionKeys[boKey] ?? [];
}

export async function fetchActiveRulesForBO(
  boKey: string,
  tenantId: string
): Promise<ValidationRuleDescriptor[]> {
  const now = Date.now();
  const cached = rulesCache[boKey];
  if (cached && now - cached.fetchedAt < CACHE_TTL_MS) {
    return cached.rules.filter((r) => r.is_active);
  }

  const res = await fetch(
    `/api/validation-rule-nodes?bo_name=${encodeURIComponent(boKey)}`,
    {
      headers: {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json',
      },
    }
  );

  if (!res.ok) {
    throw new Error(`Failed to fetch rules for BO ${boKey}: ${res.statusText}`);
  }

  const data = await res.json();
  const rules: ValidationRuleDescriptor[] = data.validationRules ?? [];

  rulesCache[boKey] = { rules, fetchedAt: now };
  return rules.filter((r) => r.is_active);
}

function canEvaluateClientSide(ruleAst: unknown): boolean {
  if (!ruleAst || typeof ruleAst !== 'object') return false;
  const ast = ruleAst as Record<string, unknown>;

  function inspect(node: unknown): boolean {
    if (!node || typeof node !== 'object') return true;
    const n = node as Record<string, unknown>;
    if (n.type === 'condition' || n.type === 'expression') {
      const fieldPath = n.field_path ?? n.FieldPath ?? n.fieldPath;
      if (typeof fieldPath === 'string' && fieldPath.includes('.')) {
        const collectionKey = fieldPath.split('.')[0];
        const knownCollections = ['OrderAllocations'];
        if (!knownCollections.includes(collectionKey)) {
          return false;
        }
      }
      const field = n.field ?? n.Field ?? n.fieldName;
      if (typeof field === 'string') {
        const serverSideFields = [
          'account_status',
          'placement_routed_sum',
          'settlement_risk_score',
          'regulatory_flag_count',
        ];
        if (serverSideFields.includes(field)) {
          return false;
        }
      }
    }
    if (Array.isArray(n.conditions)) {
      return (n.conditions as unknown[]).every(inspect);
    }
    if (Array.isArray(n.args)) {
      return (n.args as unknown[]).every(inspect);
    }
    if (Array.isArray(n.left)) {
      return (n.left as unknown[]).every(inspect);
    }
    if (Array.isArray(n.right)) {
      return (n.right as unknown[]).every(inspect);
    }
    return true;
  }

  return inspect(ast);
}

export async function evaluateRulesForGrid(
  boKey: string,
  parentRecord: Record<string, unknown>,
  collectionRows: Record<string, unknown>[],
  tenantId: string
): Promise<RuleGridEvaluation> {
  if (!boHasCollectionKeys(boKey)) {
    return { ruleResults: [], allPassed: true, summary: '' };
  }

  const activeRules = await fetchActiveRulesForBO(boKey, tenantId);

  if (activeRules.length === 0) {
    return { ruleResults: [], allPassed: true, summary: '' };
  }

  const collectionKeys = getCollectionKeysForBO(boKey);
  const ctx: Record<string, unknown> = { ...parentRecord };
  for (const key of collectionKeys) {
    ctx[key] = collectionRows;
  }

  const results: RuleEvaluationResult[] = [];

  for (const rule of activeRules) {
    const clientEvaluable = canEvaluateClientSide(rule.rule_ast);

    if (!clientEvaluable) {
      results.push({
        ruleId: rule.id,
        ruleName: rule.name,
        ruleDescription: rule.description,
        passed: false,
        isServerSideOnly: true,
        serverSideReason: 'References server-computed fields',
      });
      continue;
    }

    try {
      const passed = await evaluateRuleWasm(rule.rule_ast, ctx);
      results.push({
        ruleId: rule.id,
        ruleName: rule.name,
        ruleDescription: rule.description,
        passed,
        isServerSideOnly: false,
      });
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      const isServerField =
        msg.includes('not found') || msg.includes('unresolved') || msg.includes('field');

      results.push({
        ruleId: rule.id,
        ruleName: rule.name,
        ruleDescription: rule.description,
        passed: false,
        isServerSideOnly: isServerField,
        serverSideReason: isServerField ? 'References fields unavailable in browser context' : msg,
        error: isServerField ? undefined : msg,
      });
    }
  }

  const allPassed = results.every((r) => r.passed || r.isServerSideOnly);
  const serverSideCount = results.filter((r) => r.isServerSideOnly).length;
  const evaluatedCount = results.filter((r) => !r.isServerSideOnly).length;
  const passedCount = results.filter((r) => r.passed).length;

  let summary: string;
  if (evaluatedCount === 0) {
    summary = `${serverSideCount} rule${serverSideCount !== 1 ? 's' : ''} — server-side only`;
  } else {
    summary = `${passedCount}/${evaluatedCount} rules passing`;
    if (serverSideCount > 0) {
      summary += ` (+ ${serverSideCount} server-side)`;
    }
  }

  return { ruleResults: results, allPassed, summary };
}
