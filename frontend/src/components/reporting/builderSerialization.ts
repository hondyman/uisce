export interface BOBinding {
  qualifiedPath: string;
  alias?: string;
  /**
   * The BO's real id (business_objects.id), used to populate
   * primary_business_object_id on save so it survives reload without
   * relying on the qualifiedPath/bo_path string-match, which is
   * currently unreliable (see the AI report-generation feature's notes:
   * both call sites that build a BOBinding today cast from an object that
   * doesn't actually carry qualifiedPath/alias at runtime, so bo_path has
   * always been undefined for reports created via the normal flow -
   * boId is additive here, not a fix for that separate, pre-existing gap).
   */
  boId?: string;
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
    primary_business_object_id: selectedBO?.boId,
  };

  return payload;
}
