import React, { useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControlLabel,
  Paper,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { accountGoldApi, AccountFieldOverride } from './api';

const AccountGoldPage: React.FC = () => {
  const qc = useQueryClient();
  const [accountCd, setAccountCd] = useState('');
  const [filterCd, setFilterCd] = useState('');
  const [fieldCd, setFieldCd] = useState('');
  const [overrideValue, setOverrideValue] = useState('');
  const [reason, setReason] = useState('');
  const [publish, setPublish] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const overridesQ = useQuery({
    queryKey: ['account-gold-overrides', filterCd],
    queryFn: () => accountGoldApi.listOverrides(filterCd || undefined),
  });

  const createMut = useMutation({
    mutationFn: () =>
      accountGoldApi.createOverride({
        account_cd: accountCd.trim(),
        field_cd: fieldCd.trim(),
        override_value: overrideValue,
        reason: reason.trim(),
      }),
    onSuccess: () => {
      setMessage('Override submitted (pending approval)');
      setError(null);
      setFieldCd('');
      setOverrideValue('');
      setReason('');
      qc.invalidateQueries({ queryKey: ['account-gold-overrides'] });
    },
    onError: (e: Error) => setError(e.message || 'Create failed'),
  });

  const approveMut = useMutation({
    mutationFn: (id: string) => accountGoldApi.approve(id),
    onSuccess: () => {
      setMessage('Override approved');
      qc.invalidateQueries({ queryKey: ['account-gold-overrides'] });
    },
    onError: (e: Error) => setError(e.message || 'Approve failed'),
  });

  const rejectMut = useMutation({
    mutationFn: (id: string) => accountGoldApi.reject(id),
    onSuccess: () => {
      setMessage('Override rejected');
      qc.invalidateQueries({ queryKey: ['account-gold-overrides'] });
    },
    onError: (e: Error) => setError(e.message || 'Reject failed'),
  });

  const buildMut = useMutation({
    mutationFn: () =>
      accountGoldApi.build({
        account_cd: accountCd.trim(),
        publish,
        change_reason: reason.trim() || 'steward gold build',
      }),
    onSuccess: (rec) => {
      setMessage(
        `Gold v${rec.gold_version} materialised for ${rec.account_cd}${publish ? ' (publish requested)' : ''}`,
      );
      setError(null);
    },
    onError: (e: Error) => setError(e.message || 'Build failed'),
  });

  const overrides = overridesQ.data?.overrides ?? [];
  const pending = useMemo(
    () => overrides.filter((o) => o.approval_status === 'pending'),
    [overrides],
  );

  const statusChip = (o: AccountFieldOverride) => {
    const color =
      o.approval_status === 'approved'
        ? 'success'
        : o.approval_status === 'rejected'
          ? 'error'
          : 'warning';
    return <Chip size="small" color={color} label={o.approval_status} />;
  };

  return (
    <Box sx={{ p: 3, maxWidth: 1100, mx: 'auto' }}>
      <Typography variant="h4" gutterBottom>
        Account Gold Copy
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        Steward overrides beat survivorship for the named field once approved. Build materialises a
        gold version (and optionally publishes to Kafka when KAFKA_BROKERS is set).
      </Typography>

      {message && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setMessage(null)}>
          {message}
        </Alert>
      )}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Paper sx={{ p: 2, mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Propose override / build gold
        </Typography>
        <Stack spacing={2}>
          <TextField
            size="small"
            label="Account code"
            value={accountCd}
            onChange={(e) => setAccountCd(e.target.value)}
            helperText="Natural key on mdm.account_master"
          />
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              size="small"
              label="Field code"
              value={fieldCd}
              onChange={(e) => setFieldCd(e.target.value)}
              helperText="attribute_def field_cd / semantic term field"
              sx={{ minWidth: 220 }}
            />
            <TextField
              size="small"
              label="Override value"
              value={overrideValue}
              onChange={(e) => setOverrideValue(e.target.value)}
              sx={{ flex: 1 }}
            />
          </Stack>
          <TextField
            size="small"
            label="Reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            multiline
            minRows={2}
          />
          <Stack direction="row" spacing={2} flexWrap="wrap" useFlexGap>
            <Button
              variant="contained"
              disabled={
                createMut.isPending || !accountCd.trim() || !fieldCd.trim() || !reason.trim()
              }
              onClick={() => createMut.mutate()}
            >
              Submit override
            </Button>
            <FormControlLabel
              control={<Switch checked={publish} onChange={(e) => setPublish(e.target.checked)} />}
              label="Publish to Kafka on build"
            />
            <Button
              variant="outlined"
              disabled={buildMut.isPending || !accountCd.trim()}
              onClick={() => buildMut.mutate()}
            >
              Build gold copy
            </Button>
          </Stack>
        </Stack>
      </Paper>

      <Paper sx={{ p: 2 }}>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }} alignItems="center">
          <Typography variant="h6" sx={{ flex: 1 }}>
            Overrides {pending.length > 0 ? `(${pending.length} pending)` : ''}
          </Typography>
          <TextField
            size="small"
            label="Filter account"
            value={filterCd}
            onChange={(e) => setFilterCd(e.target.value)}
          />
          <Button onClick={() => qc.invalidateQueries({ queryKey: ['account-gold-overrides'] })}>
            Refresh
          </Button>
        </Stack>

        {overridesQ.isLoading ? (
          <CircularProgress size={28} />
        ) : overrides.length === 0 ? (
          <Typography color="text.secondary">No overrides yet.</Typography>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Account</TableCell>
                <TableCell>Field</TableCell>
                <TableCell>Value</TableCell>
                <TableCell>Reason</TableCell>
                <TableCell>Status</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {overrides.map((o) => (
                <TableRow key={o.id}>
                  <TableCell>{o.account_cd}</TableCell>
                  <TableCell>
                    <Typography variant="body2" fontFamily="monospace">
                      {o.field_cd}
                    </Typography>
                  </TableCell>
                  <TableCell>{o.override_value}</TableCell>
                  <TableCell>
                    <Typography variant="caption">{o.reason}</Typography>
                  </TableCell>
                  <TableCell>{statusChip(o)}</TableCell>
                  <TableCell align="right">
                    {o.approval_status === 'pending' && (
                      <Stack direction="row" spacing={1} justifyContent="flex-end">
                        <Button
                          size="small"
                          color="success"
                          onClick={() => approveMut.mutate(o.id)}
                          disabled={approveMut.isPending}
                        >
                          Approve
                        </Button>
                        <Button
                          size="small"
                          color="warning"
                          onClick={() => rejectMut.mutate(o.id)}
                          disabled={rejectMut.isPending}
                        >
                          Reject
                        </Button>
                      </Stack>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Paper>
    </Box>
  );
};

export default AccountGoldPage;
