import React from 'react';
import { Box, Typography, TextField } from '@mui/material';

interface Props {
  rrule: string;
  onChange: (rrule: string) => void;
  disabled?: boolean;
}

export const RecurrencePicker: React.FC<Props> = ({ rrule, onChange, disabled }) => {
  // A full implementation would parse the RRULE and show a UI builder (daily, weekly, yearly, ends on/after).
  // For now, it's just a text input that outputs standard RRULE formats.
  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>Recurrence</Typography>
      <TextField
        fullWidth
        size="small"
        placeholder="FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR"
        value={rrule || ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        helperText="Advanced recurrence rules (RRULE format)"
      />
    </Box>
  );
};
