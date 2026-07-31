import { useState, useEffect } from 'react'

interface MigrationPanelProps {
  kernel: any;
}

export function MigrationPanel({kernel }: MigrationPanelProps) {
  const [migrated, setMigrated] = useState<any>(null)

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