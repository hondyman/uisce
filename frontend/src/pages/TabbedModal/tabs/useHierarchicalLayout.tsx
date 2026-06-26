/* eslint-disable @typescript-eslint/no-unused-vars */
import { useMemo } from 'react';
import { devDebug } from '../../../utils/devLogger';
import { Node, Edge } from 'reactflow';

export interface HierarchicalAsset {
  id: string;
  name: string;
  type: string;
  qualifiedPath?: string;
  schema?: string;
  table?: string;
  column?: string;
}

export interface HierarchicalData {
  nodes: any[];
  edges: any[];
  hierarchy?: Record<string, string[]>;
  metadata?: any;
}

// Custom hook to handle hierarchical layout logic
export const useHierarchicalLayout = (
  selectedAsset: HierarchicalAsset | null,
  hierarchicalData: HierarchicalData | null,
  regularData: any | null
): { nodes: Node[]; edges: Edge[]; isHierarchical: boolean } => {
  
  return useMemo(() => {
    // Determine if we should use hierarchical layout
    const shouldUseHierarchical = selectedAsset && 
      hierarchicalData && 
      hasParentHierarchy(selectedAsset);

    if (shouldUseHierarchical && hierarchicalData) {
  // dev logging
  devDebug('Using hierarchical layout for asset:', selectedAsset);
      return {
        nodes: processHierarchicalNodes(hierarchicalData.nodes, selectedAsset),
        edges: hierarchicalData.edges || [],
        isHierarchical: true,
      };
    }

    // Fallback to regular layout
    if (regularData) {
      return {
        nodes: regularData.nodes || [],
        edges: regularData.edges || [],
        isHierarchical: false,
      };
    }

    return {
      nodes: [],
      edges: [],
      isHierarchical: false,
    };
  }, [selectedAsset, hierarchicalData, regularData]);
};

// Check if the selected asset has a hierarchical path
function hasParentHierarchy(asset: HierarchicalAsset): boolean {
  if (!asset.qualifiedPath) return false;
  
  const parts = asset.qualifiedPath.split('.');
  return parts.length >= 2; // At least schema.table or table.column
}

// Process hierarchical nodes to ensure proper container structure
function processHierarchicalNodes(nodes: any[], selectedAsset: HierarchicalAsset): Node[] {
  if (!nodes || !selectedAsset) return [];

  const processedNodes: Node[] = [];
  const { schema: selectedSchema, table: selectedTable, column: selectedColumn } = parseAssetPath(selectedAsset);

  // Group nodes by their hierarchy level
  const schemaNodes = nodes.filter(n => n.data?.nodeType === 'schema');
  const tableNodes = nodes.filter(n => n.data?.nodeType === 'table');
  const columnNodes = nodes.filter(n => n.data?.nodeType === 'column');

  // Process schema nodes
  schemaNodes.forEach(schemaNode => {
    const isSelectedSchema = schemaNode.data?.label === selectedSchema;
    
    // Always show the schema that contains our selected asset
    if (isSelectedSchema || shouldShowSchema(schemaNode, selectedAsset)) {
      processedNodes.push({
        ...schemaNode,
        data: {
          ...schemaNode.data,
          expanded: isSelectedSchema,
          highlighted: isSelectedSchema,
        }
      });
    }
  });

  // Process table nodes
  tableNodes.forEach(tableNode => {
    const tableSchema = tableNode.data?.schema;
    const isSelectedTable = tableNode.data?.label === selectedTable && tableSchema === selectedSchema;
    const isInSelectedSchema = tableSchema === selectedSchema;
    
    // Show tables in the selected schema, or tables that have relationships
    if (isInSelectedSchema || shouldShowTable(tableNode, selectedAsset)) {
      processedNodes.push({
        ...tableNode,
        data: {
          ...tableNode.data,
          expanded: isSelectedTable,
          highlighted: isSelectedTable,
        },
        parentNode: isInSelectedSchema ? `schema_${tableSchema}` : undefined,
        extent: isInSelectedSchema ? 'parent' : undefined,
      });
    }
  });

  // Process column nodes
  columnNodes.forEach(columnNode => {
    const columnSchema = columnNode.data?.schema;
    const columnTable = columnNode.data?.table;
    const isSelectedColumn = columnNode.data?.label === selectedColumn && 
                            columnTable === selectedTable && 
                            columnSchema === selectedSchema;
    const isInSelectedTable = columnTable === selectedTable && columnSchema === selectedSchema;
    
    // Show columns in the selected table, or columns that have relationships
    if (isInSelectedTable || shouldShowColumn(columnNode, selectedAsset)) {
      processedNodes.push({
        ...columnNode,
        data: {
          ...columnNode.data,
          isCenter: isSelectedColumn,
          highlighted: isSelectedColumn,
        },
        parentNode: isInSelectedTable ? columnNode.parentNode : undefined,
        extent: isInSelectedTable ? 'parent' : undefined,
      });
    }
  });

  return processedNodes;
}

// Parse the asset's qualified path into components
function parseAssetPath(asset: HierarchicalAsset): { schema?: string; table?: string; column?: string } {
  if (asset.schema && asset.table && asset.column) {
    return { schema: asset.schema, table: asset.table, column: asset.column };
  }
  
  if (asset.qualifiedPath) {
    const parts = asset.qualifiedPath.split('.');
    switch (parts.length) {
      case 3:
        return { schema: parts[0], table: parts[1], column: parts[2] };
      case 2:
        return { table: parts[0], column: parts[1] };
      case 1:
        return { column: parts[0] };
    }
  }
  
  return {};
}

// Determine if a schema should be shown (has relationships or contains selected asset)
function shouldShowSchema(_schemaNode: any, _selectedAsset: HierarchicalAsset): boolean {
  // For now, show all schemas - you can add relationship logic here
  return true;
}

// Determine if a table should be shown
function shouldShowTable(_tableNode: any, _selectedAsset: HierarchicalAsset): boolean {
  // Show if it has foreign key relationships or other criteria
  // This would need to be enhanced based on your relationship data
  return true;
}

// Determine if a column should be shown
function shouldShowColumn(_columnNode: any, _selectedAsset: HierarchicalAsset): boolean {
  // Show if it has lineage relationships or other criteria
  return true;
}

// Calculate optimal layout positions
export function calculateHierarchicalPositions(
  nodes: Node[], 
  _hierarchy: Record<string, string[]>
): Node[] {
  const positioned = [...nodes];
  const schemaSpacing = 600;
  const tableSpacing = 350;
  const columnSpacing = 140;
  
  let schemaIndex = 0;
  
  positioned.forEach(node => {
    const data = node.data;
    
    if (data?.nodeType === 'schema') {
      // Position schemas horizontally with spacing
      node.position = {
        x: schemaIndex * schemaSpacing,
        y: 0
      };
      schemaIndex++;
      
    } else if (data?.nodeType === 'table') {
      // Position tables within their schema
      const parentSchema = positioned.find(n => n.id === node.parentNode);
      if (parentSchema) {
        const siblingTables = positioned.filter(n => 
          n.data?.nodeType === 'table' && n.parentNode === parentSchema.id
        );
        const tableIndex = siblingTables.indexOf(node);
        
        node.position = {
          x: parentSchema.position.x + 40 + (tableIndex * tableSpacing),
          y: parentSchema.position.y + 80
        };
      }
      
    } else if (data?.nodeType === 'column') {
      // Position columns within their table
      const parentTable = positioned.find(n => n.id === node.parentNode);
      if (parentTable) {
        const siblingColumns = positioned.filter(n => 
          n.data?.nodeType === 'column' && n.parentNode === parentTable.id
        );
        const columnIndex = siblingColumns.indexOf(node);
        
        node.position = {
          x: parentTable.position.x + 20 + (columnIndex * columnSpacing),
          y: parentTable.position.y + 60
        };
      }
    }
  });
  
  return positioned;
}

// Helper to expand/collapse hierarchy levels
export function updateNodeExpansion(
  nodes: Node[], 
  nodeId: string, 
  expanded: boolean
): Node[] {
  return nodes.map(node => {
    if (node.id === nodeId) {
      return {
        ...node,
        data: { ...node.data, expanded }
      };
    }
    
    // Hide/show child nodes based on parent expansion
    if (node.parentNode === nodeId) {
      return {
        ...node,
        hidden: !expanded
      };
    }
    
    return node;
  });
}
