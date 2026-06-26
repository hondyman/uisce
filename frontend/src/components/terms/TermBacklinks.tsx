import React, { useEffect, useState } from 'react'
import { rulesApi } from '../../services/rulesApi'

interface TermBacklinksProps {
  termId: string
}

interface RuleSummary {
  id: string
  name: string
}

export const TermBacklinks: React.FC<TermBacklinksProps> = ({ termId }) => {
  const [rules, setRules] = useState<RuleSummary[]>([])

  useEffect(() => {
    let isMounted = true
    const loadRules = async () => {
      const result = await rulesApi.getRulesForTerm(termId)
      if (isMounted) {
        setRules(result || [])
      }
    }

    loadRules()

    return () => {
      isMounted = false
    }
  }, [termId])

  return (
    <div>
      {rules.map((rule) => (
        <div key={rule.id}>{rule.name}</div>
      ))}
    </div>
  )
}
