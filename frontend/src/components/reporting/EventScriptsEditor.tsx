import type { FC } from 'react';
import { Box, Typography, TextField } from '@mui/material';
import { EventScripts } from './reportingUtils';
import { eventScriptLabels } from './reportingUtils';

type Props = {
  eventScripts: EventScripts;
  onEventScriptChange: (key: keyof EventScripts, value: string) => void;
};

const EventScriptsEditor: FC<Props> = ({ eventScripts, onEventScriptChange }) => {
  return (
    <>
      <Box>
        <Typography variant="subtitle1">Event Scripts</Typography>
        {(Object.keys(eventScripts) as Array<keyof EventScripts>).map((scriptKey) => (
          <Box key={`script_${String(scriptKey)}`} sx={{ mb: 1.5 }}>
            <Typography variant="subtitle2" gutterBottom>{eventScriptLabels[scriptKey]}</Typography>
            <TextField fullWidth size="small" multiline minRows={2} value={eventScripts[scriptKey]} onChange={(e) => onEventScriptChange(scriptKey, e.target.value)} />
          </Box>
        ))}
      </Box>
    </>
  );
};

export default EventScriptsEditor;
