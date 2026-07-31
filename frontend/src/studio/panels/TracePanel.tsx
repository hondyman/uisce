import { useState, useEffect } from 'react'

interface TracePanelProps {
  kernel: any;
}

interface TraceNodeType {
  passed: boolean;
  nodeType: string;
  details?: any;
  children?: TraceNodeType[];
}

export function TracePanel({kernel }: TracePanelProps) {
  const [trace, setTrace] = useState<TraceNodeType | null>(null)

  useEffect(() => {
    kernel.events.on("simulationComplete", async () => {
      const rule = JSON.parse(kernel.state.rule)
      const ctx = kernel.state.context
      const t = await kernel.services.trace.run(rule, ctx)
      setTrace(t)
      kernel.events.dispatch("traceUpdated", t)
    })
  }, [])

  return (
    <div className="trace-panel">
      <h3>Trace</h3>
      {trace && <TraceNode node={trace} />}
    </div>
  )
}

function TraceNode({node }: { node: TraceNodeType }) {
  return (
    <div className={`trace-node ${node.passed ? "pass" : "fail"}`}>
      <div className="header">
        {node.nodeType} — {node.passed ? "✔" : "✘"}
      </div>
      {node.details && (
        <pre className="details">{JSON.stringify(node.details, null, 2)}</pre>
      )}
      {node.children?.map((c: TraceNodeType, i: number) => (
        <TraceNode key={i} node={c} />
      ))}
    </div>
  )
}