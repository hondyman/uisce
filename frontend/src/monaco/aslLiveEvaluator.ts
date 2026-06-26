import { evaluateRuleWasm } from "../rules/wasmRuntime"

export function attachLiveEvaluation(_monaco, editor) {
  let timeout

  editor.onDidChangeModelContent(async () => {
    clearTimeout(timeout)
    timeout = setTimeout(async () => {
      const text = editor.getValue()
      try {
        const rule = JSON.parse(text)
        const result = await evaluateRuleWasm(rule, {})
        showStatus(`Result: ${result}`)
      } catch (e) {
        showStatus(`Error: ${e.message}`)
      }
    }, 300)
  })
}

function showStatus(msg) {
  const el = document.getElementById("asl-status")
  if (el) el.textContent = msg
}