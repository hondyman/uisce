import React, { useEffect, useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  CircularProgress,
  Alert,
  Chip,
  Box,
  Stack,
} from '@mui/material';
import { Delete as DeleteIcon, Edit as EditIcon } from '@mui/icons-material';
import { gql, useQuery, useMutation } from '@apollo/client';

const GET_CALENDARS = gql`
  query GetCalendars {
    calendars(where: { valid_to: { _is_null: true } }) {
      id
      logical_id
      tenant_id
      name
      description
      timezone
      created_at
      updated_at
    }
  }
`;

const DELETE_CALENDAR = gql`
  mutation DeleteCalendar($id: uuid!) {
    update_calendars_by_pk(
      pk_columns: { id: $id }
      _set: { valid_to: "now" }
    ) {
      id
      valid_to
    }
  }
`;

interface Calendar {
  id: string;
  logical_id: string;
  tenant_id: string;
  name: string;
  description?: string;
  timezone: string;
  created_at: string;
  updated_at: string;
}

interface CalendarListProps {
  onEdit?: (calendar: Calendar) => void;
}

export const CalendarList: React.FC<CalendarListProps> = ({ onEdit }) => {
  const { data, loading, error, refetch } = useQuery(GET_CALENDARS);
  const [deleteCalendar] = useMutation(DELETE_CALENDAR);

  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedCalendar, setSelectedCalendar] = useState<Calendar | null>(null);

  useEffect(() => {
    if (data?.calendars) {
      setCalendars(data.calendars);
    }
  }, [data]);

  const handleDeleteClick = (calendar: Calendar) => {
    setSelectedCalendar(calendar);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!selectedCalendar) return;
    try {
      await deleteCalendar({ variables: { id: selectedCalendar.id } });
      setDeleteDialogOpen(false);
      refetch();
    } catch (err) {
      console.error('Delete error:', err);
    }
  };

  if (loading)
    return (
      <Box display="flex" justifyContent="center" p={3}>
        <CircularProgress />
      </Box>
    );
  if (error) return <Alert severity="error">Error: {error.message}</Alert>;

  return (
    <>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell sx={{ fontWeight: 'bold' }}>Name</TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>Description</TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>Timezone</TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>Created</TableCell>
              <TableCell sx={{ fontWeight: 'bold' }} align="right">
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {calendars.map((calendar) => (
              <TableRow
                key={calendar.id}
                sx={{
                  '&:hover': { backgroundColor: '#f9f9f9' },
                  '&:last-child td, &:last-child th': { border: 0 },
                }}
              >
                <TableCell sx={{ fontWeight: 500 }}>{calendar.name}</TableCell>
                <TableCell>{calendar.description || '-'}</TableCell>
                <TableCell>
                  <Chip label={calendar.timezone} size="small" variant="outlined" />
                </TableCell>
                <TableCell>{new Date(calendar.created_at).toLocaleDateString()}</TableCell>
                <TableCell align="right">
                  <Stack direction="row" spacing={1} justifyContent="flex-end">
                    <Button
                      variant="outlined"
                      size="small"
                      startIcon={<EditIcon />}
                      onClick={() => onEdit && onEdit(calendar)}
                    >
                      Edit
                    </Button>
                    <Button
                      variant="outlined"
                      color="error"
                      size="small"
                      startIcon={<DeleteIcon />}
                      onClick={() => handleDeleteClick(calendar)}
                    >
                      Delete
                    </Button>
                  </Stack>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>Delete Calendar</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete "{selectedCalendar?.name}"? This action cannot be
            undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteConfirm} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};
