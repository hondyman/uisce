import { useState, useEffect } from 'react'

interface ContextPanelProps {
  kernel: any;
}

export function ContextPanel({kernel }: ContextPanelProps) {
  const [context, setContext] = useState<any>(kernel.state.context)

  const update = (key: any, value: any) => {
    const newCtx: any = { ...context, [key]: value }
    setContext(newCtx)
    kernel.state.context = newCtx
    kernel.events.dispatch("contextChanged", newCtx)
    kernel.services.persistence.save(kernel)
  }

  return (
    <div className="context-panel">
      <h3>Context</h3>
      {Object.entries(context).map(([k, v]: [any, any]) => (
        <div key={k}>
          <label>{k}</label>
          <input value={v} onChange={(e: any) => update(k, e.target.value)} />
        </div>
      ))}
    </div>
  )
}