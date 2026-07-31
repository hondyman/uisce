export class StudioState {
  mode: any;
  rule: any;
  context: any;
  trace: any;
  diffs: any[];
  impact: any[];
  health: any;

  constructor() {
    this.mode = "editing"
    this.rule = ""
    this.context = {}
    this.trace = null
    this.diffs = []
    this.impact = []
    this.health = null
  }

  setMode(mode: any) {
    this.mode = mode
  }
}