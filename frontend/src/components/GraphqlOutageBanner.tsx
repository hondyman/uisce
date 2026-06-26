import React from 'react';
import { Alert, AlertTitle, Button, Box } from '@mui/material';

// show a persistent, dismissible banner that remains while Hasura is down.
// It listens for window 'semlayer.graphqlOutage' events.
export default function GraphqlOutageBanner() {
  const [isDown, setIsDown] = React.useState(false);
  const [dismissed, setDismissed] = React.useState(false);

  React.useEffect(() => {
    const handler = (e: CustomEvent) => {
      const down = !!e.detail?.down;
      setIsDown(down);
      if (!down) {
        // clear dismissal when recovered so banner can appear again if outage occurs later
        setDismissed(false);
      }
    };

    window.addEventListener('semlayer.graphqlOutage', handler as EventListener);

    return () => {
      window.removeEventListener('semlayer.graphqlOutage', handler as EventListener);
    };
  }, []);

  if (!isDown || dismissed) return null;

  return (
    <Box sx={{ position: 'fixed', top: 0, left: 0, right: 0, zIndex: 1400 }}>
      <Alert
        severity="warning"
        action={
          <Button color="inherit" size="small" onClick={() => setDismissed(true)}>
            Dismiss
          </Button>
        }
      >
        <AlertTitle>GraphQL unavailable</AlertTitle>
        A required GraphQL service is unavailable — some features will be disabled until it is back online.
      </Alert>
    </Box>
  );
}
