export interface BOBinding {
  qualifiedPath: string;
  alias?: string;
}

interface BuilderDefinition {
  elements: unknown[];
  reportTitle?: string;
  sectionConfig?: Record<string, unknown>;
  layoutSettings?: Record<string, unknown>;
  parameters?: unknown[];
}

export function buildSavePayload(
  def: BuilderDefinition,
  selectedBO: BOBinding | null,
  reportId?: string
): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    name: def.reportTitle || 'Untitled Report',
    report_key: reportId || `rep-custom-${Date.now()}`,
    metadata: {
      version: 2,
      data_bindings: selectedBO ? [{ bo_path: selectedBO.qualifiedPath, alias: selectedBO.alias }] : [],
      sectionConfig: def.sectionConfig || {},
      layoutSettings: def.layoutSettings || {},
    },
    elements: def.elements,
    parameters: def.parameters || [],
  };

  return payload;
}
