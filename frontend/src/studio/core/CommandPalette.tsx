import React, { useState, useEffect } from 'react'

export function CommandPalette({ kernel }) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [results, setResults] = useState([])

  useEffect(() => {
    const handler = e => {
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

  const handleExecute = (cmd) => {
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