Monaco integration notes

- We use `monaco-editor` dynamically via the ESM entry `monaco-editor/esm/vs/editor/editor.api` so Vite can resolve it.
- Install in CI to ensure consistent builds. Example (in CI job):

  npm ci --legacy-peer-deps
  npm run build

- We installed with `--legacy-peer-deps` locally due to peer dependency ranges; consider pinning compatible React versions in CI or adding a resolution in package-lock.
- To enable Monaco in the app at runtime, toggle the editor in the Code view or set localStorage key `semlayer.preferMonaco=true`.
- For diagnostics to be precise, backend validation should include line/column offsets in the issue payload (fields: `line`, `col`, `endLine`, `endCol`). The frontend maps these to Monaco markers for squiggles and gutter markers.

