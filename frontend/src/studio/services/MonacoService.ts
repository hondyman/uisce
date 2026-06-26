export class MonacoService {
  constructor() {
    this.editor = null
  }

  createEditor(container, options) {
    // Create Monaco editor
    this.editor = { container, options }
  }

  setValue(_value) {
    if (this.editor) {
      // Set editor value
    }
  }
}