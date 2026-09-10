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
  reportId?: string,
  tenantId?: string
): Record<string, unknown> {
  const title = def.reportTitle || 'Untitled Report';
  const layoutConfig = {
    elements: def.elements,
    sectionConfig: def.sectionConfig || {},
    layoutSettings: def.layoutSettings || {},
    reportTitle: title,
    parameters: def.parameters || [],
  };

  const payload: Record<string, unknown> = {
    id: reportId,
    name: title,
    template_name: title,
    title: title,
    tenant_id: tenantId || '00000000-0000-0000-0000-000000000000',
    report_key: reportId || `rep-custom-${Date.now()}`,
    layout_config: layoutConfig,
    definition: layoutConfig,
    parameter_schema: {
      parameters: def.parameters || [],
    },
    metadata: {
      version: 2,
      data_bindings: selectedBO ? [{ bo_path: selectedBO.qualifiedPath, alias: selectedBO.alias }] : [],
      sectionConfig: def.sectionConfig || {},
      layoutSettings: def.layoutSettings || {},
      parameters: def.parameters || [],
    },
    elements: def.elements,
    parameters: def.parameters || [],
  };

  return payload;
}
