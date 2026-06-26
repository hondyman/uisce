import { useState, useEffect } from 'react'

export function ContextPanel({ kernel }) {
  const [context, setContext] = useState(kernel.state.context)

  const update = (key, value) => {
    const newCtx = { ...context, [key]: value }
    setContext(newCtx)
    kernel.state.context = newCtx
    kernel.events.dispatch("contextChanged", newCtx)
    kernel.services.persistence.save(kernel)
  }

  return (
    <div className="context-panel">
      <h3>Context</h3>
      {Object.entries(context).map(([k, v]) => (
        <div key={k}>
          <label>{k}</label>
          <input value={v} onChange={e => update(k, e.target.value)} />
        </div>
      ))}
    </div>
  )
}