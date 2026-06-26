import React from 'react';
import { Box, Typography, Stepper, Step, StepLabel, StepContent, Chip, Paper } from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import PersonIcon from '@mui/icons-material/Person';
import SettingsIcon from '@mui/icons-material/Settings';

// Mock data structure - in reality this would come from the BP instance events
export type TimelineEvent = {
  id: string;
  stepName: string;
  stepType: 'Human' | 'System' | 'LLM' | 'Routing';
  status: 'completed' | 'in_progress' | 'failed' | 'pending';
  timestamp: string;
  actor?: string; // User or System Name
  details?: string;
  llmReasoning?: string;
};

interface TaskTimelineProps {
  events: TimelineEvent[];
}

const getIconForType = (type: TimelineEvent['stepType']) => {
  switch (type) {
    case 'Human': return <PersonIcon fontSize="small" />;
    case 'LLM': return <SmartToyIcon fontSize="small" />;
    case 'System': return <SettingsIcon fontSize="small" />;
    default: return <SettingsIcon fontSize="small" />;
  }
};

const getStepIcon = (status: TimelineEvent['status']) => {
  switch (status) {
    case 'completed': return <CheckCircleIcon color="success" />;
    case 'failed': return <ErrorIcon color="error" />;
    case 'in_progress': return <AccessTimeIcon color="warning" />;
    default: return <RadioButtonUncheckedIcon color="disabled" />;
  }
};

export const TaskTimeline: React.FC<TaskTimelineProps> = ({ events }) => {
  return (
    <Box sx={{ maxWidth: 600 }}>
      <Typography variant="h6" gutterBottom>
        Process Timeline
      </Typography>
      <Stepper orientation="vertical">
        {events.map((event, index) => (
          <Step key={event.id} active={true} expanded={true}>
            <StepLabel
              StepIconComponent={() => getStepIcon(event.status)}
            >
              <Box display="flex" alignItems="center" gap={1}>
                {getIconForType(event.stepType)}
                <Typography variant="subtitle2" fontWeight="bold">
                  {event.stepName}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {new Date(event.timestamp).toLocaleTimeString()}
                </Typography>
              </Box>
            </StepLabel>
            <StepContent>
              <Paper variant="outlined" sx={{ p: 2, bgcolor: 'background.default' }}>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  {event.details}
                </Typography>
                
                {event.actor && (
                   <Chip 
                     label={event.actor} 
                     size="small" 
                     icon={<PersonIcon />} 
                     variant="outlined" 
                     sx={{ mt: 1, mr: 1 }} 
                   />
                )}

                {event.llmReasoning && (
                  <Box mt={1} p={1} bgcolor="action.hover" borderRadius={1} borderLeft={3} borderColor="primary.main">
                     <Box display="flex" alignItems="center" gap={0.5} mb={0.5}>
                        <SmartToyIcon fontSize="inherit" color="primary" />
                        <Typography variant="caption" fontWeight="bold" color="primary">
                            AI Reasoning
                        </Typography>
                     </Box>
                     <Typography variant="body2" fontStyle="italic">
                       "{event.llmReasoning}"
                     </Typography>
                  </Box>
                )}
              </Paper>
            </StepContent>
          </Step>
        ))}
      </Stepper>
    </Box>
  );
};
