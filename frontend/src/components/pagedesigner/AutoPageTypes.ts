import { PageWidgetDef } from './PageDesignerTypes';

export type RelationshipCardinality = '1:1' | '1:N' | 'N:M';

export interface DiscoveredSubtype {
  subtypeCode: string;
  displayName: string;
  isSatelliteTable: boolean;
  satelliteTable?: string;
  assignedFieldsCount: number;
}

export interface DiscoveredRelationship {
  relKey: string;
  relName: string;
  targetBoKey: string;
  targetBoName: string;
  cardinality: RelationshipCardinality;
  isSubtypeScoped: boolean;
  scopedSubtypeKey?: string | null;
}

export interface AutoPageGenerationRequest {
  tenantId?: string;
  rootBoKey: string;
  bindingId?: string;
  pageGroupTitle: string;
  layoutTopology: 'TABBED_BY_SUBTYPE' | 'SINGLE_SCROLL_PANE' | 'MASTER_DETAIL_SPLIT';
  includeSubtypes: string[]; // Selected subtype codes
  includeRelationships: string[]; // Selected relationship keys
  crudEntitlements: {
    allowCreate: boolean;
    allowUpdate: boolean;
    allowDelete: boolean;
    requiredRoles?: string[];
  };
}

export interface PageGroupSpec {
  pageGroupId: string;
  rootBoKey: string;
  title: string;
  tabs: Array<{
    tabId: string;
    tabTitle: string;
    subtypeCode?: string | null; // null = Core Root BO
    sections: Array<{
      id: string;
      title: string;
      flow: 'ROW' | 'COLUMN';
      widgets: PageWidgetDef[];
    }>;
  }>;
}
