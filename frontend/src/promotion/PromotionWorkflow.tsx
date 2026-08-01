import { useState, useEffect } from 'react'
import { ImpactPanel } from './ImpactPanel'

const diffRules = (oldRule: any, newRule: any): any[] => [];
const analyzeImpact = (oldRule: any, newRule: any, contexts: any): Promise<any[]> => Promise.resolve([] as any[]);

export function PromotionWorkflow({oldRule, newRule, contexts }: any) {
  const [diffs, setDiffs] = useState<any[]>([])
  const [impact, setImpact] = useState<any[]>([])
  const [regressions, setRegressions] = useState<any[]>([])

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