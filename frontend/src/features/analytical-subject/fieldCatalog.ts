import type { SemanticTermView } from '../query-builder/types/queryDef';
import type { CubeDefinition, CubeMetricOption } from '../cubes/types';

/**
 * Build a SemanticTermView-compatible field list from a cube contract.
 * Dimensions keep their catalog termNodeIds; measures use metric definition
 * IDs as termNodeId so CubeRouter measure matching works (metric IDs).
 */
export function buildCubeFieldCatalog(
  cube: CubeDefinition,
  boTerms: SemanticTermView[],
  metrics: CubeMetricOption[],
): SemanticTermView[] {
  const byId = new Map(boTerms.map((t) => [t.termNodeId.toLowerCase(), t]));
  const out: SemanticTermView[] = [];

  for (const dim of cube.dimensions || []) {
    const id = dim.termNodeId;
    if (!id) continue;
    const existing = byId.get(id.toLowerCase());
    if (existing) {
      out.push({
        ...existing,
        role: 'DIMENSION',
        drillPath: dim.drillPath?.length ? dim.drillPath : existing.drillPath,
      });
      continue;
    }
    out.push({
      termNodeId: id,
      termKey: id,
      termName: id,
      displayName: humanize(id),
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
      dataType: 'text',
      drillPath: dim.drillPath,
    });
  }

  if (cube.timeDimension?.termNodeId) {
    const tid = cube.timeDimension.termNodeId;
    if (!out.some((t) => t.termNodeId.toLowerCase() === tid.toLowerCase())) {
      const existing = byId.get(tid.toLowerCase());
      out.push(
        existing
          ? { ...existing, role: 'DIMENSION' }
          : {
              termNodeId: tid,
              termKey: tid,
              termName: tid,
              displayName: humanize(tid),
              role: 'DIMENSION',
              bindingStatus: 'RESOLVED',
              dataType: 'date',
            },
      );
    }
  }

  const metricById = new Map(metrics.map((m) => [m.id.toLowerCase(), m]));
  for (const mid of cube.metricIds || []) {
    const m = metricById.get(mid.toLowerCase());
    out.push({
      termNodeId: mid,
      termKey: m?.name || mid,
      termName: m?.name || mid,
      displayName: m?.name || humanize(mid),
      description: m?.description,
      role: 'MEASURE',
      bindingStatus: 'RESOLVED',
      dataType: 'number',
      defaultAggregation: 'SUM',
    });
  }

  return out;
}

function humanize(raw: string): string {
  const spaced = raw.replace(/_/g, ' ').replace(/([a-z])([A-Z])/g, '$1 $2').trim();
  return spaced.replace(/\b\w/g, (c) => c.toUpperCase());
}
