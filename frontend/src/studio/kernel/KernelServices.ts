import { WasmService } from "../services/WasmService"
import { WorkerPoolService } from "../services/WorkerPoolService"
import { MonacoService } from "../services/MonacoService"
import { LintService } from "../services/LintService"
import { MigrationService } from "../services/MigrationService"
import { DiffService } from "../services/DiffService"
import { TraceService } from "../services/TraceService"
import { ImpactService } from "../services/ImpactService"
import { HealthService } from "../services/HealthService"
import { PersistenceService } from "../services/PersistenceService"
import { TelemetryService } from "../services/TelemetryService"
import { PluginService } from "../services/PluginService"
import { ThemeService } from "../services/ThemeService"
import { CommandService } from "../services/CommandService"
import { SimulationService } from "../services/SimulationService"

export class KernelServices {
  constructor() {
    this.wasm = new WasmService()
    this.pool = new WorkerPoolService()
    this.monaco = new MonacoService()
    this.lint = new LintService()
    this.migration = new MigrationService()
    this.diff = new DiffService()
    this.trace = new TraceService()
    this.impact = new ImpactService()
    this.health = new HealthService()
    this.persistence = new PersistenceService()
    this.telemetry = new TelemetryService()
    this.plugins = new PluginService()
    this.theme = new ThemeService()
    this.commands = new CommandService()
    this.simulation = new SimulationService()
  }
}