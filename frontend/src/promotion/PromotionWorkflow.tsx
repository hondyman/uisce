import { useState, useEffect } from 'react'

export function PromotionWorkflow({ oldRule, newRule, contexts }) {
  const [diffs, setDiffs] = useState([])
  const [impact, setImpact] = useState([])
  const [regressions, setRegressions] = useState([])

  useEffect(() => {
    setDiffs(diffRules(oldRule, newRule))
    analyzeImpact(oldRule, newRule, contexts).then(setImpact)
  }, [oldRule, newRule])

  return (
    <div className="promotion-workflow">
      <DiffViewer diffs={diffs} />
      <ImpactPanel diffs={impact} />
      <RegressionPanel regressions={regressions} />
    </div>
  )
}