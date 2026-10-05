import type { QuerySubject } from '../../features/analytical-subject';

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

export type BuildSavePayloadOptions = {
  /** PR7 / CUBE-3.3 pinned QuerySubject (business_object | cube). */
  subject?: QuerySubject | null;
};

export function buildSavePayload(
  def: BuilderDefinition,
  selectedBO: BOBinding | null,
  reportId?: string,
  tenantId?: string,
  options?: BuildSavePayloadOptions,
): Record<string, unknown> {
  const title = def.reportTitle || 'Untitled Report';
  const subject = options?.subject ?? null;
  const layoutConfig: Record<string, unknown> = {
    elements: def.elements,
    sectionConfig: def.sectionConfig || {},
    layoutSettings: def.layoutSettings || {},
    reportTitle: title,
    parameters: def.parameters || [],
  };
  // Persist pin on layout so reloads do not depend on name-only cube strings.
  if (subject) {
    layoutConfig.subject = subject;
  }

  const dataBindings =
    selectedBO && subject?.kind !== 'cube'
      ? [{ bo_path: selectedBO.qualifiedPath, alias: selectedBO.alias, boId: selectedBO.boId }]
      : subject?.kind === 'cube'
        ? [{ subject }]
        : selectedBO
          ? [{ bo_path: selectedBO.qualifiedPath, alias: selectedBO.alias, boId: selectedBO.boId }]
          : [];

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
      data_bindings: dataBindings,
      ...(subject ? { subject } : {}),
      sectionConfig: def.sectionConfig || {},
      layoutSettings: def.layoutSettings || {},
      parameters: def.parameters || [],
    },
    elements: def.elements,
    parameters: def.parameters || [],
    primary_business_object_id:
      subject?.kind === 'business_object' ? subject.boId : selectedBO?.boId,
  };

  return payload;
}
