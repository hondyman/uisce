export function DebuggerPanel({ trace }) {
  return (
    <div className="debugger-panel">
      <h3>Rule Debugger</h3>
      <pre>{JSON.stringify(trace, null, 2)}</pre>
    </div>
  )
}