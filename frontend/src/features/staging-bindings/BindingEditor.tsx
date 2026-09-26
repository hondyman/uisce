import React, { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Autocomplete, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, LinearProgress, MenuItem,
  Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography,
} from '@mui/material';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { Column, pipelinesApi, platformApi, Suggestion } from '../data-pipelines/api';
import { Binding, stagingBindingsApi } from './api';

interface Props {
  open: boolean;
  onClose: () => void;
  /** Propose a change to this binding; omit to propose a new one. */
  binding?: Binding;
}

/**
 * Proposes a staging binding: which staging column each business object field
 * is read from. Nothing changes until another administrator approves it.
 */
export default function BindingEditor({ open, onClose, binding }: Props) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [boKey, setBoKey] = useState(binding?.bo_key ?? '');
  const [table, setTable] = useState(binding?.staging_table ?? '');
  const [mapping, setMapping] = useState<Record<string, string>>(binding?.fields ?? {});
  const [suggested, setSuggested] = useState<Record<string, Suggestion>>({});
  const [reason, setReason] = useState('');

  const bos = useQuery({ queryKey: ['sb-bos'], queryFn: platformApi.businessObjects, enabled: open });
  const tables = useQuery({ queryKey: ['sb-staging-tables'], queryFn: pipelinesApi.stagingTables, enabled: open });
  const schema = useQuery({ queryKey: ['sb-bo-schema', boKey], queryFn: () => platformApi.boSchema(boKey), enabled: open && !!boKey });

  const fields = useMemo(() => (schema.data?.fields ?? []).map((f) => ({ name: f.name, label: f.displayName || f.name, type: f.type })), [schema.data]);
  const columns = useMemo(() => tables.data?.find((x) => x.table === table)?.columns ?? [], [tables.data, table]);
  const bound = Object.entries(mapping).filter(([, c]) => !!c);

  // A new object or table starts from nothing (an edit keeps its binding).
  useEffect(() => {
    if (!binding) setMapping({});
    setSuggested({});
  }, [boKey, table, binding]);

  const suggest = useMutation({
    mutationFn: () => pipelinesApi.suggestMapping(
      columns.map((c) => ({ name: c.name, type: (c.type || 'string') as Column['type'] })),
      fields.map((f) => ({ name: f.name, label: f.label, type: f.type })),
    ),
    onSuccess: (list) => {
      const byField: Record<string, Suggestion> = {};
      for (const s of list) if (!byField[s.to] || byField[s.to].confidence < s.confidence) byField[s.to] = s;
      setSuggested(byField);
      // Fill only fields not already bound; the user reviews every one.
      setMapping((m) => {
        const next = { ...m };
        for (const [field, s] of Object.entries(byField)) if (!next[field]) next[field] = s.from;
        return next;
      });
    },
  });

  const propose = useMutation({
    mutationFn: () => stagingBindingsApi.propose({
      bo_key: boKey, staging_table: table, action: 'upsert', reason: reason.trim() || undefined,
      fields: Object.fromEntries(bound),
    }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sb-changes'] });
      onClose();
    },
  });

  const canPropose = !!boKey && !!table && bound.length > 0 && !propose.isPending;

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>{binding ? t('stagingBindings.editor.editTitle') : t('stagingBindings.editor.newTitle')}</DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2.5}>
          <Alert severity="info">{t('stagingBindings.editor.makerChecker')}</Alert>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
            <TextField select fullWidth label={t('stagingBindings.columns.businessObject')} value={boKey}
              onChange={(e) => setBoKey(e.target.value)} disabled={!!binding}>
              {(bos.data ?? []).map((b) => <MenuItem key={b.name} value={b.name}>{b.display_name} ({b.name})</MenuItem>)}
            </TextField>
            <TextField select fullWidth label={t('stagingBindings.columns.stagingTable')} value={table}
              onChange={(e) => setTable(e.target.value)} disabled={!!binding}>
              {(tables.data ?? []).map((x) => <MenuItem key={x.table} value={x.table}>{x.table}</MenuItem>)}
            </TextField>
          </Stack>
          {(bos.error || tables.error || schema.error) && <CatalogErrorAlert error={bos.error || tables.error || schema.error} />}

          {boKey && table && (
            <Box>
              <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
                <Typography variant="subtitle2" sx={{ flex: 1 }}>
                  {t('stagingBindings.editor.mapping', { bound: bound.length, total: fields.length })}
                </Typography>
                <Button size="small" startIcon={<AutoFixHighIcon />} onClick={() => suggest.mutate()}
                  disabled={suggest.isPending || columns.length === 0 || fields.length === 0}>
                  {t('stagingBindings.editor.suggest')}
                </Button>
              </Stack>
              {(schema.isLoading || suggest.isPending) && <LinearProgress />}
              {suggest.error && <CatalogErrorAlert error={suggest.error} />}
              <TableContainer sx={{ maxHeight: 380, border: 1, borderColor: 'divider', borderRadius: 1 }}>
                <Table size="small" stickyHeader>
                  <TableHead>
                    <TableRow>
                      <TableCell>{t('stagingBindings.editor.field')}</TableCell>
                      <TableCell sx={{ width: '45%' }}>{t('stagingBindings.editor.column')}</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {fields.map((f) => {
                      const s = suggested[f.name];
                      const value = mapping[f.name] ?? '';
                      return (
                        <TableRow key={f.name} hover>
                          <TableCell>
                            <Typography variant="body2" fontWeight={500}>{f.label}</Typography>
                            <Typography variant="caption" color="text.secondary" fontFamily="monospace">{f.name}{f.type ? ` · ${f.type}` : ''}</Typography>
                          </TableCell>
                          <TableCell>
                            <Stack direction="row" spacing={1} alignItems="center">
                              <Autocomplete
                                size="small" sx={{ flex: 1 }}
                                options={columns.map((c) => c.name)}
                                value={value || null}
                                onChange={(_, v) => setMapping((m) => ({ ...m, [f.name]: v ?? '' }))}
                                renderInput={(params) => (
                                  <TextField {...params} placeholder={t('stagingBindings.editor.notBound')}
                                    inputProps={{ ...params.inputProps, 'aria-label': t('stagingBindings.editor.columnFor', { field: f.label }) }} />
                                )}
                              />
                              {s && s.from === value && (
                                <Tooltip title={s.reason}>
                                  <Chip size="small" variant="outlined" color={s.confidence >= 0.8 ? 'success' : 'warning'}
                                    label={t('stagingBindings.editor.suggested', { pct: Math.round(s.confidence * 100) })} />
                                </Tooltip>
                              )}
                            </Stack>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}

          <TextField label={t('stagingBindings.editor.reason')} value={reason} onChange={(e) => setReason(e.target.value)}
            multiline minRows={2} helperText={t('stagingBindings.editor.reasonHelp')} />
          {propose.error && <CatalogErrorAlert error={propose.error} />}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('stagingBindings.cancel')}</Button>
        <Button variant="contained" onClick={() => propose.mutate()} disabled={!canPropose}>
          {t('stagingBindings.editor.propose')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
