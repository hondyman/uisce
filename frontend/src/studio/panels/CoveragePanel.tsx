import React, { useState, useEffect } from 'react'

export function CoveragePanel({ kernel }) {
  const [coverage, setCoverage] = useState(null)

  useEffect(() => {
    kernel.events.on("traceUpdated", () => {
      const trace = kernel.state.trace || []
      const coverageData = computeCoverage(trace)
      setCoverage(coverageData)
    })
  }, [])

  const computeCoverage = (trace) => {
    // Simple coverage computation
    const covered = new Set()
    const total = new Set()

    // Walk through trace and mark covered nodes
    const walk = (node) => {
      total.add(JSON.stringify(node))
      if (node.executed) {
        covered.add(JSON.stringify(node))
      }
      if (node.children) {
        for (const child of node.children) {
          walk(child)
        }
      }
    }

    for (const item of trace) {
      walk(item)
    }

    return {
      covered: covered.size,
      total: total.size,
      percentage: total.size > 0 ? (covered.size / total.size) * 100 : 0
    }
  }

  if (!coverage) {
    return (
      <div className="panel coverage-panel">
        <h3>Coverage</h3>
        <p>No coverage data available. Run a simulation to see coverage.</p>
      </div>
    )
  }

  return (
    <div className="panel coverage-panel">
      <h3>Coverage</h3>

      <div className="coverage-stats">
        <div className="stat">
          <span className="label">Covered:</span>
          <span className="value">{coverage.covered}</span>
        </div>
        <div className="stat">
          <span className="label">Total:</span>
          <span className="value">{coverage.total}</span>
        </div>
        <div className="stat">
          <span className="label">Percentage:</span>
          <span className="value">{coverage.percentage.toFixed(1)}%</span>
        </div>
      </div>

      <div className="coverage-bar">
        <div
          className="coverage-fill"
          style={{ width: `${coverage.percentage}%` }}
        ></div>
      </div>
    </div>
  )
}