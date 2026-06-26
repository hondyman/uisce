import React, { useState, useEffect } from 'react'

export function PromotionWorkflowPanel({ kernel }) {
  const [status, setStatus] = useState({
    lint: [],
    impact: [],
    health: { score: 0 }
  })

  useEffect(() => {
    const updateStatus = () => {
      setStatus({
        lint: kernel.state.lint || [],
        impact: kernel.state.impact || [],
        health: kernel.state.health || { score: 0 }
      })
    }

    kernel.events.on("lintUpdated", updateStatus)
    kernel.events.on("impactComputed", updateStatus)
    kernel.events.on("healthUpdated", updateStatus)

    updateStatus()
  }, [])

  const canPromote = () => {
    return status.lint.length === 0 &&
           status.impact.length === 0 &&
           status.health.score >= 80
  }

  const handlePromote = () => {
    if (canPromote()) {
      // Perform promotion
      kernel.events.dispatch("rule.promoted", kernel.state.rule)
      window.notify("Rule promoted successfully!", "success")
    }
  }

  return (
    <div className="panel promotion-workflow-panel">
      <h3>Promotion Workflow</h3>

      <div className="promotion-checklist">
        <div className={`check-item ${status.lint.length === 0 ? 'pass' : 'fail'}`}>
          <span className="check-icon">{status.lint.length === 0 ? '✓' : '✗'}</span>
          <span>No lint errors ({status.lint.length})</span>
        </div>

        <div className={`check-item ${status.impact.length === 0 ? 'pass' : 'fail'}`}>
          <span className="check-icon">{status.impact.length === 0 ? '✓' : '✗'}</span>
          <span>No impact regressions ({status.impact.length})</span>
        </div>

        <div className={`check-item ${status.health.score >= 80 ? 'pass' : 'fail'}`}>
          <span className="check-icon">{status.health.score >= 80 ? '✓' : '✗'}</span>
          <span>Health score ≥ 80 ({status.health.score})</span>
        </div>
      </div>

      <div className="promotion-actions">
        <button
          className="btn btn-primary"
          disabled={!canPromote()}
          onClick={handlePromote}
        >
          {canPromote() ? 'Promote Rule' : 'Cannot Promote'}
        </button>
      </div>
    </div>
  )
}