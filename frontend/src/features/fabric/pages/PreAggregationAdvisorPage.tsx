import React, { useState } from 'react';
import { Box, Typography, Tabs, Tab, Paper } from '@mui/material';
import QueryStatsIcon from '@mui/icons-material/QueryStats';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import SettingsIcon from '@mui/icons-material/Settings';
import CleaningServicesIcon from '@mui/icons-material/CleaningServices';
import MonitorPage from './preaggregations/MonitorPage';
import SuggestionsPage from './preaggregations/SuggestionsPage';
import AutoAcceptConfigPage from './preaggregations/AutoAcceptConfigPage';
import CleanupPage from './preaggregations/CleanupPage';

const PreAggregationAdvisorPage: React.FC = () => {
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
          <Tabs value={activeTab} onChange={handleTabChange} aria-label="pre-aggregation advisor tabs">
            <Tab label="Monitor" icon={<QueryStatsIcon />} iconPosition="start" />
            <Tab label="Suggestions" icon={<LightbulbIcon />} iconPosition="start" />
            <Tab label="Settings" icon={<SettingsIcon />} iconPosition="start" />
            <Tab label="Cleanup" icon={<CleaningServicesIcon />} iconPosition="start" />
          </Tabs>
        </Box>

        <Box sx={{ flexGrow: 1, overflow: 'auto', p: 2 }}>
          {activeTab === 0 && <MonitorPage />}
          {activeTab === 1 && <SuggestionsPage />}
          {activeTab === 2 && <AutoAcceptConfigPage />}
          {activeTab === 3 && <CleanupPage />}
        </Box>
      </Paper>
    </Box>
  );
};

export default PreAggregationAdvisorPage;