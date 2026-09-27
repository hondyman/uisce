import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { alpha } from '@mui/material/styles';
import { Box, Chip, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Tooltip, Typography } from '@mui/material';
import { Candidate, GoldenDetail, showValue } from './api';

type Decision = GoldenDetail['decisions'][number];

const same = (a: unknown, b: unknown) => showValue(a) === showValue(b);

/** The strategy that decided a value: the reason's prefix ("SOURCE_PRIORITY: ..."). */
function strategyOf(d?: Decision): { strategy: string; detail: string } {
  const r = d?.reason ?? '';
  const i = r.indexOf(':');
  if (i > 0 && /^[A-Z_]+$/.test(r.slice(0, i))) return { strategy: r.slice(0, i), detail: r.slice(i + 1).trim() };
  return { strategy: '', detail: r };
}

/**
 * Side by side: each attribute (by its semantic term) with the golden value,
 * every source's value - the one selected on green - and what selected it
 * (strategy, rule, selection rule, steward override).
 */
export default function SourceMatrix({ d }: { d: GoldenDetail }) {
  const { t } = useTranslation();
  const decisions = useMemo(() => new Map(d.decisions.map((x) => [x.field, x])), [d.decisions]);

  // Source columns: the sources that contributed, the most often selected first.
  const sources = useMemo(() => {
    const wins = new Map<string, number>();
    for (const x of d.decisions) {
      for (const c of x.competing ?? []) if (!wins.has(c.source)) wins.set(c.source, 0);
      if (x.source && wins.has(x.source)) wins.set(x.source, (wins.get(x.source) ?? 0) + 1);
    }
    return [...wins.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(([s]) => s);
  }, [d.decisions]);

  const rows = useMemo(() => [...d.fields].sort((a, b) =>
    (d.terms?.[a.name] ?? a.name).localeCompare(d.terms?.[b.name] ?? b.name)), [d.fields, d.terms]);

  const cell = (dec: Decision | undefined, golden: unknown, source: string) => {
    const cands: Candidate[] = (dec?.competing ?? []).filter((c) => c.source === source);
    if (cands.length === 0) return <Typography variant="body2" color="text.disabled">—</Typography>;
    return (
      <Stack spacing={0.5}>
        {cands.map((c) => {
          const won = dec?.source === source && same(c.value, golden);
          const excluded = c.selected === false;
          const body = (
            <Box key={`${c.source_key}`} sx={(th) => ({
              px: 1, py: 0.25, borderRadius: 1,
              bgcolor: won ? alpha(th.palette.success.main, 0.22) : 'transparent',
              outline: won ? `1px solid ${alpha(th.palette.success.main, 0.6)}` : 'none',
            })}>
              <Typography variant="body2" sx={{
                fontWeight: won ? 600 : 400, wordBreak: 'break-word',
                textDecoration: excluded ? 'line-through' : 'none', color: excluded ? 'text.secondary' : 'text.primary',
              }}>
                {showValue(c.value)}
              </Typography>
              {(c.stale || cands.length > 1) && (
                <Stack direction="row" spacing={0.5} sx={{ mt: 0.25 }}>
                  {c.stale && <Chip size="small" color="warning" variant="outlined" label={t('mastering.golden.stale')} />}
                  {cands.length > 1 && <Typography variant="caption" color="text.secondary" fontFamily="monospace">{c.source_key}</Typography>}
                </Stack>
              )}
            </Box>
          );
          return c.note ? <Tooltip key={c.source_key} title={c.note}>{body}</Tooltip> : body;
        })}
      </Stack>
    );
  };

  return (
    <>
      <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 1 }} flexWrap="wrap" useFlexGap>
        <Stack direction="row" spacing={0.75} alignItems="center">
          <Box sx={(th) => ({ width: 14, height: 14, borderRadius: 0.5, bgcolor: alpha(th.palette.success.main, 0.22), outline: `1px solid ${alpha(th.palette.success.main, 0.6)}` })} />
          <Typography variant="caption">{t('mastering.compare.selected')}</Typography>
        </Stack>
        <Typography variant="caption" sx={{ textDecoration: 'line-through' }} color="text.secondary">{t('mastering.compare.excluded')}</Typography>
        <Typography variant="caption" color="text.secondary">{t('mastering.compare.help')}</Typography>
      </Stack>
      <TableContainer sx={{ maxHeight: '70vh', border: 1, borderColor: 'divider', borderRadius: 1 }}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell sx={{ minWidth: 200, position: 'sticky', left: 0, zIndex: 3 }}>{t('mastering.compare.term')}</TableCell>
              <TableCell sx={{ minWidth: 170 }}>{t('mastering.compare.golden')}</TableCell>
              {sources.map((s) => <TableCell key={s} sx={{ minWidth: 150 }}>{s}</TableCell>)}
              <TableCell sx={{ minWidth: 260 }}>{t('mastering.compare.selection')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((f) => {
              const dec = decisions.get(f.name);
              const { strategy, detail } = strategyOf(dec);
              const override = strategy === 'OVERRIDE' || f.source === 'STEWARD' || dec?.source === 'STEWARD';
              const rule = dec?.rule_id ? d.rules?.[dec.rule_id] : undefined;
              return (
                <TableRow key={f.name} hover sx={{ verticalAlign: 'top' }}>
                  <TableCell sx={{ position: 'sticky', left: 0, bgcolor: 'background.paper', zIndex: 1 }}>
                    <Typography variant="body2" fontWeight={600}>{d.terms?.[f.name] ?? f.name}</Typography>
                    <Typography variant="caption" color="text.secondary" fontFamily="monospace">{f.name}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" fontWeight={700} sx={{ wordBreak: 'break-word' }}>{showValue(f.value)}</Typography>
                    {override && <Chip size="small" color="secondary" sx={{ mt: 0.5 }} label={t('mastering.overrides.steward')} />}
                  </TableCell>
                  {sources.map((s) => <TableCell key={s}>{cell(dec, f.value, s)}</TableCell>)}
                  <TableCell>
                    {strategy && <Chip size="small" variant="outlined" color={override ? 'secondary' : 'default'}
                      label={t(`mastering.compare.strategy.${strategy}`, strategy)} sx={{ mb: 0.5 }} />}
                    {rule && <Typography variant="caption" component="div" color="text.secondary">{rule}</Typography>}
                    <Typography variant="caption" component="div">{detail || '—'}</Typography>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
}
