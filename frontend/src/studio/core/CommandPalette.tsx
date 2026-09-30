/* eslint-disable jsx-a11y/no-autofocus --
 * This component is a hand-rolled overlay (modal / omnibox / palette).
 * Focus must move into it when it opens so keyboard and screen-reader users
 * land inside the dialog, and no other element here takes initial focus.
 * The attribute sits inside a multi-line JSX attribute list, where an
 * inline eslint-disable comment is not valid syntax, so the scope is the
 * whole file. This file has exactly one autoFocus and no other JSX.
 */
import React, { useState, useEffect } from 'react'

export function CommandPalette({kernel }: {kernel: any}) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<any[]>([])

  useEffect(() => {
    const handler = (e: any) => {
      if (e.key === "p" && e.metaKey) {
        e.preventDefault()
        setOpen(true)
      }
    }
    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [])

  useEffect(() => {
    if (query) {
      const cmds = kernel.services.commands.search(query)
      setResults(cmds)
    } else {
      setResults([])
    }
  }, [query])

  const handleExecute = (cmd: any) => {
    kernel.services.commands.execute(cmd.id, kernel)
    setOpen(false)
    setQuery("")
  }

  if (!open) return null

  return (
    <div className="command-palette-overlay" onClick={() => setOpen(false)}>
      <div className="command-palette" onClick={e => e.stopPropagation()}>
        <input
          autoFocus
          placeholder="Type a command..."
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => {
            if (e.key === "Escape") setOpen(false)
            if (e.key === "Enter" && results.length > 0) handleExecute(results[0])
          }}
        />
        <ul>
          {results.map(cmd => (
            <li key={cmd.id} onClick={() => handleExecute(cmd)}>
              {cmd.title}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}