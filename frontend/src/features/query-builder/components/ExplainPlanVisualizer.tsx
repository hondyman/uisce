/**
 * Federated Explain Plan visualizer.
 *
 * Renders the backend's FederatedPlan as an interactive React Flow graph.
 * Each PlanNode becomes a node; parent/child relationships become edges.
 * The component is dialect-agnostic: it simply visualizes whatever
 * metadata the backend's explain planner returns (Postgres indexes,
 * StarRocks/partition pruning ratios, etc.).
 */

import React, { useMemo } from 'react';
import ReactFlow, {
  Background,
  Controls,
  Edge,
  Handle,
  Node,
  Position,
  ReactFlowProvider,
  useReactFlow,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Box, Chip, Typography } from '@mui/material';
import SecurityIcon from '@mui/icons-material/Security';
import DialectIcon from './DialectIcon';
import type { FederatedPlan, PlanNode } from '../types/queryDef';

interface Props {
  plan: FederatedPlan;
}

const NODE_WIDTH = 260;
const NODE_HEIGHT = 120;
const LEVEL_GAP = 160;
const SIBLING_GAP = 40;

interface LayoutNode {
  node: PlanNode;
  depth: number;
  x: number;
  y: number;
  parentX?: number;
}

/**
 * Compute a simple layered tree layout.
 *
 * Each level is placed `LEVEL_GAP` pixels below the previous one. Children
 * are centered under their parent when possible, otherwise spread with
 * `SIBLING_GAP` between them.
 */
function layoutTree(root: PlanNode): LayoutNode[] {
  const result: LayoutNode[] = [];

  const measureSubtreeWidth = (node: PlanNode): number => {
    if (!node.children || node.children.length === 0) {
      return NODE_WIDTH;
    }
    const childrenWidth = node.children.reduce(
      (sum, child) => sum + measureSubtreeWidth(child),
      0
    );
    return Math.max(NODE_WIDTH, childrenWidth + (node.children.length - 1) * SIBLING_GAP);
  };

  const place = (node: PlanNode, depth: number, x: number, parentX?: number) => {
    result.push({ node, depth, x, y: depth * LEVEL_GAP, parentX });

    if (node.children && node.children.length > 0) {
      const totalWidth = measureSubtreeWidth(node);
      let childX = x - totalWidth / 2;
      node.children.forEach((child) => {
        const childWidth = measureSubtreeWidth(child);
        place(child, depth + 1, childX + childWidth / 2, x);
        childX += childWidth + SIBLING_GAP;
      });
    }
  };

  place(root, 0, 0);
  return result;
}

const PlanNodeCard: React.FC<{ data: PlanNode }> = ({ data }) => {
  const details = data.details || {};
  const detailEntries = Object.entries(details).slice(0, 3);

  return (
    <Box
      sx={{
        width: NODE_WIDTH,
        minHeight: NODE_HEIGHT,
        bgcolor: 'background.paper',
        border: '1px solid',
        borderColor: data.isSecured ? 'success.main' : 'divider',
        borderRadius: 2,
        p: 1.5,
        boxShadow: 2,
        position: 'relative',
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#90a4ae' }} />
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
        <DialectIcon dialect={data.dataSource} size="small" />
        <Typography variant="caption" color="text.secondary" noWrap>
          {data.dataSource}
        </Typography>
        {data.isSecured && (
          <Chip
            icon={<SecurityIcon fontSize="small" />}
            label="RLS"
            size="small"
            color="success"
            variant="outlined"
            sx={{ height: 18, fontSize: '0.6rem', ml: 'auto' }}
          />
        )}
      </Box>
      <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 0.5 }}>
        {data.nodeType}
      </Typography>
      {data.cost > 0 && (
        <Typography variant="caption" color="text.secondary" display="block">
          Cost: {data.cost.toFixed(2)}
        </Typography>
      )}
      {detailEntries.map(([key, value]) => (
        <Typography
          key={key}
          variant="caption"
          color="text.secondary"
          display="block"
          sx={{ fontFamily: 'monospace', fontSize: '0.65rem' }}
        >
          {key}: {typeof value === 'object' ? JSON.stringify(value) : String(value)}
        </Typography>
      ))}
      <Handle type="source" position={Position.Bottom} style={{ background: '#90a4ae' }} />
    </Box>
  );
};

const nodeTypes = { planNode: PlanNodeCard };

const Flow: React.FC<{ plan: FederatedPlan }> = ({ plan }) => {
  const { fitView } = useReactFlow();

  const { nodes, edges } = useMemo(() => {
    const layout = layoutTree(plan.root);
    const rfNodes: Node<PlanNode>[] = [];
    const rfEdges: Edge[] = [];

    layout.forEach((item, index) => {
      rfNodes.push({
        id: item.node.id || `node-${index}`,
        type: 'planNode',
        position: { x: item.x - NODE_WIDTH / 2, y: item.y },
        data: item.node,
      });

      // Find parent in layout and create edge
      if (item.parentX !== undefined) {
        const parent = layout.find(
          (p) => p.x === item.parentX && p.depth === item.depth - 1
        );
        if (parent) {
          rfEdges.push({
            id: `e-${parent.node.id || `node-${layout.indexOf(parent)}`}-${item.node.id || `node-${index}`}`,
            source: parent.node.id || `node-${layout.indexOf(parent)}`,
            target: item.node.id || `node-${index}`,
            type: 'smoothstep',
            animated: true,
          });
        }
      }
    });

    return { nodes: rfNodes, edges: rfEdges };
  }, [plan]);

  React.useEffect(() => {
    // Defer fitView so React Flow has measured the nodes.
    const timer = setTimeout(() => fitView({ padding: 0.2 }), 50);
    return () => clearTimeout(timer);
  }, [fitView, nodes, edges]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      fitView
      attributionPosition="bottom-left"
    >
      <Background gap={16} size={1} />
      <Controls />
    </ReactFlow>
  );
};

export const ExplainPlanVisualizer: React.FC<Props> = ({ plan }) => {
  if (!plan?.root) {
    return (
      <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
        <Typography>No explain plan available.</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', width: '100%' }}>
      <ReactFlowProvider>
        <Flow plan={plan} />
      </ReactFlowProvider>
    </Box>
  );
};

export default ExplainPlanVisualizer;
