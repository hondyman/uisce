import { useState, useEffect } from 'react'

interface HealthPanelProps {
  kernel: any;
}

export function HealthPanel({kernel }: HealthPanelProps) {
  const [health, setHealth] = useState<any>(null)

  useEffect(() => {
    kernel.events.on("lintUpdated", async () => {
      const rule = JSON.parse(kernel.state.rule)
      const contexts = kernel.state.contexts
      const h = await kernel.services.health.compute(rule, contexts)
      setHealth(h)
      kernel.events.dispatch("healthUpdated", h)
    })
  }, [])

  return (
    <div className="health-panel">
      <h3>Rule Health</h3>
      <pre>{JSON.stringify(health, null, 2)}</pre>
    </div>
  )
}