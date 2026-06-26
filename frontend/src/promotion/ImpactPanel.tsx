export function ImpactPanel({ diffs }) {
  return (
    <div className="impact-panel">
      <h3>Promotion Impact</h3>
      {diffs.length === 0 && <p>No impact detected</p>}
      {diffs.map((d, i) => (
        <div key={i} className="impact-item">
          <strong>{d.path.join(".")}</strong>: {d.before} → {d.after}
        </div>
      ))}
    </div>
  )
}