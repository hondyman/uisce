export function MigrationPreview({ rule }) {
  const migrated = migrateRule(rule)
  return (
    <div>
      <h3>Migration Preview</h3>
      <DiffViewer diffs={diffRules(rule, migrated)} />
    </div>
  )
}