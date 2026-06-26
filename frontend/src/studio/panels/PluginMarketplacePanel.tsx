import React, { useState, useEffect } from 'react'

export function PluginMarketplacePanel({ kernel }) {
  const [plugins, setPlugins] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Load available plugins
    loadPlugins()
  }, [])

  const loadPlugins = async () => {
    try {
      // Mock plugin data - in real implementation, this would fetch from a server
      const mockPlugins = [
        {
          id: "advanced-linter",
          name: "Advanced Linter",
          description: "Enhanced linting with custom rules and patterns",
          version: "1.0.0",
          installed: false
        },
        {
          id: "performance-monitor",
          name: "Performance Monitor",
          description: "Real-time performance monitoring and optimization suggestions",
          version: "1.2.0",
          installed: false
        },
        {
          id: "rule-templates",
          name: "Rule Templates",
          description: "Pre-built rule templates for common patterns",
          version: "0.9.0",
          installed: true
        },
        {
          id: "collaboration-tools",
          name: "Collaboration Tools",
          description: "Real-time collaboration and review tools",
          version: "2.1.0",
          installed: false
        }
      ]

      setPlugins(mockPlugins)
    } catch (error) {
      window.notify("Failed to load plugins", "error")
    } finally {
      setLoading(false)
    }
  }

  const installPlugin = async (plugin) => {
    try {
      // Mock installation
      await new Promise(resolve => setTimeout(resolve, 1000))

      setPlugins(plugins.map(p =>
        p.id === plugin.id ? { ...p, installed: true } : p
      ))

      kernel.services.plugins.install(plugin)
      window.notify(`${plugin.name} installed`, "success")
    } catch (error) {
      window.notify(`Failed to install ${plugin.name}`, "error")
    }
  }

  const uninstallPlugin = async (plugin) => {
    try {
      // Mock uninstallation
      await new Promise(resolve => setTimeout(resolve, 500))

      setPlugins(plugins.map(p =>
        p.id === plugin.id ? { ...p, installed: false } : p
      ))

      kernel.services.plugins.uninstall(plugin.id)
      window.notify(`${plugin.name} uninstalled`, "success")
    } catch (error) {
      window.notify(`Failed to uninstall ${plugin.name}`, "error")
    }
  }

  if (loading) {
    return (
      <div className="panel plugin-marketplace-panel">
        <h3>Plugin Marketplace</h3>
        <div className="loading">Loading plugins...</div>
      </div>
    )
  }

  return (
    <div className="panel plugin-marketplace-panel">
      <h3>Plugin Marketplace</h3>

      <div className="plugin-grid">
        {plugins.map(plugin => (
          <div key={plugin.id} className="plugin-card">
            <div className="plugin-header">
              <h4>{plugin.name}</h4>
              <span className="plugin-version">v{plugin.version}</span>
            </div>

            <p className="plugin-description">{plugin.description}</p>

            <div className="plugin-actions">
              {plugin.installed ? (
                <button
                  className="btn btn-secondary"
                  onClick={() => uninstallPlugin(plugin)}
                >
                  Uninstall
                </button>
              ) : (
                <button
                  className="btn btn-primary"
                  onClick={() => installPlugin(plugin)}
                >
                  Install
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}