import React, { useState } from 'react';
import { Box, Typography, Tabs, Tab, Paper } from '@mui/material';
import QueryStatsIcon from '@mui/icons-material/QueryStats';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import SettingsIcon from '@mui/icons-material/Settings';

// Placeholder components for each tab
const QueryMonitorView = () => <Typography sx={{ p: 2 }}>Query hit/miss history will be displayed here.</Typography>;
const SuggestionsView = () => <Typography sx={{ p: 2 }}>Pre-aggregation suggestions will be displayed here.</Typography>;
const SettingsView = () => <Typography sx={{ p: 2 }}>Auto-accept and cleanup settings will be configured here.</Typography>;

const TelemetryPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Typography variant="h4" gutterBottom>
        Pre-Aggregation Advisor
      </Typography>
      <Typography paragraph color="text.secondary">
        Monitor query performance, review optimization suggestions, and configure automation to keep your semantic layer fast.
      </Typography>

      <Paper sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={activeTab} onChange={handleTabChange} aria-label="telemetry tabs">
            <Tab label="Monitor" icon={<QueryStatsIcon />} iconPosition="start" />
            <Tab label="Suggestions" icon={<LightbulbIcon />} iconPosition="start" />
            <Tab label="Settings" icon={<SettingsIcon />} iconPosition="start" />
          </Tabs>
        </Box>

        <Box sx={{ flexGrow: 1, overflow: 'auto', p: 2 }}>
          {activeTab === 0 && <QueryMonitorView />}
          {activeTab === 1 && <SuggestionsView />}
          {activeTab === 2 && <SettingsView />}
        </Box>
      </Paper>
    </Box>
  );
};

export default TelemetryPage;