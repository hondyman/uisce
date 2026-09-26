import React, { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Badge, Box, Button, Card, CardContent, Chip, IconButton, LinearProgress, Paper, Stack, Tab, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, Tabs, TextField, Tooltip, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import LinkIcon from '@mui/icons-material/Link';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { msgcatApi } from '../message-catalog/api';
import { fmt } from '../schedules/api';
import { Binding, Change, ChangeStatus, diffFields, stagingBindingsApi } from './api';
import BindingEditor from './BindingEditor';

const STATUS_COLOR: Record<ChangeStatus, 'warning' | 'success' | 'default' | 'error'> = {
  pending: 'warning', applied: 'success', rejected: 'error', withdrawn: 'default',
};

function StatusChip({ status }: { status: ChangeStatus }) {
  const { t } = useTranslation();
  return <Chip size="small" color={STATUS_COLOR[status]} label={t(`stagingBindings.status.${status}`)} />;
}

/** Field by field: what the change does to the binding it was proposed on. */
function ChangeDiff({ c }: { c: Change }) {
  const { t } = useTranslation();
  if (c.action === 'delete') {
    return <Typography variant="body2" color="error.main">{t('stagingBindings.approvals.deleteBinding')}</Typography>;
  }
  const rows = diffFields(c.before, c.fields).filter((d) => d.kind !== 'same');
  if (rows.length === 0) return <Typography variant="body2" color="text.secondary">{t('stagingBindings.approvals.noChange')}</Typography>;
  return (
    <Table size="small">
      <TableBody>
        {rows.map((d) => (
          <TableRow key={d.field}>
            <TableCell sx={{ fontFamily: 'monospace', border: 0, py: 0.25, pl: 0 }}>{d.field}</TableCell>
            <TableCell sx={{ border: 0, py: 0.25 }}>
              {d.before && <Typography component="span" variant="body2" sx={{ textDecoration: 'line-through', color: 'text.secondary', mr: 1 }}>{d.before}</Typography>}
              {d.after ? <Typography component="span" variant="body2" fontWeight={500}>{d.after}</Typography>
                : <Typography component="span" variant="caption" color="error.main">{t('stagingBindings.approvals.unbound')}</Typography>}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function PendingCard({ c, meId }: { c: Change; meId?: string }) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const [comment, setComment] = useState('');
  const mine = !!meId && c.requested_by === meId;
  const decide = useMutation({
    mutationFn: (a: 'approve' | 'reject' | 'withdraw') =>
      a === 'withdraw' ? stagingBindingsApi.withdraw(c.id) : stagingBindingsApi[a](c.id, comment.trim() || undefined),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sb-changes'] });
      qc.invalidateQueries({ queryKey: ['sb-bindings'] });
    },
  });
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
          <Box sx={{ minWidth: 220 }}>
            <Typography fontWeight={600}>{c.staging_table}</Typography>
            <Typography variant="body2" color="text.secondary">→ {c.bo_key}</Typography>
            <Typography variant="caption" color="text.secondary" display="block" sx={{ mt: 1 }}>
              {t('stagingBindings.approvals.proposedBy', { who: c.requested_by_name || c.requested_by, when: fmt(c.requested_at, i18n.language) })}
            </Typography>
            {c.reason && <Typography variant="body2" sx={{ mt: 1, fontStyle: 'italic' }}>“{c.reason}”</Typography>}
          </Box>
          <Box sx={{ flex: 1 }}><ChangeDiff c={c} /></Box>
          <Stack spacing={1} sx={{ minWidth: 240 }}>
            {mine ? (
              <>
                <Alert severity="info" sx={{ py: 0 }}>{t('stagingBindings.approvals.yours')}</Alert>
                <Button size="small" onClick={() => decide.mutate('withdraw')} disabled={decide.isPending}>{t('stagingBindings.approvals.withdraw')}</Button>
              </>
            ) : (
              <>
                <TextField size="small" label={t('stagingBindings.approvals.comment')} value={comment} onChange={(e) => setComment(e.target.value)} />
                <Stack direction="row" spacing={1}>
                  <Button size="small" variant="contained" color="success" onClick={() => decide.mutate('approve')} disabled={decide.isPending}>
                    {t('stagingBindings.approvals.approve')}
                  </Button>
                  <Button size="small" color="error" onClick={() => decide.mutate('reject')} disabled={decide.isPending}>
                    {t('stagingBindings.approvals.reject')}
                  </Button>
                </Stack>
              </>
            )}
            {decide.error && <CatalogErrorAlert error={decide.error} />}
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
}

export default function StagingBindingsPage() {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const [tab, setTab] = useState<'bindings' | 'approvals' | 'history'>('bindings');
  const [editing, setEditing] = useState<Binding | 'new' | null>(null);
  const bindings = useQuery({ queryKey: ['sb-bindings'], queryFn: stagingBindingsApi.list });
  const pending = useQuery({ queryKey: ['sb-changes', 'pending'], queryFn: () => stagingBindingsApi.changes('pending'), refetchInterval: 15000 });
  const history = useQuery({ queryKey: ['sb-changes', 'all'], queryFn: () => stagingBindingsApi.changes(), enabled: tab === 'history' });
  const me = useQuery({ queryKey: ['msgcat-me'], queryFn: msgcatApi.me });
  const remove = useMutation({
    mutationFn: (b: Binding) => stagingBindingsApi.propose({ bo_key: b.bo_key, staging_table: b.staging_table, action: 'delete' }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['sb-changes'] }); setTab('approvals'); },
  });
  const list = bindings.data?.bindings ?? [];
  const pendingList = pending.data?.changes ?? [];
  const meId = (me.data as { user_id?: string } | undefined)?.user_id;
  const pendingFor = useMemo(() => new Set(pendingList.map((c) => `${c.bo_key}|${c.staging_table}`)), [pendingList]);

  return (
    // Own themed surface: the shell's canvas is dark whatever the MUI mode.
    <Box sx={{ bgcolor: 'background.default', color: 'text.primary', minHeight: '100%' }}>
      <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1400, mx: 'auto' }}>
        <Stack direction={{ xs: 'column', md: 'row' }} alignItems={{ md: 'center' }} spacing={2} sx={{ mb: 2 }}>
          <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 1 }}>
            <LinkIcon color="primary" fontSize="large" />
            <Box>
              <Typography variant="h5" fontWeight={700}>{t('stagingBindings.title')}</Typography>
              <Typography color="text.secondary">{t('stagingBindings.subtitle')}</Typography>
            </Box>
          </Stack>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => setEditing('new')}>{t('stagingBindings.new')}</Button>
        </Stack>

        <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Tab value="bindings" label={t('stagingBindings.tabs.bindings')} />
          <Tab value="approvals" label={
            <Badge color="warning" badgeContent={pendingList.length} sx={{ pr: pendingList.length ? 1.5 : 0 }}>
              {t('stagingBindings.tabs.approvals')}
            </Badge>
          } />
          <Tab value="history" label={t('stagingBindings.tabs.history')} />
        </Tabs>
        {remove.error && <Box sx={{ mb: 2 }}><CatalogErrorAlert error={remove.error} /></Box>}

        {tab === 'bindings' && (
          <>
            {bindings.isLoading && <LinearProgress />}
            {bindings.error && <CatalogErrorAlert error={bindings.error} />}
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>{t('stagingBindings.columns.stagingTable')}</TableCell>
                    <TableCell>{t('stagingBindings.columns.businessObject')}</TableCell>
                    <TableCell>{t('stagingBindings.columns.fields')}</TableCell>
                    <TableCell>{t('stagingBindings.columns.origin')}</TableCell>
                    <TableCell>{t('stagingBindings.columns.updated')}</TableCell>
                    <TableCell align="right" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {list.map((b) => {
                    const waiting = pendingFor.has(`${b.bo_key}|${b.staging_table}`);
                    return (
                      <TableRow key={b.id} hover>
                        <TableCell><Typography variant="body2" fontFamily="monospace">{b.staging_table}</Typography></TableCell>
                        <TableCell>{b.bo_name} <Typography component="span" variant="caption" color="text.secondary">({b.bo_key})</Typography></TableCell>
                        <TableCell>
                          <Tooltip title={Object.entries(b.fields).map(([f, c]) => `${f} ← ${c}`).join('\n')} componentsProps={{ tooltip: { sx: { whiteSpace: 'pre-line' } } }}>
                            <Chip size="small" label={t('stagingBindings.fieldCount', { count: Object.keys(b.fields).length })} />
                          </Tooltip>
                        </TableCell>
                        <TableCell>
                          <Chip size="small" variant="outlined" color={b.origin === 'core' ? 'primary' : 'default'} label={t(`stagingBindings.origin.${b.origin}`)} />
                          {waiting && <Chip size="small" color="warning" sx={{ ml: 1 }} label={t('stagingBindings.changePending')} />}
                        </TableCell>
                        <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(b.updated_at, i18n.language)} · v{b.version}</TableCell>
                        <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                          <Tooltip title={t('stagingBindings.proposeChange')}><IconButton size="small" onClick={() => setEditing(b)}><EditIcon fontSize="small" /></IconButton></Tooltip>
                          <Tooltip title={t('stagingBindings.proposeDelete')}>
                            <span>
                              <IconButton size="small" disabled={b.origin === 'core' || remove.isPending}
                                onClick={() => { if (window.confirm(t('stagingBindings.confirmDelete', { table: b.staging_table, bo: b.bo_key }))) remove.mutate(b); }}>
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                  {!bindings.isLoading && !bindings.error && list.length === 0 && (
                    <TableRow><TableCell colSpan={6}><Typography color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>{t('stagingBindings.empty')}</Typography></TableCell></TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </>
        )}

        {tab === 'approvals' && (
          <Stack spacing={2}>
            {pending.isLoading && <LinearProgress />}
            {pending.error && <CatalogErrorAlert error={pending.error} />}
            {pendingList.map((c) => <PendingCard key={c.id} c={c} meId={meId} />)}
            {!pending.isLoading && !pending.error && pendingList.length === 0 && (
              <Typography color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>{t('stagingBindings.approvals.none')}</Typography>
            )}
          </Stack>
        )}

        {tab === 'history' && (
          <>
            {history.isLoading && <LinearProgress />}
            {history.error && <CatalogErrorAlert error={history.error} />}
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>{t('stagingBindings.history.when')}</TableCell>
                    <TableCell>{t('stagingBindings.columns.stagingTable')}</TableCell>
                    <TableCell>{t('stagingBindings.history.change')}</TableCell>
                    <TableCell>{t('stagingBindings.history.status')}</TableCell>
                    <TableCell>{t('stagingBindings.history.maker')}</TableCell>
                    <TableCell>{t('stagingBindings.history.checker')}</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(history.data?.changes ?? []).map((c) => (
                    <TableRow key={c.id} hover>
                      <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(c.requested_at, i18n.language)}</TableCell>
                      <TableCell><Typography variant="body2" fontFamily="monospace">{c.staging_table}</Typography>
                        <Typography variant="caption" color="text.secondary">→ {c.bo_key}</Typography></TableCell>
                      <TableCell>{c.action === 'delete' ? t('stagingBindings.history.delete') : t('stagingBindings.fieldCount', { count: Object.keys(c.fields || {}).length })}</TableCell>
                      <TableCell><StatusChip status={c.status} /></TableCell>
                      <TableCell>{c.requested_by_name || c.requested_by}</TableCell>
                      <TableCell>
                        {c.reviewed_by_name || c.reviewed_by || '—'}
                        {c.review_comment && <Typography variant="caption" display="block" color="text.secondary">“{c.review_comment}”</Typography>}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </>
        )}

        {editing && <BindingEditor open onClose={() => setEditing(null)} binding={editing === 'new' ? undefined : editing} />}
      </Box>
    </Box>
  );
}
