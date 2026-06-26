/**
 * Node Type ID to Node Type String Mapping
 * Used throughout the application to convert database IDs to semantic type names
 */

const NODE_TYPE_ID_MAP: Record<string, string> = {
  '21645d21-de5f-4feb-af99-99273ea75626': 'business_term',
  '820b942a-9c9e-4abc-acdc-84616db33098': 'semantic_term',
  '1439f761-606a-44cb-b4f8-7aa6b27a9bf5': 'semantic_column',
  '49a50271-ae58-4d3e-ae1c-2f5b89d89192': 'table',
  'a64c1011-16e8-4ddf-b447-363bf8e15c9a': 'database_column',
};

/**
 * Convert node type ID to node type string name
 * @param nodeTypeId The UUID of the node type
 * @returns The string name of the node type (e.g., 'business_term', 'semantic_term')
 */
export function getNodeTypeFromId(nodeTypeId: string): string {
  return NODE_TYPE_ID_MAP[nodeTypeId] || 'unknown';
}

/**
 * Convert node type string to display name
 * @param nodeType The node type string (e.g., 'business_term')
 * @returns Human-readable display name
 */
export function getNodeTypeDisplayName(nodeType: string): string {
  const displayNames: Record<string, string> = {
    'business_term': 'Business Term',
    'semantic_term': 'Semantic Term',
    'semantic_column': 'Semantic Column',
    'table': 'Table',
    'database_column': 'Database Column',
  };
  return displayNames[nodeType] || 'Unknown';
}

/**
 * Enrich a node object with the node_type field derived from node_type_id
 * @param node The node object with node_type_id
 * @returns The enriched node with node_type field added
 */
export function enrichNodeWithType<T extends { node_type_id?: string; node_type?: string }>(node: T): T & { node_type: string } {
  if (node.node_type) {
    // Already has node_type, just ensure the type includes it
    return node as T & { node_type: string };
  }
  
  const nodeType = node.node_type_id ? getNodeTypeFromId(node.node_type_id) : 'unknown';
  return {
    ...node,
    node_type: nodeType
  } as T & { node_type: string };
}

/**
 * Enrich an array of nodes with node_type field
 * @param nodes Array of node objects
 * @returns Array of enriched nodes
 */
export function enrichNodesWithTypes<T extends { node_type_id?: string; node_type?: string }>(
  nodes: T[]
): (T & { node_type: string })[] {
  return nodes.map(enrichNodeWithType);
}
