import React from 'react';
import { ExternalTask } from '../types';
import { ExternalTaskCard } from './ExternalTaskCard';
import { Box, Typography } from '@mui/material';

interface ExternalTaskListProps {
  tasks: ExternalTask[];
}

export const ExternalTaskList: React.FC<ExternalTaskListProps> = ({ tasks }) => {
  if (!tasks || tasks.length === 0) {
    return <Typography color="text.secondary">No external tasks found.</Typography>;
  }

  return (
    <Box>
      {tasks.map(task => (
        <ExternalTaskCard key={task.id} task={task} />
      ))}
    </Box>
  );
};
