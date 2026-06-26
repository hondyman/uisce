export class StudioState {
  constructor() {
    this.mode = "editing"
    this.rule = ""
    this.context = {}
    this.trace = null
    this.diffs = []
    this.impact = []
    this.health = null
  }

  setMode(mode) {
    this.mode = mode
  }
}