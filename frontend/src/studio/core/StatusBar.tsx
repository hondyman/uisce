import React, { useState, useEffect } from 'react'
import { ThemeToggle } from '../theme/ThemeToggle'

export function StatusBar({ kernel }) {
  const [status, setStatus] = useState({
    wasm: "loading",
    workers: 0,
    lint: 0,
    version: "",
    schema: "",
  })

  useEffect(() => {
    kernel.events.on("wasm.ready", () => setStatus(s => ({ ...s, wasm: "ready" })))
    kernel.events.on("pool.updated", n => setStatus(s => ({ ...s, workers: n })))
    kernel.events.on("lintUpdated", w => setStatus(s => ({ ...s, lint: w.length })))
    kernel.events.on("version.loaded", v => setStatus(s => ({ ...s, version: v })))
    kernel.events.on("schema.loaded", sc => setStatus(s => ({ ...s, schema: sc })))
  }, [])

  return (
    <div className="status-bar">
      <span>WASM: {status.wasm}</span>
      <span>Workers: {status.workers}</span>
      <span>Lint: {status.lint}</span>
      <span>Version: {status.version}</span>
      <span>Schema: {status.schema}</span>
      <ThemeToggle kernel={kernel} />
    </div>
  )
}