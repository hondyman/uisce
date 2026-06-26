// React default import removed — using automatic JSX runtime
import { Handle, NodeProps, Position } from 'reactflow';

interface Column {
  id: string;
  name: string;
  assetType: string;
}

const ParentNode: React.FC<NodeProps> = ({ data }) => {
  // Define background colors for the body based on assetType
  const backgroundColors: { [key: string]: string } = {
    cube: '#fbe9e7', // Light peach
    table: '#e8f5e9', // Light green
    default: '#f9f9f9', // Light grey
  };

  const backgroundColor = backgroundColors[data.assetType] || backgroundColors.default;

  const injectedCSS = `
    .parent-node-root[data-asset-type="${data.assetType}"]{ --parent-bg: ${backgroundColor}; }
  `;

  return (
    <>
      <style dangerouslySetInnerHTML={{ __html: injectedCSS }} />
      <div className="parent-node-root" data-asset-type={data.assetType}>
      {/* Header */}
      <div className="parent-node-header">
        <div>Name: {data.parentName}</div>
        <div className="parent-node-subtitle">Asset Type: {data.assetType.charAt(0).toUpperCase() + data.assetType.slice(1)}</div>
      </div>

      {/* Content Area */}
      <div className="parent-node-content">
        {data.children.map((child: Column) => (
          <div key={child.id} className="parent-node-child">
            <div className="parent-node-child-title">
              {child.assetType.charAt(0).toUpperCase() + child.assetType.slice(1)}s
            </div>
            <div className="parent-node-child-name">{child.name}</div>
            <div className="parent-node-handles">
              {/* Handles */}
              <Handle type="source" position={Position.Right} id={child.id} className="parent-handle" />
              <Handle type="target" position={Position.Left} id={child.id} className="parent-handle" />
            </div>
          </div>
        ))}
      </div>
    </div>
    </>
  );
};

export default ParentNode;
