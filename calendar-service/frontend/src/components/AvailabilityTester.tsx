import React, { useState } from 'react';
import {
  Card,
  Button,
  Box,
  Stack,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  CircularProgress,
  Alert,
  Chip,
  Paper,
  Typography,
  Grid,
} from '@mui/material';
import { CheckCircle as CheckCircleIcon, Cancel as CancelIcon } from '@mui/icons-material';
import dayjs from 'dayjs';
import { gql, useQuery } from '@apollo/client';

const GET_PROFILES = gql`
  query GetProfiles {
    schedule_profiles(where: { valid_to: { _is_null: true } }) {
      id
      name
      timezone
    }
  }
`;

interface AvailabilityResult {
  available: boolean;
  reasons?: string[];
  checked_at: string;
}

interface AvailabilityTesterProps {
  apiUrl = '/api/v1',
}) => {
  const [profileName, setProfileName] = useState('');
  const [date, setDate] = useState<string>('');
  const [startTime, setStartTime] = useState<string>('09:00');
  const [endTime, setEndTime] = useState<string>('10:00');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<AvailabilityResult | null>(null);
  const { data: profilesData } = useQuery(GET_PROFILES);

  const handleCheckAvailability = async () => {
    if (!profileName || !date) {
      alert('Please fill all required fields');
      return;
    }

    setLoading(true);
    try {
      const dateObj = new Date(date);
      const [startHour, startMin] = startTime.split(':');
      const [endHour, endMin] = endTime.split(':');

      const start = new Date(dateObj);
      start.setHours(parseInt(startHour), parseInt(startMin), 0, 0);

      const end = new Date(dateObj);
      end.setHours(parseInt(endHour), parseInt(endMin), 0, 0);

      const response = await fetch(`${apiUrl}/check-availability`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Hasura-Tenant-Id': 'default-tenant',
        },
        body: JSON.stringify({
          profile_name: profileName,
          start: start.toISOString(),
          end: end.toISOString(),
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data: AvailabilityResult = await response.json();
      setResult(data);
    } catch (error) {
      console.error('Availability check error:', error);
      setResult({
        available: false,
        reasons: [`Error: ${error instanceof Error ? error.message : 'Unknown error'}`],
        checked_at: new Date().toISOString(),
      });
    } finally {
      setLoading(false);
    }
  };

  const handleFindNextSlot = async () => {
    if (!profileName) {
      alert('Please select a profile');
      return;
    }

    setLoading(true);
    try {
      const after = new Date();
      const response = await fetch(`${apiUrl}/next-available-slot`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Hasura-Tenant-Id': 'default-tenant',
        },
        body: JSON.stringify({
          profile_name: profileName,
          after: after.toISOString(),
          duration: 3600000,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();
      const nextSlot = new Date(data.next_slot);

      setDate(nextSlot.toISOString().split('T')[0]);
      setStartTime(
        `${String(nextSlot.getHours()).padStart(2, '0')}:${String(nextSlot.getMinutes()).padStart(2, '0')}`
      );

      const endSlot = new Date(nextSlot.getTime() + 3600000);
      setEndTime(
        `${String(endSlot.getHours()).padStart(2, '0')}:${String(endSlot.getMinutes()).padStart(2, '0')}`
      );

      setResult({
        available: true,
        reasons: [`Next available slot: ${nextSlot.toLocaleString()}`],
        checked_at: data.found_at,
      });
    } catch (error) {
      console.error('Find next slot error:', error);
      setResult({
        available: false,
        reasons: [`Error: ${error instanceof Error ? error.message : 'Unknown error'}`],
        checked_at: new Date().toISOString(),
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card sx={{ marginBottom: 3 }}>
      <Box component={Paper} p={3}>
        <Typography variant="h5" sx={{ marginBottom: 3, fontWeight: 'bold' }}>
          Availability Tester
        </Typography>

        <Box sx={{ display: loading ? 'flex' : 'hidden', justifyContent: 'center', mb: 2 }}>
          {loading && <CircularProgress />}
        </Box>

        <Grid container spacing={2} sx={{ marginBottom: 3 }}>
          <Grid item xs={12} sm={6}>
            <FormControl fullWidth>
              <InputLabel>Profile</InputLabel>
              <Select
                value={profileName}
                label="Profile"
                onChange={(e) => setProfileName(e.target.value)}
              >
                {profilesData?.schedule_profiles?.map((profile: any) => (
                  <MenuItem key={profile.id} value={profile.name}>
                    {profile.name} ({profile.timezone})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
        </Grid>

        <Grid container spacing={2} sx={{ marginBottom: 3 }}>
          <Grid item xs={12} sm={4}>
            <TextField
              label="Date"
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              InputLabelProps={{ shrink: true }}
              fullWidth
            />
          </Grid>
          <Grid item xs={12} sm={4}>
            <TextField
              label="Start Time"
              type="time"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              InputLabelProps={{ shrink: true }}
              fullWidth
            />
          </Grid>
          <Grid item xs={12} sm={4}>
            <TextField
              label="End Time"
              type="time"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              InputLabelProps={{ shrink: true }}
              fullWidth
            />
          </Grid>
        </Grid>

        <Stack direction="row" spacing={2} sx={{ marginBottom: 3 }}>
          <Button
            variant="contained"
            onClick={handleCheckAvailability}
            disabled={loading}
          >
            Check Availability
          </Button>
          <Button
            variant="outlined"
            onClick={handleFindNextSlot}
            disabled={loading}
          >
            Find Next Available Slot
          </Button>
        </Stack>

        {result && (
          <Box sx={{ marginTop: 3 }}>
            {result.available ? (
              <Alert severity="success" icon={<CheckCircleIcon />}>
                <strong>Available</strong> - The selected time slot is available
              </Alert>
            ) : (
              <Alert severity="error" icon={<CancelIcon />}>
                <strong>Not Available</strong> - The selected time slot is not available
              </Alert>
            )}

            {result.reasons && result.reasons.length > 0 && (
              <Box sx={{ marginTop: 2 }}>
                <Typography variant="subtitle2" sx={{ marginBottom: 1, fontWeight: 'bold' }}>
                  Reasons:
                </Typography>
                <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap' }}>
                  {result.reasons.map((reason, idx) => (
                    <Chip
                      key={idx}
                      label={reason}
                      color={result.available ? 'success' : 'error'}
                      variant="outlined"
                    />
                  ))}
                </Stack>
              </Box>
            )}

            <Typography
              variant="caption"
              sx={{
                marginTop: 2,
                display: 'block',
                color: '#999',
              }}
            >
              Checked at: {new Date(result.checked_at).toLocaleString()}
            </Typography>
          </Box>
        )}
      </Box>
    </Card>
  );
};

export default AvailabilityTester;
