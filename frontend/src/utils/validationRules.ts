/**
 * Utility helpers for filtering validation rules to a specific entity and its subtypes
 * 
 * Rules can be:
 * 1. GLOBAL - target_entities contains 'global' (applies to all entities)
 * 2. ENTITY-SPECIFIC - target_entities contains specific entity names
 * 3. MIXED - target_entities contains both 'global' and specific entities
 */

import { devLog } from '../utils/devLogger';

export interface AnnotatedValidationRule {
  id: string;
  rule_name: string;
  entity?: string;
  target_entity?: string;
  target_entities: string[];
  sub_entity_type?: string;
  entity_id?: string;
  severity: 'error' | 'warning' | 'info';
  isGlobal: boolean;
  isEntitySpecific: boolean;
  assignmentType: 'global' | 'direct' | 'mixed';
  [key: string]: any;
}

export function buildMatchSet(entityKey: string, entity: any): Set<string> {
  const set = new Set<string>();
  const add = (v: any) => {
    if (!v) return;
    if (typeof v !== 'string') return;
    set.add(v.toLowerCase());
  };

  add(entityKey);
  add(entity.name);
  add(entity.businessName);
  add((entity as any).technicalName);
  add((entity as any).technical_name);

  if (entity.subtypes) {
    Object.entries(entity.subtypes).forEach(([k, st]: [string, any]) => {
      add(k);
      add(st?.name);
      add(st?.businessName);
      add(st?.technicalName);
      add(st?.technical_name);
    });
  }

  devLog('buildMatchSet for entity', entityKey, ':', Array.from(set));
  return set;
}

export function filterValidationRulesForEntity(entityKey: string, entity: any, rawRules: any[]): AnnotatedValidationRule[] {
  const matchSet = buildMatchSet(entityKey, entity);
  devLog('filterValidationRulesForEntity called with:', { entityKey, entity: entity?.name, rulesCount: rawRules?.length });

  const normalized = (rawRules || []).map((r: any) => {
    const targetEntities = r?.target_entities || [];
    const hasGlobal = Array.isArray(targetEntities) && targetEntities.includes('global');

    // Check if rule targets this specific entity
    const hasEntitySpecific = Array.isArray(targetEntities) &&
      targetEntities.some((te: any) => te !== 'global' && matchSet.has(String(te).toLowerCase()));

    // Fallback to legacy fields if target_entities is empty
    let isEntitySpecific = hasEntitySpecific;
    if (!isEntitySpecific && targetEntities.length === 0) {
      // Check legacy fields
      if (r?.target_entity && matchSet.has(String(r.target_entity).toLowerCase())) isEntitySpecific = true;
      if (r?.entity && matchSet.has(String(r.entity).toLowerCase())) isEntitySpecific = true;
      if (r?.sub_entity_type && matchSet.has(String(r.sub_entity_type).toLowerCase())) isEntitySpecific = true;
      if (r?.entity_id && matchSet.has(String(r.entity_id).toLowerCase())) isEntitySpecific = true;
    }

    const assignmentType: 'global' | 'direct' | 'mixed' =
      hasGlobal && isEntitySpecific ? 'mixed' :
        hasGlobal ? 'global' :
          'direct';

    devLog('Rule matching result:', {
      ruleName: r?.rule_name,
      targetEntities,
      targetEntity: r?.target_entity,
      hasGlobal,
      hasEntitySpecific,
      isEntitySpecific,
      assignmentType,
    });

    return {
      id: r?.id,
      rule_name: r?.rule_name || r?.name,
      entity: r?.entity,
      target_entity: r?.target_entity,
      target_entities: targetEntities,
      sub_entity_type: r?.sub_entity_type,
      entity_id: r?.entity_id || r?.target_entity_id,
      isGlobal: hasGlobal,
      isEntitySpecific,
      assignmentType,
      ...r, // Include all original fields for compatibility
    };
  });

  // Filter to only rules that apply to this entity (global or entity-specific or mixed)
  const filtered = normalized.filter((rule: AnnotatedValidationRule) => {
    return rule.isGlobal || rule.isEntitySpecific;
  });

  devLog('Filtered rules for entity', entityKey, ':', {
    total: filtered.length,
    byType: {
      global: filtered.filter(r => r.assignmentType === 'global').length,
      direct: filtered.filter(r => r.assignmentType === 'direct').length,
      mixed: filtered.filter(r => r.assignmentType === 'mixed').length,
    }
  });

  return filtered;
}

export function categorizeValidationRules(rules: AnnotatedValidationRule[]): {
  global: AnnotatedValidationRule[];
  direct: AnnotatedValidationRule[];
  mixed: AnnotatedValidationRule[];
} {
  return {
    global: rules.filter(r => r.assignmentType === 'global'),
    direct: rules.filter(r => r.assignmentType === 'direct'),
    mixed: rules.filter(r => r.assignmentType === 'mixed'),
  };
}

export default { buildMatchSet, filterValidationRulesForEntity, categorizeValidationRules };
