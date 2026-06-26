import { Handle, Position } from 'reactflow';

interface DefaultNodeData {
  assetType?: string;
  label?: string;
}

const DefaultNode = ({ data = {} }: { data?: DefaultNodeData }) => {
  const {
    assetType = 'default',
    label = '',
  } = data as DefaultNodeData;

  // background colors intentionally omitted — styling handled via CSS classes

  return (
    <div className="default-node-root">
      {/* Handles */}
      <Handle
        type="target"
        position={Position.Left}
        className="default-node-handle"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="default-node-handle"
      />

      {/* Header */}
      <div className="default-node-header">
        <div>Name: {label}</div>
        <div className="default-node-sub">Asset Type: {assetType.charAt(0).toUpperCase() + assetType.slice(1)}</div>
      </div>

  {/* Content Area */}
  {/* background handled via asset-type classes */}
  <div className={`default-node-content default-node-bg-${assetType}`} />
    </div>
  );
};

export default DefaultNode;
