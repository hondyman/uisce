// React import removed (automatic JSX runtime in use)
import { Handle, NodeProps, Position } from 'reactflow';
import '../styles/PageNode.css'; // Make sure to create and style this CSS file

const PageNode: React.FC<NodeProps> = ({ data }) => {
  return (
    <div className="page-node">
      <div className="page-header">{data.title}</div>
      {data.description && <div className="page-description">{data.description}</div>}
      {/* Add handles for connections */}
      <Handle
        type="source"
        position={Position.Right}
        id="page-source"
        className="page-handle source-handle"
      />
      <Handle
        type="target"
        position={Position.Left}
        id="page-target"
        className="page-handle target-handle"
      />
    </div>
  );
};

export default PageNode;
