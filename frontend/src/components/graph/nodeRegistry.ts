import { ComponentType } from 'react';
import { NodeProps } from 'reactflow';
import { CatalogNodeProps } from './CatalogNodeProps';
import { ClusterNode } from './ClusterNode';
import { DefaultCatalogNode } from './DefaultCatalogNode';

export type CatalogNodeComponent = ComponentType<NodeProps<CatalogNodeProps>>;

// Base registry keyed by resolved catalog_type_name. 'cluster' is always
// present — it's a synthetic type the backend emits for grouped fan-outs,
// never a real catalog_node_types row. Any type without an entry here falls
// back to DefaultCatalogNode, so a tenant-added type renders immediately
// (styled from its own type config) without code changes.
const baseNodeRegistry: Record<string, CatalogNodeComponent> = {
  cluster: ClusterNode,
};

// Builds the `nodeTypes` map ReactFlow needs for one CatalogGraph render.
// `overrides` lets a specific view swap in a specialized renderer for a type —
// e.g. the ERD view registers its own table-with-inline-columns renderer for
// 'table' instead of the generic default, without changing any other view.
export function getNodeTypes(overrides?: Record<string, CatalogNodeComponent>): Record<string, CatalogNodeComponent> {
  return { ...baseNodeRegistry, ...overrides };
}

export function resolveNodeComponent(
  nodeType: string,
  overrides?: Record<string, CatalogNodeComponent>
): CatalogNodeComponent {
  return overrides?.[nodeType] || baseNodeRegistry[nodeType] || DefaultCatalogNode;
}
