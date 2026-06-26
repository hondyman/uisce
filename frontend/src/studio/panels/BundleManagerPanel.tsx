import React, { useState, useEffect } from 'react'

export function BundleManagerPanel({ kernel }) {
  const [bundle, setBundle] = useState({ rules: [] })

  useEffect(() => {
    // Load bundle from kernel state
    if (kernel.state.bundle) {
      setBundle(kernel.state.bundle)
    }
  }, [])

  const addRule = () => {
    const newRule = {
      Type: "Condition",
      Condition: {
        Field: "",
        Operator: "eq",
        Value: ""
      }
    }
    const updatedBundle = {
      ...bundle,
      rules: [...bundle.rules, newRule]
    }
    setBundle(updatedBundle)
    kernel.state.bundle = updatedBundle
    kernel.events.dispatch('bundleChanged', updatedBundle)
  }

  const updateRule = (index, rule) => {
    const updatedBundle = { ...bundle }
    updatedBundle.rules[index] = rule
    setBundle(updatedBundle)
    kernel.state.bundle = updatedBundle
    kernel.events.dispatch('bundleChanged', updatedBundle)
  }

  const removeRule = (index) => {
    const updatedBundle = { ...bundle }
    updatedBundle.rules.splice(index, 1)
    setBundle(updatedBundle)
    kernel.state.bundle = updatedBundle
    kernel.events.dispatch('bundleChanged', updatedBundle)
  }

  return (
    <div className="panel bundle-manager-panel">
      <h3>Rule Bundle</h3>
      <button className="btn btn-primary" onClick={addRule}>
        Add Rule
      </button>

      <div className="bundle-rules">
        {bundle.rules.map((rule, index) => (
          <div key={index} className="bundle-rule">
            <div className="rule-header">
              <span>Rule {index + 1}</span>
              <button
                className="btn btn-secondary"
                onClick={() => removeRule(index)}
              >
                Remove
              </button>
            </div>
            <pre className="rule-json">
              {JSON.stringify(rule, null, 2)}
            </pre>
          </div>
        ))}
      </div>

      {bundle.rules.length > 0 && (
        <div className="bundle-actions">
          <button className="btn btn-primary">
            Validate Bundle
          </button>
          <button className="btn btn-primary">
            Simulate Bundle
          </button>
        </div>
      )}
    </div>
  )
}