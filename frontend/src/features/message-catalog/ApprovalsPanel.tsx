import React, { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Box, Button, Card, CardContent, Chip, LinearProgress, Stack, TextField, Tooltip, Typography,
} from '@mui/material';
import { Change, Me, msgcatApi } from './api';
import { CatalogErrorAlert, SeverityChip, StatusChip } from './parts';

function Diff({ c }: { c: Change }) {
  const { t } = useTranslation();
  if (c.action === 'delete') {
    return (
      <Box>
        <Typography variant="caption" color="error.main">{t('messageCatalog.approvals.deleteAction')}</Typography>
        <Typography variant="body2" sx={{ textDecoration: 'line-through', color: 'text.secondary' }}>{c.before?.text}</Typography>
      </Box>
    );
  }
  return (
    <Stack spacing={0.5}>
      {c.before ? (
        <Typography variant="body2" sx={{ textDecoration: 'line-through', color: 'text.secondary' }}>{c.before.text}</Typography>
      ) : (
        <Typography variant="caption" color="text.secondary">{t('messageCatalog.approvals.newText')}</Typography>
      )}
      <Typography variant="body2" fontWeight={500}>{c.text}</Typography>
      {c.user_action && c.user_action !== c.before?.user_action && (
        <Typography variant="caption" color="text.secondary">{t('messageCatalog.editor.userAction')}: {c.user_action}</Typography>
      )}
      {c.severity && c.before && c.severity !== c.before.severity && (
        <Stack direction="row" spacing={1} alignItems="center">
          <SeverityChip severity={c.before.severity} /><span>→</span><SeverityChip severity={c.severity} />
        </Stack>
      )}
    </Stack>
  );
}

function ChangeCard({ c, me }: { c: Change; me: Me }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [comment, setComment] = useState('');
  const mine = c.requested_by === me.user_id;
  const done = () => {
    qc.invalidateQueries({ queryKey: ['msgcat-changes'] });
    qc.invalidateQueries({ queryKey: ['msgcat-messages'] });
  };
  const decide = useMutation({
    mutationFn: (action: 'approve' | 'reject' | 'withdraw') =>
      action === 'withdraw' ? msgcatApi.withdraw(c.id) : msgcatApi[action](c.id, comment || undefined),
    onSuccess: done,
  });

  return (
    <Card variant="outlined">
      <CardContent>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
          <Box sx={{ minWidth: 180 }}>
            <Typography fontFamily="monospace" fontWeight={600}>{c.set_nbr}-{c.message_nbr}</Typography>
            <Stack direction="row" spacing={0.5} sx={{ mt: 0.5 }} flexWrap="wrap" useFlexGap>
              <Chip size="small" variant="outlined" label={c.language} />
              <Chip size="small" label={t(`messageCatalog.scope.${c.scope}`)} />
              <StatusChip status={c.status} />
            </Stack>
            <Typography variant="caption" color="text.secondary" display="block" sx={{ mt: 1 }}>
              {t('messageCatalog.approvals.requestedBy', { user: c.requested_by, when: new Date(c.requested_at).toLocaleString() })}
            </Typography>
          </Box>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Diff c={c} />
            {c.reason && (
              <Typography variant="body2" sx={{ mt: 1 }} color="text.secondary">
                {t('messageCatalog.approvals.reason')}: {c.reason}
              </Typography>
            )}
          </Box>
        </Stack>
        {c.status === 'pending' && (
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} alignItems={{ sm: 'center' }} sx={{ mt: 2 }}>
            {mine ? (
              <>
                <Typography variant="body2" color="text.secondary" sx={{ flex: 1 }}>{t('messageCatalog.approvals.yours')}</Typography>
                <Button onClick={() => decide.mutate('withdraw')} disabled={decide.isPending}>{t('messageCatalog.approvals.withdraw')}</Button>
              </>
            ) : (
              <>
                <TextField size="small" sx={{ flex: 1 }} label={t('messageCatalog.approvals.comment')} value={comment}
                  onChange={(e) => setComment(e.target.value)} />
                <Button color="error" onClick={() => decide.mutate('reject')} disabled={decide.isPending}>{t('messageCatalog.approvals.reject')}</Button>
                <Button variant="contained" onClick={() => decide.mutate('approve')} disabled={decide.isPending}>{t('messageCatalog.approvals.approve')}</Button>
              </>
            )}
          </Stack>
        )}
        {decide.error && <Box sx={{ mt: 1 }}><CatalogErrorAlert error={decide.error} /></Box>}
      </CardContent>
    </Card>
  );
}

export default function ApprovalsPanel({ me }: { me: Me }) {
  const { t } = useTranslation();
  const [showAll, setShowAll] = useState(false);
  const changes = useQuery({
    queryKey: ['msgcat-changes', showAll],
    queryFn: () => msgcatApi.changes(showAll ? 'all' : 'pending'),
  });
  const list = changes.data?.changes ?? [];

  return (
    <Stack spacing={2}>
      <Stack direction="row" alignItems="center" spacing={2}>
        <Typography color="text.secondary" sx={{ flex: 1 }}>{t('messageCatalog.approvals.intro')}</Typography>
        <Tooltip title={t('messageCatalog.approvals.historyTooltip')}>
          <Button size="small" onClick={() => setShowAll((v) => !v)}>
            {showAll ? t('messageCatalog.approvals.showPending') : t('messageCatalog.approvals.showAll')}
          </Button>
        </Tooltip>
      </Stack>
      {changes.isLoading && <LinearProgress />}
      {changes.error && <CatalogErrorAlert error={changes.error} />}
      {!changes.isLoading && list.length === 0 && (
        <Alert severity="success">{showAll ? t('messageCatalog.approvals.noneEver') : t('messageCatalog.approvals.none')}</Alert>
      )}
      {list.map((c) => <ChangeCard key={c.id} c={c} me={me} />)}
    </Stack>
  );
}
