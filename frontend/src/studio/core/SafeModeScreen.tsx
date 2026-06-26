
export function SafeModeScreen({ kernel }) {
  return (
    <div className="safe-mode">
      <h1>Safe Mode</h1>
      <p>WASM failed to load. You can still edit rules, but simulation is disabled.</p>
      <button onClick={() => kernel.start()}>Retry</button>
    </div>
  )
}