import { useState, useEffect } from 'react'

interface DiffPanelProps {
  kernel: any;
}

interface DiffItem {
  path: string[];
  type: string;
}

export function DiffPanel({kernel }: DiffPanelProps) {
  const [diffs, setDiffs] = useState<DiffItem[]>([])

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
      {diffs.map((d: DiffItem, i: number) => (
        <div key={i} className="diff-item">
          <strong>{d.path.join(".")}</strong>: {d.type}
        </div>
      ))}
    </div>
  )
}