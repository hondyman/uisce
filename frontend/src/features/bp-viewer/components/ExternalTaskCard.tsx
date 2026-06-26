import React from 'react';
import { ExternalTask } from '../types';
import { Card, CardContent, Typography, Link, Chip, Box } from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';

interface ExternalTaskCardProps {
  task: ExternalTask;
}

const buildExternalUrl = (task: ExternalTask): string => {
  if (!task.externalId) return '#';
  switch (task.system) {
    case 'Salesforce':
      return `https://salesforce.com/cases/${task.externalId}`;
    case 'ServiceNow':
      return `https://servicenow.com/incidents/${task.externalId}`;
    case 'Jira':
      return `https://jira.com/browse/${task.externalId}`;
    default:
      return '#';
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case 'resolved': return 'success';
    case 'failed': return 'error';
    case 'in_progress': return 'warning';
    default: return 'default';
  }
};

export const ExternalTaskCard: React.FC<ExternalTaskCardProps> = ({ task }) => {
  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
      <CardContent>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
          <Typography variant="h6" component="div">
            {task.system}
          </Typography>
          <Chip label={task.status} color={getStatusColor(task.status) as any} size="small" />
        </Box>
        
        <Typography color="text.secondary" gutterBottom>
           Action: {task.action}
        </Typography>

        <Typography variant="body2">
          External ID: {task.externalId ?? 'Pending'}
        </Typography>

        {task.llmDecision && (
          <Box mt={1} p={1} bgcolor="action.hover" borderRadius={1}>
            <Typography variant="caption" display="block" gutterBottom>
              <strong>🤖 LLM Routing:</strong>
            </Typography>
            <Typography variant="caption" fontStyle="italic">
              "{task.llmDecision.reason}"
            </Typography>
          </Box>
        )}

        {task.externalId && (
          <Box mt={2}>
            <Link href={buildExternalUrl(task)} target="_blank" rel="noreferrer" sx={{ display: 'flex', alignItems: 'center' }}>
              View in {task.system} <OpenInNewIcon fontSize="small" sx={{ ml: 0.5 }} />
            </Link>
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
