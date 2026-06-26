import React, { useState, useEffect } from 'react'

export function RuleHistoryPanel({ kernel }) {
  const [versions, setVersions] = useState([])

  useEffect(() => {
    // Load versions from persistence
    const loadedVersions = kernel.services.persistence.getVersions() || []
    setVersions(loadedVersions)
  }, [])

  const restoreVersion = (version) => {
    kernel.state.rule = version.content
    kernel.events.dispatch('ruleChanged', version.content)
    window.notify('Version restored', 'success')
  }

  return (
    <div className="panel rule-history-panel">
      <h3>Rule History</h3>
      {versions.length === 0 ? (
        <p>No versions saved yet</p>
      ) : (
        <div className="history-list">
          {versions.map(v => (
            <div key={v.id} className="history-item">
              <div className="history-meta">
                <span className="timestamp">{new Date(v.timestamp).toLocaleString()}</span>
                <span className="type">{v.type || 'manual'}</span>
              </div>
              <button
                className="btn btn-secondary"
                onClick={() => restoreVersion(v)}
              >
                Restore
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}