import React from 'react';
import { Button, CircularProgress } from '@mui/material';

interface Props {
  isConnected: boolean;
  isLoading: boolean;
  onConnect: () => void;
}

export const ConnectAppleButton: React.FC<Props> = ({ isConnected, isLoading, onConnect }) => {
  if (isLoading) {
    return <CircularProgress size={24} />;
  }

  if (isConnected) {
    return (
      <Button variant="outlined" color="success" disabled>
        Connected to Apple Calendar
      </Button>
    );
  }

  return (
    <Button
      variant="contained"
      sx={{ bgcolor: '#000', color: 'white', '&:hover': { bgcolor: '#333' } }}
      onClick={onConnect}
    >
      Connect Apple Calendar
    </Button>
  );
};
