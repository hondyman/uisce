
import { fetchAPI } from '../api';
import type { Entities, Entity } from '../types/entity-schema';
import { devLog, devDebug, devWarn } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';

function normalizeTechnicalName(input: string): string {
  return (input || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '');
}

function mapFieldType(input: any): 'text' | 'number' | 'date' | 'boolean' | 'json' | 'array' {
  const t = String(input || '').toLowerCase();
  if (t.includes('bool')) return 'boolean';
  if (t.includes('date') || t.includes('time')) return 'date';
  if (t.includes('int') || t.includes('num') || t.includes('decimal') || t.includes('float') || t.includes('double')) return 'number';
  if (t.includes('json') || t.includes('object') || t.includes('map')) return 'json';
  if (t.includes('array') || t.endsWith('[]')) return 'array';
  return 'text';
}

function coerceEntitySchema(result: any): Entities {
  devDebug('[coerceEntitySchema] INPUT:', result);
  // Backend returns a map keyed by UUID with values containing config.fields/entity_fields.
  if (!result || typeof result !== 'object' || Array.isArray(result) || result.changed || result.deleted) {
    devWarn('[coerceEntitySchema] Invalid input or delta format');
    return {};
  }

  const entries = Object.entries(result as Record<string, any>);
  if (entries.length === 0) {
    devWarn('[coerceEntitySchema] Empty input');
    return {};
  }

  const looksLikeBusinessEntities = entries.some(([, v]) => v && typeof v === 'object' && v.config && typeof v.config === 'object');
  if (!looksLikeBusinessEntities) {
    devDebug('[coerceEntitySchema] Returning raw');
    return result as Entities;
  }

  devDebug(`[coerceEntitySchema] Processing ${entries.length} entries`);

  const out: Entities = {};
  for (const [id, raw] of entries) {
    if (!raw || typeof raw !== 'object') continue;
    const config = (raw.config && typeof raw.config === 'object') ? raw.config : {};
    const rawFields: any[] = (config.entity_fields as any[]) || (config.fields as any[]) || [];

    // ... rest of logic ...

    // Minimal normalization logging
    const technicalName = String(raw.technical_name || raw.technicalName || raw.name || id);
    const entityKey = normalizeTechnicalName(technicalName) || id;

    // ... keep existing mapping logic ...
    // Note: I need to reproduce the logic I'm replacing or just wrap the function.
    // Since replace_file_content is full block replacement, I should re-include the logic.

    // Wait, avoiding retyping the whole function logic complexity.
    // I can just add the logs at the start.

    const entityFields = (Array.isArray(rawFields) ? rawFields : []).map((f: any, idx: number) => {
      const key = String(f?.key || f?.name || `field_${idx}`);
      const display = String(f?.displayName || f?.display_name || f?.display_label || f?.label || f?.name || key);
      const technical = String(f?.technicalName || f?.technical_name || f?.technical || key);
      return {
        key,
        name: display,
        businessName: display,
        technicalName: normalizeTechnicalName(technical),
        type: mapFieldType(f?.type || f?.field_type),
        isCore: Boolean(f?.isCore ?? f?.is_core ?? true),
        sequence: typeof f?.displayOrder === 'number' ? f.displayOrder : (typeof f?.display_order === 'number' ? f.display_order : idx),
      };
    });

    const businessName = String(raw.display_name || raw.displayName || raw.name || technicalName);
    // key already computed above

    const entity: Entity = {
      id: id,
      key: entityKey,
      name: businessName,
      businessName,
      technicalName: normalizeTechnicalName(technicalName),
      description: raw.description,
      entity_fields: entityFields,
      subtypes: {},
      isCore: Boolean(raw.is_core ?? raw.isCore ?? false),
      coreFields: entityFields.filter((f) => f.isCore),
      customFields: entityFields.filter((f) => !f.isCore),
    };

    const rawSubtypes = raw.subtypes && typeof raw.subtypes === 'object' ? raw.subtypes : undefined;
    if (rawSubtypes) {
      for (const [subId, s] of Object.entries(rawSubtypes as Record<string, any>)) {
        if (!s || typeof s !== 'object') continue;
        const sConfig = (s.config && typeof s.config === 'object') ? s.config : {};
        const inherited = Array.isArray(sConfig.inheritedFields) ? sConfig.inheritedFields : [];
        const custom = Array.isArray(sConfig.customFields) ? sConfig.customFields : [];
        const subtypeKey = normalizeTechnicalName(String(s.technical_name || s.technicalName || s.name || subId)) || subId;
        entity.subtypes[subtypeKey] = {
          key: subtypeKey,
          name: String(s.display_name || s.displayName || s.name || subtypeKey),
          businessName: String(s.display_name || s.displayName || s.name || subtypeKey),
          technicalName: subtypeKey,
          entity_fields: inherited.map((f: any, idx: number) => ({
            key: String(f?.key || f?.name || `field_${idx}`),
            name: String(f?.displayName || f?.display_name || f?.display_label || f?.label || f?.name || `field_${idx}`),
            businessName: String(f?.displayName || f?.display_name || f?.display_label || f?.label || f?.name || `field_${idx}`),
            technicalName: normalizeTechnicalName(String(f?.technicalName || f?.technical_name || f?.technical || f?.key || f?.name || `field_${idx}`)),
            type: mapFieldType(f?.type || f?.field_type),
            isCore: true,
            sequence: idx,
            inheritedFrom: entityKey,
            inheritedFromKey: entityKey,
          })),
          subtype_fields: custom.map((f: any, idx: number) => ({
            key: String(f?.key || f?.name || `field_${idx}`),
            name: String(f?.displayName || f?.display_name || f?.display_label || f?.label || f?.name || `field_${idx}`),
            businessName: String(f?.displayName || f?.display_name || f?.display_label || f?.label || f?.name || `field_${idx}`),
            technicalName: normalizeTechnicalName(String(f?.technicalName || f?.technical_name || f?.technical || f?.key || f?.name || `field_${idx}`)),
            type: mapFieldType(f?.type || f?.field_type),
            isCore: false,
            sequence: idx,
            createdBy: 'system',
          })),
          isCore: Boolean(s.is_core ?? s.isCore ?? false),
          basedOnEntity: entityKey,
        };
      }
    }

    out[entityKey] = entity;
  }

  return out;
}

export interface EntitySchemaDelta {
  changed?: Record<string, Entity>;
  deleted?: string[];
}

export type EntitySchemaPayload = Entities | EntitySchemaDelta;

export function fetchEntitySchema(tenantId?: string, datasourceId?: string): Promise<Entities> {
  devLog('[fetchEntitySchema] Fetching schema from backend', { tenantId, datasourceId });

  const headers: Record<string, string> = { 'Content-Type': 'application/json' };

  // Add tenant headers if provided
  if (tenantId) {
    headers['X-Tenant-ID'] = tenantId;
  }
  if (datasourceId) {
    headers['X-Tenant-Datasource-ID'] = datasourceId;
  }
  headers['X-Tenant-Region'] = getSelectedRegion();
  try {
    const token = localStorage.getItem('auth_token');
    if (token) headers['Authorization'] = `Bearer ${token}`;
  } catch { /* ignore */ }

  return fetchAPI('/entity-schema', {
    method: 'GET',
    headers,
    cache: 'no-cache', // Ensure fresh data is fetched every time
  }).then((result: any) => {
    devLog('[fetchEntitySchema] Fetch successful:', { result });

    // If result is a delta, it won't have the entities directly - it has changed/deleted
    // For GET, we expect the full schema as the top-level object
    if (result && typeof result === 'object' && !result.changed && !result.deleted) {
      return coerceEntitySchema(result);
    }

    // Fallback: if for some reason we got a delta, return empty object
    devLog('[fetchEntitySchema] Unexpected response format:', { result });
    return {};
  }).catch((error: any) => {
    devLog('[fetchEntitySchema] Fetch failed:', { error });
    // Return empty object on error instead of throwing - allows page to still load
    return {};
  });
}

export function saveEntitySchema(payload: EntitySchemaPayload, tenantId?: string, datasourceId?: string): Promise<void> {
  devLog('[saveEntitySchema] Saving schema:', { payload, tenantId, datasourceId });

  const headers: Record<string, string> = { 'Content-Type': 'application/json' };

  // Add tenant headers if provided
  if (tenantId) {
    headers['X-Tenant-ID'] = tenantId;
  }
  if (datasourceId) {
    headers['X-Tenant-Datasource-ID'] = datasourceId;
  }
  headers['X-Tenant-Region'] = getSelectedRegion();
  try {
    const token = localStorage.getItem('auth_token');
    if (token) headers['Authorization'] = `Bearer ${token}`;
  } catch { /* ignore */ }

  const body = JSON.stringify(payload);
  devLog('[saveEntitySchema] Request body size:', { size: body.length, body });

  return fetchAPI('/entity-schema', {
    method: 'POST',
    headers,
    body,
  }).then((result) => {
    devLog('[saveEntitySchema] Save successful:', { result });
  }).catch((error) => {
    devLog('[saveEntitySchema] Save failed:', { error });
    throw error;
  });
}
