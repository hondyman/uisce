import { EditorPanel } from '../panels/EditorPanel'
import { SimulationPanel } from '../panels/SimulationPanel'
import { TracePanel } from '../panels/TracePanel'
import { LintPanel } from '../panels/LintPanel'
import { HealthPanel } from '../panels/HealthPanel'
import { ContextPanel } from '../panels/ContextPanel'
import { DiffPanel } from '../panels/DiffPanel'
import { ImpactPanel } from '../panels/ImpactPanel'
import { MigrationPanel } from '../panels/MigrationPanel'
import { ExecutionPanel } from '../panels/ExecutionPanel'
import { RuleHistoryPanel } from '../panels/RuleHistoryPanel'
import { BundleManagerPanel } from '../panels/BundleManagerPanel'
import { RuleGraphPanel } from '../panels/RuleGraphPanel'
import { CoveragePanel } from '../panels/CoveragePanel'
import { ProfilerPanel } from '../panels/ProfilerPanel'
import { PromotionWorkflowPanel } from '../panels/PromotionWorkflowPanel'
import { ExportPanel } from '../panels/ExportPanel'
import { PluginMarketplacePanel } from '../panels/PluginMarketplacePanel'

export class PluginService {
  constructor() {
    this.plugins = []
    this.available = this.getAvailablePlugins()
    this.registerCorePanels()
  }

  registerCorePanels() {
    const corePanels = [
      { id: 'simulation', component: SimulationPanel },
      { id: 'trace', component: TracePanel },
      { id: 'lint', component: LintPanel },
      { id: 'health', component: HealthPanel },
      { id: 'context', component: ContextPanel },
      { id: 'diff', component: DiffPanel },
      { id: 'impact', component: ImpactPanel },
      { id: 'migration', component: MigrationPanel },
      { id: 'execution', component: ExecutionPanel },
      { id: 'history', component: RuleHistoryPanel },
      { id: 'bundle', component: BundleManagerPanel },
      { id: 'graph', component: RuleGraphPanel },
      { id: 'coverage', component: CoveragePanel },
      { id: 'profiler', component: ProfilerPanel },
      { id: 'promotion', component: PromotionWorkflowPanel },
      { id: 'export', component: ExportPanel },
      { id: 'marketplace', component: PluginMarketplacePanel }
    ]

    for (const panel of corePanels) {
      this.register(panel)
    }
  }

  async load(kernel) {
    for (const plugin of this.plugins) {
      await plugin.activate?.(kernel)
    }
  }

  register(plugin) {
    this.plugins.push(plugin)
  }

  getPanels() {
    return this.plugins.filter(p => p.component)
  }

  getAvailablePlugins() {
    // Mock available plugins
    return [
      {
        id: "advanced-linter",
        name: "Advanced Linter",
        description: "Enhanced linting with custom rules",
        install: () => {
          // Install logic
        }
      }
    ]
  }

  install(plugin) {
    // Mock installation
    this.register(plugin)
  }

  uninstall(pluginId) {
    this.plugins = this.plugins.filter(p => p.id !== pluginId)
  }
}