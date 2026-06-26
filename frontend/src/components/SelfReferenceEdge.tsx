// SelfReferenceEdge.tsx - Custom edge component for self-referencing relationships
// React import removed (automatic JSX runtime in use)
import { EdgeProps, getSmoothStepPath } from 'reactflow';

interface SelfReferenceEdgeData {
  label?: string;
  offset?: number;
  isSelfReference?: boolean;
  originalEdge?: any;
}

const SelfReferenceEdge: React.FC<EdgeProps<SelfReferenceEdgeData>> = ({
  id,
  source,
  target,
  sourceX,
  sourceY,
  targetX,
  targetY,
  data,
  style = {},
}) => {
  // Only render if it's actually a self-reference
  if (source !== target) {
    // Fallback to regular edge if not self-referencing
    const [edgePath] = getSmoothStepPath({
      sourceX,
      sourceY,
      targetX,
      targetY,
    });

    return (
      <path
        id={id}
        d={edgePath}
        fill="none"
        stroke={style.stroke || '#8b5cf6'}
        strokeWidth={style.strokeWidth || 2}
        markerEnd="url(#arrowhead)"
      />
    );
  }

  // Create a circular loop for self-reference
  const radius = 40;
  const offset = data?.offset || 60;
  
  // Position the loop above the node with optional offset for multiple loops
  const centerX = sourceX;
  const centerY = sourceY - offset;
  
  // Create a circular path starting and ending at the node
  const path = `
    M ${sourceX} ${sourceY - 10}
    C ${sourceX + 20} ${sourceY - 30}, ${centerX + radius} ${centerY}, ${centerX} ${centerY - radius}
    C ${centerX - radius} ${centerY}, ${sourceX - 20} ${sourceY - 30}, ${sourceX} ${sourceY - 10}
  `;

  // Label position
  const labelX = centerX;
  const labelY = centerY - radius - 10;

  return (
    <g id={id}>
      {/* Self-reference loop path */}
      <path
        d={path}
        fill="none"
        stroke={style.stroke || '#8b5cf6'}
        strokeWidth={style.strokeWidth || 2}
        strokeDasharray={style.strokeDasharray}
        markerEnd="url(#arrowhead-self)"
      />
      
      {/* Arrow marker specifically for self-reference */}
      <defs>
        <marker
          id="arrowhead-self"
          markerWidth="10"
          markerHeight="7"
          refX="9"
          refY="3.5"
          orient="auto"
        >
          <polygon
            points="0 0, 10 3.5, 0 7"
            fill={style.stroke || '#8b5cf6'}
          />
        </marker>
      </defs>
      
      {/* Edge label */}
      {data?.label && (
        <text
          x={labelX}
          y={labelY}
          textAnchor="middle"
          fontSize="12"
          fill="#6b7280"
          className="react-flow__edge-label react-flow__edge-label--noselect"
        >
          {data.label}
        </text>
      )}
    </g>
  );
};

export default SelfReferenceEdge;