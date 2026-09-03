// Shared props contract for every node renderer registered in nodeRegistry.ts.
// Replaces the untyped `data: any` pattern used by the older per-feature node
// components (BONode, ColumnNode, TableNode, TermNode, CalculationNode, etc.).
export interface CatalogNodeProps {
  id: string;
  type: string;
  label: string;
  properties: Record<string, any>;
  isCluster?: boolean;
  memberIds?: string[];
  onExpandCluster?: (clusterId: string) => void;
}
