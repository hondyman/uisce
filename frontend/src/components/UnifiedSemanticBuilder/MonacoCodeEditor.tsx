import MonacoCodeEditorLazy from './MonacoCodeEditor.lazy';

// Re-export helpers so tests can import from the same module path.
export { computeQuickFixActions, convertActionsToMonacoEdits } from './MonacoCodeEditor.impl';

// Default runtime export is the lazy-loaded Monaco implementation.
export default MonacoCodeEditorLazy;
