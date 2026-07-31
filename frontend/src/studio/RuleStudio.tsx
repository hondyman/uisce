import { useState } from 'react'
import { EditorPanel } from './panels/EditorPanel'
import { ContextPanel } from './panels/ContextPanel'
import { SimulationPanel } from './panels/SimulationPanel'
import { TracePanel } from './panels/TracePanel'
import { LintPanel } from './panels/LintPanel'
import { HealthPanel } from './panels/HealthPanel'
import { DiffPanel } from './panels/DiffPanel'
import { ImpactPanel } from './panels/ImpactPanel'
import { MigrationPanel } from './panels/MigrationPanel'
import { ExecutionPanel } from './panels/ExecutionPanel'

export function RuleStudio() {
  const [rule, setRule] = useState("")
  const [context, setContext] = useState({})
  const [trace, setTrace] = useState(null)
  const [diffs, setDiffs] = useState([])
  const [impact, setImpact] = useState([])
  const [health, setHealth] = useState(null)

  return (
    <div className="rule-studio">
      <EditorPanel value={rule} onChange={setRule} />
      <ContextPanel context={context} onChange={setContext} />
      <SimulationPanel rule={rule} context={context} onTrace={setTrace} />
      <TracePanel trace={trace} />
      <LintPanel rule={rule} />
      <HealthPanel health={health} />
      <DiffPanel diffs={diffs} />
      <ImpactPanel impact={impact} />
      <MigrationPanel rule={rule} />
      <ExecutionPanel rule={rule} context={context} />
    </div>
  )
}