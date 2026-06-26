import React from 'react';
import { Button, CircularProgress } from '@mui/material';

interface Props {
  isConnected: boolean;
  isLoading: boolean;
  onConnect: () => void;
}

export const ConnectMicrosoftButton: React.FC<Props> = ({ isConnected, isLoading, onConnect }) => {
  if (isLoading) {
    return <CircularProgress size={24} />;
  }

  if (isConnected) {
    return (
      <Button variant="outlined" color="success" disabled>
        Connected to Microsoft Calendar
      </Button>
    );
  }

  return (
    <Button 
      variant="contained" 
      sx={{ bgcolor: '#00a4ef', color: 'white', '&:hover': { bgcolor: '#0078d4' } }}
      onClick={onConnect}
    >
      Connect Microsoft Calendar
    </Button>
  );
};
