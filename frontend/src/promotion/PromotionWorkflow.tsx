import { useState, useEffect } from 'react'
import { ImpactPanel } from './ImpactPanel'

export function PromotionWorkflow({oldRule, newRule, contexts }) {
  const [diffs, setDiffs] = useState([])
  const [impact, setImpact] = useState([])
  const [regressions, setRegressions] = useState([])

  useEffect(() => {
    setDiffs(diffRules(oldRule, newRule))
    analyzeImpact(oldRule, newRule, contexts).then(setImpact)
  }, [oldRule, newRule])

  return (
    <div className="promotion-workflow">
      {/* DiffViewer and RegressionPanel not yet implemented; see docs/archives/uisce_frontend-patches */}
      <ImpactPanel diffs={impact} />
    </div>
  )
}