import React, { useState } from 'react';
import { Box, TextField, FormControlLabel, Switch, Button } from '@mui/material';
import { InternalEvent } from '../../types/calendar';
import dayjs from 'dayjs';

interface Props {
  initialEvent?: Partial<InternalEvent>;
  onSubmit: (event: Omit<InternalEvent, 'id' | 'tenant_id'>) => void;
  isLoading: boolean;
}

export const EventForm: React.FC<Props> = ({ initialEvent, onSubmit, isLoading }) => {
  const [formData, setFormData] = useState<Partial<InternalEvent>>({
    title: initialEvent?.title || '',
    description: initialEvent?.description || '',
    location: initialEvent?.location || '',
    start_time: initialEvent?.start_time ? dayjs(initialEvent.start_time).format('YYYY-MM-DDTHH:mm') : dayjs().format('YYYY-MM-DDTHH:mm'),
    end_time: initialEvent?.end_time ? dayjs(initialEvent.end_time).format('YYYY-MM-DDTHH:mm') : dayjs().add(1, 'hour').format('YYYY-MM-DDTHH:mm'),
    timezone: initialEvent?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
    is_all_day: initialEvent?.is_all_day || false,
    rrule: initialEvent?.rrule || '',
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSwitchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData(prev => ({ ...prev, is_all_day: e.target.checked }));
  };

  const submitForm = () => {
    // Basic validation
    if (!formData.title || !formData.start_time || !formData.end_time) {
      alert('Please fill out all required fields.');
      return;
    }

    const eventToSubmit = {
      title: formData.title!,
      description: formData.description,
      location: formData.location,
      start_time: dayjs(formData.start_time).toISOString(),
      end_time: dayjs(formData.end_time).toISOString(),
      timezone: formData.timezone || 'UTC',
      is_all_day: formData.is_all_day || false,
      rrule: formData.rrule,
    };
    
    onSubmit(eventToSubmit);
  };

  return (
    <Box component="form" noValidate sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
      <TextField
        required
        label="Event Title"
        name="title"
        value={formData.title}
        onChange={handleChange}
        fullWidth
        disabled={isLoading}
      />
      <TextField
        label="Description"
        name="description"
        value={formData.description}
        onChange={handleChange}
        multiline
        rows={3}
        fullWidth
        disabled={isLoading}
      />
      <TextField
        label="Location"
        name="location"
        value={formData.location}
        onChange={handleChange}
        fullWidth
        disabled={isLoading}
      />
      <Box display="flex" gap={2}>
        <TextField
          required
          label="Start Time"
          type={formData.is_all_day ? "date" : "datetime-local"}
          name="start_time"
          value={formData.is_all_day ? formData.start_time?.split('T')[0] : formData.start_time}
          onChange={handleChange}
          InputLabelProps={{ shrink: true }}
          fullWidth
          disabled={isLoading}
        />
        <TextField
          required
          label="End Time"
          type={formData.is_all_day ? "date" : "datetime-local"}
          name="end_time"
          value={formData.is_all_day ? formData.end_time?.split('T')[0] : formData.end_time}
          onChange={handleChange}
          InputLabelProps={{ shrink: true }}
          fullWidth
          disabled={isLoading}
        />
      </Box>
      <FormControlLabel
        control={
          <Switch
            checked={formData.is_all_day}
            onChange={handleSwitchChange}
            name="is_all_day"
            color="primary"
            disabled={isLoading}
          />
        }
        label="All Day Event"
      />
      
      {/* We can add RecurrencePicker here later. For now just text field */}
      <TextField
        label="Recurrence Rule (RRULE)"
        name="rrule"
        value={formData.rrule}
        onChange={handleChange}
        helperText="e.g. FREQ=WEEKLY;INTERVAL=1;BYDAY=MO"
        fullWidth
        disabled={isLoading}
      />

      <Box display="flex" justifyContent="flex-end" mt={2}>
        <Button 
          variant="contained" 
          color="primary" 
          onClick={submitForm} 
          disabled={isLoading}
        >
          {initialEvent?.id ? 'Save Changes' : 'Create Event'}
        </Button>
      </Box>
    </Box>
  );
};
