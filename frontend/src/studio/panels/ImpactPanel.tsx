import { useState, useEffect } from 'react'

export function ImpactPanel({ kernel }) {
  const [impact, setImpact] = useState([])

  useEffect(() => {
    kernel.events.on("diffComputed", async () => {
      const oldRule = kernel.state.previousRule
      const newRule = JSON.parse(kernel.state.rule)
      const contexts = kernel.state.contexts
      const imp = await kernel.services.impact.compute(oldRule, newRule, contexts)
      setImpact(imp)
      kernel.events.dispatch("impactComputed", imp)
    })
  }, [])

  return (
    <div className="impact-panel">
      <h3>Impact</h3>
      {impact.map((i, idx) => (
        <div key={idx} className="impact-item">
          <strong>Context {i.index}</strong>: {i.before} → {i.after}
        </div>
      ))}
    </div>
  )
}