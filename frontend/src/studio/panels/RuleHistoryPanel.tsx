import React, { useState, useEffect } from 'react'

interface RuleHistoryPanelProps {
  kernel: any;
}

interface Version {
  id: string;
  content: string;
  timestamp: string;
  type?: string;
}

export function RuleHistoryPanel({kernel }: RuleHistoryPanelProps) {
  const [versions, setVersions] = useState<Version[]>([])

  useEffect(() => {
    const loadedVersions = kernel.services.persistence.getVersions() || []
    setVersions(loadedVersions)
  }, [])

  const restoreVersion = (version: Version) => {
    kernel.state.rule = version.content
    kernel.events.dispatch('ruleChanged', version.content)
    window.notify?.('Version restored', 'success')
  }

  return (
    <div className="panel rule-history-panel">
      <h3>Rule History</h3>
      {versions.length === 0 ? (
        <p>No versions saved yet</p>
      ) : (
        <div className="history-list">
          {versions.map((v: Version) => (
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