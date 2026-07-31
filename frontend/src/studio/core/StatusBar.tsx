import React, { useState, useEffect } from 'react'
import { ThemeToggle } from '../theme/ThemeToggle'

export function StatusBar({kernel }: {kernel: any}) {
  const [status, setStatus] = useState({
    wasm: "loading",
    workers: 0,
    lint: 0,
    version: "",
    schema: "",
  })

  useEffect(() => {
    kernel.events.on("wasm.ready", () => setStatus((s: any) => ({ ...s, wasm: "ready" })))
    kernel.events.on("pool.updated", (n: any) => setStatus((s: any) => ({ ...s, workers: n })))
    kernel.events.on("lintUpdated", (w: any) => setStatus((s: any) => ({ ...s, lint: w.length })))
    kernel.events.on("version.loaded", (v: any) => setStatus((s: any) => ({ ...s, version: v })))
    kernel.events.on("schema.loaded", (sc: any) => setStatus((s: any) => ({ ...s, schema: sc })))
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