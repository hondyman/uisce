// Minimal shim for monaco-yaml used during development to avoid pre-bundling
// resolution issues with the real package and its transitive dependencies.
// The real `monaco-yaml` package is still available in node_modules; this
// shim simply provides a safe no-op API for environments where full
// YAML language server integration is not required.

// keep parameters intentionally unused to act as a safe dev-time shim
export function setDiagnosticsOptions(_monaco: any, _options: any) {
  // no-op: the real monaco-yaml will provide this when available
  // When running in environments that support monaco-yaml, the dynamic
  // import in CodeEditor.tsx will load the real package instead of this shim.
  return;
}

export default { setDiagnosticsOptions };
