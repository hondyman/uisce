import { useState, useEffect } from 'react'

export function DiffPanel({ kernel }) {
  const [diffs, setDiffs] = useState([])

  useEffect(() => {
    kernel.events.on("ruleChanged", () => {
      const oldRule = kernel.state.previousRule
      const newRule = JSON.parse(kernel.state.rule)
      const d = kernel.services.diff.compute(oldRule, newRule)
      setDiffs(d)
      kernel.events.dispatch("diffComputed", d)
    })
  }, [])

  return (
    <div className="diff-panel">
      <h3>Diff</h3>
      {diffs.map((d, i) => (
        <div key={i} className="diff-item">
          <strong>{d.path.join(".")}</strong>: {d.type}
        </div>
      ))}
    </div>
  )
}