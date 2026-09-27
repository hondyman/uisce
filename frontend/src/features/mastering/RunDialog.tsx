import React, { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, LinearProgress, MenuItem,
  Stack, Switch, TextField, Typography,
} from '@mui/material';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { stagingBindingsApi } from '../staging-bindings/api';
import { fmt } from '../schedules/api';
import { Counts, masteringApi, Preview, Profile, Run } from './api';

const COUNT_KEYS: (keyof Counts)[] = ['records', 'invalid', 'xref', 'deterministic', 'fuzzy', 'review', 'new', 'conflicts', 'published', 'held_for_review', 'unchanged', 'exceptions'];

export function CountChips({ counts }: { counts: Partial<Counts> }) {
  const { t } = useTranslation();
  return (
    <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap>
      {COUNT_KEYS.filter((k) => (counts[k] ?? 0) > 0).map((k) => (
        <Chip key={k} size="small" variant="outlined"
          color={k === 'invalid' || k === 'conflicts' ? 'error' : k === 'review' || k === 'held_for_review' || k === 'exceptions' ? 'warning' : k === 'published' ? 'success' : 'default'}
          label={`${t(`mastering.counts.${k}`)} ${counts[k]}`} />
      ))}
    </Stack>
  );
}

/**
 * Masters one staging load: pick the load and the staging table it landed
 * in, preview what would happen (nothing is kept), then run.
 */
export default function RunDialog({ profile, open, onClose, onDone }: {
  profile: Profile; open: boolean; onClose: () => void; onDone: (r: Run) => void;
}) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const entity = profile.entity_cd.toLowerCase();
  const loads = useQuery({ queryKey: ['mastering-loads', entity], queryFn: () => masteringApi.loads(entity), enabled: open });
  const bindings = useQuery({ queryKey: ['sb-bindings'], queryFn: stagingBindingsApi.list, enabled: open });
  const tables = useMemo(() => (bindings.data?.bindings ?? []).filter((b) => b.bo_key === profile.bo_key).map((b) => b.staging_table), [bindings.data, profile.bo_key]);

  const [loadId, setLoadId] = useState('');
  const [table, setTable] = useState('');
  const [again, setAgain] = useState(false);
  const [preview, setPreview] = useState<Preview | null>(null);
  const load = loads.data?.loads.find((l) => l.id === loadId);

  useEffect(() => { if (!table && tables.length === 1) setTable(tables[0]); }, [tables, table]);
  useEffect(() => { setPreview(null); setAgain(false); }, [loadId, table]);

  const req = () => ({
    staging_table: table, load_run_id: loadId,
    // A load is mastered once per key; "master again" gives it a new one.
    idempotency_key: again ? `load:${loadId}:${Date.now()}` : undefined,
  });
  const doPreview = useMutation({ mutationFn: () => masteringApi.preview(entity, req()), onSuccess: (r) => setPreview(r.preview) });
  const [stage, setStage] = useState<string | null>(null);
  const doRun = useMutation({
    mutationFn: () => masteringApi.runToEnd(entity, req(), (r) => setStage(r.stage)),
    onSuccess: (r) => {
      qc.invalidateQueries({ queryKey: ['mastering'] });
      onDone(r.run);
    },
  });
  const ready = !!loadId && !!table;
  const alreadyMastered = !!load?.mastering_run_id;

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>{t('mastering.runDialog.title', { entity: profile.display_name })}</DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2.5}>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
            <TextField select fullWidth label={t('mastering.runDialog.load')} value={loadId} onChange={(e) => setLoadId(e.target.value)}
              helperText={loads.data && loads.data.loads.length === 0 ? t('mastering.runDialog.noLoads') : undefined}>
              {(loads.data?.loads ?? []).map((l) => (
                <MenuItem key={l.id} value={l.id}>
                  <Box>
                    <Typography variant="body2">{l.source} · {l.run_ref ?? l.id.slice(0, 8)} · {fmt(l.started_at, i18n.language)}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {[
                        l.received_rows !== undefined && l.received_rows !== null ? t('mastering.runDialog.rows', { n: l.received_rows }) : '',
                        l.mastering_status ? t('mastering.runDialog.mastered', { status: t(`mastering.runStatus.${l.mastering_status}`) }) : '',
                      ].filter(Boolean).join(' · ')}
                    </Typography>
                  </Box>
                </MenuItem>
              ))}
            </TextField>
            <TextField select fullWidth label={t('mastering.runDialog.table')} value={table} onChange={(e) => setTable(e.target.value)}
              helperText={bindings.data && tables.length === 0 ? t('mastering.runDialog.noBinding', { bo: profile.bo_key }) : t('mastering.runDialog.tableHelp')}>
              {tables.map((x) => <MenuItem key={x} value={x}>{x}</MenuItem>)}
            </TextField>
          </Stack>
          {(loads.error || bindings.error) && <CatalogErrorAlert error={loads.error || bindings.error} />}
          {alreadyMastered && (
            <Alert severity="info">
              {t('mastering.runDialog.alreadyMastered')}
              <FormControlLabel sx={{ display: 'block', mt: 1 }} control={<Switch checked={again} onChange={(e) => setAgain(e.target.checked)} />}
                label={t('mastering.runDialog.again')} />
            </Alert>
          )}

          {(doPreview.isPending || doRun.isPending) && <LinearProgress />}
          {doRun.isPending && stage && (
            <Typography variant="caption" color="text.secondary">{t('mastering.runDialog.stage', { stage: t(`mastering.stage.${stage}`, stage) })}</Typography>
          )}
          {doPreview.error && <CatalogErrorAlert error={doPreview.error} />}
          {doRun.error && <CatalogErrorAlert error={doRun.error} />}
          {preview && (
            <Box>
              <Typography variant="subtitle2" gutterBottom>{t('mastering.runDialog.previewTitle')}</Typography>
              <CountChips counts={preview.counts} />
              {preview.exceptions.length > 0 && (
                <Box sx={{ mt: 1.5, maxHeight: 220, overflow: 'auto' }}>
                  {preview.exceptions.map((x, i) => (
                    <Typography key={i} variant="body2" sx={{ mb: 0.5 }}>
                      <Chip size="small" color={x.severity === 'ERROR' ? 'error' : 'warning'} label={x.code} sx={{ mr: 1 }} />{x.message}
                    </Typography>
                  ))}
                </Box>
              )}
              {!!preview.unmastered_fields?.length && (
                <Typography variant="caption" color="text.secondary" component="p" sx={{ mt: 1 }}>
                  {t('mastering.runDialog.unmastered', { fields: preview.unmastered_fields.join(', ') })}
                </Typography>
              )}
            </Box>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('mastering.cancel')}</Button>
        <Button onClick={() => doPreview.mutate()} disabled={!ready || doPreview.isPending || doRun.isPending}>{t('mastering.runDialog.preview')}</Button>
        <Button variant="contained" onClick={() => doRun.mutate()} disabled={!ready || doRun.isPending || (alreadyMastered && !again)}>
          {t('mastering.runDialog.run')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
