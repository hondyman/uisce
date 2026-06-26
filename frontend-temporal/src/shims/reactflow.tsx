import React, { useState, useCallback } from 'react'

export type Node = any

// Minimal useNodesState / useEdgesState hooks compatible with the prototype
export function useNodesState(initial: Node[]) {
  const [nodes, setNodes] = useState<Node[]>(initial)
  const onNodesChange = useCallback((changes: any) => {
    // naive: replace nodes if provided
    if (Array.isArray(changes)) setNodes(changes)
  }, [])
  return [nodes, setNodes, onNodesChange] as const
}

export function useEdgesState(initial: any[]) {
  const [edges, setEdges] = useState<any[]>(initial)
  const onEdgesChange = useCallback((changes: any) => {
    if (Array.isArray(changes)) setEdges(changes)
  }, [])
  return [edges, setEdges, onEdgesChange] as const
}

export function addEdge(params: any, edges: any[]) {
  return [...edges, { id: `e-${edges.length + 1}`, ...params }]
}

export const Background: React.FC = () => null

// Simple ReactFlow component wrapper — renders children inside a container
export default function ReactFlow(props: any) {
  const { nodes, edges, children } = props
  return (
    <div className="rf-container">
      {/* Render a simple visualization of nodes for environments without reactflow */}
      <div className="rf-layer">
        {Array.isArray(nodes) && nodes.map((n: any) => (
          <div key={n.id} className={`rf-node ${n.className || ''}`} style={{ left: `${(n.position?.x || 0)}px`, top: `${(n.position?.y || 0)}px` }}>
            <div className="rf-node-label">{n.data?.label || n.id}</div>
          </div>
        ))}
      </div>
      {children}
    </div>
  )
}

// export a minimal default style import path so code importing 'reactflow/dist/style.css' can be redirected
export const dist = {}
