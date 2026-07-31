import { useState, useEffect } from 'react'

interface LintPanelProps {
  kernel: any;
}

export function LintPanel({kernel }: LintPanelProps) {
  const [warnings, setWarnings] = useState<any[]>([])

  useEffect(() => {
    kernel.events.on("ruleChanged", () => {
      const rule = JSON.parse(kernel.state.rule)
      const w = kernel.services.lint.run(rule)
      setWarnings(w)
      kernel.events.dispatch("lintUpdated", w)
    })
  }, [])

  return (
    <div className="lint-panel">
      <h3>Lint</h3>
      {warnings.map((w: any, i: number) => (
        <div key={i} className="lint-warning">{w}</div>
      ))}
    </div>
  )
}