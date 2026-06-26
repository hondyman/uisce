import { useState, useEffect } from 'react'

export function TracePanel({ kernel }) {
  const [trace, setTrace] = useState(null)

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

function TraceNode({ node }) {
  return (
    <div className={`trace-node ${node.passed ? "pass" : "fail"}`}>
      <div className="header">
        {node.nodeType} — {node.passed ? "✔" : "✘"}
      </div>
      {node.details && (
        <pre className="details">{JSON.stringify(node.details, null, 2)}</pre>
      )}
      {node.children?.map((c, i) => (
        <TraceNode key={i} node={c} />
      ))}
    </div>
  )
}