import { CommandRegistry } from '../commands/CommandRegistry'

export class CommandService {
  constructor() {
    this.registry = new CommandRegistry()
    this.registerDefaults()
  }

  registerDefaults() {
    this.registry.register({
      id: "save",
      title: "Save Rule",
      run: (kernel) => kernel.services.persistence.save(kernel)
    })

    this.registry.register({
      id: "format",
      title: "Format Rule",
      run: (kernel) => kernel.services.lint.format(kernel.state.rule)
    })

    this.registry.register({
      id: "simulate",
      title: "Run Simulation",
      run: (kernel) => kernel.services.simulation.run(kernel.state.rule, kernel.state.context)
    })

    this.registry.register({
      id: "toggle-theme",
      title: "Toggle Theme",
      run: (kernel) => {
        const current = kernel.services.theme.current
        const next = current === "dark" ? "light" : "dark"
        kernel.services.theme.setTheme(next)
      }
    })
  }

  search(query) {
    return this.registry.search(query)
  }

  execute(id, kernel) {
    return this.registry.execute(id, kernel)
  }
}