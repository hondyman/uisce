import { useState, useEffect } from 'react'

export function MigrationPanel({ kernel }) {
  const [migrated, setMigrated] = useState(null)

  useEffect(() => {
    kernel.events.on("ruleChanged", () => {
      const rule = JSON.parse(kernel.state.rule)
      const m = kernel.services.migration.run(rule)
      setMigrated(m)
    })
  }, [])

  return (
    <div className="migration-panel">
      <h3>Migration Preview</h3>
      {migrated && (
        <pre>{JSON.stringify(migrated, null, 2)}</pre>
      )}
    </div>
  )
}