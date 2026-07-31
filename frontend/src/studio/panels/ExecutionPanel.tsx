import { useState } from 'react'

interface ExecutionPanelProps {
  kernel: any;
}

export function ExecutionPanel({kernel }: ExecutionPanelProps) {
  const [results, setResults] = useState<any[]>([])

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