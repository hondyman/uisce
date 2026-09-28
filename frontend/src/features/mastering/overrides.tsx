import React, { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Autocomplete, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, LinearProgress,
  MenuItem, Radio, RadioGroup, Stack, TextField, Typography,
} from '@mui/material';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { masteringApi, Override, OverrideStatus, Policy } from './api';

const STATUS_COLOR: Record<OverrideStatus, 'warning' | 'success' | 'error' | 'default'> = {
  PENDING: 'warning', APPLIED: 'success', REJECTED: 'error', WITHDRAWN: 'default',
};

export function OverrideStatusChip({ o }: { o: Override }) {
  const { t } = useTranslation();
  if (o.status === 'APPLIED' && o.action === 'SET') {
    return <Chip size="small" color={o.active ? 'success' : 'default'} label={t(o.active ? 'mastering.overrides.active' : 'mastering.overrides.ended')} />;
  }
  return <Chip size="small" color={STATUS_COLOR[o.status]} label={t(`mastering.overrides.status.${o.status}`)} />;
}

/** How the policy reads in plain words. */
export function policyText(t: (k: string, o?: Record<string, unknown>) => string, p?: Policy, attr?: string) {
  if (!p) return '';
  if (p.mode === 'DIRECT') return t('mastering.policy.directShort');
  const n = attr && p.high_risk_attributes.includes(attr) ? Math.max(p.high_risk_approvals, p.approvals_required) : p.approvals_required;
  return t('mastering.policy.approvalShort', { count: n });
}

/** Propose an override of one golden attribute (set a value, or clear an active override). */
export function OverrideDialog({ entity, goldenId, attribute, current, hasActive, open, onClose }: {
  entity: string; goldenId: string; attribute: string; current?: string; hasActive: boolean; open: boolean; onClose: () => void;
}) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const policy = useQuery({ queryKey: ['mastering', 'policy', entity], queryFn: () => masteringApi.policy(entity), enabled: open });
  const [action, setAction] = useState<'SET' | 'CLEAR'>('SET');
  const [value, setValue] = useState('');
  const [reason, setReason] = useState('');
  useEffect(() => { if (open) { setAction('SET'); setValue(current ?? ''); setReason(''); } }, [open, current]);

  const propose = useMutation({
    mutationFn: () => {
      // Numbers go as numbers; anything else as text (the server checks codes).
      const n = Number(value);
      const v = value.trim() !== '' && !Number.isNaN(n) && /^-?\d+(\.\d+)?$/.test(value.trim()) ? n : value.trim();
      return masteringApi.proposeOverride(entity, goldenId, { attribute, action, value: action === 'SET' ? v : undefined, reason: reason.trim() });
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['mastering'] }); onClose(); },
  });
  const p = policy.data?.policy;
  const ready = reason.trim() !== '' && (action === 'CLEAR' || value.trim() !== '');

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{t('mastering.overrides.title', { attribute })}</DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2}>
          {p && (
            <Alert severity={p.mode === 'DIRECT' ? 'warning' : 'info'}>
              {p.mode === 'DIRECT' ? t('mastering.overrides.appliesNow') : t('mastering.overrides.needsApproval', { count: Math.max(1, policyNeed(p, attribute)) })}
            </Alert>
          )}
          <Typography variant="body2">{t('mastering.overrides.current')}: <b>{current ?? '—'}</b></Typography>
          {hasActive && (
            <RadioGroup row value={action} onChange={(e) => setAction(e.target.value as 'SET' | 'CLEAR')}>
              <FormControlLabel value="SET" control={<Radio size="small" />} label={t('mastering.overrides.set')} />
              <FormControlLabel value="CLEAR" control={<Radio size="small" />} label={t('mastering.overrides.clear')} />
            </RadioGroup>
          )}
          {action === 'SET' && (
            <TextField autoFocus label={t('mastering.overrides.newValue')} value={value} onChange={(e) => setValue(e.target.value)} />
          )}
          <TextField label={t('mastering.overrides.reason')} value={reason} onChange={(e) => setReason(e.target.value)} multiline minRows={2} required
            helperText={t('mastering.overrides.reasonHelp')} />
          <Typography variant="caption" color="text.secondary">{t('mastering.overrides.sticky')}</Typography>
          {propose.isPending && <LinearProgress />}
          {(policy.error || propose.error) && <CatalogErrorAlert error={policy.error || propose.error} />}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('mastering.cancel')}</Button>
        <Button variant="contained" disabled={!ready || propose.isPending} onClick={() => propose.mutate()}>
          {p?.mode === 'DIRECT' ? t('mastering.overrides.apply') : t('mastering.overrides.propose')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function policyNeed(p: Policy, attr: string) {
  if (p.mode === 'DIRECT') return 0;
  return p.high_risk_attributes.includes(attr) ? Math.max(p.high_risk_approvals, p.approvals_required) : p.approvals_required;
}

/** The entity's override policy; administrators can change it (audited). */
export function PolicyDialog({ entity, open, onClose }: { entity: string; open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ['mastering', 'policy', entity], queryFn: () => masteringApi.policy(entity), enabled: open });
  const [p, setP] = useState<Policy | null>(null);
  useEffect(() => { if (q.data) setP(q.data.policy); }, [q.data]);
  const save = useMutation({
    mutationFn: () => {
      const { attributes: _known, ...policy } = p!;
      return masteringApi.setPolicy(entity, policy);
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['mastering', 'policy', entity] }); onClose(); },
  });
  const edit = !!q.data?.can_edit;
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{t('mastering.policy.title')}</DialogTitle>
      <DialogContent dividers>
        {q.isLoading && <LinearProgress />}
        {q.error && <CatalogErrorAlert error={q.error} />}
        {p && (
          <Stack spacing={2}>
            <Typography variant="body2" color="text.secondary">{t('mastering.policy.help')}</Typography>
            <RadioGroup value={p.mode} onChange={(e) => setP({ ...p, mode: e.target.value as Policy['mode'] })}>
              <FormControlLabel value="APPROVAL" disabled={!edit} control={<Radio size="small" />} label={t('mastering.policy.approval')} />
              <FormControlLabel value="DIRECT" disabled={!edit} control={<Radio size="small" />} label={t('mastering.policy.direct')} />
            </RadioGroup>
            {p.mode === 'DIRECT' && <Alert severity="warning">{t('mastering.policy.directWarning')}</Alert>}
            {p.mode === 'APPROVAL' && (
              <>
                <TextField select label={t('mastering.policy.approvers')} value={p.approvals_required} disabled={!edit}
                  onChange={(e) => setP({ ...p, approvals_required: Number(e.target.value) })} sx={{ maxWidth: 240 }}>
                  {[1, 2, 3, 4, 5].map((n) => <MenuItem key={n} value={n}>{n}</MenuItem>)}
                </TextField>
                <Autocomplete multiple freeSolo options={p.attributes ?? []} value={p.high_risk_attributes} disabled={!edit}
                  onChange={(_, v) => setP({ ...p, high_risk_attributes: v as string[] })}
                  renderInput={(params) => <TextField {...params} label={t('mastering.policy.highRisk')} helperText={t('mastering.policy.highRiskHelp')} />} />
                <TextField select label={t('mastering.policy.highRiskApprovers')} value={p.high_risk_approvals} disabled={!edit || p.high_risk_attributes.length === 0}
                  onChange={(e) => setP({ ...p, high_risk_approvals: Number(e.target.value) })} sx={{ maxWidth: 240 }}>
                  {[1, 2, 3, 4, 5].map((n) => <MenuItem key={n} value={n}>{n}</MenuItem>)}
                </TextField>
              </>
            )}
            <Typography variant="caption" color="text.secondary">
              {p.inherited ? t('mastering.policy.inherited') : t('mastering.policy.own', { by: p.updated_by ?? '—' })}
            </Typography>
            {!edit && <Alert severity="info">{t('mastering.policy.adminsOnly')}</Alert>}
            {save.error && <CatalogErrorAlert error={save.error} />}
          </Stack>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{edit ? t('mastering.cancel') : t('mastering.close')}</Button>
        {edit && <Button variant="contained" disabled={!p || save.isPending} onClick={() => save.mutate()}>{t('mastering.policy.save')}</Button>}
      </DialogActions>
    </Dialog>
  );
}
