
export function ExportPanel({ kernel }) {
  const exportRule = () => {
    const blob = new Blob([kernel.state.rule], { type: "application/json" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "rule.json"
    a.click()
    URL.revokeObjectURL(url)
    window.notify("Rule exported", "success")
  }

  const exportBundle = () => {
    const bundle = kernel.state.bundle || { rules: [] }
    const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "rule-bundle.json"
    a.click()
    URL.revokeObjectURL(url)
    window.notify("Bundle exported", "success")
  }

  const exportTrace = () => {
    const trace = kernel.state.trace || []
    const blob = new Blob([JSON.stringify(trace, null, 2)], { type: "application/json" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "rule-trace.json"
    a.click()
    URL.revokeObjectURL(url)
    window.notify("Trace exported", "success")
  }

  const handleImport = (event) => {
    const file = event.target.files[0]
    if (file) {
      const reader = new FileReader()
      reader.onload = (e) => {
        try {
          const content = e.target.result
          kernel.state.rule = content
          kernel.events.dispatch("ruleChanged", content)
          window.notify("Rule imported", "success")
        } catch (error) {
          window.notify("Import failed", "error")
        }
      }
      reader.readAsText(file)
    }
  }

  return (
    <div className="panel export-panel">
      <h3>Export / Import</h3>

      <div className="export-section">
        <h4>Export</h4>
        <div className="export-buttons">
          <button className="btn btn-secondary" onClick={exportRule}>
            Export Rule
          </button>
          <button className="btn btn-secondary" onClick={exportBundle}>
            Export Bundle
          </button>
          <button className="btn btn-secondary" onClick={exportTrace}>
            Export Trace
          </button>
        </div>
      </div>

      <div className="import-section">
        <h4>Import</h4>
        <input
          type="file"
          accept=".json"
          onChange={handleImport}
          style={{ display: 'none' }}
          id="import-file"
        />
        <label htmlFor="import-file" className="btn btn-secondary">
          Import Rule
        </label>
      </div>
    </div>
  )
}