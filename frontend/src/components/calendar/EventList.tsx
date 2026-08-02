import React from 'react';
import {
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Paper, Typography, Chip
} from '@mui/material';
import type { SyncedEvent } from '../../types/calendar';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

interface Props {
  events: SyncedEvent[];
}

export const EventList: React.FC<Props> = ({ events }) => {
  if (!events || events.length === 0) {
    return <Typography color="textSecondary">No synced events found.</Typography>;
  }

  return (
    <TableContainer component={Paper}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Title</TableCell>
            <TableCell>Provider</TableCell>
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
              <TableCell>
                <Chip size="small" label={event.provider} variant="outlined" />
              </TableCell>
              <TableCell>{dayjs(event.start_time).format('MMM D, YYYY h:mm A')}</TableCell>
              <TableCell>{dayjs(event.end_time).format('MMM D, YYYY h:mm A')}</TableCell>
              <TableCell>
                <Chip
                  size="small"
                  label={event.sync_status}
                  color={event.sync_status === 'synced' ? 'success' : 'default'}
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
