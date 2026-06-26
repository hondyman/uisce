import React, { useMemo } from 'react'
import ReactFlow, { Background } from 'reactflow'

interface LineageNode {
  id: string
  type: string
  name: string
}

interface LineageEdge {
  from: string
  to: string
  type: string
}

export interface RuleLineageGraphProps {
  lineage: {
    nodes: LineageNode[]
    edges: LineageEdge[]
  }
}

export const RuleLineageGraph: React.FC<RuleLineageGraphProps> = ({ lineage }) => {
  const nodes = useMemo(
    () =>
      lineage.nodes.map((node, index) => ({
        id: node.id,
        data: { label: node.name },
        position: { x: index * 200, y: 0 }
      })),
    [lineage.nodes]
  )

  const edges = useMemo(
    () =>
      lineage.edges.map((edge) => ({
        id: `${edge.from}-${edge.to}`,
        source: edge.from,
        target: edge.to,
        label: edge.type
      })),
    [lineage.edges]
  )

  return (
    <div style={{ width: '100%', height: 300 }}>
      <ReactFlow nodes={nodes} edges={edges} fitView>
        <Background />
      </ReactFlow>
    </div>
  )
}
