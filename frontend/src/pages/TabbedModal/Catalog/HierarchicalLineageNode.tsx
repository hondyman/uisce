import React from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Tooltip } from '@mui/material';

interface HierarchicalNodeData {
  label: string;
  tableName?: string;
  nodeType?: string;
  node_type_id?: string;
  parent_name?: string;
  description?: string;
  catalog_type_name?: string;
}

const HierarchicalLineageNode: React.FC<NodeProps<HierarchicalNodeData>> = ({ data }) => {
  const isSemanticTerm = data.node_type_id === '820b942a-9c9e-4abc-acdc-84616db33098';
  const isColumn = data.parent_name; // Has a parent table
  
  // Color scheme
  const semanticColor = '#6366f1'; // Indigo for semantic terms
  const columnColor = '#10b981'; // Green for columns
  const tableColor = '#6b7280'; // Gray for table containers
  
  const backgroundColor = isSemanticTerm ? semanticColor : (isColumn ? columnColor : tableColor);
  
  // Build tooltip content
  const tooltipContent = (
    <div style={{ padding: '4px' }}>
      <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>{data.label}</div>
      {data.catalog_type_name && (
        <div style={{ fontSize: '12px', opacity: 0.9 }}>
          Type: {data.catalog_type_name}
        </div>
      )}
      {data.parent_name && (
        <div style={{ fontSize: '12px', opacity: 0.9 }}>
          Table: {data.parent_name}
        </div>
      )}
      {data.description && (
        <div style={{ fontSize: '12px', marginTop: '4px', opacity: 0.9 }}>
          {data.description}
        </div>
      )}
    </div>
  );
  
  if (isColumn && data.parent_name) {
    // Hierarchical display: table container with column inside
    return (
      <Tooltip title={tooltipContent} arrow placement="top">
        <div style={{
          background: '#ffffff',
          border: `2px solid ${tableColor}`,
          borderRadius: '8px',
          minWidth: '180px',
          boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
          cursor: 'pointer'
        }}>
          {/* Table header */}
          <div style={{
            background: tableColor,
            color: '#ffffff',
            padding: '8px 12px',
            borderTopLeftRadius: '6px',
            borderTopRightRadius: '6px',
            fontSize: '12px',
            fontWeight: '600',
            textTransform: 'uppercase',
            letterSpacing: '0.5px'
          }}>
            📊 {data.parent_name}
          </div>
          
          {/* Column content */}
          <div style={{
            padding: '12px',
            background: '#f9fafb'
          }}>
            <div style={{
              background: columnColor,
              color: '#ffffff',
              padding: '8px 12px',
              borderRadius: '4px',
              fontSize: '14px',
              fontWeight: '500',
              display: 'flex',
              alignItems: 'center',
              gap: '6px'
            }}>
              <span style={{ fontSize: '16px' }}>🔹</span>
              {data.label}
            </div>
          </div>
          
          {/* Connection handles */}
          <Handle type="target" position={Position.Left} style={{ background: columnColor }} />
          <Handle type="source" position={Position.Right} style={{ background: columnColor }} />
        </div>
      </Tooltip>
    );
  }
  
  // Simple node for semantic terms
  return (
    <Tooltip title={tooltipContent} arrow placement="top">
      <div style={{
        background: backgroundColor,
        color: '#ffffff',
        padding: '12px 16px',
        borderRadius: '8px',
        minWidth: '150px',
        boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
        fontSize: '14px',
        fontWeight: '600',
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        border: `2px solid ${backgroundColor}`,
        transition: 'all 0.2s ease',
        cursor: 'pointer'
      }}>
        <span style={{ fontSize: '18px' }}>
          {isSemanticTerm ? '🏷️' : '📄'}
        </span>
        {data.label}
        
        <Handle type="target" position={Position.Left} style={{ background: backgroundColor }} />
        <Handle type="source" position={Position.Right} style={{ background: backgroundColor }} />
      </div>
    </Tooltip>
  );
};

export default HierarchicalLineageNode;
