import React, { useState, useEffect } from 'react';
import { devError } from '../utils/devLogger';
import {
  Box,
  Typography,
  Paper,
  Switch,
  FormControlLabel,
  TextField,
  Button,
  Grid,
  Divider,
  Alert,
  CircularProgress,
  Select,
  MenuItem,
  FormControl,
} from '@mui/material';
import { TimePicker } from '@mui/x-date-pickers/TimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { useNotificationAPI, UserNotificationPreferences } from '../hooks/useNotificationAPI';

interface NotificationPreferencesProps {
  userId: string;
  onPreferencesUpdated?: (preferences: UserNotificationPreferences) => void;
}

export const NotificationPreferences: React.FC<NotificationPreferencesProps> = ({
  userId,
  onPreferencesUpdated
}) => {
  const [preferences, setPreferences] = useState<UserNotificationPreferences | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const { getUserPreferences, updateUserPreferences } = useNotificationAPI();

  const loadPreferences = React.useCallback(async () => {
    try {
      setLoading(true);
      const data = await getUserPreferences(userId);
      setPreferences(data);
      setError(null);
    } catch (err) {
      setError('Failed to load notification preferences');
      devError('Error loading preferences:', err);
    } finally {
      setLoading(false);
    }
  }, [userId, getUserPreferences]);

  useEffect(() => {
    loadPreferences();
  }, [loadPreferences]);

  const handleSave = async () => {
    if (!preferences) return;

      try {
        setSaving(true);
        setError(null);
        await updateUserPreferences(userId, {
          ...preferences,
        });
        setSuccess(true);
        onPreferencesUpdated?.(preferences);

        // Hide success message after 3 seconds
        setTimeout(() => setSuccess(false), 3000);
      } catch (err) {
        setError('Failed to save notification preferences');
        devError('Error saving preferences:', err);
      } finally {
        setSaving(false);
      }
  };

  const handleChannelToggle = (channel: string, enabled: boolean) => {
    if (!preferences) return;

    setPreferences({
      ...preferences,
      [channel]: enabled,
      channel_preferences: {
        ...preferences.channel_preferences,
        [channel]: enabled,
      },
    });
  };

  const handleTypePreferenceChange = (type: string, enabled: boolean) => {
    if (!preferences) return;

    setPreferences({
      ...preferences,
      type_preferences: {
        ...preferences.type_preferences,
        [type]: enabled,
      },
    });
  };

  const handleFrequencyChange = (type: string, frequency: string) => {
    if (!preferences) return;

    setPreferences({
      ...preferences,
      frequency_preferences: {
        ...preferences.frequency_preferences,
        [type]: frequency,
      },
    });
  };

  const handleQuietHoursChange = (field: 'start' | 'end', value: any) => {
    if (!preferences || !value) return;

    // Value might be a Date or a Dayjs-like object depending on adapter
    const time: Date | null = value instanceof Date ? value : (value?.toDate ? value.toDate() : null);
    if (!time) return;

    const timeString = time.toTimeString().slice(0, 5); // HH:MM format

    setPreferences({
      ...preferences,
      [field === 'start' ? 'quiet_hours_start' : 'quiet_hours_end']: timeString,
    });
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!preferences) {
    return (
      <Alert severity="error">
        Failed to load notification preferences. Please try again.
      </Alert>
    );
  }

  return (
    <LocalizationProvider dateAdapter={AdapterDateFns}>
      <Paper sx={{ p: 3, maxWidth: 800, mx: 'auto' }}>
        <Typography variant="h5" gutterBottom>
          Notification Preferences
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Customize how and when you receive notifications
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {success && (
          <Alert severity="success" sx={{ mb: 2 }}>
            Preferences saved successfully!
          </Alert>
        )}

        <Grid container spacing={3}>
          {/* Channel Preferences */}
          <Grid item xs={12}>
            <Typography variant="h6" gutterBottom>
              Notification Channels
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Choose which channels you want to receive notifications through
            </Typography>

            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={preferences.email_enabled}
                    onChange={(e) => handleChannelToggle('email_enabled', e.target.checked)}
                  />
                }
                label="Email Notifications"
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={preferences.sms_enabled}
                    onChange={(e) => handleChannelToggle('sms_enabled', e.target.checked)}
                  />
                }
                label="SMS Notifications"
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={preferences.push_enabled}
                    onChange={(e) => handleChannelToggle('push_enabled', e.target.checked)}
                  />
                }
                label="Push Notifications"
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={preferences.in_app_enabled}
                    onChange={(e) => handleChannelToggle('in_app_enabled', e.target.checked)}
                  />
                }
                label="In-App Notifications"
              />
            </Box>
          </Grid>

          <Grid item xs={12}>
            <Divider />
          </Grid>

          {/* Quiet Hours */}
          <Grid item xs={12}>
            <Typography variant="h6" gutterBottom>
              Quiet Hours
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Set times when you don't want to receive notifications
            </Typography>

            <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
              <TimePicker
                label="Start Time"
                value={preferences.quiet_hours_start ? new Date(`1970-01-01T${preferences.quiet_hours_start}`) : null}
                onChange={(time) => handleQuietHoursChange('start', time)}
                slotProps={{
                  textField: {
                    helperText: 'Start of quiet hours',
                  },
                }}
              />

              <Typography>to</Typography>

              <TimePicker
                label="End Time"
                value={preferences.quiet_hours_end ? new Date(`1970-01-01T${preferences.quiet_hours_end}`) : null}
                onChange={(time) => handleQuietHoursChange('end', time)}
                slotProps={{
                  textField: {
                    helperText: 'End of quiet hours',
                  },
                }}
              />
            </Box>
          </Grid>

          <Grid item xs={12}>
            <Divider />
          </Grid>

          {/* Notification Types */}
          <Grid item xs={12}>
            <Typography variant="h6" gutterBottom>
              Notification Types
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Choose which types of notifications you want to receive
            </Typography>

            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2 }}>
              {Object.entries(preferences.type_preferences).map(([type, enabled]) => (
                <FormControlLabel
                  key={type}
                  control={
                    <Switch
                      checked={enabled}
                      onChange={(e) => handleTypePreferenceChange(type, e.target.checked)}
                    />
                  }
                  label={type.charAt(0).toUpperCase() + type.slice(1).replace('_', ' ')}
                />
              ))}
            </Box>
          </Grid>

          <Grid item xs={12}>
            <Divider />
          </Grid>

          {/* Frequency Preferences */}
          <Grid item xs={12}>
            <Typography variant="h6" gutterBottom>
              Notification Frequency
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Set how often you want to receive different types of notifications
            </Typography>

            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              {Object.entries(preferences.frequency_preferences).map(([type, frequency]) => (
                <Box key={type} sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Typography sx={{ minWidth: 120 }}>
                    {type.charAt(0).toUpperCase() + type.slice(1).replace('_', ' ')}:
                  </Typography>
                  <FormControl size="small" sx={{ minWidth: 120 }}>
                    <Select
                      value={frequency}
                      onChange={(e) => handleFrequencyChange(type, e.target.value)}
                    >
                      <MenuItem value="immediate">Immediate</MenuItem>
                      <MenuItem value="hourly">Hourly</MenuItem>
                      <MenuItem value="daily">Daily</MenuItem>
                      <MenuItem value="weekly">Weekly</MenuItem>
                      <MenuItem value="never">Never</MenuItem>
                    </Select>
                  </FormControl>
                </Box>
              ))}
            </Box>
          </Grid>

          <Grid item xs={12}>
            <Divider />
          </Grid>

          {/* Timezone */}
          <Grid item xs={12} sm={6}>
            <TextField
              fullWidth
              label="Timezone"
              value={preferences.timezone}
              onChange={(e) => setPreferences({ ...preferences, timezone: e.target.value })}
              helperText="Your timezone for scheduling notifications"
            />
          </Grid>

          {/* Save Button */}
          <Grid item xs={12}>
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2 }}>
              <Button
                variant="outlined"
                onClick={loadPreferences}
                disabled={loading}
              >
                Reset
              </Button>
              <Button
                variant="contained"
                onClick={handleSave}
                disabled={saving}
              >
                {saving ? <CircularProgress size={20} /> : 'Save Preferences'}
              </Button>
            </Box>
          </Grid>
        </Grid>
      </Paper>
    </LocalizationProvider>
  );
};
