import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Stack,
  Typography,
  Checkbox,
  ListItemText,
  OutlinedInput,
  Alert,
  Chip,
  Box,
} from '@mui/material';
import { Schedule as ScheduleIcon, AlarmOn as ActiveIcon } from '@mui/icons-material';
import type { ScheduleConfig } from '../types/dataExplorerTypes';
import { EXPLORER_BORDER } from '../types/dataExplorerTypes';

interface ScheduleQueryModalProps {
  open: boolean;
  onClose: () => void;
  queryName: string;
  onSaveSchedule: (config: ScheduleConfig) => void;
}

const CRON_PRESETS = [
  { label: 'Daily at 08:00 AM', expr: '0 8 * * *' },
  { label: 'Every Weekday (Mon-Fri 09:00 AM)', expr: '0 9 * * 1-5' },
  { label: 'Weekly on Monday (06:00 AM)', expr: '0 6 * * 1' },
  { label: 'Monthly on 1st day (00:00)', expr: '0 0 1 * *' },
  { label: 'Quarterly', expr: '0 0 1 1,4,7,10 *' },
];

const OUTPUT_FORMATS: ('csv' | 'json' | 'pdf' | 'excel')[] = ['csv', 'json', 'pdf', 'excel'];

export const ScheduleQueryModal: React.FC<ScheduleQueryModalProps> = ({
  open,
  onClose,
  queryName,
  onSaveSchedule,
}) => {
  const [scheduleName, setScheduleName] = useState(`${queryName} - Automated Run`);
  const [cronExpression, setCronExpression] = useState('0 8 * * *');
  const [timezone, setTimezone] = useState(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC');
  const [outputFormats, setOutputFormats] = useState<('csv' | 'json' | 'pdf' | 'excel')[]>(['csv']);
  const [emailTarget, setEmailTarget] = useState('');
  const [webhookTarget, setWebhookTarget] = useState('');
  const [saveToStorage, setSaveToStorage] = useState(true);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const handleSave = () => {
    const deliveryChannels: ScheduleConfig['deliveryChannels'] = [];
    if (saveToStorage) {
      deliveryChannels.push({ type: 'storage', target: 'tenant-warehouse-vault' });
    }
    if (emailTarget.trim()) {
      deliveryChannels.push({ type: 'email', target: emailTarget.trim() });
    }
    if (webhookTarget.trim()) {
      deliveryChannels.push({ type: 'webhook', target: webhookTarget.trim() });
    }

    onSaveSchedule({
      scheduleName: scheduleName.trim() || queryName,
      cronExpression,
      timezone,
      outputFormats,
      deliveryChannels,
      isActive: true,
    });
    setSuccessMsg('Query schedule successfully configured & registered with GSIFI compliant temporal scheduler.');
    setTimeout(() => {
      setSuccessMsg(null);
      onClose();
    }, 1200);
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1, fontWeight: 700 }}>
        <ScheduleIcon sx={{ color: '#0D9488' }} />
        Schedule Automated Query Execution
      </DialogTitle>
      <DialogContent dividers sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
        {successMsg && <Alert severity="success">{successMsg}</Alert>}

        <TextField
          label="Schedule Name"
          value={scheduleName}
          onChange={(e) => setScheduleName(e.target.value)}
          fullWidth
          size="small"
        />

        <Box>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#64748B', mb: 1, display: 'block' }}>
            CRON FREQUENCY PRESETS
          </Typography>
          <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
            {CRON_PRESETS.map((p) => (
              <Chip
                key={p.expr}
                label={p.label}
                size="small"
                onClick={() => setCronExpression(p.expr)}
                sx={{
                  bgcolor: cronExpression === p.expr ? '#0D9488' : '#F1F5F9',
                  color: cronExpression === p.expr ? '#FFF' : '#334155',
                  fontWeight: 600,
                  fontSize: '0.72rem',
                  cursor: 'pointer',
                  border: `1px solid ${EXPLORER_BORDER}`,
                }}
              />
            ))}
          </Stack>
        </Box>

        <Stack direction="row" spacing={2}>
          <TextField
            label="Cron Expression"
            value={cronExpression}
            onChange={(e) => setCronExpression(e.target.value)}
            fullWidth
            size="small"
            helperText="Standard 5-field cron syntax"
          />
          <TextField
            label="Timezone"
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
            fullWidth
            size="small"
          />
        </Stack>

        <FormControl size="small" fullWidth>
          <InputLabel>Output Formats</InputLabel>
          <Select
            multiple
            value={outputFormats}
            onChange={(e) => setOutputFormats(typeof e.target.value === 'string' ? (e.target.value.split(',') as any) : e.target.value)}
            input={<OutlinedInput label="Output Formats" />}
            renderValue={(selected) => (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {selected.map((val) => (
                  <Chip key={val} label={val.toUpperCase()} size="small" />
                ))}
              </Box>
            )}
          >
            {OUTPUT_FORMATS.map((f) => (
              <MenuItem key={f} value={f}>
                <Checkbox checked={outputFormats.indexOf(f) > -1} size="small" />
                <ListItemText primary={f.toUpperCase()} />
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <TextField
          label="Email Notification Recipients"
          placeholder="e.g. portfolio.manager@firm.com, operations@fund.com"
          value={emailTarget}
          onChange={(e) => setEmailTarget(e.target.value)}
          fullWidth
          size="small"
        />

        <TextField
          label="Webhook Notification URL (Optional)"
          placeholder="https://api.internal.firm/webhooks/query-results"
          value={webhookTarget}
          onChange={(e) => setWebhookTarget(e.target.value)}
          fullWidth
          size="small"
        />
      </DialogContent>
      <DialogActions sx={{ px: 3, py: 1.5 }}>
        <Button onClick={onClose} sx={{ textTransform: 'none' }}>
          Cancel
        </Button>
        <Button
          onClick={handleSave}
          variant="contained"
          sx={{
            bgcolor: '#0D9488',
            color: '#FFF',
            textTransform: 'none',
            fontWeight: 700,
            '&:hover': { bgcolor: '#0F766E' },
          }}
        >
          Confirm Schedule
        </Button>
      </DialogActions>
    </Dialog>
  );
};
