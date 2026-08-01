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

  const EditorPanelAny = EditorPanel as any;
  const ContextPanelAny = ContextPanel as any;
  const SimulationPanelAny = SimulationPanel as any;
  const TracePanelAny = TracePanel as any;
  const LintPanelAny = LintPanel as any;
  const HealthPanelAny = HealthPanel as any;
  const DiffPanelAny = DiffPanel as any;
  const ImpactPanelAny = ImpactPanel as any;
  const MigrationPanelAny = MigrationPanel as any;
  const ExecutionPanelAny = ExecutionPanel as any;

  return (
    <div className="rule-studio">
      <EditorPanelAny value={rule} onChange={setRule} />
      <ContextPanelAny context={context} onChange={setContext} />
      <SimulationPanelAny rule={rule} context={context} onTrace={setTrace} />
      <TracePanelAny trace={trace} />
      <LintPanelAny rule={rule} />
      <HealthPanelAny health={health} />
      <DiffPanelAny diffs={diffs} />
      <ImpactPanelAny impact={impact} />
      <MigrationPanelAny rule={rule} />
      <ExecutionPanelAny rule={rule} context={context} />
    </div>
  )
}