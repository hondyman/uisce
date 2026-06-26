import { useMemo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

interface Column {
  name: string;
  type: string;
  isPrimaryKey?: boolean;
  isForeignKey?: boolean;
  nullable?: boolean;
}

interface TableNodeData {
  label: string;
  schema?: string;
  columns?: Column[];
  showColumns?: boolean;
  highlightedItem?: string | null;
}

const TableNode: React.FC<NodeProps<TableNodeData>> = ({ data, id, selected }) => {
  const { label, columns = [], showColumns = true, highlightedItem } = data;
  
  // Check if this table is highlighted
  const isHighlighted = highlightedItem === `table-${id}`;
  
  // Calculate optimal width based on content
  const nodeWidth = useMemo(() => {
    if (!showColumns || columns.length === 0) {
      // Minimum width based on table name
      return Math.max(180, label.length * 8 + 60);
    }
    
    // Find the longest column name and type
    const longestColumnName = Math.max(...columns.map(col => col.name.length));
    const longestColumnType = Math.max(...columns.map(col => col.type.length));
    
    // Calculate width: icon (20px) + name + padding + type + margins
    const nameWidth = longestColumnName * 7;
    const typeWidth = longestColumnType * 6;
    const totalContentWidth = 20 + nameWidth + 16 + typeWidth + 32;
    
    // Also consider table name width
    const titleWidth = label.length * 8 + 40;
    
    return Math.max(200, Math.min(400, Math.max(totalContentWidth, titleWidth)));
  }, [label, columns, showColumns]);

  // removed unused helper _getColumnClassName to satisfy noUnusedLocals

  const tableClasses = [
    'professional-table-node',
    selected ? 'selected' : '',
    isHighlighted ? 'highlighted' : ''
  ].filter(Boolean).join(' ');

  const nodeStyleId = `table-node-style-${id}`;

  return (
    <div id={nodeStyleId} className={tableClasses}>
      <style>{`#${nodeStyleId} { --node-width: ${nodeWidth}px; }`}</style>
      {/* Connection Handles */}
  <Handle type="target" position={Position.Top} className="handle" />
  <Handle type="source" position={Position.Bottom} className="handle" />
  <Handle type="target" position={Position.Left} className="handle" />
  <Handle type="source" position={Position.Right} className="handle" />
      
      {/* Table Header */}
      <div className="table-header">
        <div className="table-title">{label}</div>
      </div>
      
      {/* Columns Section */}
      {showColumns && columns.length > 0 && (
  <div>
          {columns.map((column, index) => {
            const isColumnHighlighted = highlightedItem === `column-${id}-${index}`;
            const isPK = column.isPrimaryKey;
            const isFK = column.isForeignKey;
            
            const extraClasses: string[] = [];
            if (isColumnHighlighted) extraClasses.push('column-row--highlighted');
            else if (isPK) extraClasses.push('column-row--pk');
            else if (isFK) extraClasses.push('column-row--fk');

              return (
              <div
                key={index}
                className={[ 'column-row', ...extraClasses ].join(' ')}
                title={`${column.name}: ${column.type}${column.nullable === false ? ' NOT NULL' : ''}`}
              >
                {/* Key Icon */}
                <div className="col-key">
                  {isPK && (
                    <span className="col-key--pk">🔑</span>
                  )}
                  {isFK && !isPK && (
                    <span className="col-key--fk">🔗</span>
                  )}
                </div>
                
                {/* Column Details */}
                <div className="column-row-inner">
                  <span className="col-name">{column.name}</span>
                  <span className="col-type">{column.type}</span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default TableNode;