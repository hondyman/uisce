import { useState } from 'react'

export function ExecutionPanel({ kernel }) {
  const [results, setResults] = useState([])

  const run = async () => {
    const bundle = { rules: [JSON.parse(kernel.state.rule)] }
    const contexts = kernel.state.contexts
    const r = await kernel.services.pool.simulateBundle(bundle, contexts)
    setResults(r)
  }

  return (
    <div className="execution-panel">
      <h3>Portfolio Simulation</h3>
      <button onClick={run}>Run</button>
      <pre>{JSON.stringify(results, null, 2)}</pre>
    </div>
  )
}