import React from 'react';
import { 
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, 
  Paper, Typography, Chip 
} from '@mui/material';
import { SyncedEvent } from '../../types/calendar';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

interface Props {
  events: SyncedEvent[];
}

export const EventList: React.FC<Props> = ({ events }) => {
  if (events.length === 0) {
    return <Typography color="textSecondary">No synced events found.</Typography>;
  }

  return (
    <TableContainer component={Paper}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Title</TableCell>
            <TableCell>Start</TableCell>
            <TableCell>End</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Last Synced</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {events.map((event) => (
            <TableRow key={event.id}>
              <TableCell>{event.title}</TableCell>
              <TableCell>{dayjs(event.start_time).format('MMM D, YYYY h:mm A')}</TableCell>
              <TableCell>{dayjs(event.end_time).format('MMM D, YYYY h:mm A')}</TableCell>
              <TableCell>
                <Chip 
                  size="small" 
                  label={event.status} 
                  color={event.status === 'confirmed' ? 'success' : 'default'} 
                />
              </TableCell>
              <TableCell>{dayjs(event.last_synced_at).fromNow ? dayjs(event.last_synced_at).fromNow() : dayjs(event.last_synced_at).format('MMM D, YYYY')}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};
