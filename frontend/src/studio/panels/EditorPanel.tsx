import { useRef, useEffect } from 'react'

export function EditorPanel({ kernel }) {
  const ref = useRef(null)

  useEffect(() => {
    const monaco = kernel.services.monaco.instance
    const editor = monaco.editor.create(ref.current, {
      value: kernel.state.rule,
      language: "asl",
      theme: kernel.services.theme.current,
      automaticLayout: true,
      minimap: { enabled: false },
    })

    editor.onDidChangeModelContent(() => {
      const value = editor.getValue()
      kernel.state.rule = value
      kernel.events.dispatch("ruleChanged", value)
      kernel.services.persistence.save(kernel)
    })

    return () => editor.dispose()
  }, [])

  return <div className="editor-panel" ref={ref} />
}