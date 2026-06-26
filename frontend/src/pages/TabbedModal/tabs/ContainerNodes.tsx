import { useState } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

// Schema Container Node
export const SchemaContainerNode: React.FC<NodeProps> = ({ data, selected }) => {

  return (
    <div className={`schema-container ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="handle-hidden" />
      <Handle type="source" position={Position.Right} className="handle-hidden" />
      
  <div className={`container-header header-schema`}>
        <div className="container-header-left">
          <span>🗄️</span>
          <span>Schema: {data.label}</span>
        </div>
        <div className="container-header-right">{data.expanded ? '−' : '+'}</div>
      </div>
      
      {data.expanded && (
        <div className="container-expanded-info">
          {data.childCount} table{data.childCount !== 1 ? 's' : ''}
        </div>
      )}
      
      <div className="container-footer">{data.qualifiedPath}</div>
    </div>
  );
};

// Table Container Node
export const TableContainerNode: React.FC<NodeProps> = ({ data, selected }) => {

  return (
    <div className={`table-container ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="handle-hidden" />
      <Handle type="source" position={Position.Right} className="handle-hidden" />
      
  <div className={`container-header header-table`}>
        <div className="container-header-left"><span>📋</span><span>{data.label}</span></div>
        <div className="container-header-right">{data.expanded ? '−' : '+'}</div>
      </div>
      
      {data.expanded && (
        <div className="container-expanded-info">{data.childCount} column{data.childCount !== 1 ? 's' : ''}</div>
      )}
      
      <div className="container-footer">{data.schema}</div>
    </div>
  );
};

// Column Node (leaf node)
export const ColumnNode: React.FC<NodeProps> = ({ data, selected }) => {
  const [showTooltip, setShowTooltip] = useState(false);
  
  const nodeClasses = ['column-node'];
  if (data.isCenter) nodeClasses.push('column-node--center');
  if (selected) nodeClasses.push('column-node--selected');
  if (showTooltip) nodeClasses.push('column-node--tooltip');

  return (
    <div 
      className={nodeClasses.join(' ')}
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <Handle type="target" position={Position.Left} className="handle-low" />
      <Handle type="source" position={Position.Right} className="handle-low" />
      
      <div className="column-content">
        <div className="column-content-row">
          <span>🔗</span>
          <span>{data.label}</span>
        </div>
        <div className="column-dataType">{data.dataType}</div>
      </div>

      <div className={`column-tooltip ${showTooltip ? 'visible' : ''}`}>
        <div className="column-tooltip-title">Column</div>
        <div className="column-tooltip-path">{data.qualifiedPath}</div>
        <div className="column-tooltip-type">Type: {data.dataType}</div>
      </div>
    </div>
  );
};

// Export node types object for ReactFlow
export const hierarchicalNodeTypes = {
  schemaContainer: SchemaContainerNode,
  tableContainer: TableContainerNode,
  columnNode: ColumnNode,
};