import React from 'react';
import { Dialog, DialogTitle, DialogContent, IconButton } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { EventForm } from './EventForm';
import { InternalEvent } from '../../types/calendar';

interface Props {
  open: boolean;
  onClose: () => void;
  eventToEdit?: InternalEvent;
  onSubmit: (event: Omit<InternalEvent, 'id' | 'tenant_id'>) => void;
  isLoading: boolean;
}

export const EventDialog: React.FC<Props> = ({ open, onClose, eventToEdit, onSubmit, isLoading }) => {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {eventToEdit ? 'Edit Event' : 'Create Event'}
        <IconButton
          aria-label="close"
          onClick={onClose}
          sx={{ position: 'absolute', right: 8, top: 8 }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers>
        <EventForm 
          initialEvent={eventToEdit} 
          onSubmit={(data) => {
            onSubmit(data);
          }} 
          isLoading={isLoading} 
        />
      </DialogContent>
    </Dialog>
  );
};
