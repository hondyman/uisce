import { useState, useEffect } from 'react'

export function SimulationPanel({ kernel }) {
  const [result, setResult] = useState(null)

  useEffect(() => {
    const handler = async () => {
      const rule = JSON.parse(kernel.state.rule)
      const ctx = kernel.state.context
      const res = await kernel.services.pool.evaluate(rule, ctx)
      setResult(res)
      kernel.events.dispatch("simulationComplete", res)
    }

    kernel.events.on("ruleChanged", handler)
    kernel.events.on("contextChanged", handler)

    return () => {}
  }, [])

  return (
    <div className="simulation-panel">
      <h3>Simulation</h3>
      <pre>{JSON.stringify(result, null, 2)}</pre>
    </div>
  )
}