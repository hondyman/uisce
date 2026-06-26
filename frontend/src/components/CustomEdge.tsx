// React import removed (automatic JSX runtime in use)
import { EdgeProps, getBezierPath } from 'reactflow';

const CustomEdge: React.FC<EdgeProps> = ({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}) => {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
  });

  return (
    <>
      {/* Render the edge path */}
      <path
        id={id}
        className="react-flow__edge-path"
        d={edgePath}
        markerEnd="url(#reactflow__arrowhead)"
      />
      {/* Render the label for predicateName */}
      {data?.edge_type_nameName && (
        <text
          x={labelX}
          y={labelY}
          fill="#333"
          fontSize="12px"
          textAnchor="middle"
          dominantBaseline="middle"
          className="edge-label"
        >
          {data.edge_type_nameName}
        </text>
      )}
    </>
  );
};

export default CustomEdge;
