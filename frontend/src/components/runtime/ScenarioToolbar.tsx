import React, { useState, useEffect } from 'react';
import { Box, Paper, Typography, FormControl, Select, MenuItem, InputLabel, Chip } from '@mui/material';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import { usePageContextStore } from '../../store/usePageContextStore';

export function ScenarioToolbar() {
  const [scenarios, setScenarios] = useState<any[]>([]);
  const [activeScenarioId, setActiveScenarioId] = useState<string>('');
  const setContextValue = usePageContextStore((state) => state.setContextValue);
  const clearContextValue = usePageContextStore((state) => state.clearContextValue);

  useEffect(() => {
    async function fetchScenarios() {
      try {
        const res = await fetch('/api/simulations/scenarios');
        const data = await res.json();
        setScenarios(data.scenarios || []);
      } catch (err) {
        console.error('Failed to load simulation scenarios', err);
      }
    }
    fetchScenarios();
  }, []);

  const handleChange = (id: string) => {
    setActiveScenarioId(id);
    if (id) {
      const selected = scenarios.find((s) => s.scenario_id === id);
      setContextValue('active_scenario', selected);
    } else {
      clearContextValue('active_scenario');
    }
  };

  return (
    <Paper
      elevation={1}
      sx={{
        p: 1.5,
        mb: 2,
        display: 'flex',
        alignItems: 'center',
        gap: 3,
        backgroundColor: activeScenarioId ? '#14532d' : '#1e293b',
        border: activeScenarioId ? '1px solid #22c55e' : '1px solid #334155',
        color: '#f8fafc',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <TrendingUpIcon sx={{ color: activeScenarioId ? '#4ade80' : '#94a3b8' }} />
        <Typography variant="subtitle2" fontWeight="bold">
          What-If Simulation & Projections Engine
        </Typography>
      </Box>

      <FormControl size="small" sx={{ minWidth: 280, bgcolor: '#0f172a' }}>
        <InputLabel sx={{ color: '#94a3b8' }}>Select Macro Shock / Scenario</InputLabel>
        <Select
          value={activeScenarioId}
          label="Select Macro Shock / Scenario"
          onChange={(e) => handleChange(e.target.value)}
          sx={{ color: '#fff' }}
        >
          <MenuItem value="">
            <em>None (Live Base Data)</em>
          </MenuItem>
          {scenarios.map((s) => (
            <MenuItem key={s.scenario_id} value={s.scenario_id}>
              {s.scenario_name}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      {activeScenarioId && (
        <Chip
          label="Simulation Mode Active — Non-Destructive AST Projections"
          color="success"
          size="small"
          variant="outlined"
          sx={{ color: '#4ade80', borderColor: '#22c55e' }}
        />
      )}
    </Paper>
  );
}
