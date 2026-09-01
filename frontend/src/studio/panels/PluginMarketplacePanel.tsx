import React, { useState, useEffect } from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import CircularProgress from '@mui/material/CircularProgress';

export function PluginMarketplacePanel({ kernel }: { kernel?: any }) {
  const theme = useTheme();
  const [plugins, setPlugins] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const isDark = theme.palette.mode === 'dark';

  useEffect(() => {
    loadPlugins()
  }, [])

  const loadPlugins = async () => {
    try {
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
      window.notify?.("Failed to load plugins", "error")
    } finally {
      setLoading(false)
    }
  }

  const installPlugin = async (plugin: any) => {
    try {
      await new Promise(resolve => setTimeout(resolve, 1000))

      setPlugins(plugins.map(p =>
        p.id === plugin.id ? { ...p, installed: true } : p
      ))

      kernel.services.plugins.install(plugin)
      window.notify?.(`${plugin.name} installed`, "success")
    } catch (error) {
      window.notify?.(`Failed to install ${plugin.name}`, "error")
    }
  }

  const uninstallPlugin = async (plugin: any) => {
    try {
      await new Promise(resolve => setTimeout(resolve, 500))

      setPlugins(plugins.map(p =>
        p.id === plugin.id ? { ...p, installed: false } : p
      ))

      kernel.services.plugins.uninstall(plugin.id)
      window.notify?.(`${plugin.name} uninstalled`, "success")
    } catch (error) {
      window.notify?.(`Failed to uninstall ${plugin.name}`, "error")
    }
  }

  if (loading) {
    return (
      <Paper sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
        <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
          Plugin Marketplace
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', flex: 1 }}>
          <CircularProgress size="small" />
          <Typography variant="body2" sx={{ ml: 1 }}>Loading plugins...</Typography>
        </Box>
      </Paper>
    )
  }

  return (
    <Paper sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
        Plugin Marketplace
      </Typography>

      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: 2, overflow: 'auto' }}>
        {plugins.map(plugin => (
          <Paper
            key={plugin.id}
            variant="outlined"
            sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1 }}
          >
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                {plugin.name}
              </Typography>
              <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                v{plugin.version}
              </Typography>
            </Box>

            <Typography variant="body2" sx={{ color: 'text.secondary', flex: 1 }}>
              {plugin.description}
            </Typography>

            <Box sx={{ mt: 1 }}>
              {plugin.installed ? (
                <Button
                  variant="outlined"
                  size="small"
                  fullWidth
                  onClick={() => uninstallPlugin(plugin)}
                >
                  Uninstall
                </Button>
              ) : (
                <Button
                  variant="contained"
                  size="small"
                  fullWidth
                  onClick={() => installPlugin(plugin)}
                >
                  Install
                </Button>
              )}
            </Box>
          </Paper>
        ))}
      </Box>
    </Paper>
  )
}
