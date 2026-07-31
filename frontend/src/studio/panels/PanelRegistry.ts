export class PanelRegistry {
  panels: Map<string, any>;
  constructor() {
    this.panels = new Map()
  }

  register(panel: any) {
    this.panels.set(panel.id, panel)
  }

  get(id: string) {
    return this.panels.get(id)
  }

  list() {
    return [...this.panels.values()]
  }
}