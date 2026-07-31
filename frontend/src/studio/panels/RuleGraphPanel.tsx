import React, { useState, useEffect } from 'react'

export function RuleGraphPanel({ kernel }: { kernel?: any }) {
  const [graph, setGraph] = useState<{ nodes: any[]; edges: any[] }>({ nodes: [], edges: [] })

  useEffect(() => {
    if (!kernel) return
    kernel.events.on("ruleChanged", () => {
      const rule = JSON.parse(kernel.state.rule || "{}")
      const graphData = buildRuleGraph(rule)
      setGraph(graphData)
    })
  }, [])

  const buildRuleGraph = (rule: any) => {
    const nodes: any[] = []
    const edges: any[] = []
    let nodeId = 0

    const walk = (node: any, parentId: string | null = null) => {
      const id = String(nodeId++)
      nodes.push({
        id,
        label: node.Type || "Unknown",
        data: node
      })

      if (parentId !== null) {
        edges.push({
          from: parentId,
          to: id,
          label: "child"
        })
      }

      if (node.Group && node.Group.Children) {
        for (const child of node.Group.Children) {
          walk(child, id as any)
        }
      }
    }

    if (rule) {
      walk(rule)
    }

    return { nodes, edges }
  }

  return (
    <div className="panel rule-graph-panel">
      <h3>Rule Graph</h3>

      <div className="graph-container">
        <div className="graph-nodes">
          <h4>Nodes ({graph.nodes.length})</h4>
          {graph.nodes.map(node => (
            <div key={node.id} className="graph-node">
              <strong>{node.label}</strong>
              <pre>{JSON.stringify(node.data, null, 2)}</pre>
            </div>
          ))}
        </div>

        <div className="graph-edges">
          <h4>Edges ({graph.edges.length})</h4>
          {graph.edges.map((edge, index) => (
            <div key={index} className="graph-edge">
              {edge.from} → {edge.to} ({edge.label})
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}