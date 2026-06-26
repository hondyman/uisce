import React from 'react';
import { 
  List, ListItem, ListItemText, ListItemSecondaryAction, 
  IconButton, Typography, Paper, Chip 
} from '@mui/material';
import SyncIcon from '@mui/icons-material/Sync';
import { GoogleCalendar } from '../../types/calendar';

interface Props {
  calendars: GoogleCalendar[];
  onSync: (calendarId: string) => void;
  isSyncing: boolean;
}

export const CalendarList: React.FC<Props> = ({ calendars, onSync, isSyncing }) => {
  if (calendars.length === 0) {
    return <Typography color="textSecondary">No calendars found.</Typography>;
  }

  return (
    <Paper elevation={2}>
      <List>
        {calendars.map((calendar) => (
          <ListItem key={calendar.id} divider>
            <ListItemText
              primary={
                <React.Fragment>
                  {calendar.summary}
                  {calendar.primary && (
                    <Chip size="small" label="Primary" color="primary" sx={{ ml: 1 }} />
                  )}
                </React.Fragment>
              }
              secondary={calendar.description || calendar.timezone}
            />
            <ListItemSecondaryAction>
              <IconButton 
                edge="end" 
                aria-label="sync"
                onClick={() => onSync(calendar.id)}
                disabled={isSyncing}
                color="primary"
              >
                <SyncIcon sx={{ animation: isSyncing ? 'spin 2s linear infinite' : 'none' }} />
              </IconButton>
            </ListItemSecondaryAction>
          </ListItem>
        ))}
      </List>
      <style>
        {`
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
        `}
      </style>
    </Paper>
  );
};
