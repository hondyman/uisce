import React from 'react';
import { Button, CircularProgress } from '@mui/material';

interface Props {
  isConnected: boolean;
  isLoading: boolean;
  onConnect: () => void;
}

export const ConnectGoogleButton: React.FC<Props> = ({ isConnected, isLoading, onConnect }) => {
  if (isLoading) {
    return <CircularProgress size={24} />;
  }

  if (isConnected) {
    return (
      <Button variant="outlined" color="success" disabled>
        Connected to Google Calendar
      </Button>
    );
  }

  return (
    <Button 
      variant="contained" 
      color="primary" 
      onClick={onConnect}
      // You can add a Google icon here
    >
      Connect Google Calendar
    </Button>
  );
};
