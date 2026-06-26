import React from 'react';
import { Box, Typography, Grid, Paper, Divider } from '@mui/material';
import { ConnectGoogleButton } from './ConnectGoogleButton';
import { CalendarList } from './CalendarList';
import { EventList } from './EventList';
import { ConnectMicrosoftButton } from './ConnectMicrosoftButton';
import { SyncStatus } from './SyncStatus';
import { useCalendarSync } from '../../hooks/useCalendarSync';
import { LoadingSpinner, ErrorAlert } from '../common/LoadingComponents';

interface Props {
  tenantId: string;
  userId: string;
}

export const CalendarDashboard: React.FC<Props> = ({ tenantId, userId }) => {
  const {
    isConnected,
    isLoadingAuth,
    calendars,
    isLoadingCalendars,
    handleConnect,
    handleSync,
    isSyncing,
    syncStatus,
    syncedEvents,
    isLoadingEvents,
    error,
    isLoadingMsAuth,
    isMicrosoftConnected,
    msCalendars,
    isLoadingMsCalendars,
    handleMicrosoftConnect,
    handleMicrosoftSync,
    isMsSyncing,
    msError,
  } = useCalendarSync(tenantId, userId);

  return (
    <Box sx={{ p: 3, maxWidth: 1200, margin: '0 auto' }}>
      <Typography variant="h4" gutterBottom>Calendar Integrations</Typography>
      
      {error && <ErrorAlert error={error} />}
      {msError && <ErrorAlert error={msError} />}

      <Grid container spacing={4}>
        <Grid item xs={12} md={4}>
          <Paper sx={{ p: 3, mb: 3 }}>
            <Typography variant="h6" gutterBottom>Google Calendar</Typography>
            
            <Box mb={3} display="flex" justifyContent="center">
              <ConnectGoogleButton 
                isConnected={!!isConnected} 
                isLoading={isLoadingAuth} 
                onConnect={handleConnect} 
              />
            </Box>

            <Divider sx={{ my: 2 }} />

            {isConnected && (
              <React.Fragment>
                <Typography variant="subtitle1" gutterBottom>Connected Calendars</Typography>
                {isLoadingCalendars ? (
                  <LoadingSpinner message="Loading calendars..." />
                ) : (
                  <CalendarList 
                    calendars={calendars || []} 
                    onSync={handleSync} 
                    isSyncing={!!isSyncing} 
                  />
                )}
              </React.Fragment>
            )}
          </Paper>

          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>Microsoft Calendar</Typography>
            
            <Box mb={3} display="flex" justifyContent="center">
              <ConnectMicrosoftButton 
                isConnected={!!isMicrosoftConnected} 
                isLoading={isLoadingMsAuth} 
                onConnect={handleMicrosoftConnect} 
              />
            </Box>

            <Divider sx={{ my: 2 }} />

            {isMicrosoftConnected && (
              <React.Fragment>
                <Typography variant="subtitle1" gutterBottom>Connected Calendars</Typography>
                {isLoadingMsCalendars ? (
                  <LoadingSpinner message="Loading calendars..." />
                ) : (
                  <CalendarList 
                    calendars={msCalendars || []} 
                    onSync={handleMicrosoftSync} 
                    isSyncing={!!isMsSyncing} 
                  />
                )}
              </React.Fragment>
            )}
          </Paper>
        </Grid>

        <Grid item xs={12} md={8}>
          <SyncStatus status={syncStatus || null} />
          
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>Recent Synced Events</Typography>
            {isLoadingEvents ? (
              <LoadingSpinner message="Loading events..." />
            ) : (
              <EventList events={syncedEvents || []} />
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};
