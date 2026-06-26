export type ScriptState = 'draft' | 'certified' | 'published' | 'deprecated';

export interface ScriptSummary {
  id: string;
  name: string;
  description?: string;
  state: ScriptState;
  scope?: string;
  latestVersion?: string;
  steward?: string;
  domainTags?: string[];
  updatedAt?: string;
}

export interface ScriptVersionTestInfo {
  pass: boolean;
  notes?: string;
}

export interface ScriptVersionApproval {
  by: string;
  at?: string;
}

export interface ScriptVersion {
  version: string;
  createdAt: string;
  createdBy: string;
  content: string;
  hash: string;
  tests?: ScriptVersionTestInfo;
  approvals?: ScriptVersionApproval[];
}

export interface ScriptLineage {
  attachedTo: string[];
}

export interface ScriptDetail extends ScriptSummary {
  versions: ScriptVersion[];
  lineage: ScriptLineage;
}

export interface ImpactedBundle {
  id: string;
  name: string;
  version: string;
  state: string;
}

export interface ImpactedView {
  bundleId: string;
  bundleName: string;
  name: string;
  state: string;
}

export interface ImpactedObject {
  type: string;
  id: string;
  bundleId?: string;
}

export interface ImpactReport {
  scriptId: string;
  scriptVersion?: string;
  summary?: string;
  impactedBundles: ImpactedBundle[];
  impactedViews: ImpactedView[];
  impactedObjects: ImpactedObject[];
}

export interface ScriptSearchFilters {
  state?: ScriptState;
  scope?: string;
  tag?: string;
  steward?: string;
}
