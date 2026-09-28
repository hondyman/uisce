import React, { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Box, Button, Chip, Drawer, IconButton, LinearProgress, Stack, Table, TableBody, TableCell,
  TableHead, TableRow, Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { fmt } from '../schedules/api';
import { masteringApi, pct, PriceCandidate } from './api';
import { GoldenStatusChip } from './parts';
import { OverrideDialog, OverrideStatusChip } from './overrides';

const num = (v: number | undefined | null, lang: string, digits = 4) =>
  v === undefined || v === null ? '—' : v.toLocaleString(lang, { maximumFractionDigits: digits });

/** A move or disagreement in percent, coloured by size. */
function PctChip({ v, warn = 1, bad = 5 }: { v?: number | null; warn?: number; bad?: number }) {
  const { i18n } = useTranslation();
  if (v === undefined || v === null) return <Typography variant="body2" color="text.secondary">—</Typography>;
  const a = Math.abs(v);
  return (
    <Typography variant="body2" sx={{ fontVariantNumeric: 'tabular-nums', color: a >= bad ? 'error.main' : a >= warn ? 'warning.main' : 'text.primary' }}>
      {num(v, i18n.language, 2)}%
    </Typography>
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

function CandidateRow({ c, winner }: { c: PriceCandidate; winner?: string }) {
  const { t, i18n } = useTranslation();
  const out = !!c.excluded || c.selected === false;
  return (
    <TableRow selected={c.source === winner}>
      <TableCell sx={{ whiteSpace: 'nowrap' }}>
        {c.source}
        {c.source === winner && <Chip size="small" color="primary" sx={{ ml: 1 }} label={t('mastering.prices.won')} />}
      </TableCell>
      <TableCell align="right" sx={{ fontVariantNumeric: 'tabular-nums', whiteSpace: 'nowrap' }}>
        <Typography variant="body2" sx={out ? { textDecoration: 'line-through', color: 'text.secondary' } : undefined}>
          {num(c.value, i18n.language)} <Typography component="span" variant="caption" color="text.secondary">{c.currency}</Typography>
        </Typography>
      </TableCell>
      <TableCell align="right"><PctChip v={c.diff_pct} /></TableCell>
      <TableCell align="center">{c.rank || '—'}</TableCell>
      <TableCell sx={{ whiteSpace: 'nowrap' }}>
        {c.as_of ? fmt(c.as_of, i18n.language) : '—'}
        {c.stale && <Chip size="small" color="warning" variant="outlined" sx={{ ml: 1 }} label={t('mastering.golden.stale')} />}
      </TableCell>
      <TableCell>
        {c.excluded && <Typography variant="caption" color="warning.main">{t('mastering.prices.excluded', { why: c.excluded })}</Typography>}
        {c.note && <Typography variant="caption" color="warning.main" component="div">{c.note}</Typography>}
      </TableCell>
    </TableRow>
  );
}

/** A golden price: why this value, every quote considered, controls, versions. */
export function PriceDrawer({ entity, id, onClose }: { entity: string; id: string | null; onClose: () => void }) {
  const { t, i18n } = useTranslation();
  const q = useQuery({ queryKey: ['mastering-price', entity, id], queryFn: () => masteringApi.priceById(entity, id!), enabled: !!id });
  const d = q.data?.price;
  const [version, setVersion] = useState<number | undefined>();
  useEffect(() => setVersion(undefined), [id]);
  const latest = d?.versions[0];
  const v = d?.versions.find((x) => x.version === version) ?? latest;
  const historical = !!v && !!latest && v.version !== latest.version;
  const prov = v?.provenance;
  const th = prov?.threshold;
  // The steward's decisions on this price (overrides keyed TYPE@date).
  const attr = d ? `${d.price_type}@${d.date}` : '';
  const ovs = useQuery({
    queryKey: ['mastering', 'overrides', entity, 'golden', d?.entity_id], queryFn: () => masteringApi.overrides(entity, { golden: d!.entity_id }),
    enabled: !!d,
  });
  const mine = (ovs.data?.overrides ?? []).filter((o) => o.attribute === attr);
  const active = mine.find((o) => o.active);
  const pending = mine.find((o) => o.status === 'PENDING');
  const [deciding, setDeciding] = useState(false);

  return (
    <Drawer anchor="right" open={!!id} onClose={onClose} PaperProps={{ sx: { width: { xs: '100%', md: 760 } } }}>
      <Box sx={{ p: 3 }}>
        <Stack direction="row" alignItems="flex-start" spacing={2}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="overline" color="text.secondary">{d ? `${d.code ?? ''} · ${d.price_type} · ${d.date}` : ''}</Typography>
            <Typography variant="h6" fontWeight={700} noWrap>{d?.name ?? d?.entity_id ?? '…'}</Typography>
          </Box>
          <IconButton onClick={onClose} aria-label={t('mastering.close')}><CloseIcon /></IconButton>
        </Stack>
        {q.isLoading && <LinearProgress sx={{ my: 2 }} />}
        {q.error && <CatalogErrorAlert error={q.error} />}

        {v && prov && (
          <>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap alignItems="center" sx={{ mt: 1 }}>
              <Typography variant="h4" fontWeight={700} sx={{ fontVariantNumeric: 'tabular-nums' }}>{num(v.value, i18n.language)}</Typography>
              <Typography color="text.secondary">{v.currency}</Typography>
              <GoldenStatusChip status={v.status} />
              <Chip size="small" variant="outlined" label={t('mastering.golden.version', { v: v.version })} />
              {v.is_stale && <Chip size="small" color="warning" variant="outlined" label={t('mastering.golden.stale')} />}
              <Chip size="small" variant="outlined" label={`${t('mastering.golden.confidence')} ${pct(v.confidence)}`} />
            </Stack>
            {!historical && (
              <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 2 }}>
                <Button size="small" variant={latest!.status === 'REVIEW' ? 'contained' : 'outlined'} disabled={!!pending} onClick={() => setDeciding(true)}>
                  {latest!.status === 'REVIEW' ? t('mastering.prices.decideHeld') : active ? t('mastering.prices.changeOverride') : t('mastering.prices.correct')}
                </Button>
                {active && <Chip size="small" color="secondary" label={t('mastering.prices.stewardPrice')} />}
                {pending && <><OverrideStatusChip o={pending} /><Typography variant="caption" color="text.secondary">{t('mastering.prices.pending', { value: String(pending.value ?? '—') })}</Typography></>}
              </Stack>
            )}
            {historical && (
              <Alert severity="info" sx={{ mt: 2 }}>
                {t('mastering.prices.viewingVersion', { v: v.version, latest: latest!.version, at: fmt(v.knowledge_at, i18n.language) })}
              </Alert>
            )}
            <Typography variant="body2" sx={{ mt: 2 }}>
              <b>{v.winner ?? t('mastering.prices.previous')}</b> — {prov.reason}
            </Typography>
            {prov.prior && (
              <Typography variant="body2" color="text.secondary">
                {t('mastering.prices.prior', { date: prov.prior.date, value: num(prov.prior.value, i18n.language) })}
                {prov.change_pct !== undefined && <> · {t('mastering.prices.moved', { pct: num(prov.change_pct, i18n.language, 2) })}</>}
              </Typography>
            )}

            <Section title={t('mastering.prices.quotes')}>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>{t('mastering.golden.source')}</TableCell>
                    <TableCell align="right">{t('mastering.prices.price')}</TableCell>
                    <TableCell align="right">{t('mastering.prices.diff')}</TableCell>
                    <TableCell align="center">{t('mastering.prices.rank')}</TableCell>
                    <TableCell>{t('mastering.golden.asOf')}</TableCell>
                    <TableCell />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {prov.candidates.map((c) => <CandidateRow key={c.source} c={c} winner={v.winner} />)}
                </TableBody>
              </Table>
              <Typography variant="caption" color="text.secondary" component="div" sx={{ mt: 1 }}>
                {t('mastering.prices.ranking', { order: prov.ranking.join(' › ') || '—' })}
                {th && th.warning !== undefined && <> · {t('mastering.prices.thresholds', { w: th.warning, e: th.error, c: th.critical })}</>}
              </Typography>
            </Section>

            {(prov.controls?.length ?? 0) > 0 && (
              <Section title={t('mastering.prices.controls')}>
                {prov.controls!.map((c) => (
                  <Chip key={c.control} sx={{ mr: 1, mb: 1 }} color={c.level === 'WARNING' ? 'warning' : 'error'} variant={c.action === 'HOLD' ? 'filled' : 'outlined'}
                    label={t('mastering.prices.control', { control: t(`mastering.exceptionType.${c.control}`, c.control), level: c.level.toLowerCase(), pct: num(c.pct, i18n.language, 2), action: c.action.toLowerCase() })} />
                ))}
              </Section>
            )}

            {d!.variances.length > 0 && (
              <Section title={t('mastering.prices.variances')}>
                <Table size="small">
                  <TableBody>
                    {d!.variances.map((x) => (
                      <TableRow key={x.id}>
                        <TableCell>{x.source_a} {num(x.price_a, i18n.language)}</TableCell>
                        <TableCell>{x.source_b} {num(x.price_b, i18n.language)}</TableCell>
                        <TableCell align="right"><PctChip v={x.variance_pct} /></TableCell>
                        <TableCell><Chip size="small" color={x.severity === 'WARNING' ? 'warning' : 'error'} label={x.severity} /></TableCell>
                        <TableCell>{x.status}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </Section>
            )}

            {d!.exceptions.length > 0 && (
              <Section title={t('mastering.golden.openExceptions')}>
                {d!.exceptions.map((x) => (
                  <Box key={x.id} sx={{ mb: 1 }}>
                    <Chip size="small" color={x.severity === 'ERROR' ? 'error' : 'warning'} label={t(`mastering.exceptionType.${x.type}`, x.type)} sx={{ mr: 1 }} />
                    <Typography component="span" variant="body2">{x.description}</Typography>
                  </Box>
                ))}
              </Section>
            )}

            <Section title={t('mastering.golden.versionsHelp')}>
              <Table size="small">
                <TableBody>
                  {d!.versions.map((x) => (
                    <TableRow key={x.id} hover selected={x.version === v.version} sx={{ cursor: 'pointer' }}
                      onClick={() => setVersion(x.version === latest?.version ? undefined : x.version)}>
                      <TableCell>v{x.version}</TableCell>
                      <TableCell align="right" sx={{ fontVariantNumeric: 'tabular-nums' }}>{num(x.value, i18n.language)} {x.currency}</TableCell>
                      <TableCell>{x.winner ?? '—'}</TableCell>
                      <TableCell><GoldenStatusChip status={x.status} /></TableCell>
                      <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(x.knowledge_at, i18n.language)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <Typography variant="caption" color="text.secondary">{t('mastering.prices.bitemporal')}</Typography>
            </Section>
          </>
        )}
      </Box>
      {d && latest && deciding && (
        <OverrideDialog entity={entity} goldenId={d.id} attribute={attr} current={String(latest.value)} hasActive={!!active}
          open={deciding} onClose={() => setDeciding(false)} />
      )}
    </Drawer>
  );
}
