import React, { useState } from 'react';
import { Box, Typography, TextField, Chip, Button } from '@mui/material';

interface Attendee {
  email: string;
  responseStatus?: string;
}

interface Props {
  attendees: Attendee[];
  onChange: (attendees: Attendee[]) => void;
  disabled?: boolean;
}

export const AttendeeManager: React.FC<Props> = ({ attendees, onChange, disabled }) => {
  const [newEmail, setNewEmail] = useState('');

  const handleAdd = () => {
    if (newEmail && !attendees.find(a => a.email === newEmail)) {
      onChange([...attendees, { email: newEmail }]);
      setNewEmail('');
    }
  };

  const handleDelete = (emailToDelete: string) => {
    onChange(attendees.filter(a => a.email !== emailToDelete));
  };

  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>Attendees</Typography>
      <Box display="flex" gap={1} mb={2}>
        <TextField
          size="small"
          placeholder="Add email address"
          value={newEmail}
          onChange={(e) => setNewEmail(e.target.value)}
          disabled={disabled}
          fullWidth
        />
        <Button variant="outlined" onClick={handleAdd} disabled={disabled || !newEmail}>Add</Button>
      </Box>
      <Box display="flex" flexWrap="wrap" gap={1}>
        {attendees.map((attendee) => (
          <Chip
            key={attendee.email}
            label={attendee.email}
            onDelete={disabled ? undefined : () => handleDelete(attendee.email)}
            color={attendee.responseStatus === 'accepted' ? 'success' : 'default'}
          />
        ))}
      </Box>
    </Box>
  );
};
