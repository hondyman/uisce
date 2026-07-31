import React, { useState, useEffect } from 'react'
import { StudioLayout } from '../layout/StudioLayout'
import { StatusBar } from './StatusBar'
import { CommandPalette } from './CommandPalette'
import { Notifications } from '../notifications/Notifications'
import { SafeModeScreen } from './SafeModeScreen'
import { Onboarding } from '../onboarding'
import { ThemeProvider } from '../theme/ThemeProvider'

interface StudioShellProps {
  kernel: any;
}

export function StudioShell({kernel }: StudioShellProps) {
  const [ready, setReady] = useState(kernel.ready)
  const [safeMode, setSafeMode] = useState(false)

  useEffect(() => {
    kernel.events.on("kernel.ready", () => setReady(true))
    kernel.events.on("kernel.error", () => setSafeMode(true))

    const autosaveInterval = setInterval(() => {
      if (kernel.ready && kernel.state.rule) {
        kernel.services.persistence.save(kernel)
        kernel.services.persistence.saveVersion(kernel.state.rule, "autosave")
      }
    }, 2000)

    const handleKeyDown = (e: any) => {
      if (e.metaKey || e.ctrlKey) {
        switch (e.key) {
          case 's':
            e.preventDefault()
            kernel.services.persistence.save(kernel)
            window.notify?.('Rule saved', 'success')
            break
          case 'p':
            break
          case 'r':
            if (e.shiftKey) {
              e.preventDefault()
              kernel.services.simulation.run(kernel.state.rule, kernel.state.context)
              window.notify?.('Simulation started', 'info')
            }
            break
          case 't':
            if (e.shiftKey) {
              e.preventDefault()
              kernel.events.dispatch('panel.show', 'trace')
            }
            break
          case 'd':
            if (e.shiftKey) {
              e.preventDefault()
              kernel.events.dispatch('panel.show', 'diff')
            }
            break
          case 'i':
            if (e.shiftKey) {
              e.preventDefault()
              kernel.events.dispatch('panel.show', 'impact')
            }
            break
          case 'm':
            if (e.shiftKey) {
              e.preventDefault()
              kernel.events.dispatch('panel.show', 'migration')
            }
            break
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.removeEventListener('keydown', handleKeyDown)
      clearInterval(autosaveInterval)
    }
  }, [])

  if (safeMode) {
    return <SafeModeScreen kernel={kernel} />
  }

  if (!ready) {
    return <LoadingScreen />
  }

  if (!kernel.services.persistence.hasSeenOnboarding()) {
    return <Onboarding kernel={kernel} />
  }

  return (
    <ThemeProvider theme={kernel.services.theme.current}>
      <StudioLayout kernel={kernel} />
      <StatusBar kernel={kernel} />
      <CommandPalette kernel={kernel} />
      <Notifications />
    </ThemeProvider>
  )
}

function LoadingScreen() {
  return (
    <div className="loading-screen">
      <h1>Loading Rule Studio...</h1>
      <div className="spinner"></div>
    </div>
  )
}