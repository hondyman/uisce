import React, { useEffect, useState } from 'react';
import { Box, Tabs, Tab, Container, Typography } from '@mui/material';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  InstanceExplorer, 
  LLMReasoningInspector, 
  RoutingDebugger,
  WorkQueueHeatmap,
  OrchestrationMonitor,
  BPDefinitionBrowser
} from '../index';
import BusinessCenterIcon from '@mui/icons-material/BusinessCenter';

const TAB_MAP: Record<string, number> = {
  'instances': 0,
  'monitor': 1,
  'queues': 2,
  'llm': 3,
  'routing': 4,
  'definitions': 5
};

const INDEX_MAP: Record<number, string> = {
  0: 'instances',
  1: 'monitor',
  2: 'queues',
  3: 'llm',
  4: 'routing',
  5: 'definitions'
};

const BPConsolePage: React.FC = () => {
  const { tab } = useParams<{ tab: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState(0);

  useEffect(() => {
    if (tab && TAB_MAP[tab] !== undefined) {
      setActiveTab(TAB_MAP[tab]);
    } else {
      // Default to instances if no tab or invalid tab
      setActiveTab(0);
    }
  }, [tab]);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
    navigate(`/bp-console/${INDEX_MAP[newValue]}`);
  };

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" gutterBottom fontWeight={600} display="flex" alignItems="center">
          <BusinessCenterIcon sx={{ mr: 2, fontSize: 32 }} />
          Business Process Console
        </Typography>
        <Typography variant="body1" color="text.secondary">
          Operations dashboard for monitoring workflows, LLM reasoning, and routing decisions.
        </Typography>
      </Box>

      <Box sx={{ width: '100%', mb: 3 }}>
        <Tabs value={activeTab} onChange={handleTabChange} aria-label="bp console tabs" variant="scrollable" scrollButtons="auto">
          <Tab label="Instance Explorer" />
          <Tab label="Orchestration Monitor" />
          <Tab label="Work Queues" />
          <Tab label="LLM Inspector" />
          <Tab label="Routing Debugger" />
          <Tab label="Definitions" />
        </Tabs>
      </Box>

      <Box sx={{ mt: 2 }}>
        {activeTab === 0 && <InstanceExplorer />}
        {activeTab === 1 && <OrchestrationMonitor />}
        {activeTab === 2 && <WorkQueueHeatmap />}
        {activeTab === 3 && <LLMReasoningInspector />}
        {activeTab === 4 && <RoutingDebugger />}
        {activeTab === 5 && <BPDefinitionBrowser />}
      </Box>
    </Container>
  );
};

export default BPConsolePage;
