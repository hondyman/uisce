// HoverableNode.tsx - Enhanced node component with hover tooltip and qualified path support
import React, { useState, useMemo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import './HoverableNode.css';

interface HoverableNodeData {
  label: string;
  nodeType: string;
  isCenter?: boolean;
  direction?: 'upstream' | 'downstream';
  description?: string;
  style?: React.CSSProperties;
  width?: number;
  height?: number;
  // Enhanced properties for database columns with qualified paths
  schema?: string;
  table?: string;
  tableName?: string;
  column?: string;
  columnName?: string;
  schemaName?: string;
  qualifiedPath?: string; // e.g., "schema.table.column"
  // Additional metadata
  isHighlighted?: boolean;
  properties?: Record<string, any>;
  columnCount?: number;
  columns?: Array<{
    name: string;
    type: string;
    isCore: boolean;
    nullable?: boolean;
    default?: string;
    schema?: string;
    table?: string;
    qualifiedPath?: string;
  }>;
}

// Color mapping for different node types
const getNodeTypeColor = (nodeType: string): { bg: string; border: string; text: string } => {
  const colorMap: Record<string, { bg: string; border: string; text: string }> = {
    'business_object': { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' }, // Blue
    'business_term': { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' }, // Blue
    'semantic_term': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' }, // Purple
    'semantic_model': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' }, // Purple
    'semantic_view': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' }, // Purple
    'semantic_column': { bg: '#FED7AA', border: '#92400E', text: '#3F2305' }, // Orange
    'database_column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' }, // Green
    'db_column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' }, // Green
    'column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' }, // Green
    'table': { bg: '#F3E8FF', border: '#7E22CE', text: '#3F0F5C' }, // Purple-pink
    'schema': { bg: '#FCE7F3', border: '#BE185D', text: '#500724' }, // Pink
    'database': { bg: '#FEE2E2', border: '#DC2626', text: '#4C0519' }, // Red
    'bo_field': { bg: '#DBEAFE', border: '#0284C7', text: '#001F3F' }, // Light Blue
  };
  return colorMap[nodeType] || { bg: '#F3F4F6', border: '#9CA3AF', text: '#374151' }; // Default gray
};

const getNodeTypeDisplayName = (nodeType: string) => {
  const typeMap = {
    'business_term': 'Business Term',
    'business_object': 'Business Object',
    'semantic_term': 'Semantic Term', 
    'semantic_model': 'Semantic Model',
    'semantic_view': 'Semantic View',
    'semantic_column': 'Semantic Column',
    'database_column': 'Database Column',
    'db_column': 'Database Column',
    'column': 'Column',
    'table': 'Table',
    'schema': 'Schema',
    'database': 'Database',
    'bo_field': 'Business Object Field'
  };
  return typeMap[nodeType as keyof typeof typeMap] || nodeType?.replace('_', ' ') || 'Unknown';
};

const HoverableNode: React.FC<NodeProps<HoverableNodeData>> = ({ data }) => {
  const [showTooltip, setShowTooltip] = useState(false);
  
  // Get colors based on node type
  const nodeColors = useMemo(() => getNodeTypeColor(data.nodeType), [data.nodeType]);
  
  // Determine node type categories
  const isDatabaseColumn = data.nodeType === 'database_column' || data.nodeType === 'db_column' || data.nodeType === 'column';
  const isTable = data.nodeType === 'table';
  const isSemanticColumn = data.nodeType === 'semantic_column';
  
  // Enhanced column info with better qualified path handling
  const columnInfo = useMemo(() => {
    if (!isDatabaseColumn) return null;

    // 1. First try to use the qualifiedPath directly
    if (data.qualifiedPath && data.qualifiedPath.includes('.')) {
      const parts = data.qualifiedPath.split('.');
      if (parts.length >= 3) {
        return { schema: parts[0], table: parts[1], column: parts[2] };
      }
      if (parts.length === 2) {
        return { schema: '', table: parts[0], column: parts[1] };
      }
    }
    
    // 2. Fall back to individual properties if qualifiedPath is not available/valid
    const schema = data.schema || data.properties?.schema || '';
    const table = data.table || data.tableName || data.properties?.table || '';
    const column = data.column || data.columnName || data.properties?.column || data.label;

    if (table && column) {
      return { schema, table, column };
    }

    return null; // Not enough info
  }, [isDatabaseColumn, data]);

  // Enhanced table info for table nodes
  const tableInfo = useMemo(() => {
    if (!isTable) return null;
    
    const schema = data.schema || data.schemaName || '';
    const table = data.tableName || data.label;
    const columnCount = data.columnCount || data.columns?.length || 0;
    
    return { schema, table, columnCount };
  }, [isTable, data]);

  // Enhanced display label based on node type
  const displayLabel = useMemo(() => {
    if (isDatabaseColumn && columnInfo?.table && columnInfo?.column) {
      return `${columnInfo.table}.${columnInfo.column}`;
    }
    if (isTable && tableInfo?.table) {
      return tableInfo.table;
    }
    return data.label;
  }, [isDatabaseColumn, isTable, columnInfo, tableInfo, data.label]);

  // Enhanced tooltip content with qualified paths
  const tooltipContent = useMemo(() => {
    if (isDatabaseColumn && columnInfo) {
      const parts = [columnInfo.schema, columnInfo.table, columnInfo.column].filter(Boolean);
      return parts.join('.');
    }
    if (isTable && tableInfo) {
      const parts = [tableInfo.schema, tableInfo.table].filter(Boolean);
      return parts.join('.');
    }
    if (data.qualifiedPath) {
      return data.qualifiedPath;
    }
    return data.label;
  }, [isDatabaseColumn, isTable, columnInfo, tableInfo, data]);

  // Enhanced description for tooltip
  const tooltipDescription = useMemo(() => {
    if (isDatabaseColumn && columnInfo) {
      return `Database column in ${columnInfo.schema ? `${columnInfo.schema}.` : ''}${columnInfo.table}`;
    }
    if (isTable && tableInfo) {
      const schemaText = tableInfo.schema ? ` in schema ${tableInfo.schema}` : '';
      const columnText = tableInfo.columnCount > 0 ? ` (${tableInfo.columnCount} columns)` : '';
      return `Table${schemaText}${columnText}`;
    }
    if (isSemanticColumn) {
      return `Semantic representation of a database column`;
    }
    return data.description;
  }, [isDatabaseColumn, isTable, isSemanticColumn, columnInfo, tableInfo, data.description]);

  const handleClick = (e: React.MouseEvent) => {
    // For database columns, we want to show the tooltip on hover,
    // but we don't want clicking it to trigger any navigation or selection change.
    if (isDatabaseColumn) {
      e.preventDefault();
      e.stopPropagation();
    }
    // For other nodes, the click will propagate to the onNodeClick handler in ReactFlow.
  };

  const nodeClasses = [
    'hoverable-node',
    showTooltip ? 'hovered' : '',
    data.isHighlighted ? 'highlighted' : '',
    isDatabaseColumn ? 'not-clickable' : (data.isCenter ? 'center-node' : 'clickable')
  ].join(' ');

  const nodeDynamicStyles = {
    ['--node-width' as any]: data.width ? `${data.width}px` : undefined,
    ['--node-height' as any]: data.height ? `${data.height}px` : undefined,
  };

  const contentDynamicStyles = {
    ['--node-border-color' as any]: data.style?.borderColor || nodeColors.border,
    ['--node-background' as any]: data.style?.background || nodeColors.bg,
    ['--node-color' as any]: data.style?.color || nodeColors.text,
    ['--node-font-weight' as any]: data.style?.fontWeight,
  };

  return (
    <div
      className={nodeClasses}
      style={nodeDynamicStyles}
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
      onClick={handleClick}
    >
      {/* Connection handles for React Flow */}
      <Handle 
        type="target" 
        position={Position.Left} 
        className="handle"
      />
      <Handle 
        type="source" 
        position={Position.Right} 
        className="handle"
      />

      {/* Main node content */}
      <div className="node-content" style={contentDynamicStyles}>
        {displayLabel}
      </div>

      {/* Enhanced Hover Tooltip */}
      <div className="node-tooltip">
          <div className="tooltip-title">
            {getNodeTypeDisplayName(data.nodeType)}
          </div>
          <div className={`tooltip-content-path ${tooltipDescription ? 'with-margin' : ''}`}>
            {tooltipContent}
          </div>
          {tooltipDescription && (
            <div className="tooltip-description">
              {tooltipDescription.length > 100 
                ? `${tooltipDescription.substring(0, 100)}...` 
                : tooltipDescription
              }
            </div>
          )}
          {/* Additional metadata for table nodes */}
          {isTable && data.columns && data.columns.length > 0 && (
            <div className="tooltip-meta">
              {data.columns.length} column{data.columns.length !== 1 ? 's' : ''}
            </div>
          )}
          {/* Arrow pointing down */}
          <div className="tooltip-arrow" />
      </div>
    </div>
  );
};

export default HoverableNode;