import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Box, Chip, Collapse, Divider, Drawer, IconButton, LinearProgress, Stack, Table, TableBody, TableCell, TableHead, TableRow,
  Tooltip, Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import EditIcon from '@mui/icons-material/Edit';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { fmt } from '../schedules/api';
import { GoldenDetail, masteringApi, pct, showValue } from './api';
import { GoldenStatusChip } from './parts';
import { OverrideDialog, OverrideStatusChip } from './overrides';

const show = (v: unknown) => (v === undefined || v === null || v === '' ? '—' : String(v));

/** One attribute: the surviving value, its source and why, and every competing value. */
function FieldRow({ f, decision, onOverride, overridden }: {
  f: GoldenDetail['fields'][number]; decision?: GoldenDetail['decisions'][number]; onOverride: () => void; overridden: boolean;
}) {
  const { t, i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const competing = decision?.competing ?? [];
  return (
    <>
      <TableRow hover>
        <TableCell sx={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>{f.name}</TableCell>
        <TableCell><Typography variant="body2" fontWeight={500}>{show(f.value)}</Typography></TableCell>
        <TableCell sx={{ whiteSpace: 'nowrap' }}>
          {overridden
            ? <Chip size="small" color="secondary" label={t('mastering.overrides.steward')} />
            : f.source && <Chip size="small" variant="outlined" label={f.source} />}
        </TableCell>
        <TableCell>
          <Tooltip title={t('mastering.golden.confidenceHelp')}>
            <Chip size="small" color={(f.confidence ?? 0) >= 0.8 ? 'success' : (f.confidence ?? 0) >= 0.5 ? 'warning' : 'error'}
              variant="outlined" label={pct(f.confidence)} />
          </Tooltip>
        </TableCell>
        <TableCell padding="none" sx={{ whiteSpace: 'nowrap' }}>
          <Tooltip title={t('mastering.overrides.action', { field: f.name })}>
            <IconButton size="small" onClick={onOverride} aria-label={t('mastering.overrides.action', { field: f.name })}><EditIcon fontSize="small" /></IconButton>
          </Tooltip>
          {decision && (
            <IconButton size="small" onClick={() => setOpen(!open)} aria-label={t('mastering.golden.why', { field: f.name })}>
              {open ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
            </IconButton>
          )}
        </TableCell>
      </TableRow>
      {decision && (
        <TableRow>
          <TableCell colSpan={5} sx={{ py: 0, borderBottom: open ? undefined : 0 }}>
            <Collapse in={open} unmountOnExit>
              <Box sx={{ py: 1.5 }}>
                <Typography variant="body2" sx={{ mb: 1 }}>{decision.reason}</Typography>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>{t('mastering.golden.source')}</TableCell>
                      <TableCell>{t('mastering.golden.value')}</TableCell>
                      <TableCell>{t('mastering.golden.asOf')}</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {competing.map((c) => (
                      <TableRow key={`${c.source}|${c.source_key}`}>
                        <TableCell>
                          {c.source} <Typography component="span" variant="caption" color="text.secondary" fontFamily="monospace">{c.source_key}</Typography>
                        </TableCell>
                        <TableCell>{show(c.value)}</TableCell>
                        <TableCell sx={{ whiteSpace: 'nowrap' }}>
                          {fmt(c.as_of, i18n.language)}
                          {c.stale && <Chip size="small" color="warning" variant="outlined" sx={{ ml: 1 }} label={t('mastering.golden.stale')} />}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </Box>
            </Collapse>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Box sx={{ mt: 3 }}>
      <Typography variant="subtitle2" sx={{ mb: 1 }}>{title}</Typography>
      {children}
    </Box>
  );
}

/** A golden record's provenance: fields and why, identifiers, sources, versions, open exceptions. */
export default function GoldenDrawer({ entity, id, onClose }: { entity: string; id: string | null; onClose: () => void }) {
  const { t, i18n } = useTranslation();
  const q = useQuery({ queryKey: ['mastering-golden', entity, id], queryFn: () => masteringApi.goldenById(entity, id!), enabled: !!id });
  const d = q.data?.golden;
  const current = d?.versions[0];
  const name = current?.attributes?.name as string | undefined;
  const decisionFor = new Map((d?.decisions ?? []).map((x) => [x.field, x]));
  const overrides = useQuery({ queryKey: ['mastering', 'overrides', entity, 'golden', id], queryFn: () => masteringApi.overrides(entity, { golden: id! }), enabled: !!id });
  const activeAttrs = new Set((overrides.data?.overrides ?? []).filter((o) => o.active).map((o) => o.attribute));
  const [editing, setEditing] = useState<{ attribute: string; current?: string } | null>(null);

  return (
    <Drawer anchor="right" open={!!id} onClose={onClose} PaperProps={{ sx: { width: { xs: '100%', md: 760 } } }}>
      <Box sx={{ p: 3 }}>
        <Stack direction="row" alignItems="flex-start" spacing={2}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="overline" color="text.secondary">{d?.code}</Typography>
            <Typography variant="h6" fontWeight={700} noWrap>{name ?? d?.code ?? '…'}</Typography>
          </Box>
          <IconButton onClick={onClose} aria-label={t('mastering.close')}><CloseIcon /></IconButton>
        </Stack>
        {q.isLoading && <LinearProgress sx={{ my: 2 }} />}
        {q.error && <CatalogErrorAlert error={q.error} />}

        {current && (
          <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap sx={{ mt: 1 }}>
            <GoldenStatusChip status={current.status} />
            <Chip size="small" variant="outlined" label={t('mastering.golden.version', { v: current.version })} />
            <Chip size="small" variant="outlined" label={t('mastering.golden.dq', { v: show(current.dq_score) })} />
            <Chip size="small" variant="outlined" label={t('mastering.golden.identity', { v: pct(current.identity_confidence) })} />
          </Stack>
        )}

        {d && (
          <>
            <Section title={t('mastering.golden.fields')}>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>{t('mastering.golden.attribute')}</TableCell>
                    <TableCell>{t('mastering.golden.value')}</TableCell>
                    <TableCell>{t('mastering.golden.source')}</TableCell>
                    <TableCell>{t('mastering.golden.confidence')}</TableCell>
                    <TableCell padding="checkbox" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {d.fields.map((f) => (
                    <FieldRow key={f.name} f={f} decision={decisionFor.get(f.name)} overridden={activeAttrs.has(f.name)}
                      onOverride={() => setEditing({ attribute: f.name, current: f.value })} />
                  ))}
                </TableBody>
              </Table>
            </Section>

            <Section title={t('mastering.golden.identifiers')}>
              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                {d.identifiers.map((i) => (
                  <Chip key={`${i.type}|${i.value}`} size="small" color={i.is_primary ? 'primary' : 'default'} variant={i.is_primary ? 'filled' : 'outlined'}
                    label={<span><b>{i.type}</b> {i.value}{i.source ? ` · ${i.source}` : ''}</span>} />
                ))}
                {d.identifiers.length === 0 && <Typography variant="body2" color="text.secondary">{t('mastering.none')}</Typography>}
              </Stack>
            </Section>

            <Section title={t('mastering.golden.sources')}>
              <Table size="small">
                <TableBody>
                  {d.sources.map((s) => (
                    <TableRow key={`${s.source}|${s.source_key}`}>
                      <TableCell>{s.source}</TableCell>
                      <TableCell sx={{ fontFamily: 'monospace' }}>{s.source_key}</TableCell>
                      <TableCell>
                        <Chip size="small" variant="outlined" label={t(`mastering.method.${s.method}`, s.method)} />
                        {s.score !== undefined && s.score !== null && <Typography component="span" variant="caption" sx={{ ml: 1 }}>{pct(s.score)}</Typography>}
                      </TableCell>
                      <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(s.updated_at, i18n.language)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Section>

            <Section title={t('mastering.golden.versions')}>
              <Table size="small">
                <TableBody>
                  {d.versions.map((v) => (
                    <TableRow key={v.id}>
                      <TableCell>v{v.version}</TableCell>
                      <TableCell><GoldenStatusChip status={v.status} /></TableCell>
                      <TableCell>{t('mastering.golden.dq', { v: show(v.dq_score) })}</TableCell>
                      <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(v.published_at ?? v.knowledge_at, i18n.language)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Section>

            {(overrides.data?.overrides.length ?? 0) > 0 && (
              <Section title={t('mastering.overrides.forRecord')}>
                <Table size="small">
                  <TableBody>
                    {overrides.data!.overrides.map((o) => (
                      <TableRow key={o.id}>
                        <TableCell sx={{ fontFamily: 'monospace' }}>{o.attribute}</TableCell>
                        <TableCell>{o.action === 'CLEAR' ? t('mastering.overrides.clearDesc') : showValue(o.value)}</TableCell>
                        <TableCell><OverrideStatusChip o={o} /></TableCell>
                        <TableCell><Typography variant="caption">{o.requested_by_name} · {o.reason}</Typography></TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </Section>
            )}

            {d.exceptions.length > 0 && (
              <Section title={t('mastering.golden.openExceptions')}>
                {d.exceptions.map((x) => (
                  <Box key={x.id} sx={{ mb: 1 }}>
                    <Chip size="small" color={x.severity === 'ERROR' ? 'error' : 'warning'} label={x.type} sx={{ mr: 1 }} />
                    <Typography component="span" variant="body2">{x.description}</Typography>
                  </Box>
                ))}
              </Section>
            )}
            <Divider sx={{ mt: 3 }} />
          </>
        )}
      </Box>
      {d && editing && (
        <OverrideDialog entity={entity} goldenId={d.id} attribute={editing.attribute} current={editing.current}
          hasActive={activeAttrs.has(editing.attribute)} open={!!editing} onClose={() => setEditing(null)} />
      )}
    </Drawer>
  );
}
