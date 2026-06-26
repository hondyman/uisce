
import { useMemo, memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Key, Table } from 'lucide-react';

interface Column {
  name: string;
  type: string;
  isPrimaryKey?: boolean;
  isForeignKey?: boolean;
  nullable?: boolean;
}

interface TableNodeData {
  label: string;
  columns?: Column[];
  showColumns?: boolean;
  highlightedItem?: string | null;
}

const DatabaseTableNode: React.FC<NodeProps<TableNodeData>> = ({ data, id, selected }) => {
  const { label, columns = [], showColumns = true, highlightedItem } = data;

  const isTableHighlighted = highlightedItem === `table-${id}`;

  const nodeDimensions = useMemo(() => {
    const baseWidth = 280;
    const titleWidth = label.length * 9 + 80;
    const maxColumnNameWidth = Math.max(...(columns.map(c => c.name.length) || [0]));
    const maxColumnTypeWidth = Math.max(...(columns.map(c => c.type.length) || [0]));
    const columnsWidth = maxColumnNameWidth * 7 + maxColumnTypeWidth * 6 + 80;
    
    const width = showColumns ? Math.max(baseWidth, titleWidth, columnsWidth) : Math.max(220, titleWidth);
    const height = showColumns && columns.length > 0 ? 60 + (columns.length * 38) : 60;
    
    return { width, height };
  }, [label, columns, showColumns]);

  const styleId = `database-table-style-${id}`;

  return (
    <div id={styleId} className={`database-table-node ${selected ? 'selected' : ''} ${isTableHighlighted ? 'highlighted' : ''}`}>
      <style>{`#${styleId} { --node-width: ${nodeDimensions.width}px; --node-height: ${nodeDimensions.height}px; }`}</style>
      <Handle type="target" position={Position.Top} className="connection-handle" />
      <Handle type="source" position={Position.Bottom} className="connection-handle" />
      <Handle type="target" position={Position.Left} className="connection-handle" />
      <Handle type="source" position={Position.Right} className="connection-handle" />

      <div className="table-header">
        <div className="table-icon"><Table size={16} /></div>
        <div className="table-info">
          <div className="table-name">{label}</div>
          {showColumns && <div className="column-count">{columns.length} columns</div>}
        </div>
      </div>

      {showColumns && columns.length > 0 && (
        <div className="table-columns">
          {columns.map((col, index) => {
            const isColumnHighlighted = highlightedItem === `column-${id}-${index}`;
            const isPK = col.isPrimaryKey;
            const isFK = col.isForeignKey;
            const columnClasses = `table-column 
              ${isPK ? 'primary-key' : ''} 
              ${isFK ? 'foreign-key' : ''} 
              ${isColumnHighlighted ? 'highlighted' : ''}
              ${col.nullable === false ? '' : 'nullable'}`;

            return (
              <div key={index} className={columnClasses} title={`${col.name}: ${col.type}${col.nullable === false ? ' NOT NULL' : ''}`}>
                <div className="column-left">
                  <div className="key-indicator">
                    {(isPK || isFK) && <Key size={12} />}
                  </div>
                  <div className="column-name">{col.name}</div>
                </div>
                <div className="column-right">
                  <div className="column-type">{col.type}</div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default memo(DatabaseTableNode);
