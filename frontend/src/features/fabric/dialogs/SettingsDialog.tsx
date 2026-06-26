import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Stack,
  Box,
  Typography,
  FormControlLabel,
  Switch,
  TextField,
  Select,
  MenuItem,
  Divider,
} from '@mui/material';

export interface ValidationSettings {
  stopOnFirstError: boolean;
  notifyOnFailures: boolean;
  emailDigest: boolean;
  cacheResults: boolean;
  cacheDurationMinutes: number;
  logAllValidations: boolean;
  maxRulesPerObject: number;
  timeoutSeconds: number;
}

interface SettingsDialogProps {
  open: boolean;
  onClose: () => void;
  onSave?: (settings: ValidationSettings) => void;
  initialSettings?: ValidationSettings;
}

const defaultSettings: ValidationSettings = {
  stopOnFirstError: true,
  notifyOnFailures: true,
  emailDigest: false,
  cacheResults: true,
  cacheDurationMinutes: 5,
  logAllValidations: true,
  maxRulesPerObject: 100,
  timeoutSeconds: 30,
};

const SettingsDialog: React.FC<SettingsDialogProps> = ({
  open,
  onClose,
  onSave,
  initialSettings = defaultSettings,
}) => {
  const [settings, setSettings] = useState<ValidationSettings>(initialSettings);
  const [hasChanges, setHasChanges] = useState(false);

  const handleChange = <K extends keyof ValidationSettings>(key: K, value: ValidationSettings[K]) => {
    setSettings(prev => ({ ...prev, [key]: value }));
    setHasChanges(true);
  };

  const handleSave = () => {
    onSave?.(settings);
    onClose();
  };

  const handleClose = () => {
    setSettings(initialSettings);
    setHasChanges(false);
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Validation Rules Settings</DialogTitle>
      <DialogContent sx={{ pt: 2 }}>
        <Stack spacing={3}>
          {/* Validation Behavior */}
          <Box>
            <Typography variant="subtitle2" fontWeight={600} gutterBottom>
              Validation Behavior
            </Typography>
            <Stack spacing={1.5} sx={{ mt: 1 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.stopOnFirstError}
                    onChange={(e) => handleChange('stopOnFirstError', e.target.checked)}
                  />
                }
                label="Stop on first error"
              />
              <Typography variant="caption" color="text.secondary" sx={{ ml: 4, mt: -1 }}>
                Stop validation process when first rule fails
              </Typography>
            </Stack>
          </Box>

          <Divider />

          {/* Notifications */}
          <Box>
            <Typography variant="subtitle2" fontWeight={600} gutterBottom>
              Notifications
            </Typography>
            <Stack spacing={1.5} sx={{ mt: 1 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.notifyOnFailures}
                    onChange={(e) => handleChange('notifyOnFailures', e.target.checked)}
                  />
                }
                label="Notify on rule failures"
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.emailDigest}
                    onChange={(e) => handleChange('emailDigest', e.target.checked)}
                  />
                }
                label="Email digest on validation errors"
              />
              <Typography variant="caption" color="text.secondary" sx={{ ml: 4, mt: -1 }}>
                Receive daily summary of validation failures
              </Typography>
            </Stack>
          </Box>

          <Divider />

          {/* Performance */}
          <Box>
            <Typography variant="subtitle2" fontWeight={600} gutterBottom>
              Performance
            </Typography>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.cacheResults}
                    onChange={(e) => handleChange('cacheResults', e.target.checked)}
                  />
                }
                label="Cache validation results"
              />
              {settings.cacheResults && (
                <Box sx={{ ml: 4 }}>
                  <TextField
                    type="number"
                    size="small"
                    label="Cache Duration (minutes)"
                    value={settings.cacheDurationMinutes}
                    onChange={(e) => handleChange('cacheDurationMinutes', parseInt(e.target.value) || 5)}
                    inputProps={{ min: 1, max: 60 }}
                  />
                </Box>
              )}
              <TextField
                type="number"
                size="small"
                label="Max Rules per Object"
                value={settings.maxRulesPerObject}
                onChange={(e) => handleChange('maxRulesPerObject', parseInt(e.target.value) || 100)}
                inputProps={{ min: 10, max: 1000 }}
              />
              <TextField
                type="number"
                size="small"
                label="Timeout (seconds)"
                value={settings.timeoutSeconds}
                onChange={(e) => handleChange('timeoutSeconds', parseInt(e.target.value) || 30)}
                inputProps={{ min: 5, max: 300 }}
              />
            </Stack>
          </Box>

          <Divider />

          {/* Logging */}
          <Box>
            <Typography variant="subtitle2" fontWeight={600} gutterBottom>
              Logging
            </Typography>
            <Stack spacing={1.5} sx={{ mt: 1 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.logAllValidations}
                    onChange={(e) => handleChange('logAllValidations', e.target.checked)}
                  />
                }
                label="Log all validations"
              />
              <Typography variant="caption" color="text.secondary" sx={{ ml: 4, mt: -1 }}>
                Enable audit logging for compliance and troubleshooting
              </Typography>
            </Stack>
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions sx={{ p: 2 }}>
        <Button onClick={handleClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={!hasChanges}>
          Save Settings
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default SettingsDialog;
