import React, { useState } from 'react'

export function ProfilerPanel({ kernel }) {
  const [profile, setProfile] = useState(null)
  const [running, setRunning] = useState(false)

  const runProfile = async () => {
    setRunning(true)
    try {
      // Simulate profiling
      await new Promise(resolve => setTimeout(resolve, 1000))

      const mockProfile = {
        averageTime: 0.023,
        minTime: 0.015,
        maxTime: 0.045,
        totalRuns: 100,
        slowestContexts: [
          { context: { user: "admin" }, time: 0.045 },
          { context: { user: "user1" }, time: 0.038 }
        ],
        slowestBranches: [
          { path: "root.Group[0]", time: 0.030 },
          { path: "root.Condition", time: 0.025 }
        ]
      }

      setProfile(mockProfile)
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="panel profiler-panel">
      <h3>Performance Profiler</h3>

      <button
        className="btn btn-primary"
        onClick={runProfile}
        disabled={running}
      >
        {running ? 'Profiling...' : 'Run Profile'}
      </button>

      {profile && (
        <div className="profile-results">
          <div className="profile-stats">
            <div className="stat">
              <span className="label">Average Time:</span>
              <span className="value">{profile.averageTime.toFixed(3)}ms</span>
            </div>
            <div className="stat">
              <span className="label">Min Time:</span>
              <span className="value">{profile.minTime.toFixed(3)}ms</span>
            </div>
            <div className="stat">
              <span className="label">Max Time:</span>
              <span className="value">{profile.maxTime.toFixed(3)}ms</span>
            </div>
            <div className="stat">
              <span className="label">Total Runs:</span>
              <span className="value">{profile.totalRuns}</span>
            </div>
          </div>

          <div className="profile-details">
            <h4>Slowest Contexts</h4>
            {profile.slowestContexts.map((ctx, index) => (
              <div key={index} className="profile-item">
                <pre>{JSON.stringify(ctx.context, null, 2)}</pre>
                <span>{ctx.time.toFixed(3)}ms</span>
              </div>
            ))}

            <h4>Slowest Branches</h4>
            {profile.slowestBranches.map((branch, index) => (
              <div key={index} className="profile-item">
                <span>{branch.path}</span>
                <span>{branch.time.toFixed(3)}ms</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}