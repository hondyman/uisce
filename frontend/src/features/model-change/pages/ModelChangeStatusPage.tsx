import React, { useState } from 'react';
import { 
  Box, Typography, Paper, Tabs, Tab, Grid, Chip, Button, Alert, Divider
} from '@mui/material';
import { TaskTimeline } from '../../bp-viewer/components/TaskTimeline';
import { ExternalTaskList } from '../../bp-viewer/components/ExternalTaskList';
import { ReportSection } from '../../bp-viewer/components/ReportSection';

// Mock Data representing a live BP instance state
const MOCK_INSTANCE = {
  id: 'MC-2026-000123',
  status: 'In Progress',
  client: 'Jane Doe',
  advisor: 'Alex Smith',
  currentStep: 'Client Approval',
  events: [
    { id: '1', stepName: 'Capture Intent', stepType: 'LLM', status: 'completed', timestamp: '2026-01-01T10:00:00Z', llmReasoning: 'Extracted intent: Growth strategy shift.' },
    { id: '2', stepName: 'Suitability Check', stepType: 'System', status: 'completed', timestamp: '2026-01-01T10:00:05Z', details: 'Passed with Warnings' },
    { id: '3', stepName: 'Advisor Review', stepType: 'Human', status: 'completed', timestamp: '2026-01-01T10:15:00Z', actor: 'Alex Smith' },
    { id: '4', stepName: 'Draft Explanation', stepType: 'LLM', status: 'completed', timestamp: '2026-01-01T10:15:30Z', llmReasoning: 'Generated client-friendly email draft.' },
    { id: '5', stepName: 'Client Approval', stepType: 'Human', status: 'in_progress', timestamp: '2026-01-01T10:16:00Z', details: 'Waiting for Jane Doe...' }
  ],
  externalTasks: [
      { id: 't1', system: 'Salesforce', action: 'create_case', status: 'resolved', externalId: 'CAS-9912', createdAt: '2026-01-01T10:00:00Z' }
  ],
  suitability: {
      status: 'WARNING',
      warnings: ['Asset allocation deviation > 5%', 'Turnover > 20%']
  },
  clientExplanation: "Dear Jane, based on our discussion, I recommend shifting 20% of your portfolio to the Growth strategy to capture...",
  report: null
};

export const ModelChangeStatusPage: React.FC = () => {
  const [tabIndex, setTabIndex] = useState(0);

  return (
    <Box p={3} maxWidth={1400} margin="auto">
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
         <Box>
            <Typography variant="h4">
                Model Change: {MOCK_INSTANCE.client} 
                <Chip label={MOCK_INSTANCE.status} color="warning" sx={{ ml: 2, verticalAlign: 'middle' }} />
            </Typography>
            <Typography color="text.secondary">
                ID: {MOCK_INSTANCE.id} • Advisor: {MOCK_INSTANCE.advisor}
            </Typography>
         </Box>
         <Button variant="outlined">Actions</Button>
      </Box>

      {/* Main Content */}
      <Grid container spacing={3}>
         {/* Left: Detail Tabs */}
         <Grid item xs={12} md={8}>
            <Paper>
                <Tabs value={tabIndex} onChange={(_, v) => setTabIndex(v)} sx={{ borderBottom: 1, borderColor: 'divider' }}>
                    <Tab label="Overview" />
                    <Tab label="Suitability" />
                    <Tab label="Communication" />
                    <Tab label="External & Reports" />
                </Tabs>
                
                {/* Tab 0: Overview */}
                {tabIndex === 0 && (
                    <Box p={3}>
                        <Typography variant="h6" gutterBottom>Impact Summary</Typography>
                        <Grid container spacing={2}>
                            <Grid item xs={4}>
                                <Paper variant="outlined" sx={{ p: 2, textAlign: 'center' }}>
                                    <Typography variant="caption">Risk Delta</Typography>
                                    <Typography variant="h5" color="error">+0.8%</Typography>
                                </Paper>
                            </Grid>
                            <Grid item xs={4}>
                                <Paper variant="outlined" sx={{ p: 2, textAlign: 'center' }}>
                                    <Typography variant="caption">Tax Impact</Typography>
                                    <Typography variant="h5" color="text.secondary">$1,250</Typography>
                                </Paper>
                            </Grid>
                        </Grid>
                    </Box>
                )}

                {/* Tab 1: Suitability */}
                {tabIndex === 1 && (
                    <Box p={3}>
                        {MOCK_INSTANCE.suitability.status === 'WARNING' && (
                             <Alert severity="warning" sx={{ mb: 2 }}>This proposal has suitability warnings that require acknowledgment.</Alert>
                        )}
                        <Typography variant="subtitle1" fontWeight="bold">Detected Issues:</Typography>
                        <ul>
                            {MOCK_INSTANCE.suitability.warnings.map((w, i) => (
                                <li key={i}><Typography variant="body2">{w}</Typography></li>
                            ))}
                        </ul>
                    </Box>
                )}

                 {/* Tab 2: Communication */}
                 {tabIndex === 2 && (
                    <Box p={3}>
                        <Typography variant="h6" gutterBottom>Client Explanation (LLM Drafted)</Typography>
                        <Paper variant="outlined" sx={{ p: 2, bgcolor: 'background.default', fontStyle: 'italic' }}>
                            {MOCK_INSTANCE.clientExplanation}
                        </Paper>
                        <Typography variant="caption" display="block" sx={{ mt: 1, color: 'text.secondary' }}>
                            Generated by Step S4 (Drafting)
                        </Typography>
                    </Box>
                )}

                {/* Tab 3: External */}
                {tabIndex === 3 && (
                     <Box p={3}>
                        <Typography variant="h6" gutterBottom>External Tasks</Typography>
                        <ExternalTaskList tasks={MOCK_INSTANCE.externalTasks as any} />
                        <Divider sx={{ my: 3 }} />
                        <Typography variant="h6" gutterBottom>Reports</Typography>
                        <ReportSection report={MOCK_INSTANCE.report} />
                     </Box>
                )}
            </Paper>
         </Grid>

         {/* Right: Timeline */}
         <Grid item xs={12} md={4}>
            <Paper sx={{ p: 2 }}>
                <TaskTimeline events={MOCK_INSTANCE.events as any} />
            </Paper>
         </Grid>
      </Grid>
    </Box>
  );
};
