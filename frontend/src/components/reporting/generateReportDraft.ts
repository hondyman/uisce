import type { SemanticTermView } from '../../features/query-builder/types/queryDef';
import type { ReportGenerationSpec, GeneratedReportField } from '../../api/reporting';
import { fetchBusinessObjectBindings } from '../../features/query-builder/services/queryBuilderApi';
import { ELEMENT_TYPES, REPORT_SECTIONS, buildDataBindingForType, type ReportElement } from './reportingUtils';

/**
 * Expands a ReportGenerationSpec into a merged element list - the single
 * function behind both AI entry points in SSRSReportBuilder.tsx
 * ("Generate with AI"/"Regenerate" and the in-editor copilot bar), per
 * this session's confirmed decision that regenerate stays additive on
 * existing reports too, not destructive. Merging into an empty
 * `existingElements` array (a brand-new report) is exactly equivalent to
 * building fresh, so one function covers both cases - mirroring Page
 * Studio's mergeGeneratedSpecIntoDraft, adapted for a free-position
 * canvas instead of a slotted layout tree.
 *
 * Dedup key is `${type}:${boKey}:${sortedDimensions}:${sortedMeasures}`,
 * NOT Page Studio's `${type}:${dataSourceId}` - report elements bake in
 * specific fields, so "same type + BO, different columns" is a
 * legitimately different widget, not a duplicate. This key still catches
 * "ran the same prompt twice."
 *
 * Placement is a real flow layout (no-overlap), replacing the weak
 * `{x:30,y:30}`-always / `%5`-cascade heuristics the manual paths use:
 * new elements start below the lowest existing body element and flow
 * left-to-right, wrapping past a ~1100px canvas width.
 */

const CANVAS_WIDTH = 1100;
const GUTTER = 24;
const START_X = 30;

const defaultSizes: Record<string, { width: number; height: number }> = {
  [ELEMENT_TYPES.TABLE]: { width: 560, height: 180 },
  [ELEMENT_TYPES.MATRIX]: { width: 480, height: 160 },
  [ELEMENT_TYPES.LIST]: { width: 340, height: 140 },
  [ELEMENT_TYPES.CHART]: { width: 340, height: 200 },
  [ELEMENT_TYPES.GAUGE]: { width: 160, height: 120 },
  [ELEMENT_TYPES.SPARKLINE]: { width: 180, height: 50 },
  [ELEMENT_TYPES.SLICER]: { width: 220, height: 140 },
  [ELEMENT_TYPES.FORM]: { width: 320, height: 320 },
};

function elementKey(type: string, boKey: string, dimensions: string[], measures: string[]): string {
  return `${type}:${boKey}:${[...dimensions].sort().join(',')}:${[...measures].sort().join(',')}`;
}

/** Converts a generated spec's flat field metadata into the SemanticTermView-shaped objects buildDataBindingForType expects, in the order the model chose them. */
function resolveFields(termNodeIds: string[] | undefined, pool: GeneratedReportField[]): SemanticTermView[] {
  if (!termNodeIds || termNodeIds.length === 0) return [];
  const byId = new Map(pool.map((f) => [f.termNodeId, f]));
  const out: SemanticTermView[] = [];
  for (const id of termNodeIds) {
    const f = byId.get(id);
    if (!f) continue;
    out.push({
      termNodeId: f.termNodeId,
      termKey: f.key,
      termName: f.key,
      displayName: f.displayName,
      dataType: f.dataType,
      role: f.role as SemanticTermView['role'],
      bindingStatus: 'RESOLVED',
    });
  }
  return out;
}

/** Resolves a related BO's default binding id, needed once per distinct boKey a generated element actually references - the primary BO's binding is already known client-side. */
async function resolveBindingId(boId: string): Promise<string> {
  const bindings = await fetchBusinessObjectBindings(boId).catch(() => []);
  const def = bindings.find((b) => b.isDefault) || bindings[0];
  return def?.bindingId || '';
}

export async function mergeGeneratedReportSpecIntoDraft(
  existingElements: ReportElement[],
  spec: ReportGenerationSpec,
  primary: { boId: string; bindingId: string; tenantId: string },
): Promise<ReportElement[]> {
  const bindingIdByBoKey = new Map<string, string>([['', primary.bindingId]]);
  const boIdByBoKey = new Map<string, string>([['', primary.boId]]);
  for (const rel of spec.relatedBusinessObjects || []) {
    boIdByBoKey.set(rel.boKey, rel.boId);
  }
  for (const boKey of new Set((spec.elements || []).map((e) => e.boKey).filter((k) => k && k !== ''))) {
    const boId = boIdByBoKey.get(boKey);
    if (boId && !bindingIdByBoKey.has(boKey)) {
      bindingIdByBoKey.set(boKey, await resolveBindingId(boId));
    }
  }

  const existingKeys = new Set(
    existingElements
      .filter((e) => e.section === REPORT_SECTIONS.BODY)
      .map((e) => {
        const dims = (e.properties?.dimensions || []).map((d: any) => d.termNodeId);
        const measures = (e.properties?.measures || []).map((m: any) => m.termNodeId);
        return elementKey(e.type, e.properties?.boKey || '', dims, measures);
      }),
  );

  // Flow-layout placement: start below the lowest existing body element.
  let cursorX = START_X;
  let cursorY = Math.max(
    0,
    ...existingElements
      .filter((e) => e.section === REPORT_SECTIONS.BODY)
      .map((e) => e.position.y + e.size.height),
  ) + (existingElements.length > 0 ? GUTTER : START_X);
  let rowMaxHeight = 0;

  const merged: ReportElement[] = [...existingElements];

  for (const el of spec.elements || []) {
    const boId = el.boKey === '' ? primary.boId : boIdByBoKey.get(el.boKey) || primary.boId;
    const bindingId = el.boKey === '' ? primary.bindingId : bindingIdByBoKey.get(el.boKey) || primary.bindingId;
    const fieldsPool = spec.fields?.[el.boKey] || [];
    const dims = resolveFields(el.dimensions, fieldsPool);
    const measures = resolveFields(el.measures, fieldsPool);

    const key = elementKey(el.type, el.boKey, dims.map((d) => d.termNodeId), measures.map((m) => m.termNodeId));
    if (existingKeys.has(key)) continue;
    existingKeys.add(key);

    const dataBinding = buildDataBindingForType(el.type, boId, bindingId, primary.tenantId, dims, measures);
    const size = defaultSizes[el.type] || { width: 320, height: 160 };

    if (cursorX + size.width > CANVAS_WIDTH && cursorX > START_X) {
      cursorX = START_X;
      cursorY += rowMaxHeight + GUTTER;
      rowMaxHeight = 0;
    }

    merged.push({
      id: `${el.type}_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
      type: el.type as ReportElement['type'],
      section: REPORT_SECTIONS.BODY,
      position: { x: cursorX, y: cursorY },
      size,
      properties: {
        name: el.title || `${el.type.charAt(0).toUpperCase()}${el.type.slice(1)} 1`,
        fontSize: 12,
        columns: el.type === ELEMENT_TYPES.TABLE || el.type === ELEMENT_TYPES.MATRIX || el.type === ELEMENT_TYPES.LIST ? [] : undefined,
        boKey: el.boKey,
        ...dataBinding,
      },
    });

    cursorX += size.width + GUTTER;
    rowMaxHeight = Math.max(rowMaxHeight, size.height);
  }

  return merged;
}
