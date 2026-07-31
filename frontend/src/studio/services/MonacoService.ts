export class MonacoService {
  editor: any;
  constructor() {
    this.editor = null
  }

  createEditor(container: any, options: any) {
    this.editor = { container, options }
  }

  setValue(_value: any) {
    if (this.editor) {
    }
  }
}