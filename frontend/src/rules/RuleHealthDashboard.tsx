export function RuleHealthDashboard({ health }) {
  return (
    <div className="rule-health">
      <HealthItem label="Schema" value={health.schema} />
      <HealthItem label="Lint" value={health.lint} />
      <HealthItem label="Migration" value={health.migration} />
      <HealthItem label="Simulation" value={health.simulation} />
    </div>
  )
}

function HealthItem({ label, value }) {
  return (
    <div className="health-item">
      <span>{label}:</span>
      <span className={value ? 'pass' : 'fail'}>{value ? '✓' : '✗'}</span>
    </div>
  )
}